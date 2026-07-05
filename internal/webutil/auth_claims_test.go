package webutil

import (
	"context"
	"encoding/json"
	"testing"
)

func TestAuthClaims_Validate(t *testing.T) {
	tests := []struct {
		name    string
		claims  AuthClaims
		wantErr bool
	}{
		{"authenticated with sub", AuthClaims{Sub: "abc", Role: RoleAuthenticated}, false},
		{"anon with sub", AuthClaims{Sub: "abc", Role: RoleAnon}, false},
		{"empty sub", AuthClaims{Sub: "", Role: RoleAuthenticated}, true},
		{"disallowed role", AuthClaims{Sub: "abc", Role: "service_role"}, true},
		{"injection attempt in role", AuthClaims{Sub: "abc", Role: "authenticated; DROP TABLE users"}, true},
		{"empty role", AuthClaims{Sub: "abc", Role: ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.claims.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthClaims_JSON(t *testing.T) {
	claims := AuthClaims{Sub: "user-123", Role: RoleAuthenticated, IsAnonymous: true}
	got, err := claims.JSON()
	if err != nil {
		t.Fatalf("JSON() error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("JSON() produced invalid JSON: %v", err)
	}
	if decoded["sub"] != "user-123" {
		t.Errorf("sub = %v, want user-123", decoded["sub"])
	}
	if decoded["role"] != RoleAuthenticated {
		t.Errorf("role = %v, want authenticated", decoded["role"])
	}
	if decoded["is_anonymous"] != true {
		t.Errorf("is_anonymous = %v, want true", decoded["is_anonymous"])
	}
}

func TestAuthClaimsContext(t *testing.T) {
	ctx := context.Background()

	if _, ok := AuthClaimsFromContext(ctx); ok {
		t.Error("empty context should have no claims")
	}

	claims := AuthClaims{Sub: "user-123", Role: RoleAuthenticated}
	ctx = WithAuthClaims(ctx, claims)
	got, ok := AuthClaimsFromContext(ctx)
	if !ok {
		t.Fatal("claims not found after WithAuthClaims")
	}
	if got != claims {
		t.Errorf("claims = %+v, want %+v", got, claims)
	}
}
