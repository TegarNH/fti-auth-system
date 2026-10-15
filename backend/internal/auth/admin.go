package auth

import (
	"context"
	"fmt"
)

// UserWithSessions extends User with the count of active sessions.
type UserWithSessions struct {
	User
	ActiveSessions int `json:"active_sessions"`
}

// ListAllUsers returns all users along with their active session count.
func (s *Service) ListAllUsers(ctx context.Context) ([]UserWithSessions, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			u.id, u.email, u.username, u.role, u.is_active, u.created_at, u.updated_at,
			COUNT(sess.id) FILTER (WHERE sess.revoked_at IS NULL AND sess.expires_at > now()) AS active_sessions
		FROM users u
		LEFT JOIN sessions sess ON sess.user_id = u.id
		GROUP BY u.id, u.email, u.username, u.role, u.is_active, u.created_at, u.updated_at
		ORDER BY u.created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying all users: %w", err)
	}
	defer rows.Close()

	var users []UserWithSessions
	for rows.Next() {
		var u UserWithSessions
		err := rows.Scan(
			&u.ID, &u.Email, &u.Username, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
			&u.ActiveSessions,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning user row: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating user rows: %w", err)
	}

	if users == nil {
		users = []UserWithSessions{}
	}

	return users, nil
}

// ForceLogout revokes all active sessions for a target user.
func (s *Service) ForceLogout(ctx context.Context, targetUserID string) error {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, targetUserID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("checking user existence: %w", err)
	}
	if !exists {
		return ErrNotFound
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = now(), revoked_reason = 'force_logout'
		WHERE user_id = $1 AND revoked_at IS NULL
	`, targetUserID)
	if err != nil {
		return fmt.Errorf("revoking all sessions: %w", err)
	}

	return nil
}
