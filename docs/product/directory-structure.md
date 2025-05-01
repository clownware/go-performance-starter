# Project Structure

This document outlines the recommended project structure for the Micro SaaS Starter Kit, following the Go Web Application Implementation Guide.

> **Note:** In the Go Web Application Implementation Guide, file indices are zero-based (e.g., 01_Foundation_Kickoff.md corresponds to Phase 0), while phase numbers are one-based.

## Project Structure

```
microsaas-starter-kit/
├── cmd/
│   └── api/
│       └── main.go               # Application entrypoint
├── internal/                     # Private application code 
│   ├── auth/                     # Authentication related code
│   │   ├── middleware.go         # JWT verification middleware
│   │   ├── handler.go            # Auth handlers (login, register, etc.)
│   │   └── models.go             # Auth-related types
│   ├── billing/                  # Billing related code
│   │   ├── interface.go          # BillingProvider interface
│   │   └── stripe.go             # Stripe implementation
│   ├── config/                   # Configuration handling
│   │   └── config.go             # Environment config loader
│   ├── database/                 # Database connections/utilities
│   │   └── db.go                 # DB setup and connection pooling
│   ├── email/                    # Email sending functionality 
│   │   ├── interface.go          # EmailProvider interface
│   │   └── console.go            # Development console email impl
│   ├── handler/                  # HTTP handlers
│   │   ├── handler.go            # Common handler utilities
│   │   └── routes.go             # Route definition
│   ├── items/                    # Example CRUD resource
│   │   ├── handler.go            # Item CRUD handlers
│   │   └── models.go             # Item-related types
│   ├── middleware/               # Application middleware
│   │   ├── logging.go            # Request logging
│   │   ├── recover.go            # Panic recovery
│   │   └── security.go           # Security headers, CSRF, etc.
│   ├── server/                   # Server setup
│   │   └── server.go             # HTTP server configuration
│   └── view/                     # View rendering
│       └── renderer.go           # HTML template renderer
├── migrations/                   # Database migrations
│   ├── 0001_init.up.sql
│   └── 0001_init.down.sql
├── sql/                          # SQLC query files
│   ├── items.sql                 # Item queries
│   ├── subscriptions.sql         # Subscription queries
│   ├── users.sql                 # User settings queries
│   └── schema.sql                # Combined schema for sqlc
├── sqlc.yaml                     # SQLC configuration
├── web/
│   ├── static/                   # Static assets
│   │   ├── css/                  # CSS files
│   │   │   └── output.css        # Compiled Tailwind CSS
│   │   ├── js/                   # JavaScript files
│   │   │   ├── htmx.min.js
│   │   │   └── alpine.min.js
│   │   └── img/                  # Image assets
│   └── templates/                # HTML templates
│       ├── layouts/              # Base layouts
│       │   ├── guest.html        # Layout for unauthenticated users
│       │   └── app.html          # Layout for authenticated users
│       ├── partials/             # Reusable template parts
│       │   ├── header.html
│       │   ├── footer.html
│       │   └── nav.html
│       ├── auth/                 # Auth-related templates
│       │   ├── login.html
│       │   └── register.html
│       ├── items/                # Item CRUD templates
│       │   ├── list.html
│       │   ├── create.html
│       │   ├── edit.html
│       │   └── item-row.html     # HTMX partial for single item
│       └── pages/                # Static pages
│           ├── home.html
│           └── dashboard.html
├── .air.toml                     # Hot reload configuration
├── .env.example                  # Example environment variables
├── .golangci.yml                 # Linting configuration
├── docker-compose.yml            # Local development setup
├── Dockerfile                    # Production container
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
├── Taskfile.yml                  # Development tasks
└── README.md                     # Project documentation
```

This structure follows the standard Go project layout patterns and is organized to support the implementation phases outlined in the Go Web Application Implementation Guide.
