package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTree lays out files under root; keys are relative paths.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCheckMigrationPairs(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name: "paired migrations pass",
			files: map[string]string{
				"migrations/000001_init.up.sql":   "CREATE TABLE t (id int);",
				"migrations/000001_init.down.sql": "DROP TABLE t;",
			},
			violations: 0,
		},
		{
			name: "missing down migration flagged",
			files: map[string]string{
				"migrations/000002_orphan.up.sql": "CREATE TABLE o (id int);",
			},
			violations: 1,
		},
		{
			name: "missing up migration flagged",
			files: map[string]string{
				"migrations/000003_orphan.down.sql": "DROP TABLE o;",
			},
			violations: 1,
		},
		{
			name: "non-sql file in migrations flagged",
			files: map[string]string{
				"migrations/notes.txt": "not a migration",
			},
			violations: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkMigrationPairs(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestCheckNoSQLInHandlers(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name: "repository-interface handler passes",
			files: map[string]string{
				"internal/handler/clean.go": "package handler\n\nfunc List(repo repository.QuizRepository) {}\n",
			},
			violations: 0,
		},
		{
			name: "sqlc type usage passes",
			files: map[string]string{
				"internal/handler/types.go": "package handler\n\nimport \"github.com/clownware/go-performance-starter/internal/database\"\n\nvar _ database.QuizQuestion\n",
			},
			violations: 0,
		},
		{
			name: "raw SELECT literal flagged",
			files: map[string]string{
				"internal/handler/bad.go": "package handler\n\nconst q = \"SELECT id FROM users WHERE email = $1\"\n",
			},
			violations: 1,
		},
		{
			name: "pgx import flagged",
			files: map[string]string{
				"internal/handler/bad.go": "package handler\n\nimport \"github.com/jackc/pgx/v5\"\n",
			},
			violations: 1,
		},
		{
			name: "database/sql import flagged",
			files: map[string]string{
				"internal/handler/bad.go": "package handler\n\nimport \"database/sql\"\n",
			},
			violations: 1,
		},
		{
			name: "test files exempt",
			files: map[string]string{
				"internal/handler/x_test.go": "package handler\n\nconst q = \"SELECT 1 FROM t\"\n",
			},
			violations: 0,
		},
		{
			name: "comment naming a query is not a finding",
			files: map[string]string{
				"internal/handler/c.go": "package handler\n\n// was: \"SELECT id FROM users WHERE id = $1\"\n/* `DELETE FROM users` moved to sql/queries */\nfunc f() {}\n",
			},
			violations: 0,
		},
		{
			name: "literal after a comment is still flagged on its own line",
			files: map[string]string{
				"internal/handler/d.go": "package handler\n\n// note\nconst q = \"SELECT id FROM users WHERE id = $1\" // trailing\n",
			},
			violations: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkNoSQLInHandlers(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestCheckNoHeavyFrameworks(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name: "tailwind-only passes",
			files: map[string]string{
				"package.json": `{"devDependencies":{"tailwindcss":"^4.0.0","@tailwindcss/cli":"^4.3.2"}}`,
			},
			violations: 0,
		},
		{
			name: "react dependency flagged",
			files: map[string]string{
				"package.json": `{"dependencies":{"react":"^19.0.0"}}`,
			},
			violations: 1,
		},
		{
			name: "vue devDependency flagged",
			files: map[string]string{
				"package.json": `{"devDependencies":{"vue":"^3.0.0"}}`,
			},
			violations: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkNoHeavyFrameworks(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestCheckADRMetadata(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name: "heading-style Status passes",
			files: map[string]string{
				"docs/adr/ADR-001-Foundation.md": "# ADR-001\n\n## Status\n\nAccepted\n",
			},
			violations: 0,
		},
		{
			name: "bold-label Status passes",
			files: map[string]string{
				"docs/adr/ADR-004-Bold.md": "# ADR-004\n\n**Status**: Accepted\n",
			},
			violations: 0,
		},
		{
			name: "list-item Status passes",
			files: map[string]string{
				"docs/adr/ADR-005-List.md": "# ADR-005\n\n*   **Status:** Accepted\n",
			},
			violations: 0,
		},
		{
			name: "missing Status marker flagged",
			files: map[string]string{
				"docs/adr/ADR-002-No-Status.md": "# ADR-002\n\nNothing marks acceptance here.\n",
			},
			violations: 1,
		},
		{
			name: "bad filename flagged",
			files: map[string]string{
				"docs/adr/ADR-3-bad-name.md": "# ADR-3\n\n## Status\n\nAccepted\n",
			},
			violations: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkADRMetadata(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestCheckEnvOnlyConfig(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name: "env read in internal/config passes",
			files: map[string]string{
				"internal/config/config.go": "package config\n\nimport \"os\"\n\nvar v = os.Getenv(\"PORT\")\n",
				".gitignore":                ".env\n",
				".env.example":              "PORT=4000\n",
			},
			violations: 0,
		},
		{
			name: "bootstrap allowlist passes",
			files: map[string]string{
				"cmd/api/main.go": "package main\n\nimport \"os\"\n\nvar v = os.Getenv(\"ENV\")\n",
				".gitignore":      ".env\n",
				".env.example":    "PORT=4000\n",
			},
			violations: 0,
		},
		{
			name: "env read in handler flagged",
			files: map[string]string{
				"internal/handler/h.go": "package handler\n\nimport \"os\"\n\nvar v = os.Getenv(\"SECRET\")\n",
				".gitignore":            ".env\n",
				".env.example":          "PORT=4000\n",
			},
			violations: 1,
		},
		{
			name: "comment naming os.Getenv is not a finding",
			files: map[string]string{
				"internal/handler/h.go": "package handler\n\n// do not call os.Getenv(\"SECRET\") here; config owns it\nfunc f() {}\n",
				".gitignore":            ".env\n",
				".env.example":          "PORT=4000\n",
			},
			violations: 0,
		},
		{
			name: "missing env.example and unignored .env flagged",
			files: map[string]string{
				".gitignore": "dist/\n",
			},
			violations: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkEnvOnlyConfig(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestCheckNoHTMLTemplate(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name: "text/template-free repo passes",
			files: map[string]string{
				"internal/view/render.go": "package view\n\nimport \"net/http\"\n\nvar _ = http.StatusOK\n",
			},
			violations: 0,
		},
		{
			name: "html/template import flagged",
			files: map[string]string{
				"internal/handler/legacy.go": "package handler\n\nimport \"html/template\"\n\nvar _ = template.New\n",
			},
			violations: 1,
		},
		{
			name: "generated templ files exempt",
			files: map[string]string{
				"internal/view/page_templ.go": "package view\n\nimport \"html/template\"\n",
			},
			violations: 0,
		},
		{
			name: "string literal naming the package is not an import",
			files: map[string]string{
				"scripts/tool/main.go": "package main\n\nvar needle = \"html/template\"\n",
			},
			violations: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkNoHTMLTemplate(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestCheckTypedViewProps(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name: "typed props pass",
			files: map[string]string{
				"internal/view/props.go": "package view\n\ntype HomeProps struct{ Title string }\n",
			},
			violations: 0,
		},
		{
			name: "map[string]interface{} in view flagged",
			files: map[string]string{
				"internal/view/props.go": "package view\n\nvar bad map[string]interface{}\n",
			},
			violations: 1,
		},
		{
			name: "map[string]any in templ flagged",
			files: map[string]string{
				"internal/view/pages/home.templ": "package pages\n\nvar bad map[string]any\n",
			},
			violations: 1,
		},
		{
			name: "go comment naming the untyped map is not a finding",
			files: map[string]string{
				"internal/view/props.go": "package view\n\n// HomeProps replaces the old map[string]interface{} bag.\ntype HomeProps struct{ Title string }\n",
			},
			violations: 0,
		},
		{
			name: "templ comments naming the untyped map are not findings",
			files: map[string]string{
				"internal/view/pages/home.templ": "package pages\n\n// typed props, never map[string]any\ntempl Home() {\n\t<!-- was map[string]interface{} -->\n\t<a href=\"https://example.com\">x</a>\n}\n",
			},
			violations: 0,
		},
		{
			name: "handler usage out of scope",
			files: map[string]string{
				"internal/handler/h.go": "package handler\n\nvar meta map[string]interface{}\n",
			},
			violations: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkTypedViewProps(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestCheckDeployScope(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name:       "single root fly.toml passes",
			files:      map[string]string{"fly.toml": "app = \"demo\"\n"},
			violations: 0,
		},
		{
			name: "second fly.toml flagged",
			files: map[string]string{
				"fly.toml":        "app = \"demo\"\n",
				"deploy/fly.toml": "app = \"other\"\n",
			},
			violations: 1,
		},
		{
			name: "terraform file flagged",
			files: map[string]string{
				"fly.toml":     "app = \"demo\"\n",
				"infra/x.tf":   "resource {}\n",
				".gitignore":   "",
				".env.example": "",
			},
			violations: 1,
		},
		{
			name:       "missing root fly.toml flagged",
			files:      map[string]string{".gitignore": ""},
			violations: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkDeployScope(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestCheckSlogOnly(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name: "slog usage passes",
			files: map[string]string{
				"go.mod":          "module example.com/m\n\ngo 1.25\n",
				"internal/x/x.go": "package x\n\nimport \"log/slog\"\n\nfunc f() { slog.Info(\"ok\") }\n",
			},
			violations: 0,
		},
		{
			name: "zerolog in go.mod flagged",
			files: map[string]string{
				"go.mod": "module example.com/m\n\nrequire github.com/rs/zerolog v1.33.0\n",
			},
			violations: 1,
		},
		{
			name: "log.Printf call site flagged",
			files: map[string]string{
				"go.mod":          "module example.com/m\n",
				"internal/x/x.go": "package x\n\nimport \"log\"\n\nfunc f() { log.Printf(\"no\") }\n",
			},
			violations: 1,
		},
		{
			name: "comment naming log.Printf is not a finding",
			files: map[string]string{
				"go.mod":          "module example.com/m\n",
				"internal/x/x.go": "package x\n\n// log.Printf(\"x\") was retired by ADR-026\nfunc f() {}\n",
			},
			violations: 0,
		},
		{
			name: "slog.Info not a false positive",
			files: map[string]string{
				"go.mod":          "module example.com/m\n",
				"internal/x/x.go": "package x\n\nfunc f() { myslog.Printver() }\n",
			},
			violations: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkSlogOnly(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestCheckCIInvokesGate(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name: "workflow invoking task ci passes",
			files: map[string]string{
				".github/workflows/ci.yml": "steps:\n  - run: task ci\n",
			},
			violations: 0,
		},
		{
			name: "workflow re-implementing steps flagged",
			files: map[string]string{
				".github/workflows/ci.yml": "steps:\n  - run: go test ./...\n",
			},
			violations: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkCIInvokesGate(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestRunSuite(t *testing.T) {
	tests := []struct {
		name         string
		config       []CheckConfig
		wantBlockers int
		wantExitFail bool
	}{
		{
			name: "warn-status failure never fails the run",
			config: []CheckConfig{
				{ID: "adr002-migration-pairs", ADR: "ADR-002", TC: "TC-1", Status: "warn", Added: "2026-07-12"},
			},
			wantBlockers: 0,
			wantExitFail: false,
		},
		{
			name: "block-status failure fails the run",
			config: []CheckConfig{
				{ID: "adr002-migration-pairs", ADR: "ADR-002", TC: "TC-1", Status: "block", Added: "2026-07-12"},
			},
			wantBlockers: 1,
			wantExitFail: true,
		},
		{
			name: "unknown check id is always a blocker",
			config: []CheckConfig{
				{ID: "adr999-not-wired", ADR: "ADR-999", TC: "TC-1", Status: "warn", Added: "2026-07-12"},
			},
			wantBlockers: 1,
			wantExitFail: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			// One orphaned migration so adr002 has a violation to report.
			writeTree(t, root, map[string]string{
				"migrations/000001_orphan.up.sql": "CREATE TABLE o (id int);",
			})
			results, failed := runSuite(root, tt.config)
			blockers := 0
			for _, r := range results {
				if r.Result == "BLOCKER" {
					blockers++
				}
			}
			if blockers != tt.wantBlockers {
				t.Errorf("blockers = %d, want %d", blockers, tt.wantBlockers)
			}
			if failed != tt.wantExitFail {
				t.Errorf("failed = %v, want %v", failed, tt.wantExitFail)
			}
		})
	}
}

func TestCheckNoHardcodedSecrets(t *testing.T) {
	// Synthetic credentials assembled at runtime so this test file never
	// itself contains a string the scanner would flag.
	awsKey := "AKIA" + strings.Repeat("Q", 16)
	ghPat := "ghp_" + strings.Repeat("a1", 18)
	jwt := "eyJ" + strings.Repeat("A", 20) + "." + strings.Repeat("B", 20) + "." + strings.Repeat("C", 20)
	pem := "-----BEGIN RSA " + "PRIVATE KEY-----\nMIIB...\n-----END RSA PRIVATE KEY-----"
	opaque := "sk9Qm2Vt7ZpL4xR8cNb3Wy6Hd1" // 26 chars, no placeholder words

	tests := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: "clean tree passes",
			files: map[string]string{
				"internal/config/config.go": "package config\nvar key = os.Getenv(\"API_KEY\")\n",
				".env.example":              "SUPABASE_ANON_KEY=your-anon-key-here\n",
				".env.tpl":                  "SUPABASE_SERVICE_ROLE_KEY=op://Personal/go-perf-starter/service_role\n",
			},
		},
		{
			name:  "AWS access key id in Go source",
			files: map[string]string{"internal/auth/x.go": "package auth\nconst k = \"" + awsKey + "\"\n"},
			want:  1,
		},
		{
			name:  "GitHub token in a workflow",
			files: map[string]string{".github/workflows/ci.yml": "env:\n  TOKEN: " + ghPat + "\n"},
			want:  1,
		},
		{
			name:  "private key block in fly.toml",
			files: map[string]string{"fly.toml": "[env]\nKEY='''\n" + pem + "\n'''\n"},
			want:  1,
		},
		{
			name:  "JWT literal (Supabase keys are JWTs) in SQL",
			files: map[string]string{"sql/demo/seed.sql": "-- " + jwt + "\n"},
			want:  1,
		},
		{
			name:  "generic secret assignment with an opaque value",
			files: map[string]string{"cmd/api/main.go": "package main\nvar apiKey = \"" + opaque + "\"\n"},
			want:  1,
		},
		{
			name:  "generic pattern ignores placeholders",
			files: map[string]string{"cmd/api/main.go": "package main\nvar apiKey = \"your-api-key-goes-here-please\"\nvar pw = \"changeme-changeme-changeme\"\n"},
			want:  0,
		},
		{
			name:  "generic pattern skips test files (fixtures), vendor patterns do not",
			files: map[string]string{"internal/auth/x_test.go": "package auth\nvar password = \"" + opaque + "\"\nvar k = \"" + awsKey + "\"\n"},
			want:  1,
		},
		{
			name: "generated, vendored, and doc files are out of scope",
			files: map[string]string{
				"internal/view/pages/x_templ.go": "package pages\nvar k = \"" + awsKey + "\"\n",
				"web/static/js/htmx.min.js":      "var k=\"" + awsKey + "\"",
				"docs/adr/ADR-099-X.md":          "Example: " + awsKey + "\n",
				"node_modules/x/index.js":        "var k=\"" + awsKey + "\"",
			},
			want: 0,
		},
		{
			name:  "one violation per line, line number reported",
			files: map[string]string{"scripts/x/main.go": "package main\n\nvar a = \"" + awsKey + "\"\n"},
			want:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkNoHardcodedSecrets(root)
			if len(got) != tt.want {
				t.Errorf("got %d violations, want %d: %v", len(got), tt.want, got)
			}
			if tt.name == "one violation per line, line number reported" && len(got) == 1 && !strings.Contains(got[0], "scripts/x/main.go:3:") {
				t.Errorf("violation should name file:line, got %q", got[0])
			}
		})
	}
}

func TestCheckCSPSafeAlpine(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name: "parser-safe inline expressions pass",
			files: map[string]string{
				"internal/view/pages/p.templ": "package pages\n\ntempl P() {\n\t<div x-data=\"{ open: false, tab: 'a' }\">\n\t\t<button @click=\"open = !open\" :aria-expanded=\"open.toString()\">x</button>\n\t\t<p x-show=\"touched && !email.includes('@')\" x-text=\"$store.demo.count\"></p>\n\t</div>\n}\n",
			},
			violations: 0,
		},
		{
			name: "registered component reference passes",
			files: map[string]string{
				"internal/view/pages/p.templ": "package pages\n\ntempl P() {\n\t<div x-data=\"confirmDialog\" @htmx:confirm.prevent=\"pause($event)\"></div>\n}\n",
			},
			violations: 0,
		},
		{
			name: "arrow function in x-init flagged",
			files: map[string]string{
				"internal/view/partials/f.templ": "package partials\n\ntempl F() {\n\t<div x-data=\"{ show: true }\" x-init=\"setTimeout(() => show = false, 3000)\"></div>\n}\n",
			},
			violations: 1,
		},
		{
			name: "browser global in x-data flagged",
			files: map[string]string{
				"internal/view/layouts/b.templ": "package layouts\n\ntempl B() {\n\t<body x-data=\"{ dark: localStorage.getItem('dark') === 'true' }\"></body>\n}\n",
			},
			violations: 1,
		},
		{
			name: "template literal and window access flagged per attribute",
			files: map[string]string{
				"internal/view/pages/p.templ": "package pages\n\ntempl P() {\n\t<span x-text=\"`hi ${name}`\"></span>\n\t<button x-on:click=\"window.scrollTo(0, 0)\">top</button>\n}\n",
			},
			violations: 2,
		},
		{
			name: "hx-trigger filter and js: value flagged",
			files: map[string]string{
				"internal/view/pages/p.templ": "package pages\n\ntempl P() {\n\t<button hx-get=\"/x\" hx-trigger=\"click[ctrlKey]\">a</button>\n\t<button hx-post=\"/y\" hx-vals=\"js:{n: 1}\">b</button>\n}\n",
			},
			violations: 2,
		},
		{
			name: "plain hx-trigger and JSON hx-vals pass",
			files: map[string]string{
				"internal/view/pages/p.templ": "package pages\n\ntempl P() {\n\t<div hx-get=\"/t\" hx-trigger=\"load, every 5s\" hx-vals='{\"a\": \"b\"}'></div>\n}\n",
			},
			violations: 0,
		},
		{
			name: "templ comments are not findings",
			files: map[string]string{
				"internal/view/pages/p.templ": "package pages\n\n// was x-init=\"setTimeout(() => show = false)\"\ntempl P() {\n\t<!-- not x-data=\"{ d: localStorage }\" -->\n\t<div x-data=\"{ show: true }\"></div>\n}\n",
			},
			violations: 0,
		},
		{
			name: "files outside internal/view are out of scope",
			files: map[string]string{
				"internal/handler/source.templ": "package handler\n\ntempl S() {\n\t<div x-init=\"setTimeout(() => x(), 1)\"></div>\n}\n",
			},
			violations: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkCSPSafeAlpine(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestCheckCSPBuildVendored(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name:       "CSP build passes",
			files:      map[string]string{"web/static/js/alpine.min.js": `(()=>{throw new Error("CSP Parser Error: Unexpected token")})();version:"3.17.4"`},
			violations: 0,
		},
		{
			name:       "standard build flagged",
			files:      map[string]string{"web/static/js/alpine.min.js": `(()=>{new Function("return 1")})();version:"3.17.4"`},
			violations: 1,
		},
		{
			name:       "missing file flagged",
			files:      map[string]string{},
			violations: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkCSPBuildVendored(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}

func TestStripGoComments(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"line comment blanked, newline kept", "a // b\nc\n", "a     \nc\n"},
		{"block comment keeps its newlines", "a /* b\nc */ d\n", "a     \n     d\n"},
		{"comment marker inside a string survives", "x := \"// not a comment\"\n", "x := \"// not a comment\"\n"},
		{"comment marker inside a raw string survives", "x := `/* kept */`\n", "x := `/* kept */`\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(stripGoComments([]byte(tt.in))); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckNoCOPBypass(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		violations int
	}{
		{
			name: "plain CrossOriginProtection passes",
			files: map[string]string{
				"internal/middleware/x.go": "package middleware\n\nimport \"net/http\"\n\nvar c = http.NewCrossOriginProtection()\n",
			},
			violations: 0,
		},
		{
			name: "AddInsecureBypassPattern under internal flagged",
			files: map[string]string{
				"internal/middleware/x.go": "package middleware\n\nfunc f() { cop.AddInsecureBypassPattern(\"/hook\") }\n",
			},
			violations: 1,
		},
		{
			name: "AddTrustedOrigin under cmd flagged",
			files: map[string]string{
				"cmd/server/main.go": "package main\n\nfunc f() { _ = cop.AddTrustedOrigin(\"https://x.example\") }\n",
			},
			violations: 1,
		},
		{
			name: "comment naming the bypass is not a finding",
			files: map[string]string{
				"internal/middleware/x.go": "package middleware\n\n// no AddTrustedOrigin or AddInsecureBypassPattern (ADR-035)\nfunc f() {}\n",
			},
			violations: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, tt.files)
			got := checkNoCOPBypass(root)
			if len(got) != tt.violations {
				t.Errorf("got %d violations %v, want %d", len(got), got, tt.violations)
			}
		})
	}
}
