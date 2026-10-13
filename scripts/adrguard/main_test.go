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
		want     string // "ask", "deny" or "" for no verdict
		wantADR  string // substring the reason must cite
	}{
		{
			name:     "existing ADR asks the operator",
			filePath: "docs/adr/ADR-017-Templ-Adoption.md",
			want:     "ask",
			wantADR:  "ADR-033",
		},
		{
			name:     "new ADR file may be created",
			filePath: "docs/adr/ADR-034-Future-Decision.md",
			want:     "",
		},
		{
			name:     "sqlc-generated code is protected",
			filePath: "internal/database/db.go",
			want:     "deny",
			wantADR:  "ADR-019",
		},
		{
			name:     "templ-generated code is protected",
			filePath: "internal/view/pages/home_templ.go",
			want:     "deny",
			wantADR:  "ADR-019",
		},
		{
			name:     "AGENTS.md is protected",
			filePath: "AGENTS.md",
			want:     "deny",
			wantADR:  "ADR-022",
		},
		{
			name:     "hand-written templ source is editable",
			filePath: "internal/view/pages/home.templ",
			want:     "",
		},
		{
			name:     "ordinary source is editable",
			filePath: "internal/handler/h.go",
			want:     "",
		},
		{
			name:     "path outside the repo is not the guard's business",
			filePath: "/somewhere/else/AGENTS.md",
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abs := tt.filePath
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(root, tt.filePath)
			}
			decision, reason := decide(root, abs)
			if decision != tt.want {
				t.Fatalf("decide(%q) = %q (%q), want %q", tt.filePath, decision, reason, tt.want)
			}
			if tt.wantADR != "" && !strings.Contains(reason, tt.wantADR) {
				t.Errorf("reason %q does not cite %s", reason, tt.wantADR)
			}
		})
	}
}

func TestAskReasonNamesTheLegalMoves(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "docs", "adr", "ADR-001-Foundation.md")
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("# ADR-001\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, reason := decide(root, abs)
	for _, want := range []string{"append", "supersede", "never rewrite"} {
		if !strings.Contains(strings.ToLower(reason), want) {
			t.Errorf("reason %q does not mention %q", reason, want)
		}
	}
	if strings.Contains(reason, "ADR_GUARD_OFF") {
		t.Errorf("an ask needs no kill-switch; reason still names it: %q", reason)
	}
}

func TestHookOutput(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"docs/adr/ADR-001-Foundation.md", "AGENTS.md"} {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	payload := func(file string) string {
		return `{"cwd":"` + root + `","tool_input":{"file_path":"` + file + `"}}`
	}
	tests := []struct {
		name    string
		payload string
		off     bool
		want    string // substring of stdout, or "" for no output
	}{
		{name: "existing ADR emits an ask decision", payload: payload("docs/adr/ADR-001-Foundation.md"), want: `"permissionDecision":"ask"`},
		{name: "generated file emits a deny decision", payload: payload("AGENTS.md"), want: `"permissionDecision":"deny"`},
		{name: "ordinary file emits nothing", payload: payload("internal/handler/h.go"), want: ""},
		{name: "unreadable payload emits nothing", payload: "not json", want: ""},
		{name: "kill-switch silences a deny", payload: payload("AGENTS.md"), off: true, want: ""},
		{name: "kill-switch does not silence an ask", payload: payload("docs/adr/ADR-001-Foundation.md"), off: true, want: `"permissionDecision":"ask"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hookOutput(strings.NewReader(tt.payload), tt.off)
			if tt.want == "" && got != "" {
				t.Fatalf("got %q, want no output", got)
			}
			if !strings.Contains(got, tt.want) {
				t.Fatalf("got %q, want it to contain %q", got, tt.want)
			}
			if got != "" && !strings.Contains(got, `"hookEventName":"PreToolUse"`) {
				t.Errorf("output %q is not a PreToolUse decision", got)
			}
		})
	}
}
