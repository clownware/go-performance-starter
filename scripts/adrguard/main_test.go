package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecide(t *testing.T) {
	root := t.TempDir()
	for rel, content := range map[string]string{
		"docs/adr/ADR-017-Templ-Adoption.md": "# ADR-017\n\n## Status\n\nAccepted\n",
		"internal/database/db.go":            "package database\n",
		"internal/view/pages/home_templ.go":  "package pages\n",
		"AGENTS.md":                          "generated\n",
		"internal/handler/h.go":              "package handler\n",
	} {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name     string
		filePath string
		wantDeny bool
		wantADR  string // substring the denial reason must cite
	}{
		{
			name:     "existing ADR is protected",
			filePath: "docs/adr/ADR-017-Templ-Adoption.md",
			wantDeny: true,
			wantADR:  "ADR-033",
		},
		{
			name:     "new ADR file may be created",
			filePath: "docs/adr/ADR-034-Future-Decision.md",
			wantDeny: false,
		},
		{
			name:     "sqlc-generated code is protected",
			filePath: "internal/database/db.go",
			wantDeny: true,
			wantADR:  "ADR-019",
		},
		{
			name:     "templ-generated code is protected",
			filePath: "internal/view/pages/home_templ.go",
			wantDeny: true,
			wantADR:  "ADR-019",
		},
		{
			name:     "AGENTS.md is protected",
			filePath: "AGENTS.md",
			wantDeny: true,
			wantADR:  "ADR-022",
		},
		{
			name:     "hand-written templ source is editable",
			filePath: "internal/view/pages/home.templ",
			wantDeny: false,
		},
		{
			name:     "ordinary source is editable",
			filePath: "internal/handler/h.go",
			wantDeny: false,
		},
		{
			name:     "path outside the repo is not the guard's business",
			filePath: "/somewhere/else/AGENTS.md",
			wantDeny: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abs := tt.filePath
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(root, tt.filePath)
			}
			reason := decide(root, abs)
			if (reason != "") != tt.wantDeny {
				t.Fatalf("decide(%q) = %q, wantDeny=%v", tt.filePath, reason, tt.wantDeny)
			}
			if tt.wantDeny && tt.wantADR != "" && !strings.Contains(reason, tt.wantADR) {
				t.Errorf("denial reason %q does not cite %s", reason, tt.wantADR)
			}
		})
	}
}
