---
title: "CLI Reference"
weight: 1
---

# olake-tui

Terminal UI for managing OLake sources, destinations, jobs, and streams.

## Synopsis

```
olake-tui [flags]
```

## Flags

| Flag | Type | Env Var | Default | Description |
|------|------|---------|---------|-------------|
| `--db-url` | string | `OLAKE_DB_URL` | _(required)_ | PostgreSQL connection string |
| `--temporal-host` | string | `TEMPORAL_ADDRESS` | `localhost:7233` | Temporal frontend address |
| `--project-id` | string | `OLAKE_PROJECT_ID` | `123` | OLake project ID |
| `--run-mode` | string | `OLAKE_RUN_MODE` | `dev` | Table prefix mode: `dev`, `prod`, or `staging` |
| `--encryption-key` | string | `OLAKE_SECRET_KEY` | _(none)_ | AES-256 key for config encryption |
| `--migrate` | bool | — | `false` | Create tables + seed admin user, then start TUI |
| `--migrate-only` | bool | — | `false` | Run migration and exit |
| `--admin-user` | string | `OLAKE_ADMIN_USER` | `admin` | Admin username for seed |
| `--admin-pass` | string | `OLAKE_ADMIN_PASSWORD` | `admin` | Admin password for seed |
| `--release-url` | string | `OLAKE_RELEASE_URL` | _(none)_ | URL to `releases.json` for update checks |
| `--version` | bool | — | — | Print version and exit |

## Flag Details

### --db-url (required)

PostgreSQL connection string. This is the only required flag—without it, `olake-tui` will not start.

```
--db-url "postgres://user:pass@localhost:5432/olake?sslmode=disable"
```

### --run-mode

Controls the table prefix. Use `dev` during development, `prod` for production, and `staging` for staging environments. Tables are prefixed as `{mode}_sources`, `{mode}_destinations`, etc.

### --encryption-key

When set, source and destination configs are encrypted at rest using AES-256-GCM. The key must be exactly 32 bytes (or a hex/base64-encoded 32-byte value). If omitted, configs are stored in plaintext.

### --migrate / --migrate-only

`--migrate` creates all required tables, seeds an admin user, then launches the TUI. `--migrate-only` does the same but exits immediately after migration—useful for CI/CD pipelines.

## Examples

**First run with migration:**

```bash
olake-tui --db-url "postgres://olake:secret@localhost:5432/olake" --migrate
```

**Production with encryption:**

```bash
olake-tui \
  --db-url "postgres://olake:secret@db.prod:5432/olake?sslmode=require" \
  --run-mode prod \
  --encryption-key "my-32-byte-aes-encryption-key!!" \
  --temporal-host temporal.prod:7233
```

**Using environment variables:**

```bash
export OLAKE_DB_URL="postgres://olake:secret@localhost:5432/olake"
export TEMPORAL_ADDRESS="localhost:7233"
export OLAKE_RUN_MODE=staging
export OLAKE_SECRET_KEY="my-32-byte-aes-encryption-key!!"
olake-tui
```

**Migration only (CI/CD):**

```bash
olake-tui \
  --db-url "postgres://olake:secret@localhost:5432/olake" \
  --migrate-only \
  --admin-user deploy-bot \
  --admin-pass "$DEPLOY_PASSWORD"
```

**Check version:**

```bash
olake-tui --version
```

## Environment Variable Precedence

Flags always override environment variables. If both `--db-url` and `OLAKE_DB_URL` are set, the flag value wins.
