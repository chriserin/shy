# Yesterday's Work Summary - Design Document

## Overview

A feature to generate structured summaries of shell command activity from the previous day's work, organized by repository/directory and branch, with timelines and command analysis.

## Goals

1. Provide developers with a quick recap of yesterday's work across multiple projects
2. Group commands by meaningful contexts (directory/repo, then branch)
3. Highlight important commands (long-running, failures, unique operations)
4. Show rough timelines of when work occurred in each context
5. Support daily standup preparation and work journaling

## Command Structure

### Primary Command

```bash
shy summary [options]
```

### Options

- `--date <date>` - Date to summarize (default: yesterday). Accepts:
  - `yesterday` (default)
  - `today`
  - `YYYY-MM-DD` (e.g., `2026-01-14`)
- `--all-commands` - Display all commands in each time bucket with timestamps (default: show summary only)

### Examples

```bash
# Yesterday's summary with all commands listed
shy summary --yesterday --all-commands

# Specific date with all commands
shy summary --date 2026-01-14 --all-commands

# Today's work
shy summary --date today --all-commands
```

## Data Model

### Database Query Strategy

The summary needs to aggregate data across multiple dimensions:

1. **Date Range Filter**: Commands where `timestamp` falls within the target day (00:00:00 to 23:59:59)
2. **Context Grouping**: Group by `(working_dir, git_repo, git_branch)`
3. **Timeline Buckets**: Divide day into time ranges (e.g., morning/afternoon/evening)
4. **Command Classification**: Categorize commands as important, unique, or routine

### Key Queries

#### 1. Get All Commands for Date

```sql
SELECT
    id,
    timestamp,
    exit_status,
    duration,
    command_text,
    working_dir,
    git_repo,
    git_branch
FROM commands
WHERE timestamp >= ? AND timestamp < ?
ORDER BY timestamp ASC;
```

#### 2. Group by Context (in-memory after fetch)

After fetching, group commands by:

- Primary key: `git_repo` (if available) or `working_dir`
- Secondary key: `git_branch` (if in a git repo)

This creates a hierarchical structure:

```
Repository/Directory
├── Branch A
│   ├── Morning (6am-12pm): N commands
│   ├── Afternoon (12pm-6pm): M commands
│   └── Evening (6pm-12am): K commands
└── Branch B
    └── ...
```

## Command Classification Logic

### Important Commands

A command is "important" if it meets any of these criteria:

1. **Long-running**: `duration > 10000` (10 seconds)
2. **Failed**: `exit_status != 0`
3. **Build/Deploy/Test**: Matches patterns like:
   - `go build`, `go test`, `npm run`, `make`, `docker build`
   - `git push`, `git merge`, `git rebase`
   - `deploy`, `release`, `publish`
4. **File modifications**: `rm`, `mv`, `cp` with important paths
5. **System changes**: `sudo`, `apt`, `brew`, `systemctl`

### Unique Commands

Commands that appear only once or infrequently (< 3 times) in the day and aren't trivial.

### Trivial Commands (to filter)

Common navigation and inspection commands (when `--include-trivial` is not set):

- `ls`, `cd`, `pwd`
- `cat`, `less`, `more`
- `echo`, `printf`
- `git status`, `git diff` (without important flags)
- `which`, `type`, `alias`

## Output Format

### Human-Readable Text Format

The output is designed for command-line display with hierarchical grouping.

#### With `--all-commands` flag (Phase 1 implementation)

```
Yesterday's Work Summary - 2026-01-14
========================================

/home/user/projects/shy (github.com/chris/shy)
  Branch: yesterdays-summary
    Morning (8:23am - 11:47am) - 23 commands
      08:23:15  git checkout -b yesterdays-summary
      08:24:02  vim docs/yesterday_summary_design.md
      08:45:31  go build -o shy .
      09:12:44  go test ./... -v
      09:15:02  git add .
      09:15:18  git commit -m "Add summary design doc"
      ...

    Afternoon (12:15pm - 4:35pm) - 15 commands
      12:15:23  shy fc -l
      12:18:45  vim cmd/summary.go
      ...

  Branch: main
    Evening (7:15pm - 9:32pm) - 12 commands
      19:15:12  git checkout main
      19:15:20  git merge yesterdays-summary
      19:15:45  go test ./...
      19:18:33  git push origin main
      ...

/home/user/projects/webapp
  No git repository
    Afternoon (2:10pm - 4:35pm) - 8 commands
      14:10:15  npm install
      14:12:33  npm run test
      14:15:42  rm -rf node_modules
      ...

Summary Statistics:
  Total commands: 58
  Unique contexts: 3 (2 repos, 1 non-repo dir)
  Branches worked on: 2
```

#### Without `--all-commands` flag (Future enhancement)

Shows only summary statistics per time bucket:

```
Yesterday's Work Summary - 2026-01-14
========================================

/home/user/projects/shy (github.com/chris/shy)
  Branch: yesterdays-summary
    Morning (8:23am - 11:47am) - 23 commands
    Afternoon (12:15pm - 4:35pm) - 15 commands

  Branch: main
    Evening (7:15pm - 9:32pm) - 12 commands

/home/user/projects/webapp
  No git repository
    Afternoon (2:10pm - 4:35pm) - 8 commands

Summary Statistics:
  Total commands: 58
  Unique contexts: 3 (2 repos, 1 non-repo dir)
  Branches worked on: 2
```

## Implementation Plan

### Phase 1: Minimal Viable Implementation (`--all-commands`)

**Goal**: Implement `shy summary --yesterday --all-commands` to display all commands grouped by context and time period.

1. **Create `cmd/summary.go`**:
   - Command definition with flags: `--date` (default: yesterday), `--all-commands`
   - Date parsing logic:
     - Handle `yesterday`, `today`, `YYYY-MM-DD`
     - Convert to Unix timestamp range (start of day 00:00:00 to 23:59:59)
   - Call to database layer to fetch commands
   - Pass commands to formatter

2. **Add DB method `GetCommandsByDateRange(startTime, endTime int64)` in `internal/db/db.go`**:
   - Query: `SELECT * FROM commands WHERE timestamp >= ? AND timestamp < ? ORDER BY timestamp ASC`
   - Return slice of `models.Command`

3. **Create `internal/summary/` package**:

   **`grouper.go`**: Context grouping logic
   - Function: `GroupByContext(commands []models.Command) map[string]map[string][]models.Command`
   - Primary grouping: By `git_repo` (if set) or `working_dir`
   - Secondary grouping: By `git_branch` (if set) or "No branch"
   - Return nested map structure for hierarchical display

   **`timeline.go`**: Time period bucketing
   - Function: `BucketByTimePeriod(commands []models.Command) map[string][]models.Command`
   - Time periods:
     - Morning: 6am-12pm (06:00:00 - 11:59:59)
     - Afternoon: 12pm-6pm (12:00:00 - 17:59:59)
     - Evening: 6pm-12am (18:00:00 - 23:59:59)
     - Night: 12am-6am (00:00:00 - 05:59:59)
   - Calculate first/last timestamp for each bucket
   - Return map of period name to commands

   **`formatter.go`**: Human-readable output
   - Function: `FormatSummary(grouped data, allCommands bool) string`
   - Hierarchical display:
     - Level 1: Repository/Directory (with git URL if available)
     - Level 2: Branch
     - Level 3: Time period (with time range and command count)
     - Level 4 (if `--all-commands`): Individual commands with HH:MM:SS timestamp
   - Summary statistics at the end

4. **Command timestamp formatting**:
   - Display time as HH:MM:SS (e.g., "08:23:15")
   - Use local timezone from Unix timestamp

5. **Empty state handling**:
   - If no commands found for the date: "No commands found for [date]"
   - If all commands are in one bucket: Still show the bucket structure

### Phase 2: Command Classification and Filtering

**Goal**: Add intelligence to identify and highlight important commands (future enhancement).

1. **Important command detection**:
   - Duration threshold check (> 10 seconds)
   - Exit status check (non-zero)
   - Pattern matching against known important command patterns
   - Use regex for flexible matching

2. **Unique command detection**:
   - Build frequency map of commands
   - Filter for commands appearing ≤ 2 times
   - Exclude trivial commands

3. **Trivial command filtering**:
   - Maintain list of trivial command patterns
   - Option to disable filtering with `--include-trivial`

4. **Summary view without `--all-commands`**:
   - Display only summary statistics per bucket
   - Show important/unique commands as highlights

## File Structure

```
shy/
├── cmd/
│   └── summary.go           # Command definition and flag handling
├── internal/
│   └── summary/
│       ├── grouper.go       # Context grouping logic (repo/dir → branch)
│       ├── timeline.go      # Time period bucketing
│       └── formatter.go     # Human-readable text output
└── docs/
    └── yesterday_summary_design.md  # This document
```

Note: `classifier.go` will be added in Phase 2 for command classification.

## Testing Strategy

### Phase 1 Tests

1. **Unit tests for `internal/summary/grouper.go`**:
   - Group commands from single repo, single branch
   - Group commands from multiple repos
   - Group commands from mixed git/non-git directories
   - Handle nil git_repo and git_branch properly

2. **Unit tests for `internal/summary/timeline.go`**:
   - Bucket commands across all four time periods
   - Handle commands at bucket boundaries (5:59:59, 6:00:00, etc.)
   - Handle commands spanning midnight
   - Handle single time period (all commands in morning)
   - Handle empty time periods

3. **Unit tests for `internal/summary/formatter.go`**:
   - Format single context with all commands
   - Format multiple contexts
   - Format with empty buckets
   - Verify timestamp formatting (HH:MM:SS)
   - Verify hierarchical indentation

4. **Integration tests in `cmd/summary_test.go`**:
   - Create in-memory DB with sample data spanning a day
   - Test `shy summary --yesterday --all-commands`
   - Test `shy summary --date 2026-01-14 --all-commands`
   - Verify output structure and correctness

5. **Test data scenarios**:
   - Single repo, single branch, one time period
   - Multiple repos, multiple branches, scattered across day
   - Mixed git/non-git directories
   - Commands at time period boundaries
   - Empty day (no commands) → should show appropriate message
   - Commands in only one time period

## Future Considerations

### Phase 2 Enhancements
- Summary view without `--all-commands` showing only important/unique commands
- Command classification (important, unique, trivial)
- Filtering options: `--include-trivial`, `--min-commands`

### Multi-day Summaries
```bash
shy summary --week        # Last 7 days
shy summary --range 2026-01-01 2026-01-07
```

### Additional Output Formats
- Markdown format for journaling: `shy summary --format markdown`
- JSON format for programmatic use: `shy summary --format json`

## Design Decisions

1. **Working directory normalization**:
   - Use exact `working_dir` from database as primary key
   - Display `git_repo` URL alongside directory for context when available
   - Do not attempt to normalize subdirectories within a repo (keep them separate)

2. **Branch switching**:
   - If commands span multiple branches in same repo, create separate sections
   - This shows parallel work and branch context clearly

3. **Time period bucketing**:
   - Hardcoded periods for Phase 1: Morning (6am-12pm), Afternoon (12pm-6pm), Evening (6pm-12am), Night (12am-6am)
   - Use local timezone for display
   - Show "start - end" time range based on actual first/last command in bucket

4. **Context grouping order**:
   - Sort contexts alphabetically by working_dir
   - Within a context, sort branches alphabetically
   - Within a branch, show time periods chronologically (Morning → Afternoon → Evening → Night)

## Success Metrics

### Phase 1
1. Command executes in < 1 second for typical day (< 500 commands)
2. All commands are correctly grouped by context (repo/dir) and branch
3. All commands are correctly bucketed by time period
4. Output is readable with proper hierarchical indentation
5. Timestamps are correctly formatted and displayed

### Phase 2
1. Successfully identifies important commands with 90%+ precision
2. Summary view (without `--all-commands`) is concise and actionable for standups
