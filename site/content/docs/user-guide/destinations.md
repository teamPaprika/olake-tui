---
title: "Destinations"
weight: 4
---

Destinations define where OLake writes your synced data. OLake TUI supports Apache Iceberg and S3 Parquet as destination types.

## Listing Destinations

Press `3` or **Tab** to the Destinations tab. All configured destinations are listed with their name, type, and creation date. Navigate with `↑`/`↓` or `j`/`k`.

## Creating a Destination

Press `a` to open the destination creation form. Select a writer type and fill in the required fields.

### Supported Writer Types

| Type | Key Fields |
|------|-----------|
| **Apache Iceberg** | Catalog type, warehouse path, S3 bucket, region, access key, secret key, namespace |
| **S3 Parquet** | Bucket, region, access key, secret key, prefix, compression |

### Example: Iceberg Destination

```
Name:         lakehouse-iceberg
Type:         Iceberg
Catalog:      glue
Warehouse:    s3://my-lake/warehouse
S3 Bucket:    my-lake
Region:       us-east-1
Access Key:   AKIA************
Secret Key:   ********
Namespace:    analytics
```

### Example: S3 Parquet Destination

```
Name:         raw-parquet
Type:         S3 Parquet
Bucket:       my-data-lake
Region:       us-east-1
Access Key:   AKIA************
Secret Key:   ********
Prefix:       raw/
Compression:  snappy
```

Press **Enter** to save the destination.

## Testing a Connection

Select a destination and press `t` to test. OLake TUI verifies access to the target storage:

```
✓ Connection successful — bucket accessible (312ms)
```

```
✗ Connection failed: Access Denied (check IAM permissions)
```

The test writes no data — it only checks that credentials and bucket/path are valid.

## Viewing Destination Details

Press **Enter** on a destination to see the full configuration. Sensitive fields are masked:

```
Name:         lakehouse-iceberg
Type:         Iceberg
Catalog:      glue
Warehouse:    s3://my-lake/warehouse
S3 Bucket:    my-lake
Region:       us-east-1
Access Key:   AKIA****EXAMPLE
Secret Key:   ●●●●●●●●
Namespace:    analytics
Created:      2025-02-10 14:22:00
```

## Editing a Destination

Press `e` to edit the selected destination. Fields are pre-filled with current values. Modify and press **Enter** to save.

Changes take effect on the next sync run — active jobs are not interrupted.

## Deleting a Destination

Press `d` to delete. A confirmation prompt appears:

```
Delete destination "lakehouse-iceberg"? (y/n)
```

Like sources, deletion is a **soft delete**. If the destination is referenced by active jobs, deletion is blocked:

```
Cannot delete: destination is used by 1 job(s)
```

## Refreshing the List

Press `r` to reload destinations from the database.

## Writer Documentation

For detailed writer configuration, see the OLake docs:

- [Iceberg Writer](https://olake.io/docs/writers/iceberg/overview)
- [S3 Parquet Writer](https://olake.io/docs/writers/s3/overview)
