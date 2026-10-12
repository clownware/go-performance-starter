package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckADRIndexComplete(t *testing.T) {
	const header = "# ADRs\n\n| ADR | Title |\n|---|---|\n"
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name: "every file has a row and every row a file",
			files: map[string]string{
				"docs/adr/README.md":             header + "| [001](ADR-001-Foundation.md) | Foundation |\n",
				"docs/adr/ADR-001-Foundation.md": "# ADR-001\n",
				"docs/adr/TEMPLATE.md":           "# ADR-NNN\n",
			},
			violations: 0,
		},
		{
			name: "ADR file with no index row flagged",
			files: map[string]string{
				"docs/adr/README.md":             header + "| [001](ADR-001-Foundation.md) | Foundation |\n",
				"docs/adr/ADR-001-Foundation.md": "# ADR-001\n",
				"docs/adr/ADR-002-Migrations.md": "# ADR-002\n",
			},
			violations: 1,
		},
		{
			name: "index row naming a missing file flagged",
			files: map[string]string{
				"docs/adr/README.md":             header + "| [001](ADR-001-Foundation.md) | Foundation |\n| [002](ADR-002-Gone.md) | Gone |\n",
				"docs/adr/ADR-001-Foundation.md": "# ADR-001\n",
			},
			violations: 1,
		},
		{
			name: "a mention in prose is not a row",
			files: map[string]string{
				"docs/adr/README.md":             "# ADRs\n\nSee [ADR-001](ADR-001-Foundation.md) for the base.\n",
				"docs/adr/ADR-001-Foundation.md": "# ADR-001\n",
			},
			violations: 1,
		},
		{
			name: "missing index flagged once",
			files: map[string]string{
				"docs/adr/ADR-001-Foundation.md": "# ADR-001\n",
			},
			violations: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkADRIndexComplete(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestRemovedLinesFromExistingADR(t *testing.T) {
	const hdr = "diff --git a/docs/adr/ADR-001-Foundation.md b/docs/adr/ADR-001-Foundation.md\n--- a/docs/adr/ADR-001-Foundation.md\n+++ b/docs/adr/ADR-001-Foundation.md\n"
	tests := []struct {
		name       string
		diff       string
		violations int
	}{
		{
			name:       "pure append passes",
			diff:       hdr + "@@ -10,0 +11,2 @@\n+> **Amended 2026-10-12**: a note.\n+\n",
			violations: 0,
		},
		{
			name:       "decision text rewritten in place flagged",
			diff:       hdr + "@@ -21 +21 @@\n-We use the module path `example.com/old-name`.\n+We use the module path `example.com/new-name`.\n",
			violations: 1,
		},
		{
			name:       "line deleted outright flagged",
			diff:       hdr + "@@ -30 +29,0 @@\n-- Cloudflare Workers is ruled out.\n",
			violations: 1,
		},
		{
			name:       "status line may change",
			diff:       hdr + "@@ -5 +5 @@\n-Proposed\n+Accepted\n",
			violations: 0,
		},
		{
			name:       "bold-label status line may change",
			diff:       hdr + "@@ -3 +3 @@\n-*   **Status:** Accepted\n+*   **Status:** Superseded by ADR-017\n",
			violations: 0,
		},
		{
			name:       "text kept and extended on the same line passes",
			diff:       hdr + "@@ -40 +40 @@\n-  - TC-1 → `some-check` (status: **warn**)\n+  - TC-1 → `some-check` (status: **warn**) — promoted, see the graduation log\n",
			violations: 0,
		},
		{
			name:       "struck-through text is kept, so it passes",
			diff:       hdr + "@@ -50 +50,2 @@\n-- We assume a single trusted hop.\n+- ~~We assume a single trusted hop.~~\n+  **Amendment**: falsified on Fly.\n",
			violations: 0,
		},
		{
			name:       "empty graduation-log placeholder may be replaced",
			diff:       hdr + "@@ -60 +60,2 @@\n-- **Graduation log:** _(empty)_\n+- **Graduation log:**\n+  - 2026-10-12 — promoted.\n",
			violations: 0,
		},
		{
			name:       "blank-line removal is not a finding",
			diff:       hdr + "@@ -70 +69,0 @@\n-\n",
			violations: 0,
		},
		{
			name:       "deleted ADR flagged",
			diff:       "diff --git a/docs/adr/ADR-002-Gone.md b/docs/adr/ADR-002-Gone.md\ndeleted file mode 100644\n--- a/docs/adr/ADR-002-Gone.md\n+++ /dev/null\n@@ -1,2 +0,0 @@\n-# ADR-002\n-Accepted\n",
			violations: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removedLinesFromExistingADRs(tt.diff)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

// gitRun runs git in dir with an identity that needs no global config.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestCheckADRAppendOnly(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	const adr = "docs/adr/ADR-001-Foundation.md"
	const original = "# ADR-001\n\n## Status\n\nAccepted\n\n## Decision\n\nWe use Chi.\n"

	tests := []struct {
		name       string
		withBase   bool
		edit       map[string]string
		commit     bool
		violations int
	}{
		{name: "untouched ADR passes", withBase: true, violations: 0},
		{name: "appended note passes", withBase: true, edit: map[string]string{adr: original + "\n> **Amended**: still Chi.\n"}, violations: 0},
		{name: "uncommitted rewrite flagged", withBase: true, edit: map[string]string{adr: "# ADR-001\n\n## Status\n\nAccepted\n\n## Decision\n\nWe use Gin.\n"}, violations: 1},
		{name: "committed rewrite flagged", withBase: true, edit: map[string]string{adr: "# ADR-001\n\n## Status\n\nAccepted\n\n## Decision\n\nWe use Gin.\n"}, commit: true, violations: 1},
		{name: "new ADR is free to change", withBase: true, edit: map[string]string{"docs/adr/ADR-002-New.md": "# ADR-002\n\nProposed\n"}, violations: 0},
		{name: "no base ref means skipped, not failed", withBase: false, edit: map[string]string{adr: "# ADR-001\n\nrewritten\n"}, violations: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			gitRun(t, root, "init", "-q", "-b", "work")
			writeTree(t, root, map[string]string{adr: original})
			gitRun(t, root, "add", "-A")
			gitRun(t, root, "commit", "-q", "-m", "base")
			if tt.withBase {
				gitRun(t, root, "update-ref", "refs/remotes/origin/master", "HEAD")
			}
			writeTree(t, root, tt.edit)
			if tt.commit {
				gitRun(t, root, "add", "-A")
				gitRun(t, root, "commit", "-q", "-m", "edit")
			}
			got := checkADRAppendOnly(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}

	t.Run("not a git repository means skipped", func(t *testing.T) {
		root := t.TempDir()
		writeTree(t, root, map[string]string{filepath.FromSlash(adr): original})
		if got := checkADRAppendOnly(root); len(got) != 0 {
			t.Errorf("got %v, want none", got)
		}
	})
}
