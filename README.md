# shy - Shell History Tracker

A lightweight command-line tool for tracking shell command history in SQLite with rich metadata.

## Features

- **SQLite Database**: Commands stored in a structured, queryable database
- **Rich Metadata**: Captures command text, working directory, exit status, timestamp
- **Git Context**: Auto-detects git repository and branch information
- **Concurrent Safe**: Handles concurrent inserts using SQLite WAL mode
- **Flexible**: Manual insertion with optional parameter overrides

## Installation

```bash
go build -o shy .
```

## Zsh Integration

Shy can automatically track your shell history by integrating with zsh. Add this to your `.zshrc`:

```bash
# Basic integration (auto-tracking + Ctrl-R listing)
eval "$(shy init zsh)"
```

### Features

The zsh integration provides:

1. **Automatic command tracking** - Commands are automatically inserted into the database after execution with metadata (timestamps, exit codes, duration, git context)
2. **History usage** - The zsh features that rely on history will now rely on shy's commands.

- up and down arrows to navigate history one command at a time
- aliases for `history`, `fc` and `r` to ensure scripts using these commands still function
- ctrl-R functionality for different integrations including fzf and tv
- zsh-autosuggestion integration

### fzf Integration (Optional)

For a better history search experience with fzf:

```sh
# Load fzf key-bindings first (if not already loaded)
source /usr/share/fzf/key-bindings.zsh  # Adjust path for your system

# Load shy with fzf integration
eval "$(shy init zsh)"

# Bind Ctrl-R to shy's fzf widget (replaces fzf's default history widget)
bindkey '^R' shy-fzf-history-widget
```

### zsh-autosuggestions Integration (Optional)

Shy provides custom suggestion strategies for [zsh-autosuggestions](https://github.com/zsh-users/zsh-autosuggestions):

```sh
# Load zsh-autosuggestions plugin first
source /path/to/zsh-autosuggestions/zsh-autosuggestions.zsh

# Load shy integration
eval "$(shy init zsh)"

# Configure suggestion strategies (choose one or combine)
ZSH_AUTOSUGGEST_STRATEGY=(shy_history)              # Simple prefix matching
ZSH_AUTOSUGGEST_STRATEGY=(shy_match_prev_cmd)      # Context-aware (matches after previous command)
ZSH_AUTOSUGGEST_STRATEGY=(shy_pwd)                 # Directory-specific suggestions
ZSH_AUTOSUGGEST_STRATEGY=(shy_session)             # Current session only
```

### Configuration

Control tracking behavior with environment variables:

```bash
# Temporarily disable tracking
SHY_DISABLE=1

# Use a custom database location
SHY_DB_PATH=/path/to/custom.db
```

## Usage

### Insert Command

Insert a command into the history database:

```bash
shy insert --command "ls -la" --dir /home/user/project --status 0
```

### Flags

- `--command` (required): The command text to store
- `--dir` (required): The working directory where the command was executed
- `--status`: Exit status code (default: 0)
- `--git-repo`: Git repository URL (auto-detected if not provided)
- `--git-branch`: Git branch name (auto-detected if not provided)
- `--timestamp`: Unix timestamp (default: current time)
- `--db`: Database file path (default: $XDG_DATA_HOME/shy/history.db or ~/.local/share/shy/history.db)

### Examples

#### Basic command insertion

```bash
shy insert --command "npm test" --dir /home/user/myapp --status 0
```

#### Command with explicit git context

```bash
shy insert --command "git commit -m 'fix bug'" \
  --dir /home/user/myapp \
  --git-repo "https://github.com/user/myapp.git" \
  --git-branch "feature/bugfix"
```

#### Command with custom timestamp

```bash
shy insert --command "make build" --dir /home/user/project --timestamp 1704067200
```

## Testing

Run all tests:

```bash
go test ./... -v
```

## Future Iterations

See `DEVELOPMENT_PLAN.md` for the roadmap:

- Iteration 2: Basic query interface (list, search, stats)
- Iteration 3: Rich metadata (duration, hostname, etc.)
- Iteration 4: Privacy & configuration
- Iteration 5: Work summarization
- And more...
