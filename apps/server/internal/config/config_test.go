package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want 0.0.0.0", cfg.Host)
	}
	if cfg.Port != 8081 {
		t.Errorf("Port = %d, want 8081", cfg.Port)
	}
	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("RequestTimeout = %v, want 30s", cfg.RequestTimeout)
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 15s", cfg.ShutdownTimeout)
	}
	if !cfg.EnablePprof {
		t.Error("EnablePprof = false, want true")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.Addr() != "0.0.0.0:8081" {
		t.Errorf("Addr() = %q, want 0.0.0.0:8081", cfg.Addr())
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("HTTP_PORT", "9099")
	t.Setenv("HTTP_REQUEST_TIMEOUT", "5s")
	t.Setenv("ENABLE_PPROF", "false")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != 9099 {
		t.Errorf("Port = %d, want 9099", cfg.Port)
	}
	if cfg.RequestTimeout != 5*time.Second {
		t.Errorf("RequestTimeout = %v, want 5s", cfg.RequestTimeout)
	}
	if cfg.EnablePprof {
		t.Error("EnablePprof = true, want false")
	}
}
