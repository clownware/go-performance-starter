// Command adrcheck runs the warn-only ADR enforcement suite (ADR-033).
//
// Each check verifies a testable consequence (TC) declared in an ADR's
// Enforcement section. Check status comes from checks/enforcement.config.json:
// "warn" failures report but never fail the run; "block" failures exit
// non-zero. Promotion from warn to block is governed by the graduation rule
// in ADR-033.
//
// Usage: go run ./scripts/adrcheck [--json]  (from the repo root)
package main

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type CheckConfig struct {
	ID        string `json:"id"`
	ADR       string `json:"adr"`
	TC        string `json:"tc"`
	Status    string `json:"status"` // "warn" | "block"
	Added     string `json:"added"`
	Graduated string `json:"graduated,omitempty"`
}

type CheckResult struct {
	CheckConfig
	Result     string   `json:"result"` // "PASS" | "WARNING" | "BLOCKER"
	Violations []string `json:"violations,omitempty"`
	Remedy     string   `json:"remedy,omitempty"`
}

type checkFn func(root string) []string

var checks = map[string]struct {
	fn     checkFn
	remedy string
}{
	"adr002-migration-pairs":     {checkMigrationPairs, "add the missing .up.sql/.down.sql twin in migrations/ (ADR-002)"},
	"adr003-no-sql-in-handlers":  {checkNoSQLInHandlers, "move the query to sql/queries/, run `task db:generate`, call it through a repository interface (ADR-003)"},
	"adr007-no-heavy-frameworks": {checkNoHeavyFrameworks, "remove the framework; UI is server-rendered templ + HTMX, Alpine for light interactivity (ADR-007)"},
	"adr011-adr-metadata":        {checkADRMetadata, "rename to ADR-NNN-Title.md and/or add a Status marker (ADR-011)"},
	"adr015-env-only-config":     {checkEnvOnlyConfig, "read the value in internal/config and pass it down; keep .env gitignored with .env.example current (ADR-015)"},
	"adr017-no-html-template":    {checkNoHTMLTemplate, "render with templ in internal/view instead of html/template (ADR-017)"},
	"adr017-typed-view-props":    {checkTypedViewProps, "define a typed props struct; map[string]interface{} is forbidden in internal/view (ADR-017)"},
	"adr025-deploy-scope":        {checkDeployScope, "keep exactly one fly.toml at the repo root and no Terraform/k8s manifests (ADR-025)"},
	"adr026-slog-only":           {checkSlogOnly, "use log/slog with structured fields; zerolog and log.Printf are retired (ADR-026)"},
	"adr021-ci-invokes-gate":     {checkCIInvokesGate, "have .github/workflows/ci.yml run `task ci` instead of re-implementing gate steps (ADR-021)"},
}

// walkGoFiles calls fn for every non-generated .go file under dir (relative
// to root), skipping vendored/build trees. Test files are skipped unless
// includeTests is set.
func walkGoFiles(root, dir string, includeTests bool, fn func(rel string, content []byte)) {
	base := filepath.Join(root, dir)
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "vendor" || name == "tmp" || name == "dist" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_templ.go") {
			return nil
		}
		if !includeTests && strings.HasSuffix(name, "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		fn(rel, content)
		return nil
	})
}

func lineOf(content []byte, idx int) int {
	return strings.Count(string(content[:idx]), "\n") + 1
}

func checkMigrationPairs(root string) []string {
	var violations []string
	dir := filepath.Join(root, "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{"migrations/ directory not readable"}
	}
	ups := map[string]bool{}
	downs := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		switch {
		case strings.HasSuffix(name, ".up.sql"):
			ups[strings.TrimSuffix(name, ".up.sql")] = true
		case strings.HasSuffix(name, ".down.sql"):
			downs[strings.TrimSuffix(name, ".down.sql")] = true
		default:
			violations = append(violations, fmt.Sprintf("migrations/%s: not a paired raw-SQL migration file", name))
		}
	}
	for base := range ups {
		if !downs[base] {
			violations = append(violations, fmt.Sprintf("migrations/%s.up.sql has no matching .down.sql", base))
		}
	}
	for base := range downs {
		if !ups[base] {
			violations = append(violations, fmt.Sprintf("migrations/%s.down.sql has no matching .up.sql", base))
		}
	}
	return violations
}

var (
	sqlLiteralRe   = regexp.MustCompile("(?i)[\"`](SELECT\\s+.+\\s+FROM\\s|INSERT\\s+INTO\\s|UPDATE\\s+\\S+\\s+SET\\s|DELETE\\s+FROM\\s)")
	envReadRe      = regexp.MustCompile(`\bos\.(Getenv|LookupEnv)\(`)
	logPrintRe     = regexp.MustCompile(`\blog\.Print(f|ln)?\(`)
	untypedPropsRe = regexp.MustCompile(`map\[string\](interface\{\}|any)`)
	adrNameRe      = regexp.MustCompile(`^ADR-\d{3}-[A-Za-z0-9-]+\.md$`)
	// The repo's ADRs mark acceptance three ways: a `## Status` heading,
	// a `**Status**:` bold label, or a `*   **Status:**` list item.
	adrStatusRe = regexp.MustCompile(`(?mi)^(\s*[*-]\s*)?(#{1,4}\s+status\b|\*\*status:?\*\*)`)
)

// goImports returns the import paths of a Go file, with the line each
// appears on. String literals elsewhere in the file are not imports.
func goImports(rel string, content []byte) map[string]int {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, content, parser.ImportsOnly)
	if err != nil {
		return nil
	}
	imports := make(map[string]int, len(f.Imports))
	for _, imp := range f.Imports {
		imports[strings.Trim(imp.Path.Value, `"`)] = fset.Position(imp.Path.Pos()).Line
	}
	return imports
}

func checkNoSQLInHandlers(root string) []string {
	var violations []string
	walkGoFiles(root, "internal/handler", false, func(rel string, content []byte) {
		if loc := sqlLiteralRe.FindIndex(content); loc != nil {
			violations = append(violations, fmt.Sprintf("%s:%d: hand-written SQL string literal", rel, lineOf(content, loc[0])))
		}
		for path, line := range goImports(rel, content) {
			if strings.HasPrefix(path, "github.com/jackc/pgx") || path == "database/sql" {
				violations = append(violations, fmt.Sprintf("%s:%d: handler imports a database driver (%s)", rel, line, path))
			}
		}
	})
	return violations
}

var heavyFrameworks = []string{"react", "react-dom", "vue", "svelte", "angular", "@angular/core", "jquery", "next", "nuxt", "preact", "solid-js"}

func checkNoHeavyFrameworks(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nil // no package.json is fine
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return []string{"package.json: unparseable"}
	}
	var violations []string
	for _, banned := range heavyFrameworks {
		if _, ok := pkg.Dependencies[banned]; ok {
			violations = append(violations, fmt.Sprintf("package.json: dependency %q is a heavy client framework", banned))
		}
		if _, ok := pkg.DevDependencies[banned]; ok {
			violations = append(violations, fmt.Sprintf("package.json: devDependency %q is a heavy client framework", banned))
		}
	}
	return violations
}

func checkADRMetadata(root string) []string {
	var violations []string
	entries, err := os.ReadDir(filepath.Join(root, "docs", "adr"))
	if err != nil {
		return []string{"docs/adr/ not readable"}
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "ADR-") || !strings.HasSuffix(name, ".md") {
			continue
		}
		if !adrNameRe.MatchString(name) {
			violations = append(violations, fmt.Sprintf("docs/adr/%s: does not match ADR-NNN-Title.md", name))
			continue
		}
		content, err := os.ReadFile(filepath.Join(root, "docs", "adr", name))
		if err != nil || !adrStatusRe.Match(content) {
			violations = append(violations, fmt.Sprintf("docs/adr/%s: no Status marker found", name))
		}
	}
	return violations
}

// Bootstrap files that legitimately read the environment before config exists.
var envReadAllowlist = map[string]bool{
	filepath.Join("cmd", "api", "main.go"): true,
}

func checkEnvOnlyConfig(root string) []string {
	var violations []string
	for _, dir := range []string{"cmd", "internal"} {
		walkGoFiles(root, dir, false, func(rel string, content []byte) {
			if envReadAllowlist[rel] || strings.HasPrefix(rel, filepath.Join("internal", "config")+string(filepath.Separator)) {
				return
			}
			if loc := envReadRe.FindIndex(content); loc != nil {
				violations = append(violations, fmt.Sprintf("%s:%d: environment read outside internal/config", rel, lineOf(content, loc[0])))
			}
		})
	}
	if _, err := os.Stat(filepath.Join(root, ".env.example")); err != nil {
		violations = append(violations, ".env.example missing at repo root")
	}
	gi, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil || !strings.Contains(string(gi), ".env") {
		violations = append(violations, ".gitignore does not ignore .env")
	}
	return violations
}

func checkNoHTMLTemplate(root string) []string {
	var violations []string
	for _, dir := range []string{"cmd", "internal", "scripts"} {
		walkGoFiles(root, dir, true, func(rel string, content []byte) {
			if line, ok := goImports(rel, content)["html/template"]; ok {
				violations = append(violations, fmt.Sprintf("%s:%d: imports html/template", rel, line))
			}
		})
	}
	return violations
}

func checkTypedViewProps(root string) []string {
	var violations []string
	base := filepath.Join(root, "internal", "view")
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		isGo := strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_templ.go")
		isTempl := strings.HasSuffix(name, ".templ")
		if !isGo && !isTempl {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if loc := untypedPropsRe.FindIndex(content); loc != nil {
			rel, _ := filepath.Rel(root, path)
			violations = append(violations, fmt.Sprintf("%s:%d: untyped props map in internal/view", rel, lineOf(content, loc[0])))
		}
		return nil
	})
	return violations
}

func checkDeployScope(root string) []string {
	var violations []string
	rootFly := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "vendor" || name == "tmp" || name == "dist" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		switch {
		case d.Name() == "fly.toml":
			if rel == "fly.toml" {
				rootFly = true
			} else {
				violations = append(violations, fmt.Sprintf("%s: deployment config outside the single root fly.toml", rel))
			}
		case strings.HasSuffix(d.Name(), ".tf"):
			violations = append(violations, fmt.Sprintf("%s: Terraform is out of template scope", rel))
		}
		return nil
	})
	if !rootFly {
		violations = append(violations, "fly.toml missing at repo root (ADR-025 permits exactly one, as the worked example)")
	}
	return violations
}

func checkSlogOnly(root string) []string {
	var violations []string
	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err == nil && strings.Contains(string(gomod), "rs/zerolog") {
		violations = append(violations, "go.mod: github.com/rs/zerolog is retired (ADR-026)")
	}
	for _, dir := range []string{"cmd", "internal"} {
		walkGoFiles(root, dir, false, func(rel string, content []byte) {
			if loc := logPrintRe.FindIndex(content); loc != nil {
				violations = append(violations, fmt.Sprintf("%s:%d: log.Print* call site — use log/slog", rel, lineOf(content, loc[0])))
			}
		})
	}
	return violations
}

func checkCIInvokesGate(root string) []string {
	content, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		return []string{".github/workflows/ci.yml not readable"}
	}
	if !strings.Contains(string(content), "task ci") {
		return []string{".github/workflows/ci.yml does not invoke `task ci`"}
	}
	return nil
}

// runSuite executes every configured check. failed is true iff a
// block-status check found violations or a config entry names no wired check.
func runSuite(root string, config []CheckConfig) (results []CheckResult, failed bool) {
	for _, cfg := range config {
		res := CheckResult{CheckConfig: cfg, Result: "PASS"}
		impl, ok := checks[cfg.ID]
		if !ok {
			res.Result = "BLOCKER"
			res.Violations = []string{fmt.Sprintf("config names check %q but no such check is wired (a key with no wired check fails, same rule as ADR-030)", cfg.ID)}
			res.Remedy = "wire the check in scripts/adrcheck or remove the config entry"
			failed = true
			results = append(results, res)
			continue
		}
		if violations := impl.fn(root); len(violations) > 0 {
			res.Violations = violations
			res.Remedy = impl.remedy
			if cfg.Status == "block" {
				res.Result = "BLOCKER"
				failed = true
			} else {
				res.Result = "WARNING"
			}
		}
		results = append(results, res)
	}
	return results, failed
}

func main() {
	jsonOut := len(os.Args) > 1 && os.Args[1] == "--json"

	data, err := os.ReadFile(filepath.Join("checks", "enforcement.config.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "adrcheck: cannot read checks/enforcement.config.json: %v (run from the repo root)\n", err)
		os.Exit(1)
	}
	var config []CheckConfig
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Fprintf(os.Stderr, "adrcheck: invalid config: %v\n", err)
		os.Exit(1)
	}

	results, failed := runSuite(".", config)

	pass, warn, block := 0, 0, 0
	for _, r := range results {
		switch r.Result {
		case "PASS":
			pass++
		case "WARNING":
			warn++
		case "BLOCKER":
			block++
		}
	}

	if jsonOut {
		out, _ := json.MarshalIndent(map[string]any{
			"results": results,
			"summary": map[string]int{"pass": pass, "warning": warn, "blocker": block},
		}, "", "  ")
		fmt.Println(string(out))
	} else {
		for _, r := range results {
			fmt.Printf("%-8s %-28s (%s, %s)\n", r.Result, r.ID, r.ADR, r.TC)
			for _, v := range r.Violations {
				fmt.Printf("         %s\n", v)
			}
			if r.Remedy != "" {
				fmt.Printf("         remedy: %s\n", r.Remedy)
			}
		}
		fmt.Printf("SUMMARY: %d pass, %d warning, %d blocker\n", pass, warn, block)
	}

	if failed {
		os.Exit(1)
	}
}
