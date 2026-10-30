package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Profile is what the user brings: the other side of the fit score.
type Profile struct {
	Skills          []string   `json:"skills"`
	ExperienceYears *int       `json:"experienceYears"`
	Locations       []string   `json:"locations"`
	WorkModes       []string   `json:"workModes"`
	MinPay          *int       `json:"minPay"`
	PayCurrency     *string    `json:"payCurrency"`
	PayPeriod       *string    `json:"payPeriod"`
	UpdatedAt       *time.Time `json:"updatedAt"` // null until first saved
}

// Profile returns the user's profile, or an empty one if they haven't saved it.
func (s *Store) Profile(ctx context.Context, userID string) (Profile, error) {
	p := Profile{Skills: []string{}, Locations: []string{}, WorkModes: []string{}}
	err := s.pool.QueryRow(ctx,
		`SELECT skills, experience_years, locations, work_modes, min_pay, pay_currency, pay_period, updated_at
		 FROM profiles WHERE user_id = $1`, userID,
	).Scan(&p.Skills, &p.ExperienceYears, &p.Locations, &p.WorkModes, &p.MinPay, &p.PayCurrency, &p.PayPeriod, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil
	}
	return p, err
}

// SaveProfile creates or replaces the user's profile.
func (s *Store) SaveProfile(ctx context.Context, userID string, p Profile) (Profile, error) {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO profiles (user_id, skills, experience_years, locations, work_modes, min_pay, pay_currency, pay_period)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (user_id) DO UPDATE SET
		   skills = $2, experience_years = $3, locations = $4, work_modes = $5,
		   min_pay = $6, pay_currency = $7, pay_period = $8, updated_at = now()`,
		userID, p.Skills, p.ExperienceYears, p.Locations, p.WorkModes, p.MinPay, p.PayCurrency, p.PayPeriod)
	if err != nil {
		return Profile{}, err
	}
	return s.Profile(ctx, userID)
}
