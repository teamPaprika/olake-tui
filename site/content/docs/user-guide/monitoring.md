---
title: "Monitoring"
weight: 6
---

OLake TUI provides real-time monitoring of sync jobs through Temporal workflow status, task history, a log viewer, and dashboard statistics.

## Real-Time Job Status

Job statuses are pulled directly from Temporal and update in near real-time on the Jobs tab:

| Status | Meaning |
|--------|---------|
| **Idle** | Job has no active workflow; waiting for next schedule or manual trigger |
| **Running** | Sync workflow is currently executing |
| **Completed** | Last sync finished successfully |
| **Failed** | Last sync encountered an error |
| **Canceled** | Last sync was manually canceled |
| **Paused** | Job schedule is paused; no syncs will trigger |

The status column refreshes automatically. Press `r` to force a refresh.

## Task History

Open a job's detail view (press **Enter**) to see its task history — a list of past and current sync runs:

```
 #   Status      Started              Duration   Rows
 5   Completed   2025-03-15 06:00:01  4m 32s     1,248,301
 4   Completed   2025-03-15 00:00:01  4m 18s     1,201,455
 3   Failed      2025-03-14 18:00:01  0m 12s     0
 2   Completed   2025-03-14 12:00:02  4m 45s     1,305,112
 1   Completed   2025-03-14 06:00:01  4m 22s     1,190,877
```

Each entry shows:

- **Run number** — sequential within the job
- **Status** — outcome of the workflow
- **Started** — timestamp when the sync began
- **Duration** — wall-clock time
- **Rows** — number of rows synced (0 for failures)

## Log Viewer

Press `l` on a job (or select a task from history) to open the log viewer. Logs are displayed with color-coded severity levels:

```
[INFO]  2025-03-15 06:00:01  Starting sync for job "daily-pg-sync"
[INFO]  2025-03-15 06:00:02  Discovered 4 streams, 3 selected
[INFO]  2025-03-15 06:00:03  Syncing public.users (full refresh)
[WARN]  2025-03-15 06:02:15  Slow query detected on public.orders (>30s)
[INFO]  2025-03-15 06:04:32  Sync completed: 1,248,301 rows in 4m 32s
```

### Log Viewer Navigation

| Key | Action |
|-----|--------|
| `↑` / `↓` | Scroll line by line |
| `PgUp` / `PgDn` | Scroll one page |
| `p` | Load older (previous) log page |
| `n` | Load newer (next) log page |
| `Esc` | Close the log viewer |

Logs are paginated — each page loads a batch of log entries. Use `p` and `n` to navigate between pages when logs exceed a single screen.

### Color Coding

| Level | Color |
|-------|-------|
| `INFO` | Default (white/gray) |
| `WARN` | Yellow |
| `ERROR` | Red |
| `DEBUG` | Dim/gray |

## Dashboard Statistics

The Jobs tab header displays aggregate statistics:

```
Jobs: 12 total | 3 running | 1 failed | 8 idle
```

This gives you an at-a-glance view of your pipeline health without opening individual jobs.

## Troubleshooting Failed Syncs

1. Select the failed job and press `l` to view logs
2. Look for `[ERROR]` entries — they contain the failure reason
3. Common causes:
   - Source database unreachable (network/firewall)
   - Authentication failure (credentials changed)
   - Temporal worker not running
   - Destination write permission denied
4. Fix the underlying issue, then press `s` to retry the sync
