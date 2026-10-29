package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/Arpan7125/jobpilot/apps/api/internal/auth"
	"github.com/Arpan7125/jobpilot/apps/api/internal/store"
)

type ctxKey struct{}

func userID(r *http.Request) string { return r.Context().Value(ctxKey{}).(string) }

// requireAuth rejects requests without a valid "Authorization: Bearer <jwt>"
// header and puts the user id on the request context.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		id, err := s.tokens.Verify(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
	})
}

type credentials struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Name     *string `json:"name"`
}

type authResponse struct {
	Token string     `json:"token"`
	User  store.User `json:"user"`
}

func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !decode(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	errs := fieldErrors{}
	if addr, err := mail.ParseAddress(in.Email); err != nil || addr.Address != in.Email || len(in.Email) > 254 {
		errs["email"] = "must be a valid email address"
	}
	// bcrypt only reads the first 72 bytes, so longer passwords are refused.
	if len(in.Password) < 8 || len(in.Password) > 72 {
		errs["password"] = "must be 8 to 72 characters"
	}
	in.Name = trimmed(in.Name)
	checkLen(errs, "name", in.Name, 100)
	if len(errs) > 0 {
		writeValidation(w, errs)
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		serverError(w, r, err)
		return
	}
	user, err := s.store.CreateUser(r.Context(), in.Email, hash, in.Name)
	if errors.Is(err, store.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "an account with that email already exists")
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	s.respondWithToken(w, r, http.StatusCreated, user)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !decode(w, r, &in) {
		return
	}
	user, err := s.store.UserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(in.Email)))
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		serverError(w, r, err)
		return
	}
	// Same message and timing for unknown email and wrong password.
	if !auth.CheckPassword(user.PasswordHash, in.Password) {
		writeError(w, http.StatusUnauthorized, "wrong email or password")
		return
	}
	s.respondWithToken(w, r, http.StatusOK, user)
}

func (s *Server) respondWithToken(w http.ResponseWriter, r *http.Request, status int, user store.User) {
	token, err := s.tokens.Issue(user.ID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, status, authResponse{Token: token, User: user})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user, err := s.store.UserByID(r.Context(), userID(r))
	if errors.Is(err, store.ErrNotFound) {
		// Valid token for a deleted account.
		writeError(w, http.StatusUnauthorized, "account no longer exists")
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}
