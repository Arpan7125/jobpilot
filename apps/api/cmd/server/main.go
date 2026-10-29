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

	origins := strings.Split(envOr("CORS_ORIGINS", "http://localhost:3000"), ",")
	srv := &http.Server{
		Addr:              ":" + envOr("PORT", "4000"),
		Handler:           httpapi.New(store.New(pool), tokens, origins).Handler(),
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
