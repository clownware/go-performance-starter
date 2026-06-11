package auth

import "testing"

func TestNewAuthClient(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		anonKey string
		wantErr bool
	}{
		{name: "missing url and key", url: "", anonKey: "", wantErr: true},
		{name: "missing key", url: "https://example.supabase.co", anonKey: "", wantErr: true},
		{name: "missing url", url: "", anonKey: "anon-key", wantErr: true},
		{name: "valid url and key", url: "https://example.supabase.co", anonKey: "anon-key", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewAuthClient(tt.url, tt.anonKey)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewAuthClient(%q, %q) error = nil, want non-nil", tt.url, tt.anonKey)
				}
				if client != nil {
					t.Errorf("NewAuthClient() returned a client alongside an error")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewAuthClient(%q, %q) unexpected error = %v", tt.url, tt.anonKey, err)
			}
			if client == nil || client.Client == nil {
				t.Error("NewAuthClient() returned a nil client without an error")
			}
		})
	}
}
