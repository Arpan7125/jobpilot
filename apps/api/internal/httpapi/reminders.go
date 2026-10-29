package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Arpan7125/jobpilot/apps/api/internal/store"
)

func (s *Server) listReminders(w http.ResponseWriter, r *http.Request) {
	reminders, err := s.store.Reminders(r.Context(), userID(r))
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, reminders)
}

func (s *Server) dismissReminder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "reminderID")
	if !uuidPattern.MatchString(id) {
		writeError(w, http.StatusNotFound, "reminder not found")
		return
	}
	err := s.store.DismissReminder(r.Context(), userID(r), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "reminder not found")
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
