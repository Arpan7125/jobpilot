// Package httpapi is the JobPilot REST API.
package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
	"github.com/Arpan7125/jobpilot/apps/api/internal/auth"
	"github.com/Arpan7125/jobpilot/apps/api/internal/store"
)

type Server struct {
	store       *store.Store
	tokens      *auth.Tokens
	corsOrigins []string
	parser      agent.Parser // nil when ANTHROPIC_API_KEY is not set
}

func New(s *store.Store, t *auth.Tokens, corsOrigins []string, parser agent.Parser) *Server {
	return &Server{store: s, tokens: t, corsOrigins: corsOrigins, parser: parser}
}

// Model calls take longer than plain CRUD, so they get their own deadline.
const (
	requestTimeout = 15 * time.Second
	modelTimeout   = 90 * time.Second
)

func timeouts(next http.Handler) http.Handler {
	short, long := middleware.Timeout(requestTimeout)(next), middleware.Timeout(modelTimeout)(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/parse") {
			long.ServeHTTP(w, r)
			return
		}
		short.ServeHTTP(w, r)
	})
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	r.Use(timeouts)
	r.Use(s.cors)

	r.Get("/health", s.health)
	r.Post("/auth/signup", s.signup)
	r.Post("/auth/login", s.login)

	// Everything below needs a valid token.
	r.Group(func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Get("/me", s.me)
		r.Get("/reminders", s.listReminders)
		r.Post("/reminders/{reminderID}/dismiss", s.dismissReminder)
		r.Route("/jobs", func(r chi.Router) {
			r.Get("/", s.listJobs)
			r.Post("/", s.createJob)
			r.Route("/{jobID}", func(r chi.Router) {
				r.Use(validJobID)
				r.Get("/", s.getJob)
				r.Put("/", s.updateJob)
				r.Delete("/", s.deleteJob)
				r.Put("/status", s.setStatus)
				r.Get("/history", s.history)
				r.Post("/parse", s.parseJob)
			})
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) { writeError(w, http.StatusNotFound, "not found") })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	})
	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "db": "unreachable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "db": "ok"})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		for _, allowed := range s.corsOrigins {
			if origin != "" && strings.EqualFold(origin, allowed) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Add("Vary", "Origin")
				break
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
