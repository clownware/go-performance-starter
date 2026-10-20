package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// BenchmarkServer measures a request through the real router: the full
// global middleware chain (security headers, request ID, real IP, body cap,
// rate limit, compression, metrics, logging, recovery, timeout, CSRF) plus
// the handler. /healthz isolates the chain's overhead; / adds the templ
// render of the landing page, with and without gzip. This is the in-process
// cost behind the P95 < 100ms budget (ADR-000). Uses b.Loop (Go 1.24).
func BenchmarkServer(b *testing.B) {
	// The request logger writes a line per request; keep it out of the
	// numbers and the terminal.
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	b.Cleanup(func() { slog.SetDefault(prev) })

	srv, err := New(testConfig("production"), nil)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(srv.Close)

	// The global limiter allows 50 rps (burst 10) per client IP. Rotating
	// over 64k addresses keeps every bucket well under its rate, so the
	// benchmark measures the allowed path rather than 429s.
	const clients = 1 << 16
	addrs := make([]string, clients)
	for i := range addrs {
		addrs[i] = "10." + strconv.Itoa(i>>8&255) + "." + strconv.Itoa(i&255) + ".1:4000"
	}

	cases := []struct {
		name, path, encoding string
	}{
		{name: "healthz", path: "/healthz"},
		{name: "home", path: "/"},
		{name: "home-gzip", path: "/", encoding: "gzip"},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				req := httptest.NewRequest(http.MethodGet, c.path, nil)
				req.RemoteAddr = addrs[i%clients]
				if c.encoding != "" {
					req.Header.Set("Accept-Encoding", c.encoding)
				}
				rec := httptest.NewRecorder()
				srv.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					b.Fatalf("GET %s: status %d", c.path, rec.Code)
				}
				i++
			}
		})
	}
}
