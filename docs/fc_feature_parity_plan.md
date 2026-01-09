# Plan: shy fc Feature Parity with zsh Builtin fc

**Document Version:** 1.0
**Date:** 2026-01-08
**Status:** Draft for Review

---

## Executive Summary

This document outlines a plan to achieve complete feature parity between `shy fc` and zsh's builtin `fc` command. The zsh `fc` command is a powerful history manipulation tool with multiple operation modes. Currently, `shy fc` implements only the listing mode (`fc -l`) with basic functionality.

**Key Goals:**

1. Implement all fc operation modes (list, edit-and-execute, history file operations, history stack operations)
2. Support all zsh fc flags and options
3. Maintain backward compatibility with existing shy commands
4. Provide clear migration path for users
5. Enable database switching/merging through fc mechanisms

**Implementation Complexity:** High (8-10 weeks estimated)

---

## Current State Analysis

### Existing shy Commands Overview

| Command            | Purpose                              | Relationship to fc                         |
| ------------------ | ------------------------------------ | ------------------------------------------ |
| `shy fc`           | Process command history (fc builtin) | **Primary implementation target**          |
| `shy history`      | Display command history              | Alias for `fc -l`, shares flags/logic      |
| `shy list`         | List recent commands                 | Overlaps with `fc -l`, uses custom `--fmt` |
| `shy list_all`     | List all commands                    | Similar to `fc -l` without range limit     |
| `shy insert`       | Insert command into database         | Backend for history recording              |
| `shy last_command` | Get most recent command              | Could use `fc -1`                          |
| `shy like_recent`  | Get recent command by prefix         | Could use `fc -m pattern`                  |

### Current shy fc Implementation

**Implemented Features:**

- ✅ `-l` (list mode)
- ✅ `-n` (no line numbers)
- ✅ `-r` (reverse order)
- ✅ `-d` (display timestamps)
- ✅ `-i` (ISO8601 format)
- ✅ `-f` (US format)
- ✅ `-E` (European format)
- ✅ `-t fmt` (custom strftime format)
- ✅ `-D` (duration display)
- ✅ `--last N` (convenience for `-N`)
- ✅ Range parsing: `[first [last]]` with numbers and string matching
- ✅ Negative offsets (e.g., `-10`)
- ✅ String-based event lookup

**Not Implemented:**

- ❌ Edit-and-execute mode (default fc behavior without `-l`)
- ❌ `-e editor` (specify editor)
- ❌ `-s` (re-execute without editing, equivalent to `-e -`)
- ❌ `-m pattern` (filter by pattern)
- ❌ `-L` (local events only)
- ❌ `-I` (internal events only)
- ❌ `old=new` substitutions
- ❌ `-p` (push history stack)
- ❌ `-P` (pop history stack)
- ❌ `-R` (read history file)
- ❌ `-W` (write history file)
- ❌ `-A` (append to history file)

---

## Complete Feature Comparison

### Mode 1: List Mode (`fc -l`)

| Feature             | zsh fc | shy fc | Implementation Status | Priority |
| ------------------- | ------ | ------ | --------------------- | -------- |
| `-l` flag           | ✅     | ✅     | Complete              | -        |
| Range: `first last` | ✅     | ✅     | Complete              | -        |
| Negative offsets    | ✅     | ✅     | Complete              | -        |
| String matching     | ✅     | ✅     | Complete              | -        |
| `-n` no numbers     | ✅     | ✅     | Complete              | -        |
| `-r` reverse        | ✅     | ✅     | Complete              | -        |
| `-d` timestamps     | ✅     | ✅     | Complete              | -        |
| `-f` US format      | ✅     | ✅     | Complete              | -        |
| `-E` EU format      | ✅     | ✅     | Complete              | -        |
| `-i` ISO format     | ✅     | ✅     | Complete              | -        |
| `-t fmt` custom     | ✅     | ✅     | Complete              | -        |
| `-D` duration       | ✅     | ✅     | Complete              | -        |
| `-m pattern` filter | ✅     | ❌     | **Missing**           | HIGH     |
| `-L` local only     | ✅     | ❌     | **N/A** (see notes)   | LOW      |
| `-I` internal only  | ✅     | ❌     | **N/A** (see notes)   | LOW      |

**Notes:**

- `-L` and `-I` are specific to zsh's SHARE_HISTORY feature (multiple shells sharing one history file)
- shy has a single-database model; equivalent would be filtering by session/source
- Could implement as: `-L` = "current session only", `-I` = "not from file imports"
- old=new should also work in the context of `-l`

### Mode 2: Edit-and-Execute Mode (default without `-l`)

| Feature         | zsh fc | shy fc | Implementation Status | Priority |
| --------------- | ------ | ------ | --------------------- | -------- |
| Edit + execute  | ✅     | ❌     | **Missing**           | HIGH     |
| `-e editor`     | ✅     | ❌     | **Missing**           | HIGH     |
| `-s` quick exec | ✅     | ❌     | **Missing**           | MEDIUM   |
| `old=new` subst | ✅     | ❌     | **Missing**           | MEDIUM   |
| Range support   | ✅     | ❌     | **Missing**           | HIGH     |

**Implementation Challenges:**

- Requires executing shell commands from Go
- Security considerations: command injection
- Need to handle multi-line commands
- Editor integration (respect $EDITOR, $FCEDIT)
- Temporary file management

### Mode 3: History File Operations

| Feature          | zsh fc | shy fc | Implementation Status | Priority |
| ---------------- | ------ | ------ | --------------------- | -------- |
| `-R` read file   | ✅     | ❌     | **Missing**           | HIGH     |
| `-W` write file  | ✅     | ❌     | **Missing**           | HIGH     |
| `-A` append file | ✅     | ❌     | **Missing**           | MEDIUM   |
| `-I` incremental | ✅     | ❌     | **Missing**           | LOW      |

**shy Database Implications:**

- zsh fc operates on text history files
- shy uses SQLite database
- Need to define import/export formats

**Design Decision Required:**
Should `-R`/`-W`/`-A` work with:

1. **Option A:** Text files (zsh history format) - requires parser (parser should understand the different zsh file variations `: <timestamp>:<duration>;cmd` and `cmd`)
2. **Option B:** shy database files - enables database switching
3. **Option C:** Both - format detection or explicit flag

**Recommended: Option C** - Support both with format detection

### Mode 4: History Stack Operations

| Feature         | zsh fc | shy fc | Implementation Status | Priority |
| --------------- | ------ | ------ | --------------------- | -------- |
| `-p` push stack | ✅     | ❌     | **Missing**           | MEDIUM   |
| `-P` pop stack  | ✅     | ❌     | **Missing**           | MEDIUM   |
| `-a` auto-pop   | ✅     | ❌     | **Missing**           | LOW      |

**zsh Context:**

- `fc -p` pushes current history to stack, starts new history
- Used for temporary isolated history contexts
- Primarily for shell functions/scripts

**shy Context:**

- Could enable "session switching" or "project-specific histories"
- **This is the database switching mechanism!**

**Proposed Design:**

```bash
# Push current database, start using a new one
shy fc -p ~/project/.shy_history.db

# Do work in isolated history context
command1
command2

# Pop back to original database
shy fc -P
```

**added context**

`shy` will need to keep track of the last database within the pushed database.
`-P` can look at that database to revert. `shy` should add the `.db` extension
to a file argument when pushing if the `.db` extension does not exist. `-P`
when no last db exists is a no-op with a 1 exit code.

## Design Proposals

### 1. Edit-and-Execute Mode

**Architecture:**

```go
type EditExecuteMode struct {
    editor      string          // from -e flag, $FCEDIT, $EDITOR
    substitutions map[string]string  // old=new pairs
    executeOnly bool            // -s flag (skip editing)
}

func (e *EditExecuteMode) Run(commands []string) error {
    // 1. Apply old=new substitutions
    modified := e.applySubstitutions(commands)

    // 2. If not -s, open in editor
    if !e.executeOnly {
        modified = e.editInEditor(modified)
    }

    // 3. Execute modified commands
    return e.executeCommands(modified)
}
```

**Security Considerations:**

- Validate editor path (prevent path injection)
- Use secure temp file creation
- Warn on dangerous operations
- Option to preview before execution

**User Flow:**

```bash
# Edit and run command 123
shy fc 123

# Opens $EDITOR with:
# command text from event 123

# User edits, saves, exits
# shy executes the modified command
```

### 2. Pattern Filtering (`-m`)

**Implementation:**

Zsh uses **glob patterns** (wildcards), not regex. Need to translate glob syntax to SQL LIKE:

- `*` → `%` (zero or more characters)
- `?` → `_` (exactly one character)

```go
// Translate glob pattern to SQL LIKE pattern
func globToLike(pattern string) string {
    // Escape existing % and _ in the pattern
    escaped := strings.ReplaceAll(pattern, "%", "\\%")
    escaped = strings.ReplaceAll(escaped, "_", "\\_")

    // Translate glob wildcards to SQL wildcards
    escaped = strings.ReplaceAll(escaped, "*", "%")
    escaped = strings.ReplaceAll(escaped, "?", "_")

    return escaped
}

// In database layer
func (db *DB) GetCommandsByRangeWithPattern(first, last int64, pattern string) {
    likePattern := globToLike(pattern)
    query := `
        SELECT * FROM commands
        WHERE id >= ? AND id <= ?
        AND command_text LIKE ? ESCAPE '\'
        ORDER BY id ASC
    `
    rows, err := db.conn.Query(query, first, last, likePattern)
}
```

**Note:** Multiple `-m` patterns is not existing functionality and should not be allowed

**Usage:**

```bash
# List all git commands from last 100 events
shy fc -l -100 -1 -m "git*"
```

### 3. History File I/O

**File Format Support:**

only read and write to text file in non EXTENDED format

**Usage:**

```bash
# Import from zsh history
shy fc -R ~/.zsh_history

# Export to file
shy fc -W ~/backup_history.txt

# Append new commands to export file
shy fc -A ~/backup_history.txt

```

### 4. History Stack (Database Switching)

**Core Concept:**

- Stack of database paths/contexts
- Each entry stores: db_path, HISTSIZE equivalent
- ignore histsize
- Current context tracked globally
- only keep track of last db in the current db in a keyvalue table with `insert or replace`

```txt
/home/user/project/.shy_history.db
/home/user/.local/share/shy/history.db
```

**Integration with Shell Hooks:**

```bash
# In precmd hook, check current database
__shy_precmd() {
    # Get current database from stack
    current_db=$(shy fc --stack-current)

    # Insert to current database
    shy insert --db "$current_db" ...
}
```

**Usage Scenarios:**

**Scenario A: Project-Specific History**

```bash
# Working on project A
cd ~/projectA
shy fc -p ./.shy_history.db  # Push, start project-local history

# All commands now go to projectA/.shy_history.db
git status
make build

cd ~/projectB
shy fc -p ./.shy_history.db  # Push again, nested context

# Commands go to projectB/.shy_history.db
npm test

# Done with projectB
shy fc -P  # Pop back to projectA context

# Done with projectA
shy fc -P  # Pop back to global history
```

**Scenario B: Temporary Clean History**

```bash
# Start recording demo commands
shy fc -p /tmp/demo_history.db

# Record demo
echo "Step 1: Install"
echo "Step 2: Configure"

# Export for documentation
shy fc -W demo_commands.txt

# Return to normal history
shy fc -P
```

**Design Decisions:**

1. **Stack Persistence:**
   - Persist stack to config file (survive shell restarts)
   - OR keep in-memory (cleared on shell exit)
   - **Recommendation:** Persist to file, safer for shell crashes

2. **Auto-Pop:**
   - Support `-a` flag for function scope auto-pop?
   - Challenging in Go - would need shell integration
   - **Recommendation:** Manual pop only (Phase 1)

3. **Default Database:**
   - Bottom of stack is always default ~/.local/share/shy/history.db
   - Can't pop past the bottom
   - **Recommendation:** Yes, enforce this invariant

---

## Redundancy Analysis

### Overlapping Commands

1. **`shy fc -l` vs `shy list`**
   - `fc -l`: Standard interface, range-based, event IDs
   - `list`: Custom format strings (`--fmt`), time-based filters
   - **Decision:** Keep both. `fc` for POSIX compatibility, `list` for power users
   - **Integration:** `list` could internally use `fc` logic

2. **`shy fc -l` vs `shy history`**
   - `history` is explicitly an alias for `fc -l`
   - **Decision:** Keep as-is. Standard pattern.

3. **`shy fc -l -1` vs `shy last_command`**
   - Completely redundant
   - **Decision:** Keep `last_command` for scripts (simpler, stable API)
   - Document `fc` equivalent for users wanting standard tool

4. **`shy fc -l -m pattern` vs `shy like_recent`**
   - `like_recent`: prefix matching only
   - `fc -m`: pattern/glob matching
   - **Decision:** Keep both. Different capabilities.

### Command Consolidation Opportunities

**Option 1: Deprecate Redundant Commands**

- Remove `last_command`, `like_recent`, `list_all`
- Force users to use `fc`
- **Pros:** Simpler codebase
- **Cons:** Breaking change, no backward compatibility

**Option 2: Keep All Commands**

- Document equivalents
- Maintain backward compatibility
- **Pros:** No breaking changes
- **Cons:** More code to maintain

**Option 3: Implement as Aliases (Recommended)**

- Keep command names but implement as wrappers to `fc`
- Share code, reduce duplication
- **Pros:** Backward compatible, reduced code duplication
- **Cons:** None significant

**Recommendation: Option 3**

```go
// last_command.go
var lastCommandCmd = &cobra.Command{
    Use: "last_command",
    Short: "Get the most recent command",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Delegate to fc
        fcList = true
        fcNoNum = true
        return runFc(cmd, []string{"-1"})
    },
}
```

## Open Questions for User

### 1. Security Model for Edit-and-Execute

**Question:** How should shy handle command execution security?

**Options:**

- **A:** Always execute (trust user)
- **B:** Prompt for confirmation before execution
- **C:** Preview mode + confirmation flag (`--execute`, `-y`)
- **D:** Dry-run default with `--execute` to actually run

**Recommendation:** Option C (preview with explicit flag)

**User Decision:** I believe that `fc` trusts the user, and `shy fc` is trying to replicate this behaviour. Trust the user.

### 2. History Stack Default Behavior

**Question:** Should history stack persist across shell sessions?

**Options:**

- **A:** Persist to config file (survive restarts)
- **B:** In-memory only (clear on shell exit)
- **C:** Configurable via setting

**Recommendation:** Option A (persist)

**User Decision:** All user sessions are synced

### 3. Database Merge Strategy

**Question:** When importing from another database, how to handle ID conflicts?

**Options:**

- **A:** Re-number imported commands (assign new IDs)
- **B:** Keep original IDs, fail on conflict
- **C:** Keep original IDs, overwrite on conflict
- **D:** Interleave by timestamp, assign new IDs

**Recommendation:** Option D (timestamp-ordered re-numbering)

**User Decision:** always keep original ids. re-number imported commands.

### 4. Format for `-W` Export

**Question:** What should be the default export format for `shy fc -W`?

**Options:**

- **A:** Zsh extended history (most compatible)
- **B:** Bash history (simpler, wide compatibility)
- **C:** JSON (most data, shy-specific)
- **D:** Auto-detect from file extension

**Recommendation:** Option D (auto-detect: .zsh → zsh, .json → json, else bash)

**User Decision:** For these use cases, the following behaviours should apply

Force a history save `fc -W <no arg>` -> do nothing, all history is synced and saved
sync a history across sessions `fc -W <no arg>` -> do nothing, all history is synced and saved
create a custom history file `fc -W <filepath>` -> save current history of session (commands matching zsh source) to file

### 5. Local/Internal Semantics

**Question:** How should `-L` and `-I` flags work in shy context?

**Background:** In zsh:

- `-L`: local to current shell (not from other shells via SHARE_HISTORY)
- `-I`: internal (not from $HISTFILE)

**Options for shy:**

- **A:** Not implemented (N/A, document as unsupported)
- **B:** `-L` = current session only, `-I` = not from imports
- **C:** `-L` = current database only (if using stack), `-I` = programmatic inserts only

**Recommendation:** Option B

**User Decision:** -L and -I as local/internal semantics only work in the
context of `-l`. `-L` should provide no behaviour differences as all history
is assumed synchronized and local. `-I` should only show history related to
the current session which implies a source field in the database that would
capture the shell (zsh, bash, etc) and the associated pid, returning only
commands that matched that particular pid. When a zsh session closes, it
should update the source values for all "internal" with an `X` or other char to
indicate that this session is closed, preventing reused pids from matching
against this session in the future.

### 6. Command Consolidation

**Question:** Should redundant commands be deprecated?

**Background:** `last_command`, `like_recent`, `list_all` overlap with `fc` functionality

**Options:**

- **A:** Keep all commands, implement as wrappers to fc
- **B:** Deprecate redundant commands in next major version
- **C:** Keep as-is, maintain separately

**Recommendation:** Option A (wrappers)

**User Decision:** all these commands will function differently and will provide the primary listing for the shy command. the `fc` command is for backward compatibility only.

### 7. Pattern Syntax

**Decision:** Use **glob patterns** (wildcards) to match zsh behavior.

**Implementation:**

- `-m` flag accepts glob patterns: `*` (any chars), `?` (single char)
- Translate to SQL LIKE internally: `*` → `%`, `?` → `_`
- Example: `shy fc -l -m "git*"` matches all commands starting with "git"

**Status:** ✅ Decided (glob patterns required for parity)

---

## Appendix A: zsh fc Complete Syntax

```
fc [ -e ename ] [ -s ] [ -LI ] [ -m match ] [ old=new ... ] [ first [ last ] ]
fc -l [ -LI ] [ -nrdfEiD ] [ -t timefmt ] [ -m match ] [ old=new ... ] [ first [ last ] ]
fc -p [ -a ] [ filename [ histsize [ savehistsize ] ] ]
fc -P
fc -ARWI [ filename ]
```

### Flag Reference

| Flag         | Mode      | Description                           |
| ------------ | --------- | ------------------------------------- |
| `-l`         | List      | List commands instead of editing      |
| `-n`         | List      | Suppress event numbers                |
| `-r`         | List      | Reverse order                         |
| `-d`         | List      | Display timestamps                    |
| `-f`         | List      | US format timestamps (MM/DD/YY hh:mm) |
| `-E`         | List      | European format (dd.mm.yyyy hh:mm)    |
| `-i`         | List      | ISO8601 format (yyyy-mm-dd hh:mm)     |
| `-t fmt`     | List      | Custom strftime format                |
| `-D`         | List      | Display elapsed time                  |
| `-m pattern` | List/Edit | Filter by pattern                     |
| `-L`         | List/Edit | Local events only                     |
| `-I`         | List/Edit | Internal events only                  |
| `-e editor`  | Edit      | Specify editor                        |
| `-s`         | Edit      | Execute without editing               |
| `-p`         | Stack     | Push history stack                    |
| `-P`         | Stack     | Pop history stack                     |
| `-a`         | Stack     | Auto-pop on function exit             |
| `-R`         | File      | Read history from file                |
| `-W`         | File      | Write history to file                 |
| `-A`         | File      | Append to history file                |

---

## Appendix B: Code Structure

### Proposed File Organization

which augments the existing file structure

```
cmd/
├── fc.go                    # Main fc command implementation
├── fc_list.go               # List mode logic
├── fc_edit.go               # Edit-and-execute mode
├── fc_file.go               # File I/O operations
├── fc_stack.go              # History stack operations
├── fc_test.go               # Tests
├── history.go               # Wrapper for fc -l
internal/
├── db/
│   ├── db.go
│   ├── import.go            # History import logic
│   └── export.go            # History export logic
├── history/
│   ├── parser.go            # Parser for zsh/bash formats
│   ├── stack.go             # History stack implementation
│   └── merge.go             # Database merge logic
└── exec/
    ├── editor.go            # Editor integration
    └── execute.go           # Command execution
```
