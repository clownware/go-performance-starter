# ADR-001: Foundation Architectural Decisions

**Date**: 2025-04-30

## Status

Accepted (§3 superseded by ADR-026; §5 superseded by ADR-025)

## Context

The Alpine Go Performance Starter requires several foundational architectural decisions that are difficult to change later. These decisions establish the development environment, framework choices, and operational approach that will shape the entire project. Making these decisions early creates a consistent foundation that improves maintainability and developer experience.

This ADR documents these critical choices to ensure alignment and provide rationale for future reference, following the principle that architectural decisions should be documented with clear reasoning.

## Decision

### 1. Go Version & Module Path

We will use **Go 1.24+** with the module path `github.com/clownware/alpine-go-performance-starter`.

Rationale:
- Go 1.22+ includes improved handler signatures that simplify HTTP request handling
- Support for generics (introduced in 1.18) enables more type-safe code patterns
- Handler improvements in 1.22 reduce boilerplate in web applications
- The module path follows standard Go naming conventions and should be updated to reflect the actual repository location

### 2. Web Framework

We will use **Chi** as our primary web framework.

Rationale:
- Better alignment with Go's standard library compared to alternatives
- Lightweight but feature-complete for building HTTP services
- Strong middleware support and good community adoption
- Compatible with standard `net/http` interfaces, maximizing interoperability
- Simpler mental model compared to Echo or Fiber
- HTTP/2 support out of the box
- Maintained and actively developed

### 3. Structured Logging

> **Amended 2026-07-05**: Superseded by [ADR-026](ADR-026-Logging-Standardization.md) — structured logging is standardized on stdlib `log/slog`.

We will use **zerolog** as our structured logging library.

Rationale:
- Performance-focused with minimal allocations for high-throughput environments
- JSON-structured logging format that integrates well with modern log aggregation systems
- API design that encourages adding context to log messages
- Low overhead compared to alternatives
- Ability to log at various levels (debug, info, warn, error)
- Simple integration with HTTP middleware for request logging

### 4. Secret Management Strategy

We adopt a two-tier approach to secret management:

**Development Environment:**
- Use `.env` files loaded via `godotenv` for local development only
- Ensure `.env` is included in `.gitignore` to prevent accidental commits of secrets
- Include `.env.example` in the repository with placeholder values as documentation

**Production Environment:**
- `.env` files are explicitly NOT for production use
- Use Cloudflare Environment Variables (or equivalent platform-specific solution) for production secrets
- Implement runtime secret rotation capability where applicable
- Secrets are injected as environment variables at runtime
- No hardcoded secrets or configuration in application code

### 5. Deployment Considerations

> **Amended 2026-07-05**: Superseded by [ADR-025](ADR-025-Deployment-Target.md) — the deployment target is a single stateless container behind the Cloudflare proxy; Cloudflare Workers is explicitly ruled out as an application runtime.

We acknowledge the following deployment constraints:

- Cloudflare Workers Classic has a 1MB WASM binary size limit
- Dependencies should be carefully selected with size constraints in mind
- For larger applications, consider Cloudflare Durable Objects or alternative deployment approaches
- Static assets will be served via Cloudflare Pages for optimized global distribution

### 6. License

We select the **MIT License** for this project to enable broad adoption and usage.

Rationale:
- Permissive license encouraging both commercial and non-commercial use
- Compatible with open-source principles and widely recognized
- Aligns with the goal of creating a freely accessible starter kit
- Minimal restrictions on derivative works

## Consequences

### Positive

- Clear, documented decisions reduce onboarding time for new developers
- Technology choices emphasize simplicity, performance, and maintainability
- Security-first approach with clear separation of development and production secrets
- License choice enables broad adoption and contribution

### Negative

- The 1MB WASM size limit for Cloudflare Workers may restrict certain dependencies
- Some advanced use cases may require a more complex framework than Chi
- Our chosen patterns may not suit all types of SaaS applications

## Alternatives Considered

### Web Framework
- **Gin**: Popular but less aligned with standard library
- **Echo**: More features but larger footprint
- **Fiber**: Optimized for performance but less idiomatic Go
- **Gorilla Mux**: Simple but less actively maintained

### Logging
- **zap**: Feature-rich but slightly more complex API
- **logrus**: Popular but slower performance characteristics
- **log/slog**: Standard library option but newer and less mature ecosystem

### License
- **GPL**: Too restrictive for a starter kit
- **Apache 2.0**: More complex than needed for this project

## References

- [Go 1.22 Release Notes](https://go.dev/doc/go1.22)
- [Chi Router GitHub](https://github.com/go-chi/chi)
- [Zerolog GitHub](https://github.com/rs/zerolog)
- [Cloudflare Workers Documentation](https://developers.cloudflare.com/workers/)
- [MIT License](https://opensource.org/licenses/MIT)
- [Twelve-Factor App Configuration](https://12factor.net/config)

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: The repo builds with the Go toolchain pinned in `go.mod`, with Chi as the router.
- **Checks:**
  - TC-1 → `go build` via `task ci`; CI reads the toolchain from `go.mod` (status: **block**, pre-existing)
- **Not machine-checkable:** §3 (logging) is superseded by ADR-026 and §5 (deployment) by ADR-025 — their enforcement lives there. Secrets-handling intent is enforced under ADR-015.
- **Graduation log:** _(empty)_
