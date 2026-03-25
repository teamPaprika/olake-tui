---
title: "Database Schema"
weight: 4
---


olake-tui stores all metadata in PostgreSQL. Running with `--migrate` creates the required tables automatically.

## Tables

The following tables are created by the migration:

### `user`

Authentication and user management.

| Column | Type | Description |
|--------|------|-------------|
| `id` | `SERIAL PRIMARY KEY` | Auto-increment ID |
| `email` | `VARCHAR(255) UNIQUE` | Login email |
| `password` | `VARCHAR(255)` | Bcrypt-hashed password |
| `name` | `VARCHAR(255)` | Display name |
| `created_at` | `TIMESTAMP` | Record creation time |
| `updated_at` | `TIMESTAMP` | Last modification time |
| `deleted_at` | `TIMESTAMP NULL` | Soft-delete marker |

### `source`

Data source configurations (MongoDB, MySQL, PostgreSQL, etc.).

| Column | Type | Description |
|--------|------|-------------|
| `id` | `SERIAL PRIMARY KEY` | Auto-increment ID |
| `name` | `VARCHAR(255)` | User-defined source name |
| `source_type` | `VARCHAR(50)` | Connector type identifier |
| `config` | `TEXT` | AES-256-GCM encrypted JSON config |
| `created_at` | `TIMESTAMP` | Record creation time |
| `updated_at` | `TIMESTAMP` | Last modification time |
| `deleted_at` | `TIMESTAMP NULL` | Soft-delete marker |

### `destination`

Data destination configurations (S3, Apache Iceberg, etc.).

| Column | Type | Description |
|--------|------|-------------|
| `id` | `SERIAL PRIMARY KEY` | Auto-increment ID |
| `name` | `VARCHAR(255)` | User-defined destination name |
| `destination_type` | `VARCHAR(50)` | Connector type identifier |
| `config` | `TEXT` | AES-256-GCM encrypted JSON config |
| `created_at` | `TIMESTAMP` | Record creation time |
| `updated_at` | `TIMESTAMP` | Last modification time |
| `deleted_at` | `TIMESTAMP NULL` | Soft-delete marker |

### `job`

Sync job definitions linking a source to a destination.

| Column | Type | Description |
|--------|------|-------------|
| `id` | `SERIAL PRIMARY KEY` | Auto-increment ID |
| `name` | `VARCHAR(255)` | User-defined job name |
| `source_id` | `INTEGER` | Foreign key → `source.id` |
| `destination_id` | `INTEGER` | Foreign key → `destination.id` |
| `config` | `TEXT` | Job-specific configuration JSON |
| `schedule` | `VARCHAR(255)` | Cron expression for scheduling (nullable) |
| `status` | `VARCHAR(50)` | Current job status |
| `created_at` | `TIMESTAMP` | Record creation time |
| `updated_at` | `TIMESTAMP` | Last modification time |
| `deleted_at` | `TIMESTAMP NULL` | Soft-delete marker |

### `project_settings`

Global project configuration (key-value pairs).

| Column | Type | Description |
|--------|------|-------------|
| `id` | `SERIAL PRIMARY KEY` | Auto-increment ID |
| `key` | `VARCHAR(255) UNIQUE` | Setting key |
| `value` | `TEXT` | Setting value |
| `created_at` | `TIMESTAMP` | Record creation time |
| `updated_at` | `TIMESTAMP` | Last modification time |

### `catalog`

Stream catalog data for jobs (discovered schemas, selected streams).

| Column | Type | Description |
|--------|------|-------------|
| `id` | `SERIAL PRIMARY KEY` | Auto-increment ID |
| `job_id` | `INTEGER` | Foreign key → `job.id` |
| `catalog_data` | `TEXT` | JSON catalog payload |
| `created_at` | `TIMESTAMP` | Record creation time |
| `updated_at` | `TIMESTAMP` | Last modification time |

### `session`

Active user sessions for TUI authentication.

| Column | Type | Description |
|--------|------|-------------|
| `id` | `SERIAL PRIMARY KEY` | Auto-increment ID |
| `user_id` | `INTEGER` | Foreign key → `user.id` |
| `token` | `VARCHAR(255)` | Session token |
| `created_at` | `TIMESTAMP` | Session start time |
| `expires_at` | `TIMESTAMP` | Session expiry time |

## Temporal Resource Naming

olake-tui creates Temporal resources using a consistent naming pattern:

```
olake-{runMode}-{entity}-{id}
```

| Resource | Pattern | Example |
|----------|---------|---------|
| Workflow ID | `olake-{runMode}-sync-{jobID}` | `olake-local-sync-42` |
| Schedule ID | `olake-{runMode}-schedule-{jobID}` | `olake-local-schedule-42` |
| Task Queue | `olake-{runMode}-queue` | `olake-local-queue` |

The `runMode` (set via `--mode` flag) isolates resources across environments, preventing conflicts between local development and production.

## Soft Delete Convention

All entity tables use a `deleted_at` column for soft deletes. A `NULL` value means the record is active; a timestamp means it has been deleted. This matches the Beego ORM convention used by the BFF server.

## Further Reading

- [OLake Architecture](https://olake.io/blog/architecture) — Platform-level architecture and data flow
- [BFF Compatibility](../bff-compatibility/) — How the schema aligns with the web UI
