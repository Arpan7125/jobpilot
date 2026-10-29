// Package store holds the SQL for users, jobs and applications.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrEmailTaken = errors.New("email already registered")
	ErrSameStatus = errors.New("application already has that status")
)

// Statuses lists every application status in funnel order.
var Statuses = []string{"saved", "applied", "replied", "interview", "offer", "rejected", "withdrawn"}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         *string   `json:"name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string, name *string) (User, error) {
	u := User{Email: email, Name: name, PasswordHash: passwordHash}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		email, passwordHash, name,
	).Scan(&u.ID, &u.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return User{}, ErrEmailTaken
	}
	return u, err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	return s.scanUser(s.pool.QueryRow(ctx,
		`SELECT id, email, name, password_hash, created_at FROM users WHERE email = $1`, email))
}

func (s *Store) UserByID(ctx context.Context, id string) (User, error) {
	return s.scanUser(s.pool.QueryRow(ctx,
		`SELECT id, email, name, password_hash, created_at FROM users WHERE id = $1`, id))
}

func (s *Store) scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}
