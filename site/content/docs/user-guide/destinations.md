---
title: "Destinations"
weight: 4
---

Destinations define where OLake writes your synced data. OLake TUI supports two destination types: **Apache Iceberg** and **Amazon S3 (Parquet)**. This page walks through a complete setup scenario, explains every configuration field, and covers common issues.

## Prerequisites

- OLake TUI is running and connected (see [Connecting](../connecting/))
- An S3 bucket (or compatible object storage) with write access
- For Apache Iceberg: a catalog service (Hive Metastore, AWS Glue, or REST catalog)
- For **test connection**: Temporal and the OLake worker must be running

---

## Scenario: "Setting up Apache Iceberg with Glue Catalog"

You have an AWS account, an S3 bucket called `my-data-lake`, and want to use AWS Glue as the Iceberg catalog. Here's the full walkthrough.

### Step 1: Open the Destinations Tab

Press `3` (or `Tab` to navigate):

```
 ⬡ OLake   logged in as admin

 ╭──────╮  ╭──────╮  ╭──────╮  ╭──────╮
 │ Jobs │  │Source│  │ Dest │  │ Set  │
 ╰──────╯  ╰──────╯  ╰━━━━━━╯  ╰──────╯

 NAME                TYPE            CREATED
 ─────────────────────────────────────────────
 (no destinations configured yet)

 a:add  e:edit  d:delete  t:test  r:refresh  ?:help
```

### Step 2: Press `a` to Add a New Destination

The writer type selector appears:

```
 ┌─ New Destination ────────────────────────┐
 │                                          │
 │  Name: [                              ]  │
 │                                          │
 │  Writer Type:                            │
 │  > Apache Iceberg                        │
 │    Amazon S3 (Parquet)                   │
 │                                          │
 │  ↑↓:navigate  enter:select  esc:back    │
 └──────────────────────────────────────────┘
```

Type a name like `lakehouse-iceberg` and select **Apache Iceberg**.

### Step 3: Fill in the Iceberg Configuration Form

```
 ┌─ New Destination: Apache Iceberg ────────┐
 │                                          │
 │  Name:         lakehouse-iceberg         │
 │                                          │
 │  catalog_type: [glue                  ]  │
 │  warehouse:    [s3://my-data-lake/wh  ]  │
 │  uri:          [                      ]  │
 │  credentials:  [{"aws_access_key_id": ]  │
 │                                          │
 │  tab:next field  enter:save  esc:cancel  │
 └──────────────────────────────────────────┘
```

**Field-by-field breakdown:**

#### `catalog_type`

The Iceberg catalog backend. This determines how table metadata is managed.

| Value | Description | When to Use |
|-------|-------------|-------------|
| `glue` | AWS Glue Data Catalog | AWS-native; Athena/Redshift Spectrum queries |
| `hive` | Hive Metastore (Thrift) | On-prem Hadoop; existing Hive infrastructure |
| `rest` | REST Catalog (Iceberg spec) | Tabular, Nessie, Polaris, or custom catalog |

#### `warehouse`

The S3 (or S3-compatible) path where Iceberg data and metadata files are stored.

**Format:** `s3://<bucket>/<prefix>/`

**Examples:**
```
s3://my-data-lake/warehouse          # AWS S3
s3://my-data-lake/iceberg/analytics  # With subdirectory prefix
s3a://minio-bucket/warehouse         # MinIO / S3-compatible
```

> **Important:** The OLake worker process (not the TUI) writes to this path. The IAM role or access keys must have `s3:PutObject`, `s3:GetObject`, `s3:ListBucket`, and `s3:DeleteObject` permissions on this prefix.

#### `uri`

The catalog service endpoint. What goes here depends on `catalog_type`:

| Catalog Type | URI Value | Example |
|-------------|-----------|---------|
| `glue` | Leave empty or set to the Glue endpoint | _(empty)_ |
| `hive` | Thrift URI of the Hive Metastore | `thrift://hive-metastore:9083` |
| `rest` | HTTP(S) URL of the REST catalog | `https://catalog.example.com/api/v1` |

For **Glue**, the URI is typically not needed — the SDK uses the AWS region and credentials to find the Glue service automatically.

#### `credentials`

A JSON string with authentication credentials. The exact fields depend on your catalog and storage:

**For AWS (Glue + S3):**
```json
{
  "aws_access_key_id": "AKIAIOSFODNN7EXAMPLE",
  "aws_secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
  "aws_region": "us-east-1"
}
```

**For Hive + S3:**
```json
{
  "aws_access_key_id": "AKIA...",
  "aws_secret_access_key": "...",
  "aws_region": "us-east-1"
}
```

**For MinIO:**
```json
{
  "aws_access_key_id": "minioadmin",
  "aws_secret_access_key": "minioadmin",
  "aws_region": "us-east-1",
  "s3_endpoint": "http://minio:9000",
  "s3_path_style_access": "true"
}
```

> **Tip:** The credentials field is encrypted at rest using the `ENCRYPTION_KEY`. It's safe to paste secrets here.

### Step 4: Save the Destination

Press `Enter`. The destination appears in the list:

```
 NAME                TYPE            CREATED
 ─────────────────────────────────────────────
 lakehouse-iceberg   Apache Iceberg  2025-03-15 14:22
```

### Step 5: Test the Connection

Select the destination and press `t`:

**Success:**
```
 ┌─ Test Connection ───────────────────────┐
 │                                         │
 │  ✓ Connection successful (1.2s)         │
 │                                         │
 │  Destination: lakehouse-iceberg         │
 │  Type:        Apache Iceberg            │
 │                                         │
 │           [ OK ]                        │
 └─────────────────────────────────────────┘
```

The test verifies:
- S3 bucket is accessible with the provided credentials
- Catalog service is reachable (for Hive/REST)
- Write permissions exist on the warehouse path

**Failure:**
```
 ┌─ Test Connection ───────────────────────┐
 │                                         │
 │  ✗ Connection failed                    │
 │                                         │
 │  Access Denied: User arn:aws:iam::...   │
 │  is not authorized to perform            │
 │  s3:PutObject on resource               │
 │  "s3://my-data-lake/warehouse"          │
 │                                         │
 │           [ OK ]                        │
 └─────────────────────────────────────────┘
```

---

## Setting Up S3 Parquet

For simpler use cases where you just want raw Parquet files in S3 (no catalog, no table management):

```
 ┌─ New Destination: Amazon S3 (Parquet) ───┐
 │                                          │
 │  Name:         raw-parquet               │
 │                                          │
 │  storage_type: [s3                    ]  │
 │  path:         [s3://my-bucket/raw/   ]  │
 │  credentials:  [{"aws_access_key_id": ]  │
 │                                          │
 │  tab:next field  enter:save  esc:cancel  │
 └──────────────────────────────────────────┘
```

**Fields:**

| Field | Description | Example |
|-------|-------------|---------|
| `storage_type` | `s3` for Amazon S3, `local` for local filesystem | `s3` |
| `path` | S3 URI or local path where Parquet files are written | `s3://my-bucket/raw/` |
| `credentials` | JSON with AWS credentials (same format as Iceberg) | `{"aws_access_key_id": "..."}` |

**When to use S3 Parquet vs Iceberg:**

| Feature | Iceberg | S3 Parquet |
|---------|---------|------------|
| Schema evolution | ✅ Automatic | ❌ Manual |
| ACID transactions | ✅ | ❌ |
| Time travel queries | ✅ | ❌ |
| Partition evolution | ✅ | ❌ |
| Query engine support | Athena, Spark, Trino, Flink | Any Parquet reader |
| Setup complexity | Higher (needs catalog) | Lower (just S3) |
| Best for | Production data lakehouse | Quick exports, staging |

---

## Viewing Destination Details

Press `Enter` on a destination:

```
 ┌─ Destination: lakehouse-iceberg ─────────┐
 │                                          │
 │  Type:       Apache Iceberg              │
 │  Version:    latest                      │
 │  Created by: admin                       │
 │                                          │
 │  Configuration:                          │
 │    catalog_type: glue                    │
 │    warehouse:    s3://my-data-lake/wh    │
 │    uri:          (empty)                 │
 │    credentials:  ●●●●●●●●               │
 │                                          │
 │  Used by jobs:                           │
 │    • daily-pg-sync (active)              │
 │                                          │
 │  Created:  2025-03-15 14:22:00           │
 │  Updated:  2025-03-15 14:22:00           │
 │                                          │
 │  e:edit  t:test  esc:back                │
 └──────────────────────────────────────────┘
```

---

## Editing a Destination

Press `e` on a selected destination. The form opens pre-filled. Same behavior as source editing:

- Credential fields show masked placeholders
- If referenced by active jobs, a confirmation prompt appears
- Changes apply to the **next sync run**, not the current one

---

## Deleting a Destination

Press `d` to delete. Deletion is blocked if any job references this destination:

```
 ┌─ Cannot Delete ──────────────────────────┐
 │                                          │
 │  Destination "lakehouse-iceberg" is      │
 │  used by 1 job(s):                       │
 │    • daily-pg-sync                       │
 │                                          │
 │  Delete or reassign those jobs first.    │
 │                                          │
 │              [ OK ]                      │
 └──────────────────────────────────────────┘
```

Like sources, deletion is a **soft delete**.

---

## Keyboard Shortcuts Reference

| Key | Context | Action |
|-----|---------|--------|
| `3` | Any screen | Switch to Destinations tab |
| `a` | Destination list | Add new destination |
| `e` | Destination list / detail | Edit selected destination |
| `d` | Destination list | Delete selected destination |
| `t` | Destination list / detail | Test connection |
| `r` | Destination list | Refresh list from database |
| `Enter` | Destination list | Open detail view |
| `↑`/`k` | Destination list | Move selection up |
| `↓`/`j` | Destination list | Move selection down |
| `Esc` | Any sub-screen | Go back |

---

## Troubleshooting

### "Access Denied" on S3

```
✗ Connection failed: Access Denied
```

**Cause:** The IAM credentials don't have sufficient permissions on the S3 bucket.

**Required IAM policy:**
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
      "s3:ListBucket"
    ],
    "Resource": [
      "arn:aws:s3:::my-data-lake",
      "arn:aws:s3:::my-data-lake/*"
    ]
  }]
}
```

> **Common mistake:** Granting permissions on `arn:aws:s3:::my-data-lake/*` but forgetting `arn:aws:s3:::my-data-lake` (without `/*`). You need both — the first for `ListBucket`, the second for object operations.

For **Glue catalog**, you also need:
```json
{
  "Effect": "Allow",
  "Action": [
    "glue:GetDatabase",
    "glue:CreateDatabase",
    "glue:GetTable",
    "glue:CreateTable",
    "glue:UpdateTable",
    "glue:GetPartitions"
  ],
  "Resource": "*"
}
```

### "No such bucket"

```
✗ Connection failed: The specified bucket does not exist
```

**Cause:** Typo in the bucket name, or the bucket is in a different AWS account/region.

**Fix:**
```bash
aws s3 ls s3://my-data-lake/ --region us-east-1
```

### Hive Metastore Unreachable

```
✗ Connection failed: dial tcp hive-metastore:9083: connection refused
```

**Cause:** The Hive Metastore Thrift service is not running or not accessible from the OLake worker.

**Fix:**
1. Verify the URI: `thrift://hive-metastore:9083` (not `http://`)
2. Check from the worker container:
   ```bash
   docker compose exec olake-worker nc -z hive-metastore 9083
   ```
3. Ensure the Hive Metastore service is running and configured for the Iceberg catalog

### Credential Parsing Error

```
✗ Connection failed: invalid credentials JSON
```

**Cause:** The `credentials` field is not valid JSON.

**Fix:** Make sure credentials are valid JSON. Common mistakes:
```json
// ❌ Wrong — single quotes
{'aws_access_key_id': 'AKIA...'}

// ❌ Wrong — trailing comma
{"aws_access_key_id": "AKIA...", }

// ✅ Correct
{"aws_access_key_id": "AKIA...", "aws_secret_access_key": "..."}
```

### Cannot Test: Worker Not Running

Connection tests run on the **OLake Temporal worker**, not inside the TUI. If the worker container is down, tests will hang and eventually timeout.

```bash
# Check worker status
docker compose ps olake-worker

# Restart if needed
docker compose restart olake-worker
```

---

## Further Reading

- [Iceberg Writer Reference](https://olake.io/docs/writers/iceberg/overview) — full config reference
- [S3 Parquet Writer Reference](https://olake.io/docs/writers/s3/overview) — full config reference

## Next Steps

With source and destination configured:
1. **[Create a Job](../jobs/)** — wire them together with stream selection and scheduling
