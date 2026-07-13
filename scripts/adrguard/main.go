// Command adrguard is the PreToolUse hook for Edit/Write (ADR-033).
//
// It reads the hook payload from stdin and denies writes to paths the ADRs
// protect: existing ADRs (append-only historical record), AGENTS.md
// (generated, ADR-022), and sqlc/templ-generated code (ADR-019). Everything
// else passes through untouched.
//
// Kill-switch: ADR_GUARD_OFF=1 disables the guard for one operator-authorized
// amendment.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type hookInput struct {
	CWD       string `json:"cwd"`
	ToolInput struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
}

// decide returns a denial reason for protected paths, or "" to allow.
// root is the repo root; absPath is the file the tool wants to write.
func decide(root, absPath string) string {
	rel, err := filepath.Rel(root, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "" // outside the repo — not this guard's business
	}
	rel = filepath.ToSlash(rel)

	switch {
	case rel == "AGENTS.md":
		return "AGENTS.md is generated (ADR-022). Legal move: edit the source layer (CLAUDE.md or .claude/*.md) and run `task agents:build`. Kill-switch: ADR_GUARD_OFF=1."
	case strings.HasPrefix(rel, "internal/database/") && strings.HasSuffix(rel, ".go"):
		return "internal/database/ is sqlc-generated — never hand-edit (ADR-019). Legal move: change sql/queries/ or sql/schema and run `task db:generate`. Kill-switch: ADR_GUARD_OFF=1."
	case strings.HasPrefix(rel, "internal/view/") && strings.HasSuffix(rel, "_templ.go"):
		return "*_templ.go files are templ-generated — never hand-edit (ADR-019). Legal move: edit the .templ source and run `task templ:generate`. Kill-switch: ADR_GUARD_OFF=1."
	case strings.HasPrefix(rel, "docs/adr/ADR-") && strings.HasSuffix(rel, ".md"):
		if _, err := os.Stat(absPath); err == nil {
			return fmt.Sprintf("%s is an existing ADR — the record is append-only (ADR-033). Legal moves: append to its Enforcement section (graduation log) via an operator-reviewed change, or supersede it with a new ADR. Kill-switch: ADR_GUARD_OFF=1.", rel)
		}
		return "" // creating a new ADR is legal
	}
	return ""
}

func main() {
	if os.Getenv("ADR_GUARD_OFF") == "1" {
		return
	}
	var in hookInput
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil || in.ToolInput.FilePath == "" {
		return // unparseable or non-file tool call — allow
	}
	root := in.CWD
	if root == "" {
		root, _ = os.Getwd()
	}
	absPath := in.ToolInput.FilePath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, absPath)
	}
	reason := decide(root, absPath)
	if reason == "" {
		return
	}
	out, _ := json.Marshal(map[string]any{
		"hookSpecificOutput": map[string]string{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": reason,
		},
	})
	fmt.Println(string(out))
}
