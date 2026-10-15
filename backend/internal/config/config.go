package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL       string
	Port              string
	FrontendOrigin    string
	CookieSecure      bool
	SessionTTLHours   int
	MaxActiveSessions int
	AdminEmail        string
	AdminUsername     string
	AdminPassword     string

	// Argon2id parameters
	Argon2Memory      uint32
	Argon2Iterations  uint32
	Argon2Parallelism uint8
	Argon2SaltLength  uint32
	Argon2KeyLength   uint32
}

func Load() *Config {
	return &Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://app:change_me@localhost:5432/authdb?sslmode=disable"),
		Port:              getEnv("BACKEND_PORT", "8080"),
		FrontendOrigin:    getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
		CookieSecure:      getEnvBool("COOKIE_SECURE", false),
		SessionTTLHours:   getEnvInt("SESSION_TTL_HOURS", 168),
		MaxActiveSessions: getEnvInt("MAX_ACTIVE_SESSIONS", 2),
		AdminEmail:        getEnv("ADMIN_EMAIL", "admin@fti.com"),
		AdminUsername:     getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:     getEnv("ADMIN_PASSWORD", "FtiKki123!"),

		// Argon2id OWASP recommended defaults: 64MB memory, 3 iterations, 2 parallelism threads
		Argon2Memory:      uint32(getEnvInt("ARGON2_MEMORY", 64*1024)),
		Argon2Iterations:  uint32(getEnvInt("ARGON2_ITERATIONS", 3)),
		Argon2Parallelism: uint8(getEnvInt("ARGON2_PARALLELISM", 2)),
		Argon2SaltLength:  uint32(getEnvInt("ARGON2_SALT_LENGTH", 16)),
		Argon2KeyLength:   uint32(getEnvInt("ARGON2_KEY_LENGTH", 32)),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvBool(key string, defaultVal bool) bool {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
