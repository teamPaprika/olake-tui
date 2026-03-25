---
title: "Settings"
weight: 8
---

The Settings screen lets you configure system-wide options: webhook notifications for sync events and software update checks. Settings are stored in the OLake metadata database and persist across TUI restarts.

## Accessing Settings

There are two ways to open Settings:

| From | Key | What Opens |
|------|-----|------------|
| Any tab | `4` | Settings tab (direct) |
| Jobs tab | `S` (capital S) | Job Settings (per-job), which also has a link to system settings |

> **`4` vs `S`:** The `4` key switches to the system-wide Settings tab. Capital `S` opens settings for a specific job (frequency, pause, clear destination). They're different screens.

## What the Settings Screen Looks Like

```
┌──────────────────────────────────────────────────────────────────┐
│  ⚙  System Settings                                             │
│                                                                  │
│  Webhook Notifications                                           │
│  Receive sync status alerts via webhook (Slack, Discord, etc.)   │
│                                                                  │
│  Webhook URL    [https://hooks.slack.com/services/YOUR/WEBHOOK/URL ] │
│                                                                  │
│         ╭────────╮    ╭──────────╮                               │
│         │  Save  │    │  Cancel  │                               │
│         ╰────────╯    ╰──────────╯                               │
│                                                                  │
│  ────────────────────────────────────────────────────            │
│  olake-tui v0.5.0                                                │
│                                                                  │
│  tab/↑↓: navigate  •  enter: activate  •  esc: cancel            │
└──────────────────────────────────────────────────────────────────┘
```

### Navigation

| Key | Action |
|-----|--------|
| `Tab` / `↓` | Move to next element (URL field → Save → Cancel → back to URL) |
| `Shift+Tab` / `↑` | Move to previous element |
| `Enter` (on URL field) | Jump to Save button |
| `Enter` (on Save) | Save settings to database |
| `Enter` (on Cancel) | Discard changes and go back |
| `Esc` | Cancel and close (same as Cancel button) |

## Webhook URL

The webhook URL is the primary setting. When configured, OLake TUI sends HTTP POST requests to this URL whenever a sync event occurs.

### What Events Trigger Webhooks

| Event | When It Fires |
|-------|---------------|
| `sync.completed` | A sync job finished successfully |
| `sync.failed` | A sync job encountered an error |
| `sync.canceled` | A sync was manually canceled |
| `clear.completed` | A destination clear finished |

### Webhook Payload Format

Every webhook sends a JSON POST request with this structure:

```json
{
  "event": "sync.completed",
  "job_name": "daily-pg-sync",
  "status": "completed",
  "duration_seconds": 272,
  "rows_synced": 1248301,
  "timestamp": "2025-03-15T06:04:32Z"
}
```

Field details:

| Field | Type | Description |
|-------|------|-------------|
| `event` | string | One of: `sync.completed`, `sync.failed`, `sync.canceled`, `clear.completed` |
| `job_name` | string | The human-readable name of the job |
| `status` | string | Final status: `completed`, `failed`, or `canceled` |
| `duration_seconds` | number | How long the operation took |
| `rows_synced` | number | Number of rows processed (0 for clear events) |
| `timestamp` | string | ISO 8601 timestamp in UTC |

### Endpoint Requirements

Your webhook endpoint must:

- Accept **HTTP POST** requests
- Accept **`Content-Type: application/json`**
- Return a **2xx status code** to acknowledge receipt
- Respond within a **reasonable timeout** (the TUI doesn't retry on failure)

## Example: Setting Up Slack Webhooks

Here's a complete walkthrough for sending OLake sync notifications to a Slack channel.

### Step 1: Create a Slack Incoming Webhook

1. Go to [api.slack.com/apps](https://api.slack.com/apps)
2. Click **Create New App** → **From scratch**
3. Name it "OLake Notifications", pick your workspace
4. Go to **Incoming Webhooks** → Toggle **Activate Incoming Webhooks** ON
5. Click **Add New Webhook to Workspace**
6. Pick the channel (e.g., `#data-pipelines`)
7. Copy the webhook URL — it looks like:
   ```
   https://hooks.slack.com/services/YOUR/WEBHOOK/URL
   ```

### Step 2: Enter the URL in OLake TUI

1. Open Settings (press `4` from any tab)
2. The Webhook URL field is focused by default
3. Paste the Slack webhook URL
4. Press `Enter` to jump to Save, then `Enter` again to save

```
  Webhook URL    [https://hooks.slack.com/services/YOUR/WEBHOOK/URL ]
```

### Step 3: Verify

Trigger a manual sync on any job (press `s` on the Jobs tab). When it completes, you should see a message in your Slack channel.

> **Note:** Slack expects a specific payload format for rich messages. The default OLake webhook payload is plain JSON, which Slack will display as raw text. For formatted Slack messages, you'd need a middleware that transforms the OLake payload into Slack's Block Kit format.

## Example: Setting Up Discord Webhooks

Discord also supports incoming webhooks with JSON payloads.

### Step 1: Create a Discord Webhook

1. Open your Discord server
2. Go to **Server Settings** → **Integrations** → **Webhooks**
3. Click **New Webhook**
4. Name it "OLake", pick a channel
5. Copy the webhook URL:
   ```
   https://discord.com/api/webhooks/1234567890/abcdefghijklmnop
   ```

### Step 2: Enter the URL in OLake TUI

Same as Slack — paste the URL in Settings and save.

### Step 3: Adapt the Payload (Optional)

Discord webhooks expect a `content` field for simple text messages. The raw OLake JSON payload will be rejected by Discord. You have two options:

**Option A: Use a proxy/middleware** that transforms OLake's JSON into Discord's format:
```json
{
  "content": "✅ **daily-pg-sync** completed in 4m 32s (1,248,301 rows)"
}
```

**Option B: Use a webhook relay service** like [Pipedream](https://pipedream.com) or [Make](https://www.make.com) that sits between OLake and Discord, transforming the payload.

## Update Checks

OLake TUI can check for new releases. When a newer version is available, a notification appears on the Jobs tab:

```
┌──────────────────────────────────────────────────────────────────┐
│  Jobs  Sources  Destinations  Settings                           │
│                                                                  │
│  Update available: v0.5.0 → v0.6.0 (press 'u' to view)         │
│ ─────────────────────────────────────────────────────────────    │
│  ▶ daily-pg-sync        Running   2m ago                        │
│    weekly-backup         Idle      6h ago                        │
└──────────────────────────────────────────────────────────────────┘
```

Press `u` to open the Updates modal.

### The Updates Modal

The updates modal shows release categories and their versions in a two-pane layout:

```
╭────────────────────────────────────────────────────────────╮
│  OLake Updates                                             │
│                                                            │
│  Categories         │  Releases                            │
│  ─────────          │  ────────                            │
│  ▶ OLake TUI  ●    │  ▶ v0.6.0  2025-03-15  [new]        │
│    OLake Core       │    Bug fixes and perf improvements   │
│    Connectors       │                                      │
│                     │    v0.5.0  2025-02-01                │
│                     │    v0.4.0  2025-01-10                │
│                                                            │
│  tab: switch pane  ↑↓/j/k: navigate  enter: expand        │
│  esc: close                                                │
╰────────────────────────────────────────────────────────────╯
```

Navigation:
- `Tab` — switch between category list and release list
- `↑`/`↓` or `j`/`k` — navigate within the current pane
- `Enter` or `Space` — expand/collapse release notes
- `Esc` or `q` — close the modal

The red `●` dot next to a category means it has new releases you haven't seen.

### Custom Release URL

By default, OLake TUI checks the official GitHub releases. For internal builds, private mirrors, or air-gapped environments, override the release URL:

```bash
olake-tui --release-url "https://internal.example.com/releases/latest.json"

# Via environment variable
export RELEASE_URL="https://internal.example.com/releases/latest.json"
```

The URL must return JSON in this format:

```json
{
  "version": "0.6.0",
  "release_notes": "Bug fixes and performance improvements.",
  "download_url": "https://github.com/datazip-inc/olake-tui/releases/tag/v0.6.0"
}
```

### Air-Gapped Mode

If you don't set a release URL and the TUI can't reach GitHub, update checks silently fail. The Updates modal will show:

```
╭──────────────────────────────────────────────╮
│  OLake Updates                               │
│                                              │
│  No updates available.                       │
│                                              │
│  esc: close                                  │
╰──────────────────────────────────────────────╯
```

This is expected behavior, not an error. The TUI works perfectly fine without update checks.

### Disabling Update Checks Entirely

To explicitly disable update checks (no network calls at all):

```bash
olake-tui --release-url ""
```

## Version Display

The current version is always shown at the bottom of the Settings screen:

```
  ────────────────────────────────────────────────────
  olake-tui v0.5.0
```

The `v` prefix is stripped from the stored version string for display. If the version is unknown (e.g., development build), it shows `olake-tui vunknown`.

## Settings Persistence

All settings are stored in the OLake metadata database (PostgreSQL), in the project settings table. This means:

- Settings persist across TUI restarts
- Settings are global — there are no per-user settings
- Settings are shared with the BFF web UI (same database)
- Changing settings in the TUI affects the web UI, and vice versa

## Troubleshooting

### Webhook Not Firing

**Symptoms:** You configured a webhook URL, but no notifications are arriving.

**Check these in order:**

1. **URL validation** — Make sure the URL is well-formed (`https://...`). The TUI saves whatever you type without strict validation.

2. **Network connectivity** — Can the machine running OLake TUI reach the webhook URL?
   ```bash
   curl -X POST -H "Content-Type: application/json" \
     -d '{"test": true}' \
     "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
   ```

3. **Firewall/proxy** — Corporate networks may block outbound HTTPS connections to external services.

4. **Payload format** — Some services (Discord) require specific payload formats and will reject the raw OLake JSON. See the Discord example above.

5. **Endpoint timeout** — If the webhook endpoint takes too long to respond, the TUI may time out and silently drop the notification.

6. **Did you save?** — Make sure you pressed Enter on the "Save" button, not just typed the URL and pressed Esc.

### Updates Modal Shows "No updates available"

**This is normal in these situations:**

- You're already on the latest version
- You're in an air-gapped environment without internet access
- The `--release-url` is set to an empty string (updates disabled)
- The release URL returns invalid JSON or is unreachable

**If you expect to see updates:**
1. Check that `RELEASE_URL` points to a valid endpoint
2. Test the URL manually:
   ```bash
   curl -s "https://internal.example.com/releases/latest.json" | jq .
   ```
3. Verify the JSON response matches the expected format (see Custom Release URL above)

### Settings Not Persisting

**If settings revert after restart:**
1. Verify you pressed "Save" (not "Cancel" or Esc)
2. Check database connectivity — if the DB write failed, the setting wasn't persisted
3. Check for database permissions — the TUI needs INSERT/UPDATE on the project settings table

### "Database error" When Saving Settings

The TUI stores settings in the `olake-<mode>-project-settings` table. If this table doesn't exist:

```bash
olake-tui --migrate
```

This creates all required tables, including the project settings table.
