---
title: "Installation"
weight: 3
---

# Installation

olake-tui is a single Go binary. You can build from source, use the Makefile targets, or run it as a Docker container.

## Prerequisites

### Required

- **Go 1.22+** — for building from source ([download](https://go.dev/dl/))
- **PostgreSQL 13+** — stores OLake data (sources, destinations, jobs, users)
- **Temporal Server** — orchestrates sync workflows ([docs](https://docs.temporal.io/self-hosted-guide))

### Optional

- **Docker & Docker Compose** — for running infrastructure locally
- **golangci-lint** — for running linters (`make lint`)

## Build from source

```bash
git clone https://github.com/teamPaprika/olake-tui.git
cd olake-tui
```

### Using Make (recommended)

Build the binary to `bin/olake-tui`:

```bash
make build
```

Or install directly to `$GOPATH/bin`:

```bash
make install
```

### Using go build directly

```bash
go build -ldflags "-X main.version=v0.2.0-direct" -o bin/olake-tui ./cmd/olake-tui/
```

### Using go install

```bash
go install -ldflags "-X main.version=v0.2.0-direct" ./cmd/olake-tui/
```

This places the binary in `$GOPATH/bin/olake-tui`.

### Verify the build

```bash
bin/olake-tui --help
```

## Makefile targets

The project Makefile provides these targets:

| Target | Command | Description |
|--------|---------|-------------|
| `build` | `go build ... -o bin/olake-tui` | Compile binary to `bin/` |
| `install` | `go install ...` | Install to `$GOPATH/bin` |
| `run` | `go run ./cmd/olake-tui/` | Build and run in one step |
| `test` | `go test ./...` | Run all tests (112 tests) |
| `lint` | `golangci-lint run` | Run linters |
| `clean` | `rm -rf bin/` | Remove build artifacts |

All build targets inject the version string via `-ldflags`.

## Docker image

A pre-built Docker image is available:

```bash
docker pull teampaprika/olake-tui
```

### Run with Docker

```bash
docker run -it --rm \
  --network host \
  teampaprika/olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --temporal-host localhost:7233
```

{{< callout type="warning" >}}
The `-it` flags are required — olake-tui is an interactive terminal application. The `--network host` flag lets the container reach PostgreSQL and Temporal on localhost.
{{< /callout >}}

### Run migration only (CI/init containers)

```bash
docker run --rm \
  --network host \
  teampaprika/olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --migrate-only \
  --admin-user admin \
  --admin-pass changeme
```

No `-it` needed for `--migrate-only` since it exits after migration.

## Infrastructure with Docker Compose

The repository includes a `docker-compose.yml` that runs all required infrastructure:

```bash
cd olake-tui
docker compose up -d
```

This starts:

- **PostgreSQL 13** on port `5432` (user: `temporal`, password: `temporal`)
- **Temporal Server** on port `7233` (gRPC frontend)
- **Elasticsearch** on port `9200` (Temporal visibility)
- **OLake Worker** (executes sync workflows)

Data is persisted in a Docker volume (`temporal-postgresql-data`) and a local `olake-data/` directory.

### Connect olake-tui to local infrastructure

```bash
olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --temporal-host localhost:7233 \
  --migrate \
  --admin-user admin \
  --admin-pass changeme
```

### Debug profile

To also start the Temporal Web UI (useful for inspecting workflows):

```bash
docker compose --profile debug up -d
# Temporal UI available at http://localhost:8081
```

## Kubernetes

If your OLake infrastructure runs in Kubernetes, use port-forwarding to connect:

```bash
kubectl port-forward svc/olake-postgresql 5432:5432 &
kubectl port-forward svc/olake-temporal-frontend 7233:7233 &

olake-tui \
  --db-url "postgres://olake:pass@localhost:5432/olake?sslmode=disable" \
  --temporal-host localhost:7233
```

## Connecting to an existing OLake deployment

If you already have OLake running (with or without the web UI), olake-tui can connect to the same database:

```bash
olake-tui --db-url "postgres://USER:PASS@HOST:5432/DBNAME?sslmode=disable"
```

olake-tui reads and writes the same schema as the BFF server. No migration is needed if the database was already initialized by the web UI stack. Use your existing login credentials.

## Next steps

- [Quick Start]({{< relref "quick-start" >}}) — End-to-end walkthrough
- [Introduction]({{< relref "introduction" >}}) — Architecture and design rationale
