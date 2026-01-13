# Mode 4 Design Summary: PPID-Based Session Stack

## Core Concept

Each shell session maintains its own database stack, tracked by the shell's process ID (PPID). The stack is stored in a simple text file, with no need for environment variables or shell wrapper functions.

## How It Works

### 1. Session File Location
```
$XDG_CACHE_HOME/shy/sessions/{ppid}.txt
```
Default: `~/.cache/shy/sessions/{ppid}.txt`

### 2. File Format
Plain text, one database path per line:
- **Line 1:** Current active database
- **Lines 2+:** Stack in LIFO order

**Example:**
```
/tmp/current.db
/tmp/previous.db
/home/user/.local/share/shy/history.db
```

### 3. Database Resolution
Every `shy` command:
1. Calls `os.Getppid()` to get parent shell's PID
2. Reads `~/.cache/shy/sessions/{ppid}.txt`
3. Uses line 1 as current database
4. Falls back to default database if file doesn't exist

### 4. Push Operation (`shy fc -p /tmp/new.db`)
1. Get PPID
2. Read session file (or create if doesn't exist)
3. Normalize new path (expand `~`, resolve relative, add `.db`)
4. Create new database if needed
5. Prepend current database to stack
6. Write new database as line 1

**Before:**
```
/tmp/old.db
~/.local/share/shy/history.db
```

**After:**
```
/tmp/new.db
/tmp/old.db
~/.local/share/shy/history.db
```

### 5. Pop Operation (`shy fc -P`)
1. Get PPID
2. Read session file
3. Error if only 1 line (can't pop default)
4. Remove line 1, shift remaining up

**Before:**
```
/tmp/current.db
/tmp/previous.db
~/.local/share/shy/history.db
```

**After:**
```
/tmp/previous.db
~/.local/share/shy/history.db
```

### 6. Cleanup (`zshexit` hook)
When shell exits, the hook runs:
```bash
command shy cleanup-session $$
```
This deletes the session file, preventing stale PID files.

## Key Advantages

### No Shell Wrapper Required
- Works immediately without any wrapper functions
- Can't be bypassed (uses PPID directly)
- No eval needed
- No environment variable management

### Per-Session Isolation
- Each terminal has its own stack
- No cross-session interference
- Different projects in different terminals

### Simple Implementation
- Plain text files (easy to debug)
- Fast file I/O (<1ms)
- No database schema changes
- Shell-agnostic design

### Automatic Cleanup
- zshexit hook removes file on exit
- No stale session files accumulate
- Cache directory appropriate for transient data

## Implementation Components

### 1. Session Package (`internal/session/`)
- `stack.go` - Push/pop logic
- `file.go` - File read/write with locking
- `path.go` - Path normalization

### 2. CLI Integration
- `cmd/fc.go` - `-p [path]` flag for push, `-P` flag for pop
- `cmd/cleanup_session.go` - `shy cleanup-session <pid>` command

### 3. Shell Integration (`cmd/integration_scripts/zsh_use.sh`)
- `zshexit()` hook for cleanup

### 4. Database Resolution
- Every shy command checks session file first
- Transparent to users
- No configuration needed

## Example Workflows

### Project-Specific History
```bash
cd ~/projectA
shy fc -p ./.shy_history.db  # Start project-local history

# Work in projectA
git commit -m "feature"
make build

# Switch projects
cd ~/projectB
shy fc -p ./.shy_history.db  # Different project history

# Work in projectB
npm test

# Done with projectB
shy fc -P  # Back to projectA context

# Done with projectA
shy fc -P  # Back to global history
```

### Demo Recording
```bash
# Start clean history for demo
shy fc -p /tmp/demo.db

# Record demo
echo "Step 1: Install"
echo "Step 2: Configure"

# Export for documentation
shy fc -W demo_commands.txt

# Return to normal
shy fc -P
```

### Multiple Terminals
```bash
# Terminal 1
shy fc -p ~/work/.shy_history.db

# Terminal 2 (independent)
shy fc -p ~/personal/.shy_history.db

# Each terminal has its own stack
# No interference between them
```

## Error Handling

- **Pop with empty stack:** Exit code 1, clear error message
- **Invalid path:** Permission/validation error before push
- **Missing session file:** Use default database (not an error)
- **Corrupted file:** Clear error, suggest manual cleanup
- **Concurrent access:** File locking prevents corruption

## Testing Strategy

1. **Unit tests:** Session file operations, path normalization
2. **Integration tests:** Push/pop with actual databases
3. **Scenario tests:** All workflows in feature file
4. **Cleanup tests:** Verify zshexit removes files
5. **Isolation tests:** Multiple PIDs don't interfere

## Migration Path

- **Existing users:** No changes needed, feature is additive
- **New users:** Works immediately after sourcing zsh_use.sh
- **No database migrations:** Session files are separate from history DB
- **Backward compatible:** Old shy versions ignore session files

## Performance

- Session file read: <1ms typical
- Session file write: <1ms with atomic rename
- File locking: Minimal overhead
- No network I/O
- No database queries for stack operations

## Future Enhancements (Not in Phase 1)

- `-a` auto-pop flag (requires shell integration)
- Bash/fish support (need hooks similar to zshexit)
- Stack depth limits (if needed)
- Session file cleanup daemon (if zshexit insufficient)
