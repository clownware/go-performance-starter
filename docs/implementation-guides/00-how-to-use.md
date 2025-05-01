# Using This Guide with AI Assistants

This guide has been structured into focused phase documents to maximize effectiveness when working with AI coding assistants in your IDE.

## File Naming Convention vs. Phase Numbers

The filenames use a zero-based numbering system (`00-overview.md`, `01-foundation.md`), while the content refers to phases with their traditional numbers (Phase 0, Phase 1). This is intentional:

- File `01-foundation.md` contains "Phase 0 — Foundation Kick-off"
- File `02-data-architecture.md` contains "Phase 1 — Data Architecture"

And so on. This pattern is common in development documentation.

## Benefits of Phase-Based Documents

- **Focused Context**: Each phase document contains only the relevant information for that stage of development
- **Targeted Assistance**: AI can provide more accurate help when given focused context
- **Improved Performance**: Smaller documents process more efficiently
- **Better Information Retrieval**: Makes it easier to find specific details

## How to Use with AI Assistants

### 1. Determine Your Current Development Phase

Before asking questions, identify which phase of development you're working on:

- Setting up project structure? → Phase 0 (Foundation) → `01-foundation.md`
- Designing data models? → Phase 1 (Data Architecture) → `02-data-architecture.md`
- Building the frontend? → Phase 2 (Interface Design) → `03-interface-design.md`
- Implementing API routes? → Phase 4 (Routing & Handlers) → `05-routing-handlers.md`

### 2. Provide Relevant Document as Context

When prompting your AI assistant, explicitly reference the relevant phase document:

```
Using the 02-data-architecture.md as context, help me design a repository interface for user management.
```

Or:

```
I'm working on implementing HTMX patterns. Based on 06-htmx-alpine.md, what's the recommended approach for handling form validation?
```

### 3. Ask Specific Questions

Frame your questions to target specific aspects of the methodology:

- "What's the recommended middleware order according to 05-routing-handlers.md?"
- "According to 07-auth-security.md, what are the key security considerations for JWT tokens?"
- "Based on 03-interface-design.md, how should I structure my UI components?"

### 4. Combine Multiple Phases When Needed

When working on tasks that span multiple phases, reference the relevant documents:

```
Using 02-data-architecture.md and 05-routing-handlers.md as context, help me implement a handler that creates a new user record.
```

### 5. Start with the Simplified Guide for Small Projects

If building a smaller application, begin with the simplified approach:

```
Based on 14-simplified-app.md, what's the minimal implementation I need for a simple blog application?
```

## Document Quick Reference

| File Name | Content | Traditional Phase |
|-----------|---------|-------------------|
| `00-overview.md` | Technology stack summary and rationale | Overview |
| `01-foundation.md` | Initial project setup and structure | Phase 0 |
| `02-data-architecture.md` | Database schema and repositories | Phase 1 |
| `03-interface-design.md` | Frontend organization and components | Phase 2 |
| `04-tooling-quality.md` | Development workflow and quality gates | Phase 3 |
| `05-routing-handlers.md` | API structure and HTTP handlers | Phase 4 |
| `06-htmx-alpine.md` | Frontend interaction patterns | Phase 5 |
| `07-auth-security.md` | Authentication and security | Phase 6 |
| `08-background-jobs.md` | Asynchronous processing | Phase 7 |
| `09-testing-qa.md` | Testing strategy and implementation | Phase 8 |
| `10-performance.md` | Optimization principles | Phase 9 |
| `11-deployment.md` | Deployment and monitoring | Phase 10 |
| `12-documentation.md` | Documentation strategy | Phase 11 |
| `13-advanced.md` | Optional advanced features | Phase 12 |
| `14-simplified-app.md` | Streamlined approach for smaller projects | Alternative Path |

## Example Workflow

When building a Go HTMX application, a typical workflow might be:

1. Start with `00-overview.md` to understand the overall approach
2. Reference `01-foundation.md` to set up the initial project structure
3. Use `02-data-architecture.md` to design your database schema
4. Follow `03-interface-design.md` to establish UI patterns
5. Implement API routes using `05-routing-handlers.md`
6. Add HTMX interactions with `06-htmx-alpine.md`
7. Secure your application using `07-auth-security.md`
8. Deploy your application following `11-deployment.md`

For smaller projects, you might skip directly to `14-simplified-app.md` and reference specific phase documents only as needed.

