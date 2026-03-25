---
title: "Authentication"
weight: 2
---

OLake TUI requires you to log in every time it starts. There are no persistent sessions — authentication is checked once at launch, and the session lives only in memory until you quit.

This page covers the full login flow, how the admin user gets created, password management, and what to do when things go wrong.

## What the Login Screen Looks Like

When OLake TUI starts, it presents a centered login box in your terminal:

```
┌──────────────────────────────────────────────────┐
│                                                  │
│                  ⬡ OLake                         │
│           Data Pipeline Management               │
│                                                  │
│          Username  [admin             ]          │
│                                                  │
│          Password  [••••••••          ]          │
│                                                  │
│            tab: next field • enter: submit        │
│                                                  │
└──────────────────────────────────────────────────┘
```

The login screen fills the entire terminal and centers itself automatically. You'll interact with two fields:

- **Username** — your OLake username (not email, despite BFF's web UI using email)
- **Password** — masked with `•` characters as you type

### Navigating the Login Screen

| Key | What It Does |
|-----|-------------|
| `Tab` / `↓` | Move to the next field |
| `Shift+Tab` / `↑` | Move to the previous field |
| `Enter` (on username) | Jump to the password field |
| `Enter` (on password) | Submit the login form |

After a successful login, you're taken directly to the Jobs tab — the main dashboard.

## First-Time Setup: Creating the Admin User

Before you can log in, you need at least one user in the database. The `--migrate` flag handles this:

```bash
olake-tui --migrate
```

This does two things:

1. **Creates the database schema** — all OLake tables (`user`, `source`, `destination`, `job`, etc.) are created with `IF NOT EXISTS`, so it's safe to run multiple times
2. **Seeds an admin user** — if no users exist, you'll be prompted to enter a username and password

### What Happens Step by Step

```
$ olake-tui --migrate
Migrating database schema... ✓
No users found. Creating admin user.

Enter admin username: admin
Enter admin password: ********
Confirm password:     ********

Admin user created successfully.
Launching OLake TUI...
```

After seeding, the TUI launches normally and you'll see the login screen.

### What If You Run --migrate Again?

Nothing bad happens. Schema creation uses `IF NOT EXISTS` and the seed step checks `SELECT COUNT(*) FROM users` — if users already exist, it skips seeding entirely. You can safely run `--migrate` on every startup if you want.

## How Passwords Are Stored

OLake TUI uses **bcrypt** with the default cost factor of 10. When you create a user (via `--migrate` or directly in the database), the password goes through:

```
plaintext → bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost) → stored hash
```

A stored hash looks like this:

```
$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

The breakdown:
- `$2a$` — bcrypt algorithm identifier
- `10$` — cost factor (10 rounds of key expansion)
- The rest — salt + hash combined

During login, the TUI runs `bcrypt.CompareHashAndPassword()` against the stored hash. The plaintext password is never stored anywhere.

## BFF Compatibility: TUI ↔ Web UI

OLake TUI shares the same database and the same bcrypt scheme as the OLake BFF (Backend-for-Frontend) web server. This means:

| Created in... | Works in TUI? | Works in Web UI? |
|---------------|:------------:|:----------------:|
| TUI (`--migrate`) | ✅ | ✅ |
| BFF web UI | ✅ | ✅ |
| Direct SQL insert | ✅ | ✅ |

**One caveat:** The TUI's login form uses `username`, while the BFF web UI typically uses `email`. The TUI stores both — on seed, it generates an email as `username@olake.local`. If you create a user via the BFF's web interface using an email, you'll need to log into the TUI with the `username` field from the database, not the email.

## Session Behavior

OLake TUI does not use tokens, cookies, or session files. Here's what happens:

1. You start the TUI → login screen appears
2. You enter credentials → TUI calls `Login(username, password)`
3. On success → `authenticated = true` is stored in memory
4. You use the TUI normally
5. You quit (or the process dies) → session is gone

**Every restart requires logging in again.** There is no "remember me" option. This is by design — the TUI runs in a terminal, and persistent sessions would require storing secrets on disk.

## Adding More Users

The `--migrate` flag only creates a single admin user. To add more users, you have two options:

### Option 1: Via the BFF Web UI

If you're running the OLake web interface alongside the TUI, create users there. They'll work in the TUI immediately.

### Option 2: Direct SQL Insert

Connect to the OLake PostgreSQL database and insert a user with a bcrypt hash:

```sql
-- Generate the hash outside of SQL first (see below)
INSERT INTO "olake-dev-user" (username, password, email, created_at, updated_at)
VALUES ('newuser', '$2a$10$YOUR_BCRYPT_HASH_HERE', 'newuser@example.com', NOW(), NOW());
```

To generate a bcrypt hash, use any of these:

```bash
python3 -c "import bcrypt; print(bcrypt.hashpw(b'mypassword', bcrypt.gensalt(10)).decode())"

# Node.js
node -e "const b=require('bcryptjs');console.log(b.hashSync('mypassword',10))"

# htpasswd (Apache)
htpasswd -nbBC 10 "" mypassword | cut -d: -f2
```

> **Note:** The table name depends on your run mode. Default is `olake-dev-user`. If you're running in production mode, it would be `olake-prod-user`.

## Forgot Your Password?

There's no "forgot password" flow in the TUI. To reset a password:

### Step 1: Generate a New bcrypt Hash

```bash
python3 -c "import bcrypt; print(bcrypt.hashpw(b'newpassword123', bcrypt.gensalt(10)).decode())"
# Output: $2a$10$xJ5Kd...
```

### Step 2: Update the Database

```sql
UPDATE "olake-dev-user"
SET password = '$2a$10$xJ5Kd...'
WHERE username = 'admin';
```

### Step 3: Log In with the New Password

Restart the TUI and log in using the new password. The old password will no longer work.

## Complete Login Flow: Start to Dashboard

Here's the full sequence from launching the binary to landing on the dashboard:

```
┌─────────────────────────────────────────────────────────┐
│  $ olake-tui                                            │
│                                                         │
│  Connecting to database...            ✓                 │
│  Connecting to Temporal...            ✓                 │
│  Starting TUI...                                        │
│                                                         │
│  ┌────────────────────────────────────────────────┐     │
│  │              ⬡ OLake                           │     │
│  │       Data Pipeline Management                 │     │
│  │                                                │     │
│  │      Username  [admin             ]            │     │
│  │      Password  [                  ]            │     │
│  │                                                │     │
│  └────────────────────────────────────────────────┘     │
│                                                         │
│  (enter credentials, press Enter)                       │
│                                                         │
│  ┌────────────────────────────────────────────────┐     │
│  │  Jobs  Sources  Destinations  Settings         │     │
│  │ ──────────────────────────────────────────     │     │
│  │  ▶ daily-pg-sync        Running   2m ago      │     │
│  │    weekly-backup         Idle      6h ago      │     │
│  │    staging-mirror        Paused    1d ago      │     │
│  └────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────┘
```

If login fails, you stay on the login screen with an error message:

```
┌────────────────────────────────────────────────┐
│              ⬡ OLake                           │
│       Data Pipeline Management                 │
│                                                │
│      Username  [admin             ]            │
│      Password  [                  ]            │
│                                                │
│       ✗ invalid credentials                    │
│                                                │
│        tab: next field • enter: submit         │
└────────────────────────────────────────────────┘
```

You can retry immediately — just re-enter your password and press Enter.

## Troubleshooting

### "invalid credentials"

**What it means:** Either the username doesn't exist in the database, or the password is wrong. The TUI intentionally gives the same error for both cases (to avoid leaking whether a username exists).

**What to do:**
1. Double-check your username (it's case-sensitive)
2. Make sure you're using the username, not an email address
3. If you're sure the username is right, reset the password via SQL (see above)
4. Verify you're connecting to the right database — check `DATABASE_URL`

### "No users found" or Login Screen Doesn't Appear

**If you see "no users found":** Run `olake-tui --migrate` to create the schema and seed the admin user.

**If the login screen never appears:**
- Check that the TUI is connecting to the database successfully (look for connection errors in the startup output)
- Verify your `DATABASE_URL` environment variable points to the correct PostgreSQL instance
- See [Connecting](../connecting/) for database connection troubleshooting

### "Database error" on Login

**What it means:** The TUI can't reach the PostgreSQL database.

**What to do:**
1. Verify `DATABASE_URL` is set and correct
2. Check that PostgreSQL is running and accepting connections
3. Test the connection manually:
   ```bash
   psql "$DATABASE_URL" -c "SELECT 1"
   ```
4. Check for firewall rules or network issues if the database is remote

### BFF-Created User Can't Log In to TUI

**What's happening:** The BFF web UI may store users with an email as the primary identifier, while the TUI login expects the `username` field.

**What to do:**
1. Check what's in the database:
   ```sql
   SELECT username, email FROM "olake-dev-user";
   ```
2. Use the value from the `username` column (not `email`) when logging into the TUI
3. If the username field is empty, update it:
   ```sql
   UPDATE "olake-dev-user" SET username = 'admin' WHERE email = 'admin@example.com';
   ```

### Password Hash Format Mismatch

If you inserted a user manually and login fails even with the correct password, verify the hash format:

```sql
SELECT password FROM "olake-dev-user" WHERE username = 'admin';
```

The hash **must** start with `$2a$10$` (or `$2b$10$`). If it starts with anything else, the bcrypt comparison will fail. Regenerate the hash using the commands in the "Adding More Users" section.
