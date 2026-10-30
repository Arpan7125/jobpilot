package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
	"github.com/Arpan7125/jobpilot/apps/api/internal/store"
)

// parseJob runs the posting parser on the job's description and stores the result.
func (s *Server) parseJob(w http.ResponseWriter, r *http.Request) {
	if s.parser == nil {
		writeError(w, http.StatusServiceUnavailable, "posting parser is not configured")
		return
	}
	uid, jobID := userID(r), chi.URLParam(r, "jobID")
	job, err := s.store.Job(r.Context(), uid, jobID)
	if err != nil {
		s.respondJob(w, r, job, err)
		return
	}

	start := time.Now()
	posting, call, err := s.parser.Parse(r.Context(), job.Description)
	s.logCall(r.Context(), uid, jobID, "parse", call, time.Since(start), err)
	switch {
	case errors.Is(err, agent.ErrRefused):
		writeError(w, http.StatusUnprocessableEntity, "the model declined to parse this posting")
		return
	case err != nil:
		slog.Error("posting parser failed", "job", jobID, "err", err)
		writeError(w, http.StatusBadGateway, "posting parser failed, try again")
		return
	}

	job, err = s.store.SetParsed(r.Context(), uid, jobID, posting)
	s.respondJob(w, r, job, err)
}

// logCall records a model request. A logging failure never fails the request.
func (s *Server) logCall(ctx context.Context, uid, jobID, feature string, call agent.Call, d time.Duration, callErr error) {
	c := store.ModelCall{
		UserID: uid, JobID: &jobID, Feature: feature, Model: call.Model,
		InputTokens: call.InputTokens, OutputTokens: call.OutputTokens,
		LatencyMS: int(d.Milliseconds()), OK: callErr == nil,
	}
	if callErr != nil {
		msg := callErr.Error()
		c.Error = &msg
	}
	if err := s.store.LogModelCall(context.WithoutCancel(ctx), c); err != nil {
		slog.Error("model call log failed", "err", err)
	}
	slog.Info("model call", "feature", feature, "model", call.Model, "input_tokens", call.InputTokens,
		"output_tokens", call.OutputTokens, "latency_ms", c.LatencyMS, "ok", c.OK)
}
