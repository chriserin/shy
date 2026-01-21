# shy - Shell History Tracker

A command-line tool for tracking shell command history in SQLite with rich metadata and conventional behaviour.

## Features

- **Rich Metadata**: working directory, start time, duration, git repo, git branch, session
- **Tool Integrations**: zsh, television, fzf, zsh-autosuggestion
- **Work Summary**: Yesterday's history in a report format
- **Sensible Duplication Behaviour**
- **Sensible Scopes**

## Scopes: Session, Current Session, and All

There are 3 different scopes for `shy`.  The current session, the current
working directory, and all.  These scopes support 3 different use cases:

1. up/down arrow: Traverse previous commands of the **current session**.
2. autosuggest: Suggest previous commands that have been used in the **current working directory**.
3. ctrl-r: Search through **all** previous commands

This is achievable with the zsh `SHARE_HISTORY` option, but needs tinkering to
separate the up/down history from the all history, and this is mutually
exclusive from collecting duration with the `INC_APPEND_HISTORY_TIME` option.

## Duplicates

There is a tension between collection duplicates and not collecting duplicates.

1. **DUPS**: Command Context: Useful when re-executing commands in order with
   the "accept-line-and-down-history" widget, or understanding how a command
   was in context with other commands. 
2. **NO DUPS**: Searching: It is not useful to find 10, 20, or 100 of the same
   command when searching through commands
3. **NO SEQ DUPS**: Traversing:  When moving through previous commands with the
   up or down errors, it's not useful to traverse the same command more than
   once.

It is not possible in zsh to support all 3 use cases.  Duplicates in zsh are managed
at the command time, not read time, so you have to choose one of the scenarios
supported by the HIST_IGNORE_ALL_DUPS or HIST_IGNORE_DUPS options.

In `shy`, all commands are stored, enabling the application to be intentional
about when duplicates are presented or not presented.

## Integration

### ZSH

Shy can automatically track your shell history by integrating with zsh. Add this to your `.zshrc`:

```bash
eval "$(shy init zsh)"
```


### FZF

For a better history search experience with fzf:

```sh
# Load fzf key-bindings first (if not already loaded)
source /usr/share/fzf/key-bindings.zsh  # Adjust path for your system

# Bind ctrl-r for history search
bindkey '^R' shy-fzf-history-widget
```

### zsh-autosuggestions

Shy provides custom suggestion strategies for [zsh-autosuggestions](https://github.com/zsh-users/zsh-autosuggestions):

```sh
# Load zsh-autosuggestions plugin first
source /path/to/zsh-autosuggestions/zsh-autosuggestions.zsh

# Load shy integration
eval "$(shy init zsh)"

# Configure suggestion strategy with your preferred strategy
ZSH_AUTOSUGGEST_STRATEGY=(shy_history)              # Simple prefix matching
ZSH_AUTOSUGGEST_STRATEGY=(shy_match_prev_cmd)      # Context-aware (matches after previous command)
ZSH_AUTOSUGGEST_STRATEGY=(shy_pwd)                 # Directory-specific suggestions
ZSH_AUTOSUGGEST_STRATEGY=(shy_session)             # Current session only

# or choose multiple for graceful fallback
# I prefer session before current directory before all history
ZSH_AUTOSUGGEST_STRATEGY=(shy_session shy_pwd shy_history)
```


### television Integration (Optional)

Browse your command history with [television](https://github.com/alexpasmantier/television), a fuzzy finder TUI:

```bash
# Generate the television channel configuration
shy tv config > ~/.config/television/cable/shy-history.toml

# Launch television with the shy channel
tv shy-history
```


### Configuration

Control tracking behavior with environment variables:

```bash
# disable tracking
SHY_DISABLE=1

# insert to a db of your choosing
SHY_DB_PATH=/path/to/custom.db
```

## Commands

### Command Overview

| Command | Default Scope | Duplicates | Description |
|---------|--------------|------------|-------------|
| `fc` / `history` | ALL | NO DUPS | List or edit command history (use `--internal` for session, `--local` for pwd) |
| `list` | ALL | DUPS | List recent commands (use `--session` or `--current-session` to filter) |
| `list-all` | ALL | DUPS | List all commands (use `--session` or `--current-session` to filter) |
| `last-command` | SESSION + PWD | NO SEQ DUPS | Get most recent command (unions session with current directory, skips consecutive duplicates) |
| `fzf` | ALL | NO DUPS | Output history for fzf integration (SQL-based deduplication) |
| `like-recent` | ALL | Single Result | Find most recent command with prefix (use `--pwd` or `--session` to filter) |
| `like-recent-after` | ALL | Single Result | Context-aware command suggestions based on previous command |
| `summary` | ALL | N/A | Show aggregated activity report (use `--source-app` to filter) |
| `tabsum` | ALL | N/A | Tabular summary of command activity |
| `insert` | N/A | N/A | Manually insert a command into the database |
| `close-session` | N/A | N/A | Mark a session as inactive |
| `init` | N/A | N/A | Generate shell integration scripts |
| `tv` | N/A | N/A | Generate television channel configuration |


**Scope Types:**
- **ALL**: Searches entire command history across all sessions and directories
- **SESSION**: Current shell session only (filtered by `source_app` and `source_pid`)
- **PWD**: Current working directory only
- **ALL + PWD**: Session results first, then union with current directory results

**Duplicate Behavior:**
- **DUPS**: Returns all commands including duplicates
- **NO DUPS**: Filters out duplicate commands (keeps most recent)
- **NO SEQ DUPS**: Filters out consecutive duplicate commands only
- **Single Result**: Returns only the most recent matching command
- **N/A**: Not applicable (command doesn't return history)

### Insert Command

Insert a command into the history database:

```bash
shy insert --command "ls -la" --dir /home/user/project --status 0
```
