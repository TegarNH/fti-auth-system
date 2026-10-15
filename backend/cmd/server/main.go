package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TegarNH/fti-auth-system/backend/internal/auth"
	"github.com/TegarNH/fti-auth-system/backend/internal/config"
	"github.com/TegarNH/fti-auth-system/backend/internal/db"
	"github.com/TegarNH/fti-auth-system/backend/internal/handlers"
	appMiddleware "github.com/TegarNH/fti-auth-system/backend/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("Connecting to database at %s...", cfg.DatabaseURL)
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(pool); err != nil {
		log.Fatalf("Failed to execute database migrations: %v", err)
	}

	argonParams := &auth.Argon2Params{
		Memory:      cfg.Argon2Memory,
		Iterations:  cfg.Argon2Iterations,
		Parallelism: cfg.Argon2Parallelism,
		SaltLength:  cfg.Argon2SaltLength,
		KeyLength:   cfg.Argon2KeyLength,
	}

	if err := auth.SeedInitialAdmin(ctx, pool, cfg.AdminEmail, cfg.AdminUsername, cfg.AdminPassword, argonParams); err != nil {
		log.Fatalf("Failed to seed initial admin user: %v", err)
	}

	authService, err := auth.NewService(pool, argonParams, cfg.SessionTTLHours, cfg.MaxActiveSessions)
	if err != nil {
		log.Fatalf("Failed to initialize auth service: %v", err)
	}

	authHandler := handlers.NewAuthHandler(authService, cfg.CookieSecure)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(appMiddleware.CORS(cfg.FrontendOrigin))

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "ok"
		pingCtx, pingCancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer pingCancel()

		if err := pool.Ping(pingCtx); err != nil {
			dbStatus = "error"
		}

		handlers.WriteJSON(w, http.StatusOK, healthResponse{
			Status:   "ok",
			Database: dbStatus,
		})
	})

	// Public auth endpoints
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)

		// Protected auth endpoints
		r.Group(func(r chi.Router) {
			r.Use(appMiddleware.RequireAuth(authService))
			r.Post("/logout", authHandler.Logout)
		})
	})

	// Protected user endpoints
	r.Route("/api/me", func(r chi.Router) {
		r.Use(appMiddleware.RequireAuth(authService))
		r.Get("/", authHandler.Me)
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Backend HTTP server listening on %s", addr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server stopped with error: %v", err)
		}
	case sig := <-shutdown:
		log.Printf("Received signal %v, initiating graceful shutdown...", sig)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}

	log.Println("Backend server stopped cleanly")
}
