# ADR-015: Configuration Management Strategy

## Status

Accepted

## Context

Applications require different configuration across environments (development, staging, production). Configuration must be secure, manageable, and follow the [Twelve-Factor App](https://12factor.net/config) principle of storing config in the environment. This ADR establishes patterns for configuration management, secrets handling, and environment-specific settings.

## Decision

### 1. Configuration Principles

Follow Twelve-Factor App principles:
- **Store config in environment variables**, not in code
- **Strict separation** between code and config
- **No hardcoded credentials or API keys**
- **Environment parity**: Same code runs in all environments with different config

### 2. Configuration Structure

```go
package config

import (
    "fmt"
    "os"
    "strconv"
    "time"
)

// Config holds all application configuration
type Config struct {
    // Server
    Port         int
    Environment  string // dev, staging, production
    
    // Database
    DatabaseURL  string
    MaxDBConns   int
    DBTimeout    time.Duration
    
    // Authentication
    JWTSecret    string
    JWTExpiry    time.Duration
    
    // External Services
    SupabaseURL string
    SupabaseKey string
    
    // Performance
    EnableCache  bool
    CacheTTL     time.Duration
    
    // Observability
    LogLevel     string
    MetricsPort  int
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
    cfg := &Config{
        Port:        getEnvInt("PORT", 8080),
        Environment: getEnv("ENVIRONMENT", "development"),
        
        DatabaseURL: getEnv("DATABASE_URL", ""),
        MaxDBConns:  getEnvInt("MAX_DB_CONNS", 25),
        DBTimeout:   getEnvDuration("DB_TIMEOUT", 10*time.Second),
        
        JWTSecret:   getEnv("JWT_SECRET", ""),
        JWTExpiry:   getEnvDuration("JWT_EXPIRY", 1*time.Hour),
        
        SupabaseURL: getEnv("SUPABASE_URL", ""),
        SupabaseKey: getEnv("SUPABASE_ANON_KEY", ""),
        
        EnableCache: getEnvBool("ENABLE_CACHE", true),
        CacheTTL:    getEnvDuration("CACHE_TTL", 5*time.Minute),
        
        LogLevel:    getEnv("LOG_LEVEL", "info"),
        MetricsPort: getEnvInt("METRICS_PORT", 9090),
    }
    
    // Validate required configuration
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("config validation: %w", err)
    }
    
    return cfg, nil
}

// Validate ensures required configuration is present
func (c *Config) Validate() error {
    if c.DatabaseURL == "" {
        return fmt.Errorf("DATABASE_URL is required")
    }
    
    if c.JWTSecret == "" && c.Environment == "production" {
        return fmt.Errorf("JWT_SECRET is required in production")
    }
    
    if c.SupabaseURL == "" {
        return fmt.Errorf("SUPABASE_URL is required")
    }
    
    if c.SupabaseKey == "" {
        return fmt.Errorf("SUPABASE_ANON_KEY is required")
    }
    
    return nil
}

// IsProd returns true if running in production
func (c *Config) IsProd() bool {
    return c.Environment == "production"
}

// IsDev returns true if running in development
func (c *Config) IsDev() bool {
    return c.Environment == "development"
}

// Helper functions
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if i, err := strconv.Atoi(value); err == nil {
            return i
        }
    }
    return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        if b, err := strconv.ParseBool(value); err == nil {
            return b
        }
    }
    return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if d, err := time.ParseDuration(value); err == nil {
            return d
        }
    }
    return defaultValue
}
```

### 3. Environment-Specific Configuration

#### Development Environment (`.env`)

```bash
# .env (local development only - never commit!)
ENVIRONMENT=development
PORT=8080

# Database
DATABASE_URL=postgresql://user:password@localhost:5432/dbname

# Authentication (use weak secret in dev)
JWT_SECRET=dev-secret-not-for-production

# Supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key

# Performance
ENABLE_CACHE=true
CACHE_TTL=5m

# Observability
LOG_LEVEL=debug
METRICS_PORT=9090
```

#### Example Configuration (`.env.example`)

```bash
# .env.example (committed to repository as template)
ENVIRONMENT=development
PORT=8080

# Database
DATABASE_URL=postgresql://user:password@localhost:5432/dbname

# Authentication
JWT_SECRET=generate-strong-secret-for-production

# Supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key-here

# Performance
ENABLE_CACHE=true
CACHE_TTL=5m

# Observability
LOG_LEVEL=info
METRICS_PORT=9090
```

#### Production Environment (Cloudflare Environment Variables)

```bash
# Set via Cloudflare Dashboard or CLI
ENVIRONMENT=production
PORT=8080
DATABASE_URL=<managed-db-connection-string>
JWT_SECRET=<strong-secret-from-secret-manager>
SUPABASE_URL=<production-supabase-url>
SUPABASE_ANON_KEY=<production-key>
ENABLE_CACHE=true
CACHE_TTL=10m
LOG_LEVEL=info
METRICS_PORT=9090
```

### 4. Loading Configuration

#### Application Startup

```go
package main

import (
    "log"
    "github.com/joho/godotenv"
    "yourapp/internal/config"
)

func main() {
    // Load .env file in development only
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, using environment variables")
    }
    
    // Load and validate configuration
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Use configuration
    server := NewServer(cfg)
    server.Start()
}
```

### 5. Secrets Management

#### Development
- Use `.env` file with weak secrets (e.g., `JWT_SECRET=dev-secret`)
- **NEVER commit `.env` to version control**
- Rotate development secrets quarterly

#### Production
- Use **Cloudflare Environment Variables** or equivalent platform
- Use **secret rotation** (every 90 days minimum)
- Use **per-environment secrets** (staging ≠ production)
- Consider **secret management service** (AWS Secrets Manager, HashiCorp Vault)

#### Secret Rotation Process

1. Generate new secret
2. Deploy new secret to environment variables
3. Deploy application with backward compatibility (accept old + new)
4. Verify new secret works
5. Remove old secret after grace period
6. Update documentation

### 6. Configuration Testing

```go
// config_test.go
package config

import (
    "os"
    "testing"
)

func TestLoadConfig(t *testing.T) {
    // Set required environment variables
    os.Setenv("DATABASE_URL", "postgres://localhost/test")
    os.Setenv("JWT_SECRET", "test-secret")
    os.Setenv("SUPABASE_URL", "https://test.supabase.co")
    os.Setenv("SUPABASE_ANON_KEY", "test-key")
    
    defer func() {
        os.Unsetenv("DATABASE_URL")
        os.Unsetenv("JWT_SECRET")
        os.Unsetenv("SUPABASE_URL")
        os.Unsetenv("SUPABASE_ANON_KEY")
    }()
    
    cfg, err := LoadConfig()
    if err != nil {
        t.Fatalf("LoadConfig failed: %v", err)
    }
    
    if cfg.DatabaseURL != "postgres://localhost/test" {
        t.Errorf("DatabaseURL = %s; want postgres://localhost/test", cfg.DatabaseURL)
    }
}

func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name    string
        cfg     *Config
        wantErr bool
    }{
        {
            name: "valid config",
            cfg: &Config{
                DatabaseURL: "postgres://localhost/db",
                SupabaseURL: "https://test.supabase.co",
                SupabaseKey: "key",
                JWTSecret:   "secret",
            },
            wantErr: false,
        },
        {
            name: "missing database URL",
            cfg: &Config{
                SupabaseURL: "https://test.supabase.co",
                SupabaseKey: "key",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.cfg.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 7. Configuration Documentation

Maintain configuration documentation in:
- **`.env.example`**: Template with all required variables
- **`README.md`**: Setup instructions and configuration guide
- **ADRs**: Architectural decisions about configuration strategy

## Consequences

### Positive

- **Environment Parity**: Same code runs everywhere with different config
- **Security**: Secrets never committed to version control
- **Flexibility**: Easy to change configuration without code changes
- **Testability**: Configuration can be easily mocked in tests
- **Twelve-Factor Compliance**: Follows industry best practices

### Negative

- **Environment Setup**: Requires manual configuration of environment variables
- **Discovery**: Need documentation to know which variables are required
- **Debugging**: Misconfigured environments can be hard to diagnose

### Risks

- **Secret Leakage**: Risk of accidentally committing `.env` file
- **Missing Variables**: Application fails to start if required variables missing
- **Drift**: Development and production configs may drift over time

## Alternatives Considered

### 1. Configuration Files (YAML/JSON)
- **Rejected**: Files encourage committing secrets to repository
- **Note**: Config files appropriate for non-secret settings (if needed)

### 2. Hardcoded Configuration
- **Rejected**: Violates Twelve-Factor principles, creates security risks

### 3. Command-Line Flags
- **Rejected**: Flags don't scale well for many configuration options
- **Note**: Flags acceptable for simple utilities, not web applications

## Implementation Checklist

- [ ] Create `internal/config/config.go` with Config struct
- [ ] Implement `LoadConfig()` with environment variable parsing
- [ ] Add `Validate()` method to check required configuration
- [ ] Create `.env.example` with all configuration variables
- [ ] Add `.env` to `.gitignore`
- [ ] Document configuration in README.md
- [ ] Add configuration tests
- [ ] Set up production environment variables in Cloudflare
- [ ] Implement secret rotation process
- [ ] Create configuration troubleshooting guide

## References

- [The Twelve-Factor App: Config](https://12factor.net/config)
- [ADR-001: Foundation](./ADR-001-Foundation.md) (Secret management strategy)
- [godotenv](https://github.com/joho/godotenv)
- [Cloudflare Environment Variables](https://developers.cloudflare.com/workers/configuration/environment-variables/)

## Review Cadence

**Review Date**: 2026-05-15

---

**Date**: 2025-11-15
**Author**: System Architecture Team

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: Environment reads (`os.Getenv`/`os.LookupEnv`) happen only in `internal/config` (allowlisted exception: the `cmd/api/main.go` bootstrap, which loads dotenv and the logger before config exists).
  - TC-2: `.env` is gitignored; `.env.example` exists at the repo root.
- **Checks:**
  - TC-1, TC-2 → `adr015-env-only-config` in `scripts/adrcheck` (status: **warn**)
- **Not machine-checkable:** "No hardcoded secrets" beyond structural patterns — no secret scanner is wired (recorded as a TODO in ADR-033). Environment parity is deployment discipline.
- **Graduation log:** _(empty)_
