# olake-tui

**Terminal UI for [OLake](https://github.com/datazip-inc/olake) — manage data pipelines without leaving the command line.**

olake-tui connects directly to the OLake PostgreSQL database and Temporal cluster, bypassing the BFF HTTP layer entirely.  It gives you a fast, keyboard-driven interface for monitoring and controlling sync jobs from any terminal.

---

## Screenshot

```
┌──────────────────────────────────────────────────────────────────┐
│  OLake TUI  ·  Sources: 3  Destinations: 2  Jobs: 5              │
├──────────────────────────────────────────────────────────────────┤
│  [1] Jobs  [2] Sources  [3] Destinations  [4] Settings [5] System│
├──────────────────────────────────────────────────────────────────┤
│  ▶  nightly-postgres-sync     postgres → s3    RUNNING   *       │
│     hourly-mysql-export       mysql    → bq    SUCCESS           │
│     …                                                            │
└──────────────────────────────────────────────────────────────────┘
  s:sync  c:cancel  l:logs  p:pause  d:delete  r:refresh  q:quit
```

*(screenshot placeholder — run `olake-tui` to see the real thing)*

---

## Prerequisites

| Requirement | Notes |
|-------------|-------|
| **PostgreSQL** | The same database the OLake server uses |
| **Temporal** | Frontend reachable at `localhost:7233` (or override with `--temporal-host`) |
| **OLake worker** | Must be running for sync / connection-test operations |
| **Go 1.22+** | Only required when building from source |

---

## Installation

### From source

```bash
git clone https://github.com/datazip-inc/olake-tui.git
cd olake-tui
make install          # go install ./cmd/olake-tui/
```

### Pre-built binary

```bash
# macOS (arm64)
curl -L https://github.com/datazip-inc/olake-tui/releases/latest/download/olake-tui-darwin-arm64 \
  -o /usr/local/bin/olake-tui && chmod +x /usr/local/bin/olake-tui
```

---

## Usage

```bash
olake-tui --db-url "postgres://user:pass@localhost:5432/olake?sslmode=disable" \
          --temporal-host localhost:7233
```

All flags can be replaced by environment variables (see [Configuration](#configuration)).

```bash
export OLAKE_DB_URL="postgres://user:pass@localhost:5432/olake?sslmode=disable"
olake-tui
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--db-url` | `$OLAKE_DB_URL` | PostgreSQL connection string (**required**) |
| `--temporal-host` | `localhost:7233` | Temporal frontend address |
| `--project-id` | `123` | OLake project ID |
| `--run-mode` | `dev` | Beego run mode — controls table name prefix (`dev`\|`prod`\|`staging`) |
| `--encryption-key` | `$OLAKE_SECRET_KEY` | AES-256 key used to decrypt connector configs |
| `--version` | | Print version and exit |

---

## Key Bindings

### Global

| Key | Action |
|-----|--------|
| `1` – `5` / `Tab` | Switch tabs |
| `q` / `Ctrl+C` | Quit |
| `Esc` | Back / close overlay |

### Jobs tab

| Key | Action |
|-----|--------|
| `↑` / `↓` / `j` / `k` | Navigate list |
| `Enter` | Open job detail |
| `s` | Trigger sync |
| `c` | Cancel running sync |
| `p` | Pause / resume job |
| `l` | View logs |
| `S` | Job settings |
| `n` | New job wizard |
| `d` | Delete job |
| `r` | Refresh list |

### Sources / Destinations tabs

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate list |
| `d` | Delete |
| `r` | Refresh |

### Log viewer

| Key | Action |
|-----|--------|
| `↑` / `↓` / `PgUp` / `PgDn` | Scroll |
| `Esc` | Back |

---

## Configuration

All configuration can be supplied as CLI flags or environment variables:

| Variable | Flag | Description |
|----------|------|-------------|
| `OLAKE_DB_URL` | `--db-url` | PostgreSQL connection string |
| `TEMPORAL_ADDRESS` | `--temporal-host` | Temporal frontend host:port |
| `OLAKE_SECRET_KEY` | `--encryption-key` | AES encryption key for connector configs |
| `OLAKE_RUN_MODE` | `--run-mode` | Table prefix mode (`dev`\|`prod`\|`staging`) |
| `OLAKE_PROJECT_ID` | `--project-id` | Project ID (numeric string) |

### `.env` example

```dotenv
OLAKE_DB_URL=postgres://olake:secret@localhost:5432/olake?sslmode=disable
TEMPORAL_ADDRESS=localhost:7233
OLAKE_SECRET_KEY=my-aes-key
OLAKE_RUN_MODE=dev
```

Load with: `set -a && source .env && set +a && olake-tui`

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

**No HTTP.** olake-tui talks directly to:

- **PostgreSQL** — reads/writes sources, destinations, jobs, and settings using the exact same schema as the OLake server.
- **Temporal** — triggers syncs, pauses/resumes schedules, and queries workflow history via the Go SDK client.

This means the OLake BFF server does **not** need to be running; only the database and Temporal cluster are required.

### Package layout

```
cmd/olake-tui/      — entry point, flag parsing
internal/
  app/              — root Bubble Tea model + key bindings
  service/          — Service interface + Manager implementation
    interface.go    — Service interface (UI depends on this)
    service.go      — *Manager: direct DB + Temporal client
    schema.go       — ValidateSchema() startup check
  ui/               — individual screen models (jobs, sources, …)
```

---

## Contributing

1. Fork and clone the repository.
2. Create a feature branch: `git checkout -b feat/my-feature`.
3. Make changes — run `make lint` and `make test` before committing.
4. Open a PR against `main`.

Please follow the existing code style and keep commits focused.  For larger changes, open an issue first to discuss the design.

---

## License

Apache 2.0 — same as [OLake](https://github.com/datazip-inc/olake/blob/main/LICENSE).

---

## Compatibility Matrix

| olake-tui version | OLake version |
|-------------------|---------------|
| `0.1.x`           | `>= 1.0.0`    |
| `0.2.x`           | `>= 1.0.0`    |

> **Note:** The TUI performs a schema validation check on startup (`ValidateSchema()`).  If your OLake database schema is incompatible you will receive a descriptive error with migration instructions.
