---
title: "Jobs"
weight: 5
---

Jobs are the core of OLake — they define what data to sync, from where, to where, and on what schedule. The Jobs tab is the default view when you launch OLake TUI.

## Job List

The Jobs tab (`1`) shows all jobs with their name, source, destination, status, and last run time. Navigate with `↑`/`↓` or `j`/`k`.

## Creating a Job

Press `n` to start the job creation wizard. It walks through four steps:

### Step 1: Name & Source

```
Job Name:   daily-pg-sync
Source:      [production-pg     ▼]
```

Enter a unique job name. Names must be unique across all jobs — the TUI rejects duplicates. Select a source from the dropdown.

### Step 2: Destination

```
Destination: [lakehouse-iceberg  ▼]
```

Select where synced data will be written.

### Step 3: Stream Selection

After selecting source and destination, OLake runs a **discover** operation to fetch available streams (tables/collections) from the source:

```
Discovering streams... ⣾
```

Once discovery completes, you see the stream selection screen:

```
 [x] public.users          Full Refresh
 [x] public.orders         CDC         cursor: updated_at
 [ ] public.audit_log
 [x] public.products       Full Refresh
```

For each stream you can configure:

- **Selected** — toggle with **Space** to include/exclude
- **Sync Mode** — choose between `Full Refresh` and `CDC` (Change Data Capture)
- **Cursor Field** — for CDC mode, select the column used to track changes

### Step 4: Schedule

Set a cron expression for automatic syncing:

```
Schedule (cron): 0 */6 * * *
```

Common examples:

| Expression | Meaning |
|-----------|---------|
| `0 * * * *` | Every hour |
| `0 */6 * * *` | Every 6 hours |
| `0 0 * * *` | Daily at midnight |
| `0 0 * * 1` | Every Monday at midnight |

Leave empty for manual-only syncing. Press **Enter** to create the job.

## Running a Sync

Select a job and press `s` to trigger an immediate sync. The status changes to `Running` and you can monitor progress in the detail view.

## Canceling a Sync

Press `c` on a running job to cancel its current sync. The Temporal workflow receives a cancellation signal and stops gracefully.

## Pausing a Job

Press `p` to pause a scheduled job. Paused jobs skip their cron triggers until resumed. Press `p` again to resume.

## Viewing Job Details

Press **Enter** to open the job detail view showing full configuration, sync history, and current status.

## Editing a Job

From the job detail view, you can edit:

- **Job name** — must remain unique
- **Schedule (cron expression)** — change sync frequency

Source, destination, and stream selections are set at creation and cannot be changed. To modify these, delete the job and create a new one.

## Deleting a Job

Press `d` to delete a job. A confirmation prompt appears:

```
Delete job "daily-pg-sync"? (y/n)
```

Running jobs must be stopped before deletion.

## Viewing Logs

Press `l` on a job to open the log viewer for its most recent run. See [Monitoring](../monitoring/) for details on the log viewer.

## Refreshing

Press `r` to reload the job list and statuses from the database and Temporal.

## Further Reading

- [Creating Your First Pipeline](https://olake.io/docs/getting-started/creating-first-pipeline/) — step-by-step guide on the OLake website
