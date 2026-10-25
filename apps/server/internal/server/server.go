// Package server owns process lifecycle: building the handler, serving
// traffic and draining gracefully on SIGTERM/SIGINT.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/dkplus-pro/go-shell/apps/server/internal/config"
	"github.com/dkplus-pro/go-shell/apps/server/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// Server wraps an *http.Server with configuration-driven lifecycle.
type Server struct {
	cfg    config.Config
	logger *slog.Logger
	http   *http.Server
}

// New builds a Server. registry may be nil, in which case a fresh registry
// is created (useful for tests).
func New(cfg config.Config, logger *slog.Logger, registry *prometheus.Registry) *Server {
	if registry == nil {
		registry = metrics.NewRegistry()
	}
	return &Server{
		cfg:    cfg,
		logger: logger,
		http: &http.Server{
			Addr:              cfg.Addr(),
			Handler:           NewHandler(cfg, logger, registry),
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		},
	}
}

// Run blocks serving traffic until ctx is cancelled (SIGTERM/SIGINT), then
// drains in-flight requests for at most cfg.ShutdownTimeout.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("http server listening", "addr", s.http.Addr)
		err := s.http.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	s.logger.Info("shutdown signal received, draining connections",
		"timeout", s.cfg.ShutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("graceful shutdown failed", "error", err)
		return err
	}
	s.logger.Info("http server drained and stopped")
	return <-errCh
}
