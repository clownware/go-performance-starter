# Next Steps: Integration & Testing

**Created**: 2025-11-15  
**Status**: Ready for implementation

---

## ✅ What's Complete

- 5 new ADRs documenting architecture
- Performance budget code (`internal/performance/`)
- Metrics middleware (`internal/middleware/metrics.go`)
- Alpine-optimized Dockerfile
- CI integration for performance checks
- Server integration (metrics middleware + `/metrics` endpoint)

---

## 🚀 Phase 9: Integration & Testing (Start Here)

### Step 1: Install Dependencies

```bash
# Add Prometheus client
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto  
go get github.com/prometheus/client_golang/prometheus/promhttp

# Add zerolog (if not present)
go get github.com/rs/zerolog

# Add security libraries (for future use)
go get github.com/gorilla/csrf
go get golang.org/x/time/rate
go get github.com/microcosm-cc/bluemonday

# Update dependencies
go mod tidy
```

### Step 2: Test the Build

```bash
# Run performance tests
task test:performance

# Build the binary
task build

# Check binary size
task test:binary-size

# Expected output: ✅ Binary size check passed
```

### Step 3: Start the Server

```bash
# Start with hot reload
task dev

# OR build and run
task build
./dist/app
```

### Step 4: Verify Metrics Endpoint

```bash
# Check metrics are being collected
curl http://localhost:8080/metrics

# You should see Prometheus metrics like:
# http_requests_total
# http_request_duration_seconds
# go_memory_usage_bytes
# performance_budget_violations_total
```

### Step 5: Test Performance Monitoring

```bash
# Make some requests
for i in {1..100}; do curl http://localhost:8080/healthz; done

# Check metrics again
curl http://localhost:8080/metrics | grep http_requests_total

# Check for slow requests in logs (should see warnings if > 100ms)
```

---

## 📊 Verify Everything is Working

### Checklist

- [ ] Server starts without errors
- [ ] `/metrics` endpoint returns Prometheus metrics
- [ ] Memory metrics update every 30 seconds
- [ ] Slow requests (> 100ms) logged with warnings
- [ ] `http_request_duration_seconds` histogram populated
- [ ] `performance_budget_violations_total` counter exists
- [ ] Binary size < 20MB (check with `task test:binary-size`)

---

## 🐳 Test Docker Build

```bash
# Build Docker image
docker build -t mssk:latest .

# Check image size (should be < 30MB)
docker images mssk:latest

# Run container
docker run -p 8080:8080 --env-file .env mssk:latest

# Test from another terminal
curl http://localhost:8080/healthz
curl http://localhost:8080/metrics
```

---

## 🔧 Phase 10: Optional Enhancements

### A. Implement Configuration Management (ADR-015)

The configuration patterns from ADR-015 should work out of the box since you already have `internal/config/`. Just verify:

```bash
# Check config package exists
ls internal/config/

# Verify .env loading works
grep "DATABASE_URL" .env
```

### B. Add Caching Layer (ADR-016)

Create `internal/cache/cache.go` with the in-memory cache from ADR-016:

```go
package cache

import (
    "sync"
    "time"
)

type CacheItem struct {
    Value      interface{}
    Expiration time.Time
}

type Cache struct {
    items map[string]CacheItem
    mu    sync.RWMutex
    ttl   time.Duration
}

// ... (rest of implementation from ADR-016)
```

Usage in handlers:

```go
var postCache = cache.NewCache(5 * time.Minute)

func GetPost(slug string) (*Post, error) {
    if cached, ok := postCache.Get(slug); ok {
        return cached.(*Post), nil
    }
    // ... fetch from DB and cache
}
```

### C. Add Security Middleware (ADR-014)

Create `internal/middleware/security.go`:

```go
package middleware

import "net/http"

func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        next.ServeHTTP(w, r)
    })
}
```

Add to server middleware stack:

```go
func (s *Server) setupMiddleware() {
    s.router.Use(mw.SecurityHeaders) // Add this
    s.router.Use(mw.RequestID)
    // ... rest
}
```

### D. Add Rate Limiting (ADR-014)

```go
package middleware

import (
    "net/http"
    "golang.org/x/time/rate"
)

func RateLimit(limiter *rate.Limiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 📈 Monitoring Setup (Production)

### Prometheus Configuration

Create `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'mssk'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### Grafana Dashboard

Key metrics to track:

1. **Request Rate**: `rate(http_requests_total[5m])`
2. **P95 Latency**: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`
3. **Error Rate**: `rate(http_requests_total{status=~"5.."}[5m])`
4. **Budget Violations**: `rate(performance_budget_violations_total[5m])`
5. **Memory Usage**: `go_memory_usage_bytes`

### Alert Rules

```yaml
groups:
  - name: performance_budgets
    rules:
      - alert: HighResponseTime
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.1
        annotations:
          summary: "P95 response time exceeds 100ms budget"
      
      - alert: HighMemoryUsage
        expr: go_memory_usage_bytes{type="alloc"} > 128000000
        annotations:
          summary: "Memory usage exceeds 128MB budget"
```

---

## 🎯 Quick Wins

### 1. Enable HTTP Compression

Already implemented in `server.go` with cache headers! Static assets are cached for 1 year.

### 2. Database Connection Pooling

Already configured in your pgxpool setup. Verify settings:

```go
// In config/config.go
MaxDBConns: 25  // Good default for most use cases
```

### 3. Static Asset Precompression

Add to build process:

```bash
# In Taskfile or CI
- name: Compress static assets
  run: |
    find web/static -type f \( -name "*.css" -o -name "*.js" \) -exec brotli -k {} \;
    find web/static -type f \( -name "*.css" -o -name "*.js" \) -exec gzip -k {} \;
```

---

## 🐛 Troubleshooting

### Issue: Import errors for prometheus packages

**Solution**: Run `go get` commands from Step 1

### Issue: Metrics endpoint returns 404

**Solution**: Verify `/metrics` route is registered in `server.go`:
```go
s.router.Handle("/metrics", promhttp.Handler())
```

### Issue: No metrics being collected

**Solution**: Check metrics middleware is registered BEFORE other middleware:
```go
s.router.Use(mw.Metrics) // Should be early in stack
```

### Issue: Binary size exceeds 20MB

**Solution**: Ensure build flags include stripping:
```bash
go build -ldflags="-s -w" -trimpath -o app ./cmd/api
```

### Issue: Docker build fails

**Solution**: Check all COPY paths exist:
```bash
ls -la web/static/css/app.css
ls -la migrations/
```

---

## 📝 Documentation to Review

Before implementing features, read these ADRs:

- **[ADR-000](./adr/ADR-000-Performance-Budgets-and-Quality-Attributes.md)** - Performance budgets and targets
- **[ADR-013](./adr/ADR-013-Error-Handling-and-Observability.md)** - Error handling patterns
- **[ADR-014](./adr/ADR-014-Security-Patterns-and-Threat-Model.md)** - Security implementation guide
- **[ADR-015](./adr/ADR-015-Configuration-Management-Strategy.md)** - Configuration patterns
- **[ADR-016](./adr/ADR-016-Caching-Strategy.md)** - Caching implementation guide

---

## 🎓 Learning Resources

- [Prometheus Go Client Documentation](https://prometheus.io/docs/guides/go-application/)
- [Writing Idiomatic Go](https://go.dev/doc/effective_go)
- [OWASP Go Secure Coding Practices](https://github.com/OWASP/Go-SCP)
- [Performance Budget Calculator](https://perf-budget-calculator.firebaseapp.com/)

---

## 🚦 Success Criteria

You'll know everything is working when:

1. ✅ Server starts in < 500ms
2. ✅ Binary is < 20MB
3. ✅ Docker image is < 30MB  
4. ✅ `/metrics` returns Prometheus data
5. ✅ Slow requests logged with warnings
6. ✅ Memory metrics update every 30s
7. ✅ All tests pass: `task test && task test:performance`

---

**Next**: Run Step 1 (Install Dependencies) and start testing! 🚀
