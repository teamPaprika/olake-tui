---
title: "BFF Compatibility"
weight: 3
---


olake-tui is designed to be **fully cross-compatible** with the existing OLake web UI and its BFF (Backend-For-Frontend) server. Data created in one interface works seamlessly in the other.

## What "Compatible" Means

You can:

- Create a source in the TUI → see it in the web UI
- Create a job in the web UI → manage it from the TUI
- Run syncs from either interface against the same Temporal server
- Switch between interfaces at any time without data migration

## Compatibility Points

### 1. Database Schema (Beego ORM)

olake-tui uses the **exact same table structure** as the BFF. Both use Beego ORM conventions:

- Same table names, column names, and types
- Same `created_at` / `updated_at` / `deleted_at` timestamp patterns
- Same soft-delete semantics (`deleted_at IS NOT NULL` marks a record as deleted)
- Same auto-increment primary keys

The TUI's `--migrate` flag creates tables identical to what the BFF's Beego migrations produce.

### 2. AES-256-GCM Encryption

Connector credentials (database passwords, API keys, etc.) are encrypted at rest using **AES-256-GCM**. Both the TUI and BFF use the same encryption format:

```
JSON-quoted Base64 string
  └── AES-256-GCM encrypted blob
        ├── 12-byte nonce (prepended)
        └── Ciphertext + 16-byte auth tag
```

The encryption key is derived from the same source in both implementations, so credentials encrypted by the BFF can be decrypted by the TUI and vice versa.

### 3. Temporal Workflow Naming

Both systems use identical naming conventions for Temporal workflows and schedules:

| Entity | Naming Pattern |
|--------|---------------|
| Workflow ID | `olake-{runMode}-sync-{jobID}` |
| Schedule ID | `olake-{runMode}-schedule-{jobID}` |
| Task Queue | `olake-{runMode}-queue` |

This means:
- Schedules created by the TUI are visible and manageable via the BFF
- Workflow history is shared — logs from either interface appear in both
- Cancelling a workflow from the TUI stops it in the web UI too

### 4. Soft Delete

Both implementations use soft deletes. When a record is "deleted":

- The `deleted_at` column is set to the current timestamp
- The record remains in the database
- Queries filter with `WHERE deleted_at IS NULL`

Records soft-deleted in the TUI are hidden in the web UI, and vice versa. No data is permanently destroyed by either interface.

### 5. Run Mode Isolation

The `runMode` parameter (e.g., `local`, `docker`) namespaces all Temporal resources. Both systems respect this namespace, ensuring that workflows from different environments don't collide.

## Known Differences

| Area | BFF | TUI |
|------|-----|-----|
| Authentication | JWT tokens over HTTP | Local DB session record |
| API layer | REST endpoints | Direct DB/Temporal calls |
| Real-time updates | WebSocket push | Polling + Bubble Tea messages |
| Deployment | Multi-container | Single binary |

These differences are in the **transport layer only** — the underlying data model is identical.

## Switching Between Interfaces

To use both interfaces against the same environment:

1. Point both at the same PostgreSQL instance
2. Point both at the same Temporal server
3. Use the same `runMode` value
4. Use the same encryption key

No migration or conversion is needed.

## Detailed Comparison

For a comprehensive field-by-field comparison of every operation, see the full comparison document:

📄 [BFF Comparison Document](https://github.com/teamPaprika/olake-tui/blob/master/docs/BFF_COMPARISON.md)

This document covers every API endpoint in the BFF and its equivalent operation in the TUI, including request/response formats, error handling, and edge cases.
