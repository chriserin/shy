# Plan: `shy history` - Feature Parity with zsh builtin `history`

## Overview

Implement `shy history` command to provide feature parity with zsh's builtin `history` command (which is an alias for `fc -l`). This will allow users to use `shy` as a drop-in replacement for their shell's built-in history, with the added benefits of shy's rich data storage (git context, exit status, working directory, etc.).

## Current State Analysis

### What we have:

- `shy list` - Lists recent commands (default: 20)
  - `--limit` flag to control count
  - `--fmt` flag for custom column output
  - Time filtering: `--today`, `--yesterday`, `--this-week`, `--last-week`
- `shy list-all` - Lists all commands without limit
- `shy last-command` - Get Nth most recent command (`-n` flag)
- `shy like-recent` - Get most recent command matching prefix

### What's missing for `history` parity:

- Numbered output (line numbers for each history entry)
- Range selection (e.g., `history 100 200`, `history -100`)
- Timestamp formatting options (multiple formats)
- Reverse order flag
- Pattern matching/filtering
- Event number references

## zsh `history` / `fc -l` Feature Analysis

Based on `man zshbuiltins`, the `fc -l` command (aliased as `history`) supports:

### Range Selection:

- `history` - Default shows last 16 events
- `history 20` - Shows events from event with id 20 through the most recent event
- `history 100 200` - Shows events 100 through 200
- `history -20` - Shows last 20 events (negative offset)
- `history string` - Shows events from most recent event matching string to most recent event
- `history string 4119` - shows events from most recent matching string to event with id 4119
- `history string string` - shows events from most recent matching first string to most recent matching second string

### Output Formatting:

- **Default**: Event number + command text
- `-n` - Suppress event numbers
- `-r` - Reverse order (oldest first instead of newest first)
- `-d` - Print timestamps for each event
- `-f` - Full timestamp in US format `MM/DD/YY hh:mm`
- `-E` - Full timestamp in European format `dd.mm.yyyy hh:mm`
- `-i` - Full timestamp in ISO8601 format `yyyy-mm-dd hh:mm`
- `-t fmt` - Custom timestamp format (strftime)
- `-D` - Print elapsed times (can combine with above)

### Filtering:

- `-m pattern` - Only show events matching pattern
  - pattern is a glob pattern that matches only against the events returned in accordance to the range arguments
- `-I` - Restrict to internal events only
- `-L` - Restrict to local events only

### Other Features:

- Event numbers are persistent and sequential
- Can reference by number for re-execution (`!123`, `!!`, `!-1`)
- Can reference by string prefix (`!git`)

## Implementation Plan

### Phase 1: Core `shy history` Command (Iteration 3)

#### 1.1 Basic Command Structure

- Create `cmd/history.go`
- Implmenent as `cmd/fc` but with the `-l` option, the `cmd/history` should refer to the `fc` implementation
- Make `shy history` an alias/wrapper for enhanced list functionality
- Default: Show last 16 commands with line numbers (match zsh default)

#### 1.2 Event Numbering

- use existing row id for event numbering

#### 1.3 Range Selection

- `shy history` - Default shows last 16 events
- `shy history 20` - Shows events from event with id 20 through the most recent event
- `shy history 100 200` - Shows events 100 through 200
- `shy history -20` - Shows last 20 events (negative offset)
- `shy history string` - Shows events from most recent event matching string to most recent event
- `shy history string 4119` - shows events from most recent matching string to event with id 4119
- `shy history string string` - shows events from most recent matching first string to most recent matching second string

#### 1.4 Basic Output Flags

- `-n` - Suppress event numbers (equivalent to current `list`)
- `--reverse` / `-r` - Show in reverse chronological order

### Phase 2: Timestamp Formatting (Iteration 3)

#### 2.1 Timestamp Display Flags

- `-d` - Show timestamp with each event
- `-f` - US format: `MM/DD/YY hh:mm`
- `-E` - European format: `dd.mm.yyyy hh:mm`
- `-i` - ISO8601 format: `yyyy-mm-dd hh:mm`
- `-t FMT` - Custom strftime format
- `-D` - Show elapsed time since command ran

#### 2.2 Default Timestamp Behavior

- No timestamp by default (match zsh behavior)
- When timestamp flag used, format: `<event_number> <timestamp> <command>`

### Phase 3: Pattern Matching & Filtering (Iteration 3)

#### 3.1 Pattern Matching

- `-m PATTERN` - Filter by pattern (regex or glob)

## Backwards Compatibility

## Documentation Updates

- Update README.md with `shy history` examples
- Add comparison table: zsh `history` vs `shy history`
- Update shell integration to add alias

## Open Questions

1. **Should event numbers be global or per-user?**
   - Decision: Per-database (per-user), matches zsh behavior

2. **How to handle deleted commands?**
   - Decision: Event numbers are never reused, even if command deleted
   - Gap in numbering is acceptable

3. **Default number of entries?**
   - Decision: Match zsh default (16)
   - Recommendation: Use 16 to match zsh exactly

4. **Should we support editing history like `fc` (without `-l`)?**
   - Decision: Phase 5 / future feature, focus on listing first
