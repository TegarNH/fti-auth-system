package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedInitialAdmin seeds the initial admin user idempotently without modifying existing records.
func SeedInitialAdmin(ctx context.Context, pool *pgxpool.Pool, email, username, password string, params *Argon2Params) error {
	trimmedEmail := strings.ToLower(strings.TrimSpace(email))
	trimmedUsername := strings.TrimSpace(username)

	var existingID string
	err := pool.QueryRow(ctx, `
		SELECT id FROM users
		WHERE lower(email) = lower($1) OR lower(username) = lower($2)
		LIMIT 1
	`, trimmedEmail, trimmedUsername).Scan(&existingID)

	if err == nil {
		log.Printf("Initial admin user already exists (%s), skipping seed", trimmedEmail)
		return nil
	}

	if !errors.Is(err, pgx.ErrNoRows) && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("checking existing admin user: %w", err)
	}

	hashedPassword, err := HashPassword(password, params)
	if err != nil {
		return fmt.Errorf("hashing admin password: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO users (email, username, password_hash, role, is_active)
		VALUES ($1, $2, $3, 'admin', true)
		ON CONFLICT DO NOTHING
	`, trimmedEmail, trimmedUsername, hashedPassword)
	if err != nil {
		return fmt.Errorf("inserting initial admin user: %w", err)
	}

	log.Printf("Created initial admin user (%s)", trimmedEmail)
	return nil
}
