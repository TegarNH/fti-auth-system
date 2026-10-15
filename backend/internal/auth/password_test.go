package auth

import (
	"testing"
)

func TestHashAndVerifySuccess(t *testing.T) {
	// Use lighter parameters for fast unit tests
	testParams := &Argon2Params{
		Memory:      16 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	password := "SecretPassword123!"
	encodedHash, err := HashPassword(password, testParams)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if encodedHash == "" {
		t.Fatal("expected non-empty hash string")
	}

	match, err := VerifyPassword(password, encodedHash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !match {
		t.Fatal("expected password to verify successfully")
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	testParams := &Argon2Params{
		Memory:      16 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	password := "CorrectPassword123!"
	wrongPassword := "WrongPassword456!"

	encodedHash, err := HashPassword(password, testParams)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	match, err := VerifyPassword(wrongPassword, encodedHash)
	if err != nil {
		t.Fatalf("unexpected error during verification: %v", err)
	}
	if match {
		t.Fatal("expected verification to fail for wrong password")
	}
}

func TestVerifyInvalidHashFormat(t *testing.T) {
	invalidHashes := []string{
		"",
		"plain-string",
		"$bcrypt$v=19$...",
		"$argon2id$v=99$m=1024,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$invalidparams$c2FsdA$aGFzaA",
	}

	for _, ih := range invalidHashes {
		match, err := VerifyPassword("anypassword", ih)
		if err == nil && match {
			t.Errorf("expected error or match=false for invalid hash %q, got match=%v, err=%v", ih, match, err)
		}
	}
}

func TestDefaultParameters(t *testing.T) {
	password := "MyP@ssw0rd!"
	encodedHash, err := HashPassword(password, nil)
	if err != nil {
		t.Fatalf("HashPassword with default params failed: %v", err)
	}

	match, err := VerifyPassword(password, encodedHash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !match {
		t.Fatal("expected password to match with default parameters")
	}
}
