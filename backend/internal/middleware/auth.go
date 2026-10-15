package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/TegarNH/fti-auth-system/backend/internal/auth"
)

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// RequireAuth enforces that a valid, unrevoked session cookie exists.
func RequireAuth(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.SessionCookieName)
			if err != nil || cookie.Value == "" {
				writeJSONError(w, http.StatusUnauthorized, "belum terautentikasi: sesi tidak ditemukan")
				return
			}

			user, session, err := authService.Authenticate(r.Context(), cookie.Value)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "sesi tidak valid atau telah berakhir")
				return
			}

			ctx := auth.WithUser(r.Context(), user)
			ctx = auth.WithSession(ctx, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
