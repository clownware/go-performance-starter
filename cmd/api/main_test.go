package main

import (
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"", slog.LevelInfo},
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"nonsense", slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run("level_"+tt.in, func(t *testing.T) {
			if got := parseLogLevel(tt.in); got != tt.want {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestNewHTTPServerTimeouts guards against connection-exhaustion regressions:
// a server without read/write/idle timeouts holds slow or idle connections
// forever (2026-07-05 deployment-readiness audit; ADR-025).
func TestNewHTTPServerTimeouts(t *testing.T) {
	srv := newHTTPServer(":4000", http.NewServeMux())

	tests := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"ReadHeaderTimeout", srv.ReadHeaderTimeout, 5 * time.Second},
		{"ReadTimeout", srv.ReadTimeout, 15 * time.Second},
		{"WriteTimeout", srv.WriteTimeout, 45 * time.Second},
		{"IdleTimeout", srv.IdleTimeout, 120 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
			if tt.got <= 0 {
				t.Errorf("%s must be set (connection exhaustion risk)", tt.name)
			}
		})
	}

	// WriteTimeout must exceed the 30s request middleware timeout, or
	// in-flight responses get cut off before the handler deadline fires.
	if srv.WriteTimeout <= 30*time.Second {
		t.Errorf("WriteTimeout = %v, must exceed the 30s request timeout middleware", srv.WriteTimeout)
	}

	if srv.Addr != ":4000" {
		t.Errorf("Addr = %q, want %q", srv.Addr, ":4000")
	}
	if srv.Handler == nil {
		t.Error("Handler not set")
	}
}
