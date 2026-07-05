# ADR-013: Error Handling and Observability Strategy

## Status

Accepted

## Context

Robust error handling is critical for production reliability, debugging, and operational visibility. Without a consistent error handling strategy, issues become difficult to diagnose, and production incidents take longer to resolve. The application needs clear patterns for:

- Handling errors at different layers (HTTP, business logic, database)
- Logging errors with appropriate context
- Monitoring and alerting on error conditions
- Graceful degradation when dependencies fail

This ADR establishes patterns for error handling, structured logging, and observability.

## Decision

### 1. Error Handling Patterns

#### HTTP Layer Error Handling

```go
// Standard error response structure
type ErrorResponse struct {
    Error   string `json:"error"`
    Code    string `json:"code,omitempty"`
    Details string `json:"details,omitempty"`
}

// Error handler middleware
func errorHandlerMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Error().
                    Interface("error", err).
                    Str("path", r.URL.Path).
                    Msg("panic recovered")
                
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        
        next.ServeHTTP(w, r)
    })
}

// Handler error pattern with graceful degradation
func HandleBlogPost(w http.ResponseWriter, r *http.Request) {
    slug := chi.URLParam(r, "slug")
    
    post, err := getPost(slug)
    if err != nil {
        log.Error().
            Err(err).
            Str("slug", slug).
            Str("path", r.URL.Path).
            Msg("failed to fetch post")
        
        // Graceful degradation: redirect to 404, don't crash
        http.Redirect(w, r, "/404", http.StatusSeeOther)
        return
    }
    
    if post == nil {
        http.Redirect(w, r, "/404", http.StatusSeeOther)
        return
    }
    
    render(w, post)
}
```

#### Business Logic Error Handling

```go
// Define custom error types for domain errors
type NotFoundError struct {
    Resource string
    ID       string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// Service layer returns typed errors
func GetPost(slug string) (*Post, error) {
    post, err := db.QueryPost(slug)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, &NotFoundError{Resource: "post", ID: slug}
        }
        return nil, fmt.Errorf("query post: %w", err)
    }
    return post, nil
}
```

#### Database Error Handling

```go
// Wrap database errors with context
func (r *PostRepository) GetBySlug(ctx context.Context, slug string) (*Post, error) {
    var post Post
    
    err := r.db.QueryRowContext(ctx, 
        "SELECT id, title, content FROM posts WHERE slug = $1", 
        slug,
    ).Scan(&post.ID, &post.Title, &post.Content)
    
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, &NotFoundError{Resource: "post", ID: slug}
        }
        return nil, fmt.Errorf("get post by slug %s: %w", slug, err)
    }
    
    return &post, nil
}
```

### 2. Structured Logging

> **Amended 2026-07-05**: [ADR-026](ADR-026-Logging-Standardization.md) standardizes logging on stdlib `log/slog`. The level semantics, required context fields, and scrubbing rules below are unchanged; the zerolog code samples are illustrative of the pattern, not the library.

Use **zerolog** (as defined in ADR-001) with consistent context:

```go
// Request logging middleware
func requestLoggerMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Create request-scoped logger with context
        logger := log.With().
            Str("method", r.Method).
            Str("path", r.URL.Path).
            Str("remote_addr", r.RemoteAddr).
            Str("request_id", getRequestID(r)).
            Logger()
        
        // Add logger to request context
        ctx := logger.WithContext(r.Context())
        
        // Wrap response writer to capture status code
        ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
        
        next.ServeHTTP(ww, r.WithContext(ctx))
        
        duration := time.Since(start)
        
        logger.Info().
            Int("status", ww.Status()).
            Int("bytes", ww.BytesWritten()).
            Dur("duration_ms", duration).
            Msg("request completed")
        
        // Alert if exceeding performance budget
        if duration > MaxResponseTime {
            logger.Warn().
                Dur("duration_ms", duration).
                Dur("budget_ms", MaxResponseTime).
                Msg("response time exceeded budget")
        }
    })
}
```

#### Log Levels and Usage

- **Error**: Application errors, failed operations, panics
- **Warn**: Degraded performance, budget violations, retry attempts
- **Info**: Request/response logs, significant state changes
- **Debug**: Development-only, verbose internal state (disabled in production)

#### Log Context Standards

Every log entry MUST include:
- **Timestamp**: Automatic via zerolog
- **Level**: error/warn/info/debug
- **Message**: Human-readable description

Additional context when applicable:
- **request_id**: Trace requests across services
- **user_id**: User performing action (if authenticated)
- **error**: Wrapped error with stack trace
- **duration_ms**: Operation duration
- **resource**: Database table, API endpoint, etc.

### 3. Observability and Monitoring

#### Prometheus Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path", "status"},
    )
    
    errorCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_errors_total",
            Help: "Total number of HTTP errors",
        },
        []string{"method", "path", "status"},
    )
)

func init() {
    prometheus.MustRegister(requestDuration)
    prometheus.MustRegister(errorCounter)
}

// Metrics middleware
func metricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
        
        next.ServeHTTP(ww, r)
        
        duration := time.Since(start).Seconds()
        status := strconv.Itoa(ww.Status())
        
        requestDuration.WithLabelValues(r.Method, r.URL.Path, status).Observe(duration)
        
        if ww.Status() >= 400 {
            errorCounter.WithLabelValues(r.Method, r.URL.Path, status).Inc()
        }
    })
}
```

#### Health Check Endpoint

```go
// Health check with dependency status
func HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()
    
    health := HealthStatus{
        Status:    "healthy",
        Timestamp: time.Now(),
        Checks:    make(map[string]CheckResult),
    }
    
    // Check database
    if err := db.PingContext(ctx); err != nil {
        health.Status = "unhealthy"
        health.Checks["database"] = CheckResult{
            Status: "down",
            Error:  err.Error(),
        }
    } else {
        health.Checks["database"] = CheckResult{Status: "up"}
    }
    
    // Set appropriate status code
    statusCode := http.StatusOK
    if health.Status == "unhealthy" {
        statusCode = http.StatusServiceUnavailable
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(health)
}
```

### 4. Error Reporting and Alerting

#### Development Environment
- Errors logged to stdout/stderr
- Panic stack traces printed in full
- No external error tracking service

#### Production Environment
- Errors logged to structured JSON format
- Critical errors trigger alerts (via monitoring platform)
- Error tracking service (Sentry, Rollbar) integration optional

#### Alert Conditions
- Error rate exceeds 1% of requests
- P95 response time exceeds budget (100ms)
- Health check failures
- Database connection pool exhaustion
- Memory usage exceeds 80% of budget

## Consequences

### Positive

- **Faster Debugging**: Structured logs with context accelerate root cause analysis
- **Proactive Monitoring**: Metrics enable detection of issues before user impact
- **Graceful Degradation**: Clear error handling prevents cascading failures
- **Production Readiness**: Comprehensive observability enables confident deployments
- **Performance Visibility**: Metrics track budget compliance in real-time

### Negative

- **Overhead**: Logging and metrics add minimal performance cost
- **Storage Costs**: Structured logs require log aggregation infrastructure
- **Maintenance**: Alert thresholds need tuning to avoid alert fatigue

### Risks

- **Over-logging**: Excessive debug logs in production impact performance
- **PII Leakage**: Risk of logging sensitive data (require log scrubbing)
- **Alert Fatigue**: Too many alerts desensitize engineers

## Alternatives Considered

### 1. Application-Level Error Tracking Only
- **Rejected**: No visibility into production issues without centralized logging

### 2. Different Logging Library (zap, log/slog)
- **Decision**: Stick with zerolog (defined in ADR-001) for consistency
- **Note**: log/slog is now standard library (Go 1.21+) but zerolog ecosystem mature

### 3. No Structured Logging (plain text logs)
- **Rejected**: Structured logs essential for log aggregation and analysis

## Implementation Checklist

- [ ] Implement error handling middleware with panic recovery
- [ ] Add request logger middleware with request_id context
- [ ] Define custom error types (NotFoundError, ValidationError)
- [ ] Add Prometheus metrics endpoint at `/metrics`
- [ ] Implement health check endpoint at `/health`
- [ ] Configure log levels per environment (debug in dev, info in prod)
- [ ] Add performance budget tracking in metrics middleware
- [ ] Document log scrubbing policy for PII
- [ ] Set up alert rules in monitoring platform
- [ ] Test error scenarios in staging environment

## References

- [ADR-001: Foundation Architectural Decisions](./ADR-001-Foundation.md) (zerolog selection)
- [Twelve-Factor App: Logs](https://12factor.net/logs)
- [Google SRE Book: Monitoring](https://sre.google/sre-book/monitoring-distributed-systems/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Zerolog Documentation](https://github.com/rs/zerolog)

## Review Cadence

**Review Date**: 2026-05-15

---

**Date**: 2025-11-15
**Author**: System Architecture Team
