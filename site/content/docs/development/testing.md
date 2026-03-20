---
title: "Testing"
weight: 2
---

# Testing

olake-tui has two test layers: **unit tests** using `MockService` and **end-to-end tests** against real PostgreSQL and Temporal instances.

## Test Summary

| Type | Count | Location | Dependencies |
|------|-------|----------|-------------|
| Unit | 112 | `internal/` packages | None (MockService) |
| E2E | 14 | `tests/compat/` | PostgreSQL + Temporal |

## Unit Tests

### Overview

Unit tests cover service logic and UI component behavior using `MockService` — an in-memory implementation of the `Service` interface defined in `internal/service/mock.go`.

`MockService` stores all data in Go maps and slices, requiring no external dependencies. This makes unit tests fast, isolated, and deterministic.

### Running Unit Tests

```bash
# All unit tests
go test ./internal/...

# Specific package
go test ./internal/service/...
go test ./internal/ui/jobs/...

# With verbose output
go test -v ./internal/...

# With coverage
go test -cover ./internal/...
```

### Test Structure

Unit tests follow standard Go conventions:

```go
func TestCreateSource(t *testing.T) {
    svc := service.NewMockService()

    src, err := svc.CreateSource("test-mongo", "mongodb", config)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if src.Name != "test-mongo" {
        t.Errorf("expected name 'test-mongo', got '%s'", src.Name)
    }
}
```

### What's Covered

- **Service methods** — All 36 interface methods have corresponding unit tests
- **Validation logic** — Input validation for connector configs, job parameters
- **Encryption round-trips** — AES-256-GCM encrypt → decrypt produces original data
- **State transitions** — UI model updates in response to messages
- **Edge cases** — Empty lists, duplicate names, soft-deleted records

## End-to-End Tests

### Overview

E2E tests in `tests/compat/` verify that olake-tui works correctly against real infrastructure and that data created by the TUI is compatible with the BFF-based web UI.

### Prerequisites

Running E2E tests requires:

1. **PostgreSQL** — A running instance with a test database
2. **Temporal** — A running dev server

```bash
# Start dependencies
docker run -d --name pg-test -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:16
temporal server start-dev --db-filename temporal-test.db
```

### Running E2E Tests

```bash
# Set connection details
export TEST_DSN="postgres://postgres:postgres@localhost:5432/olake_test?sslmode=disable"
export TEST_TEMPORAL="localhost:7233"

# Run E2E tests
go test ./tests/compat/...

# Verbose output
go test -v ./tests/compat/...
```

### What's Covered

The 14 E2E tests verify:

- **Migration** — `--migrate` creates all expected tables
- **CRUD operations** — Create, read, update, delete for sources, destinations, and jobs
- **Encryption compatibility** — Credentials encrypted by TUI can be decrypted by BFF logic
- **Temporal integration** — Workflow and schedule creation with correct naming
- **Soft delete** — Deleted records are hidden but preserved
- **Cross-compatibility** — Data format matches BFF expectations exactly

### Test Isolation

Each E2E test:
1. Creates a fresh test database (or truncates tables)
2. Runs migrations
3. Performs operations
4. Validates results
5. Cleans up

Tests are safe to run in parallel with `--parallel` flag.

## CI Integration

Tests run automatically on pull requests. The CI pipeline:

1. Starts PostgreSQL and Temporal via Docker
2. Runs `go vet ./...`
3. Runs unit tests: `go test ./internal/...`
4. Runs E2E tests: `go test ./tests/compat/...`
5. Reports coverage

## Writing New Tests

- **Service logic** → Add unit tests with `MockService`
- **UI behavior** → Test the Bubble Tea model's `Update` method
- **Data compatibility** → Add E2E tests in `tests/compat/`
- **New connectors** → Test field validation and JSON conversion in unit tests
