// Command server runs the JobPilot API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Arpan7125/jobpilot/apps/api/internal/agent"
	"github.com/Arpan7125/jobpilot/apps/api/internal/auth"
	"github.com/Arpan7125/jobpilot/apps/api/internal/db"
	"github.com/Arpan7125/jobpilot/apps/api/internal/httpapi"
	"github.com/Arpan7125/jobpilot/apps/api/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is not set")
	}
	tokens, err := auth.NewTokens(os.Getenv("JWT_SECRET"))
	if err != nil {
		return err
	}
	pool, err := db.Open(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}

	st := store.New(pool)
	go flagFollowUps(ctx, st, 24*time.Hour)

	// The AI features are optional: without a key the rest of the API still works.
	var parser agent.Parser
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		parser = agent.NewClaudeParser()
	} else {
		slog.Warn("ANTHROPIC_API_KEY is not set, posting parser disabled")
	}

	origins := strings.Split(envOr("CORS_ORIGINS", "http://localhost:3000"), ",")
	srv := &http.Server{
		Addr:              ":" + envOr("PORT", "4000"),
		Handler:           httpapi.New(st, tokens, origins, parser).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("JobPilot API listening", "addr", srv.Addr)
		errc <- srv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// flagFollowUps runs the follow-up reminder job at startup and then every
// interval. The job is idempotent, so restarts and overlapping runs are harmless.
func flagFollowUps(ctx context.Context, st *store.Store, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		n, err := st.FlagStaleApplications(ctx, store.FollowUpAfter)
		if err != nil {
			slog.Error("follow-up job failed", "err", err)
		} else {
			slog.Info("follow-up job finished", "new_reminders", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
