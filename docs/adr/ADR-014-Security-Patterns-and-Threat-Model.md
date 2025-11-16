# ADR-014: Security Patterns and Threat Model

## Status

Accepted

## Context

Security must be designed into the application from the start. Without explicit security patterns and threat modeling, vulnerabilities emerge organically. This ADR establishes security baseline requirements, covering authentication, authorization, input validation, CSRF protection, and secrets management.

The application must protect against the OWASP Top 10 threats while maintaining performance and usability.

## Decision

### 1. Authentication and Authorization

#### Authentication Strategy
- Use **Supabase Auth** for identity management (as referenced in ADR-001)
- JWT tokens validated on every request
- Token expiration: 1 hour (refresh tokens: 30 days)
- Session management via HTTP-only cookies

#### Authorization Strategy
- **Row Level Security (RLS)** for data access control (as defined in ADR-004)
- Multi-tenant isolation enforced at database layer
- Application-level permission checks for complex business logic

```go
// Authentication middleware
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractToken(r)
        if token == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        claims, err := validateJWT(token)
        if err != nil {
            log.Warn().Err(err).Msg("invalid token")
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Add user context to request
        ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
        ctx = context.WithValue(ctx, "org_id", claims.OrgID)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 2. Input Validation and Sanitization

#### Validation Principles
- **Never trust client input** - validate on server
- **Whitelist over blacklist** - allow known-good inputs only
- **Fail closed** - reject invalid input, don't try to fix it
- **Type safety** - use Go's type system to enforce constraints

#### Input Validation Patterns

```go
// Validation helper functions
func validateEmail(email string) error {
    if email == "" {
        return errors.New("email is required")
    }
    if !emailRegex.MatchString(email) {
        return errors.New("invalid email format")
    }
    if len(email) > 255 {
        return errors.New("email too long")
    }
    return nil
}

func validateSlug(slug string) error {
    if slug == "" {
        return errors.New("slug is required")
    }
    // Whitelist: alphanumeric and hyphens only
    if !slugRegex.MatchString(slug) {
        return errors.New("invalid slug format")
    }
    if len(slug) > 100 {
        return errors.New("slug too long")
    }
    return nil
}

// Form validation example
func HandleContactForm(w http.ResponseWriter, r *http.Request) {
    email := r.FormValue("email")
    message := r.FormValue("message")
    
    // Server-side validation (never rely on client-side alone)
    if err := validateEmail(email); err != nil {
        renderError(w, err.Error())
        return
    }
    
    if len(message) == 0 || len(message) > 5000 {
        renderError(w, "message must be 1-5000 characters")
        return
    }
    
    // Sanitize HTML input
    message = bluemonday.StrictPolicy().Sanitize(message)
    
    // Process form...
}
```

#### SQL Injection Prevention
- **Always use parameterized queries** - never string concatenation
- Use SQLC for compile-time SQL validation (as defined in ADR-003)
- No dynamic SQL construction from user input

```go
// ✅ Safe: parameterized query
func GetPost(slug string) (*Post, error) {
    var post Post
    err := db.QueryRow(
        "SELECT id, title, content FROM posts WHERE slug = $1",
        slug, // Parameter binding prevents SQL injection
    ).Scan(&post.ID, &post.Title, &post.Content)
    return &post, err
}

// ❌ NEVER DO THIS: SQL injection vulnerability
func GetPostUnsafe(slug string) (*Post, error) {
    query := "SELECT * FROM posts WHERE slug = '" + slug + "'"
    // Attacker can inject: ' OR '1'='1
    // ...
}
```

### 3. CSRF Protection

#### CSRF Token Strategy
- Generate unique token per session
- Include token in forms via hidden input
- Validate token on state-changing operations (POST, PUT, DELETE)
- Token rotation on sensitive actions

```go
import "github.com/gorilla/csrf"

// CSRF middleware
func setupCSRF(r chi.Router) {
    csrfMiddleware := csrf.Protect(
        []byte("32-byte-secret-key"), // Store in environment variable
        csrf.Secure(isProd),           // HTTPS-only in production
        csrf.SameSite(csrf.SameSiteStrictMode),
    )
    
    r.Use(csrfMiddleware)
}

// Template helper
func renderForm(w http.ResponseWriter, r *http.Request) {
    tmpl.Execute(w, map[string]interface{}{
        "CSRFToken": csrf.Token(r),
    })
}
```

#### Template Usage
```html
<form action="/contact" method="POST">
    <input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">
    <input type="email" name="email" required>
    <button type="submit">Submit</button>
</form>
```

### 4. Rate Limiting

#### Rate Limit Strategy
- **Authentication endpoints**: 5 attempts per minute per IP
- **API endpoints**: 100 requests per minute per user
- **Public pages**: 1000 requests per minute per IP
- **Password reset**: 3 attempts per hour per email

```go
import "golang.org/x/time/rate"

// Rate limiter middleware
func rateLimitMiddleware(limiter *rate.Limiter) func(http.Handler) http.Handler {
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

// Per-user rate limiting
func perUserRateLimit(next http.Handler) http.Handler {
    limiters := sync.Map{}
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := getUserID(r.Context())
        
        limiter, _ := limiters.LoadOrStore(userID, 
            rate.NewLimiter(rate.Every(time.Minute/100), 10))
        
        if !limiter.(*rate.Limiter).Allow() {
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### 5. Secrets Management

#### Development Environment
- Use `.env` files loaded via `godotenv` (as defined in ADR-001)
- **NEVER commit `.env` to version control**
- Provide `.env.example` with placeholder values

#### Production Environment
- Use **Cloudflare Environment Variables** or equivalent
- Rotate secrets regularly (every 90 days minimum)
- Use different secrets per environment (dev/staging/prod)
- **No hardcoded secrets in code**

```go
// ✅ Safe: load from environment
func loadConfig() (*Config, error) {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        return nil, errors.New("DATABASE_URL not set")
    }
    
    jwtSecret := os.Getenv("JWT_SECRET")
    if jwtSecret == "" {
        return nil, errors.New("JWT_SECRET not set")
    }
    
    return &Config{
        DatabaseURL: dbURL,
        JWTSecret:   jwtSecret,
    }, nil
}

// ❌ NEVER DO THIS: hardcoded secrets
const (
    DatabaseURL = "postgres://user:password@localhost/db"
    APIKey      = "sk_live_abc123"
)
```

### 6. Security Headers

#### Required HTTP Headers

```go
func securityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Prevent clickjacking
        w.Header().Set("X-Frame-Options", "DENY")
        
        // Prevent MIME sniffing
        w.Header().Set("X-Content-Type-Options", "nosniff")
        
        // XSS protection
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        // Content Security Policy
        w.Header().Set("Content-Security-Policy", 
            "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
        
        // HTTPS enforcement (production only)
        if isProd {
            w.Header().Set("Strict-Transport-Security", 
                "max-age=31536000; includeSubDomains; preload")
        }
        
        // Referrer policy
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        // Permissions policy
        w.Header().Set("Permissions-Policy", 
            "geolocation=(), microphone=(), camera=()")
        
        next.ServeHTTP(w, r)
    })
}
```

### 7. Sensitive Data Handling

#### Logging Security
- **NEVER log passwords, tokens, or API keys**
- **NEVER log full credit card numbers** (PCI DSS)
- Scrub sensitive fields from logs automatically

```go
// Log scrubbing example
func sanitizeLog(data map[string]interface{}) map[string]interface{} {
    sensitiveFields := []string{"password", "token", "api_key", "secret"}
    
    for _, field := range sensitiveFields {
        if _, exists := data[field]; exists {
            data[field] = "[REDACTED]"
        }
    }
    
    return data
}
```

#### Database Encryption
- Encrypt sensitive columns at rest (e.g., PII, payment info)
- Use PostgreSQL `pgcrypto` extension for column-level encryption
- Store encryption keys separately from database

### 8. Dependency Security

#### Supply Chain Security
- Pin exact dependency versions in `go.mod`
- Run `go mod verify` in CI to detect tampering
- Scan dependencies for known vulnerabilities (`govulncheck`)
- Review dependency licenses for compliance

```yaml
# GitHub Actions security scanning
- name: Run govulncheck
  run: |
    go install golang.org/x/vuln/cmd/govulncheck@latest
    govulncheck ./...

- name: Verify dependencies
  run: go mod verify
```

## OWASP Top 10 Coverage

| Threat | Mitigation | Status |
|--------|-----------|--------|
| A01: Broken Access Control | RLS + Authentication middleware | ✅ Implemented |
| A02: Cryptographic Failures | HTTPS, encrypted secrets, JWT | ✅ Implemented |
| A03: Injection | Parameterized queries, input validation | ✅ Implemented |
| A04: Insecure Design | Threat modeling, secure defaults | ✅ Implemented |
| A05: Security Misconfiguration | Security headers, hardened defaults | ✅ Implemented |
| A06: Vulnerable Components | Dependency scanning, pinning | ✅ Implemented |
| A07: Authentication Failures | Rate limiting, JWT expiration | ✅ Implemented |
| A08: Data Integrity Failures | CSRF tokens, signed requests | ✅ Implemented |
| A09: Logging Failures | Structured logging, log scrubbing | ✅ Implemented |
| A10: Server-Side Request Forgery | URL validation, allowlist | ⚠️ Review per feature |

## Consequences

### Positive

- **Defense in Depth**: Multiple layers of security controls
- **Compliance Ready**: OWASP Top 10 coverage enables easier audits
- **Reduced Risk**: Proactive threat modeling prevents vulnerabilities
- **User Trust**: Strong security posture builds confidence

### Negative

- **Development Overhead**: Security checks add complexity
- **Performance Impact**: Rate limiting and validation add latency
- **Maintenance Burden**: Security requires ongoing attention

### Risks

- **False Security**: Following patterns doesn't guarantee security
- **Over-engineering**: Excessive controls can harm usability
- **Alert Fatigue**: Too many security logs can obscure real threats

## Alternatives Considered

### 1. Client-Side Validation Only
- **Rejected**: Never trust client-side validation alone

### 2. No Rate Limiting
- **Rejected**: Exposes application to abuse and DoS attacks

### 3. Basic HTTP Authentication
- **Rejected**: JWT + Supabase Auth provides better UX and security

## Implementation Checklist

- [ ] Implement authentication middleware with JWT validation
- [ ] Add CSRF protection to all state-changing endpoints
- [ ] Implement rate limiting for authentication and API endpoints
- [ ] Add security headers middleware
- [ ] Create input validation helpers (email, slug, etc.)
- [ ] Set up log scrubbing for sensitive fields
- [ ] Configure govulncheck in CI pipeline
- [ ] Document secrets rotation process
- [ ] Conduct security review of all handlers
- [ ] Perform penetration testing in staging

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP Cheat Sheet Series](https://cheatsheetseries.owasp.org/)
- [ADR-001: Foundation](./ADR-001-Foundation.md) (Secrets management)
- [ADR-004: Authorization Strategy](./ADR-004-Authorization-Strategy-RLS.md) (RLS)
- [Go Security Best Practices](https://github.com/OWASP/Go-SCP)
- [Gorilla CSRF](https://github.com/gorilla/csrf)

## Review Cadence

**Review Date**: 2026-02-15 (Quarterly security review)

---

**Date**: 2025-11-15
**Author**: System Architecture Team
