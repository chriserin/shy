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
- `--db`: Database file path (default: ~/.local/share/shy/history.db)

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
