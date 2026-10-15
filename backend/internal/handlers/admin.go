package handlers

import (
	"errors"
	"net/http"

	"github.com/TegarNH/fti-auth-system/backend/internal/auth"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	service *auth.Service
}

func NewAdminHandler(service *auth.Service) *AdminHandler {
	return &AdminHandler{service: service}
}

type UsersListResponse struct {
	Users []auth.UserWithSessions `json:"users"`
}

// ListUsers returns all users with their active session count.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListAllUsers(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "gagal mengambil daftar pengguna")
		return
	}

	WriteJSON(w, http.StatusOK, UsersListResponse{Users: users})
}

// ForceLogout revokes all sessions for the target user.
func (h *AdminHandler) ForceLogout(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	if targetID == "" {
		WriteError(w, http.StatusBadRequest, "ID pengguna tidak valid")
		return
	}

	if err := h.service.ForceLogout(r.Context(), targetID); err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "pengguna tidak ditemukan")
			return
		}
		WriteError(w, http.StatusInternalServerError, "gagal melakukan force logout")
		return
	}

	WriteJSON(w, http.StatusOK, MessageResponse{Message: "force logout berhasil"})
}
