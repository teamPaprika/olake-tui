---
title: "Connecting"
weight: 1
---

OLake TUI needs three things to start: a database URL, a Temporal server address, and an encryption key. You can provide these via CLI flags, environment variables, or a `.env` file.

## Required Configuration

| Parameter | CLI Flag | Env Variable | Description |
|-----------|----------|--------------|-------------|
| Database URL | `--db-url` | `DATABASE_URL` | PostgreSQL connection string for OLake metadata |
| Temporal Address | `--temporal-address` | `TEMPORAL_ADDRESS` | Host and port of your Temporal server |
| Encryption Key | `--encryption-key` | `ENCRYPTION_KEY` | 32-byte hex key for encrypting sensitive config |

## Database URL

OLake TUI stores all metadata (sources, destinations, jobs) in PostgreSQL. Provide a standard connection string:

```bash
DATABASE_URL="postgres://olake:secret@localhost:5432/olake?sslmode=disable"
```

The database must already exist. Run with `--migrate` on first launch to create the schema (see [Authentication](../authentication/)).

## Temporal Address

OLake uses [Temporal](https://temporal.io) to orchestrate sync workflows. Point to your Temporal frontend service:

```bash
TEMPORAL_ADDRESS="localhost:7233"
```

This is the same Temporal instance used by the OLake backend workers.

## Encryption Key

Sensitive fields (database passwords, access keys) are encrypted at rest using AES-256-GCM. Generate a 32-byte hex key:

```bash
openssl rand -hex 32
```

Set it once and **do not change it** — existing encrypted values become unreadable if the key changes.

```bash
ENCRYPTION_KEY="a3f1b2c4d5e6f7081920abcdef1234567890abcdef1234567890abcdef123456"
```

## Using a `.env` File

Create a `.env` file in your working directory:

```dotenv
DATABASE_URL=postgres://olake:secret@localhost:5432/olake?sslmode=disable
TEMPORAL_ADDRESS=localhost:7233
ENCRYPTION_KEY=a3f1b2c4d5e6f7081920abcdef1234567890abcdef1234567890abcdef123456
```

OLake TUI loads `.env` automatically on startup. CLI flags take precedence over environment variables.

## CLI Flags

You can pass everything on the command line:

```bash
olake-tui \
  --db-url "postgres://olake:secret@localhost:5432/olake?sslmode=disable" \
  --temporal-address "localhost:7233" \
  --encryption-key "a3f1b2c4d5e6..."
```

## Precedence Order

1. CLI flags (highest)
2. Environment variables
3. `.env` file (lowest)

## Startup Checks

On launch, OLake TUI verifies:

- Database is reachable and migrations are up to date
- Temporal server responds to a health check
- Encryption key is exactly 32 bytes (64 hex characters)

If any check fails, the TUI prints an error and exits.

## Further Reading

- [OLake UI Installation Guide](https://olake.io/docs/install/olake-ui/) — full setup instructions including Docker Compose and backend workers
