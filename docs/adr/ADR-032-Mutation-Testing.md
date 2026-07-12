# ADR-032: Mutation Testing

## Status

Accepted

## Date

2026-07-12

## Context

ADR-023 established behaviour-first, table-driven tests and deliberately rejected a coverage percentage target as gameable. That leaves a second-order question unanswered: do the tests *assert* enough, or merely execute lines? A 2026-07-12 test audit ran a mutation baseline and found two surviving mutants in `internal/config` — boundary conditions (`port > 65535`, `DBMaxConns < 1`) that the validation tests execute but never pin — proving the question is live even in an 83%-covered package.

Mutation testing answers exactly that question, but only where tests are already meaningful: on low-coverage packages the score is coverage restated at many times the runtime, and on generated code (sqlc, templ) or env-gated integration suites the mutants are noise.

## Decision

Adopt [go-gremlins](https://github.com/go-gremlins/gremlins) (v0.6.0, pinned), run via `task test:mutation`:

1. **Scope: well-covered pure-logic packages only** — currently `internal/middleware`, `internal/auth`, `internal/config`, `internal/validate`, `internal/jobs`. Generated code, env-gated repository/integration suites, and render-only view code are out of scope.
2. **Sequencing rule: a package earns mutation testing only after first-order coverage exists.** Closing a coverage gap comes first; mutation testing is the second-order check on suites that already pass the first-order one. When a gap package (e.g. `internal/validate`) gains direct tests, add it to the task's package list.
3. **Placement: manual / scheduled, never in the PR gate.** `task ci` stays first-order (ADR-021). Mutation runs cost minutes per package and give second-order signal; they do not belong on every iteration.
4. **Lived mutants are findings, not thresholds.** Consistent with ADR-023's rejection of coverage percentages, no `--threshold-efficacy` is enforced; a surviving mutant is triaged into either a missing test case or an accepted note in the run output.

## Consequences

- Assertion gaps in security- and boot-critical logic (session validation, rate limits, config validation) surface mechanically instead of in incident review.
- A new pinned dev tool (`gremlins`) joins the toolchain; it is not required for `task ci`, so CI and new contributors are unaffected until they opt in.
- Timed-out mutants (negated loop conditions become infinite loops) are counted as caught; the task passes `--timeout-coefficient 5` so slow-but-legitimate kills are not misreported.

## Alternatives Considered

- **avito-tech/go-mutesting.** Rejected — inactive since January 2026; gremlins releases and merges through June 2026.
- **gtramontina/ooze.** Rejected — library-style API, no tagged releases; harder to pin per ADR-030 discipline.
- **Mutation score threshold in `task ci`.** Rejected — same gameability ADR-023 rejected for coverage percentages, at much higher runtime cost on every PR.

## References

- [ADR-010](ADR-010-Testing-and-Code-Quality.md), [ADR-021](ADR-021-Halt-On-Violation-Quality-Gate.md), [ADR-023](ADR-023-Testing-Philosophy.md), [ADR-030](ADR-030-Versions-Manifest-Contract.md)
