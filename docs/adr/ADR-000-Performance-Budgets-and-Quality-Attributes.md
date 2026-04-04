# ADR-000: Performance Budgets and Quality Attributes

## Status

Accepted

## Context

Performance is a first-class requirement for this Go/Alpine SaaS starter kit. Without explicit, measurable performance budgets defined upfront, architectural decisions may inadvertently degrade user experience. Performance budgets serve as guardrails during development, ensuring that dependencies, features, and code changes align with our commitment to lightweight, fast web applications.

This ADR establishes hard performance budgets that will be enforced in CI/CD, guiding decisions about dependencies, caching strategies, and optimization priorities.

## Decision

### 1. Performance Budgets

We establish the following **hard performance budgets** that MUST be enforced in CI:

#### Response Time Budgets
- **P50 Response Time**: < 50ms (median request)
- **P95 Response Time**: < 100ms (95th percentile)
- **P99 Response Time**: < 200ms (99th percentile)
- **Database Query Time**: < 10ms (P95 for single queries)

#### Resource Budgets
- **Binary Size**: < 20MB (compiled Go binary)
- **Docker Image Size**: < 30MB (Alpine-based image)
- **Memory Usage (RSS)**: < 128MB (steady state under normal load)
- **Memory Usage (Peak)**: < 256MB (under high load)
- **Startup Time**: < 500ms (application ready to serve requests)

#### Bundle Size Budgets (Frontend)
- **JavaScript Bundle**: < 50KB (compressed)
- **CSS Bundle**: < 30KB (compressed)
- **Total Page Weight**: < 500KB (including HTML, CSS, JS, fonts)
- **Critical Path Resources**: < 150KB (above-the-fold assets)

#### Core Web Vitals
- **Largest Contentful Paint (LCP)**: < 2.5s
- **First Input Delay (FID)**: < 100ms
- **Cumulative Layout Shift (CLS)**: < 0.1
- **Time to First Byte (TTFB)**: < 200ms

#### Scalability Targets
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
```yaml
# Performance test suite runs on every PR
- name: Performance Budget Tests
  run: |
    task test:performance
    task test:binary-size
    task test:memory-profile
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

### Phase 1: Baseline Measurement (Week 1)
1. Instrument existing application with Prometheus metrics
2. Run load tests to establish current performance baseline
3. Document actual P50/P95/P99 response times

### Phase 2: Budget Definition (Week 2)
4. Define budgets based on baseline + 20% headroom
5. Implement performance test suite in `internal/performance/`
6. Add CI checks to Taskfile and GitHub Actions

### Phase 3: Enforcement (Week 3)
7. Make performance tests mandatory for PR approval
8. Add automated PR comments with performance delta
9. Document budget violation process in CONTRIBUTING.md

### Phase 4: Continuous Improvement (Ongoing)
10. Review budgets quarterly based on production metrics
11. Adjust thresholds via ADR amendments
12. Share learnings in team retrospectives

## References

- [Astro Performance Starter Template ADRs](https://github.com/example/astro-perf) - Inspiration for budget-first approach
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

---

**Date**: 2025-11-15  
**Author**: System Architecture Team  
**Reviewers**: Engineering, Product
