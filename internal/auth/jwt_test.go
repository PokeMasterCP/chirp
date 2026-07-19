package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("expected no error creating token, got %v", err)
	}

	if token == "" {
		t.Fatal("expected a token to be returned")
	}

	validatedUserID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("expected token to be valid, got %v", err)
	}

	if validatedUserID != userID {
		t.Fatalf("expected user ID %v, got %v", userID, validatedUserID)
	}
}

func TestValidateJWTRejectsWrongSecret(t *testing.T) {
	userID := uuid.New()

	token, err := MakeJWT(userID, "correct-secret", time.Hour)
	if err != nil {
		t.Fatalf("could not create test token: %v", err)
	}

	validatedUserID, err := ValidateJWT(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected validation to fail with the wrong secret")
	}

	if validatedUserID != uuid.Nil {
		t.Fatalf("expected uuid.Nil, got %v", validatedUserID)
	}
}

func TestValidateJWTRejectsExpiredToken(t *testing.T) {
	userID := uuid.New()

	// A negative duration makes the token expired immediately.
	token, err := MakeJWT(userID, "test-secret", -time.Minute)
	if err != nil {
		t.Fatalf("could not create test token: %v", err)
	}

	validatedUserID, err := ValidateJWT(token, "test-secret")
	if err == nil {
		t.Fatal("expected expired token to be rejected")
	}

	if validatedUserID != uuid.Nil {
		t.Fatalf("expected uuid.Nil, got %v", validatedUserID)
	}
}
