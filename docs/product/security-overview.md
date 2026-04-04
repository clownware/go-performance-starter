# Security Overview

This document outlines the security practices implemented in the Alpine Go Performance Starter.

## Authentication

### JWT-Based Authentication
- **Implementation**: Supabase Auth provides JWT tokens
- **Validation**: Server-side validation of JWT signatures
- **Storage**: Tokens stored in HttpOnly cookies
- **Refresh**: Token refresh strategy with sliding expiration
- **Protection**: Protection against common JWT attacks (algorithm confusion, etc.)

### Configuration
```go
// Example JWT validation middleware
func JWTMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract JWT from Authorization header
        tokenString := extractToken(r)
        
        // Validate JWT
        token, err := validateToken(tokenString)
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Set user claims in context
        ctx := context.WithValue(r.Context(), "user", token.Claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## CSRF Protection

### Implementation
- Tokens generated per-session
- Transmitted via HTML meta tag (for HTMX) or hidden form fields
- Verified on all state-changing operations (non-GET requests)
- Token rotation on authentication events

### Implementation Pattern
```go
// Generate CSRF token
func generateCSRFToken(userID string) string {
    // Generate a unique token with expiration
    // ...
}

// Middleware to validate CSRF tokens
func CSRFProtection(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip validation for safe methods
        if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
            next.ServeHTTP(w, r)
            return
        }
        
        // Get token from request header (HTMX) or form field
        requestToken := r.Header.Get("X-CSRF-Token")
        if requestToken == "" {
            requestToken = r.FormValue("csrf_token")
        }
        
        // Validate token
        if !validateCSRFToken(requestToken) {
            http.Error(w, "Invalid CSRF token", http.StatusForbidden)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

## Rate Limiting

### Implementation
- Applied to authentication endpoints to prevent brute force
- Applied to API endpoints to prevent abuse
- Configurable per-endpoint with different thresholds

### Configuration
```go
// Example rate limiting middleware (using ulule/limiter)
rate := limiter.Rate{
    Period: 1 * time.Minute,
    Limit:  5, // 5 attempts per minute
}

// Create store (memory for development, Redis for production)
store := memory.NewStore()

// Create middleware
rateLimiterMiddleware := stdlib.NewMiddleware(
    limiter.New(store, rate),
    stdlib.WithForwardHeader(true),
)

// Apply to sensitive routes
authGroup.Use(rateLimiterMiddleware.Handler)
```

## Database Security

### Row Level Security (RLS)
- Enforces user-scoped data access at the database level
- Prevents unauthorized data access even with valid queries
- Applied to all tables containing user data

### Example RLS Policy
```sql
-- Enable RLS on items table
ALTER TABLE public.items ENABLE ROW LEVEL SECURITY;

-- Create policy for users to access only their own items
CREATE POLICY user_items_policy ON public.items
    USING (user_id = auth.uid())
    WITH CHECK (user_id = auth.uid());
```

## Secure Headers

### Implementation
- Content-Security-Policy to prevent XSS
- X-Frame-Options to prevent clickjacking
- X-Content-Type-Options to prevent MIME sniffing
- Referrer-Policy to control referrer information
- Other headers as recommended by OWASP

### Configuration
```go
// Secure headers middleware
func SecureHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        next.ServeHTTP(w, r)
    })
}
```

## Input Validation

### Implementation
- Server-side validation for all user inputs
- Strict type checking and sanitization
- Protection against SQL injection
- Validation integrated with HTMX for immediate feedback

### Example
```go
// Validate item creation
func validateItem(item *Item) error {
    if item.Name == "" {
        return errors.New("name is required")
    }
    
    if len(item.Name) > 100 {
        return errors.New("name must be less than 100 characters")
    }
    
    if item.Description != "" && len(item.Description) > 1000 {
        return errors.New("description must be less than 1000 characters")
    }
    
    return nil
}
```

## Additional Security Measures

- All dependencies regularly updated and scanned
- Production secrets managed through Cloudflare environments, not .env files
- Structured logging with sensitive data redaction
- Panic recovery middleware to prevent information disclosure
- Database connection parameters properly tuned

## Security Best Practices

- **Defense in Depth**: Multiple security layers independent of each other
- **Least Privilege**: Components access only what they need
- **Secure Defaults**: Security enabled by default, not opt-in
- **Fail Secure**: Errors default to denying access
- **Keep It Simple**: Simple security is more likely to be correct

## Further Reading

- [OWASP Top Ten](https://owasp.org/www-project-top-ten/)
- [JWT Best Practices](https://auth0.com/blog/a-look-at-the-latest-draft-for-jwt-bcp/)
- [Supabase Security Documentation](https://supabase.com/docs/guides/auth/overview#security)
- [HTMX Security Considerations](https://htmx.org/docs/#security)
