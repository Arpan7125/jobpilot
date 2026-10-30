package store

import "context"

// ModelCall is one LLM request, logged for cost and failure tracking.
type ModelCall struct {
	UserID       string
	JobID        *string
	Feature      string
	Model        string
	InputTokens  int64
	OutputTokens int64
	LatencyMS    int
	OK           bool
	Error        *string
}

func (s *Store) LogModelCall(ctx context.Context, c ModelCall) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO model_calls (user_id, job_id, feature, model, input_tokens, output_tokens, latency_ms, ok, error)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		c.UserID, c.JobID, c.Feature, c.Model, c.InputTokens, c.OutputTokens, c.LatencyMS, c.OK, c.Error)
	return err
}
