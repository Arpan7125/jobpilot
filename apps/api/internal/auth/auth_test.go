package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const secret = "test-secret-that-is-at-least-32-characters"

func TestTokenRoundTrip(t *testing.T) {
	tokens, err := NewTokens(secret)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := tokens.Issue("user-1")
	if err != nil {
		t.Fatal(err)
	}
	got, err := tokens.Verify(tok)
	if err != nil || got != "user-1" {
		t.Fatalf("Verify = %q, %v", got, err)
	}
}

func TestVerifyRejects(t *testing.T) {
	tokens, _ := NewTokens(secret)
	other, _ := NewTokens("a-different-secret-also-32-characters-long")
	foreign, _ := other.Issue("user-1")

	expired, _ := NewTokens(secret)
	expired.now = func() time.Time { return time.Now().Add(-48 * time.Hour) }
	old, _ := expired.Issue("user-1")

	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject: "user-1", Issuer: "jobpilot",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)

	for name, tok := range map[string]string{
		"wrong secret": foreign,
		"expired":      old,
		"alg none":     none,
		"garbage":      "not.a.token",
	} {
		if _, err := tokens.Verify(tok); err == nil {
			t.Errorf("%s: Verify accepted the token", name)
		}
	}
}

func TestShortSecretRejected(t *testing.T) {
	if _, err := NewTokens("short"); err == nil {
		t.Fatal("NewTokens accepted a short secret")
	}
}

func TestPasswords(t *testing.T) {
	h, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(h, "correct horse") {
		t.Error("right password rejected")
	}
	if CheckPassword(h, "wrong") {
		t.Error("wrong password accepted")
	}
	if CheckPassword("", "anything") {
		t.Error("empty hash accepted")
	}
}
