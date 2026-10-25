// Package config loads server configuration from environment variables.
package config

import (
	"net"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds all server runtime configuration. Loading never dials any
// external service; in Phase 0 DATABASE_URL is parsed but unused.
type Config struct {
	Host              string        `env:"HTTP_HOST" envDefault:"0.0.0.0"`
	Port              int           `env:"HTTP_PORT" envDefault:"8081"`
	ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" envDefault:"5s"`
	ReadTimeout       time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"30s"`
	WriteTimeout      time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"30s"`
	IdleTimeout       time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"120s"`

	// RequestTimeout bounds a single request via the timeout middleware.
	RequestTimeout time.Duration `env:"HTTP_REQUEST_TIMEOUT" envDefault:"30s"`

	// ShutdownTimeout bounds graceful drain after SIGTERM/SIGINT.
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"15s"`

	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	// EnablePprof keeps /debug/pprof/* mounted; set false to harden prod.
	EnablePprof bool `env:"ENABLE_PPROF" envDefault:"true"`

	ServiceName string `env:"SERVICE_NAME" envDefault:"go-shell-server"`
	Environment string `env:"APP_ENV" envDefault:"dev"`

	// Database: Phase 0 parses only, never dials at startup.
	DatabaseURL string `env:"DATABASE_URL" envDefault:"postgres://go_shell:go_shell@localhost:5432/go_shell?sslmode=disable"`
}

// Load parses configuration from the process environment.
func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Addr returns the listen address in "host:port" form.
func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}
