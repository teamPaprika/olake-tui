---
title: "Authentication"
weight: 2
---

OLake TUI requires authentication before use. On first launch you seed an admin user, then log in with those credentials on every subsequent session.

## First-Time Setup

Run with the `--migrate` flag to create the database schema and seed the initial admin user:

```bash
olake-tui --migrate
```

You will be prompted to enter:

1. **Email** — the admin login email
2. **Password** — minimum 8 characters

The password is hashed with **bcrypt** (cost factor 10) and stored in the `users` table. The plaintext is never persisted.

After seeding, the TUI launches normally and presents the login screen.

## Login Screen

Every time OLake TUI starts, you see the login form:

```
┌─────────────────────────┐
│       OLake TUI         │
│                         │
│  Email:    [          ] │
│  Password: [          ] │
│                         │
│       [ Login ]         │
└─────────────────────────┘
```

- **Tab** — move between fields
- **Enter** — submit the form

On successful login, you are taken to the main dashboard. On failure, an error message appears and you can retry.

## Password Hashing & BFF Compatibility

OLake TUI uses the same **bcrypt** hashing scheme as the OLake BFF (Backend-for-Frontend) API server. This means:

- Users created via the BFF web UI can log in to the TUI
- Users created via `--migrate` in the TUI can log in to the web UI
- Password hashes are fully interchangeable between the two interfaces

The bcrypt hash format stored in the database looks like:

```
$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

## Session Behavior

- The TUI holds an authenticated session in memory for the duration of the process
- There are no session tokens or cookies — authentication is checked once at startup
- Restarting the TUI requires logging in again

## Multiple Users

The `--migrate` flag seeds a single admin user. To add more users, use the OLake BFF API or insert directly into the `users` table with a valid bcrypt hash:

```sql
INSERT INTO users (email, password_hash, created_at)
VALUES ('user@example.com', '$2a$10$...', NOW());
```

## Troubleshooting

| Problem | Cause | Fix |
|---------|-------|-----|
| "Invalid credentials" | Wrong email or password | Re-enter credentials; reset via DB if needed |
| "No users found" | Migration not run | Run `olake-tui --migrate` |
| "Database error" on login | DB unreachable | Check `DATABASE_URL` (see [Connecting](../connecting/)) |
| BFF password doesn't work | Different bcrypt cost factor | Both use cost 10 by default; verify hash format |
