# Phase 3 — Tooling & Quality Gates

Set up infrastructure for code quality and developer productivity.

## Key Implementation Steps

| Step | Task | Why It Matters |
|------|------|----------------|
| 3.01 | Tune & enforce linting | Build on Phase 0's base linting choice |
| 3.02 | Set up testing framework | Ensures functional correctness |
| 3.03 | Add test coverage reporting | Measures testing thoroughness |
| 3.04 | Create Taskfile for operations | Automates common tasks |
| 3.05 | Configure hot reloading | Speeds up development cycle |
| 3.06 | Set up test fixtures | Enables consistent testing |
| 3.07 | Create performance benchmarks | Identifies bottlenecks |
| 3.08 | Add security scanning | Finds vulnerabilities early |

## Core Principles

- Extend the basic linting setup from Phase 0 with comprehensive rules
- Implement table-driven tests for comprehensive test coverage
- Use Taskfile (or Make) to automate common development operations
- Configure air for hot reloading during development
- Set up a CI pipeline for automated quality checks
- For database connection pooling, the actual numbers live in `internal/config/config.go` (`DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_MAX_CONN_LIFETIME`, `DB_MAX_CONN_IDLE_TIME`)

## golangci-lint Configuration

This is the repo's actual `.golangci.yml` (golangci-lint v2 format — v1-style configs are rejected):

```yaml
# .golangci.yml
version: "2"
run:
  go: "1.25"
  modules-download-mode: readonly
linters:
  default: none
  enable:
    - errcheck
    - govet
    - ineffassign
    - misspell
    - staticcheck
    - unused
    - whitespace
  exclusions:
    generated: lax
    rules:
      - linters:
          - errcheck
          - gosec
        path: _test\.go
      - path: (.+)\.go$
        text: should have a package comment
      - path: (.+)\.go$
        text: exported .* should have comment or be unexported
    paths:
      - third_party$
      - builtin$
      - examples$
issues:
  max-issues-per-linter: 0
  max-same-issues: 0
formatters:
  enable:
    - gofmt
    - goimports
  settings:
    goimports:
      local-prefixes:
        - github.com/clownware/go-performance-starter
  exclusions:
    generated: lax
    paths:
      - third_party$
      - builtin$
      - examples$
```

## Common Pitfalls

- **Over-aggressive linting**: Start with reasonable defaults, adjust as needed
- **Missing test fixtures**: Consistent test data is critical for reliable tests
- **Poor CI configuration**: Ensure CI runs all quality checks
- **Slow hot reloading**: Configure appropriate file watch settings
- **Inadequate security scanning**: Integrate govulncheck or similar tools

## Implementation Strategy

- Begin with linting configuration based on the chosen tool from Phase 0
- Set up automated testing with table-driven tests
- Configure development tools for efficient workflow
- Integrate security scanning into CI pipeline
- Establish continuous testing and quality verification

## Exit Criteria

- Linting configured with appropriate rules and passing
- Testing framework established with table-driven tests
- Test coverage reporting set up with target metrics
- Task automation implemented for common operations
- Hot reloading functioning properly
- Test fixtures created for consistent testing
- Security scanning integrated into CI pipeline


