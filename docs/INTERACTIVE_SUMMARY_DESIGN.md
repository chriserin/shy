# Interactive Summary TUI Design

This document describes the feature set for converting `shy summary` into an interactive Bubble Tea application.

## Overview

The interactive summary will be a terminal UI application that allows users to
explore their shell command history across different contexts (directories, git
branches), time periods, and with various filters. It launches with a view of
"yesterday's contexts" by default and supports keyboard-driven navigation.

## Visual Design Philosophy

Retain the core visual elements of the current `shy summary`:

- Context headers in `directory:branch` format
- Time bucket separators (hourly/period/day/week labels with dashed lines)
- Color scheme: magenta headers, blue context/labels, green stats, white commands, dim timestamps
- Command timestamps with minute-only format (`:30`, `:15`) within buckets
- Tilde-compressed paths (`~` for home directory)

### Focus Indicator

The header includes a dot after "Work Summary" to indicate focus state:

- **Focused**: Solid dot `●` in the accent color (bright-magenta) — `Work Summary ● 2026-02-04 (Yesterday)`
- **Blurred**: Outline dot `○` in dim color — `Work Summary ○ 2026-02-04 (Yesterday)`

## Views

### 1. Summary View (Default)

The landing view showing all contexts for the selected time period. Contexts are shown collapsed by default—just the context name and command count.

```
Work Summary - 2026-02-04 (Yesterday)                      [Day] Week Month
================================================================================

  ~/projects/shy:main                                         42 commands
▶ ~/projects/other:feature-x                                  18 commands
  ~/downloads                                                   5 commands

────────────────────────────────────────────────────────────────────────────────
[j/k] Select  [Enter] Drill in  [h/l] Time  [u] Uniq  [m] Multi  [a] All  [?] Help
```

**Elements:**

- Header with date and relative label ("Yesterday", "Today", "2 days ago")
- Time period selector tabs (Day/Week/Month)
- Current selection indicator (`▶`)
- Context list with command counts
- Status bar with key hints

### 2. Context Detail View

Drilled-in view showing all commands within a single context. The command count is no longer shown since you're viewing the actual commands.

```
~/projects/shy:main                                    2026-02-04 (Yesterday)
================================================================================

  8am ─────────────────────────────────────────────────────────────────────────
    :15  go build -o shy .
    :22  ./shy summary
    :30  go test ./... -v
    :45  git status
    :52  git add .

  9am ─────────────────────────────────────────────────────────────────────────
  ▶ :00  git commit -m "feat: add summary"
    :05  go build -o shy .
    :12  ./shy summary --all-commands
    :15  ./shy summary --date 2026-01-15
    :30  git push

  2pm ─────────────────────────────────────────────────────────────────────────
    :20  shy summary --all-commands
    :25  go test ./cmd -run TestSummary -v

────────────────────────────────────────────────────────────────────────────────
[j/k] Select  [Esc] Back  [h/l] Time  [H/L] Context  [/] Filter  [?] Help
```

**Elements:**

- Context header (directory:branch) with date
- All time buckets for this context
- Command selection indicator (on commands only, not bucket headers)
- Individual command list within each bucket
- Filter indicator when active

#### Context Detail View (Week Period)

When "Week" is the selected period, buckets are days instead of hours:

```
~/projects/shy:main                                   Week of 2026-02-03
================================================================================

  Mon Feb 3 ──────────────────────────────────────────────────────────────────
    9:15 AM  go build -o shy .
    9:30 AM  go test ./... -v
    2:20 PM  shy summary --all-commands

  Tue Feb 4 ──────────────────────────────────────────────────────────────────
  ▶ 8:15 AM  go build -o shy .
    8:22 AM  ./shy summary
    9:00 AM  git commit -m "feat: add summary"
    9:30 AM  git push

  Wed Feb 5 ──────────────────────────────────────────────────────────────────
    10:00 AM  npm install
    10:30 AM  npm test

────────────────────────────────────────────────────────────────────────────────
[j/k] Select  [Esc] Back  [h/l] Time  [H/L] Context  [/] Filter  [?] Help
```

### 3. Command Detail View

Detailed view of a single command (similar to `shy tv preview`).

```
Command Details                                                       Event: 1234
================================================================================

  Command:     git commit -m "feat: add interactive summary TUI"
  Exit Status: 0 ✓
  Timestamp:   2026-02-04 09:00:15
  Working Dir: ~/projects/shy
  Duration:    1.2s
  Git Repo:    github.com/chris/shy
  Git Branch:  main
  Session:     zsh:12345

  ─────────────────────────────────────────────────────────────────────────────

  Context (same session):
    1231  go build -o shy .
    1232  ./shy summary
    1233  git status
  ▶ 1234  git commit -m "feat: add interactive summary TUI"
    1235  git push
    1236  ./shy summary --date today

────────────────────────────────────────────────────────────────────────────────
[Esc] Back  [y] Copy command  [Enter] Execute  [?] Help
```

**Elements:**

- Command metadata (event number, command text, exit status, timestamp, working dir, duration, git info, session)
- Session context showing surrounding commands from the same session
- Selection indicator on current command within context list

**Navigation:**

- `j`/`k` moves through commands in the session context, updating the metadata display
- `Esc` returns to the Context Detail View
- `Enter` executes the command (with confirmation prompt)

## Navigation

### Global Keys

| Key            | Action                  |
| -------------- | ----------------------- |
| `q` / `Ctrl+C` | Quit                    |
| `?`            | Toggle help overlay     |
| `Esc`          | Go back / close overlay |

### Scroll Behavior

In Summary View and Context Detail View, the viewport scrolls to keep the selected item visible. The selection can be anywhere in the viewport, including at the top or bottom edge.

In Command Detail View, the selected command stays centered in the session context list, with surrounding commands visible above and below.

### Summary View Navigation

| Key       | Action                                |
| --------- | ------------------------------------- |
| `j` / `↓` | Move selection down                   |
| `k` / `↑` | Move selection up                     |
| `Enter`   | Drill into selected context           |
| `h`       | Previous time period (day/week/month) |
| `l`       | Next time period (day/week/month)     |
| `t`       | Go to today                           |
| `y`       | Go to yesterday                       |
| `1`       | Day view (bucket by hour)             |
| `2`       | Week view (bucket by day)             |
| `3`       | Month view (bucket by week)           |
| `u`       | Toggle unique commands only           |
| `m`       | Toggle multi-run commands only        |
| `a`       | Toggle all commands                   |
| `/`       | Open filter input                     |
| `Space`   | Expand/collapse selected context      |

### Context Detail View Navigation

| Key       | Action                                |
| --------- | ------------------------------------- |
| `j` / `↓` | Move selection down                   |
| `k` / `↑` | Move selection up                     |
| `Enter`   | View command details                  |
| `Esc`     | Back to summary view                  |
| `h`       | Previous time period (day/week/month) |
| `l`       | Next time period (day/week/month)     |
| `H`       | Previous context (same time period)   |
| `L`       | Next context (same time period)       |
| `[`       | Jump to previous bucket               |
| `]`       | Jump to next bucket                   |
| `/`       | Filter commands in context            |

### Command Detail View Navigation

| Key       | Action                              |
| --------- | ----------------------------------- |
| `Esc`     | Back to context view                |
| `j` / `↓` | Next command in context             |
| `k` / `↑` | Previous command in context         |
| `Enter`   | Execute command (with confirmation) |

## Features

### Time Period Selection

| Period | Bucket Size                | Date Range                       |
| ------ | -------------------------- | -------------------------------- |
| Day    | Hour (8am, 9am, ...)       | Single day                       |
| Week   | Day (Mon, Tue, ...)        | 7 days ending on selected date   |
| Month  | Week (Week 1, Week 2, ...) | ~30 days ending on selected date |

When switching periods:

- Day → Week: Show the week containing the current day
- Week → Month: Show the month containing the current week
- The selected date remains the "anchor" for navigation

### Date Navigation

- `h`/`←` and `l`/`→` move by the current period unit (day/week/month)

### Filtering

Filter input appears as an inline bar at the bottom:

```
Filter: go test█
```

The filter always filters commands. In Summary View, the command counts update to show only matching commands (contexts with no matches show "0 commands"). In Context Detail View, only matching commands are shown.

Filter persists when drilling in/out. Clear with empty filter.

### Display Modes

Three mutually exclusive display modes (only one active at a time):

| Mode               | Flag      | Description                                       |
| ------------------ | --------- | ------------------------------------------------- |
| Summary            | (default) | Show command counts per bucket                    |
| All Commands       | `a`       | Show all commands with timestamps                 |
| Unique Commands    | `u`       | Show commands executed exactly once               |
| Multi-run Commands | `m`       | Show commands executed multiple times with counts |

Visual indicator shows active mode in status bar.

### Context Expansion

In Summary View, contexts are collapsed by default (showing just context name and command count). Use `Enter` to drill into a context for the full detail view, or use `Space` to expand inline.

Expansion states:

- **Collapsed** (default): Just the context header with total command count
- **Expanded**: Header + bucket previews with first few commands per bucket + "more" indicator

## Data Model

### State

```go
type Model struct {
    // Data
    db          *db.DB
    contexts    []ContextData
    currentDate time.Time

    // View state
    view        View          // Summary, Context, Command
    selectedIdx int           // Selected context/command index

    // Display options
    period      Period        // Day, Week, Month
    displayMode DisplayMode   // Summary, All, Unique, Multi
    filter      string
    filterActive bool

    // UI state
    width       int
    height      int
    showHelp    bool

    // For context view
    currentContext  *ContextData
    selectedBucket  int
    selectedCommand int

    // For command view
    currentCommand *models.Command
    commandContext []models.Command // surrounding commands
}

type ContextData struct {
    Key      summary.ContextKey
    Branches map[summary.BranchKey][]models.Command
    Expanded bool
}

type View int
const (
    SummaryView View = iota
    ContextView
    CommandView
)

type Period int
const (
    Day Period = iota
    Week
    Month
)

type DisplayMode int
const (
    SummaryMode DisplayMode = iota
    AllMode
    UniqueMode
    MultiMode
)
```

### Messages

```go
// Navigation
type NavigateMsg struct { Direction Direction }
type DrillInMsg struct {}
type DrillOutMsg struct {}
type SelectDateMsg struct { Date time.Time }
type ChangePeriodMsg struct { Period Period }

// Display
type ToggleDisplayModeMsg struct { Mode DisplayMode }
type ToggleExpandMsg struct {}
type SetFilterMsg struct { Filter string }

// Data
type CommandsLoadedMsg struct { Commands []models.Command }
type ContextLoadedMsg struct { Context *ContextData }
type ErrorMsg struct { Err error }

// Actions
type ExecuteCommandMsg struct { Command string }
```

## Component Structure

```
App
├── Header (date, period tabs, mode indicators)
├── Main Content
│   ├── SummaryList (list of contexts with command counts)
│   │   └── ContextItem (context name + count, expandable)
│   ├── ContextDetail (single context with all commands)
│   │   └── BucketList (list of time buckets)
│   │       └── CommandList (commands in bucket)
│   └── CommandDetail (single command metadata + context)
├── FilterBar (when active)
├── StatusBar (key hints)
└── HelpOverlay (when toggled)
```

## CLI Interface

```bash
shy summary
```

Launches the interactive TUI starting at yesterday. All navigation (date, time period, context) and display options (all/uniq/multi, filtering) are handled interactively within the TUI.

## Implementation Phases

### Phase 1: Core Navigation

- Basic Bubble Tea app structure
- Summary View with context list
- Date navigation (prev/next day)
- Context selection

### Phase 2: Drill-down

- Context Detail View
- Command navigation within context
- Back navigation
- Context switching (prev/next)

### Phase 3: Display Modes

- All/Unique/Multi command toggles
- Context expansion/collapse
- Bucket-level navigation

### Phase 4: Advanced Features

- Command Detail View
- Filter input and application
- Week/Month period views

### Phase 5: Actions & Polish

- Command execution (with confirmation)
- Help overlay
- Responsive layout
- Error handling

## Dependencies

Add to `go.mod`:

```
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles  # for list, textinput, viewport, help
```

The project already uses `lipgloss` for styling, which integrates well with Bubble Tea.

## File Structure

```
cmd/
├── summary.go           # Existing static summary (modify to add -i flag)
└── summary_tui.go       # New: Bubble Tea entry point

internal/
├── summary/
│   ├── ... (existing)
│   └── tui/
│       ├── model.go     # Main model and Update logic
│       ├── view.go      # View rendering
│       ├── keys.go      # Key bindings
│       ├── styles.go    # Lipgloss styles (extract from formatter.go)
│       ├── summary.go   # Summary view component
│       ├── context.go   # Context detail component
│       ├── command.go   # Command detail component
│       └── filter.go    # Filter bar component
```

## Open Questions

1. **Vim vs Emacs keys?** Currently proposing vim-style (`hjkl`). Should we support both?

2. **Mouse support?** Bubble Tea supports mouse. Worth adding clickable elements?

3. **Persistence?** Remember last-used display mode, period, or expanded states between sessions?

4. **Live updates?** Should the view refresh if new commands are added while the TUI is open?
