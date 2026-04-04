package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter returns middleware that limits requests per IP address using a
// token bucket algorithm. Per ADR-014 Security Patterns, tiered rates should
// be applied via route groups:
//
//	auth routes:   RateLimiter(5, 5)    // 5 req/sec, burst 5
//	API routes:    RateLimiter(100, 20) // 100 req/sec, burst 20
//	public routes: RateLimiter(50, 10)  // 50 req/sec, burst 10
func RateLimiter(rps float64, burst int) func(http.Handler) http.Handler {
	var (
		mu       sync.Mutex
		limiters = make(map[string]*limiterEntry)
	)

	// Evict stale entries every 3 minutes
	go func() {
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			for ip, entry := range limiters {
				if time.Since(entry.lastSeen) > 10*time.Minute {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		entry, exists := limiters[ip]
		if !exists {
			entry = &limiterEntry{
				limiter:  rate.NewLimiter(rate.Limit(rps), burst),
				lastSeen: time.Now(),
			}
			limiters[ip] = entry
		} else {
			entry.lastSeen = time.Now()
		}
		return entry.limiter
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr // RealIP middleware normalizes this upstream

			if !getLimiter(ip).Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
