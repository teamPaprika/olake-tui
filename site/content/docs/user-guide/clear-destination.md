---
title: "Clear Destination"
weight: 7
---

The **Clear Destination** action removes all synced data from a job's destination, resetting it to a clean state. This is a destructive operation with a two-step confirmation process.

## What It Does

When you clear a destination for a job, OLake:

1. Triggers a Temporal workflow that deletes all data written by that job in the destination
2. Resets the job's sync state so the next run performs a full refresh
3. Does **not** delete the destination configuration itself — only the data

This is useful when:

- You need to re-sync from scratch after schema changes
- Test data needs to be purged before production use
- A sync produced corrupted or duplicate data

## Two-Step Confirmation

Because this operation is irreversible, OLake TUI requires two confirmations:

### Step 1: Initial Prompt

```
Clear all destination data for job "daily-pg-sync"? (y/n)
```

### Step 2: Type Confirmation

```
This will permanently delete all synced data. Type "clear" to confirm:
```

You must type the word `clear` exactly. Any other input cancels the operation.

## Status Check

After confirming, the TUI shows the workflow status:

```
Clearing destination... ⣾
```

Once complete:

```
✓ Destination cleared successfully
```

If the operation fails:

```
✗ Clear failed: timeout waiting for workflow completion
```

## Recovery from Stuck Workflows

If a clear-destination workflow gets stuck (e.g., Temporal worker crashed mid-operation), you may see the status remain in a `Running` state indefinitely.

To recover:

1. **Check Temporal** — verify the workflow status in Temporal UI or via `tctl`

   ```bash
   tctl workflow describe -w <workflow-id>
   ```

2. **Cancel the stuck workflow** — if the workflow is hung:

   ```bash
   tctl workflow cancel -w <workflow-id>
   ```

3. **Retry** — after canceling the stuck workflow, trigger the clear operation again from the TUI

4. **Manual cleanup** — if the workflow completed partially, you may need to manually remove remaining data from the destination (e.g., drop Iceberg tables or delete S3 prefixes)

## Important Notes

- You cannot clear a destination while a sync is running for that job — stop the sync first
- The clear operation runs as a separate Temporal workflow, so it appears in the job's task history
- Cleared data cannot be recovered — ensure you have backups if needed

## Further Reading

- [OLake Terminologies](https://olake.io/docs/understanding/terminologies/general) — understanding workflows, tasks, and sync state in OLake
