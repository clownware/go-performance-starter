package main

import (
	"bufio"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
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

// TestNewHTTPServerTimeoutsEnforced proves the timeouts above actually cut
// connections, at their production values, not just that the fields are
// set. Each case runs in a synctest bubble against httptest.NewTestServer
// (Go 1.27), whose in-memory network runs on the bubble's fake clock: a
// slowloris client is dropped exactly at ReadHeaderTimeout, an idle
// keep-alive connection exactly at IdleTimeout, and the test takes
// microseconds.
func TestNewHTTPServerTimeoutsEnforced(t *testing.T) {
	prod := newHTTPServer(":4000", nil)

	tests := []struct {
		name     string
		send     string // bytes the client writes, then it goes silent
		response bool   // whether the server answers before the timeout
		closesAt time.Duration
	}{
		{
			name:     "slowloris headers cut at ReadHeaderTimeout",
			send:     "GET / HTTP/1.1\r\nHost: example.com\r\n", // no blank line: headers never finish
			closesAt: prod.ReadHeaderTimeout,
		},
		{
			name:     "idle keep-alive closed at IdleTimeout",
			send:     "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
			response: true,
			closesAt: prod.IdleTimeout,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				}))
				srv.Config.ReadHeaderTimeout = prod.ReadHeaderTimeout
				srv.Config.ReadTimeout = prod.ReadTimeout
				srv.Config.WriteTimeout = prod.WriteTimeout
				srv.Config.IdleTimeout = prod.IdleTimeout

				tr, ok := srv.Client().Transport.(*http.Transport) // starts the in-memory server
				if !ok {
					t.Fatalf("Client().Transport is %T, want *http.Transport", srv.Client().Transport)
				}
				conn, err := tr.DialContext(t.Context(), "tcp", "example.com:80")
				if err != nil {
					t.Fatalf("dial: %v", err)
				}
				defer conn.Close()
				if _, err := io.WriteString(conn, tt.send); err != nil {
					t.Fatalf("write: %v", err)
				}

				br := bufio.NewReader(conn)
				if tt.response {
					resp, err := http.ReadResponse(br, nil)
					if err != nil {
						t.Fatalf("read response: %v", err)
					}
					resp.Body.Close()
				}

				// The server closing the connection ends this read with EOF.
				closed := make(chan struct{})
				go func() {
					_, _ = br.ReadByte()
					close(closed)
				}()
				isClosed := func() bool {
					select {
					case <-closed:
						return true
					default:
						return false
					}
				}

				synctest.Sleep(tt.closesAt - time.Nanosecond)
				if isClosed() {
					t.Fatalf("connection closed before %v", tt.closesAt)
				}
				synctest.Sleep(time.Nanosecond)
				if !isClosed() {
					t.Fatalf("connection still open at %v", tt.closesAt)
				}
			})
		})
	}
}
