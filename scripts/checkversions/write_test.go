package main

import (
	"strings"
	"testing"
)

// TestSyncManifest proves -write rewrites versions.json from the repo's pins
// without disturbing the parts the ADR-030 contract says it must not touch:
// key order (consumers diff the file), the release-stamped template field,
// and formatting (two-space indent, trailing newline).
func TestSyncManifest(t *testing.T) {
	const manifest = `{
  "template": "v0.8.0",
  "go": "1.26.5",
  "go-minimum": "1.25.0",
  "chi": "5.3.1"
}
`
	tests := []struct {
		name        string
		manifest    string
		expected    map[string]string
		want        string
		wantChanges []string
		wantErr     bool
	}{
		{
			name:     "stale pin is rewritten in place, template and order kept",
			manifest: manifest,
			expected: map[string]string{"go": "1.26.6", "go-minimum": "1.25.0", "chi": "5.3.1"},
			want: `{
  "template": "v0.8.0",
  "go": "1.26.6",
  "go-minimum": "1.25.0",
  "chi": "5.3.1"
}
`,
			wantChanges: []string{`go: "1.26.5" -> "1.26.6"`},
		},
		{
			name:        "already in sync is a byte-for-byte no-op",
			manifest:    manifest,
			expected:    map[string]string{"go": "1.26.5", "go-minimum": "1.25.0", "chi": "5.3.1"},
			want:        manifest,
			wantChanges: nil,
		},
		{
			name:     "a newly wired key is appended after the existing ones",
			manifest: manifest,
			expected: map[string]string{"go": "1.26.5", "go-minimum": "1.25.0", "chi": "5.3.1", "pgx": "5.10.0"},
			want: `{
  "template": "v0.8.0",
  "go": "1.26.5",
  "go-minimum": "1.25.0",
  "chi": "5.3.1",
  "pgx": "5.10.0"
}
`,
			wantChanges: []string{`pgx: (missing) -> "5.10.0"`},
		},
		{
			name:     "unknown manifest keys survive untouched for the check to reject",
			manifest: "{\n  \"template\": \"v0.8.0\",\n  \"mystery\": \"1\"\n}\n",
			expected: map[string]string{},
			want:     "{\n  \"template\": \"v0.8.0\",\n  \"mystery\": \"1\"\n}\n",
		},
		{
			name:     "template is never rewritten even if expected names it",
			manifest: manifest,
			expected: map[string]string{"template": "v9.9.9", "go": "1.26.5", "go-minimum": "1.25.0", "chi": "5.3.1"},
			want:     manifest,
		},
		{
			name:     "unparseable manifest errors",
			manifest: "{not json",
			expected: map[string]string{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changes, err := syncManifest([]byte(tt.manifest), tt.expected)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("manifest mismatch\n got: %q\nwant: %q", got, tt.want)
			}
			if strings.Join(changes, "|") != strings.Join(tt.wantChanges, "|") {
				t.Errorf("changes = %q, want %q", changes, tt.wantChanges)
			}
		})
	}
}
