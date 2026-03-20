---
title: "Configuration Reference"
weight: 4
---

# Configuration Reference

`olake-tui` is configured via CLI flags or environment variables. Flags take
precedence over environment variables.

## CLI Flags

| Flag | Env Variable | Default | Description |
|------|-------------|---------|-------------|
| `--db-url` | `OLAKE_DB_URL` | _(required)_ | PostgreSQL connection string |
| `--temporal-host` | `TEMPORAL_ADDRESS` | `localhost:7233` | Temporal frontend gRPC address |
| `--project-id` | `OLAKE_PROJECT_ID` | `123` | OLake project ID |
| `--run-mode` | `OLAKE_RUN_MODE` | `dev` | Beego run mode — controls table name prefixes (`dev`, `prod`, `staging`) |
| `--encryption-key` | `OLAKE_SECRET_KEY` | _(empty)_ | AES encryption key for credential storage |
| `--migrate` | — | `false` | Create OLake tables and seed admin user, then start TUI |
| `--migrate-only` | — | `false` | Run migration and exit without starting TUI |
| `--admin-user` | `OLAKE_ADMIN_USER` | `admin` | Admin username for initial seed (used with `--migrate`) |
| `--admin-pass` | `OLAKE_ADMIN_PASSWORD` | `admin` | Admin password for initial seed (used with `--migrate`) |
| `--release-url` | `OLAKE_RELEASE_URL` | _(empty)_ | URL to `releases.json` for connector version checks |
| `--version` | — | — | Print version string and exit |

## Flag Details

### --db-url

The only required flag. Accepts a standard PostgreSQL connection string:

```bash
olake-tui --db-url "postgres://user:password@host:5432/dbname?sslmode=disable"
```

The database is shared with Temporal. OLake tables are created in the same database
using a distinct schema/table prefix controlled by `--run-mode`.

### --temporal-host

Address of the Temporal frontend service. When running Docker Compose locally this
is `localhost:7233`. In Kubernetes, use the in-cluster service name
(e.g., `olake-temporal:7233`).

### --run-mode

Controls the table name prefix for OLake data:

| Mode | Table Prefix | Use Case |
|------|-------------|----------|
| `dev` | `dev_` | Local development |
| `prod` | `prod_` | Production |
| `staging` | `staging_` | Staging / QA |

### --encryption-key

When set, credentials (source/destination passwords, tokens) are encrypted at rest
in PostgreSQL using AES. This should match the `OLAKE_SECRET_KEY` value used by the
OLake Worker in `docker-compose.yml`.

If left empty, credentials are stored in plaintext. **Set this in production.**

### --migrate / --migrate-only

Both flags trigger database schema creation and admin user seeding:

- `--migrate` — runs migration, then starts the TUI normally
- `--migrate-only` — runs migration and exits (useful for CI/CD and Helm Jobs)

Migration is idempotent — safe to run on every startup.

### --release-url

Points to a JSON endpoint listing available connector versions. When omitted,
version checking is disabled entirely (suitable for [air-gapped environments]({{< relref "air-gapped" >}})).

### --version

Prints the build version (set via ldflags at compile time) and exits:

```bash
$ olake-tui --version
olake-tui v0.2.0-direct
```

## Environment Variable Quick Reference

For environments where CLI flags are inconvenient (containers, systemd units),
use environment variables:

```bash
export OLAKE_DB_URL="postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable"
export TEMPORAL_ADDRESS="localhost:7233"
export OLAKE_PROJECT_ID="123"
export OLAKE_RUN_MODE="prod"
export OLAKE_SECRET_KEY="my-32-byte-encryption-key-here!"
export OLAKE_ADMIN_USER="admin"
export OLAKE_ADMIN_PASSWORD="s3cur3-p4ssw0rd"
export OLAKE_RELEASE_URL="https://artifacts.internal.example.com/olake/releases.json"

olake-tui --migrate
```

## Helm Values Reference (olake-tui chart)

When deploying via Helm, the chart's `values.yaml` maps to the same flags:

```yaml
# values.yaml — olake-tui Helm chart
image:
  repository: olakego/olake-tui
  tag: "latest"
  pullPolicy: IfNotPresent
  # For air-gapped: override the registry
  registry: ""

db:
  # PostgreSQL connection string (maps to --db-url / OLAKE_DB_URL)
  url: "postgres://temporal:temporal@olake-postgresql:5432/temporal?sslmode=disable"

temporal:
  # Temporal frontend address (maps to --temporal-host / TEMPORAL_ADDRESS)
  host: "olake-temporal:7233"

olake:
  # Project ID (maps to --project-id / OLAKE_PROJECT_ID)
  projectId: "123"
  # Run mode (maps to --run-mode / OLAKE_RUN_MODE)
  runMode: "dev"
  # Encryption key (maps to --encryption-key / OLAKE_SECRET_KEY)
  encryptionKey: ""
  # Release URL (maps to --release-url / OLAKE_RELEASE_URL)
  releaseUrl: ""

migrate:
  # Run migration Job on install/upgrade
  enabled: true
  # Admin credentials for initial seed
  adminUser: "admin"
  adminPassword: "admin"
```

### Overriding Values

```bash
helm install olake-tui olake/olake-tui \
  --namespace olake \
  --set db.url="postgres://user:pass@db:5432/olake?sslmode=disable" \
  --set temporal.host="temporal:7233" \
  --set olake.runMode="prod" \
  --set olake.encryptionKey="my-secret-key" \
  --set migrate.adminPassword="s3cur3"
```

Or provide a custom values file:

```bash
helm install olake-tui olake/olake-tui -f my-values.yaml -n olake
```
