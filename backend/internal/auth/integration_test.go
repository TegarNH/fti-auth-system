//go:build integration

package auth_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/TegarNH/fti-auth-system/backend/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testDB opens a pool to TEST_DATABASE_URL and truncates test-related tables before returning.
func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connecting to test db: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("pinging test db: %v", err)
	}

	// Truncate in test-safe order
	_, err = pool.Exec(ctx, `TRUNCATE sessions, audit_logs, users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncating tables: %v", err)
	}

	t.Cleanup(func() { pool.Close() })

	return pool
}

// newService builds a Service with minimal safe parameters for testing.
func newService(t *testing.T, pool *pgxpool.Pool) *auth.Service {
	t.Helper()
	svc, err := auth.NewService(pool, auth.DefaultArgon2Params, 24, 2)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}
	return svc
}

// registerUser is a convenience helper that registers and returns the user.
func registerUser(t *testing.T, svc *auth.Service, email, username, password string) {
	t.Helper()
	ctx := context.Background()
	if _, err := svc.Register(ctx, email, username, password); err != nil {
		t.Fatalf("register %s: %v", email, err)
	}
}

// loginUser logs in and returns the raw session token.
func loginUser(t *testing.T, svc *auth.Service, identifier, password string) string {
	t.Helper()
	ctx := context.Background()
	_, rawToken, _, err := svc.Login(ctx, identifier, password, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("login %s: %v", identifier, err)
	}
	return rawToken
}

// countActiveSessions queries the DB directly.
func countActiveSessions(t *testing.T, pool *pgxpool.Pool, userEmail string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE lower(u.email) = lower($1)
		  AND s.revoked_at IS NULL
		  AND s.expires_at > now()
	`, userEmail).Scan(&n)
	if err != nil {
		t.Fatalf("counting active sessions for %s: %v", userEmail, err)
	}
	return n
}

// tokenIsActive returns true when Authenticate succeeds for the given raw token.
func tokenIsActive(svc *auth.Service, rawToken string) bool {
	_, _, err := svc.Authenticate(context.Background(), rawToken)
	return err == nil
}

// --- Tests ---

// Test 1: Login 3x, sesi pertama harus di-evict (401), dua lainnya tetap aktif.
func TestSessionEviction_ThirdLoginEjectFirst(t *testing.T) {
	pool := testDB(t)
	svc := newService(t, pool)

	email := fmt.Sprintf("evict-%d@test.local", time.Now().UnixNano())
	registerUser(t, svc, email, fmt.Sprintf("u_evict_%d", time.Now().UnixNano()%100000000), "Pass1234!")

	tok1 := loginUser(t, svc, email, "Pass1234!")
	tok2 := loginUser(t, svc, email, "Pass1234!")
	// Third login must evict token1
	tok3 := loginUser(t, svc, email, "Pass1234!")

	if tokenIsActive(svc, tok1) {
		t.Error("token1 (oldest) should have been evicted but is still active")
	}
	if !tokenIsActive(svc, tok2) {
		t.Error("token2 should still be active")
	}
	if !tokenIsActive(svc, tok3) {
		t.Error("token3 (newest) should be active")
	}

	active := countActiveSessions(t, pool, email)
	if active != 2 {
		t.Errorf("expected 2 active sessions, got %d", active)
	}
}

// Test 2: Login paralel tidak boleh menghasilkan lebih dari 2 sesi aktif.
func TestSessionEviction_ParallelLogins(t *testing.T) {
	pool := testDB(t)
	svc := newService(t, pool)

	email := fmt.Sprintf("parallel-%d@test.local", time.Now().UnixNano())
	registerUser(t, svc, email, fmt.Sprintf("u_par_%d", time.Now().UnixNano()%100000000), "Pass1234!")

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			ctx := context.Background()
			_, _, _, _ = svc.Login(ctx, email, "Pass1234!", "concurrent-agent", "10.0.0.1")
		}()
	}
	wg.Wait()

	// Wait briefly for all commits to settle
	time.Sleep(100 * time.Millisecond)

	active := countActiveSessions(t, pool, email)
	if active > 2 {
		t.Errorf("concurrent logins produced %d active sessions; must not exceed 2", active)
	}
}

// Test 3: ForceLogout mencabut semua sesi target, semua token menjadi 401.
func TestForceLogout_RevokesAllSessions(t *testing.T) {
	pool := testDB(t)
	svc := newService(t, pool)

	// Create target user
	email := fmt.Sprintf("forcetarget-%d@test.local", time.Now().UnixNano())
	registerUser(t, svc, email, fmt.Sprintf("u_force_%d", time.Now().UnixNano()%100000000), "Pass1234!")

	tok1 := loginUser(t, svc, email, "Pass1234!")

	// Fetch userID
	var targetID string
	err := pool.QueryRow(context.Background(), `SELECT id FROM users WHERE lower(email) = lower($1)`, email).Scan(&targetID)
	if err != nil {
		t.Fatalf("fetching user id: %v", err)
	}

	// Force logout
	if err := svc.ForceLogout(context.Background(), targetID); err != nil {
		t.Fatalf("force logout: %v", err)
	}

	if tokenIsActive(svc, tok1) {
		t.Error("tok1 should be invalid after force logout")
	}

	active := countActiveSessions(t, pool, email)
	if active != 0 {
		t.Errorf("expected 0 active sessions after force logout, got %d", active)
	}
}

// Test 4: ForceLogout dengan user ID tidak ada harus mengembalikan ErrNotFound.
func TestForceLogout_NonExistentUser_NotFound(t *testing.T) {
	pool := testDB(t)
	svc := newService(t, pool)

	err := svc.ForceLogout(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if !isNotFound(err) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

// Test 5: User biasa tidak bisa mengakses endpoint admin.
// (Diverifikasi lewat middleware RequireAdmin — unit test peran.)
func TestRequireAdmin_RejectNonAdmin(t *testing.T) {
	pool := testDB(t)
	svc := newService(t, pool)

	email := fmt.Sprintf("user-%d@test.local", time.Now().UnixNano())
	registerUser(t, svc, email, fmt.Sprintf("u_norm_%d", time.Now().UnixNano()%100000000), "Pass1234!")

	rawToken := loginUser(t, svc, email, "Pass1234!")

	user, _, err := svc.Authenticate(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}

	if user.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", user.Role)
	}

	// Middleware RequireAdmin rejects any role != "admin"
	if user.Role == "admin" {
		t.Error("non-admin user incorrectly has admin role")
	}
}

func isNotFound(err error) bool {
	return errors.Is(err, auth.ErrNotFound)
}
