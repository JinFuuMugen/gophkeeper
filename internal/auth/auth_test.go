package auth_test

import (
	"testing"
	"time"

	"github.com/JinFuuMugen/GophKeeper/internal/auth"
)

func TestTokenLifecycle(t *testing.T) {
	tok, err := auth.NewToken("userID123", "mysecret", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error from NewToken: %v", err)
	}
	claims, err := auth.ParseToken(tok, "mysecret")
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.UserID != "userID123" {
		t.Fatalf("expected userID 'userID123', got %s", claims.UserID)
	}
	if claims.RegisteredClaims.ExpiresAt.Time.Before(time.Now()) {
		t.Fatalf("token expiry should be in the future")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	tok, err := auth.NewToken("u1", "secret1", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error from NewToken: %v", err)
	}
	if _, err := auth.ParseToken(tok, "differentsecret"); err == nil {
		t.Fatalf("expected error when parsing with wrong secret")
	}
}

func TestParseToken_Expired(t *testing.T) {
	tok, err := auth.NewToken("u1", "secret", -time.Second)
	if err != nil {
		t.Fatalf("unexpected error from NewToken: %v", err)
	}
	if _, err := auth.ParseToken(tok, "secret"); err == nil {
		t.Fatalf("expected error for expired token")
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	pwd := "s3cR3t!"
	hash, err := auth.HashPassword(pwd)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == pwd {
		t.Fatalf("hash should not equal the plain password")
	}
	if !auth.CheckPassword(hash, pwd) {
		t.Fatalf("expected CheckPassword to return true for correct password")
	}
	if auth.CheckPassword(hash, "wrong") {
		t.Fatalf("expected CheckPassword to return false for wrong password")
	}
}
