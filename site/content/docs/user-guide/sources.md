---
title: "Sources"
weight: 3
---

Sources are the databases and services you replicate data **from**. This page walks through a complete scenario — from opening the Sources tab to a fully tested PostgreSQL source — then covers every supported connector type.

## Prerequisites

- OLake TUI is running and connected to the metadata database (see [Connecting](../connecting/))
- You know the connection credentials for your source database
- For **test connection** and **discover streams**: Temporal and the OLake worker must be running

---

## Scenario: "I want to replicate my PostgreSQL database to Iceberg"

This walkthrough covers the first step — configuring PostgreSQL as a source. By the end, you'll have a tested, saved source ready for job creation.

### Step 1: Open the Sources Tab

From any screen, press `2` (or `Tab` until Sources is highlighted):

```
 ⬡ OLake   logged in as admin

 ╭──────╮  ╭──────╮  ╭──────╮  ╭──────╮
 │ Jobs │  │Sources│  │ Dest │  │ Set  │
 ╰──────╯  ╰━━━━━━╯  ╰──────╯  ╰──────╯

 NAME                TYPE         CREATED
 ─────────────────────────────────────────
 (no sources configured yet)

 a:add  e:edit  d:delete  t:test  r:refresh  ?:help
```

### Step 2: Press `a` to Add a New Source

The connector type selector appears:

```
 ┌─ New Source ──────────────────────────┐
 │                                       │
 │  Name: [                           ]  │
 │                                       │
 │  Connector Type:                      │
 │  > PostgreSQL                         │
 │    MySQL                              │
 │    MongoDB                            │
 │    Oracle                             │
 │    MSSQL                              │
 │    DB2                                │
 │    Kafka                              │
 │    S3                                 │
 │                                       │
 │  ↑↓:navigate  enter:select  esc:back  │
 └───────────────────────────────────────┘
```

Type a descriptive name like `production-pg` and select **PostgreSQL**.

### Step 3: Fill in the Connection Form

After selecting PostgreSQL, the connector-specific form appears:

```
 ┌─ New Source: PostgreSQL ─────────────────┐
 │                                          │
 │  Name:     production-pg                 │
 │                                          │
 │  host:     [db.example.com            ]  │
 │  port:     [5432                      ]  │
 │  username: [readonly                  ]  │
 │  password: [••••••••                  ]  │
 │  database: [myapp                     ]  │
 │                                          │
 │  tab:next field  enter:save  esc:cancel  │
 └──────────────────────────────────────────┘
```

**Field details for PostgreSQL:**

| Field | Description | Example |
|-------|-------------|---------|
| `host` | Hostname or IP of your PostgreSQL server | `db.example.com`, `10.0.1.50` |
| `port` | PostgreSQL port (default: 5432) | `5432` |
| `username` | Database user with read access | `readonly`, `olake_repl` |
| `password` | User's password (masked during input) | `••••••••` |
| `database` | Database name to replicate from | `myapp`, `production` |

> **Tip:** For CDC (Change Data Capture), the user needs replication privileges. See [PostgreSQL CDC Setup](#postgresql-cdc-setup) below.

### Step 4: Save and Return to the List

Press `Enter` to save. The source appears in the list:

```
 NAME              TYPE         CREATED
 ─────────────────────────────────────────
 production-pg     PostgreSQL   2025-03-15 09:30
```

### Step 5: Test the Connection

Select the source and press `t`. OLake TUI dispatches a test workflow via Temporal:

**Success:**
```
 ┌─ Test Connection ───────────────────────┐
 │                                         │
 │  ✓ Connection successful (238ms)        │
 │                                         │
 │  Source: production-pg                   │
 │  Type:   PostgreSQL                     │
 │                                         │
 │           [ OK ]                        │
 └─────────────────────────────────────────┘
```

**Failure:**
```
 ┌─ Test Connection ───────────────────────┐
 │                                         │
 │  ✗ Connection failed                    │
 │                                         │
 │  dial tcp 10.0.1.50:5432: connection    │
 │  refused                                │
 │                                         │
 │           [ OK ]                        │
 └─────────────────────────────────────────┘
```

---

## Viewing Source Details

Press `Enter` on any source to see its full configuration. Sensitive values are masked:

```
 ┌─ Source: production-pg ──────────────────┐
 │                                          │
 │  Type:       PostgreSQL                  │
 │  Version:    latest                      │
 │  Created by: admin                       │
 │                                          │
 │  Configuration:                          │
 │    host:     db.example.com              │
 │    port:     5432                        │
 │    username: readonly                    │
 │    password: ●●●●●●●●                   │
 │    database: myapp                       │
 │                                          │
 │  Used by jobs:                           │
 │    • daily-pg-sync (active)              │
 │    • hourly-orders (paused)              │
 │                                          │
 │  Created:  2025-03-15 09:30:00           │
 │  Updated:  2025-03-15 09:30:00           │
 │                                          │
 │  e:edit  t:test  esc:back                │
 └──────────────────────────────────────────┘
```

The **"Used by jobs"** section shows which jobs reference this source. This matters when you want to edit or delete it.

---

## Editing a Source

Press `e` on a selected source (from the list or detail view). The form opens with all fields pre-filled:

```
 ┌─ Edit Source: production-pg ─────────────┐
 │                                          │
 │  host:     [db.example.com            ]  │
 │  port:     [5432                      ]  │
 │  username: [readonly                  ]  │
 │  password: [••••••••                  ]  │
 │  database: [myapp                     ]  │
 │                                          │
 │  tab:next field  enter:save  esc:cancel  │
 └──────────────────────────────────────────┘
```

**Important:**
- Password fields show masked placeholders. If you don't change them, the existing encrypted value is preserved.
- If the source is used by active jobs, OLake TUI shows a confirmation:
  ```
  This source is used by 2 job(s). Changes will
  affect the next sync run. Continue? (y/n)
  ```
- Changes do **not** affect running syncs — only future ones.

---

## Deleting a Source

Press `d` on a selected source. The confirmation modal appears:

```
 ┌─ Delete Source ──────────────────────────┐
 │                                          │
 │  Delete source "production-pg"?          │
 │                                          │
 │  This action cannot be undone.           │
 │                                          │
 │         [ Yes ]    [ No ]                │
 └──────────────────────────────────────────┘
```

**If the source is referenced by jobs:**
```
 ┌─ Cannot Delete ──────────────────────────┐
 │                                          │
 │  Source "production-pg" is used by       │
 │  2 job(s):                               │
 │    • daily-pg-sync                       │
 │    • hourly-orders                       │
 │                                          │
 │  Delete or reassign those jobs first.    │
 │                                          │
 │              [ OK ]                      │
 └──────────────────────────────────────────┘
```

Deletion is a **soft delete** — the record is flagged as deleted in the database but not physically removed.

---

## Supported Connector Types

### PostgreSQL

**Form fields:** `host`, `port`, `username`, `password`, `database`

**CDC setup requirements:**
1. Set `wal_level = logical` in `postgresql.conf`:
   ```sql
   ALTER SYSTEM SET wal_level = 'logical';
   -- Restart PostgreSQL after this change
   ```
2. Create a replication user with the right privileges:
   ```sql
   CREATE USER olake_repl WITH REPLICATION PASSWORD 'secret';
   GRANT USAGE ON SCHEMA public TO olake_repl;
   GRANT SELECT ON ALL TABLES IN SCHEMA public TO olake_repl;
   ```
3. Create a publication for the tables you want to replicate:
   ```sql
   CREATE PUBLICATION olake_pub FOR ALL TABLES;
   -- Or for specific tables:
   CREATE PUBLICATION olake_pub FOR TABLE users, orders, products;
   ```

The replication slot is created automatically by the OLake connector during the first CDC sync.

📖 [Full PostgreSQL Connector docs →](https://olake.io/docs/connectors/postgres/overview)

### MySQL

**Form fields:** `host`, `port`, `username`, `password`, `database`

**Binlog setup requirements:**
1. Enable binary logging in `my.cnf`:
   ```ini
   [mysqld]
   log-bin = mysql-bin
   binlog_format = ROW
   binlog_row_image = FULL
   server-id = 1
   ```
2. Grant replication privileges:
   ```sql
   GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'olake'@'%';
   GRANT SELECT ON mydb.* TO 'olake'@'%';
   FLUSH PRIVILEGES;
   ```

📖 [Full MySQL Connector docs →](https://olake.io/docs/connectors/mysql/overview)

### MongoDB

**Form fields:** `connection_string` (or `host`, `port`, `username`, `password`, `database`, `auth_source`, `replica_set`)

You can provide either:
- A full connection string: `mongodb://user:pass@host1:27017,host2:27017/mydb?replicaSet=rs0&authSource=admin`
- Or individual fields (host, port, etc.)

**Change stream requirements:**
- MongoDB must run as a **replica set** (even single-node needs `rs.initiate()`)
- The user needs the `read` role on the database and `read` on `local` for oplog access

📖 [Full MongoDB Connector docs →](https://olake.io/docs/connectors/mongodb/overview)

### Oracle

**Form fields:** `host`, `port`, `username`, `password`, `database` (SID or service name)

### MSSQL

**Form fields:** `host`, `port`, `username`, `password`, `database`

### DB2

**Form fields:** `host`, `port`, `username`, `password`, `database`

### Kafka

**Form fields:** `bootstrap_servers`, `topic`, `group_id`

Used when Kafka is the source (consuming messages as records).

### S3

**Form fields:** `bucket`, `region`, `access_key`, `secret_key`, `prefix`

Used when reading files (CSV, JSON, Parquet) from S3 as the source.

---

## Keyboard Shortcuts Reference

| Key | Context | Action |
|-----|---------|--------|
| `2` | Any screen | Switch to Sources tab |
| `a` | Source list | Add new source |
| `e` | Source list / detail | Edit selected source |
| `d` | Source list | Delete selected source |
| `t` | Source list / detail | Test connection |
| `r` | Source list | Refresh list from database |
| `Enter` | Source list | Open detail view |
| `↑`/`k` | Source list | Move selection up |
| `↓`/`j` | Source list | Move selection down |
| `Esc` | Any sub-screen | Go back |

---

## Troubleshooting

### Test Connection Fails: "connection refused"

**Applies to:** PostgreSQL, MySQL, MSSQL, Oracle, DB2

```
✗ Connection failed: dial tcp 10.0.1.50:5432: connection refused
```

**Causes:**
1. Database server is not running
2. Wrong host or port
3. Firewall blocking the connection
4. The **Temporal worker** can't reach the database (the test runs in the worker, not in TUI)

**Fix:** Remember that connection tests execute on the **OLake Temporal worker**, not on your local machine. The worker must have network access to the source database.

```bash
# Test from the worker container
docker compose exec olake-worker nc -z db.example.com 5432
```

### Test Connection Fails: "password authentication failed"

```
✗ Connection failed: pq: password authentication failed for user "olake"
```

**Cause:** Wrong credentials, or the password was encrypted with a different `ENCRYPTION_KEY`.

**Fix:**
1. Edit the source (`e`) and re-enter the password
2. Verify your `ENCRYPTION_KEY` matches the one used by the BFF (see [Connecting](../connecting/#encryption_key))

### Test Connection Fails: "SSL is not enabled on the server"

**Applies to:** PostgreSQL

**Cause:** Your source config doesn't specify an SSL mode, and the OLake worker defaults to requiring SSL.

**Fix:** For sources that don't support SSL (local dev), ensure the connector config JSON includes `"sslmode": "disable"`. You may need to edit the raw config.

### Test Connection Timeout

```
✗ Connection failed: context deadline exceeded
```

**Cause:** Network connectivity issue between the Temporal worker and the source database. The test has a 10-minute timeout.

**Fix:**
1. Verify the host is correct and reachable from the worker
2. Check for DNS resolution issues
3. Check security groups / firewall rules

### Cannot Test: "Temporal client not connected"

**Cause:** OLake TUI is not connected to Temporal, which is required to dispatch the test workflow.

**Fix:** Ensure Temporal is running and `TEMPORAL_ADDRESS` is correctly configured. See [Connecting](../connecting/).

### MongoDB: "not a replica set member"

**Cause:** MongoDB must be a replica set for change streams. Single standalone nodes don't support it.

**Fix:**
```bash
# Even for a single node, initialize as a replica set
mongosh --eval 'rs.initiate({_id: "rs0", members: [{_id: 0, host: "localhost:27017"}]})'
```

---

## Next Steps

With your source configured and tested:
1. **[Set up a Destination](../destinations/)** — configure where data lands
2. **[Create a Job](../jobs/)** — wire source → destination with stream selection
