# ADR-006: Task Automation - Taskfile

*   **Status:** Accepted
*   **Date:** 2025-05-01

## Context

The project involves various repetitive development and build tasks, such as:
*   Running database migrations (`golang-migrate` - see ADR-002)
*   Generating Go code from SQL (`sqlc` - see ADR-003)
*   Running the development server
*   Potentially linting, testing, and building the application in the future.

Managing these manually or with disparate shell scripts is inefficient, error-prone, and inconsistent across developer environments (especially considering Windows and Unix-based systems).

## Decision

Use `Taskfile` (taskfile.dev) as the project's command and task runner. Define common development and build tasks in a `Taskfile.yml` file located at the project root. Leverage features like `dotenv` loading for environment variable management within tasks.

## Consequences

### Pros:

*   **Consistency:** Provides a single, consistent interface (`task <task_name>`) for executing common project operations across different developer machines and CI/CD environments.
*   **Simplicity:** YAML syntax is generally considered simpler and more readable than traditional Makefiles.
*   **Cross-Platform:** `Taskfile` itself is a Go binary, making it inherently cross-platform (Windows, macOS, Linux).
*   **Features:** Supports task dependencies, environment variable loading (`dotenv:` directive), conditional execution, and clear task descriptions.
*   **Discoverability:** Running `task --list` provides an overview of available project tasks.
*   **CI/CD Integration:** Simplifies CI/CD pipeline scripts, as they can primarily invoke `task` commands.

### Cons:

*   **Tool Dependency:** Introduces a new development dependency (`go-task/task`) that needs to be installed by developers and in CI/CD environments.
*   **Learning Curve:** Developers unfamiliar with `Taskfile` need to learn its basic syntax and concepts (though it is relatively straightforward).

## Enforcement
<!-- added 2026-07-12, see ADR-033 (Enforcement Architecture) -->
- **Testable consequences:**
  - TC-1: `Taskfile.yml` exists at the repo root and defines the `ci` gate.
- **Checks:**
  - TC-1 → every CI run invokes `task ci`; absence fails immediately (status: **block**, pre-existing)
- **Not machine-checkable:** None — this ADR is fully structural.
- **Graduation log:** _(empty)_
