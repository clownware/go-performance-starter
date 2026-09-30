# Security policy

## Reporting a vulnerability

Use GitHub's private vulnerability reporting for this repository: **Security → Report a vulnerability** on [github.com/clownware/go-performance-starter](https://github.com/clownware/go-performance-starter/security). Do not open a public issue or discuss the problem in a pull request until a fix is released. You will get an acknowledgement within a week, and credit in the CHANGELOG if you want it.

## Scope

- The template code in this repository: the Go application, migrations, workflows, Dockerfile, and the `scripts/` tooling.
- The public demo at go-performance-starter.fly.dev runs this code with anonymous guest identities and nightly resets. Its intended abuse surface and the mitigations in place are inventoried in [ADR-031](docs/adr/ADR-031-Public-Demo-Operations.md). Rate limits (429 with `Retry-After`), a request body cap, and CSRF checks are deliberate; hitting them is not a finding.
- Row Level Security is the authorization model ([ADR-004](docs/adr/ADR-004-Authorization-Strategy-RLS.md)). A way to read or write another identity's rows through the application is the highest-priority class of report; the flashcards page carries a live isolation check you can run yourself ([ADR-034](docs/adr/ADR-034-Live-Proof-Surfaces.md)).

Out of scope: vulnerabilities in Supabase, Fly.io, Cloudflare, or a dependency's upstream (report those to the vendor; a dependency advisory that `govulncheck` misses is still worth telling us about).

## How the code is protected

The security patterns and threat model are [ADR-014](docs/adr/ADR-014-Security-Patterns-and-Threat-Model.md); the developer guide is [auth-and-security.md](docs/guides/auth-and-security.md). `task ci` runs `govulncheck`, `go mod verify`, a secret scanner (`adr015-no-hardcoded-secrets`), and the security-header tests on every push; `vuln-scan.yml` re-runs govulncheck on a schedule. Dependabot opens update PRs.

Secrets are never in the repository: 1Password injection locally, the container host's secret store in production ([ADR-015](docs/adr/ADR-015-Configuration-Management-Strategy.md)). If you find one committed, report it privately as above.
