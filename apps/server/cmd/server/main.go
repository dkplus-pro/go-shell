// Command server runs the go-shell HTTP backend.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/dkplus-pro/go-shell/apps/server/internal/config"
	"github.com/dkplus-pro/go-shell/apps/server/internal/metrics"
	"github.com/dkplus-pro/go-shell/apps/server/internal/server"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
	slog.Info("server exited cleanly")
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))
	slog.SetDefault(logger)

	// SIGTERM/SIGINT trigger graceful drain inside Server.Run.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.New(cfg, logger, metrics.NewRegistry())
	logger.Info("starting server",
		"service", cfg.ServiceName,
		"env", cfg.Environment,
		"addr", cfg.Addr(),
	)
	return srv.Run(ctx)
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
