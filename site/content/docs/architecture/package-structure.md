---
title: "Package Structure"
weight: 2
---


olake-tui follows standard Go project layout conventions with a clear separation between entry point, business logic, and UI components.

## Top-Level Layout

```
olake-tui/
├── cmd/olake-tui/       # Entry point and CLI flags
├── internal/
│   ├── app/             # Bubble Tea application core
│   ├── service/         # Data access and business logic
│   └── ui/              # Screen components and forms
├── tests/compat/        # End-to-end compatibility tests
├── docs/                # Design documents
└── site/                # Hugo documentation (this site)
```

## `cmd/olake-tui/`

The main package. Parses CLI flags (database DSN, Temporal address, run mode, `--migrate`) and initializes the application.

| Flag | Purpose |
|------|---------|
| `--dsn` | PostgreSQL connection string |
| `--temporal` | Temporal server address |
| `--mode` | Run mode (`local`, `docker`, etc.) |
| `--migrate` | Run database migrations and exit |
| `--version` | Print version and exit |

## `internal/app/`

The Bubble Tea root model that drives the entire TUI. Responsibilities:

- **Screen routing** — Manages 13 distinct screens and transitions between them
- **Key routing** — Global key bindings (quit, back, help) and per-screen delegation
- **Modal state** — Controls overlay modals (confirmations, errors, pickers)
- **Service injection** — Passes the service manager to all child components

The root model implements `tea.Model` and orchestrates the full application lifecycle.

## `internal/service/`

The data layer. All database and Temporal operations are defined here.

| File | Purpose |
|------|---------|
| `interface.go` | `Service` interface with **36 methods** covering all operations |
| `service.go` | `Manager` struct — the production implementation |
| `mock.go` | `MockService` — in-memory implementation for unit tests |
| `migrate.go` | Database migration logic (table creation, schema updates) |
| `logs.go` | Temporal workflow log retrieval and streaming |
| `schema.go` | JSON schema handling for connector configurations |
| `releases.go` | Docker image tag/version fetching for connectors |

The `Service` interface is the **central abstraction** — every UI component depends only on this interface, never on concrete implementations. This makes the entire UI layer testable with `MockService`.

## `internal/ui/`

All screen and form components. Each subdirectory is a self-contained Bubble Tea model.

### Screen Components

| Package | Description |
|---------|-------------|
| `login` | Authentication screen |
| `dashboard` | Main navigation hub |
| `sources` | Source list with create/edit/delete |
| `source_detail` | Single source detail view |
| `destinations` | Destination list with create/edit/delete |
| `dest_detail` | Single destination detail view |
| `jobs` | Job list and management |
| `job_wizard` | Multi-step job creation flow |
| `job_detail` | Job overview and actions |
| `job_logs` | Real-time workflow log viewer |
| `job_settings` | Job configuration editor |
| `settings` | Project-level settings |
| `streams` | Stream selection and configuration |

### Shared Components

| Package | Description |
|---------|-------------|
| `entity_form` | Generic form builder for sources and destinations |
| `connector_forms` | Connector-specific field definitions (MongoDB, MySQL, PostgreSQL, S3, etc.) |
| `modals` | **17 modal dialogs** — error, confirm, picker, input, progress, etc. |
| `confirm` | Standalone confirmation prompts |
| `styles` | Shared lipgloss styles, colors, and layout constants |

## `tests/compat/`

End-to-end compatibility tests that verify olake-tui works against real PostgreSQL and Temporal instances. These **14 tests** ensure data created by the TUI is fully compatible with the BFF-based web UI.

See [Testing](../../development/testing/) for how to run them.

## Dependency Flow

```
cmd/olake-tui/
    └── internal/app/
            ├── internal/service/  (interface + Manager)
            └── internal/ui/*      (all screens depend on Service interface)
```

UI components never import each other directly — screen transitions are managed by the root app model. All data access goes through the `Service` interface, keeping the UI layer decoupled from infrastructure.
