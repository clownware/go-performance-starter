# ADR-015: Configuration Management Strategy

## Status

Accepted

> **Amended 2026-08-02**: the code and env samples in this ADR predate the adoption of `kelseyhightower/envconfig` — the authoritative config surface is `internal/config/config.go` and `.env.example`. The real variables are `ENV` (not `ENVIRONMENT`), `HTTP_PORT` (default `4000`, not `PORT=8080`), `DATABASE_URL`, `SUPABASE_URL`/`SUPABASE_ANON_KEY` (set both or neither — auth is disabled when both are empty), optional `SUPABASE_SERVICE_ROLE_KEY` and `METRICS_TOKEN` (there is no separate metrics port), and pool tuning via `DB_MAX_CONNS`/`DB_MIN_CONNS`/`DB_MAX_CONN_LIFETIME`. `JWT_SECRET`, `JWT_EXPIRY`, `ENABLE_CACHE`, and `CACHE_TTL` do not exist. Production env vars are set on the container host per [ADR-025](./ADR-025-Deployment-Target.md) (Fly.io worked example), not in Cloudflare. The §3 env samples and §5 production story below have been corrected; the §2 Go sample stands as the original decision illustration.

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
ENV=development
HTTP_PORT=4000

# Database
DATABASE_URL=postgresql://user:password@localhost:5432/dbname

# Supabase (set both or neither — auth is disabled when both are empty)
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key

# Observability (read by the logger bootstrap in cmd/api/main.go)
LOG_LEVEL=debug
```

#### Example Configuration (`.env.example`)

```bash
# .env.example (committed to repository as template — the real file at the
# repo root carries the full annotated surface)
ENV=development
HTTP_PORT=4000

# Database
DATABASE_URL=postgres://postgres:postgres@localhost:5432/alpine_saas?sslmode=disable

# Supabase (set both or neither — auth is disabled when both are empty)
SUPABASE_URL=
SUPABASE_ANON_KEY=

# Optional overrides (compiled-in defaults shown)
# DB_MAX_CONNS=25
# DB_MIN_CONNS=2
# DB_MAX_CONN_LIFETIME=30m
# METRICS_TOKEN=
```

#### Production Environment (container host secrets — ADR-025)

```bash
# Secrets go in the container host's secret store (Fly.io worked example);
# non-secret vars (ENV=production, HTTP_PORT) live in fly.toml [env].
fly secrets set \
  DATABASE_URL=<managed-db-connection-string> \
  SUPABASE_URL=<production-supabase-url> \
  SUPABASE_ANON_KEY=<production-key> \
  METRICS_TOKEN=<bearer-token-gating-/metrics>
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
- Use the **container host's secret store** (`fly secrets set` in the ADR-025 worked example) or equivalent platform
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
- [ ] Set up production environment variables on the container host (Fly.io per ADR-025)
- [ ] Implement secret rotation process
- [ ] Create configuration troubleshooting guide

## References

- [The Twelve-Factor App: Config](https://12factor.net/config)
- [ADR-001: Foundation](./ADR-001-Foundation.md) (Secret management strategy)
- [godotenv](https://github.com/joho/godotenv)
- [ADR-025: Deployment Target](./ADR-025-Deployment-Target.md) (production env/secrets on the container host)
- [Fly.io Secrets](https://fly.io/docs/apps/secrets/)

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
