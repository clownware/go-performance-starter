package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() *Config {
	return &Config{
		Env:                 "development",
		HTTPPort:            "4000",
		DatabaseURL:         "postgres://localhost:5432/app",
		DBMaxConns:          25,
		DBMinConns:          2,
		DBMaxConnLifetime:   30 * time.Minute,
		DBMaxConnIdleTime:   5 * time.Minute,
		MaxRequestBodyBytes: 1048576,
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr string // substring; empty means valid
	}{
		{"valid development config", func(c *Config) {}, ""},
		{"valid production config", func(c *Config) { c.Env = "production" }, ""},
		{"missing DATABASE_URL", func(c *Config) { c.DatabaseURL = "" }, "DATABASE_URL"},
		{"unknown environment", func(c *Config) { c.Env = "prod" }, "ENV"},
		{"non-numeric port", func(c *Config) { c.HTTPPort = "http" }, "HTTP_PORT"},
		{"port out of range", func(c *Config) { c.HTTPPort = "70000" }, "HTTP_PORT"},
		// Boundary rows below pin the exact comparison operators; a 2026-07-12
		// mutation baseline (ADR-032) showed these survived operator mutants.
		{"port 1 is the lowest valid", func(c *Config) { c.HTTPPort = "1" }, ""},
		{"port 0 is invalid", func(c *Config) { c.HTTPPort = "0" }, "HTTP_PORT"},
		{"port 65535 is the highest valid", func(c *Config) { c.HTTPPort = "65535" }, ""},
		{"port 65536 is invalid", func(c *Config) { c.HTTPPort = "65536" }, "HTTP_PORT"},
		{"supabase url without anon key", func(c *Config) { c.SupabaseURL = "https://x.supabase.co" }, "SUPABASE"},
		{"anon key without supabase url", func(c *Config) { c.SupabaseAnonKey = "anon" }, "SUPABASE"},
		{"supabase fully configured is valid", func(c *Config) {
			c.SupabaseURL = "https://x.supabase.co"
			c.SupabaseAnonKey = "anon"
		}, ""},
		{"zero max conns", func(c *Config) { c.DBMaxConns = 0 }, "DB_MAX_CONNS"},
		{"single max conn is valid", func(c *Config) { c.DBMaxConns = 1; c.DBMinConns = 0 }, ""},
		{"min conns above max", func(c *Config) { c.DBMinConns = 50 }, "DB_MIN_CONNS"},
		{"min conns equal to max is valid", func(c *Config) { c.DBMinConns = c.DBMaxConns }, ""},
		{"zero min conns is valid", func(c *Config) { c.DBMinConns = 0 }, ""},
		{"negative min conns is invalid", func(c *Config) { c.DBMinConns = -1 }, "DB_MIN_CONNS"},
		{"zero max request body", func(c *Config) { c.MaxRequestBodyBytes = 0 }, "MAX_REQUEST_BODY_BYTES"},
		{"one-byte max request body is valid", func(c *Config) { c.MaxRequestBodyBytes = 1 }, ""},
		{"guest mode without supabase", func(c *Config) {
			c.GuestModeEnabled = true
			c.GuestTTL = time.Hour
			c.ReaperInterval = time.Hour
		}, "GUEST_MODE_ENABLED"},
		{"guest mode with zero ttl", func(c *Config) {
			c.GuestModeEnabled = true
			c.SupabaseURL = "https://x.supabase.co"
			c.SupabaseAnonKey = "anon"
			c.ReaperInterval = time.Hour
		}, "GUEST_TTL"},
		{"guest mode with zero reaper interval", func(c *Config) {
			c.GuestModeEnabled = true
			c.SupabaseURL = "https://x.supabase.co"
			c.SupabaseAnonKey = "anon"
			c.GuestTTL = time.Hour
		}, "REAPER_INTERVAL"},
		{"guest mode fully configured is valid", func(c *Config) {
			c.GuestModeEnabled = true
			c.SupabaseURL = "https://x.supabase.co"
			c.SupabaseAnonKey = "anon"
			c.GuestTTL = 720 * time.Hour
			c.ReaperInterval = time.Hour
		}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)

			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validate() = %q, want error containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestIsProduction(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"production", true},
		{"development", false},
		{"test", false},
		{"staging", false},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := validConfig()
			cfg.Env = tt.env
			if got := cfg.IsProduction(); got != tt.want {
				t.Errorf("IsProduction() with ENV=%s = %v, want %v", tt.env, got, tt.want)
			}
		})
	}
}

func TestPoolConfig(t *testing.T) {
	cfg := validConfig()
	cfg.DBMaxConns = 10
	cfg.DBMinConns = 3
	cfg.DBMaxConnLifetime = 15 * time.Minute
	cfg.DBMaxConnIdleTime = 2 * time.Minute

	pc, err := cfg.PoolConfig()
	if err != nil {
		t.Fatalf("PoolConfig() error: %v", err)
	}
	if pc.MaxConns != 10 {
		t.Errorf("MaxConns = %d, want 10", pc.MaxConns)
	}
	if pc.MinConns != 3 {
		t.Errorf("MinConns = %d, want 3", pc.MinConns)
	}
	if pc.MaxConnLifetime != 15*time.Minute {
		t.Errorf("MaxConnLifetime = %v, want 15m", pc.MaxConnLifetime)
	}
	if pc.MaxConnIdleTime != 2*time.Minute {
		t.Errorf("MaxConnIdleTime = %v, want 2m", pc.MaxConnIdleTime)
	}
}

func TestPoolConfig_InvalidURL(t *testing.T) {
	cfg := validConfig()
	cfg.DatabaseURL = "://not-a-url"
	if _, err := cfg.PoolConfig(); err == nil {
		t.Error("PoolConfig() with invalid DATABASE_URL should error")
	}
}
