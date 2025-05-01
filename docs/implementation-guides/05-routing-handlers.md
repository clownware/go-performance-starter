# Phase 4 — Routing & Core Handlers

Establish the structural foundation of your application.

## Key Implementation Steps

| Step | Task | Why It Matters |
|------|------|----------------|
| 4.01 | Create router with proper grouping | Organizes API structure |
| 4.02 | Implement middleware stack | Adds cross-cutting concerns |
| 4.03 | Create error handler | Consistent error responses |
| 4.04 | Implement core CRUD handlers | Basic data operations |
| 4.05 | Add request validation | Ensures data quality |
| 4.06 | Create view models | Separates DB from presentation |
| 4.07 | Set up health check endpoint | Enables monitoring |
| 4.08 | Add structured logging | Provides operational visibility |
| 4.09 | Configure timeout handling | Prevents resource exhaustion |

## Core Principles

- Group routes by feature for better organization
- Implement middleware in the correct order (recovery, logging, etc.)
- Create a consistent error handler that responds appropriately to content types
- Use view models to separate database models from presentation
- Implement proper request validation with meaningful error messages
- Add structured logging with request IDs and context information

## Common Pitfalls

- **Inconsistent error handling**: Create a central error handler
- **Missing timeouts**: Configure proper timeout handling
- **Monolithic handlers**: Separate concerns with middleware
- **Poor validation**: Validate all input with clear error messages
- **Inadequate logging**: Ensure logs include request context

## Recommended Middleware Order

1. Recovery (panic handling) - comes first so that panics still need request IDs in logs
2. Request ID generation - ensures every request gets a unique identifier
3. Logging - captures request metadata with the request ID
4. Request timeout - prevents long-running requests
5. CORS (if needed) - handles cross-origin requests
6. Authentication (if applicable) - validates user identity 
7. Request body limiting - prevents request body attacks
8. Content type validation - ensures proper request format

## Exit Criteria

- Router configured with proper route grouping
- Middleware stack implemented in correct order
- Error handler providing consistent responses
- Core CRUD handlers implemented
- Request validation functioning properly
- View models separating concerns
- Health check endpoint implemented
- Structured logging configured
- Timeout handling preventing resource exhaustion


