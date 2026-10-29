package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/Arpan7125/jobpilot/apps/api/internal/store"
)

var (
	uuidPattern     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
)

// validJobID turns a malformed {jobID} into a 404 before it reaches Postgres.
func validJobID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !uuidPattern.MatchString(chi.URLParam(r, "jobID")) {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// jobInput is the body of POST /jobs and PUT /jobs/{id}. PUT replaces every
// field, so an omitted optional field is cleared.
type jobInput struct {
	store.JobFields
	Notes *string `json:"notes"`
}

func (in *jobInput) validate() fieldErrors {
	f := &in.JobFields
	f.Company = strings.TrimSpace(f.Company)
	f.Title = strings.TrimSpace(f.Title)
	f.Description = strings.TrimSpace(f.Description)
	f.Location, f.URL, f.Platform, f.Currency = trimmed(f.Location), trimmed(f.URL), trimmed(f.Platform), trimmed(f.Currency)
	in.Notes = trimmed(in.Notes)

	errs := fieldErrors{}
	required(errs, "company", f.Company, 200)
	required(errs, "title", f.Title, 200)
	required(errs, "description", f.Description, 50_000)
	checkLen(errs, "location", f.Location, 200)
	checkLen(errs, "platform", f.Platform, 100)
	checkLen(errs, "notes", in.Notes, 10_000)
	if f.URL != nil {
		u, err := url.Parse(*f.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(*f.URL) > 2000 {
			errs["url"] = "must be an http or https URL"
		}
	}
	if f.SalaryMin != nil && *f.SalaryMin < 0 {
		errs["salaryMin"] = "must not be negative"
	}
	if f.SalaryMax != nil && *f.SalaryMax < 0 {
		errs["salaryMax"] = "must not be negative"
	}
	if f.SalaryMin != nil && f.SalaryMax != nil && *f.SalaryMin > *f.SalaryMax {
		errs["salaryMax"] = "must be at least salaryMin"
	}
	if f.Currency != nil {
		upper := strings.ToUpper(*f.Currency)
		f.Currency = &upper
		if !currencyPattern.MatchString(upper) {
			errs["currency"] = "must be a 3-letter code such as INR or USD"
		}
	}
	return errs
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	var status *string
	if v := r.URL.Query().Get("status"); v != "" {
		if !slices.Contains(store.Statuses, v) {
			writeValidation(w, fieldErrors{"status": statusHint})
			return
		}
		status = &v
	}
	jobs, err := s.store.ListJobs(r.Context(), userID(r), status)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	var in jobInput
	if !decode(w, r, &in) {
		return
	}
	if errs := in.validate(); len(errs) > 0 {
		writeValidation(w, errs)
		return
	}
	job, err := s.store.CreateJob(r.Context(), userID(r), in.JobFields, in.Notes)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	job, err := s.store.Job(r.Context(), userID(r), chi.URLParam(r, "jobID"))
	s.respondJob(w, r, job, err)
}

func (s *Server) updateJob(w http.ResponseWriter, r *http.Request) {
	var in jobInput
	if !decode(w, r, &in) {
		return
	}
	if errs := in.validate(); len(errs) > 0 {
		writeValidation(w, errs)
		return
	}
	job, err := s.store.UpdateJob(r.Context(), userID(r), chi.URLParam(r, "jobID"), in.JobFields, in.Notes)
	s.respondJob(w, r, job, err)
}

const statusHint = "must be one of saved, applied, replied, interview, offer, rejected, withdrawn"

func (s *Server) setStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string  `json:"status"`
		Note   *string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Status = strings.ToLower(strings.TrimSpace(in.Status))
	in.Note = trimmed(in.Note)
	errs := fieldErrors{}
	if !slices.Contains(store.Statuses, in.Status) {
		errs["status"] = statusHint
	}
	checkLen(errs, "note", in.Note, 2_000)
	if len(errs) > 0 {
		writeValidation(w, errs)
		return
	}
	job, err := s.store.SetStatus(r.Context(), userID(r), chi.URLParam(r, "jobID"), in.Status, in.Note)
	if errors.Is(err, store.ErrSameStatus) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	s.respondJob(w, r, job, err)
}

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	changes, err := s.store.History(r.Context(), userID(r), chi.URLParam(r, "jobID"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, changes)
}

func (s *Server) deleteJob(w http.ResponseWriter, r *http.Request) {
	err := s.store.DeleteJob(r.Context(), userID(r), chi.URLParam(r, "jobID"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// respondJob writes a job, a 404 for a missing or someone else's job, or a 500.
func (s *Server) respondJob(w http.ResponseWriter, r *http.Request, job store.Job, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "job not found")
	case err != nil:
		serverError(w, r, err)
	default:
		writeJSON(w, http.StatusOK, job)
	}
}

// trimmed trims an optional string and turns an empty one into nil.
func trimmed(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func required(errs fieldErrors, field, v string, max int) {
	switch {
	case v == "":
		errs[field] = "is required"
	case utf8.RuneCountInString(v) > max:
		errs[field] = "is too long"
	}
}

func checkLen(errs fieldErrors, field string, v *string, max int) {
	if v != nil && utf8.RuneCountInString(*v) > max {
		errs[field] = "is too long"
	}
}
