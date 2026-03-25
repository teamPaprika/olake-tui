---
title: "olake-tui"
layout: "hextra-home"
---

<div class="hx-mt-6 hx-mb-6">
{{< hextra/hero-headline >}}
  ⬡ olake-tui
{{< /hextra/hero-headline >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-subtitle >}}
  Terminal UI for OLake — manage data pipelines&nbsp;<br class="sm:hx-block hx-hidden" />without leaving the command line.
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx-mb-6">
{{< hextra/hero-badge link="docs/getting-started/quick-start/" >}}
  <span>Get Started</span>
  {{< icon name="arrow-circle-right" attributes="height=14" >}}
{{< /hextra/hero-badge >}}
</div>

<br>

```bash
# Start infrastructure
docker compose up -d

# Connect and bootstrap
olake-tui --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
          --temporal-host localhost:7233 \
          --migrate
```

```
┌──────────────────────────────────────────────────────────────────┐
│  ⬡ OLake  logged in as admin                                     │
│                                                                   │
│  ╭──────────╮  ╭──────────────╮  ╭──────╮  ╭────────────╮       │
│  │ Sources: 3│  │Destinations:2│  │Jobs:5│  │Active Jobs:3│       │
│  ╰──────────╯  ╰──────────────╯  ╰──────╯  ╰────────────╯       │
│                                                                   │
│  [1] Jobs  [2] Sources  [3] Destinations  [4] Settings            │
│                                                                   │
│  ✓  nightly-postgres-sync   pg-prod   → s3-bucket   completed  ● │
│  ⟳  hourly-mysql-export     mysql-dev → iceberg     running    ● │
│  ✗  daily-mongo-backup      mongo     → s3-archive  failed     ● │
│                                                                   │
│  n:new  Enter:detail  s:sync  c:cancel  l:logs  d:delete         │
└──────────────────────────────────────────────────────────────────┘
```

<br>

<div class="hx-mt-6">
{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="No BFF Required"
    subtitle="Connects directly to PostgreSQL + Temporal via SQL and gRPC. The web server is optional."
    icon="database"
  >}}
  {{< hextra/feature-card
    title="Air-Gap Ready"
    subtitle="Zero external network calls. Runs in isolated networks, VPNs, and air-gapped environments."
    icon="shield-check"
  >}}
  {{< hextra/feature-card
    title="Full BFF Compatibility"
    subtitle="Same DB schema, AES-256-GCM encryption, and Temporal workflows. Data created by TUI works in the web UI and vice versa."
    icon="refresh"
  >}}
  {{< hextra/feature-card
    title="Keyboard-Driven"
    subtitle="17 modal dialogs, job creation wizard, stream selector, paginated log viewer — all from the terminal."
    icon="terminal"
  >}}
  {{< hextra/feature-card
    title="Standalone Bootstrap"
    subtitle="olake-tui --migrate creates all database tables and seeds an admin user. One 10MB binary."
    icon="lightning-bolt"
  >}}
  {{< hextra/feature-card
    title="Helm Chart Included"
    subtitle="Drop-in Kubernetes deployment. Disable the BFF, run the migration Job, connect via port-forward."
    icon="cloud"
  >}}
{{< /hextra/feature-grid >}}
</div>

<br>

## Why a TUI?

| Scenario | Web UI | olake-tui |
|----------|--------|-----------|
| Air-gapped network | ❌ Needs GitHub API for versions | ✅ Works offline |
| SSH into production | ❌ Need browser + port forward | ✅ Native terminal |
| CI/CD automation | ❌ Requires API scripting | ✅ CLI flags + exit codes |
| Resource usage | ~500MB (Node + Go + browser) | ~10MB single binary |
| Keyboard speed | Click, wait, click, wait | `s` to sync, `l` for logs |

<br>

<div style="text-align: center">

[Documentation →](docs/) · [Quick Start →](docs/getting-started/quick-start/) · [GitHub →](https://github.com/teamPaprika/olake-tui) · [OLake →](https://olake.io)

</div>
