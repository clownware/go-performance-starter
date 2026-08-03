# Project Structure

This document outlines the recommended project structure for the Alpine Go Performance Starter, following the Go Web Application Implementation Guide.

> **Note:** In the Go Web Application Implementation Guide, file indices are zero-based (e.g., 01_Foundation_Kickoff.md corresponds to Phase 0), while phase numbers are one-based.

## Project Structure

```
go-performance-starter/
├── cmd/
│   └── api/
│       └── main.go               # Application entrypoint
├── internal/                     # Private application code
│   ├── auth/                     # Supabase auth client (anonymous sign-in, upgrade, recovery)
│   ├── cache/                    # In-memory caching helpers
│   ├── config/                   # Environment config loader
│   ├── database/                 # sqlc-generated queries + connection pooling
│   ├── handler/                  # HTTP handlers (auth, quiz, flashcards, patterns, profile, health)
│   ├── jobs/                     # Background jobs (guest-session reaper)
│   ├── middleware/               # Auth, CSRF, rate limiting, security headers, metrics, etc.
│   ├── performance/              # Performance budget constants (ADR-000)
│   ├── repository/               # Repository interfaces + Postgres implementations (ADR-003)
│   ├── server/                   # HTTP server + route configuration
│   ├── validate/                 # Input validation
│   ├── view/                     # templ UI with typed props (ADR-017)
│   │   ├── layouts/              # base.templ — shared page shell
│   │   ├── pages/                # Full pages (home, auth, quiz, flashcards, dashboard, …)
│   │   ├── partials/             # HTMX fragments
│   │   ├── components/           # Reusable components (button, card, form, alert, …)
│   │   └── render.go             # templ render helpers
│   └── webutil/                  # Shared request/response utilities
├── migrations/                   # golang-migrate SQL files (up/down pairs)
├── sql/
│   ├── queries/                  # sqlc query files (users, quiz, flashcards, organizations, …)
│   ├── schema/                   # Combined schema for sqlc
│   ├── demo/                     # Public-demo seed/reset scripts
│   └── test/                     # Test fixtures (auth stub)
├── sqlc.yaml                     # SQLC configuration
├── web/
│   └── static/                   # Static assets
│       ├── css/                  # input.css (Tailwind v4 source) + app.css (compiled)
│       ├── js/                   # htmx.min.js, alpine.min.js, app.js
│       └── img/                  # Image assets
├── .air.toml                     # Hot reload configuration
├── .env.example                  # Example environment variables
├── .golangci.yml                 # Linting configuration
├── docker-compose.yml            # Local development setup
├── Dockerfile                    # Production container
├── fly.toml                      # Worked-example deploy config (ADR-025)
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
├── Taskfile.yml                  # Development tasks
├── versions.json                 # Version manifest (ADR-030)
└── README.md                     # Project documentation
```

This structure follows the standard Go project layout patterns and is organized to support the implementation phases outlined in the Go Web Application Implementation Guide.
