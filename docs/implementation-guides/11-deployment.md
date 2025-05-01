# Phase 10 — Deployment & Monitoring with Cloudflare and Supabase

Set up production environment with appropriate monitoring.

## Key Implementation Steps

| Step | Task | Why It Matters |
|------|------|----------------|
| 10.01 | Configure Cloudflare Pages | Static content deployment |
| 10.02 | Set up Cloudflare Workers | Serverless API endpoints |
| 10.03 | Configure Supabase production | Database and Auth settings |
| 10.04 | Implement essential telemetry | Machine-readable logs |
| 10.05 | Create health check endpoints | Status verification (/live, /ready) |
| 10.06 | Set up basic monitoring | Performance tracking |
| 10.07 | Implement graceful shutdown | Clean termination |
| 10.08 | Configure database backups | Data protection |
| 10.09 | Set up CI/CD pipeline | Deployment automation |

## Core Principles

- Deploy static content via Cloudflare Pages
- Implement serverless API endpoints with Cloudflare Workers
- Configure Supabase for production with appropriate settings
- Use structured logging with appropriate context
- Create comprehensive health check endpoints
- Implement graceful shutdown procedures
- Configure automated database backups
- Set up CI/CD pipeline for automated deployment

## Cloudflare Deployment

### Pages Configuration
- Connect to GitHub repository
- Set up build commands and output directory
- Configure environment variables for Supabase connection

### Workers Configuration
- Create Workers for dynamic API routes
- Use Wrangler for local development and deployment
- Configure appropriate routes and bindings
- Be aware of the 1MB code size limit for classic Workers
- Consider Durable Objects for stateful applications or larger binaries

## Supabase Production Configuration

- Configure production environment variables
- Verify RLS policies against production roles
- Enable database backups
- Configure correct CORS settings
- Set up appropriate JWT expiration times

## Health Check Endpoints

Follow Kubernetes-compatible naming conventions:
- `/live`: Simple check that the service is running
- `/ready`: Deep check that the service is ready to accept traffic (DB connections, etc.)

## Feature Flag Integration

If you plan to use feature flags in Phase 12:
- Define flag naming conventions in CI/CD configuration
- Consider implementing a simple feature flag mechanism in your deployment
- Prepare for blue/green deployment strategy if needed
- Enable Dependabot auto-PRs; block merge until CI passes

## Essential Telemetry

- Implement structured logging in JSON format
- Include request ID, timestamp, log level, source
- Create health check endpoints for monitoring
- Set up basic performance monitoring
- Configure alerts for critical issues
- Expose an OTLP endpoint now if you intend to adopt OpenTelemetry in Phase 12

## Graceful Shutdown for HTTP Servers

For non-Workers deployments, implement proper shutdown:

```go
// Create server with appropriate timeout values
server := &http.Server{
    Addr:         ":8080",
    Handler:      router,
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout:  120 * time.Second,
}

// Start server in goroutine
go func() {
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("Server error: %v", err)
    }
}()

// Wait for interrupt signal
stop := make(chan os.Signal, 1)
signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
<-stop

// Create shutdown context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Attempt graceful shutdown
if err := server.Shutdown(ctx); err != nil {
    log.Fatalf("Server forced to shutdown: %v", err)
}
```

Note: Workers environments use different shutdown mechanisms via the FetchEvent lifecycle.

## Common Pitfalls

- **Missing environment variables**: Document all required variables
- **Insufficient logging**: Include context in logs
- **Poor health checks**: Create separate /live and /ready endpoints
- **No graceful shutdown**: Handle termination signals
- **Missing backups**: Configure Supabase backups
- **Worker size limits**: Be aware of the 1MB limit for Workers classic

## Implementation Strategy

- Start with static asset deployment via Cloudflare Pages
- Configure Workers for dynamic API endpoints
- Set up Supabase production environment
- Implement structured logging and monitoring
- Configure health checks and alerting
- Establish CI/CD pipeline for reliable deployments
- Verify backup and restore procedures

## Exit Criteria

- Cloudflare deployment configured
- Supabase production environment set up
- Structured logging implemented
- Health check endpoints operational
- Basic monitoring configured
- Graceful shutdown implemented
- Database backups configured
- Deployment automation working


