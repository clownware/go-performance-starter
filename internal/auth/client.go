package auth

import (
	"errors"

	supabase "github.com/supabase-community/supabase-go"
)

// AuthClient wraps the Supabase client.
type AuthClient struct {
	Client *supabase.Client // Store the Supabase client directly

	// baseURL/anonKey are kept for direct GoTrue REST calls the client
	// library doesn't cover (anonymous sign-in, ADR-024). serviceRoleKey
	// is optional and enables admin calls (reaper auth-side cleanup).
	baseURL        string
	anonKey        string
	serviceRoleKey string
}

// NewAuthClient creates and initializes a new Supabase client.
func NewAuthClient(supabaseURL, supabaseAnonKey string) (*AuthClient, error) {
	if supabaseURL == "" || supabaseAnonKey == "" {
		return nil, errors.New("supabase URL and anon key are required")
	}

	// Initialize the Supabase client
	// Note: NewClient returns (*Client, error)
	sbClient, err := supabase.NewClient(supabaseURL, supabaseAnonKey, nil) // Passing nil for options initially
	if err != nil {
		return nil, err
	}

	return &AuthClient{Client: sbClient, baseURL: supabaseURL, anonKey: supabaseAnonKey}, nil
}
