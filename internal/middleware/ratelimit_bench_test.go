package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// The rate limiter runs on every request (ADR-014), so its per-request cost
// comes straight out of the P95 budget (ADR-000). These benchmarks use
// b.Loop (Go 1.24): setup before the loop is excluded from the timing, and
// the loop body is never optimised away. Run with `task test:bench`.

var benchOK = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// BenchmarkRateLimiter measures one request through the limiter: the
// allowed path (bucket lookup + token take), the refused path (429 +
// Retry-After), and the allowed path spread across many client IPs so the
// bucket map is realistically large.
func BenchmarkRateLimiter(b *testing.B) {
	cases := []struct {
		name  string
		rps   float64
		burst int
		ips   int
	}{
		{name: "allowed/one-ip", rps: 1e9, burst: 1 << 30, ips: 1},
		{name: "allowed/64k-ips", rps: 1e9, burst: 1 << 30, ips: 1 << 16},
		{name: "refused", rps: 1e-9, burst: 1, ips: 1},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			h := RateLimiter(ctx, c.rps, c.burst)(benchOK)

			reqs := make([]*http.Request, c.ips)
			for i := range reqs {
				reqs[i] = httptest.NewRequest(http.MethodGet, "/", nil)
				reqs[i].RemoteAddr = "10." + strconv.Itoa(i>>8&255) + "." + strconv.Itoa(i&255) + ".1"
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, reqs[0]) // spend the single-burst token for "refused"

			i := 0
			for b.Loop() {
				h.ServeHTTP(httptest.NewRecorder(), reqs[i%len(reqs)])
				i++
			}
		})
	}
}

// BenchmarkLimiterStore_GetParallel measures bucket lookup under contention:
// every request takes the store's mutex, so this is the number that moves
// first as concurrency rises. RunParallel predates b.Loop and keeps its own
// pb.Next loop.
func BenchmarkLimiterStore_GetParallel(b *testing.B) {
	store := newLimiterStore(1e9, 1<<30)
	ips := make([]string, 1024)
	for i := range ips {
		ips[i] = "10.0." + strconv.Itoa(i>>8) + "." + strconv.Itoa(i&255)
	}
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			store.get(ips[i%len(ips)])
			i++
		}
	})
}

// BenchmarkCSRF measures the double-submit middleware on a safe request:
// the common path reuses the visitor's cookie, the first visit mints a
// token from crypto/rand and sets the cookie.
func BenchmarkCSRF(b *testing.B) {
	h := CSRF(true)(benchOK)

	b.Run("returning-visitor", func(b *testing.B) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		cookie := rec.Result().Cookies()[0]
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(cookie)
		for b.Loop() {
			h.ServeHTTP(httptest.NewRecorder(), req)
		}
	})

	b.Run("first-visit", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		for b.Loop() {
			h.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}
