package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// FollowUpAfter is how long an application can sit in 'applied' before it
// gets a follow-up reminder.
const FollowUpAfter = 7 * 24 * time.Hour

type Reminder struct {
	ID        string    `json:"id"`
	JobID     string    `json:"jobId"`
	Company   string    `json:"company"`
	Title     string    `json:"title"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
}

// FlagStaleApplications adds a reminder to every application that has been
// 'applied' for longer than olderThan. It is safe to run repeatedly: an
// application gets at most one reminder per stay in 'applied', so a reminder
// you dismissed does not come back until the status changes again.
func (s *Store) FlagStaleApplications(ctx context.Context, olderThan time.Duration) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO reminders (application_id, reason)
		 SELECT a.id, $2
		 FROM applications a
		 WHERE a.status = 'applied'
		   AND a.status_changed_at < now() - $1::interval
		   AND NOT EXISTS (
		       SELECT 1 FROM reminders r
		       WHERE r.application_id = a.id AND r.created_at >= a.status_changed_at)
		 ON CONFLICT (application_id) WHERE dismissed_at IS NULL DO NOTHING`,
		olderThan, fmt.Sprintf("No reply %d days after applying. Time to follow up.", int(olderThan.Hours()/24)))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Reminders returns the user's open reminders, oldest first.
func (s *Store) Reminders(ctx context.Context, userID string) ([]Reminder, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT r.id, j.id, j.company, j.title, r.reason, r.created_at
		 FROM reminders r
		 JOIN applications a ON a.id = r.application_id
		 JOIN jobs j ON j.id = a.job_id
		 WHERE j.user_id = $1 AND r.dismissed_at IS NULL
		 ORDER BY r.created_at`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Reminder, error) {
		var rm Reminder
		err := r.Scan(&rm.ID, &rm.JobID, &rm.Company, &rm.Title, &rm.Reason, &rm.CreatedAt)
		return rm, err
	})
}

// DismissReminder closes one of the user's open reminders.
func (s *Store) DismissReminder(ctx context.Context, userID, reminderID string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE reminders r SET dismissed_at = now()
		 FROM applications a JOIN jobs j ON j.id = a.job_id
		 WHERE r.id = $1 AND r.application_id = a.id AND j.user_id = $2 AND r.dismissed_at IS NULL`,
		reminderID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
