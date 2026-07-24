package auth

import (
	"net/http"
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

func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		expected    string
		expectError bool
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer exampleToken",
			expected:   "exampleToken",
		},
		{
			name:        "missing authorization header",
			expectError: true,
		},
		{
			name:        "wrong authorization scheme",
			authHeader:  "Basic exampleToken",
			expectError: true,
		},
		{
			name:        "missing space after bearer scheme",
			authHeader:  "BearerexampleToken",
			expectError: true,
		},
		{
			name:        "missing token",
			authHeader:  "Bearer ",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}
			if tt.authHeader != "" {
				headers.Set("Authorization", tt.authHeader)
			}

			token, err := GetBearerToken(headers)
			if tt.expectError {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				if token != "" {
					t.Fatalf("expected an empty token, got %q", token)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if token != tt.expected {
				t.Fatalf("expected token %q, got %q", tt.expected, token)
			}
		})
	}
}
