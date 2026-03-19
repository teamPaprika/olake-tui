# olake-tui

**Terminal UI for [OLake](https://github.com/datazip-inc/olake) — manage data pipelines without leaving the command line.**

olake-tui connects directly to the OLake PostgreSQL database and Temporal cluster, bypassing the BFF HTTP layer entirely. It gives you a fast, keyboard-driven interface for monitoring and controlling sync jobs from any terminal.

---

## Screenshot

```
┌──────────────────────────────────────────────────────────────────┐
│  ⬡ OLake  logged in as admin                                     │
│                                                                   │
│  ╭──────────╮  ╭──────────────╮  ╭──────╮  ╭────────────╮  ╭───╮│
│  │ Sources: 3│  │ Destinations:2│  │Jobs:5│  │Active Jobs:3│  │⟳:1││
│  ╰──────────╯  ╰──────────────╯  ╰──────╯  ╰────────────╯  ╰───╯│
│                                                                   │
│  [1] Jobs  [2] Sources  [3] Destinations  [4] Settings            │
│                                                                   │
│  ✓  1  nightly-postgres-sync   pg-prod    s3-bucket  completed  ● │
│  ⟳  2  hourly-mysql-export     mysql-dev  iceberg    running    ● │
│  ✗  3  daily-mongo-backup      mongo      s3-archive failed     ● │
│  ·  4  weekly-report           pg-prod    parquet    —          ○ │
│                                                                   │
│  n:new  Enter:detail  S:settings  s:sync  c:cancel  l:logs       │
└──────────────────────────────────────────────────────────────────┘
```

---

## Features

- **Direct DB + Temporal** — no BFF server required
- **Full CRUD** — sources, destinations, jobs, settings
- **Job wizard** — create jobs with stream selection and sync mode config
- **17 modal dialogs** — delete confirmations, connection tests, clear destination
- **Real-time status** — live Temporal workflow state for each job
- **Paginated log viewer** — browse task logs with older/newer pagination
- **Standalone mode** — `--migrate` bootstraps the database without BFF
- **BFF compatible** — reads/writes the same schema, encryption, and Temporal schedules
- **112 tests** — unit + E2E compatibility suite

---

## Quick Start

### With existing OLake deployment

```bash
# Connect to your OLake database
olake-tui --db-url "postgres://olake:pass@localhost:5432/olake?sslmode=disable"
```

### Standalone (no BFF server needed)

```bash
# Create tables + admin user, then start TUI
olake-tui --db-url "postgres://olake:pass@localhost:5432/olake?sslmode=disable" \
          --migrate --admin-user admin --admin-pass changeme

# Or just migrate and exit (for CI / init containers)
olake-tui --db-url "..." --migrate-only
```

### With Kubernetes (port-forward)

```bash
kubectl port-forward svc/olake-postgresql 5432:5432 &
kubectl port-forward svc/olake-temporal-frontend 7233:7233 &
olake-tui --db-url "postgres://olake:pass@localhost:5432/olake?sslmode=disable"
```

---

## Installation

### From source

```bash
git clone https://github.com/teamPaprika/olake-tui.git
cd olake-tui
make install          # go install ./cmd/olake-tui/
```

### Build manually

```bash
go build -o bin/olake-tui ./cmd/olake-tui/
```

---

## Prerequisites

| Requirement | Notes |
|-------------|-------|
| **PostgreSQL** | The OLake database (or any empty PG with `--migrate`) |
| **Temporal** | Frontend at `localhost:7233` (optional — DB features work without it) |
| **OLake worker** | Required for sync / discover / connection-test operations |
| **Go 1.22+** | Only for building from source |

---

## Flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--db-url` | `OLAKE_DB_URL` | | PostgreSQL connection string (**required**) |
| `--temporal-host` | `TEMPORAL_ADDRESS` | `localhost:7233` | Temporal frontend address |
| `--project-id` | `OLAKE_PROJECT_ID` | `123` | OLake project ID |
| `--run-mode` | `OLAKE_RUN_MODE` | `dev` | Table prefix mode (`dev`\|`prod`\|`staging`) |
| `--encryption-key` | `OLAKE_SECRET_KEY` | | AES-256 key for connector config encryption |
| `--migrate` | | `false` | Create tables + seed admin user, then start TUI |
| `--migrate-only` | | `false` | Run migration and exit (don't start TUI) |
| `--admin-user` | `OLAKE_ADMIN_USER` | `admin` | Admin username for initial seed |
| `--admin-pass` | `OLAKE_ADMIN_PASSWORD` | `admin` | Admin password for initial seed |
| `--version` | | | Print version and exit |

---

## Key Bindings

### Global

| Key | Action |
|-----|--------|
| `1`–`4` / `Tab` | Switch tabs |
| `q` / `Ctrl+C` | Quit |
| `Esc` | Back / close overlay |

### Jobs

| Key | Action |
|-----|--------|
| `↑`/`↓`/`j`/`k` | Navigate |
| `Enter` | Job detail (task history) |
| `n` | New job wizard |
| `S` | Job settings |
| `s` | Trigger sync |
| `c` | Cancel running sync |
| `p` | Pause / resume |
| `l` | View logs |
| `d` | Delete |
| `u` | Updates |
| `r` | Refresh |

### Sources / Destinations

| Key | Action |
|-----|--------|
| `a` | Add new |
| `e` | Edit |
| `d` | Delete |
| `t` | Test connection |
| `r` | Refresh |

### Log Viewer

| Key | Action |
|-----|--------|
| `↑`/`↓`/`PgUp`/`PgDn` | Scroll |
| `p` | Load older logs |
| `n` | Load newer logs |
| `Esc` | Back |

---

## Architecture

```
┌─────────────┐     SQL (lib/pq)      ┌────────────┐
│  olake-tui  │ ───────────────────▶  │ PostgreSQL │
│  (Bubble    │                        └────────────┘
│   Tea TUI)  │  gRPC (Temporal SDK)  ┌────────────┐
│             │ ───────────────────▶  │  Temporal  │
└─────────────┘                        └────────────┘
```

**No HTTP layer.** olake-tui talks directly to:

- **PostgreSQL** — same schema as BFF (Beego ORM), same AES-256-GCM encryption
- **Temporal** — triggers syncs, manages schedules, queries workflow history

### Package Layout

```
cmd/olake-tui/          Entry point, flag parsing
internal/
  app/                  Root Bubble Tea model, key routing, modal state
  service/
    interface.go        Service interface (36 methods)
    service.go          Manager: DB + Temporal implementation
    migrate.go          Schema bootstrap (--migrate)
    mock.go             MockService for testing
    logs.go             Task log file reader
    schema.go           Schema validation
  ui/
    jobs.go             Jobs list view
    sources.go          Sources list view
    destinations.go     Destinations list view
    job_wizard.go       Job creation wizard (4 steps)
    job_detail.go       Task history view
    job_logs.go         Paginated log viewer
    job_settings.go     Job settings editor
    settings.go         System settings
    entity_form.go      Source/destination create/edit form
    connector_forms.go  Connector-specific field definitions
    streams.go          Stream selection + config popup
    modals.go           17 modal dialogs
    confirm.go          Yes/no confirmation
    login.go            Login screen
    dashboard.go        Dashboard stats
    styles.go           OLake brand colors + shared styles
tests/
  compat/               E2E tests against real OLake database
docs/
  BFF_COMPARISON.md     Feature parity analysis vs official BFF
```

---

## Database Migration

When using `--migrate`, olake-tui creates these tables (matching BFF schema):

| Table | Description |
|-------|-------------|
| `olake-{mode}-user` | User accounts (bcrypt passwords) |
| `olake-{mode}-source` | Data source connectors |
| `olake-{mode}-destination` | Data destination connectors |
| `olake-{mode}-job` | Sync jobs with schedules |
| `olake-{mode}-project-settings` | Project-level settings (webhook URL) |
| `olake-{mode}-catalog` | Connector specs (for BFF compatibility) |
| `session` | BFF session table (for compatibility) |

All statements use `IF NOT EXISTS` — safe to run repeatedly.

---

## BFF Compatibility

olake-tui is designed to be a **drop-in companion or replacement** for the OLake web UI:

- ✅ Same DB schema and table naming convention
- ✅ Same AES-256-GCM config encryption (interoperable with BFF)
- ✅ Same Temporal workflow/schedule naming
- ✅ Same soft-delete semantics (`deleted_at` column)
- ✅ Data created by TUI is visible in web UI and vice versa

See [docs/BFF_COMPARISON.md](docs/BFF_COMPARISON.md) for detailed feature parity analysis.

---

## Testing

```bash
# Unit tests (no external deps)
go test ./...

# E2E tests (requires real OLake PostgreSQL)
OLAKE_DB_URL=postgres://olake:olake@localhost:5432/olake?sslmode=disable \
OLAKE_SECRET_KEY=mysecretkey \
  go test -tags e2e ./tests/compat/ -v
```

**112 tests total:** 73 unit tests + 17 UI tests + 8 stream tests + 14 E2E compatibility tests.

---

## Configuration Examples

### `.env` file

```dotenv
OLAKE_DB_URL=postgres://olake:secret@localhost:5432/olake?sslmode=disable
TEMPORAL_ADDRESS=localhost:7233
OLAKE_SECRET_KEY=my-aes-encryption-key
OLAKE_RUN_MODE=dev
```

Load: `set -a && source .env && set +a && olake-tui`

### Docker Compose (standalone)

```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: olake
      POSTGRES_USER: olake
      POSTGRES_PASSWORD: olake
    ports: ["5432:5432"]

  temporal:
    image: temporalio/auto-setup:latest
    ports: ["7233:7233"]
    depends_on: [postgres]
```

Then:
```bash
olake-tui --db-url "postgres://olake:olake@localhost:5432/olake" --migrate
```

---

## License

Apache 2.0 — same as [OLake](https://github.com/datazip-inc/olake/blob/main/LICENSE).
