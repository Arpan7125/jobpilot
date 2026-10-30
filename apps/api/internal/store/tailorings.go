package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Tailoring is one set of resume suggestions for a job. The JSON columns are
// owned by the tailor package; the store only keeps them.
type Tailoring struct {
	ID          string
	JobID       string
	Resume      string
	Suggestions json.RawMessage
	Rejected    json.RawMessage
	Relevant    json.RawMessage
	CreatedAt   time.Time
}

func (s *Store) CreateTailoring(ctx context.Context, userID, jobID, resume string, suggestions, rejected, relevant any) (Tailoring, error) {
	sug, _ := json.Marshal(suggestions)
	rej, _ := json.Marshal(rejected)
	rel, _ := json.Marshal(relevant)
	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO tailorings (job_id, resume, suggestions, rejected, relevant)
		 SELECT j.id, $3, $4, $5, $6 FROM jobs j WHERE j.id = $1 AND j.user_id = $2
		 RETURNING id`, jobID, userID, resume, sug, rej, rel).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tailoring{}, ErrNotFound
	}
	if err != nil {
		return Tailoring{}, err
	}
	return s.LatestTailoring(ctx, userID, jobID)
}

// LatestTailoring returns the newest tailoring for one of the user's jobs.
func (s *Store) LatestTailoring(ctx context.Context, userID, jobID string) (Tailoring, error) {
	var t Tailoring
	err := s.pool.QueryRow(ctx,
		`SELECT t.id, t.job_id, t.resume, t.suggestions, t.rejected, t.relevant, t.created_at
		 FROM tailorings t JOIN jobs j ON j.id = t.job_id
		 WHERE t.job_id = $1 AND j.user_id = $2
		 ORDER BY t.created_at DESC LIMIT 1`, jobID, userID,
	).Scan(&t.ID, &t.JobID, &t.Resume, &t.Suggestions, &t.Rejected, &t.Relevant, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tailoring{}, ErrNotFound
	}
	return t, err
}

// SetSuggestions replaces the suggestions (with their statuses) of the job's
// latest tailoring.
func (s *Store) SetSuggestions(ctx context.Context, userID, tailoringID string, suggestions any) error {
	raw, _ := json.Marshal(suggestions)
	tag, err := s.pool.Exec(ctx,
		`UPDATE tailorings t SET suggestions = $3
		 FROM jobs j WHERE j.id = t.job_id AND t.id = $1 AND j.user_id = $2`, tailoringID, userID, raw)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
