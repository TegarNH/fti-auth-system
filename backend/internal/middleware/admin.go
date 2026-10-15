package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/TegarNH/fti-auth-system/backend/internal/auth"
)

// RequireAdmin enforces that the authenticated user has the 'admin' role.
// Must be chained after RequireAuth.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFromContext(r.Context())
		if !ok || user == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "belum terautentikasi"})
			return
		}

		if user.Role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "akses ditolak: diperlukan hak admin"})
			return
		}

		next.ServeHTTP(w, r)
	})
}
