package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration.
type Config struct {
	Env           string
	HTTPPort      string
	DatabaseURL   string
	SupabaseURL   string
	SupabaseKey   string
	SupabaseAdmin string
}

// Load loads configuration from environment variables.
// In development, it attempts to load from a .env file first.
func Load() (*Config, error) {
	// Attempt to load .env file only in non-production environments.
	// Check for a PRODUCTION env variable or similar indicator.
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			// Log the error but don't fail, as .env might be optional or vars provided differently
			log.Printf("Warning: could not load .env file: %v", err)
		}
	}

	cfg := &Config{
		Env:           getEnv("ENV", "development"),
		HTTPPort:      getEnv("HTTP_PORT", "4000"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		SupabaseURL:   getEnv("SUPABASE_URL", ""),
		SupabaseKey:   getEnv("SUPABASE_ANON_KEY", ""),
		SupabaseAdmin: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
	}

	// TODO: Add validation for required fields

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
