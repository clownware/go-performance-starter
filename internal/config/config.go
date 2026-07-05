package config

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelseyhightower/envconfig"
)

// Config holds the application configuration.
type Config struct {
	Env                    string `envconfig:"ENV" default:"development"`
	HTTPPort               string `envconfig:"HTTP_PORT" default:"4000"`
	DatabaseURL            string `envconfig:"DATABASE_URL"`
	SupabaseURL            string `envconfig:"SUPABASE_URL"`
	SupabaseAnonKey        string `envconfig:"SUPABASE_ANON_KEY"`
	SupabaseServiceRoleKey string `envconfig:"SUPABASE_SERVICE_ROLE_KEY"`

	// MetricsToken gates /metrics: required as a bearer token when set;
	// with no token, /metrics is open in dev and hidden in production.
	MetricsToken string `envconfig:"METRICS_TOKEN"`

	// Database pool tuning (ADR-025; pgxpool defaults are too small for
	// production — MaxConns defaults to max(4, CPUs)).
	DBMaxConns        int32         `envconfig:"DB_MAX_CONNS" default:"25"`
	DBMinConns        int32         `envconfig:"DB_MIN_CONNS" default:"2"`
	DBMaxConnLifetime time.Duration `envconfig:"DB_MAX_CONN_LIFETIME" default:"30m"`
	DBMaxConnIdleTime time.Duration `envconfig:"DB_MAX_CONN_IDLE_TIME" default:"5m"`
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		slog.Error("Failed to process envconfig", "error", err)
		return nil, err
	}

	slog.Info("Configuration loaded successfully")
	return &cfg, nil
}

var validEnvs = map[string]bool{
	"development": true,
	"test":        true,
	"staging":     true,
	"production":  true,
}

// Validate fails fast on misconfiguration at boot instead of at first use
// (ADR-015; 2026-07-05 audit: a missing DATABASE_URL previously crashed at
// pool creation, an ENV typo silently disabled production behavior).
func (c *Config) Validate() error {
	if !validEnvs[c.Env] {
		return fmt.Errorf("ENV %q invalid: must be development, test, staging, or production", c.Env)
	}
	if port, err := strconv.Atoi(c.HTTPPort); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("HTTP_PORT %q invalid: must be a number between 1 and 65535", c.HTTPPort)
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if (c.SupabaseURL == "") != (c.SupabaseAnonKey == "") {
		return fmt.Errorf("SUPABASE_URL and SUPABASE_ANON_KEY must be set together (auth is disabled when both are empty)")
	}
	if c.DBMaxConns < 1 {
		return fmt.Errorf("DB_MAX_CONNS %d invalid: must be at least 1", c.DBMaxConns)
	}
	if c.DBMinConns < 0 || c.DBMinConns > c.DBMaxConns {
		return fmt.Errorf("DB_MIN_CONNS %d invalid: must be between 0 and DB_MAX_CONNS (%d)", c.DBMinConns, c.DBMaxConns)
	}
	return nil
}

// IsProduction reports whether the app runs with production behavior
// (HSTS, secure cookies, hidden unauthenticated /metrics).
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// PoolConfig builds the pgxpool configuration from DATABASE_URL plus the
// pool-tuning fields above.
func (c *Config) PoolConfig() (*pgxpool.Config, error) {
	pc, err := pgxpool.ParseConfig(c.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	pc.MaxConns = c.DBMaxConns
	pc.MinConns = c.DBMinConns
	pc.MaxConnLifetime = c.DBMaxConnLifetime
	pc.MaxConnIdleTime = c.DBMaxConnIdleTime
	return pc, nil
}
