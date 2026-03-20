---
title: "Installation"
weight: 3
---

# Installation

olake-tui is a single Go binary with no runtime dependencies. You can build from source, pull the Docker image, or install directly with `go install`. This page covers every method, plus how to connect to local, remote, and Kubernetes-hosted OLake infrastructure.

## Prerequisites

### Required

| Tool | Version | Check command | Purpose |
|------|---------|---------------|---------|
| **Go** | 1.22+ | `go version` | Building from source |
| **PostgreSQL 13+** | — | `psql --version` | OLake stores all data here |
| **Temporal Server** | — | `temporal --version` | Workflow orchestration |

**Go 1.22 is required** because olake-tui uses language features introduced in that version (range-over-int, enhanced routing patterns). Building with an older Go version will produce compile errors.

### Optional but recommended

| Tool | Purpose |
|------|---------|
| **Docker & Docker Compose** | Running infrastructure locally (PostgreSQL, Temporal, Worker) |
| **psql** (PostgreSQL client) | Debugging database issues, verifying schema |
| **golangci-lint** | Running linters (`make lint`) |
| **temporal CLI** | Debugging Temporal workflows from the command line |
| **make** | Using the project Makefile (comes pre-installed on macOS/Linux) |

### Don't have Go installed?

You have two options:

1. **Use the Docker image** — no Go required. See [Docker image](#docker-image) below.
2. **Install Go** — download from [go.dev/dl](https://go.dev/dl/). On macOS: `brew install go`. On Ubuntu: `sudo snap install go --classic`.

## Build from source

### Clone and build

```bash
git clone https://github.com/teamPaprika/olake-tui.git
cd olake-tui
make build
```

**Expected output:**

```
go build -ldflags "-X main.version=v0.2.0-direct" -o bin/olake-tui ./cmd/olake-tui/
```

The binary is at `bin/olake-tui` (~10MB). No other files are needed — it's a single statically-compiled binary.

### Install to $GOPATH/bin

If you want olake-tui available system-wide (assuming `$GOPATH/bin` is in your `$PATH`):

```bash
make install
```

**Expected output:**

```
go install -ldflags "-X main.version=v0.2.0-direct" ./cmd/olake-tui/
```

Verify it's in your path:

```bash
which olake-tui
# /Users/you/go/bin/olake-tui
```

### Using go build directly (without Make)

If you don't have `make` or prefer explicit commands:

```bash
go build -ldflags "-X main.version=v0.2.0-direct" -o bin/olake-tui ./cmd/olake-tui/
```

### Using go install directly

```bash
go install -ldflags "-X main.version=v0.2.0-direct" ./cmd/olake-tui/
```

This places the binary in `$GOPATH/bin/olake-tui`.

## Verifying the installation

After building or installing, verify olake-tui works:

```bash
# Check version
bin/olake-tui --version
```

```
olake-tui v0.2.0-direct
```

```bash
# Check help
bin/olake-tui --help
```

```
Usage of olake-tui:
  -admin-pass string
        Admin password for initial seed (default "admin")
  -admin-user string
        Admin username for initial seed (default "admin")
  -db-url string
        PostgreSQL connection string
  -encryption-key string
        AES encryption key
  -migrate
        Create OLake tables if they don't exist, seed admin user, then start TUI
  -migrate-only
        Run migration and exit (don't start TUI)
  -project-id string
        OLake project ID (default "123")
  -release-url string
        URL to releases.json for update checks (omit for air-gapped)
  -run-mode string
        Beego run mode (dev|prod|staging) (default "dev")
  -temporal-host string
        Temporal frontend address (default "localhost:7233")
  -version
        Print version and exit
```

{{< callout type="info" >}}
**All flags can also be set via environment variables.** See the [Environment variables](#environment-variables) section below.
{{< /callout >}}

## Makefile targets

The project Makefile provides these targets:

| Target | Command | Description |
|--------|---------|-------------|
| `make build` | `go build ... -o bin/olake-tui` | Compile binary to `bin/` directory |
| `make install` | `go install ...` | Install to `$GOPATH/bin` |
| `make run` | `go run ./cmd/olake-tui/` | Build and run in one step (for development) |
| `make test` | `go test ./...` | Run all tests |
| `make lint` | `golangci-lint run` | Run linters (requires golangci-lint) |
| `make clean` | `rm -rf bin/` | Remove build artifacts |

All build targets inject the version string via `-ldflags "-X main.version=$(VERSION)"`. The default version is `v0.2.0-direct`. Override it:

```bash
make build VERSION=v1.0.0
```

### Running tests

```bash
make test
```

**Expected output:**

```
go test ./...
ok      github.com/datazip-inc/olake-tui/internal/app      0.015s
ok      github.com/datazip-inc/olake-tui/internal/service   0.023s
ok      github.com/datazip-inc/olake-tui/internal/ui        0.031s
ok      github.com/datazip-inc/olake-tui/tests/compat       0.018s
```

### Running linters

```bash
# Install golangci-lint first if you don't have it
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

make lint
```

## Docker image

A pre-built Docker image is available for running olake-tui without Go:

```bash
docker pull teampaprika/olake-tui
```

### Interactive mode (standard usage)

```bash
docker run -it --rm \
  --network host \
  teampaprika/olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --temporal-host localhost:7233
```

| Flag | Required? | Why |
|------|-----------|-----|
| `-it` | **Yes** | olake-tui is an interactive TUI — it needs a terminal (stdin + TTY) |
| `--rm` | Recommended | Clean up the container after exit |
| `--network host` | For local infra | Lets the container reach PostgreSQL and Temporal on `localhost` |

### Migration-only mode (CI/CD)

For non-interactive use — bootstrapping the database in a CI pipeline or Kubernetes init container:

```bash
docker run --rm \
  --network host \
  teampaprika/olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --migrate-only \
  --admin-user admin \
  --admin-pass changeme
```

**Expected output:**

```
Running database migration...
✓ Schema migration complete
Seeding admin user (admin)...
✓ Admin user ready
Migration complete. Exiting.
```

Note: no `-it` flags needed for `--migrate-only` since it exits after migration.

### Docker with custom network

If your infrastructure runs on a Docker network instead of host networking:

```bash
docker run -it --rm \
  --network olake-tui_olake-network \
  teampaprika/olake-tui \
  --db-url "postgres://temporal:temporal@temporal-postgresql:5432/temporal?sslmode=disable" \
  --temporal-host temporal:7233
```

Note the hostnames changed from `localhost` to container names (`temporal-postgresql`, `temporal`).

## Infrastructure with Docker Compose

The repository includes a `docker-compose.yml` that runs all required infrastructure:

```bash
cd olake-tui
docker compose up -d
```

This starts:

| Service | Container | Port | Credentials |
|---------|-----------|------|-------------|
| **PostgreSQL 13** | `temporal-postgresql` | `5432` | user: `temporal`, password: `temporal` |
| **Temporal Server** | `temporal` | `7233` (gRPC) | — |
| **Elasticsearch 7.17** | `temporal-elasticsearch` | `9200` | — |
| **OLake Worker** | `olake-worker` | — | — |

Data is persisted in:
- Docker volume `temporal-postgresql-data` — PostgreSQL data
- Local directory `olake-data/` — OLake worker config and state

### Connect olake-tui to local infrastructure

```bash
bin/olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --temporal-host localhost:7233 \
  --migrate \
  --admin-user admin \
  --admin-pass changeme
```

### Debug profile (Temporal Web UI)

To also start the Temporal Web UI (useful for inspecting workflows):

```bash
docker compose --profile debug up -d
# Temporal UI available at http://localhost:8081
```

### Stopping infrastructure

```bash
docker compose down          # Stop and remove containers (data persisted in volumes)
docker compose down -v       # Stop, remove containers AND delete volumes (data lost!)
```

## Connecting to different environments

### Local development (default)

This is the most common setup — infrastructure running locally via Docker Compose:

```bash
olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --temporal-host localhost:7233
```

### Remote server via SSH

If PostgreSQL and Temporal are on a remote server, use SSH port forwarding:

```bash
# Forward PostgreSQL (5432) and Temporal (7233) from the remote server
ssh -L 5432:localhost:5432 -L 7233:localhost:7233 user@remote-server

# In another terminal, connect as if it's local
olake-tui \
  --db-url "postgres://olake:pass@localhost:5432/olake?sslmode=disable" \
  --temporal-host localhost:7233
```

Alternatively, run olake-tui directly on the remote server:

```bash
ssh user@remote-server
olake-tui \
  --db-url "postgres://olake:pass@db.internal:5432/olake?sslmode=disable" \
  --temporal-host temporal.internal:7233
```

### Kubernetes

If your OLake infrastructure runs in Kubernetes, use `kubectl port-forward`:

```bash
# Forward PostgreSQL and Temporal from Kubernetes services
kubectl port-forward svc/olake-postgresql 5432:5432 &
kubectl port-forward svc/olake-temporal-frontend 7233:7233 &

# Connect olake-tui
olake-tui \
  --db-url "postgres://olake:pass@localhost:5432/olake?sslmode=disable" \
  --temporal-host localhost:7233
```

For running olake-tui as a Kubernetes Job (e.g., in an init container):

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: olake-migrate
spec:
  template:
    spec:
      containers:
      - name: migrate
        image: teampaprika/olake-tui
        args:
          - --db-url=postgres://olake:pass@olake-postgresql:5432/olake?sslmode=disable
          - --temporal-host=olake-temporal-frontend:7233
          - --migrate-only
          - --admin-user=admin
          - --admin-pass=$(ADMIN_PASSWORD)
        env:
          - name: ADMIN_PASSWORD
            valueFrom:
              secretKeyRef:
                name: olake-secrets
                key: admin-password
      restartPolicy: Never
```

### Connecting to an existing OLake deployment

If you already have OLake running (with or without the web UI), olake-tui can connect to the same database:

```bash
olake-tui --db-url "postgres://USER:PASS@HOST:5432/DBNAME?sslmode=disable"
```

olake-tui reads and writes the **same schema** as the BFF server. No migration is needed if the database was already initialized by the web UI stack. Use your existing login credentials.

## Environment variables

Every CLI flag has a corresponding environment variable. Environment variables are overridden by CLI flags when both are set.

| Flag | Environment variable | Default | Description |
|------|---------------------|---------|-------------|
| `--db-url` | `OLAKE_DB_URL` | _(required)_ | PostgreSQL connection string |
| `--temporal-host` | `TEMPORAL_ADDRESS` | `localhost:7233` | Temporal frontend address |
| `--project-id` | `OLAKE_PROJECT_ID` | `123` | OLake project ID |
| `--run-mode` | `OLAKE_RUN_MODE` | `dev` | Beego run mode (`dev`/`prod`/`staging`) |
| `--encryption-key` | `OLAKE_SECRET_KEY` | _(empty)_ | AES encryption key for stored credentials |
| `--admin-user` | `OLAKE_ADMIN_USER` | `admin` | Admin username for `--migrate` seed |
| `--admin-pass` | `OLAKE_ADMIN_PASSWORD` | `admin` | Admin password for `--migrate` seed |
| `--release-url` | `OLAKE_RELEASE_URL` | _(empty)_ | URL to releases.json for update checks |

### Example .env file

Create a `.env` file in the project root (or export these in your shell profile):

```bash
# .env — olake-tui configuration
# Copy this file and fill in your values.

# Required: PostgreSQL connection string
OLAKE_DB_URL="postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable"

# Temporal frontend address (default: localhost:7233)
TEMPORAL_ADDRESS="localhost:7233"

# OLake project ID (default: 123)
# OLAKE_PROJECT_ID="123"

# Run mode affects database table naming: dev | prod | staging
# OLAKE_RUN_MODE="dev"

# AES encryption key for stored source/destination credentials.
# If empty, credentials are stored in plaintext.
# Generate one: openssl rand -hex 32
# OLAKE_SECRET_KEY=""

# Admin credentials for --migrate seed
# OLAKE_ADMIN_USER="admin"
# OLAKE_ADMIN_PASSWORD="admin"

# URL to releases.json for update notifications.
# Omit for air-gapped environments (no update checks).
# OLAKE_RELEASE_URL=""
```

Then load it before running:

```bash
# Option 1: Export manually
export $(grep -v '^#' .env | xargs)
olake-tui --migrate

# Option 2: Use direnv (auto-loads .env files)
# Install: brew install direnv
echo 'dotenv' > .envrc
direnv allow
olake-tui --migrate
```

### Run mode explained

The `--run-mode` flag (or `OLAKE_RUN_MODE`) affects how OLake names its database tables. This matches the Beego framework convention used in OLake's BFF server:

| Mode | Table prefix | Use case |
|------|-------------|----------|
| `dev` | `dev_` | Local development (default) |
| `prod` | `prod_` | Production deployments |
| `staging` | `staging_` | Staging environments |

If you're connecting to an existing OLake deployment, use the same run mode as the BFF server. If unsure, check the existing table names in PostgreSQL:

```bash
psql "$OLAKE_DB_URL" -c "\dt"
# Look for tables like dev_sources, prod_sources, etc.
```

## Troubleshooting

### Build errors

**"go: go.mod file not found"**

You're not in the project directory. Make sure you `cd olake-tui` after cloning.

**"go: cannot find module" or "go: version mismatch"**

Your Go version is too old. Check with `go version` — you need 1.22+.

```bash
# macOS
brew upgrade go

# Ubuntu/Debian
sudo snap refresh go

# Or download directly from https://go.dev/dl/
```

**"golangci-lint: command not found"**

The linter is optional and not required for building:

```bash
# Install if needed
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Connection errors

**"Error connecting to database: dial tcp localhost:5432: connect: connection refused"**

PostgreSQL is not running or not listening on port 5432.

```bash
# Check if anything is listening on 5432
lsof -i :5432

# If using Docker Compose
docker compose ps temporal-postgresql
docker compose logs temporal-postgresql --tail 10
```

**"Error connecting to database: password authentication failed for user"**

Wrong credentials in your `--db-url`. The default Docker Compose credentials are `temporal`/`temporal`:

```bash
--db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable"
```

**"Error connecting to database: SSL is not enabled on the server"**

You're connecting with `sslmode=require` (or no sslmode specified, which may default to `require` depending on your libpq version) to a server that doesn't have SSL configured.

```bash
# For local development, disable SSL:
--db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable"

# For production, configure SSL on PostgreSQL instead
```

**"Migration failed"**

The migration creates tables in the specified database. Common causes:

```bash
# 1. Database doesn't exist
psql "postgres://temporal:temporal@localhost:5432/" -c "\l"

# 2. User doesn't have CREATE TABLE permission
psql "$OLAKE_DB_URL" -c "SELECT current_user, has_database_privilege(current_database(), 'CREATE');"

# 3. Tables already exist with a different schema (rare)
# Check existing tables:
psql "$OLAKE_DB_URL" -c "\dt"
```

### Runtime errors

**"Temporal connection failed" / workflows hang**

Temporal takes longer to start than PostgreSQL. After `docker compose up`, wait 30-60 seconds:

```bash
# Check Temporal health
docker compose logs temporal --tail 20

# Verify gRPC connectivity
docker compose exec temporal temporal operator cluster health
```

**Terminal display is garbled**

```bash
# Set terminal type
export TERM=xterm-256color

# Verify your terminal supports Unicode
echo "╭──╮ ✓ ⟳ ⬡"
# Should show box drawing, checkmark, spinner, hexagon

# Minimum terminal size: 80x24
# Check current size:
tput cols; tput lines
```

**"Error running TUI: ..." immediately on start**

This usually means the terminal doesn't support alternate screen mode. Try a different terminal emulator. Recommended: iTerm2, Alacritty, kitty, WezTerm, or Windows Terminal.

### Docker-specific issues

**"docker: Error response from daemon: network host not found"**

On macOS and Windows, `--network host` doesn't work the same as Linux. Use the Docker network instead:

```bash
# macOS/Windows: use host.docker.internal
docker run -it --rm \
  teampaprika/olake-tui \
  --db-url "postgres://temporal:temporal@host.docker.internal:5432/temporal?sslmode=disable" \
  --temporal-host host.docker.internal:7233
```

**"the input device is not a TTY"**

You're missing the `-it` flags:

```bash
# Wrong:
docker run --rm teampaprika/olake-tui ...

# Correct:
docker run -it --rm teampaprika/olake-tui ...
```

For `--migrate-only` (non-interactive), `-it` is not needed.

## Next steps

- **[Quick Start]({{< relref "quick-start" >}})** — End-to-end walkthrough from zero to first sync
- **[Introduction]({{< relref "introduction" >}})** — Architecture and design rationale
- **[User Guide]({{< relref "../user-guide" >}})** — Detailed feature documentation
