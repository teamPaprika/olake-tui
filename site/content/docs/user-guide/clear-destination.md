---
title: "Clear Destination"
weight: 7
---

Clear Destination erases **all synced data** from a job's destination. It's a destructive operation that resets everything — every table, every row that the job ever wrote. The next sync after a clear performs a full refresh from scratch.

This page explains when you'd use it, exactly what happens behind the scenes, and how to recover if things go wrong.

## When You Need It

Clear Destination exists for situations where incremental sync state has become unreliable or you need a clean slate:

- **Schema changes** — You altered column types or renamed tables at the source, and the destination has stale schema artifacts
- **Bad sync** — A sync produced corrupted, duplicated, or incomplete data
- **Start fresh** — You want to reset a job completely, as if it had never run
- **Testing to production** — You used test data and now want a clean destination before going live

> **This is not "undo last sync."** Clear Destination removes *all* data ever written by the job. If you need to roll back a single sync run, you'll need to do that manually at the destination level.

## Where to Find It

Clear Destination is in the **Job Settings** screen. Navigate there from the Jobs tab:

1. Select a job with `↑`/`↓`
2. Press `S` (capital S) to open Job Settings
3. Navigate to the "Clear Destination" button

```
┌──────────────────────────────────────────────────────────┐
│  Job Settings — daily-pg-sync                            │
│                                                          │
│  Name         [daily-pg-sync                ]            │
│                                                          │
│  Schedule                                                │
│  Frequency    ╭─────────╮ ╭───────────────╮ ╭─────╮     │
│               │ Manual  │ │ Every X mins  │ │ ... │     │
│               ╰─────────╯ ╰───────────────╯ ╰─────╯     │
│  Preview: Every 30 minutes                               │
│                                                          │
│  Actions                                                 │
│    ╭──────────────╮                                      │
│    │  Pause Job   │                                      │
│    ╰──────────────╯                                      │
│    ╭─────────────────────╮                               │
│    │  Clear Destination  │  ← This one                   │
│    ╰─────────────────────╯                               │
│    ╭───────────────────────╮                             │
│    │  Recover from Clear   │                             │
│    ╰───────────────────────╯                             │
│    ╭──────────────╮                                      │
│    │  Delete Job  │                                      │
│    ╰──────────────╯                                      │
│                                                          │
│  tab/↑↓: navigate  •  enter/space: select  •  esc: back │
└──────────────────────────────────────────────────────────┘
```

Use `Tab` or `↓` to move focus to "Clear Destination" and press `Enter`.

## The Two-Step Confirmation Flow

Because this operation is irreversible, OLake TUI requires you to confirm **twice** before it actually clears anything.

### Step 1: First Warning

After pressing Enter on "Clear Destination," you see:

```
╭────────────────────────────────────────────────────────────╮
│                          ⚠                                 │
│                 Clear Destination                           │
│                                                            │
│  This will erase ALL data that was synced by this job      │
│  in the destination.                                       │
│                                                            │
│  Job: daily-pg-sync                                        │
│  This action cannot be undone.                             │
│                                                            │
│        ╭────────────╮    ╭──────────╮                      │
│        │ Clear Data │    │  Cancel  │                      │
│        ╰────────────╯    ╰──────────╯                      │
│                                                            │
│  ←→/tab: move  enter: confirm  esc: cancel                 │
╰────────────────────────────────────────────────────────────╯
```

Navigation:
- `←` / `→` or `Tab` — move between "Clear Data" and "Cancel"
- `Enter` — activate the focused button
- `Esc` — cancel and close the modal

If you select "Clear Data," the second confirmation appears.

### Step 2: Final Confirmation

```
╭────────────────────────────────────────────────────────────╮
│                          ⚠                                 │
│                  Final Confirmation                         │
│                                                            │
│  Clear data will delete ALL data in your job.              │
│  This is your last chance to cancel.                       │
│  Are you absolutely sure?                                  │
│                                                            │
│        ╭────────────╮    ╭──────────╮                      │
│        │ Clear Data │    │  Cancel  │                      │
│        ╰────────────╯    ╰──────────╯                      │
│                                                            │
│  ←→/tab: move  enter: confirm  esc: cancel                 │
╰────────────────────────────────────────────────────────────╯
```

Same navigation as Step 1. Selecting "Clear Data" here triggers the actual operation.

> **The default focus is on "Cancel"** in both modals. You have to actively move to "Clear Data" — this prevents accidental confirmation from rapid Enter presses.

## What Happens Behind the Scenes

When you confirm the clear, OLake TUI orchestrates a multi-step Temporal workflow. Here's exactly what happens:

```
You press "Clear Data" (final confirm)
        │
        ▼
┌─────────────────────────────┐
│ 1. Pause the schedule       │  Prevents a new sync from
│    (ScheduleClient.Pause)   │  starting during the clear
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│ 2. Wait for running sync    │  Up to 30 seconds for any
│    to finish (best effort)  │  in-flight sync to complete
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│ 3. Build clear-destination  │  Writes streams catalog to
│    execution request        │  /tmp/olake-config/<id>/
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│ 4. Update Temporal schedule │  Replaces the sync workflow
│    with clear-dest workflow │  with a clear-destination one
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│ 5. Trigger schedule         │  One-shot execution with
│    immediately              │  SKIP overlap policy
└──────────────┬──────────────┘
               │
               ▼
     Clear workflow runs in
     the Temporal worker...
               │
               ▼
     Worker deletes data at
     the destination (S3,
     Iceberg tables, etc.)
```

### Important Details

- **The schedule stays paused** after triggering the clear. It doesn't automatically resume. After the clear completes, the normal sync schedule needs to be restored (see Recovery below).
- **The clear runs as `RunSyncWorkflow`** but with a `clear-destination` command instead of `sync`. This means it shows up in the job's workflow history like any other run.
- **The timeout is 30 days.** Clear operations on large datasets can take hours. The Temporal workflow has a very generous timeout to avoid premature cancellation.
- **If the trigger fails**, the TUI automatically reverts the schedule back to the normal sync configuration and unpauses it. The operation is atomic from the user's perspective.

## Checking Clear Status

After triggering a clear, you want to know if it's still running. There are two ways:

### From the TUI

The TUI checks for running clear-destination workflows by querying Temporal:

```
WorkflowId = '<project>-<jobID>' AND ExecutionStatus = 'Running'
```

While a clear is in progress:
- **Stream editing is disabled** — you'll see an "Editing Disabled" modal if you try to edit the job
- **Job updates are blocked** — `UpdateJobFull` returns an error if clear-dest is running

### From Temporal UI

If you have access to the Temporal web UI (typically at `http://localhost:8080`):

1. Go to Workflows
2. Search for workflows with your job's workflow ID
3. Look for a running workflow with the `clear-destination` command

### From the Command Line

```bash
tctl workflow list --query "WorkflowId = 'olake-123-42' AND ExecutionStatus = 'Running'"
```

Replace `olake-123-42` with your actual workflow ID (format: `olake-<projectID>-<jobID>`).

## Recovery: When Clear Gets Stuck

Sometimes a clear-destination workflow gets stuck. Common causes:

- The Temporal worker crashed mid-operation
- Network issues between the worker and the destination
- The destination service is down (S3 outage, Iceberg metastore unavailable)
- The workflow hit an unhandled error and is retrying forever

### The "Recover from Clear" Button

In Job Settings, right below "Clear Destination," there's a **"Recover from Clear"** button:

```
  Actions
    ╭──────────────╮
    │  Pause Job   │
    ╰──────────────╯
    ╭─────────────────────╮
    │  Clear Destination  │
    ╰─────────────────────╯
    ╭───────────────────────╮
    │  Recover from Clear   │  ← Use this when clear is stuck
    ╰───────────────────────╯
    ╭──────────────╮
    │  Delete Job  │
    ╰──────────────╯
```

### What Recovery Does

When you press "Recover from Clear," the TUI performs three steps:

```
┌─────────────────────────────────────┐
│ 1. Cancel running workflows         │  Sends CancelWorkflow to
│    for this job                      │  Temporal for the job's
│                                      │  workflow ID
└───────────────┬─────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│ 2. Restore normal sync schedule     │  Updates the Temporal schedule
│                                      │  back to RunSyncWorkflow with
│                                      │  the standard sync command
└───────────────┬─────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│ 3. Resume the schedule              │  Unpauses with note:
│                                      │  "recovered from clear-
│                                      │   destination"
└─────────────────────────────────────┘
```

After recovery, the job is back to its normal state — the schedule is active and the next sync will run on its regular cadence.

> **Warning:** If the clear partially completed (e.g., deleted some tables but not all), you'll have an inconsistent destination. The next sync will be a full refresh and should rebuild everything, but verify your data after the first post-recovery sync.

## Constraints and Edge Cases

### Job Must Be Active

You cannot clear a destination for a paused job. If you try, you'll see:

```
Error: job is paused — unpause the job before running clear-destination
```

Why? The clear process works by manipulating the Temporal schedule (pause → update → trigger → restore). A paused job's schedule is already paused, which would conflict with the clear flow's pause/unpause logic.

**Fix:** Resume the job first (use the "Resume Job" button in Job Settings), then trigger the clear.

### No Concurrent Syncs

The clear flow waits up to 30 seconds for any running sync to finish before proceeding. If a sync is still running after 30 seconds, the clear proceeds anyway — the Temporal overlap policy is set to `SKIP`, so only one workflow runs at a time.

### Editing Is Disabled During Clear

While a clear-destination workflow is running:

- Opening Job Settings for editing shows an "Editing Disabled" modal
- `UpdateJobFull` returns: `"clear-destination is in progress, cannot update job"`
- Stream editing is blocked

This prevents you from changing the job configuration while the clear is mid-flight, which could cause inconsistencies.

## Complete Workflow: Clear and Re-sync

Here's a full example of clearing a destination and getting back to a healthy state:

```
1. Navigate to Jobs tab                                    Press: 1

2. Select the job you want to clear                        Press: ↓/↑

3. Open Job Settings                                       Press: S

4. Navigate to "Clear Destination"                         Press: Tab (×4)

5. Press Enter                                             → First modal appears

6. Move focus to "Clear Data"                              Press: ←

7. Confirm first modal                                     Press: Enter
                                                           → Second modal appears

8. Move focus to "Clear Data" again                        Press: ←

9. Final confirmation                                      Press: Enter
                                                           → Clear workflow starts

10. Wait for clear to complete                             (check Temporal or TUI)

11. After clear finishes, the schedule is still paused.
    The TUI will restore the sync schedule automatically
    on recovery, or you can trigger a manual sync.

12. Trigger a fresh sync                                   Press: Esc → s
```

## Troubleshooting

### "Job is paused — unpause the job before running clear-destination"

**What it means:** The job's Temporal schedule is in a paused state. Clear-destination needs to control the pause/unpause cycle itself, so it requires the job to start from an active state.

**What to do:**
1. Go back to Job Settings (`S` from the Jobs tab)
2. Navigate to "Resume Job" and press Enter
3. Try "Clear Destination" again

### Clear Destination Has Been Running for Hours

A clear that takes hours isn't necessarily stuck — large datasets at S3 or Iceberg destinations can genuinely take a long time. But if it's been running unreasonably long:

1. **Check the Temporal UI** for the workflow's actual status and any error messages
2. **Check the worker logs** — the Temporal worker might be retrying a failing operation
3. **Use "Recover from Clear"** if the workflow is genuinely stuck:
   - Navigate to Job Settings → "Recover from Clear" → Enter
   - This cancels the workflow, restores the sync schedule, and resumes the job
4. **If recovery fails**, manually cancel the workflow via `tctl`:
   ```bash
   tctl workflow cancel -w "olake-123-42"
   ```
   Then use "Recover from Clear" in the TUI to restore the schedule.

### Data Still Visible After Clear Completes

The clear workflow finished successfully, but you can still see data at the destination?

**Possible causes:**

- **Destination caching** — Some query engines (Trino, Spark) cache metadata. Refresh or restart the query engine.
- **Iceberg compaction delay** — Iceberg tables use snapshots. The clear removes the current data, but expired snapshots may still be visible until compaction runs.
- **S3 eventual consistency** — While rare with modern S3, listing operations may briefly show deleted objects.
- **Wrong job** — Verify you cleared the correct job. Multiple jobs can write to the same destination.

**What to do:**
1. Wait a few minutes and check again
2. Manually refresh your query engine's metadata cache
3. For Iceberg: run `CALL system.expire_snapshots('table_name')` if applicable
4. Verify the clear workflow actually completed (check Temporal for completion status)

### "clear-destination is in progress, cannot update job"

**What it means:** You're trying to edit a job while its clear-destination workflow is still running.

**What to do:**
- Wait for the clear to complete, then edit the job
- Or use "Recover from Clear" if the clear is stuck, then edit

### Temporal Client Not Connected

If you see `"temporal client not connected — set TEMPORAL_ADDRESS"`, the TUI can't reach the Temporal server. Clear Destination requires Temporal to function.

**What to do:**
1. Set the `TEMPORAL_ADDRESS` environment variable (default: `localhost:7233`)
2. Verify Temporal is running:
   ```bash
   tctl cluster health
   ```
3. Restart the TUI after setting the environment variable

## Further Reading

- [Jobs](../jobs/) — How to create and manage sync jobs
- [Monitoring](../monitoring/) — Watching sync and clear workflow progress
- [Settings](../settings/) — Webhook notifications for clear completion events
