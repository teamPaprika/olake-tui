---
title: "Changelog"
weight: 2
---


All notable changes to olake-tui are documented here.

## v0.2.0-direct (2025-03-20)

Full BFF API parity — olake-tui now covers every operation the OLake web UI supports, with direct database access instead of HTTP calls.

### New Features

- **UpdateJobFull** — edit all job properties (schedule, streams, config) in a single operation
- **IsNameUnique** — real-time uniqueness validation for source, destination, and job names
- **GetClearDestStatus** — check whether a destination clear operation is in progress
- **RecoverFromClearDest** — recover a destination stuck in clearing state
- **Real-time Temporal status** — job run status polls Temporal directly instead of relying on cached state
- **Direct DB mode** — bypass the BFF HTTP layer entirely; all queries go straight to PostgreSQL

### Improvements

- Job wizard now validates name uniqueness before submission
- Stream selection modal shows sync status from Temporal in real time
- Destination detail view displays clear/recover actions when applicable
- Reduced latency on all CRUD operations by eliminating HTTP round-trips

---

## v0.1.0 (2025-02-15)

Initial release — a fully functional terminal UI for OLake pipeline management.

### Features

- **Source management** — create, list, edit, delete, and test sources (MongoDB, PostgreSQL)
- **Destination management** — create, list, edit, delete destinations (Apache Iceberg, S3, local)
- **Job management** — full CRUD with schedule configuration
- **Job wizard** — step-by-step job creation: select source → destination → configure → discover streams → schedule
- **Stream selection** — browse discovered streams, toggle sync, configure sync modes per stream
- **Log viewer** — real-time log tailing for job runs with search and filtering
- **17 modal dialogs** — confirmation, input, form, error, help, and specialized modals
- **Encryption** — AES-256-GCM encryption for source/destination configs at rest
- **Temporal integration** — trigger, cancel, and monitor job runs via Temporal workflows
- **Authentication** — login screen with admin user seeding via `--migrate`
- **Table prefix modes** — `dev` / `prod` / `staging` isolation via `--run-mode`
- **Update checker** — checks `releases.json` for new versions on startup

### Architecture

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- Repository pattern with PostgreSQL via `pgx`
- Temporal SDK for workflow orchestration
- Modular view/model separation for each screen

---

## Pre-release Development Highlights

- Initial scaffold with Bubble Tea framework and router-based navigation
- PostgreSQL schema design with run-mode prefixed tables
- Temporal workflow client integration for job execution
- Config encryption layer with AES-256-GCM
- Comprehensive modal system (confirm, input, error, form, select, help)
- Job wizard with multi-step flow and stream discovery
- Migration system with admin user seeding
- CLI flag and environment variable configuration
