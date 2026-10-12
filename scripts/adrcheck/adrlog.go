package main

// Checks over the ADR log itself: the index is complete (ADR-011) and
// existing records are append-only (ADR-033).

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// An index row is a table line whose first cell links the ADR file.
var adrIndexRowRe = regexp.MustCompile(`(?m)^\|\s*\[[^\]]+\]\((ADR-\d{3}-[A-Za-z0-9-]+\.md)\)`)

// checkADRIndexComplete verifies docs/adr/README.md has one table row per
// ADR file and no row for a file that is not there.
func checkADRIndexComplete(root string) []string {
	dir := filepath.Join(root, "docs", "adr")
	index, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		return []string{"docs/adr/README.md: index not readable"}
	}
	rows := map[string]bool{}
	for _, m := range adrIndexRowRe.FindAllSubmatch(index, -1) {
		rows[string(m[1])] = true
	}
	files := map[string]bool{}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if adrNameRe.MatchString(e.Name()) {
			files[e.Name()] = true
		}
	}
	var violations []string
	for name := range files {
		if !rows[name] {
			violations = append(violations, fmt.Sprintf("docs/adr/%s: no row in docs/adr/README.md", name))
		}
	}
	for name := range rows {
		if !files[name] {
			violations = append(violations, fmt.Sprintf("docs/adr/README.md: row links %s, which does not exist", name))
		}
	}
	sort.Strings(violations)
	return violations
}

var (
	// A Status line in any of the log's five layouts: a bare status word,
	// or one behind a `**Status**:` label or list bullet.
	adrStatusLineRe = regexp.MustCompile(`(?i)^[\s*>-]*(\*\*status:?\*\*:?\s*)?(proposed|accepted|deprecated|superseded)\b`)
	hunkHeaderRe    = regexp.MustCompile(`^@@ -(\d+)`)
)

// removalAllowed reports whether dropping a line from an existing ADR is a
// legal append-only move: the Status line may change, the empty
// graduation-log placeholder may be filled, and a line may be extended or
// struck through as long as its text survives in the same hunk.
func removalAllowed(removed string, added []string) bool {
	text := strings.TrimSpace(removed)
	if text == "" || adrStatusLineRe.MatchString(text) || strings.Contains(text, "_(empty") {
		return true
	}
	// Compare without the list or quote marker, so `- text` survives as
	// `- ~~text~~`.
	body := strings.TrimLeft(text, "-*> \t")
	if body == "" {
		return true
	}
	for _, a := range added {
		if strings.Contains(a, body) {
			return true
		}
	}
	return false
}

// removedLinesFromExistingADRs reads a unified diff (-U0) of existing ADR
// files and returns one finding per line that was removed or rewritten, and
// one per deleted file.
func removedLinesFromExistingADRs(diff string) []string {
	var (
		violations []string
		file       string
		deleted    bool
		oldLine    int
		removed    []string
		removedAt  []int
		added      []string
	)
	flush := func() {
		for i, r := range removed {
			if !removalAllowed(r, added) {
				violations = append(violations, fmt.Sprintf("%s:%d: existing text removed or rewritten: %q", file, removedAt[i], strings.TrimSpace(r)))
			}
		}
		removed, removedAt, added = nil, nil, nil
	}
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flush()
			file, deleted = "", false
		case strings.HasPrefix(line, "--- a/"):
			file = strings.TrimPrefix(line, "--- a/")
		case line == "+++ /dev/null":
			deleted = true
			violations = append(violations, fmt.Sprintf("%s: existing ADR deleted or renamed", file))
		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ "):
		case strings.HasPrefix(line, "@@"):
			flush()
			if m := hunkHeaderRe.FindStringSubmatch(line); m != nil {
				oldLine, _ = strconv.Atoi(m[1])
			}
		case deleted || file == "":
		case strings.HasPrefix(line, "-"):
			removed = append(removed, line[1:])
			removedAt = append(removedAt, oldLine)
			oldLine++
		case strings.HasPrefix(line, "+"):
			added = append(added, line[1:])
		}
	}
	flush()
	return violations
}

// adrBaseRefs are tried in order to find where this branch left the
// default branch.
var adrBaseRefs = []string{"origin/master", "origin/main"}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.Output()
	return string(out), err
}

// checkADRAppendOnly compares every ADR that already exists on the default
// branch with the working tree and reports removed or rewritten lines. The
// PreToolUse guard stops an agent's Edit/Write on these files; this catches
// what the guard cannot see — a shell write, or a human's editor. With no
// reachable base (a shallow clone, a fresh fork, not a repository) there is
// nothing to compare against, so the check says so and passes.
func checkADRAppendOnly(root string) []string {
	base := mergeBase(root)
	if base == "" {
		fmt.Fprintf(os.Stderr, "adr033-append-only: skipped — no merge base with %s (shallow clone or no remote)\n", strings.Join(adrBaseRefs, " or "))
		return nil
	}
	diff, err := gitOutput(root, "diff", "--no-color", "--no-ext-diff", "--no-renames", "-U0", "--diff-filter=MD", base, "--", "docs/adr/ADR-*.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "adr033-append-only: skipped — git diff against %s failed: %v\n", base, err)
		return nil
	}
	return removedLinesFromExistingADRs(diff)
}

// mergeBase returns where this branch left the default branch, or "" when
// there is nothing to compare against.
func mergeBase(root string) string {
	for _, ref := range adrBaseRefs {
		if out, err := gitOutput(root, "merge-base", "HEAD", ref); err == nil {
			return strings.TrimSpace(out)
		}
	}
	return ""
}

// governedPaths are the files that state or enforce the rules (ADR-019,
// "Governed layer"): the constitution and its workflow and engineering
// layers, the roles, the hooks and their wiring, the check configuration,
// and the two ADR scripts. `.claude/stack.md` is facts, not rules, and is
// left out.
var governedPaths = []string{
	"CLAUDE.md",
	".claude/engineering.md",
	".claude/workflow.md",
	".claude/roles",
	".claude/hooks",
	".claude/settings.json",
	"checks",
	"scripts/adrcheck",
	"scripts/adrguard",
}

// checkGovernedLayer reports every governed file that differs from the
// default branch. It cannot tell a change made for its own reason from one
// made so that something else passes — nothing can — so it never blocks: it
// puts the edit in front of the reviewer, who checks that the PR names the
// ADR note authorising it. Without the documents, every agent in the
// 2026-10 test that renamed the `ci` task also rewrote the check guarding
// it (#162, #165).
func checkGovernedLayer(root string) []string {
	base := mergeBase(root)
	if base == "" {
		fmt.Fprintf(os.Stderr, "adr019-governed-layer: skipped — no merge base with %s (shallow clone or no remote)\n", strings.Join(adrBaseRefs, " or "))
		return nil
	}
	changed, err := gitOutput(root, append([]string{"diff", "--name-only", "--no-renames", base, "--"}, governedPaths...)...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "adr019-governed-layer: skipped — git diff against %s failed: %v\n", base, err)
		return nil
	}
	untracked, _ := gitOutput(root, append([]string{"ls-files", "--others", "--exclude-standard", "--"}, governedPaths...)...)
	seen := map[string]bool{}
	var violations []string
	for _, f := range strings.Split(changed+"\n"+untracked, "\n") {
		f = strings.TrimSpace(f)
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		violations = append(violations, fmt.Sprintf("%s: governed file changed on this branch", f))
	}
	sort.Strings(violations)
	return violations
}
