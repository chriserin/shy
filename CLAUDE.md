# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`shy` is a shell history tracker that stores command history in SQLite with rich metadata (timestamps, exit codes, git context, duration, etc.). It provides a modern alternative to traditional shell history with features like database switching, session management, and zsh-autosuggestions integration.

## Build and Test Commands

### Building
```bash
# Build for current platform
go build -o shy .

# Build for all platforms (linux/darwin/windows, amd64/arm64)
./scripts/build.sh

# Build with specific version
./scripts/build.sh v0.2.0
```

### Testing
```bash
# Run all tests
go test ./... -v

# Run tests for specific package
go test ./internal/db -v
go test ./cmd -v

# Run specific test
go test ./cmd -run TestFcListMode -v

# Run tests with coverage
go test ./... -cover
```

### Release Scripts
- `scripts/release.sh <version>` - Create GitHub release with binaries
- `scripts/release-all.sh` - Full release process (build, package, checksums)
- `scripts/version.sh` - Bump version in code

## Architecture

### Package Structure

```
shy/
├── cmd/               # CLI commands (Cobra-based)
│   ├── root.go       # Root command with --db flag resolution
│   ├── fc.go         # Main history editing/listing (fc builtin compatibility)
│   ├── history.go    # Alias for fc -l
│   ├── insert.go     # Insert commands into DB
│   ├── init.go       # Shell integration script generation
│   └── integration_scripts/  # Embedded zsh hooks
├── internal/
│   ├── db/           # SQLite abstraction layer
│   ├── git/          # Git context detection
│   └── session/      # Session stack management (fc -p/-P)
├── pkg/models/       # Domain models (Command struct)
└── main.go           # Entry point
```

### Key Components

#### Database Layer (`internal/db`)
- Uses SQLite with WAL mode for concurrent writes from multiple shell sessions
- Schema migrations via table recreation with exclusive locks
- Main operations: `InsertCommand()`, `GetCommandsByRange()`, `LikeRecent()`
- Session filtering: Commands track `source_pid` and `source_active` for per-session isolation
- Database path: `$XDG_DATA_HOME/shy/history.db` (default: `~/.local/share/shy/history.db`)

#### Session Management (`internal/session`)
- Maintains a stack of active databases per shell session in `$XDG_CACHE_HOME/shy/sessions/{ppid}.txt`
- `PushDatabase()` / `PopDatabase()` enable switching between project-specific history databases
- `GetCurrentDatabase()` reads current DB from line 1 of session file
- `NormalizeDatabasePath()` handles `~` expansion and auto-appends `.db` extension

#### Command Model (`pkg/models`)
- `Command` struct with optional fields using pointers (e.g., `*string`, `*int64`) for NULL representation
- Core fields: ID, Timestamp, ExitStatus, CommandText, WorkingDir
- Optional: GitRepo, GitBranch, Duration, SourceApp, SourcePid, SourceActive

### Manual Flag Parsing Pattern

`fc.go` and `history.go` use `DisableFlagParsing: true` to handle negative numbers (e.g., `shy fc -10`) as positional arguments rather than flags. Custom parsing logic in `parseFcArgsAndFlags()` and `parseHistoryArgsAndFlags()` manually extracts flags vs arguments.

### Root Command Hook

`root.go` defines `PersistentPreRunE` that:
1. Reads `--db` flag from session file if not explicitly provided
2. Enables dynamic database switching per shell session via session stack

### Zsh Integration

`shy init zsh` outputs shell integration scripts:
- **zsh_record.sh**: `preexec`/`precmd` hooks to auto-insert commands with metadata
- **zsh_use.sh**: Ctrl-R keybinding to list history with `shy fc -l`
- **shy_autosuggest.zsh**: Custom strategies for zsh-autosuggestions plugin
  - `shy_history`: Simple prefix matching with session and pwd filters
- **fzf.zsh** (`shy init zsh --fzf`): fzf history widget integration
  - Uses `shy fzf` command as data source
  - Replaces fzf's default history widget
  - Supports multi-select and automatic deduplication
  - All filtering done interactively in fzf
  - Bind with: `bindkey '^R' shy-fzf-history-widget`

Configuration via environment variables:
- `SHY_DISABLE=1` - Temporarily disable tracking
- `SHY_DB_PATH` - Custom database path
- `SHY_SESSION_PID` - Exported as `$$` for session filtering

### Database Switching Workflow

```bash
# Push new database onto stack
shy fc -p /path/to/project.db

# Now all commands use project.db
shy fc -l

# Pop back to previous database
shy fc -P
```

Session stack is stored per-shell in `$XDG_CACHE_HOME/shy/sessions/{ppid}.txt`. Stack is cleaned up on shell exit via `zshexit` hook.

## Development Patterns

### Adding New Commands

1. Create new file in `cmd/` (e.g., `cmd/mycommand.go`)
2. Define command struct and init function following Cobra pattern
3. Add command to root in `init()` function: `rootCmd.AddCommand(mycommandCmd)`
4. Use `db.New(dbPath)` to open database (path from `--db` flag or session file)
5. Always defer `db.Close()`

### Database Queries

For queries with filtering/patterns, prefer existing methods:
- `GetCommandsByRange(first, last int)` - Range by event numbers
- `GetCommandsByRangeWithPattern(first, last int, pattern string)` - With LIKE matching
- `GetCommandsByRangeInternal(first, last int, pid int)` - Session-filtered
- `LikeRecent(prefix string, opts LikeRecentOptions)` - Prefix matching with filters

For new query types, add methods to `internal/db/db.go` following existing patterns.

### Testing Database Operations

Tests use in-memory SQLite (`:memory:`):
```go
db, err := db.NewWithOptions(":memory:", db.Options{})
require.NoError(t, err)
defer db.Close()
```

### Handling Optional Fields

Use pointer types for optional command metadata:
```go
duration := int64(1234)
cmd := &models.Command{
    Duration: &duration,  // Converts to SQL value
    GitRepo:  nil,        // Converts to SQL NULL
}
```

## Important Implementation Details

### Concurrent Safety
- SQLite WAL mode enables concurrent writes from multiple shell sessions
- `busy_timeout` set to 5 seconds to handle contention
- Session files use atomic writes (temp file + rename)

### XDG Base Directory Compliance
- Data: `$XDG_DATA_HOME/shy/` (default: `~/.local/share/shy/`)
- Cache: `$XDG_CACHE_HOME/shy/` (default: `~/.cache/shy/`)
- Error logs: `$XDG_DATA_HOME/shy/error.log`

### Git Context Auto-Detection
`internal/git/detect.go` walks up directory tree to find `.git/`, then queries:
- Repository URL: `git config --get remote.origin.url`
- Branch: `git rev-parse --abbrev-ref HEAD`

Auto-detection skips if `--git-repo` or `--git-branch` explicitly provided.

### Command Formatting (fc -l output)
Default format: `{event_num} {timestamp} {elapsed} {command}`
- Event numbers start at 1 (database ID)
- Timestamps: ISO (default), US, EU, or custom strftime format
- Elapsed time: Human-readable duration (e.g., "2h 15m 3s")
- Use `-n` flag to suppress event numbers

## Common Tasks

### Add a new filter to like-recent
1. Add flag to `cmd/like_recent.go` (e.g., `--my-filter`)
2. Add field to `db.LikeRecentOptions` struct in `internal/db/db.go`
3. Update SQL WHERE clause in `db.LikeRecent()` to apply filter
4. Add tests in `cmd/like_recent_test.go`

### Add new metadata field to Command
1. Add field to `models.Command` struct (use pointer for optional)
2. Add column to schema in `internal/db/schema.go`
3. Update `InsertCommand()` to include new field in INSERT
4. Update query methods to SELECT new field
5. Add flag to `cmd/insert.go` for manual insertion
6. Update zsh integration script if auto-captured

### Modify fc behavior
`cmd/fc.go` is 1149 lines implementing zsh/bash `fc` builtin compatibility. Key modes:
- List mode (`fc -l` / `history`): Display history with formatting
- Edit mode (`fc` without `-l`): Open commands in editor, execute after save
- Range selection (inclusive):
  - `fc -l 10 5` or `history 10 5` - Prints 6 commands (IDs 10, 9, 8, 7, 6, 5) in reverse chronological order (10 first, 5 last)
  - `fc -l 5 10` or `history 5 10` - Prints 6 commands (IDs 5, 6, 7, 8, 9, 10) in chronological order (5 first, 10 last)
  - `fc -l -10` - Last 10 commands
- Filtering: `--internal` (session only), `--local` (current dir), `--match` (pattern)

### fzf Integration
`cmd/fzf.go` provides `shy fzf` command for fzf history widget integration:
- **Output format**: `event_number<TAB>command<NULL>` (tab-separated, null-terminated)
- **Deduplication**: SQL-based (uses window functions, keeps most recent occurrence)
- **Order**: Reverse chronological (most recent first)
- **No options**: All filtering is done interactively within fzf itself
- **Database method**: `GetCommandsForFzf()` in `internal/db/db.go` uses `ROW_NUMBER() OVER (PARTITION BY command_text)` for efficient deduplication
- **Usage**: Data source for `shy-fzf-history-widget` in `cmd/integration_scripts/fzf.zsh`
