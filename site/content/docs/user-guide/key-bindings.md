---
title: "Key Bindings"
weight: 9
---

OLake TUI is fully keyboard-driven. Every action — from navigating tabs to configuring jobs — is a keypress away. This page documents every key binding, organized by the screen where it applies.

## Global Keys

These work everywhere in the application, regardless of which screen or modal is active.

| Key | Action | Details |
|-----|--------|---------|
| `1` | Jobs tab | Jump directly to the Jobs list |
| `2` | Sources tab | Jump to Sources list |
| `3` | Destinations tab | Jump to Destinations list |
| `4` | Settings tab | Open system-wide settings |
| `Tab` | Next tab | Cycle through tabs left to right |
| `Shift+Tab` | Previous tab | Cycle through tabs right to left |
| `q` | Quit | Exit the application gracefully |
| `Ctrl+C` | Force quit | Exit immediately, no confirmation |
| `Esc` | Go back | Close current view, modal, or nested screen |
| `?` | Help | Show contextual help for the current screen |

### How Tab Navigation Works

The tab bar at the top of the screen shows all available sections:

```
┌──────────────────────────────────────────────────────────────────┐
│  Jobs    Sources    Destinations    Settings                     │
│  [1]     [2]        [3]             [4]                          │
│ ═══════                                                          │
│                                                                  │
│  (current tab content here)                                      │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

The underline (`═══`) indicates the active tab. Number keys jump directly; `Tab` cycles sequentially.

> **Tip:** Number keys are the fastest way to switch tabs. If you're on the Destinations tab and need to start a sync, press `1` to jump to Jobs — faster than pressing `Tab` twice.

## Jobs Tab

The Jobs tab is where you spend most of your time. It shows all configured sync jobs and their current status.

```
┌──────────────────────────────────────────────────────────────────┐
│  Jobs    Sources    Destinations    Settings                     │
│ ═══════                                                          │
│                                                                  │
│  ▶ daily-pg-sync        Running     ⣾  2m ago      [s] sync     │
│    weekly-backup         Idle           6h ago      [l] logs     │
│    staging-mirror        Paused         1d ago      [S] settings │
│    dev-test-job          Failed         3h ago      [d] delete   │
│                                                                  │
│  ↑↓/jk: move  enter: detail  s: sync  S: settings  l: logs     │
└──────────────────────────────────────────────────────────────────┘
```

### Job Actions

| Key | Action | What It Does |
|-----|--------|-------------|
| `↑` / `k` | Move up | Select the previous job in the list |
| `↓` / `j` | Move down | Select the next job in the list |
| `Enter` | Job detail | Open the full detail view for the selected job |
| `n` | New job | Launch the job creation wizard |
| `s` | Sync now | Trigger an immediate sync for the selected job |
| `c` | Cancel sync | Cancel the currently running sync |
| `p` | Pause/Resume | Toggle the job's scheduled sync on/off |
| `l` | Logs | Open the log viewer for the selected job |
| `S` | Job Settings | Open the job settings screen (name, frequency, clear, delete) |
| `d` | Delete | Delete the selected job (with confirmation modal) |
| `r` | Refresh | Reload the job list from the database |
| `u` | Updates | Open the updates modal (check for new versions) |

### What Each Action Opens

- **`Enter` (Job detail)** — A read-only view showing the job's source, destination, stream configuration, recent run history, and current status. Press `Esc` to return.
- **`S` (Job Settings)** — An editable screen where you can rename the job, change its schedule frequency, pause/resume, clear destination, recover from clear, or delete. See [Settings](../settings/) for the system settings (press `4`).
- **`l` (Logs)** — The log viewer showing sync output. See the [Log Viewer](#log-viewer) section below.
- **`n` (New job)** — The multi-step job creation wizard. See [Job Wizard](#job-creation-wizard) below.

> **`s` vs `S`:** Lowercase `s` triggers a sync. Capital `S` (Shift+S) opens job settings. This is intentional — sync is the most common action, so it gets the simpler keystroke.

## Sources Tab

The Sources tab lists all configured data sources (PostgreSQL, MySQL, MongoDB, etc.).

```
┌──────────────────────────────────────────────────────────────────┐
│  Jobs    Sources    Destinations    Settings                     │
│          ═══════                                                 │
│                                                                  │
│  ▶ production-pg       PostgreSQL    ✓ Connected                 │
│    staging-mysql        MySQL         ✓ Connected                │
│    analytics-mongo      MongoDB       ✗ Error                    │
│                                                                  │
│  ↑↓/jk: move  a: add  e: edit  t: test  d: delete  r: refresh  │
└──────────────────────────────────────────────────────────────────┘
```

### Source Actions

| Key | Action | What It Does |
|-----|--------|-------------|
| `↑` / `k` | Move up | Select the previous source |
| `↓` / `j` | Move down | Select the next source |
| `Enter` | View detail | Open the source detail view (connection config, associated jobs) |
| `a` | Add source | Launch the source creation form |
| `e` | Edit source | Edit the selected source's configuration |
| `d` | Delete source | Delete the source (shows warning if jobs depend on it) |
| `t` | Test connection | Test the connection to the selected source |
| `r` | Refresh | Reload the source list from the database |

### Connection Test Flow

When you press `t`, the TUI shows a spinner modal:

```
╭──────────────────────────────────────╮
│       Testing Connection             │
│                                      │
│       ⣾  Testing your connection…    │
│                                      │
│       Please wait…                   │
╰──────────────────────────────────────╯
```

On success (auto-dismisses after 1 second):

```
╭──────────────────────────────────────╮
│               ✓                      │
│       Connection Successful          │
│                                      │
│  Your connection has been verified.  │
│       Continuing automatically…      │
╰──────────────────────────────────────╯
```

On failure (stays open, shows error details):

```
╭──────────────────────────────────────╮
│               ✗                      │
│       Connection Failed              │
│                                      │
│  Error: dial tcp: connection refused │
│                                      │
│  x: show logs  (12 lines)           │
│                                      │
│  ╭──────╮    ╭──────╮               │
│  │ Back │    │ Edit │               │
│  ╰──────╯    ╰──────╯               │
│                                      │
│  ←→/tab: move  enter: select        │
╰──────────────────────────────────────╯
```

Press `x` to expand/collapse the error log. Select "Edit" to go back to the source form and fix the issue.

## Destinations Tab

The Destinations tab lists all configured data destinations (S3, Apache Iceberg, etc.).

```
┌──────────────────────────────────────────────────────────────────┐
│  Jobs    Sources    Destinations    Settings                     │
│                     ═══════════════                              │
│                                                                  │
│  ▶ data-lake-s3        S3/Iceberg    ✓ Connected                │
│    backup-s3            S3            ✓ Connected                │
│                                                                  │
│  ↑↓/jk: move  a: add  e: edit  t: test  d: delete  r: refresh  │
└──────────────────────────────────────────────────────────────────┘
```

### Destination Actions

| Key | Action | What It Does |
|-----|--------|-------------|
| `↑` / `k` | Move up | Select the previous destination |
| `↓` / `j` | Move down | Select the next destination |
| `Enter` | View detail | Open the destination detail view |
| `a` | Add destination | Launch the destination creation form |
| `e` | Edit destination | Edit the selected destination's configuration |
| `d` | Delete destination | Delete the destination (with job impact warning) |
| `t` | Test connection | Test connectivity to the destination |
| `r` | Refresh | Reload the destination list from the database |

The keyboard layout is identical to the Sources tab — once you learn one, you know the other.

## Log Viewer

The log viewer shows sync output for a specific job. Open it by pressing `l` on the Jobs tab.

```
┌──────────────────────────────────────────────────────────────────┐
│  Log Viewer — daily-pg-sync                                      │
│ ─────────────────────────────────────────────────────────────    │
│  2025-03-15 06:00:01 [INFO]  Starting sync...                   │
│  2025-03-15 06:00:02 [INFO]  Connected to source (PostgreSQL)   │
│  2025-03-15 06:00:03 [INFO]  Discovering schema...              │
│  2025-03-15 06:00:05 [INFO]  Found 12 tables, 3 selected        │
│  2025-03-15 06:00:06 [INFO]  Syncing table: users (42,301 rows) │
│  2025-03-15 06:01:12 [INFO]  Syncing table: orders (892,100)    │
│  2025-03-15 06:03:44 [INFO]  Syncing table: products (13,900)   │
│  2025-03-15 06:04:32 [INFO]  Sync completed successfully        │
│                                                                  │
│  ↑↓: scroll  PgUp/PgDn: page  p: older  n: newer  esc: close   │
└──────────────────────────────────────────────────────────────────┘
```

### Log Navigation

| Key | Action | Details |
|-----|--------|---------|
| `↑` | Scroll up | Move up one line |
| `↓` | Scroll down | Move down one line |
| `PgUp` | Page up | Scroll up one full screen height |
| `PgDn` | Page down | Scroll down one full screen height |
| `p` | Previous page | Load older log entries from the server |
| `n` | Next page | Load newer log entries from the server |
| `Esc` | Close | Return to the Jobs tab |

### Scrolling vs Pagination

There's an important distinction:

- **`↑`/`↓` and `PgUp`/`PgDn`** scroll through the logs already loaded in memory. This is instant — no network calls.
- **`p` (previous) and `n` (next)** fetch a new batch of log entries from the server. This is for when you need to go further back in history than what's currently loaded.

Think of it as: arrow keys scroll the current page, `p`/`n` flip between pages of history.

## Modal Dialogs

Modals are overlay windows that appear on top of the current screen. They're used for confirmations, connection tests, and entity management. All modals share a consistent navigation pattern.

### Modal Navigation Pattern

```
╭────────────────────────────────────────╮
│              Modal Title               │
│                                        │
│  Some message or warning text here.    │
│                                        │
│    ╭───────────╮    ╭──────────╮       │
│    │  Confirm  │    │  Cancel  │       │
│    ╰───────────╯    ╰──────────╯       │
│                                        │
│  ←→/tab: move  enter: select  esc: X  │
╰────────────────────────────────────────╯
```

| Key | Action |
|-----|--------|
| `←` / `h` | Move focus to the left button |
| `→` / `l` | Move focus to the right button |
| `Tab` | Cycle between buttons |
| `Enter` | Activate the focused button |
| `Esc` | Cancel / close the modal |
| `y` | Confirm (on yes/no prompts) |
| `n` | Decline (on yes/no prompts) |

### Confirmation Modals

Delete and destructive actions show confirmation modals. The **default focus is always on the safer option** (Cancel/No):

```
╭────────────────────────────────────────╮
│                ⚠                       │
│           Delete Job                   │
│                                        │
│  Are you sure you want to delete       │
│  job 'daily-pg-sync'?                  │
│  This will remove all run history      │
│  and cannot be undone.                 │
│                                        │
│    ╭──────────╮    ╭──────────╮        │
│    │  Delete  │    │ >Cancel< │        │
│    ╰──────────╯    ╰──────────╯        │
│                                        │
│  y/n: choose  ←→/tab: move  esc: X    │
╰────────────────────────────────────────╯
```

The `>Cancel<` styling indicates which button has focus. Press `←` to move focus to "Delete," then `Enter` to confirm. Or just press `y` as a shortcut.

## Job Creation Wizard

Creating a new job is a multi-step wizard. Press `n` on the Jobs tab to start.

### Wizard Steps

```
Step 1: Select Source       →  Step 2: Select Destination
           ↓                              ↓
Step 3: Configure Streams   →  Step 4: Job Name & Schedule
           ↓
Step 5: Review & Save
```

### Per-Step Keys

**Step 1 & 2 (Select Source / Destination):**

| Key | Action |
|-----|--------|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `Enter` | Select and proceed to next step |
| `Esc` | Cancel wizard (with confirmation) |

**Step 3 (Configure Streams):**

| Key | Action |
|-----|--------|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `Space` | Toggle stream selection (checkbox) |
| `Enter` | Proceed to next step |
| `Esc` | Go back (with "lose changes?" warning) |

**Step 4 (Job Name & Schedule):**

| Key | Action |
|-----|--------|
| `Tab` / `↓` | Next field |
| `Shift+Tab` / `↑` | Previous field |
| `←` / `→` | Cycle frequency mode (on frequency selector) |
| `Enter` | Proceed to review |
| `Esc` | Go back |

**Step 5 (Review & Save):**

| Key | Action |
|-----|--------|
| `Enter` | Save the job |
| `Esc` | Go back to edit |

### Stream Selection Detail

In step 3, streams are listed with checkboxes:

```
┌──────────────────────────────────────────────────────────────────┐
│  Select Streams — production-pg                                  │
│                                                                  │
│  [✓] public.users                                               │
│  [✓] public.orders                                              │
│  [ ] public.logs              ← unchecked, won't sync           │
│  [✓] public.products                                            │
│  [ ] public.sessions                                            │
│                                                                  │
│  ↑↓: move  space: toggle  enter: continue  esc: back            │
└──────────────────────────────────────────────────────────────────┘
```

Press `Space` to toggle individual streams. Only checked streams will be included in the job.

## Job Settings Screen

The job settings screen (opened with `S` from the Jobs tab) has its own navigation:

| Key | Action |
|-----|--------|
| `Tab` / `↓` | Next field/button |
| `Shift+Tab` / `↑` | Previous field/button |
| `←` / `→` / `h` / `l` | Cycle frequency options (when on frequency selector) |
| `Enter` / `Space` | Activate the focused button or field |
| `Esc` | Cancel and return to Jobs tab |

The focus order is:
1. Name input → 2. Frequency selector → 3. Frequency-specific inputs (minutes/hours/time/cron) → 4. Pause/Resume button → 5. Clear Destination → 6. Recover from Clear → 7. Delete Job → 8. Save → 9. Cancel

## Keyboard Workflow Recipes

Here are optimized keystroke sequences for common tasks:

### Fastest Way to Trigger a Sync

```
1 → ↓ (select job) → s
```

That's it: switch to Jobs tab, select a job, trigger sync. Three keystrokes.

### Quick Connection Test (Source)

```
2 → ↓ (select source) → t
```

Switch to Sources tab, select a source, test it.

### Quick Connection Test (Destination)

```
3 → ↓ (select destination) → t
```

Same pattern, different tab.

### Create a Complete Job from Scratch

```
1 → n → ↓ (select source) → Enter → ↓ (select dest) → Enter
→ Space (toggle streams) → Enter → (type name) → Tab (set schedule) → Enter
→ Enter (save)
```

### Check Logs for the Last Sync

```
1 → ↓ (select job) → l → PgDn (scroll to end)
```

### Change a Job's Schedule

```
1 → ↓ (select job) → S → Tab → ←/→ (pick frequency) → Tab → (enter value) → Tab (×N) → Enter (save)
```

### Navigate Between Tabs Without a Mouse

You never need a mouse. The full tab cycle:

```
1 = Jobs  →  2 = Sources  →  3 = Destinations  →  4 = Settings
```

Or use `Tab` to cycle forward, `Shift+Tab` to cycle backward. Within any list, `↑`/`↓` or `j`/`k` to navigate items.

### Delete a Job Safely

```
1 → ↓ (select job) → S → Tab (×7, to Delete) → Enter → ← (focus Delete) → Enter
```

The double confirmation prevents accidental deletion.

## Vim-Style Navigation

Throughout the TUI, `j` and `k` work as alternatives to `↓` and `↑` in all list views. Similarly, `h` and `l` work as `←` and `→` in modals and the frequency selector.

This isn't full Vim emulation — there's no `:commands`, no visual mode, no `/search`. But the basic hjkl movement works everywhere you'd expect arrow keys to work.

## Summary: Quick Reference Card

```
┌─────────────────────────────────────────────────────────────┐
│  GLOBAL           │  JOBS TAB        │  SOURCES/DEST TAB   │
│  1-4: switch tab  │  s: sync         │  a: add             │
│  Tab: next tab    │  c: cancel       │  e: edit            │
│  Esc: back        │  p: pause        │  d: delete          │
│  q: quit          │  l: logs         │  t: test            │
│  ?: help          │  S: settings     │  r: refresh         │
│                   │  n: new job      │                     │
│                   │  d: delete       │                     │
│  MODALS           │  r: refresh      │  LOG VIEWER         │
│  ←→/Tab: move     │  u: updates      │  ↑↓: scroll         │
│  Enter: confirm   │                  │  PgUp/Dn: page      │
│  Esc: cancel      │  VIM KEYS        │  p: prev batch      │
│  y/n: yes/no      │  j/k: ↓/↑       │  n: next batch      │
│                   │  h/l: ←/→       │  Esc: close         │
└─────────────────────────────────────────────────────────────┘
```
