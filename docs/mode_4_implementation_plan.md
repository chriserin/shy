# Mode 4 Implementation Plan: History Stack Operations

## Overview

Implement `fc -p` (push) and `fc -P` (pop) commands to enable database switching and isolated history contexts.

## Core Components

### 1. Session Stack Management

**Location:** `internal/session/stack.go` (new file)

**Session File Format:**
- Location: `$XDG_CACHE_HOME/shy/sessions/{ppid}.txt` (default: `~/.cache/shy/sessions/{ppid}.txt`)
- Format: Plain text, one database path per line
- Line 1: Current active database path
- Lines 2+: Stack in LIFO order (most recent first)

**Example session file:**
```
/tmp/current.db
/tmp/previous.db
~/.local/share/shy/history.db
```

**Key Functions:**

- `PushDatabase(ppid int, newPath string) error`
  - Validate and normalize the new database path
  - Auto-append `.db` extension if missing
  - Create new database if it doesn't exist
  - Read existing session file for this PPID
  - Prepend current DB to stack, write new DB as line 1

- `PopDatabase(ppid int) (string, error)`
  - Read session file for this PPID
  - Return error if stack has only 1 line (exit code 1)
  - Remove line 1, shift remaining lines up
  - Return new line 1 as previous database

- `GetCurrentDatabase(ppid int) (string, error)`
  - Read line 1 from session file
  - Returns default database if file doesn't exist

- `GetSessionFilePath(ppid int) string`
  - Returns `$XDG_CACHE_HOME/shy/sessions/{ppid}.txt`
  - Creates directory if it doesn't exist

### 2. Session File Operations

**Location:** `internal/session/file.go` (new file)

**Operations:**

- `readSessionFile(ppid int) ([]string, error)`
  - Read all lines from session file
  - Returns empty slice if file doesn't exist
  - Handles file locking for concurrent access

- `writeSessionFile(ppid int, lines []string) error`
  - Write lines to session file atomically
  - Creates parent directory if needed
  - Uses file locking to prevent race conditions

- `CleanupSession(ppid int) error`
  - Deletes session file for given PPID
  - Called from zshexit hook when shell exits
  - Fails silently if file doesn't exist

### 3. CLI Integration

**Location:** `cmd/fc.go` and `cmd/cleanup_session.go`

**fc Command Flags:**

- `-p [filename]` - Push current database, start using new one
- `-P` - Pop back to previous database

**fc Command Logic:**

```go
ppid := os.Getppid()  // Get parent shell's PID

if pushFlag {
    // Handle fc -p [filename]
    // Validate filename argument
    // Call session.PushDatabase(ppid, newPath)
}

if popFlag {
    // Handle fc -P
    // Call session.PopDatabase(ppid)
    // Error if no previous DB exists (exit code 1)
}
```

**cleanup-session Command:**

**Location:** `cmd/cleanup_session.go` (new file)

**Usage:** `shy cleanup-session <pid>`

**Logic:**
```go
func cleanupSessionCommand(pid int) error {
    // Call session.CleanupSession(pid)
    // Deletes session file for given PID
    // Fails silently if file doesn't exist
    // Always returns success (exit code 0)
}
```

### 4. Path Handling

**Location:** `internal/database/path.go` (new file)

**Functions:**

- `NormalizeDatabasePath(path string) (string, error)`
  - Expand `~` to home directory
  - Resolve relative paths to absolute
  - Auto-append `.db` if no extension present
  - Validate parent directory exists/is writable

- `ValidateDatabasePath(path string) error`
  - Check permissions
  - Verify path is not a directory
  - Ensure parent directory exists

### 5. Shell Integration

**Location:** `zsh_use.sh`

**Shell Hook:**

```bash
# Cleanup session file on shell exit
zshexit() {
    command shy cleanup-session $$ 2>/dev/null
}
```

**How It Works:**

1. Every `shy` command uses `os.Getppid()` to get the parent shell's PID
2. Reads current database from `~/.cache/shy/sessions/{ppid}.txt` line 1
3. If file doesn't exist, uses default database
4. No wrapper needed - session file is the single source of truth
5. On shell exit, zshexit hook cleans up the session file

**Advantages:**
- No shell wrapper required
- No environment variable management
- Can't be bypassed
- Works automatically with all shy commands

## Implementation Steps

### Phase 1: Foundation (Session Layer)

1. ⬜ Create `internal/session/` package
2. ⬜ Implement session file read/write operations
3. ⬜ Add file locking for concurrent access
4. ⬜ Write unit tests for file operations
5. ⬜ Implement XDG_CACHE_HOME path resolution

### Phase 2: Stack Logic

1. ⬜ Create `session/stack.go`
2. ⬜ Implement `PushDatabase(ppid, newPath)` function
   - Path normalization
   - Database creation
   - Session file update (prepend current to stack)
3. ⬜ Implement `PopDatabase(ppid)` function
   - Read session file
   - Return error if empty stack
   - Shift stack up
4. ⬜ Implement `GetCurrentDatabase(ppid)` function
5. ⬜ Write unit tests for push/pop operations
6. ⬜ Test nested push/pop scenarios

### Phase 3: Path Handling

1. ⬜ Create `database/path.go`
2. ⬜ Implement path normalization
   - Tilde expansion
   - Relative to absolute
   - Extension handling
3. ⬜ Implement path validation
4. ⬜ Write unit tests for all edge cases

### Phase 4: CLI Integration

1. ⬜ Add `-p` and `-P` flags to `fc` command
2. ⬜ Implement flag parsing and validation
3. ⬜ Connect to stack operations
4. ⬜ Add appropriate error messages
5. ⬜ Create `cleanup-session` command
6. ⬜ Update help text and documentation

### Phase 5: Shell Integration

1. ⬜ Add zshexit hook to `zsh_use.sh` (calls `shy cleanup-session $$`)
2. ⬜ Ensure all shy commands check session file for current DB
3. ⬜ Test session cleanup on shell exit
4. ⬜ Document shell integration setup

### Phase 6: Testing

1. ⬜ Implement Cucumber step definitions for scenarios
2. ⬜ Run all scenarios from `mode_4_fc_push_pop.feat`
3. ⬜ Integration tests with other fc commands
4. ⬜ Error case testing
5. ⬜ Performance testing with large stacks

### Phase 7: Documentation

1. ⬜ Update README with push/pop examples
2. ⬜ Add usage scenarios to docs
3. ⬜ Document environment variable behavior

## Technical Decisions

### 1. Stack Persistence (Per-Session)

**Decision:** Store stack in per-session text files using PPID tracking
**Rationale:**

- Stack is per-shell-session, not shared across all sessions
- No wrapper function required - uses PPID (parent process ID)
- Simple text format (one path per line)
- Stored in `XDG_CACHE_HOME` for proper cleanup
- Can't be bypassed (PPID is always available)
- Shell-agnostic solution

### 2. Auto-Extension

**Decision:** Auto-append `.db` if no extension present
**Rationale:**

- Matches user expectation
- Prevents confusion
- Allows explicit override with `.db` extension

### 3. Pop Behavior

**Decision:** Popping at bottom returns error (exit code 1)
**Rationale:**

- Prevents accidental state corruption
- Clear feedback to user
- Matches zsh behavior

### 4. Auto-Pop (-a flag)

**Decision:** Not implemented in Phase 1
**Rationale:**

- Requires shell integration
- Complex to implement correctly
- Low priority feature
- Can be added later if needed

### 5. Session Isolation

**Decision:** Each shell session has its own independent stack
**Rationale:**

- Stack is tracked by shell PID, isolated per terminal
- Different terminals can have different database contexts
- More intuitive for project-specific workflows
- Prevents cross-session interference
- Session file cleaned up on shell exit via zshexit hook

## Error Handling

### Error Cases to Handle:

1. ✅ Pop with no previous database
2. ✅ Push to invalid/restricted path
3. ✅ Push with empty filename
4. ✅ Session file read/write errors
5. ✅ Permission errors
6. ✅ Database creation failures
7. ✅ Concurrent access (file locking)
8. ✅ Stale session files (PID doesn't exist)

### Error Messages:

- Pop with no stack: "No previous database to pop to"
- Invalid path: "Cannot create database at {path}: {reason}"
- Permission denied: "Permission denied: {path}"
- Session file error: "Failed to read session state: {reason}"

## Testing Strategy

### Unit Tests:

- Session file read/write operations
- Path normalization (all edge cases)
- Push/pop logic with PPID isolation
- File locking behavior
- Error conditions

### Integration Tests:

- Push followed by commands
- Pop restores previous state
- Nested push/pop
- Interaction with other fc commands

### Scenario Tests:

- All scenarios in `mode_4_fc_push_pop.feat`
- Project-specific workflow
- Demo recording workflow
- Error cases

## Deployment Strategy

### New Installation:

1. Session directory created automatically: `$XDG_CACHE_HOME/shy/sessions/`
2. Shell integration added to `zsh_use.sh`
3. No database schema changes needed

### Shell Integration Setup:

Users already source `zsh_use.sh` for history features. The zshexit hook for cleanup will be automatically included.

### Backward Compatibility:

- No database migrations needed
- Feature is purely additive
- Works independently of existing fc commands
- No shell wrapper required - works out of the box

## Performance Considerations

1. **Session file operations:** Fast text file I/O (<1ms typical)
2. **File locking:** Minimal overhead for concurrent access
3. **Path normalization:** Minimal overhead
4. **No stack depth limit:** Could add if needed
5. **Session cleanup:** O(1) file deletion on shell exit
6. **XDG path resolution:** Cached after first access

## Open Questions

1. ~~Should we limit stack depth?~~ → No limit for now
2. ~~What if pushed database is deleted?~~ → Pop will fail with clear error
3. ~~Should push overwrite existing database?~~ → No, merge/append to existing
4. ~~Concurrent push/pop from different shells?~~ → Each shell has isolated session by PPID
5. ~~How to clean up stale session files?~~ → zshexit hook removes file on shell exit
6. ~~PID reuse after reboot?~~ → Low risk; cleaned on shell exit anyway

## Success Criteria

- ✅ All scenarios in feature file pass
- ✅ No regression in existing fc commands
- ✅ Clear error messages for all edge cases
- ✅ Documentation complete
- ✅ Performance acceptable (<100ms for push/pop)
