// Command checkversions enforces the versions.json consumption contract
// (ADR-030): every key in versions.json must match its single in-repo source
// of truth (go.mod, package-lock.json, vendored JS, sqlc headers, workflow
// pins), so the manifest external consumers read can never drift from what
// the template actually ships. Run from the repo root.
//
// Usage:
//
//	go run ./scripts/checkversions          # check (task versions:check, part of task ci)
//	go run ./scripts/checkversions -write   # rewrite versions.json from the pins (task versions:sync)
//
// -write makes a pin bump a one-file edit: change go.mod (or whichever
// source), run the sync, commit both. The check is still the gate; the
// writer only ever produces what the check would accept.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// jsVersionRe matches the version literal embedded in the vendored htmx and
// Alpine minified bundles (e.g. `version:"1.9.10"` or `version = '3.13.3'`).
var jsVersionRe = regexp.MustCompile(`version["']?\s*[:=]\s*["']([0-9]+(?:\.[0-9]+)+)`)

// sqlcVersionRe matches the generator pin sqlc stamps into every generated
// file header (`//   sqlc v1.31.1`).
var sqlcVersionRe = regexp.MustCompile(`sqlc v([0-9]+(?:\.[0-9]+)+)`)

// migrateVersionRe matches the golang-migrate release pin in workflow
// download URLs.
var migrateVersionRe = regexp.MustCompile(`golang-migrate/migrate/releases/download/v([0-9]+(?:\.[0-9]+)+)`)

// nodeVersionRe matches the CI Node pin (`node-version: '20'`).
var nodeVersionRe = regexp.MustCompile(`node-version:\s*['"]?([0-9]+(?:\.[0-9]+)*)`)

// templateRe is the only key not checked against a source file: the release
// workflow stamps it from the git tag, so CI just enforces the tag format.
var templateRe = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// goModVersions extracts the language version (`go` directive), toolchain
// version, and direct (non-indirect) requirement versions from go.mod
// content. Requirement keys are full module paths; versions are stripped of
// their leading "v".
func goModVersions(content string) map[string]string {
	out := map[string]string{}
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "go "):
			out["go"] = strings.TrimPrefix(line, "go ")
		case strings.HasPrefix(line, "toolchain go"):
			out["toolchain"] = strings.TrimPrefix(line, "toolchain go")
		case strings.HasPrefix(line, "require ("):
			inBlock = true
		case inBlock && line == ")":
			inBlock = false
		case inBlock && line != "" && !strings.Contains(line, "// indirect"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				out[fields[0]] = strings.TrimPrefix(fields[1], "v")
			}
		}
	}
	return out
}

// lockfileVersion extracts an exact package version from package-lock.json.
func lockfileVersion(content []byte, pkg string) (string, error) {
	var lock struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(content, &lock); err != nil {
		return "", fmt.Errorf("parse package-lock.json: %w", err)
	}
	entry, ok := lock.Packages["node_modules/"+pkg]
	if !ok {
		return "", fmt.Errorf("package %q not found in package-lock.json", pkg)
	}
	return entry.Version, nil
}

// firstMatch returns the first capture group of re in the named file.
func firstMatch(path string, re *regexp.Regexp) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	m := re.FindSubmatch(content)
	if m == nil {
		return "", fmt.Errorf("%s: no match for %s", path, re)
	}
	return string(m[1]), nil
}

// allMatchesAgree returns the shared capture-group value of re across the
// given files, erroring if any file has no match or the files disagree —
// used for pins that appear in more than one workflow.
func allMatchesAgree(re *regexp.Regexp, paths ...string) (string, error) {
	value := ""
	for _, path := range paths {
		v, err := firstMatch(path, re)
		if err != nil {
			return "", err
		}
		if value != "" && v != value {
			return "", fmt.Errorf("%s pins %s but an earlier file pins %s", path, v, value)
		}
		value = v
	}
	return value, nil
}

func expectedVersions() (map[string]string, error) {
	goMod, err := os.ReadFile("go.mod")
	if err != nil {
		return nil, err
	}
	mod := goModVersions(string(goMod))

	lock, err := os.ReadFile("package-lock.json")
	if err != nil {
		return nil, err
	}
	tailwind, err := lockfileVersion(lock, "tailwindcss")
	if err != nil {
		return nil, err
	}

	htmx, err := firstMatch("web/static/js/htmx.min.js", jsVersionRe)
	if err != nil {
		return nil, err
	}
	alpine, err := firstMatch("web/static/js/alpine.min.js", jsVersionRe)
	if err != nil {
		return nil, err
	}
	sqlc, err := firstMatch("internal/database/db.go", sqlcVersionRe)
	if err != nil {
		return nil, err
	}
	migrateVersion, err := allMatchesAgree(migrateVersionRe,
		".github/workflows/ci.yml", ".github/workflows/release.yml")
	if err != nil {
		return nil, err
	}
	node, err := firstMatch(".github/workflows/ci.yml", nodeVersionRe)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"go":             mod["toolchain"],
		"go-minimum":     mod["go"],
		"chi":            mod["github.com/go-chi/chi/v5"],
		"templ":          mod["github.com/a-h/templ"],
		"pgx":            mod["github.com/jackc/pgx/v5"],
		"gotrue-go":      mod["github.com/supabase-community/gotrue-go"],
		"supabase-go":    mod["github.com/supabase-community/supabase-go"],
		"htmx":           htmx,
		"alpine":         alpine,
		"tailwindcss":    tailwind,
		"sqlc":           sqlc,
		"golang-migrate": migrateVersion,
		"node":           node,
	}, nil
}

// manifestKeys returns the top-level keys of a JSON object in file order —
// consumers diff versions.json, so -write must not reshuffle it the way a
// map round-trip would.
func manifestKeys(raw []byte) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("versions.json is not a JSON object")
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("versions.json: unexpected token %v", tok)
		}
		keys = append(keys, key)
		// Consume (and discard) the value; the string values are re-read
		// through the map below.
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// syncManifest rewrites raw (the current versions.json) so every key with a
// wired source of truth carries the repo's actual pin. It preserves key
// order, leaves the release-stamped template field and any unknown keys
// alone (the check rejects those; the writer must not hide them), and
// appends newly wired keys at the end. It returns the new file content and a
// human-readable change list; content is byte-identical to raw when nothing
// changed.
func syncManifest(raw []byte, expected map[string]string) ([]byte, []string, error) {
	var manifest map[string]string
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, nil, fmt.Errorf("parse versions.json: %w", err)
	}
	keys, err := manifestKeys(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("parse versions.json: %w", err)
	}

	var changes []string
	values := make(map[string]string, len(manifest))
	for _, key := range keys {
		values[key] = manifest[key]
		want, wired := expected[key]
		if key == "template" || !wired || want == "" || want == manifest[key] {
			continue
		}
		changes = append(changes, fmt.Sprintf("%s: %q -> %q", key, manifest[key], want))
		values[key] = want
	}
	// Newly wired keys (present in expected, absent from the file) are
	// appended in a stable order so the output is deterministic.
	var added []string
	for key, want := range expected {
		if _, present := manifest[key]; present || key == "template" || want == "" {
			continue
		}
		added = append(added, key)
	}
	sortStrings(added)
	for _, key := range added {
		keys = append(keys, key)
		values[key] = expected[key]
		changes = append(changes, fmt.Sprintf("%s: (missing) -> %q", key, expected[key]))
	}

	var buf bytes.Buffer
	buf.WriteString("{\n")
	for i, key := range keys {
		k, _ := json.Marshal(key)
		v, _ := json.Marshal(values[key])
		fmt.Fprintf(&buf, "  %s: %s", k, v)
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("}\n")
	return buf.Bytes(), changes, nil
}

// sortStrings is an insertion sort — the slices here are a handful of keys,
// and it keeps the script free of imports beyond the ones it already has.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func main() {
	write := flag.Bool("write", false, "rewrite versions.json from the repo's pins instead of checking it")
	flag.Parse()

	manifestRaw, err := os.ReadFile("versions.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
	var manifest map[string]string
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		fmt.Fprintf(os.Stderr, "❌ parse versions.json: %v\n", err)
		os.Exit(1)
	}

	expected, err := expectedVersions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	if *write {
		synced, changes, err := syncManifest(manifestRaw, expected)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		if len(changes) == 0 {
			fmt.Println("✅ versions.json already matches the repo's actual pins — nothing written")
			return
		}
		if err := os.WriteFile("versions.json", synced, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "❌ write versions.json: %v\n", err)
			os.Exit(1)
		}
		for _, c := range changes {
			fmt.Printf("   %s\n", c)
		}
		fmt.Printf("✅ versions.json synced (%d key(s) updated) — review and commit it with the pin change\n", len(changes))
		return
	}

	failed := false
	fail := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "❌ "+format+"\n", args...)
		failed = true
	}

	if !templateRe.MatchString(manifest["template"]) {
		fail("versions.json template %q is not a vX.Y.Z tag", manifest["template"])
	}
	for key, want := range expected {
		if want == "" {
			fail("no in-repo source of truth found for %q", key)
			continue
		}
		if got := manifest[key]; got != want {
			fail("versions.json %s = %q but the repo actually pins %q", key, got, want)
		}
	}
	// Additive keys are fine for consumers, but a key with no source of
	// truth here would drift silently — force it to be wired up.
	for key := range manifest {
		if key == "template" {
			continue
		}
		if _, ok := expected[key]; !ok {
			fail("versions.json key %q has no check wired in scripts/checkversions", key)
		}
	}

	if failed {
		os.Exit(1)
	}
	fmt.Printf("✅ versions.json matches the repo's actual pins (%d keys)\n", len(manifest))
}
