---
title: "Key Bindings"
weight: 9
---

OLake TUI is fully keyboard-driven. This page lists every key binding organized by context.

## Global

These key bindings work everywhere in the application.

| Key | Action |
|-----|--------|
| `1` | Switch to Jobs tab |
| `2` | Switch to Sources tab |
| `3` | Switch to Destinations tab |
| `4` | Switch to fourth tab |
| `Tab` | Cycle to next tab |
| `q` | Quit the application |
| `Ctrl+C` | Quit the application (force) |
| `Esc` | Go back / close current view |

## Jobs Tab

| Key | Action |
|-----|--------|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `Enter` | Open job detail view |
| `n` | Create new job |
| `S` | Open settings |
| `s` | Start sync for selected job |
| `c` | Cancel running sync |
| `p` | Pause / resume scheduled job |
| `l` | Open log viewer for selected job |
| `d` | Delete selected job |
| `u` | Open updates modal |
| `r` | Refresh job list |

## Sources Tab

| Key | Action |
|-----|--------|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `Enter` | Open source detail view |
| `a` | Add new source |
| `e` | Edit selected source |
| `d` | Delete selected source |
| `t` | Test connection |
| `r` | Refresh source list |

## Destinations Tab

| Key | Action |
|-----|--------|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `Enter` | Open destination detail view |
| `a` | Add new destination |
| `e` | Edit selected destination |
| `d` | Delete selected destination |
| `t` | Test connection |
| `r` | Refresh destination list |

## Log Viewer

| Key | Action |
|-----|--------|
| `↑` | Scroll up one line |
| `↓` | Scroll down one line |
| `PgUp` | Scroll up one page |
| `PgDn` | Scroll down one page |
| `p` | Load older (previous) log page |
| `n` | Load newer (next) log page |
| `Esc` | Close log viewer |

## Modals & Forms

| Key | Action |
|-----|--------|
| `Tab` | Move to next field |
| `Shift+Tab` | Move to previous field |
| `Enter` | Submit form / confirm action |
| `Esc` | Cancel and close modal |
| `Space` | Toggle checkbox (stream selection) |
| `y` | Confirm deletion prompt |
| `n` | Decline deletion prompt |

## Tips

- **Vim-style navigation**: `j`/`k` work as alternatives to arrow keys in all list views
- **Capital S for settings**: Use `Shift+S` to avoid conflicting with `s` (sync) on the Jobs tab
- **Quick tab switching**: Number keys `1`–`4` jump directly to a tab without cycling through with `Tab`
- **Escape is universal**: Press `Esc` from any nested view to go back one level
