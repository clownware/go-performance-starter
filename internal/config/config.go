package config

import (
	"log/slog"

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
