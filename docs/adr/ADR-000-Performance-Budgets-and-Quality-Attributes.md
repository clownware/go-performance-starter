# ADR-000: Performance Budgets and Quality Attributes

## Status

Accepted

## Context

Performance is a first-class requirement for this Go/Alpine SaaS starter kit. Without explicit, measurable performance budgets defined upfront, architectural decisions may inadvertently degrade user experience. Performance budgets serve as guardrails during development, ensuring that dependencies, features, and code changes align with our commitment to lightweight, fast web applications.

This ADR establishes hard performance budgets that will be enforced in CI/CD, guiding decisions about dependencies, caching strategies, and optimization priorities.

## Decision

### 1. Performance Budgets

We establish the following performance budgets. Each group is classified (2026-07 amendment) by how it is actually verified today:

- **Enforced** — CI fails on violation.
- **Monitored** — measured by the Prometheus metrics middleware; no production baseline yet, so violations inform rather than gate.
- **Aspirational** — no measurement exists; retained as design targets. Promoting one to Enforced requires building the measurement first.

#### Response Time Budgets — Monitored
- **P50 Response Time**: < 50ms (median request)
- **P95 Response Time**: < 100ms (95th percentile)
- **P99 Response Time**: < 200ms (99th percentile)
- **Database Query Time**: < 10ms (P95 for single queries)

#### Resource Budgets
- **Binary Size**: < 20MB (compiled Go binary) — Enforced (`task test:binary-size`)
- **Docker Image Size**: < 30MB (Alpine-based image) — Enforced (ci.yml docker job, release.yml)
- **Memory Usage (RSS)**: < 128MB (steady state under normal load) — Monitored
- **Memory Usage (Peak)**: < 256MB (under high load) — Monitored
- **Startup Time**: < 500ms (application ready to serve requests) — Monitored

#### Bundle Size Budgets (Frontend)
- **JavaScript Bundle**: < 50KB gzipped — Enforced (`task test:asset-budgets`)
- **CSS Bundle**: < 30KB gzipped — Enforced (`task test:asset-budgets`)
- **Total Page Weight**: < 500KB (including HTML, CSS, JS, fonts) — Aspirational
- **Critical Path Resources**: < 150KB (above-the-fold assets) — Aspirational

#### Core Web Vitals — Aspirational
- **Largest Contentful Paint (LCP)**: < 2.5s
- **First Input Delay (FID)**: < 100ms
- **Cumulative Layout Shift (CLS)**: < 0.1
- **Time to First Byte (TTFB)**: < 200ms

#### Scalability Targets — Aspirational
- **Concurrent Connections**: 10,000+ simultaneous connections
- **Requests per Second**: 5,000+ RPS on single instance
- **Cold Start Time**: < 100ms (relevant for serverless deployments)

### 2. Quality Attribute Requirements

#### Lighthouse Scores (Minimum Thresholds)
- **Performance**: 95+
- **Accessibility**: 98+
- **Best Practices**: 95+
- **SEO**: 95+

#### Code Quality Metrics
- **Test Coverage**: 80%+ (excluding UI templates)
- **Cyclomatic Complexity**: < 10 per function
- **Maximum Function Length**: 50 lines
- **Maximum File Length**: 500 lines

### 3. Enforcement Strategy

#### CI/CD Integration
The Enforced budgets run inside the single quality gate (ADR-021):

```bash
task ci   # includes test:binary-size and test:asset-budgets
```

#### Automated Monitoring
- Performance tests run in CI on every pull request
- Binary size checked and reported as comment on PR
- Memory profiling executed during integration tests
- Lighthouse CI runs on every deployment to staging

#### Budget Violation Policy
- **Hard Fail**: Binary size, memory usage, critical response times
- **Warning**: Nice-to-have metrics (can merge with justification)
- **Review Required**: Any budget increase requires explicit ADR update

### 4. Measurement Tooling

#### Go Performance Tools
- `pprof` for CPU and memory profiling
- `benchstat` for benchmark comparison
- Custom middleware for response time tracking
- Prometheus metrics for production monitoring

#### Frontend Tools
- Lighthouse CI for Web Vitals
- Bundle analyzer for JavaScript/CSS size tracking
- Chrome DevTools for network waterfall analysis

#### Load Testing
- `vegeta` for HTTP load testing
- `k6` for scenario-based performance testing
- Continuous load testing in staging environment

## Consequences

### Positive

- **Quantifiable Goals**: Clear, measurable targets prevent subjective "feels fast" discussions
- **Architectural Guidance**: Budgets inform dependency selection (e.g., "Can we afford this library?")
- **Regression Prevention**: Automated enforcement catches performance degradation early
- **User Experience**: Direct impact on user satisfaction and retention
- **Cost Efficiency**: Lower resource usage translates to reduced infrastructure costs
- **Competitive Advantage**: Performance is a differentiator in SaaS markets

### Negative

- **Development Overhead**: Requires maintaining performance test infrastructure
- **PR Friction**: Budgets may slow down feature development if not balanced
- **False Positives**: Synthetic benchmarks may not reflect real-world usage patterns
- **Maintenance Burden**: Budgets need periodic review as application evolves

### Risks

- **Over-optimization**: Risk of premature optimization if budgets are too aggressive
- **Context Loss**: Budgets defined without production data may be unrealistic
- **Tool Dependency**: Reliance on specific tooling creates brittleness

## Alternatives Considered

### 1. No Formal Budgets
- **Rejected**: Without explicit budgets, performance degrades incrementally
- **Learning**: Astro starter template demonstrated value of upfront budget definition

### 2. Aspirational (Non-Enforced) Budgets
- **Rejected**: Non-binding budgets are ignored under deadline pressure
- **Approach**: Enforce in CI with ability to override via explicit ADR

### 3. Different Threshold Values
- **Considered**: More aggressive budgets (e.g., P95 < 50ms)
- **Decision**: Balance ambition with pragmatism based on Go/Alpine capabilities

## Implementation Notes

The original version of this ADR sketched a four-phase rollout (baseline load
testing, budget calibration, mandatory gates, quarterly review) that was never
executed; it predated the demo-first scope settled in ADR-024. The budget
classification in §1 replaces it: Enforced budgets are gated by `task ci`
today, Monitored budgets graduate to Enforced when production baselines exist
to calibrate them, and Aspirational targets graduate only if their measurement
(load testing, Lighthouse CI) is ever built.

## References

- [Astro Performance Starter](https://github.com/clownware/astro-performance-starter) - Sibling template; inspiration for the budget-first approach
- [Web Performance Budget Calculator](https://perf-budget-calculator.firebaseapp.com/)
- [Google Web Vitals](https://web.dev/vitals/)
- [Go Performance Tips](https://github.com/dgryski/go-perfbook)
- [Lighthouse CI](https://github.com/GoogleChrome/lighthouse-ci)

## Review Cadence

**Review Date**: 2026-07-01

This ADR should be reviewed quarterly to ensure budgets remain aligned with:
- Application complexity growth
- User expectations
- Infrastructure capabilities
- Industry benchmarks

**2026-04 Note**: Budgets have not yet been validated against real production measurements. Binary size and Docker image budgets are enforced in CI. Response time and memory budgets are tracked via Prometheus metrics middleware but lack baseline data from production load.

**2026-07 Note**: Every budget is now explicitly classified in §1 as Enforced, Monitored, or Aspirational — the ADR previously claimed all budgets "MUST be enforced in CI" while only binary and image size were. The frontend JS/CSS budgets joined the Enforced tier: `task test:asset-budgets` (wired into `task ci`) fails if the assets shipped by the base layout exceed the budgets gzipped, using the constants in `internal/performance` as the single source of truth. The unexecuted phase plan in Implementation Notes was replaced with the graduation rule, and the placeholder reference URL was corrected to the real sibling repo.

---

**Date**: 2025-11-15  
**Author**: System Architecture Team  
**Reviewers**: Engineering, Product

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: The stripped linux production binary is under 20MB.
  - TC-2: The Docker image is under 30MB.
  - TC-3: Shipped JavaScript is under 50KB gzipped.
  - TC-4: Shipped CSS is under 30KB gzipped.
- **Checks:**
  - TC-1 → `task test:binary-size` in `task ci` (status: **block**, pre-existing)
  - TC-2 → docker job in `.github/workflows/ci.yml` (status: **block**, pre-existing)
  - TC-3, TC-4 → `task test:asset-budgets` in `task ci` (status: **block**, pre-existing)
- **Not machine-checkable:** Latency percentiles (P50/P95/P99), memory, and startup budgets are monitored via Prometheus but not gated — no load-test harness in CI. Page-weight, Core Web Vitals, and Lighthouse score targets are aspirational; no measurement is wired.
- **Graduation log:** _(empty)_
