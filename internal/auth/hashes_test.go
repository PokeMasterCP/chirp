package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "ex@mpl3P@ssw0rd"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if hash == "" {
		t.Fatal("expected a hash to be returned")
	}

	match, err := CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("expected no error checking hash, got %v", err)
	}
	if !match {
		t.Fatal("expected the generated hash to match the password")
	}
}
