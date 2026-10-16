package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/TegarNH/fti-auth-system/backend/internal/auth"
	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	service      *auth.Service
	cookieSecure bool
}

func NewAuthHandler(service *auth.Service, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		service:      service,
		cookieSecure: cookieSecure,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type UserPayloadResponse struct {
	Message string        `json:"message,omitempty"`
	User    auth.UserView `json:"user"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

// Register handles user registration with email, username, and password.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := ReadJSON(w, r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "format data json tidak valid")
		return
	}

	user, err := h.service.Register(r.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrDuplicateEmail) || errors.Is(err, auth.ErrDuplicateUsername) {
			WriteError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, auth.ErrValidation) {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, "terjadi kesalahan internal pada server")
		return
	}

	WriteJSON(w, http.StatusCreated, UserPayloadResponse{
		Message: "pendaftaran berhasil",
		User:    user.ToView(),
	})
}

// Login verifies user credentials and creates a session cookie.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := ReadJSON(w, r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "format data json tidak valid")
		return
	}

	userAgent := r.UserAgent()
	ipAddress := GetClientIP(r)

	user, rawToken, expiresAt, err := h.service.Login(r.Context(), req.Identifier, req.Password, userAgent, ipAddress)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			WriteError(w, http.StatusUnauthorized, "email/username atau kata sandi salah")
			return
		}
		WriteError(w, http.StatusInternalServerError, "terjadi kesalahan internal pada server")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    rawToken,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	WriteJSON(w, http.StatusOK, UserPayloadResponse{
		Message: "login berhasil",
		User:    user.ToView(),
	})
}

// Logout revokes the current session and clears the cookie.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.SessionFromContext(r.Context())
	if ok && session != nil {
		_ = h.service.Logout(r.Context(), session.ID)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	WriteJSON(w, http.StatusOK, MessageResponse{
		Message: "berhasil logout",
	})
}

// Me returns the profile of the authenticated user.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		WriteError(w, http.StatusUnauthorized, "sesi tidak valid atau telah berakhir")
		return
	}

	WriteJSON(w, http.StatusOK, struct {
		User auth.UserView `json:"user"`
	}{
		User: user.ToView(),
	})
}

// ListSessions returns all active sessions for the authenticated user.
func (h *AuthHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		WriteError(w, http.StatusUnauthorized, "sesi tidak valid atau telah berakhir")
		return
	}

	currentSession, _ := auth.SessionFromContext(r.Context())
	var currentSessionID string
	if currentSession != nil {
		currentSessionID = currentSession.ID
	}

	sessions, err := h.service.ListActiveSessions(r.Context(), user.ID, currentSessionID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "gagal mengambil daftar sesi")
		return
	}

	WriteJSON(w, http.StatusOK, struct {
		Sessions []auth.SessionView `json:"sessions"`
	}{
		Sessions: sessions,
	})
}

// RevokeSession revokes a specific session belonging to the authenticated user.
func (h *AuthHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		WriteError(w, http.StatusUnauthorized, "sesi tidak valid atau telah berakhir")
		return
	}

	targetSessionID := chi.URLParam(r, "id")
	if targetSessionID == "" {
		WriteError(w, http.StatusBadRequest, "ID sesi tidak valid")
		return
	}

	currentSession, _ := auth.SessionFromContext(r.Context())

	if err := h.service.RevokeUserSession(r.Context(), user.ID, targetSessionID); err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "sesi tidak ditemukan atau sudah tidak aktif")
			return
		}
		WriteError(w, http.StatusInternalServerError, "gagal mencabut sesi")
		return
	}

	// If the user revoked their currently active session, clear the auth cookie
	if currentSession != nil && currentSession.ID == targetSessionID {
		http.SetCookie(w, &http.Cookie{
			Name:     auth.SessionCookieName,
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   h.cookieSecure,
			SameSite: http.SameSiteLaxMode,
		})
	}

	WriteJSON(w, http.StatusOK, MessageResponse{
		Message: "sesi berhasil dicabut",
	})
}

