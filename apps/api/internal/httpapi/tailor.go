package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
	"github.com/Arpan7125/jobpilot/apps/api/internal/store"
	"github.com/Arpan7125/jobpilot/apps/api/internal/tailor"
)

// tailoringView is what the approval screen shows.
type tailoringView struct {
	ID          string              `json:"id"`
	CreatedAt   time.Time           `json:"createdAt"`
	Suggestions []tailor.Suggestion `json:"suggestions"`
	Dropped     []tailor.Dropped    `json:"dropped"` // refused by the guards, with why
	Relevant    []string            `json:"relevant"`
	// Tailored is the resume with the approved suggestions applied.
	Tailored string `json:"tailored"`
	// Stale is true when the profile resume changed after these suggestions were made.
	Stale bool `json:"stale"`
}

func toView(t store.Tailoring, currentResume string) (tailoringView, error) {
	v := tailoringView{ID: t.ID, CreatedAt: t.CreatedAt, Stale: t.Resume != currentResume}
	if err := json.Unmarshal(t.Suggestions, &v.Suggestions); err != nil {
		return v, err
	}
	if err := json.Unmarshal(t.Rejected, &v.Dropped); err != nil {
		return v, err
	}
	if err := json.Unmarshal(t.Relevant, &v.Relevant); err != nil {
		return v, err
	}
	v.Tailored = tailor.Apply(t.Resume, v.Suggestions)
	return v, nil
}

// tailorJob asks the model for rewordings of the user's resume for this job,
// runs them through the guards and stores them as pending.
func (s *Server) tailorJob(w http.ResponseWriter, r *http.Request) {
	if s.tailorer == nil {
		writeError(w, http.StatusServiceUnavailable, "resume tailoring is not configured")
		return
	}
	uid, jobID := userID(r), chi.URLParam(r, "jobID")
	job, err := s.store.Job(r.Context(), uid, jobID)
	if err != nil {
		s.respondJob(w, r, job, err)
		return
	}
	prof, err := s.store.Profile(r.Context(), uid)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if prof.Resume == "" {
		writeError(w, http.StatusUnprocessableEntity, "add your resume on the profile page first")
		return
	}
	var skills []string
	if len(job.Parsed) > 0 {
		var p agent.Posting
		if json.Unmarshal(job.Parsed, &p) == nil {
			skills = append(p.RequiredSkills, p.PreferredSkills...)
		}
	}

	start := time.Now()
	res, call, err := s.tailorer.Tailor(r.Context(), prof.Resume, job.Description, skills)
	s.logCall(r.Context(), uid, jobID, "tailor", call, time.Since(start), err)
	switch {
	case errors.Is(err, agent.ErrRefused):
		writeError(w, http.StatusUnprocessableEntity, "the model declined to tailor for this posting")
		return
	case err != nil:
		slog.Error("tailoring failed", "job", jobID, "err", err)
		writeError(w, http.StatusBadGateway, "tailoring failed, try again")
		return
	}

	kept, dropped, relevant := tailor.Guard(prof.Resume, res, skills)
	t, err := s.store.CreateTailoring(r.Context(), uid, jobID, prof.Resume, kept, dropped, relevant)
	if err != nil {
		serverError(w, r, err)
		return
	}
	v, err := toView(t, prof.Resume)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

// getTailoring returns the job's latest tailoring, or null if there is none.
func (s *Server) getTailoring(w http.ResponseWriter, r *http.Request) {
	uid, jobID := userID(r), chi.URLParam(r, "jobID")
	if _, err := s.store.Job(r.Context(), uid, jobID); err != nil {
		s.respondJob(w, r, store.Job{}, err)
		return
	}
	s.writeLatestTailoring(w, r, uid, jobID)
}

func (s *Server) writeLatestTailoring(w http.ResponseWriter, r *http.Request, uid, jobID string) {
	t, err := s.store.LatestTailoring(r.Context(), uid, jobID)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	prof, err := s.store.Profile(r.Context(), uid)
	if err != nil {
		serverError(w, r, err)
		return
	}
	v, err := toView(t, prof.Resume)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// decideSuggestion approves or rejects one suggestion of the latest tailoring.
func (s *Server) decideSuggestion(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !slices.Contains([]string{tailor.Pending, tailor.Approved, tailor.Rejected}, in.Status) {
		writeValidation(w, fieldErrors{"status": "must be pending, approved or rejected"})
		return
	}
	uid, jobID := userID(r), chi.URLParam(r, "jobID")
	t, err := s.store.LatestTailoring(r.Context(), uid, jobID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "no tailoring for this job")
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	var sugs []tailor.Suggestion
	if err := json.Unmarshal(t.Suggestions, &sugs); err != nil {
		serverError(w, r, err)
		return
	}
	n, err := strconv.Atoi(chi.URLParam(r, "n"))
	if err != nil || n < 0 || n >= len(sugs) {
		writeError(w, http.StatusNotFound, "suggestion not found")
		return
	}
	sugs[n].Status = in.Status
	if err := s.store.SetSuggestions(r.Context(), uid, t.ID, sugs); err != nil {
		serverError(w, r, err)
		return
	}
	s.writeLatestTailoring(w, r, uid, jobID)
}
