---
title: "Connecting"
weight: 1
---

OLake TUI connects directly to a PostgreSQL metadata database and an optional Temporal server.
This page walks through every configuration parameter, how to supply them, and what to do when things go wrong.

## Prerequisites

Before you begin, make sure you have:

- **PostgreSQL 14+** running and accessible (local or remote)
- A database created for OLake metadata (e.g., `CREATE DATABASE olake;`)
- **Temporal Server** (optional but recommended) — required for sync orchestration, scheduling, and monitoring
- **OLake TUI binary** installed ([Installation Guide](../installation/))
- The OLake backend workers running if you plan to execute syncs

> **Don't have Temporal?** OLake TUI still works for managing sources, destinations, and job definitions.
> Features that require Temporal (triggering syncs, viewing task history, real-time status) will gracefully degrade.

---

## Configuration Reference

OLake TUI accepts three configuration parameters. Here's every detail you need:

### DATABASE_URL

The PostgreSQL connection string for OLake's metadata store. This is where sources, destinations, jobs, and sync history are persisted.

**Format:**
```
postgres://<user>:<password>@<host>:<port>/<database>?sslmode=<mode>
```

**Valid examples:**
```bash
DATABASE_URL="postgres://olake:mysecret@localhost:5432/olake?sslmode=disable"

# Remote server with SSL
DATABASE_URL="postgres://olake:P@ssw0rd!@db.prod.example.com:5432/olake?sslmode=require"

# Unix socket (common in Docker / local setups)
DATABASE_URL="postgres://olake:secret@/olake?host=/var/run/postgresql"

# With special characters in password (URL-encode them)
DATABASE_URL="postgres://olake:p%40ss%23word@localhost:5432/olake?sslmode=disable"
```

**Invalid examples:**
```bash
# Missing database name — will fail
DATABASE_URL="postgres://olake:secret@localhost:5432?sslmode=disable"

# MySQL URL — wrong driver entirely
DATABASE_URL="mysql://olake:secret@localhost:3306/olake"

# Missing port — may work if 5432 is the default, but be explicit
DATABASE_URL="postgres://olake:secret@localhost/olake"
```

**SSL Mode options:**
| Mode | When to Use |
|------|-------------|
| `disable` | Local development, trusted network |
| `require` | Remote server, encryption required |
| `verify-ca` | Production — verifies server certificate |
| `verify-full` | Production — verifies certificate + hostname |

### TEMPORAL_ADDRESS

The gRPC endpoint of your Temporal frontend service.

**Format:** `<host>:<port>` (no protocol prefix — this is gRPC, not HTTP)

**Valid examples:**
```bash
TEMPORAL_ADDRESS="localhost:7233"              # Local Temporal
TEMPORAL_ADDRESS="temporal.prod.internal:7233" # Remote, same VPC
TEMPORAL_ADDRESS="10.0.1.50:7233"              # IP address
```

**Invalid examples:**
```bash
# Don't add http:// — it's a gRPC address, not a URL
TEMPORAL_ADDRESS="http://localhost:7233"

# Don't use the Temporal Web UI port
TEMPORAL_ADDRESS="localhost:8080"
```

> **Default:** `localhost:7233` if not specified.

### ENCRYPTION_KEY

A 32-byte hex-encoded key used for AES-256-GCM encryption of sensitive configuration values (passwords, secret keys, tokens) stored in PostgreSQL.

**Generate one:**
```bash
openssl rand -hex 32
# Example output: a3f1b2c4d5e6f7081920abcdef1234567890abcdef1234567890abcdef123456
```

**Rules:**
- Must be exactly 64 hex characters (= 32 bytes)
- **Do not change it** after first use — all previously encrypted values become unreadable
- If you lose it, you'll need to re-enter all source/destination passwords

**Finding the BFF encryption key:** If you're connecting to an existing OLake deployment that was set up via the web UI (BFF), the encryption key is stored as the `ENCRYPTION_KEY` environment variable in the BFF container. Retrieve it with:

```bash
# Docker Compose
docker compose exec olake-bff printenv ENCRYPTION_KEY

# Kubernetes
kubectl exec deploy/olake-bff -- printenv ENCRYPTION_KEY
```

**What happens without it?**
If you omit `ENCRYPTION_KEY`, OLake TUI starts but **cannot decrypt** existing source/destination passwords. You'll see `●●●●●●●●` in detail views and connection tests will fail with authentication errors. New sources/destinations will store passwords in cleartext.

---

## Full .env File Example

Create a `.env` file in the directory where you run `olake-tui`:

```dotenv
# ─── OLake TUI Configuration ──────────────────────────────────────
#
# PostgreSQL connection string for OLake metadata storage.
# The database must already exist. Run with --migrate on first launch.
DATABASE_URL=postgres://olake:secret@localhost:5432/olake?sslmode=disable

# Temporal gRPC frontend address.
# Required for: sync execution, scheduling, real-time status, task history.
# Optional: TUI starts without it, but sync features are disabled.
TEMPORAL_ADDRESS=localhost:7233

# 32-byte hex key for AES-256-GCM encryption of sensitive fields.
# Generate with: openssl rand -hex 32
# IMPORTANT: Do not change after first use — encrypted values become unreadable.
ENCRYPTION_KEY=a3f1b2c4d5e6f7081920abcdef1234567890abcdef1234567890abcdef123456
```

---

## Precedence Rules

OLake TUI reads configuration from three sources. When the same parameter appears in multiple places, the **highest-precedence source wins**:

```
CLI flags  >  Environment variables  >  .env file
```

**Concrete example:** Suppose you have this `.env`:
```dotenv
DATABASE_URL=postgres://olake:envpass@localhost:5432/olake?sslmode=disable
TEMPORAL_ADDRESS=localhost:7233
```

And you run:
```bash
export DATABASE_URL="postgres://olake:shellpass@db.staging:5432/olake?sslmode=require"

olake-tui --db-url "postgres://olake:flagpass@db.prod:5432/olake?sslmode=verify-full"
```

**Result:** OLake TUI connects to `db.prod` with `flagpass` — the CLI flag wins over both the shell export and the `.env` file.

| Parameter | .env value | Shell export | CLI flag | **Used** |
|-----------|-----------|-------------|----------|----------|
| DATABASE_URL | `localhost/envpass` | `db.staging/shellpass` | `db.prod/flagpass` | **`db.prod/flagpass`** |
| TEMPORAL_ADDRESS | `localhost:7233` | _(not set)_ | _(not set)_ | **`localhost:7233`** |

---

## Connecting to Different Environments

### Local Development (Docker Compose)

The most common setup — everything runs on your machine:

```bash
# Start the OLake stack
docker compose up -d

# Connect TUI to local services
olake-tui \
  --db-url "postgres://olake:secret@localhost:5432/olake?sslmode=disable" \
  --temporal-address "localhost:7233" \
  --encryption-key "$(docker compose exec olake-bff printenv ENCRYPTION_KEY | tr -d '\r')"
```

### Remote Server (SSH Tunnel)

When PostgreSQL and Temporal are on a remote server without public exposure:

```bash
# Open SSH tunnels in the background
ssh -fNL 5432:localhost:5432 -L 7233:localhost:7233 user@olake-server.example.com

# Connect as if local
olake-tui \
  --db-url "postgres://olake:secret@localhost:5432/olake?sslmode=disable" \
  --temporal-address "localhost:7233"
```

### Kubernetes

Use `kubectl port-forward` to reach services inside the cluster:

```bash
# Forward PostgreSQL (adjust service names to your deployment)
kubectl port-forward svc/olake-postgres 5432:5432 &

# Forward Temporal frontend
kubectl port-forward svc/temporal-frontend 7233:7233 &

# Get the encryption key from the BFF deployment
export ENCRYPTION_KEY=$(kubectl exec deploy/olake-bff -- printenv ENCRYPTION_KEY)

# Connect
olake-tui \
  --db-url "postgres://olake:secret@localhost:5432/olake?sslmode=disable" \
  --temporal-address "localhost:7233" \
  --encryption-key "$ENCRYPTION_KEY"
```

> **Tip:** Add these port-forward commands to a shell script so you don't have to remember them each time.

---

## What You See on Startup

When OLake TUI launches successfully, it performs health checks and shows the login screen:

```
┌─────────────────────────────────────────┐
│                                         │
│            ⬡ OLake TUI                  │
│                                         │
│   Username: admin                       │
│   Password: ••••••••                    │
│                                         │
│            [ Login ]                    │
│                                         │
│   ✓ Database connected                  │
│   ✓ Temporal connected                  │
│   ✓ Encryption key loaded               │
│                                         │
└─────────────────────────────────────────┘
```

If Temporal is unreachable, you'll still see the login screen but with a warning:

```
│   ✓ Database connected                  │
│   ⚠ Temporal not connected (sync        │
│     features unavailable)               │
│   ✓ Encryption key loaded               │
```

---

## Troubleshooting

### "connection refused"

```
Error: dial tcp 127.0.0.1:5432: connect: connection refused
```

**Cause:** PostgreSQL is not running, or it's running on a different port.

**Fix:**
```bash
# Check if PostgreSQL is running
pg_isready -h localhost -p 5432

# Docker Compose
docker compose ps   # Is the postgres container running?
docker compose up -d postgres

# macOS (Homebrew)
brew services start postgresql@14
```

### "password authentication failed"

```
Error: pq: password authentication failed for user "olake"
```

**Cause:** Wrong username or password in your `DATABASE_URL`.

**Fix:**
```bash
# Verify credentials by connecting directly
psql "postgres://olake:secret@localhost:5432/olake"

# If you forgot the password, reset it
psql -U postgres -c "ALTER USER olake WITH PASSWORD 'newpassword';"
```

### "no pg_hba.conf entry for host"

```
Error: pq: no pg_hba.conf entry for host "172.17.0.1", user "olake", database "olake", SSL off
```

**Cause:** PostgreSQL's host-based authentication doesn't allow your connection. Usually happens when you use `sslmode=disable` but the server requires SSL, or when connecting from a Docker container to the host.

**Fix:**
1. Change `sslmode=disable` to `sslmode=require` in your `DATABASE_URL`
2. Or edit `pg_hba.conf` to allow the connection:
   ```
   # Allow Docker network
   host  olake  olake  172.17.0.0/16  md5
   ```
   Then `pg_ctl reload` or restart PostgreSQL.

### "database does not exist"

```
Error: pq: database "olake" does not exist
```

**Cause:** You haven't created the database yet.

**Fix:**
```bash
createdb -U postgres olake
# Then run with --migrate on first launch
olake-tui --migrate --db-url "postgres://..."
```

### "Temporal client not connected"

```
Warning: Temporal client not connected — sync features unavailable
```

**Cause:** Temporal server is not reachable at the configured address.

**What still works without Temporal:**
| Feature | Status |
|---------|--------|
| Browse/create/edit sources | ✅ Works |
| Browse/create/edit destinations | ✅ Works |
| Browse/create/edit jobs | ✅ Works |
| Test connections | ❌ Requires Temporal worker |
| Trigger syncs | ❌ Requires Temporal |
| View task history | ❌ Requires Temporal |
| Real-time job status | ❌ Shows "Unknown" |
| Scheduled syncs | ❌ Requires Temporal |

**Fix:**
```bash
# Check if Temporal is running
docker compose ps temporal

# Check the Temporal frontend port
curl -s http://localhost:8080   # Web UI — if this works, frontend is up
nc -z localhost 7233 && echo "Temporal frontend reachable"

# Restart Temporal
docker compose restart temporal temporal-worker
```

### "encryption key must be 32 bytes (64 hex characters)"

**Cause:** Your `ENCRYPTION_KEY` is the wrong length or contains non-hex characters.

**Fix:**
```bash
# Check the length
echo -n "$ENCRYPTION_KEY" | wc -c   # Should be 64

# Generate a valid key
openssl rand -hex 32
```

### "cipher: message authentication failed"

**Cause:** Your `ENCRYPTION_KEY` doesn't match the one used to encrypt the stored values. This happens when you change the key or use a different key than the BFF.

**Fix:** Use the same encryption key as the original OLake deployment. Check the BFF container:
```bash
docker compose exec olake-bff printenv ENCRYPTION_KEY
```

---

## Quick Reference

```bash
# Minimal startup (local Docker Compose)
olake-tui

# Explicit configuration
olake-tui \
  --db-url "postgres://olake:secret@localhost:5432/olake?sslmode=disable" \
  --temporal-address "localhost:7233" \
  --encryption-key "$(openssl rand -hex 32)"

# First-time setup with schema migration
olake-tui --migrate

# Using .env file (just run from the directory containing .env)
olake-tui
```

## Next Steps

Once connected, you're ready to:
1. **[Set up Sources](../sources/)** — configure your database connections
2. **[Set up Destinations](../destinations/)** — configure where data lands
3. **[Create Jobs](../jobs/)** — define sync pipelines
