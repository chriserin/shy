# Yesterday's Work Summary - Design B (Tabular)

## Overview

A simplified tabular summary showing time spent in different contexts (working directories and git branches) based on command activity.

## Goals

1. Provide a quick, scannable table of where time was spent
2. Group by working directory and git branch
3. Calculate time span (first to last command) for each context
4. Simple, clear tabular output suitable for daily standups

## Command Structure

```bash
shy summary [options]
```

### Options

- `--date <date>` - Date to summarize (default: yesterday)
  - `yesterday` (default)
  - `today`
  - `YYYY-MM-DD` (e.g., `2026-01-14`)

### Examples

```bash
# Yesterday's summary
shy summary

# Specific date
shy summary --date 2026-01-14

# Today's work
shy summary --date today
```

## Output Format

### Tabular Display

```
Yesterday's Work Summary - 2026-01-14
=====================================

Directory                          Branch              Commands  Time Span       Duration
~/projects/shy                     yesterdays-summary        45  08:23 - 16:35   8h 12m
~/projects/shy                     main                      12  19:15 - 21:32   2h 17m
~/projects/webapp                  feature/auth              23  10:15 - 15:42   5h 27m
~/projects/dotfiles                -                          8  14:10 - 14:35      25m

Total: 88 commands across 4 contexts
```

### Column Descriptions

1. **Directory**: Working directory (relative to home with ~ expansion)
2. **Branch**: Git branch name, or "-" for non-git directories
3. **Commands**: Total number of commands executed in this context
4. **Time Span**: First command time to last command time (HH:MM format)
5. **Duration**: Calculated duration between first and last command

## Calculation Logic

### Duration Calculation

Duration represents the time span between the **first** and **last** command in a context:

```
Duration = Last Command Timestamp - First Command Timestamp
```

**Important Notes:**
- Duration is the time span, not active work time
- Gaps between commands are included in the duration
- A single command has 0 duration
- This gives a rough estimate of how long you were "in" that context

### Context Definition

A context is defined by the tuple: `(working_dir, git_branch)`

- Same directory, different branches = different contexts
- Same branch, different directories = different contexts
- Non-git directories use "-" for branch

### Sorting

Contexts are sorted by:
1. Primary: Total duration (descending) - longest time first
2. Secondary: Number of commands (descending)
3. Tertiary: Working directory (alphabetical)

This puts the contexts where you spent the most time at the top.

## Data Model

### Database Query

```sql
SELECT
    working_dir,
    git_branch,
    COUNT(*) as command_count,
    MIN(timestamp) as first_command,
    MAX(timestamp) as last_command,
    MAX(timestamp) - MIN(timestamp) as duration_seconds
FROM commands
WHERE timestamp >= ? AND timestamp < ?
GROUP BY working_dir, git_branch
ORDER BY duration_seconds DESC, command_count DESC, working_dir ASC;
```

This single query provides all the data needed for the table.

### Alternative: In-Memory Grouping

If the SQL grouping is complex due to NULL handling for git_branch:

1. Query all commands for the date range
2. Group in memory by `(working_dir, git_branch)`
3. Calculate min/max timestamps and count for each group
4. Sort and format

## Implementation Plan

### Phase 1: Basic Table Output

1. **Create `cmd/summary.go`**:
   - Command definition with `--date` flag
   - Date parsing logic (yesterday, today, YYYY-MM-DD)
   - Call database method
   - Format and display table

2. **Add DB method in `internal/db/db.go`**:
   - `GetContextSummary(startTime, endTime int64) []ContextSummary`
   - Returns slice of structs with: WorkingDir, GitBranch, CommandCount, FirstTime, LastTime

3. **Create `internal/summary/table.go`**:
   - `ContextSummary` struct:
     ```go
     type ContextSummary struct {
         WorkingDir   string
         GitBranch    *string  // nil for non-git dirs
         CommandCount int
         FirstTime    int64
         LastTime     int64
     }
     ```
   - `FormatTable(summaries []ContextSummary) string`:
     - Calculate column widths
     - Format durations as human-readable (Xh Ym, Xm, Xs)
     - Format time spans as HH:MM - HH:MM
     - Handle home directory expansion (show ~)
     - Return formatted table string

4. **Duration formatting**:
   - Hours and minutes for > 1 hour: "8h 12m"
   - Minutes only for < 1 hour: "25m"
   - Seconds only for < 1 minute: "45s"
   - "0s" for single command (same first/last time)

5. **Time span formatting**:
   - Use HH:MM format for compactness
   - Local timezone

6. **Directory formatting**:
   - Replace home directory path with ~
   - Truncate very long paths to fit table width
   - Show ellipsis for truncated paths

### Phase 2: Enhancements

1. **Add filters**:
   - `--min-commands N` - Only show contexts with N+ commands
   - `--min-duration Xm` - Only show contexts with X+ minutes of duration

2. **Add alternative groupings**:
   - `--by-dir` - Group only by directory (ignore branches)
   - `--by-branch` - Group only by branch (ignore directories)

3. **Add formatting options**:
   - `--format table` (default)
   - `--format csv` - CSV output for spreadsheet import
   - `--format json` - JSON output for programmatic use

## Empty State

If no commands found for the date:

```
Yesterday's Work Summary - 2026-01-14
=====================================

No commands found for this date.
```

## Single Context

If only one context exists:

```
Yesterday's Work Summary - 2026-01-14
=====================================

Directory                          Branch              Commands  Time Span       Duration
~/projects/shy                     main                      12  09:15 - 17:32   8h 17m

Total: 12 commands across 1 context
```

## Success Metrics

1. Command executes in < 500ms for typical day (< 500 commands)
2. Table is properly aligned and readable
3. Durations are accurately calculated
4. Contexts are sorted by time spent (longest first)
5. All contexts with commands are shown

## Advantages Over Design A

1. **Simplicity**: Single table instead of hierarchical nested structure
2. **Scannable**: Easy to see at a glance where time was spent
3. **Fast**: Single SQL query with GROUP BY instead of fetching all commands
4. **Compact**: Fits in one screen for most days
5. **Focused**: Shows what matters (time allocation) without command-level detail

## Use Cases

- Daily standup preparation: "I spent 8 hours on the shy project and 2 hours on webapp"
- Time tracking: Quick overview of context switches throughout the day
- Work pattern analysis: Identify how fragmented your work was
- Retrospectives: Review time allocation across projects

## Future Considerations

### Add Command Details on Demand

Could add a `--expand <context>` flag to show commands for specific context:

```bash
shy summary --expand "~/projects/shy:main"
```

This would show a detailed view of just that context, similar to Design A's command listings.

### Add Visual Time Timeline

Could add an ASCII timeline showing when each context was active. Two orientations possible:

#### Horizontal Timeline (Time Left-to-Right)

Time flows left to right, contexts stacked vertically:

```
               08:00      12:00      16:00      20:00
shy:main       ████████   ░░░░░░░░   ░░░░░░░░   ████░░░░
webapp:auth    ░░░░░░░░   ████████   ████░░░░   ░░░░░░░░
dotfiles       ░░░░░░░░   ░░░░░░░░   ░░██░░░░   ░░░░░░░░
```

**Advantages:**
- Natural reading direction (left to right)
- Shows multiple contexts simultaneously for comparison
- Easy to see overlapping work periods
- Compact vertical space

#### Vertical Timeline (Time Top-to-Bottom)

Time flows top to bottom, contexts arranged horizontally:

```
Time   shy:main    webapp:auth  dotfiles
08:00  ████        ░░░░         ░░░░
09:00  ████        ░░░░         ░░░░
10:00  ████        ████         ░░░░
11:00  ░░░░        ████         ░░░░
12:00  ░░░░        ████         ░░░░
13:00  ░░░░        ████         ░░░░
14:00  ░░░░        ████         ██░░
15:00  ░░░░        ████         ░░░░
16:00  ░░░░        ████         ░░░░
17:00  ░░░░        ░░░░         ░░░░
18:00  ░░░░        ░░░░         ░░░░
19:00  ████        ░░░░         ░░░░
20:00  ████        ░░░░         ░░░░
21:00  ████        ░░░░         ░░░░
```

**Advantages:**
- Chronological flow matches natural time progression
- Easier to see activity patterns by hour of day
- More precise time resolution possible
- Better for long-duration contexts

#### Implementation Notes

- Use block characters: `█` (active), `░` (inactive)
- Each character represents a time bucket (30min or 1hr depending on orientation)
- Color coding could be added for different contexts (if terminal supports)
- Could show command count with different block densities:
  - `█` = many commands (5+)
  - `▓` = moderate (2-4)
  - `▒` = few (1)
  - `░` = none (0)

### Multi-Day Summaries

Weekly/monthly views showing time allocation trends:

```bash
shy summary --week
shy summary --range 2026-01-01 2026-01-31
```

This would show daily totals per context across the date range.
