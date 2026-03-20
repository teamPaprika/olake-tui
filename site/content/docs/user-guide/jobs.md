---
title: "Jobs"
weight: 5
---

Jobs are the core of OLake — they define **what** data to sync, **from where**, **to where**, and **on what schedule**. This page is a complete walkthrough: creating a job from scratch, understanding every option, managing the lifecycle, and fixing common problems.

## Prerequisites

- At least one [Source](../sources/) configured and tested
- At least one [Destination](../destinations/) configured and tested
- Temporal server and OLake worker running (required for stream discovery and syncs)

---

## Scenario: "Replicate PostgreSQL tables to Iceberg every 6 hours"

You have `production-pg` as a source and `lakehouse-iceberg` as a destination. You want to replicate the `users`, `orders`, and `products` tables with CDC, running every 6 hours.

### The Jobs Tab

Press `1` (or launch OLake TUI — Jobs is the default tab):

```
 ⬡ OLake   logged in as admin
 ╭────────────────────────────────────────────────────────────────────╮
 │ Sources: 2   Destinations: 1   Jobs: 3   Active: 2   Running: 0  │
 ╰────────────────────────────────────────────────────────────────────╯

 ╭──────╮  ╭──────╮  ╭──────╮  ╭──────╮
 │ Jobs │  │Source│  │ Dest │  │ Set  │
 ╰━━━━━━╯  ╰──────╯  ╰──────╯  ╰──────╯

 NAME              SOURCE          DESTINATION       STATUS     LAST RUN
 ──────────────────────────────────────────────────────────────────────────
 daily-pg-sync     production-pg   lakehouse-iceberg Completed  2h ago
 hourly-orders     production-pg   raw-parquet       Running    now
 mongo-backup      staging-mongo   lakehouse-iceberg Paused     3d ago

 n:new  s:sync  p:pause  d:delete  l:logs  enter:detail  ?:help
```

The dashboard stats bar at the top shows a snapshot of your pipeline health at a glance.

### Press `n` to Start the Job Creation Wizard

The wizard walks through 4 steps. You can go back at any step.

---

## Wizard Step 1: Job Name, Source & Destination

```
 ╔══════════════════════════════════════════════════════════════╗
 ║  New Job Wizard                                             ║
 ║                                                             ║
 ║  1. Config → 2. Source → 3. Destination → 4. Streams       ║
 ╚══════════════════════════════════════════════════════════════╝

 Job Name: [pg-to-iceberg-6h                 ]

 Source:
 ┌──────────────────────────────────────────┐
 │  > ● production-pg        (PostgreSQL)   │
 │      staging-pg           (PostgreSQL)   │
 │      staging-mongo        (MongoDB)      │
 └──────────────────────────────────────────┘

 Destination:
 ┌──────────────────────────────────────────┐
 │  > ● lakehouse-iceberg    (Iceberg)      │
 │      raw-parquet          (S3 Parquet)   │
 └──────────────────────────────────────────┘

 [ Next → ]

 tab:next field  ↑↓/j/k:navigate  enter:select  esc:cancel
 Step 1/4: Configure job name, source, and destination
```

**Navigation in Step 1:**
- `Tab` / `Shift+Tab` — cycle between Name, Source list, Destination list, and Next button
- `↑`/`↓` or `j`/`k` — navigate within a list
- `Enter` or `Space` — select an item in a list (shown with `●`)
- `Enter` on Next button — advance to Step 2

**Job name rules:**
- Must be unique across all jobs
- Alphanumeric, hyphens, and underscores recommended
- Used as the Temporal workflow ID prefix — keep it descriptive

**What happens with duplicate names:**
```
 ⚠ Job name "daily-pg-sync" already exists. Choose a unique name.
```

The wizard won't advance until the name is unique.

---

## Wizard Step 2: Source Review

```
 ╔══════════════════════════════════════════════════════════════╗
 ║  New Job Wizard                                             ║
 ║                                                             ║
 ║  ✓ Config → 2. Source → 3. Destination → 4. Streams        ║
 ╚══════════════════════════════════════════════════════════════╝

 Source: production-pg

 Type:       PostgreSQL
 Version:    latest
 Created by: admin

 n/→: next  p/←/b: back  esc: cancel
 Step 2/4: Review source  •  n: next  p: back
```

This is a confirmation screen. Review the source details and press `n` (or `→`) to continue.

---

## Wizard Step 3: Destination Review

```
 ╔══════════════════════════════════════════════════════════════╗
 ║  New Job Wizard                                             ║
 ║                                                             ║
 ║  ✓ Config → ✓ Source → 3. Destination → 4. Streams         ║
 ╚══════════════════════════════════════════════════════════════╝

 Destination: lakehouse-iceberg

 Type:       Apache Iceberg
 Version:    latest
 Created by: admin

 n/→: next (discover streams)  p/←/b: back  esc: cancel
 Step 3/4: Review destination  •  n: next  p: back
```

Press `n` to continue. This triggers **stream discovery** — OLake connects to the source database to list available tables.

---

## Wizard Step 4: Stream Discovery & Selection

When you advance from Step 3, OLake dispatches a discover workflow to Temporal:

```
 ╔══════════════════════════════════════════════════════════════╗
 ║  New Job Wizard                                             ║
 ║                                                             ║
 ║  ✓ Config → ✓ Source → ✓ Destination → 4. Streams          ║
 ╚══════════════════════════════════════════════════════════════╝

 Streams

 ⣾ Discovering streams from source...
```

Discovery usually takes 5–30 seconds depending on the source database size. Once complete:

```
 Streams (6 found, 0 selected)

 ┌─────────────────────────────────────────────────────────────────┐
 │                                                                 │
 │  [ ] public.users             Full Refresh                      │
 │  [ ] public.orders            Full Refresh                      │
 │  [ ] public.products          Full Refresh                      │
 │  [ ] public.order_items       Full Refresh                      │
 │  [ ] public.audit_log         Full Refresh                      │
 │  [ ] public.sessions          Full Refresh                      │
 │                                                                 │
 └─────────────────────────────────────────────────────────────────┘

 ╭─────────────────────────────────────────────────────╮
 │ Create Job  (0 streams selected)                    │
 ╰─────────────────────────────────────────────────────╯

 space:toggle  enter:configure  a:all  n:none  ctrl+enter:create  p/b:back
```

**Stream selection controls:**
| Key | Action |
|-----|--------|
| `Space` | Toggle current stream on/off |
| `Enter` | Open per-stream config popup (sync mode, cursor field) |
| `a` | Select all streams |
| `n` | Deselect all streams |
| `↑`/`↓` or `j`/`k` | Navigate the stream list |
| `Ctrl+Enter` | Create the job (at least 1 stream must be selected) |
| `p` or `b` | Go back to Step 3 |

### Configuring Individual Streams

Press `Enter` on a stream to open its configuration popup:

```
 ┌─ Configure: public.orders ──────────────────┐
 │                                             │
 │  Sync Mode:                                 │
 │  > Full Refresh                             │
 │    Full Refresh + Incremental               │
 │    Full Refresh + CDC                       │
 │    CDC Only                                 │
 │                                             │
 │  Cursor Field: [updated_at              ]   │
 │                                             │
 │  Normalize:  [✓]                            │
 │                                             │
 │         [ Confirm ]                         │
 │                                             │
 │  tab:next  ↑↓:select mode  enter:confirm    │
 └─────────────────────────────────────────────┘
```

### Sync Modes Explained

| Mode | Internal Name | What It Does | Use When |
|------|--------------|--------------|----------|
| **Full Refresh** | `full_refresh` | Reads the entire table every sync. Replaces all data in the destination. | Small reference tables, or when you need a complete snapshot every time. |
| **Full Refresh + Incremental** | `incremental` | First sync: full table scan. Subsequent syncs: only rows where `cursor_field > last_value`. | Large tables with an `updated_at` timestamp. Append-only workload. |
| **Full Refresh + CDC** | `cdc` | First sync: full table scan (snapshot). Then switches to CDC (WAL/binlog/oplog) for incremental changes. | Production tables where you need real-time-ish replication with guaranteed consistency. **Recommended for most use cases.** |
| **CDC Only** | `strict_cdc` | No initial snapshot. Only captures changes from the current WAL/binlog position forward. | When the destination already has data and you only want new changes. |

### Cursor Field

The **cursor field** tells OLake which column to use for tracking incremental progress.

- **Required for:** `incremental` mode
- **Not needed for:** `full_refresh`, `cdc`, `strict_cdc` (CDC uses the database's replication log)
- **Good cursor fields:** `updated_at`, `modified_date`, `id` (auto-increment)
- **Bad cursor fields:** non-indexed columns, columns that can be NULL

If you leave it blank for `incremental` mode, OLake will attempt to auto-detect a suitable column.

### After Selection

Select 3 streams and configure them:

```
 Streams (6 found, 3 selected)

 ┌─────────────────────────────────────────────────────────────────┐
 │                                                                 │
 │  [✓] public.users             CDC                               │
 │  [✓] public.orders            CDC           cursor: updated_at  │
 │  [✓] public.products          Full Refresh                      │
 │  [ ] public.order_items       Full Refresh                      │
 │  [ ] public.audit_log         Full Refresh                      │
 │  [ ] public.sessions          Full Refresh                      │
 │                                                                 │
 └─────────────────────────────────────────────────────────────────┘

 ╭─────────────────────────────────────────────────────╮
 │ Create Job  (3 streams selected)                    │
 ╰─────────────────────────────────────────────────────╯
```

Press `Ctrl+Enter` to create the job!

---

## Schedule Configuration

After job creation, configure the schedule via **Job Settings** (press `S` on a job from the list):

```
 ┌─ Job Settings: pg-to-iceberg-6h ─────────┐
 │                                          │
 │  Job Name: [pg-to-iceberg-6h          ]  │
 │  Schedule:  [0 */6 * * *             ]   │
 │  Activate:  [✓]                          │
 │                                          │
 │         [ Save ]                         │
 └──────────────────────────────────────────┘
```

### Cron Expression Reference

The schedule uses standard 5-field cron syntax: `minute hour day-of-month month day-of-week`

| Expression | Meaning | Syncs Per Day |
|-----------|---------|---------------|
| `0 * * * *` | Every hour, on the hour | 24 |
| `0 */6 * * *` | Every 6 hours (00:00, 06:00, 12:00, 18:00) | 4 |
| `0 0 * * *` | Daily at midnight | 1 |
| `30 2 * * *` | Daily at 2:30 AM | 1 |
| `0 0 * * 1` | Every Monday at midnight | ~0.14 |
| `0 0 1 * *` | First day of each month at midnight | ~0.03 |
| `*/15 * * * *` | Every 15 minutes | 96 |

> **No schedule?** If you leave the cron expression empty, the job is **manual-only** — you trigger syncs by pressing `s` from the Jobs tab.

Setting **Activate** to checked (`✓`) tells Temporal to start the schedule immediately. Unchecked means the schedule is created but paused.

---

## Managing Jobs After Creation

### Trigger a Manual Sync

Select a job and press `s`:
- The status changes to **Running**
- A Temporal workflow is started for the sync
- Monitor progress in the detail view or log viewer

### Cancel a Running Sync

Press `c` on a running job:
```
 Cancel sync for "pg-to-iceberg-6h"? (y/n)
```
The Temporal workflow receives a cancellation signal and stops gracefully. Partial data may have been written to the destination.

### Pause / Resume a Job

Press `p` to toggle pause:
- **Pausing:** The Temporal schedule is paused. No future cron triggers fire. Manual syncs (`s`) are still allowed.
- **Resuming:** The schedule resumes. The next sync happens at the next cron trigger.

```
 Job "pg-to-iceberg-6h" paused. Scheduled syncs will not trigger.
```

### View Job Details

Press `Enter` on a job to see full configuration and task history:

```
 ┌─ Job: pg-to-iceberg-6h ─────────────────────────────────────────┐
 │                                                                  │
 │  Source:      production-pg (PostgreSQL)                         │
 │  Destination: lakehouse-iceberg (Iceberg)                       │
 │  Schedule:    0 */6 * * * (every 6 hours)                       │
 │  Status:      Completed                                         │
 │  Streams:     3 selected                                        │
 │                                                                  │
 │  Task History:                                                   │
 │  ┌────────────────────────────────────────────────────────────┐  │
 │  │ #   Status      Started              Duration   Rows      │  │
 │  │ 5   Completed   2025-03-15 06:00:01  4m 32s     1,248,301 │  │
 │  │ 4   Completed   2025-03-15 00:00:01  4m 18s     1,201,455 │  │
 │  │ 3   Failed      2025-03-14 18:00:01  0m 12s     0         │  │
 │  │ 2   Completed   2025-03-14 12:00:02  4m 45s     1,305,112 │  │
 │  │ 1   Completed   2025-03-14 06:00:01  4m 22s     1,190,877 │  │
 │  └────────────────────────────────────────────────────────────┘  │
 │                                                                  │
 │  s:sync  c:cancel  l:logs  S:settings  p:pause  esc:back        │
 └──────────────────────────────────────────────────────────────────┘
```

### Edit a Job

From the Jobs list, press `S` (capital S) to open **Job Settings**. You can change:

| Editable | Requires Recreating Job? |
|----------|------------------------|
| Job name | No — editable in settings |
| Cron schedule | No — editable in settings |
| Activate/deactivate | No — editable in settings |
| Source | **Yes** — delete and recreate |
| Destination | **Yes** — delete and recreate |
| Stream selection | **Yes** — delete and recreate |
| Sync modes per stream | **Yes** — delete and recreate |

> **Why can't I change streams?** Stream selection and sync mode are baked into the Temporal schedule's workflow input. Changing them mid-lifecycle could cause data inconsistency (e.g., switching from CDC to full refresh would lose the replication slot position).

### Delete a Job

Press `d`:
```
 ┌─ Delete Job ─────────────────────────────┐
 │                                          │
 │  Delete job "pg-to-iceberg-6h"?          │
 │                                          │
 │  This will also remove the Temporal      │
 │  schedule and all task history.           │
 │                                          │
 │         [ Yes ]    [ No ]                │
 └──────────────────────────────────────────┘
```

**Requirements:**
- The job must not have a running sync — cancel it first with `c`
- Temporal schedule is deleted automatically
- The source and destination are **not** affected

---

## Keyboard Shortcuts Reference

| Key | Context | Action |
|-----|---------|--------|
| `1` | Any screen | Switch to Jobs tab |
| `n` | Job list | New job wizard |
| `s` | Job list / detail | Trigger sync now |
| `c` | Job list / detail | Cancel running sync |
| `p` | Job list | Pause / resume schedule |
| `d` | Job list | Delete job |
| `l` | Job list / detail | View logs for latest task |
| `S` | Job list | Open job settings (edit name, schedule) |
| `Enter` | Job list | Open job detail view |
| `r` | Job list | Refresh list from database + Temporal |
| `↑`/`k` | Job list | Move selection up |
| `↓`/`j` | Job list | Move selection down |
| `Esc` | Detail / wizard | Go back |
| `q` | Any screen | Quit OLake TUI |

---

## Troubleshooting

### Stream Discovery Fails

```
⚠ discover error: workflow execution failed
```

**Common causes:**
1. **Temporal worker is down** — Discovery runs as a Temporal workflow. Check the worker:
   ```bash
   docker compose ps olake-worker
   docker compose logs olake-worker --tail 50
   ```
2. **Source credentials are wrong** — The worker re-reads source config from the database. If credentials changed, re-test the source first.
3. **Source is unreachable from the worker** — The worker container needs network access to the source database. Test from inside the worker:
   ```bash
   docker compose exec olake-worker nc -z db.example.com 5432
   ```
4. **Timeout** — Discovery has a 10-minute timeout. Very large databases (thousands of tables) may need more time. Check worker logs for progress.

### Schedule Not Triggering

**Symptoms:** Job has a cron schedule but no syncs are happening.

**Check:**
1. Is the job **activated**? Press `S` and verify `Activate: [✓]`
2. Is the job **paused**? If status shows `Paused`, press `p` to resume
3. Is Temporal running?
   ```bash
   docker compose ps temporal
   ```
4. Is the cron expression valid? Test it at [crontab.guru](https://crontab.guru)
5. Check the Temporal Web UI at `http://localhost:8080` → Schedules

### Sync Stuck in "Running"

**Symptoms:** Job shows `Running` for an unusually long time.

**Check:**
1. Look at the worker logs for the workflow:
   ```bash
   docker compose logs olake-worker --tail 100 -f
   ```
2. Check the Temporal Web UI for the workflow execution status
3. If truly stuck, cancel with `c` and investigate the logs
4. Common cause: source table is very large and first full-refresh takes hours (this is normal for initial CDC sync)

### "Select at least 1 stream before creating the job"

You tried to create a job without selecting any streams. Go back to the stream list and use `Space` to toggle at least one stream.

### Job Creation Fails: "Job name already exists"

Each job name must be unique. Either:
- Choose a different name
- Delete the existing job with that name first

### Sync Fails Immediately

Press `l` to view logs. Common first-line errors:

| Log Error | Cause | Fix |
|-----------|-------|-----|
| `source connection failed` | Source DB credentials changed or unreachable | Re-test source, update credentials |
| `destination write failed: Access Denied` | Destination credentials/permissions changed | Re-test destination, check IAM |
| `replication slot does not exist` | PostgreSQL replication slot was dropped | The next CDC sync will recreate it |
| `workflow timeout` | Sync took longer than the configured timeout | Check for large tables, increase worker resources |

### Wizard Canceled Accidentally

If you press `Esc` during the wizard:

```
 ┌─ Cancel Wizard ──────────────────────────┐
 │                                          │
 │  Cancel job creation? All progress       │
 │  will be lost.                           │
 │                                          │
 │         [ Yes ]    [ No ]                │
 └──────────────────────────────────────────┘
```

Press `n` or `N` to go back to the wizard. Press `y` to confirm cancellation.

---

## Next Steps

With your job created and running:
1. **[Monitor syncs](../monitoring/)** — watch task history, read logs, understand status
