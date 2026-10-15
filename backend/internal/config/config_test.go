package config

import (
	"os"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	// Clear relevant env vars
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("BACKEND_PORT")
	os.Unsetenv("MAX_ACTIVE_SESSIONS")
	os.Unsetenv("COOKIE_SECURE")

	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.MaxActiveSessions != 2 {
		t.Errorf("expected default max active sessions 2, got %d", cfg.MaxActiveSessions)
	}
	if cfg.CookieSecure != false {
		t.Errorf("expected default cookie secure false, got %v", cfg.CookieSecure)
	}
}

func TestConfigCustomEnv(t *testing.T) {
	os.Setenv("BACKEND_PORT", "9090")
	os.Setenv("COOKIE_SECURE", "true")
	os.Setenv("MAX_ACTIVE_SESSIONS", "3")
	defer func() {
		os.Unsetenv("BACKEND_PORT")
		os.Unsetenv("COOKIE_SECURE")
		os.Unsetenv("MAX_ACTIVE_SESSIONS")
	}()

	cfg := Load()
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.CookieSecure != true {
		t.Errorf("expected cookie secure true, got %v", cfg.CookieSecure)
	}
	if cfg.MaxActiveSessions != 3 {
		t.Errorf("expected max active sessions 3, got %d", cfg.MaxActiveSessions)
	}
}
