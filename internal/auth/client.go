package auth

import (
	"errors"

	supabase "github.com/supabase-community/supabase-go"
)

// AuthClient wraps the Supabase client.
type AuthClient struct {
	Client *supabase.Client // Store the Supabase client directly
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

	return &AuthClient{Client: sbClient}, nil
}
