package main

import "testing"

func TestGoModVersions(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
	}{
		{
			name: "language, toolchain, and direct requires extracted",
			content: `module example.com/app

go 1.25.0

toolchain go1.26.5

require (
	github.com/a-h/templ v0.3.1020
	github.com/go-chi/chi/v5 v5.3.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
)
`,
			want: map[string]string{
				"go":                       "1.25.0",
				"toolchain":                "1.26.5",
				"github.com/a-h/templ":     "0.3.1020",
				"github.com/go-chi/chi/v5": "5.3.1",
			},
		},
		{
			name: "indirect requirements are excluded",
			content: `go 1.25.0

require (
	github.com/jackc/pgx/v5 v5.10.0
	github.com/beorn7/perks v1.0.1 // indirect
)
`,
			want: map[string]string{
				"go":                      "1.25.0",
				"github.com/jackc/pgx/v5": "5.10.0",
			},
		},
		{
			name:    "no toolchain directive leaves toolchain unset",
			content: "module example.com/app\n\ngo 1.25.0\n",
			want:    map[string]string{"go": "1.25.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := goModVersions(tt.content)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d entries (%v), want %d (%v)", len(got), got, len(tt.want), tt.want)
			}
			for k, want := range tt.want {
				if got[k] != want {
					t.Errorf("key %q = %q, want %q", k, got[k], want)
				}
			}
		})
	}
}

func TestLockfileVersion(t *testing.T) {
	lock := []byte(`{
		"packages": {
			"node_modules/tailwindcss": {"version": "4.3.2"},
			"node_modules/@tailwindcss/cli": {"version": "4.3.2"}
		}
	}`)

	tests := []struct {
		name    string
		pkg     string
		want    string
		wantErr bool
	}{
		{name: "exact version returned", pkg: "tailwindcss", want: "4.3.2"},
		{name: "scoped package resolves", pkg: "@tailwindcss/cli", want: "4.3.2"},
		{name: "missing package errors", pkg: "left-pad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lockfileVersion(lock, tt.pkg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionRegexes(t *testing.T) {
	tests := []struct {
		name  string
		re    string
		input string
		want  string
	}{
		{name: "htmx object literal", re: "js", input: `return{version:"1.9.10",config:{}}`, want: "1.9.10"},
		{name: "alpine assignment", re: "js", input: `this.version = '3.13.3'`, want: "3.13.3"},
		{name: "sqlc header", re: "sqlc", input: "// versions:\n//   sqlc v1.31.1\n", want: "1.31.1"},
		{name: "migrate download url", re: "migrate", input: "https://github.com/golang-migrate/migrate/releases/download/v4.19.1/migrate.linux-amd64.tar.gz", want: "4.19.1"},
		{name: "node ci pin", re: "node", input: "node-version: '20'", want: "20"},
	}

	res := map[string]interface{ FindStringSubmatch(string) []string }{
		"js":      jsVersionRe,
		"sqlc":    sqlcVersionRe,
		"migrate": migrateVersionRe,
		"node":    nodeVersionRe,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := res[tt.re].FindStringSubmatch(tt.input)
			if m == nil {
				t.Fatalf("no match in %q", tt.input)
			}
			if m[1] != tt.want {
				t.Errorf("got %q, want %q", m[1], tt.want)
			}
		})
	}
}
