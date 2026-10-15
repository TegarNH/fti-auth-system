package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("email/username atau kata sandi salah")
	ErrDuplicateEmail     = errors.New("email sudah terdaftar")
	ErrDuplicateUsername  = errors.New("username sudah digunakan")
	ErrUnauthorized       = errors.New("sesi tidak valid atau telah berakhir")
	ErrAccountInactive    = errors.New("akun dinonaktifkan")
	ErrValidation         = errors.New("validasi input gagal")
	ErrNotFound           = errors.New("user tidak ditemukan")
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Session struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	TokenHash     []byte     `json:"-"`
	UserAgent     string     `json:"user_agent"`
	IPAddress     *string    `json:"ip_address,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	LastSeenAt    time.Time  `json:"last_seen_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	RevokedReason *string    `json:"revoked_reason,omitempty"`
}

type UserView struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (u *User) ToView() UserView {
	return UserView{
		ID:       u.ID,
		Email:    u.Email,
		Username: u.Username,
		Role:     u.Role,
	}
}

const SessionCookieName = "auth_session"

type contextKey string

const (
	UserContextKey    contextKey = "auth_user"
	SessionContextKey contextKey = "auth_session"
)

func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, UserContextKey, u)
}

func UserFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(UserContextKey).(*User)
	return u, ok
}

func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, SessionContextKey, s)
}

func SessionFromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(SessionContextKey).(*Session)
	return s, ok
}
