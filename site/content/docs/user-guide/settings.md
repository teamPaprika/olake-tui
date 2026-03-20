---
title: "Settings"
weight: 8
---

OLake TUI has a settings screen for configuring system-wide options like webhook notifications and update checks.

## Accessing Settings

Press `S` (capital S) from the Jobs tab to open the settings screen.

## Webhook URL

Configure a webhook URL to receive notifications when sync jobs complete or fail. OLake TUI sends HTTP POST requests to this URL with a JSON payload.

```
Webhook URL: [https://hooks.example.com/olake  ]
```

### Webhook Payload

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

Supported events:

| Event | Trigger |
|-------|---------|
| `sync.completed` | Sync finished successfully |
| `sync.failed` | Sync encountered an error |
| `sync.canceled` | Sync was manually canceled |
| `clear.completed` | Destination clear finished |

Set the URL and press **Enter** to save. Leave empty to disable webhooks.

## System Settings Screen

The settings screen shows all configurable options:

```
┌─────────────────────────────────┐
│          Settings               │
│                                 │
│  Webhook URL: [              ]  │
│                                 │
│         [ Save ]  [ Cancel ]    │
└─────────────────────────────────┘
```

Navigate between fields with **Tab**. Press **Esc** to return to the Jobs tab without saving.

## Update Checks

OLake TUI can check for new releases. When an update is available, a notification appears on the Jobs tab:

```
Update available: v0.5.0 → v0.6.0 (press 'u' to view)
```

Press `u` to open the updates modal showing release notes and download instructions.

### Custom Release URL

By default, OLake TUI checks the official GitHub releases. If you host internal builds or a private release feed, use the `--release-url` flag:

```bash
olake-tui --release-url "https://internal.example.com/releases/latest.json"
```

The URL should return JSON in this format:

```json
{
  "version": "0.6.0",
  "release_notes": "Bug fixes and performance improvements.",
  "download_url": "https://github.com/datazip-inc/olake-tui/releases/tag/v0.6.0"
}
```

You can also set this via environment variable:

```bash
RELEASE_URL="https://internal.example.com/releases/latest.json"
```

### Disabling Update Checks

To disable update checks entirely, set the release URL to an empty string:

```bash
olake-tui --release-url ""
```

## Settings Persistence

All settings are stored in the OLake metadata database and persist across TUI restarts. They apply globally — there are no per-user settings.
