package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	rxEmail    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	rxUsername = regexp.MustCompile(`^[a-zA-Z0-9._]{3,30}$`)
)

type Service struct {
	pool              *pgxpool.Pool
	argonParams       *Argon2Params
	sessionTTLHours   int
	maxActiveSessions int
	dummyHash         string
}

func NewService(pool *pgxpool.Pool, argonParams *Argon2Params, sessionTTLHours, maxActiveSessions int) (*Service, error) {
	if argonParams == nil {
		argonParams = DefaultArgon2Params
	}

	// Precompute dummy hash for anti-enumeration timing mitigation
	dummyHash, err := HashPassword("anti-enumeration-timing-dummy-salt", argonParams)
	if err != nil {
		return nil, fmt.Errorf("creating dummy hash: %w", err)
	}

	return &Service{
		pool:              pool,
		argonParams:       argonParams,
		sessionTTLHours:   sessionTTLHours,
		maxActiveSessions: maxActiveSessions,
		dummyHash:         dummyHash,
	}, nil
}

func ValidateEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	return rxEmail.MatchString(email)
}

func ValidateUsername(username string) bool {
	return rxUsername.MatchString(username)
}

func ValidatePassword(password string) bool {
	return len(password) >= 8 && len(password) <= 128
}

// Register creates a new user account with role 'user'. Public registration cannot specify role.
func (s *Service) Register(ctx context.Context, email, username, password string) (*User, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	cleanUsername := strings.TrimSpace(username)

	if !ValidateEmail(cleanEmail) {
		return nil, fmt.Errorf("%w: format email tidak valid", ErrValidation)
	}
	if !ValidateUsername(cleanUsername) {
		return nil, fmt.Errorf("%w: username harus 3-30 karakter (hanya huruf, angka, titik, atau garis bawah)", ErrValidation)
	}
	if !ValidatePassword(password) {
		return nil, fmt.Errorf("%w: kata sandi minimal 8 dan maksimal 128 karakter", ErrValidation)
	}

	var existingEmail, existingUsername string
	err := s.pool.QueryRow(ctx, `
		SELECT email, username FROM users
		WHERE lower(email) = lower($1) OR lower(username) = lower($2)
		LIMIT 1
	`, cleanEmail, cleanUsername).Scan(&existingEmail, &existingUsername)

	if err == nil {
		if strings.EqualFold(existingEmail, cleanEmail) {
			return nil, ErrDuplicateEmail
		}
		return nil, ErrDuplicateUsername
	}
	if !errors.Is(err, pgx.ErrNoRows) && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("checking user duplication: %w", err)
	}

	hashedPassword, err := HashPassword(password, s.argonParams)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	var user User
	err = s.pool.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, role, is_active)
		VALUES ($1, $2, $3, 'user', true)
		RETURNING id, email, username, role, is_active, created_at, updated_at
	`, cleanEmail, cleanUsername, hashedPassword).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return &user, nil
}

// Login verifies credentials and creates a new session.
func (s *Service) Login(ctx context.Context, identifier, password, userAgent, ipAddress string) (*User, string, time.Time, error) {
	cleanIdentifier := strings.TrimSpace(identifier)
	if cleanIdentifier == "" || password == "" {
		return nil, "", time.Time{}, ErrInvalidCredentials
	}

	var user User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, username, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE lower(email) = lower($1) OR lower(username) = lower($1)
		LIMIT 1
	`, cleanIdentifier).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		// Anti-enumeration dummy hash check ensures identical timing
		_, _ = VerifyPassword(password, s.dummyHash)
		return nil, "", time.Time{}, ErrInvalidCredentials
	}
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("querying user: %w", err)
	}

	if !user.IsActive {
		return nil, "", time.Time{}, ErrInvalidCredentials
	}

	match, err := VerifyPassword(password, user.PasswordHash)
	if err != nil || !match {
		return nil, "", time.Time{}, ErrInvalidCredentials
	}

	_, rawToken, expiresAt, err := s.CreateSession(ctx, user.ID, userAgent, ipAddress)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("creating session: %w", err)
	}

	return &user, rawToken, expiresAt, nil
}

// CreateSession is the SINGLE function responsible for creating sessions and enforcing active session limits.
func (s *Service) CreateSession(ctx context.Context, userID, userAgent, ipAddress string) (string, string, time.Time, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("starting session transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Lock user row with SELECT ... FOR UPDATE to serialize concurrent logins
	var lockedID string
	err = tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&lockedID)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("locking user record: %w", err)
	}

	// Count active sessions
	var activeCount int
	err = tx.QueryRow(ctx, `
		SELECT count(*) FROM sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()
	`, userID).Scan(&activeCount)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("counting active sessions: %w", err)
	}

	// If at or exceeding max allowed, evict oldest session(s)
	if activeCount >= s.maxActiveSessions {
		evictLimit := activeCount - s.maxActiveSessions + 1
		rows, err := tx.Query(ctx, `
			SELECT id FROM sessions
			WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()
			ORDER BY created_at ASC
			LIMIT $2
		`, userID, evictLimit)
		if err != nil {
			return "", "", time.Time{}, fmt.Errorf("selecting oldest sessions: %w", err)
		}

		var idsToEvict []string
		for rows.Next() {
			var sid string
			if err := rows.Scan(&sid); err == nil {
				idsToEvict = append(idsToEvict, sid)
			}
		}
		rows.Close()

		if len(idsToEvict) > 0 {
			_, err = tx.Exec(ctx, `
				UPDATE sessions
				SET revoked_at = now(), revoked_reason = 'evicted'
				WHERE id = ANY($1)
			`, idsToEvict)
			if err != nil {
				return "", "", time.Time{}, fmt.Errorf("evicting oldest sessions: %w", err)
			}
		}
	}

	rawToken, tokenHash, err := GenerateSessionToken()
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generating session token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(s.sessionTTLHours) * time.Hour)

	var sessionID string
	var ipParam any
	if ipAddress != "" {
		ipParam = ipAddress
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, created_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, now(), now())
		RETURNING id
	`, userID, tokenHash, userAgent, ipParam, expiresAt).Scan(&sessionID)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("inserting session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", time.Time{}, fmt.Errorf("committing session transaction: %w", err)
	}

	return sessionID, rawToken, expiresAt, nil
}

// Authenticate verifies the session token and retrieves user details, updating last_seen_at.
func (s *Service) Authenticate(ctx context.Context, rawToken string) (*User, *Session, error) {
	if rawToken == "" {
		return nil, nil, ErrUnauthorized
	}

	tokenHash := HashToken(rawToken)

	var user User
	var session Session
	var ipAddr sql.NullString
	var revokedAt sql.NullTime
	var revokedReason sql.NullString

	err := s.pool.QueryRow(ctx, `
		SELECT 
			s.id, s.user_id, s.token_hash, s.user_agent, host(s.ip_address), s.created_at, s.last_seen_at, s.expires_at, s.revoked_at, s.revoked_reason,
			u.id, u.email, u.username, u.password_hash, u.role, u.is_active, u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.token_hash = $1
		LIMIT 1
	`, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.UserAgent,
		&ipAddr,
		&session.CreatedAt,
		&session.LastSeenAt,
		&session.ExpiresAt,
		&revokedAt,
		&revokedReason,
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrUnauthorized
	}
	if err != nil {
		return nil, nil, fmt.Errorf("querying session and user: %w", err)
	}

	if ipAddr.Valid {
		session.IPAddress = &ipAddr.String
	}
	if revokedAt.Valid {
		session.RevokedAt = &revokedAt.Time
	}
	if revokedReason.Valid {
		session.RevokedReason = &revokedReason.String
	}

	if session.RevokedAt != nil {
		return nil, nil, ErrUnauthorized
	}
	if session.ExpiresAt.Before(time.Now()) {
		return nil, nil, ErrUnauthorized
	}
	if !user.IsActive {
		return nil, nil, ErrAccountInactive
	}

	// Update last_seen_at asynchronously or synchronously
	_, err = s.pool.Exec(ctx, `UPDATE sessions SET last_seen_at = now() WHERE id = $1`, session.ID)
	if err != nil {
		log.Printf("Warning: failed to update session last_seen_at: %v", err)
	}

	return &user, &session, nil
}

// Logout revokes the specified session.
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = now(), revoked_reason = 'logout'
		WHERE id = $1 AND revoked_at IS NULL
	`, sessionID)
	if err != nil {
		return fmt.Errorf("revoking session: %w", err)
	}
	return nil
}

// ListActiveSessions returns all currently active sessions for a user, marking the current session.
func (s *Service) ListActiveSessions(ctx context.Context, userID, currentSessionID string) ([]SessionView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_agent, host(ip_address), created_at, last_seen_at, expires_at
		FROM sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("querying user active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionView
	for rows.Next() {
		var sess SessionView
		var ip sql.NullString
		err := rows.Scan(
			&sess.ID,
			&sess.UserAgent,
			&ip,
			&sess.CreatedAt,
			&sess.LastSeenAt,
			&sess.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning session row: %w", err)
		}
		if ip.Valid && ip.String != "" {
			sess.IPAddress = &ip.String
		}
		sess.IsCurrent = (sess.ID == currentSessionID)
		sessions = append(sessions, sess)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating session rows: %w", err)
	}

	if sessions == nil {
		sessions = []SessionView{}
	}

	return sessions, nil
}

// RevokeUserSession revokes an active session owned by the specified user.
func (s *Service) RevokeUserSession(ctx context.Context, userID, targetSessionID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = now(), revoked_reason = 'user_revoked'
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`, targetSessionID, userID)
	if err != nil {
		return fmt.Errorf("revoking user session: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

