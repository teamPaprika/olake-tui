---
title: "Contributing"
weight: 1
---

# Contributing

This guide covers how to set up your development environment, follow code conventions, and submit changes.

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Build and test |
| PostgreSQL | 13+ | Metadata storage |
| Temporal | 1.22+ | Workflow orchestration |
| Git | 2.x | Version control |

## Development Setup

### 1. Clone and Build

```bash
git clone https://github.com/teamPaprika/olake-tui.git
cd olake-tui
go build -o olake-tui ./cmd/olake-tui/
```

### 2. Start Dependencies

Start PostgreSQL and Temporal. Using Docker Compose:

```bash
# PostgreSQL
docker run -d --name pg -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:16

# Temporal (dev server)
temporal server start-dev --db-filename temporal.db
```

### 3. Run Migrations

```bash
./olake-tui --dsn "postgres://postgres:postgres@localhost:5432/olake?sslmode=disable" --migrate
```

### 4. Run the TUI

```bash
./olake-tui \
  --dsn "postgres://postgres:postgres@localhost:5432/olake?sslmode=disable" \
  --temporal "localhost:7233" \
  --mode local
```

## Code Style

### General Rules

- Follow standard Go conventions (`gofmt`, `go vet`, `golint`)
- Use `goimports` for import ordering
- Keep functions focused — one responsibility per function
- Prefer returning errors over panicking

### Project-Specific Conventions

- **UI components** implement `tea.Model` (Bubble Tea pattern)
- **All data access** goes through the `Service` interface in `internal/service/interface.go`
- **No direct DB calls** from UI code — always use the service layer
- **Modal dialogs** go in `internal/ui/modals/`
- **Connector-specific forms** go in `internal/ui/connector_forms/`

### Naming

- Screen models: `{Screen}Model` (e.g., `DashboardModel`, `JobDetailModel`)
- Service methods: verb-first (e.g., `CreateSource`, `ListJobs`, `DeleteDestination`)
- Test files: `*_test.go` in the same package

### Error Handling

```go
// Good: wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create source: %w", err)
}

// Bad: bare error return
if err != nil {
    return err
}
```

## Pull Request Guide

### Before Submitting

1. **Run all tests:**
   ```bash
   go test ./...
   ```

2. **Run the linter:**
   ```bash
   go vet ./...
   ```

3. **Test manually** — launch the TUI and verify your changes work end-to-end

### PR Requirements

- **Clear title** describing the change (e.g., "Add Redis connector form")
- **Description** explaining what and why
- **Tests** for new service methods (use `MockService` for unit tests)
- **No breaking changes** to the `Service` interface without discussion
- **Screenshots or recordings** for UI changes (terminal recordings via `asciinema` are welcome)

### Branch Naming

```
feature/add-redis-connector
fix/job-wizard-validation
refactor/service-interface
docs/update-schema-docs
```

### Review Process

1. Open a PR against `master`
2. Ensure CI passes (tests + lint)
3. Request review from a maintainer
4. Address feedback
5. Squash-merge when approved

## Getting Help

- Check existing issues on GitHub
- Read the [Package Structure](../../architecture/package-structure/) to understand the codebase
- Look at existing connectors in `connector_forms/` for implementation patterns
