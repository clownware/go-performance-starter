# ADR-023: Testing Philosophy

## Status

Accepted

## Date

2026-06-11

## Context

ADR-010 established the tooling (Go `testing`, table-driven tests, golangci-lint). It did not state *how* tests are written or *when* they must exist, which matters most when an agent is generating code: without a test-first rule, agents produce production code and either no test or a test reverse-engineered to pass. The repo also had real coverage gaps — handlers, repositories, and auth had no tests while perf/health/config did.

## Decision

Adopt these testing rules, enforced through the three-pass workflow (ADR-020) and the quality gate (ADR-021):

1. **Non-trivial production code follows a failing test.** The Architect writes the failing table-driven test before the Coder writes code (Constitution rule 8).
2. **Table-driven, one behaviour per case.** Name cases for the behaviour, not the implementation. Test both success and error paths.
3. **Use `net/http/httptest` and repository interface fakes** for handler tests; reserve a real Postgres harness for repository/integration tests.
4. **Never lower a coverage or budget threshold** to make a test pass — fix the code or open an ADR documenting the exception.
5. **Priority order for closing the existing gap:** handlers and auth first, repositories next.

## Consequences

- Tests double as executable specifications, which is why BDD/Gherkin is declined as a default (revisit only on genuinely ambiguous prose criteria).
- Agent-generated code arrives with a meaningful, pre-written test rather than a rationalised one.
- Closing the handler/auth/repository gap is tracked as explicit work, not left implicit.

## Alternatives Considered

- **Coverage percentage target only.** Rejected — a number invites gaming; behaviour-first rules plus the test-first workflow address the actual risk.
- **BDD/Gherkin specs.** Declined as default — adds ceremony; table-driven tests already read as specs.

## References

- [ADR-010](ADR-010-Testing-and-Code-Quality.md), [ADR-020](ADR-020-Agent-Roles.md), [ADR-021](ADR-021-Halt-On-Violation-Quality-Gate.md)
