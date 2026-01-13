# Autosuggest Shy Integration Plan

## Overview

Rewrite zsh-autosuggestions strategy functions to use `shy` as the data source instead of zsh's native history array.

## Current Functions Analysis

### 1. `_zsh_autosuggest_strategy_history`

**Current Behavior:**

- Takes a prefix (partial command user has typed)
- Escapes special glob characters
- Searches zsh history array for most recent match
- Returns first matching command as suggestion
- Respects `ZSH_AUTOSUGGEST_HISTORY_IGNORE` pattern

**Data Source:** `$history[(r)$pattern]` - zsh history array, reverse search

### 2. `_zsh_autosuggest_strategy_match_prev_cmd`

**Current Behavior:**

- Takes a prefix (partial command user has typed)
- Finds commands matching the prefix
- Looks for commands that came AFTER the same previous command
- More contextual: suggests what you did last time after the same command
- Checks up to 200 matches for context
- Respects `ZSH_AUTOSUGGEST_HISTORY_IGNORE` pattern

**Data Source:**

- `${(k)history[(R)$~pattern]}` - all history keys matching pattern
- `${history[$((HISTCMD-1))]}` - previous command
- `${history[$((key - 1))]}` - command before each match

**Example:**

```
Previous: git commit -m "fix"
User types: git push
Suggestion: git push origin main
(because last time after "git commit", you ran "git push origin main")
```

## Design Considerations

### Performance Requirements

- **Critical:** Must be fast (<50ms) for interactive use
- Called on every keystroke during typing
- Should not block the shell
- May need caching strategy

### Data Access Patterns

1. **Pattern matching:** Find commands starting with prefix
2. **Recency:** Get most recent matches first
3. **Context awareness:** Find what came after a specific command
4. **Session filtering:** Optionally filter by current session

### shy Command Capabilities Needed

- Fast prefix search
- Return results in reverse chronological order
- Get previous command (N-1)
- Filter by patterns
- Limit results

## Implementation Plan

### Phase 1: Enhance Existing Commands

#### 1.1 Enhance `shy like-recent` Command

**Current:** Already exists in `cmd/like_recent.go` but uses inefficient array scan

**Purpose:** Fast prefix matching for autosuggestions

**Location:** `cmd/like_recent.go` (existing file)

**Usage:**

```bash
shy like-recent "git pu"
# Output: git push origin main
```

**New Flags to Add:**

- `--limit <int>` - Number of suggestions (default: 1)
- `--pwd` - Only match commands from current directory
- `--session` - Only match from current session
- `--exclude <pattern>` - Exclude pattern (for HISTORY_IGNORE)
- `--include-shy` - Include shy commands (default: exclude)

**Backward Compatibility:**
- Positional argument still works: `shy like-recent "prefix"`
- Default behavior unchanged (most recent match, excludes shy commands)
- New flags are optional enhancements

**Performance Improvements:**

Replace current approach (loads 10k commands into memory) with SQL query:

```go
func runLikeRecent(cmd *cobra.Command, args []string) error {
    prefix := args[0]

    // Build SQL query with filters
    query := `
        SELECT command_text
        FROM commands
        WHERE command_text LIKE ? || '%'
    `

    // Add filters based on flags
    if !includeShy {
        query += ` AND command_text NOT LIKE 'shy %' AND command_text != 'shy'`
    }
    if excludePattern != "" {
        query += ` AND command_text NOT GLOB ?`
    }
    if filterPwd {
        query += ` AND working_dir = ?`
    }
    if filterSession {
        query += ` AND source_pid = ?`
    }

    query += ` ORDER BY timestamp DESC LIMIT ?`

    // Execute query
}
```

**SQL Query:**

```sql
SELECT command_text
FROM commands
WHERE command_text LIKE ? || '%'
  AND command_text NOT LIKE 'shy %'
  AND command_text != 'shy'
  AND (:exclude_pattern IS NULL OR command_text NOT GLOB :exclude_pattern)
  AND (:pwd IS NULL OR working_dir = :pwd)
  AND (:session_pid IS NULL OR source_pid = :session_pid)
ORDER BY timestamp DESC
LIMIT :limit
```

#### 1.2 Add `shy like-recent-after` Command

**Purpose:** Contextual suggestions based on previous command

**Location:** `cmd/like_recent_after.go` (new file, or add to `like_recent.go`)

**Usage:**

```bash
shy like-recent-after "git pu" --prev "git commit -m 'fix'"
# Output: git push origin main
```

**Arguments:**
- Positional: `<prefix>` - Command prefix to match

**Flags:**

- `--prev <string>` - Previous command to match context (required)
- `--limit <int>` - Number of suggestions (default: 1)
- `--exclude <pattern>` - Exclude pattern
- `--include-shy` - Include shy commands (default: exclude)

**Implementation:**

```go
func runLikeRecentAfter(cmd *cobra.Command, args []string) error {
    prefix := args[0]
    prevCmd := // from --prev flag

    // Find commands matching prefix
    // For each match, check if previous command matches prevCmd
    // Return first match with matching context
}
```

**SQL Query:**

```sql
WITH matches AS (
    SELECT id, command_text, timestamp
    FROM commands
    WHERE command_text LIKE ? || '%'
    ORDER BY timestamp DESC
    LIMIT 200
)
SELECT m.command_text
FROM matches m
JOIN commands prev ON prev.id = m.id - 1
WHERE prev.command_text = ?
ORDER BY m.timestamp DESC
LIMIT 1
```

#### 1.3 Add `shy last-command` Enhancement

**Current:** Already exists, may need optimization

**Ensure:**

- Can get last command quickly
- Can get N-th last command
- Can filter by session

### Phase 2: Create Shy-Based Strategy Functions

#### 2.1 Create `shy_autosuggest.zsh`

**Location:** `cmd/integration_scripts/shy_autosuggest.zsh` (new file)

**Contents:**

```zsh
# Strategy 1: Simple history matching using shy
_zsh_autosuggest_strategy_shy_history() {
    emulate -L zsh

    local prefix="$1"

    # Build shy command
    local shy_args=("like-recent" "$prefix")

    # Add exclude pattern if set
    if [[ -n $ZSH_AUTOSUGGEST_HISTORY_IGNORE ]]; then
        shy_args+=("--exclude" "$ZSH_AUTOSUGGEST_HISTORY_IGNORE")
    fi

    # Query shy for suggestion
    local result
    result=$(shy "${shy_args[@]}" 2>/dev/null)

    # Set global suggestion variable
    typeset -g suggestion="$result"
}

# Strategy 2: Context-aware matching using shy
_zsh_autosuggest_strategy_shy_match_prev_cmd() {
    emulate -L zsh

    local prefix="$1"

    # Get previous command
    local prev_cmd
    prev_cmd=$(shy last-command -n 2 2>/dev/null)

    # Build shy command
    local shy_args=(
        "like-recent-after"
        "$prefix"
        "--prev" "$prev_cmd"
    )

    # Add exclude pattern if set
    if [[ -n $ZSH_AUTOSUGGEST_HISTORY_IGNORE ]]; then
        shy_args+=("--exclude" "$ZSH_AUTOSUGGEST_HISTORY_IGNORE")
    fi

    # Query shy for contextual suggestion
    local result
    result=$(shy "${shy_args[@]}" 2>/dev/null)

    # Set global suggestion variable
    typeset -g suggestion="$result"
}

# Strategy 3: Directory-aware suggestions
_zsh_autosuggest_strategy_shy_pwd() {
    emulate -L zsh

    local prefix="$1"

    # Build shy command with pwd filter
    local shy_args=(
        "like-recent"
        "$prefix"
        "--pwd"
    )

    # Add exclude pattern if set
    if [[ -n $ZSH_AUTOSUGGEST_HISTORY_IGNORE ]]; then
        shy_args+=("--exclude" "$ZSH_AUTOSUGGEST_HISTORY_IGNORE")
    fi

    # Query shy for suggestion
    local result
    result=$(shy "${shy_args[@]}" 2>/dev/null)

    # Set global suggestion variable
    typeset -g suggestion="$result"
}

# Strategy 4: Session-aware suggestions
_zsh_autosuggest_strategy_shy_session() {
    emulate -L zsh

    local prefix="$1"

    # Build shy command with session filter
    local shy_args=(
        "like-recent"
        "$prefix"
        "--session"
    )

    # Add exclude pattern if set
    if [[ -n $ZSH_AUTOSUGGEST_HISTORY_IGNORE ]]; then
        shy_args+=("--exclude" "$ZSH_AUTOSUGGEST_HISTORY_IGNORE")
    fi

    # Query shy for suggestion
    local result
    result=$(shy "${shy_args[@]}" 2>/dev/null)

    # Set global suggestion variable
    typeset -g suggestion="$result"
}
```

### Phase 3: Performance Optimization

#### 3.1 Add Database Indexes

**Location:** `internal/db/db.go` schema

**Indexes to Add:**

```sql
-- For prefix matching
CREATE INDEX IF NOT EXISTS idx_command_text_prefix
ON commands(command_text);

-- For timestamp ordering
CREATE INDEX IF NOT EXISTS idx_timestamp_desc
ON commands(timestamp DESC);

-- For pwd filtering
CREATE INDEX IF NOT EXISTS idx_working_dir
ON commands(working_dir);

-- For session filtering
CREATE INDEX IF NOT EXISTS idx_source_pid
ON commands(source_pid);

-- Composite index for common query pattern
CREATE INDEX IF NOT EXISTS idx_prefix_timestamp
ON commands(command_text, timestamp DESC);
```

#### 3.2 Add Result Caching (Optional)

**Strategy:** Cache recent suggestions in memory

**Implementation:**

- Small LRU cache (10-20 entries)
- Key: prefix + context
- TTL: 5 seconds
- Clear on new command execution

#### 3.3 Async/Background Queries (Optional)

**Strategy:** Pre-fetch suggestions in background

**Challenge:** Need to coordinate with zsh-autosuggestions plugin

### Phase 4: Integration and Testing

#### 4.1 Update zsh_use.sh

**Add:**

```zsh
# Source shy autosuggest strategies if zsh-autosuggestions is installed
if [[ -n $ZSH_AUTOSUGGEST_STRATEGY ]]; then
    source "$(shy init zsh --autosuggest)"
    # Override default strategies
    ZSH_AUTOSUGGEST_STRATEGY=(shy_history shy_match_prev_cmd)
fi
```

#### 4.2 Create Integration Tests

**Test Cases:**

1. Simple prefix matching
2. Context-aware suggestions
3. Directory-specific suggestions
4. Session-specific suggestions
5. Exclude pattern handling
6. Empty result handling
7. Special character escaping

#### 4.3 Performance Benchmarks

**Measure:**

- Query latency (target: <50ms)
- Memory usage
- Database I/O
- CPU usage during typing

### Phase 5: Documentation

#### 5.1 User Documentation

**Topics:**

- How to enable shy-based autosuggestions
- Available strategies
- Configuration options
- Performance tuning

#### 5.2 Technical Documentation

**Topics:**

- SQL query patterns
- Index strategy
- Caching approach
- Integration points

## Implementation Sequence

### Milestone 1: Basic Functionality

1. ⬜ Enhance `cmd/like_recent.go` with SQL query
2. ⬜ Add flags: `--pwd`, `--session`, `--exclude`, `--limit`, `--include-shy`
3. ⬜ Add database indexes for performance
4. ⬜ Create `shy_autosuggest.zsh`
5. ⬜ Implement `_zsh_autosuggest_strategy_shy_history`
6. ⬜ Test basic functionality

### Milestone 2: Context Awareness

1. ⬜ Create `cmd/like_recent_after.go` (or add to like_recent.go)
2. ⬜ Implement `shy like-recent-after` command
3. ⬜ Add context-aware SQL query
4. ⬜ Implement `_zsh_autosuggest_strategy_shy_match_prev_cmd`
5. ⬜ Test context-aware suggestions

### Milestone 3: Advanced Filtering

1. ✅ Add `--pwd` flag support
2. ✅ Add `--session` flag support
3. ✅ Implement additional strategy functions
4. ✅ Test filtering

### Milestone 4: Performance

1. ✅ Add database indexes
2. ✅ Benchmark query performance
3. ✅ Optimize slow queries
4. ✅ Consider caching if needed

### Milestone 5: Integration

1. ✅ Update zsh_use.sh
2. ✅ Write integration tests
3. ✅ Write documentation
4. ✅ User testing

## Command Line Interface

### shy like-recent (Enhanced)

```
USAGE:
  shy like-recent <prefix> [flags]

ARGUMENTS:
  prefix               Command prefix to match

FLAGS:
  --limit int          Number of suggestions (default: 1)
  --pwd                Only match commands from current directory
  --session            Only match from current session (SHY_SESSION_PID)
  --exclude string     Exclude commands matching pattern (glob)
  --include-shy        Include shy commands in results (default: false)
  --db string         Database path (default: ~/.local/share/shy/history.db)

EXAMPLES:
  # Get most recent command starting with "git pu"
  shy like-recent "git pu"

  # Get suggestion from current directory only
  shy like-recent "make" --pwd

  # Get suggestion from current session
  shy like-recent "npm" --session

  # Exclude certain patterns
  shy like-recent "git" --exclude "git pull*"

  # Include shy commands in results
  shy like-recent "shy" --include-shy
```

### shy like-recent-after (New)

```
USAGE:
  shy like-recent-after <prefix> [flags]

ARGUMENTS:
  prefix               Command prefix to match

FLAGS:
  --prev string        Previous command to match context (required)
  --limit int          Number of suggestions (default: 1)
  --exclude string     Exclude commands matching pattern (glob)
  --include-shy        Include shy commands in results (default: false)
  --db string         Database path (default: ~/.local/share/shy/history.db)

EXAMPLES:
  # Get command that typically follows "git commit"
  shy like-recent-after "git pu" --prev "git commit -m 'fix'"

  # With pattern exclusion
  shy like-recent-after "make" --prev "make build" --exclude "make clean"
```

## SQL Query Patterns

### Basic Suggestion Query (for like-recent)

```sql
SELECT command_text
FROM commands
WHERE command_text LIKE :prefix || '%'
  AND (:include_shy = 1 OR (command_text NOT LIKE 'shy %' AND command_text != 'shy'))
  AND (:exclude_pattern IS NULL OR command_text NOT GLOB :exclude_pattern)
  AND (:pwd IS NULL OR working_dir = :pwd)
  AND (:session_pid IS NULL OR source_pid = :session_pid)
ORDER BY timestamp DESC
LIMIT :limit
```

### Context-Aware Suggestion Query (for like-recent-after)

```sql
WITH recent_matches AS (
    SELECT c.id, c.command_text, c.timestamp
    FROM commands c
    WHERE c.command_text LIKE :prefix || '%'
      AND (:include_shy = 1 OR (c.command_text NOT LIKE 'shy %' AND c.command_text != 'shy'))
      AND (:exclude_pattern IS NULL OR c.command_text NOT GLOB :exclude_pattern)
    ORDER BY c.timestamp DESC
    LIMIT 200
)
SELECT rm.command_text
FROM recent_matches rm
JOIN commands prev ON prev.id = rm.id - 1
WHERE prev.command_text = :prev_cmd
ORDER BY rm.timestamp DESC
LIMIT :limit
```

## Testing Strategy

### Unit Tests

- SQL query generation
- Pattern escaping
- Filter application
- Result formatting

### Integration Tests

- End-to-end suggestion flow
- Strategy function behavior
- zsh integration
- Error handling

### Performance Tests

- Query latency benchmarks
- Load testing (large databases)
- Memory profiling
- Index effectiveness

## Success Criteria

- ✅ Suggestions appear within 50ms
- ✅ All original functionality preserved
- ✅ New context-aware strategies work
- ✅ No shell responsiveness issues
- ✅ Works with large history databases (100k+ commands)
- ✅ Properly handles special characters
- ✅ Respects existing configuration variables

## Migration Path

### For Users

1. Update to new version with shy autosuggest support
2. Optionally enable in `.zshrc`:
   ```zsh
   # Use shy-based autosuggestions
   ZSH_AUTOSUGGEST_STRATEGY=(shy_history shy_match_prev_cmd)
   ```
3. Original zsh history still works (backward compatible)

### Backward Compatibility Notes

**`like-recent` enhancements are fully backward compatible:**
- Existing scripts using `shy like-recent "prefix"` continue to work unchanged
- Default behavior (most recent match, exclude shy commands) is preserved
- New flags are optional - no breaking changes
- Performance improved from O(n) array scan to O(log n) indexed SQL query

### Fallback Behavior

- If shy commands fail, silently return empty suggestion
- No error messages in interactive shell
- Graceful degradation to no suggestions

## Future Enhancements

### Machine Learning Suggestions

- Learn common command patterns
- Suggest based on time of day, directory, or context
- Frequency-based ranking

### Multi-Line Command Support

- Suggest multi-line commands
- Handle command continuations

### Fuzzy Matching

- Suggest commands with typos corrected
- Substring matching instead of prefix only

### Cross-Session Context

- Learn from all terminal sessions
- Suggest based on recent activity across terminals
