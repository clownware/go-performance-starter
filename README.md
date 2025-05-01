# Go Alpine SaaS Starter

_Your project description here._

## Prerequisites

- Go 1.22+
- Docker & Docker Compose (for local development database)
- `task` (Go Task runner)
- `air` (for live reload)

## Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/go-alpine-saas-starter.git
   cd go-alpine-saas-starter
   ```
2. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```
3. Update `.env` with your settings. **Crucially, set the `DATABASE_URL` to your Supabase project's connection string.**
4. Start the development database (if using Docker Compose):
   ```bash
   task db:up
   ```
5. Run database migrations:
   ```bash
   task db:migrate:up
   ```
6. Generate Go code from SQL queries:
   ```bash
   task db:generate
   ```
7. Install Go dependencies:
   ```bash
   go mod tidy
   ```
8. Start the development server (with live reload):
   ```bash
   task dev
   ```

The application should now be running at [http://localhost:4000](http://localhost:4000) (or the port specified in your `.env`).

## Available Tasks

Run `task --list` to see available development tasks defined in `Taskfile.yml`.

## Project Structure

See [docs/product/directory-structure.md](docs/product/directory-structure.md) for details.

## Architecture Decisions

See [docs/adr/ADR-001-Foundation.md](docs/adr/ADR-001-Foundation.md) for foundational architectural decisions.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
