// Package auth hashes passwords and issues and checks JWTs.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = 24 * time.Hour

// dummyHash is compared against when a login email does not exist, so a
// missing account takes as long to reject as a wrong password.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)

func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(h), err
}

// CheckPassword reports whether password matches hash. Pass an empty hash for
// an unknown user to spend the same time as a real comparison.
func CheckPassword(hash, password string) bool {
	if hash == "" {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

type Tokens struct {
	secret []byte
	now    func() time.Time
}

func NewTokens(secret string) (*Tokens, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 characters")
	}
	return &Tokens{secret: []byte(secret), now: time.Now}, nil
}

// Issue returns a signed token whose subject is the user id.
func (t *Tokens) Issue(userID string) (string, error) {
	now := t.now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		Issuer:    "jobpilot",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

// Verify checks the signature, algorithm, issuer and expiry, and returns the user id.
func (t *Tokens) Verify(token string) (string, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer("jobpilot"),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}
	if claims.Subject == "" {
		return "", errors.New("invalid token: no subject")
	}
	return claims.Subject, nil
}
