# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Building — ALWAYS use ./build.sh

**Rule: to build or verify a change, ALWAYS run `./build.sh`. Do NOT run `go build`, `templ generate`, or `go run` directly for this purpose.**

`./build.sh` is the single source of truth for building: it runs `templ generate` (regenerating the `_templ.go` files from `.templ` templates), compiles the Tailwind CSS, and builds both the Linux and Windows binaries. Running `go build` on its own can compile against stale generated templates and silently miss `.templ` changes; `build.sh` prevents that.

```bash
# Build everything (templ generate + CSS + Linux/Windows binaries). This is THE build command.
./build.sh

# Run the built binary
./bin/jiraworklog -v                       # verbose logging
./bin/jiraworklog -c /path/to/config.yaml  # custom config
```

Because `build.sh` cd's into the repo root and runs a one-shot `templ generate`, it avoids the templ caching pitfalls that can occur when mixing `templ generate` with `templ generate --watch`. Prefer `build.sh` over invoking the tools individually.

## Development Workflow (optional, human hot-reload only)

For interactive iteration a developer may run templ in watch mode and `go run` directly. This is a convenience loop for a human, **not** the build/verify path — Claude should still use `./build.sh` to build and verify.

```bash
# Terminal 1: Watch templ files and auto-regenerate
templ generate --watch

# Terminal 2: Run the application
go run ./cmd/jiraworklog -v
```

## Running Tests

Tests are integration tests that require a database connection via `bin/config.yaml`:

```bash
go test ./test/...
```

## Architecture Overview

This is a Go web application that syncs Jira worklogs/issues to PostgreSQL and provides reporting analytics.

### Key Components

- **cmd/jiraworklog/**: Entry point with Echo web server setup and HTTP handlers
- **repository/**: Repository pattern for database access (`repo.go` interface, `posgres.go` implementation)
- **job/**: Background sync jobs using a worker pattern
  - `JiraSyncWorklogsJob` (60s): Syncs updated/deleted worklogs from Jira
  - `JiraSyncIssuesJob` (30s): Syncs issues and parent issue details
- **templates/**: Templ templates (type-safe HTML generation)
  - `pages/`: Full-page templates
  - `components/`: Reusable UI components
  - `layouts/`: Base layout with navigation
- **types/**: Domain types (Worklog, Issue, ParentIssue, reporting aggregates)
- **jira.go**: Jira REST API client with retry logic
- **worker.go**: Generic job scheduler that runs jobs at configured intervals

### Tech Stack

- **Web Framework**: Echo v4
- **Templates**: templ (a-h/templ) - compiles to Go code
- **Database**: PostgreSQL via sqlx + pgx/v5 driver
- **CSS**: Tailwind CSS + Hotwire Turbo for navigation
- **Logging**: slog (structured logging)

### Request Flow

1. Echo router dispatches to handler in `cmd/jiraworklog/handlers.go`
2. Handler queries database via `repository.Repo` interface
3. Handler renders templ template or returns JSON (content negotiation via Accept header)

### Configuration

`config.yaml` stores Jira credentials, database connection, sync state (timestamps), and user filters. Environment variables `JWL_JIRA_URL`, `JWL_JIRA_USERNAME`, `JWL_JIRA_PASSWORD` can override config values.

### CLI Flags

- `-c, --config`: Config file path (default: config.yaml)
- `-p, --port`: HTTP port (default: 8380)
- `-d, --debug`: Enable debug logging
