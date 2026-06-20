# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash

# Build for both platforms and generates go code from templ templates
./build.sh

# Run with verbose logging
./bin/jiraworklog -v

# Run with custom config
./bin/jiraworklog -c /path/to/config.yaml
```

## Development Workflow

```bash
# Terminal 1: Watch templ files and auto-regenerate
templ generate --watch

# Terminal 2: Run the application
go run ./cmd/jiraworklog -v
```

After modifying `.templ` files, the generated `_templ.go` files must be regenerated.

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
