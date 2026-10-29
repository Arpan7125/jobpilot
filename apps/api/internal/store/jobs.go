package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// JobFields are the posting details a user can set.
type JobFields struct {
	Company     string  `json:"company"`
	Title       string  `json:"title"`
	Location    *string `json:"location"`
	URL         *string `json:"url"`
	Platform    *string `json:"platform"`
	Description string  `json:"description"`
	SalaryMin   *int    `json:"salaryMin"`
	SalaryMax   *int    `json:"salaryMax"`
	Currency    *string `json:"currency"`
}

// Job is a posting together with its application.
type Job struct {
	ID string `json:"id"`
	JobFields
	Status          string     `json:"status"`
	AppliedAt       *time.Time `json:"appliedAt"`
	StatusChangedAt time.Time  `json:"statusChangedAt"`
	Notes           *string    `json:"notes"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type StatusChange struct {
	FromStatus *string   `json:"fromStatus"`
	ToStatus   string    `json:"toStatus"`
	ChangedAt  time.Time `json:"changedAt"`
	Note       *string   `json:"note"`
}

const jobColumns = `j.id, j.company, j.title, j.location, j.url, j.platform, j.description,
	j.salary_min, j.salary_max, j.currency,
	a.status, a.applied_at, a.status_changed_at, a.notes, j.created_at, j.updated_at`

const jobFrom = ` FROM jobs j JOIN applications a ON a.job_id = j.id `

func scanJob(row pgx.Row) (Job, error) {
	var j Job
	err := row.Scan(&j.ID, &j.Company, &j.Title, &j.Location, &j.URL, &j.Platform, &j.Description,
		&j.SalaryMin, &j.SalaryMax, &j.Currency,
		&j.Status, &j.AppliedAt, &j.StatusChangedAt, &j.Notes, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	return j, err
}

// CreateJob saves the posting, its application (status saved) and the first
// history row in one transaction.
func (s *Store) CreateJob(ctx context.Context, userID string, f JobFields, notes *string) (Job, error) {
	var id string
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`INSERT INTO jobs (user_id, company, title, location, url, platform, description,
			                   salary_min, salary_max, currency)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`,
			userID, f.Company, f.Title, f.Location, f.URL, f.Platform, f.Description,
			f.SalaryMin, f.SalaryMax, f.Currency,
		).Scan(&id)
		if err != nil {
			return err
		}
		var appID string
		if err := tx.QueryRow(ctx,
			`INSERT INTO applications (job_id, notes) VALUES ($1, $2) RETURNING id`, id, notes,
		).Scan(&appID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO application_status_history (application_id, to_status) VALUES ($1, 'saved')`, appID)
		return err
	})
	if err != nil {
		return Job{}, err
	}
	return s.Job(ctx, userID, id)
}

// Job returns one of the user's jobs. Another user's job is ErrNotFound.
func (s *Store) Job(ctx context.Context, userID, jobID string) (Job, error) {
	return scanJob(s.pool.QueryRow(ctx,
		`SELECT `+jobColumns+jobFrom+`WHERE j.id = $1 AND j.user_id = $2`, jobID, userID))
}

// ListJobs returns the user's jobs, newest first, optionally filtered by status.
func (s *Store) ListJobs(ctx context.Context, userID string, status *string) ([]Job, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+jobColumns+jobFrom+`
		 WHERE j.user_id = $1 AND ($2::application_status IS NULL OR a.status = $2)
		 ORDER BY j.created_at DESC`, userID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []Job{}
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// UpdateJob replaces the posting fields and notes of one of the user's jobs.
func (s *Store) UpdateJob(ctx context.Context, userID, jobID string, f JobFields, notes *string) (Job, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE jobs SET company = $3, title = $4, location = $5, url = $6, platform = $7,
			        description = $8, salary_min = $9, salary_max = $10, currency = $11, updated_at = now()
			 WHERE id = $1 AND user_id = $2`,
			jobID, userID, f.Company, f.Title, f.Location, f.URL, f.Platform, f.Description,
			f.SalaryMin, f.SalaryMax, f.Currency)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		_, err = tx.Exec(ctx,
			`UPDATE applications SET notes = $2, updated_at = now() WHERE job_id = $1`, jobID, notes)
		return err
	})
	if err != nil {
		return Job{}, err
	}
	return s.Job(ctx, userID, jobID)
}

// SetStatus moves the application to a new status and records the change.
// applied_at is set the first time the application reaches 'applied'.
func (s *Store) SetStatus(ctx context.Context, userID, jobID, status string, note *string) (Job, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var appID, current string
		err := tx.QueryRow(ctx,
			`SELECT a.id, a.status FROM applications a JOIN jobs j ON j.id = a.job_id
			 WHERE j.id = $1 AND j.user_id = $2 FOR UPDATE OF a`, jobID, userID,
		).Scan(&appID, &current)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if current == status {
			return ErrSameStatus
		}
		if _, err := tx.Exec(ctx,
			`UPDATE applications
			 SET status = $2::application_status, status_changed_at = now(), updated_at = now(),
			     applied_at = CASE WHEN $2::application_status = 'applied' THEN COALESCE(applied_at, now()) ELSE applied_at END
			 WHERE id = $1`, appID, status); err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO application_status_history (application_id, from_status, to_status, note)
			 VALUES ($1, $2, $3, $4)`, appID, current, status, note)
		return err
	})
	if err != nil {
		return Job{}, err
	}
	return s.Job(ctx, userID, jobID)
}

// History returns a job's status changes, oldest first.
func (s *Store) History(ctx context.Context, userID, jobID string) ([]StatusChange, error) {
	if _, err := s.Job(ctx, userID, jobID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT h.from_status, h.to_status, h.changed_at, h.note
		 FROM application_status_history h JOIN applications a ON a.id = h.application_id
		 WHERE a.job_id = $1 ORDER BY h.changed_at, h.id`, jobID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (StatusChange, error) {
		var c StatusChange
		err := r.Scan(&c.FromStatus, &c.ToStatus, &c.ChangedAt, &c.Note)
		return c, err
	})
}

func (s *Store) DeleteJob(ctx context.Context, userID, jobID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM jobs WHERE id = $1 AND user_id = $2`, jobID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
