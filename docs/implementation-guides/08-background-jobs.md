# Phase 7 — Background Processing

Implement asynchronous processing for longer-running tasks.

## Key Implementation Steps

| Step | Task | Why It Matters |
|------|------|----------------|
| 7.01 | Evaluate processing needs | Simple vs. complex requirements |
| 7.02 | For simple: Use goroutines | With proper context handling |
| 7.03 | For complex: Set up Asynq | Only when truly needed |
| 7.04 | Implement job producer | Enqueue background tasks |
| 7.05 | Create job consumer | Process background tasks |
| 7.06 | Add retry and error handling | Handle failures gracefully |
| 7.07 | Implement scheduled jobs | Periodic tasks |
| 7.08 | Add graceful shutdown | Clean termination |
| 7.09 | Consider simpler alternatives | Scheduled tasks via Cloudflare |

## Core Principles

- For most applications, goroutines with proper context handling are sufficient
- Only add Asynq+Redis when you need reliable delivery, retries, and monitoring
- Implement proper resource cleanup and graceful shutdown
- Use context cancellation to prevent goroutine leaks
- Consider serverless scheduled tasks for simpler needs

## Common Pitfalls

- **Over-engineering**: Don't add Redis+Asynq unless truly needed
- **Goroutine leaks**: Implement proper cancellation
- **Missing error handling**: Have clear strategies for failures
- **Poor shutdown**: Ensure clean termination of background tasks
- **Lack of visibility**: Add logging for background tasks

## Decision Guide: Simple vs. Complex

### Use Simple Approach (Goroutines) When:
- Tasks complete quickly (seconds, not minutes)
- Occasional failures are acceptable
- No complex scheduling is needed
- Job volume is relatively low

### Use Complex Approach (Asynq+Redis) When:
- Tasks must be durable through restarts
- Reliable delivery is critical
- Advanced retries and scheduling needed
- Monitoring and observability required
- High throughput of jobs expected

## Exit Criteria

- Appropriate background processing chosen
- Job producer implemented if needed
- Job consumer handles background tasks
- Retry and error handling implemented
- Scheduled jobs working for periodic tasks
- Graceful shutdown ensures clean termination
- Solution matches application complexity
