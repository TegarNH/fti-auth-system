package auth

import (
	"bytes"
	"testing"
)

func TestGenerateSessionToken(t *testing.T) {
	token1, hash1, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken failed: %v", err)
	}
	if len(token1) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(token1))
	}
	if len(hash1) != 32 {
		t.Fatalf("expected 32 bytes SHA-256 hash, got %d", len(hash1))
	}

	token2, hash2, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken second call failed: %v", err)
	}
	if token1 == token2 {
		t.Fatal("expected different random tokens across calls")
	}
	if bytes.Equal(hash1, hash2) {
		t.Fatal("expected different hashes across calls")
	}

	calculatedHash := HashToken(token1)
	if !bytes.Equal(hash1, calculatedHash) {
		t.Fatal("HashToken did not match hash from GenerateSessionToken")
	}
}
