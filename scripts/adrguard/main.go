// Command adrguard is the PreToolUse hook for Edit/Write/MultiEdit (ADR-033).
//
// It reads the hook payload from stdin and gives a verdict on writes to the
// paths the ADRs protect:
//
//   - An existing ADR: ask. The record is append-only, and whether an edit is
//     an append is a judgement, so the edit goes in front of the operator. A
//     session with nobody to approve is refused. A new ADR gets no verdict.
//   - AGENTS.md (generated, ADR-022) and sqlc/templ-generated code (ADR-019):
//     deny. A generated file has no legal hand-edit.
//
// Everything else passes through untouched.
//
// Kill-switch: ADR_GUARD_OFF=1 silences the denials for one operator-authorized
// session. It does not silence the ask: approving the prompt is the way
// through, so no switch is needed (#169).
package main

import (
	"encoding/json"
	"fmt"
	"io"
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

const (
	decisionAsk  = "ask"
	decisionDeny = "deny"
)

// decide returns the guard's verdict on one write — "ask", "deny" or "" to
// leave the normal permission flow alone — and the reason shown with it.
// root is the repo root; absPath is the file the tool wants to write.
func decide(root, absPath string) (decision, reason string) {
	rel, err := filepath.Rel(root, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", "" // outside the repo — not this guard's business
	}
	rel = filepath.ToSlash(rel)

	switch {
	case rel == "AGENTS.md":
		return decisionDeny, "AGENTS.md is generated (ADR-022). Legal move: edit the source layer (CLAUDE.md or .claude/*.md) and run `task agents:build`. Kill-switch: ADR_GUARD_OFF=1."
	case strings.HasPrefix(rel, "internal/database/") && strings.HasSuffix(rel, ".go"):
		return decisionDeny, "internal/database/ is sqlc-generated — never hand-edit (ADR-019). Legal move: change sql/queries/ or sql/schema and run `task db:generate`. Kill-switch: ADR_GUARD_OFF=1."
	case strings.HasPrefix(rel, "internal/view/") && strings.HasSuffix(rel, "_templ.go"):
		return decisionDeny, "*_templ.go files are templ-generated — never hand-edit (ADR-019). Legal move: edit the .templ source and run `task templ:generate`. Kill-switch: ADR_GUARD_OFF=1."
	case strings.HasPrefix(rel, "docs/adr/ADR-") && strings.HasSuffix(rel, ".md"):
		if _, err := os.Stat(absPath); err == nil {
			return decisionAsk, fmt.Sprintf("%s is an existing ADR, and the record is append-only (ADR-033). Legal moves: append a dated note or a graduation-log entry, change the Status line, or supersede it with a new ADR. Never rewrite its text to make a change comply. This edit needs the operator's approval.", rel)
		}
		return "", "" // creating a new ADR is legal
	}
	return "", ""
}

// hookOutput turns one PreToolUse payload into the hook's stdout: a
// permission decision, or "" to say nothing. off is the kill-switch.
func hookOutput(in io.Reader, off bool) string {
	var input hookInput
	if err := json.NewDecoder(in).Decode(&input); err != nil || input.ToolInput.FilePath == "" {
		return "" // unparseable or non-file tool call — allow
	}
	root := input.CWD
	if root == "" {
		root, _ = os.Getwd()
	}
	absPath := input.ToolInput.FilePath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, absPath)
	}
	decision, reason := decide(root, absPath)
	if decision == "" || (off && decision == decisionDeny) {
		return ""
	}
	out, _ := json.Marshal(map[string]any{
		"hookSpecificOutput": map[string]string{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       decision,
			"permissionDecisionReason": reason,
		},
	})
	return string(out)
}

func main() {
	if out := hookOutput(os.Stdin, os.Getenv("ADR_GUARD_OFF") == "1"); out != "" {
		fmt.Println(out)
	}
}
