# Phase 6 — Authentication & Authorization with Supabase

Implement secure user authentication and permission control using Supabase Auth.

## Key Implementation Steps

| Step | Task | Why It Matters |
|------|------|----------------|
| 6.01 | Configure Supabase Auth | Setup authentication providers |
| 6.02 | Implement JWT validation | Secure token verification |
| 6.03 | Add authorization middleware | Role-based access control |
| 6.04 | Create permission system | Fine-grained authorization |
| 6.05 | Implement JWT refresh tokens | Manage user state securely |
| 6.06 | Add CSRF protection | Prevents cross-site attacks |
| 6.07 | Implement rate limiting | Prevents brute force attacks |
| 6.08 | Create account management | User profile and settings |
| 6.09 | Implement audit logging | Track security events |

## Core Principles

- Use Supabase Auth for authentication providers (email/password, OAuth, magic links)
- Implement proper JWT validation with appropriate audience and expiration checks
- Create role-based authorization middleware based on JWT claims
- Set up Row Level Security (RLS) in Supabase for data access control
- Implement CSRF protection for state-changing operations using token approach from Phase 5
- Add rate limiting to prevent authentication abuse
- Add audit logging for security-relevant events

## Security Considerations

- **JWT validation**: Verify signature, audience, and expiration
- **Token management**: Implement refresh token rotation for secure sessions
- **CSRF protection**: Add token validation for non-GET operations as shown in Phase 5
- **Row Level Security**: Enforce RLS policies designed in Phase 1
- **Error messages**: Return generic errors for auth failures
- **Rate limiting**: Prevent brute force attacks with middleware like ulule/limiter
- **Audit logging**: Track authentication and authorization events

## CSRF Integration with HTMX

Follow the pattern established in Phase 5 for CSRF protection:
- Generate tokens server-side for each session
- Include token in meta tag or hidden form field
- Send token with all state-changing HTMX requests
- Validate token on server for all non-GET operations
- Ensure token rotation on login/logout events

```go
// Middleware to validate CSRF tokens
func CSRFProtection(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip validation for safe methods
        if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
            next.ServeHTTP(w, r)
            return
        }
        
        // Get token from request header (sent by HTMX)
        requestToken := r.Header.Get("X-CSRF-Token")
        if requestToken == "" {
            // Try form field
            requestToken = r.FormValue("csrf_token")
        }
        
        // Validate token
        if !csrf.ValidateToken(requestToken) {
            http.Error(w, "Invalid CSRF token", http.StatusForbidden)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

## Rate Limiting Implementation

Implement rate limiting for sensitive endpoints, especially authentication:

```go
// Example using ulule/limiter with Redis store
import (
    "github.com/ulule/limiter/v3"
    "github.com/ulule/limiter/v3/drivers/store/redis"
)

// Configure rate limiter (e.g., 5 attempts per minute)
rate := limiter.Rate{
    Period: 1 * time.Minute,
    Limit:  5,
}

// Create Redis store for distributed rate limiting
store, err := redis.NewStore(redisClient)
if err != nil {
    log.Fatal(err)
}

// Create middleware instance
rateLimiterMiddleware := stdlib.NewMiddleware(
    limiter.New(store, rate),
    stdlib.WithForwardHeader(true),
)

// Apply to sensitive routes
authGroup.Use(rateLimiterMiddleware.Handler)
```

## Implementation Strategy

- Start with Supabase Auth integration for identity providers
- Implement JWT validation and refresh token handling
- Create middleware for authorization and role-based access
- Configure Row Level Security policies in Supabase
- Implement CSRF protection for state-changing operations
- Add rate limiting to authentication endpoints
- Create audit logging for security events
- Develop account management interfaces

## Common Pitfalls

- **Improper JWT validation**: Validate all claims and signature
- **Missing refresh token logic**: Implement proper token rotation
- **Overly permissive RLS**: Default to deny, explicitly grant
- **Missing CSRF protection**: Add for all state-changing forms
- **Insufficient rate limiting**: Apply to login and sensitive endpoints
- **Poor error handling**: Use generic auth error messages
- **Inadequate logging**: Track all authentication events

## Exit Criteria

- Supabase Auth providers configured and working
- JWT validation implemented correctly
- Authorization middleware controls access properly
- Row Level Security policies configured in Supabase
- CSRF protection implemented for all state-changing operations
- Rate limiting applied to authentication endpoints
- Account management features complete
- Audit logging tracks security events
- Account recovery flow implemented and tested



