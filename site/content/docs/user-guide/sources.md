---
title: "Sources"
weight: 3
---

Sources represent the databases and services you pull data from. OLake TUI lets you create, edit, test, and delete source connections from the **Sources** tab.

## Listing Sources

Press `2` or **Tab** to the Sources tab. All configured sources appear in a table showing name, type, and creation date. Navigate with `↑`/`↓` or `j`/`k`.

## Creating a Source

Press `a` to open the source creation form. Select a connector type, then fill in the required fields.

### Supported Connector Types

| Type | Key Fields |
|------|-----------|
| **PostgreSQL** | Host, port, database, user, password, SSL mode |
| **MySQL** | Host, port, database, user, password |
| **MongoDB** | Connection URI or host/port/auth fields |
| **Oracle** | Host, port, SID/service name, user, password |
| **MSSQL** | Host, port, database, user, password |
| **DB2** | Host, port, database, user, password |
| **Kafka** | Bootstrap servers, topic, group ID, SASL config |
| **S3** | Bucket, region, access key, secret key, prefix |

Each connector type shows only its relevant fields. Sensitive fields (passwords, secret keys) are masked during input.

### Example: PostgreSQL Source

```
Name:     production-pg
Type:     PostgreSQL
Host:     db.example.com
Port:     5432
Database: myapp
User:     readonly
Password: ********
SSL Mode: require
```

Press **Enter** to save. The source is created and you return to the list.

## Testing a Connection

Select a source and press `t` to run a connectivity test. The TUI attempts to connect using the stored credentials and reports success or failure:

```
✓ Connection successful (238ms)
```

```
✗ Connection failed: dial tcp 10.0.0.1:5432: connection refused
```

The test does not modify any data — it only verifies reachability and authentication.

## Viewing Source Details

Press **Enter** on a source to open the detail view. This shows the full configuration with sensitive values masked:

```
Name:     production-pg
Type:     PostgreSQL
Host:     db.example.com
Port:     5432
Database: myapp
User:     readonly
Password: ●●●●●●●●
SSL Mode: require
Created:  2025-01-15 09:30:00
```

## Editing a Source

Press `e` on a selected source to open the edit form. All fields are pre-filled with current values (passwords remain masked). Modify what you need and press **Enter** to save.

Editing a source does **not** affect running jobs. Changes apply to the next sync.

## Deleting a Source

Press `d` to delete the selected source. A confirmation prompt appears:

```
Delete source "production-pg"? (y/n)
```

Deletion is a **soft delete** — the record is marked as deleted but retained in the database. If the source is referenced by any active job, deletion is **blocked**:

```
Cannot delete: source is used by 2 job(s)
```

Remove or reassign the associated jobs first, then retry.

## Refreshing the List

Press `r` to reload the sources list from the database.

## Connector Documentation

For detailed connector configuration and setup guides, see the OLake docs:

- [PostgreSQL Connector](https://olake.io/docs/connectors/postgres/overview)
- [MySQL Connector](https://olake.io/docs/connectors/mysql/overview)
- [MongoDB Connector](https://olake.io/docs/connectors/mongodb/overview)
