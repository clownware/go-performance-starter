package config

import (
	"log/slog"
	"os"
	"github.com/kelseyhightower/envconfig"
)

// Config holds the application configuration.
type Config struct {
	Env           string `env:"ENV" default:"development"`
	HTTPPort      string `env:"HTTP_PORT" default:"8080"`
	DatabaseURL   string `env:"DATABASE_URL"` // Keep optional for now
	SupabaseURL   string // Manually assigned
	SupabaseAnonKey string // Manually assigned
	SupabaseServiceRoleKey string `env:"SUPABASE_SERVICE_ROLE_KEY"` // Optional for backend actions
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	// Debug: Check environment variables directly before processing
	slog.Debug("Checking env vars before envconfig",
		"SUPABASE_URL", os.Getenv("SUPABASE_URL"),
		"SUPABASE_ANON_KEY", os.Getenv("SUPABASE_ANON_KEY"),
	)

	// envconfig will process other fields (like HTTP_PORT, ENV)
	var cfg Config
	err := envconfig.Process("", &cfg) // Load config into cfg struct
	if err != nil {
		slog.Error("Failed to process envconfig", "error", err)
		return nil, err
	}

	// Debug: Log the loaded values to verify envconfig worked
	slog.Debug("Loaded config values",
		"SupabaseURL", cfg.SupabaseURL,
		"SupabaseAnonKey", cfg.SupabaseAnonKey,
	)

	// Manually assign Supabase keys and HTTPPort as envconfig seems unreliable here
	cfg.SupabaseURL = os.Getenv("SUPABASE_URL")
	cfg.SupabaseAnonKey = os.Getenv("SUPABASE_ANON_KEY")
	cfg.HTTPPort = os.Getenv("HTTP_PORT")

	// Re-log after manual assignment for verification
	slog.Debug("Config values after manual assignment",
		"SupabaseURL", cfg.SupabaseURL,
		"SupabaseAnonKey", cfg.SupabaseAnonKey,
		"HTTPPort", cfg.HTTPPort,
	)

	slog.Info("Configuration loaded successfully")
	return &cfg, nil
}
