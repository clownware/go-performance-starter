# Simplified CRUD Application Guide

Essential guidance for smaller projects with streamlined implementation.

## Simplified Approach

For a simple CRUD dashboard or personal project, follow this streamlined approach:

## Essential Phases

- **Focus on Phases 0-6**: Core foundations, data, UI, routing, and authentication
- **Simplify Background Jobs**: Use goroutines or scheduled tasks instead of Redis+Asynq
- **Minimize Testing**: Implement basic testing only, add more when needed
- **Defer Performance Optimization**: Start with basic indexes, add caching when necessary
- **Simplify Deployment**: Single region deployment, basic logging

## Simplified Step Mapping

For cross-reference and search purposes, simplified steps use the S.xx prefix:

| Simplified Step | Description | Maps to Phase |
|-----------------|-------------|--------------|
| S.01 | Project setup and structure | Phase 0 |
| S.02 | Data model and schema | Phase 1 |
| S.03 | UI components and design | Phase 2 |
| S.04 | Basic quality gates | Phase 3 |
| S.05 | Core API handlers | Phase 4 |
| S.06 | HTMX interactions | Phase 5 |
| S.07 | Authentication with Supabase | Phase 6 |
| S.08 | Simple deployment | Phase 10 |

## Minimal Tech Stack

| Component | Technology | Notes |
|-----------|------------|-------|
| **Language** | Go 1.22+ | Improved handler signatures |
| **Web Framework** | Chi | Standard library alignment |
| **Frontend** | HTMX + Alpine.js | Minimal JavaScript approach |
| **HTML Generation** | html/template | Standard library |
| **CSS Framework** | Tailwind CSS | Utility-first approach |
| **Database & Auth** | Supabase | Integrated PostgreSQL & Auth |
| **Testing** | Go standard testing | Table-driven tests |
| **Deployment** | Cloudflare Pages/Workers | Simple deployment |

## Implementation Priorities

1. **Data Model First**: Define schema and migrations
2. **Core CRUD Operations**: Implement basic data handling
3. **Simple UI Components**: Create reusable UI patterns
4. **Authentication**: Configure Supabase Auth
5. **Deployment**: Set up simple deployment pipeline

## Add Only When Needed

- Redis + Asynq (only when reliable job processing required)
- Advanced observability (when performance becomes critical)
- Complex caching strategies (when performance is inadequate)
- Multi-region deployment (for global user base)
- Advanced database features (for scaling beyond basics)

## Simplified Development Workflow

1. Start with basic project structure and dependencies
2. Define data model and implement migrations
3. Create repository layer with basic CRUD operations
4. Implement HTTP handlers for core operations
5. Create UI templates with HTMX for interactivity
6. Configure Supabase Auth for authentication
7. Set up basic deployment pipeline
8. Implement simple logging and monitoring
9. Add tests for critical paths

## Supabase-Specific Guidance

- Use Supabase Auth UI components for authentication flows
- Implement JWT validation for protected routes
- Configure Row Level Security (RLS) for data protection
- Use Supabase Storage for file uploads when needed
- Leverage PostgREST features for optimized queries

## Example Starter Application

A minimal starter application example would include the project structure and basic implementation of all essential components. Such a starter template is planned for future development (TBD). This would provide:

- Basic project structure following this guide
- Minimal working CRUD example
- Supabase Auth integration
- Deployment configuration for Cloudflare

## Remember

- Start simple and add complexity only when needed
- Focus on delivering core functionality first
- Implement the simplest solution that works
- Document decisions and architecture as you go
- Add advanced features only when you feel the pain



