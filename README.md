# shy - Shell History Tracker

A command-line tool for tracking shell command history in SQLite with rich metadata and opinionated behaviour.

## Features

- **Metadata**: start time, duration, working directory, git repo, git branch, session
- **Integrations**: zsh, zvm, television, fzf, zsh-autosuggestion, claude
- **Summarization**: Reports to help summarize a previous period's activity
- **Opinionated Command Duplication Behaviour**
- **Opinionated Context Scoping**

## Scopes

There are 3 different scopes for `shy`.  The current session, the current
working directory, and all.  These scopes support 3 different use cases:

1. up/down arrow: Traverse previous commands of the **current session**.
2. autosuggest: Suggest previous commands that have been used in the **current working directory**.
3. ctrl-r: Search through **all** previous commands

### Compared to ZSH

ZSH has a number of [options](https://zsh.sourceforge.io/Doc/Release/Options.html) that allow you to customize when a
command is saved to the history file and when a command is
read from the history file, but you have to choose between having access to a
command from another session and having a session history that is linear and
cohesive.  `shy` is opinionated about which use case should have access to
which commands in a prioritized manner.

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

### Compared to ZSH

It is not possible in zsh to support all 3 use cases.  Duplicates in zsh are managed
at the command time, not read time, so you have to choose one of the scenarios
supported by the HIST_IGNORE_ALL_DUPS or HIST_IGNORE_DUPS options.

In `shy`, all commands are stored, enabling the application to be intentional
about when duplicates are presented or not presented according to each use case.

## Integration

### Progressive Integration

You can integrate `shy` into your environment gradually, taking advantage of
shy's ability to record commands without utilizing shy's history or impacting
zsh's ability to collect history.

This command will enable shy to record history alongside zsh:

```sh
eval "$(shy init zsh --record)"
```

It provides the zsh hooks to record shy history, but not the zle functions to access that history through up/down arrows, ctrl-r or through auto suggestions.

To stop using zsh's history and instead use shy's history, you can use the command:

```sh
eval "$(shy init zsh --use)"
```

This command adds aliases for `history`, `fc` and `r` as well as defining zle widgets for up/down arrows and ctrl-r.

Additionally, if you use zsh_autosuggestion, you can add a zle widget with:

```sh
eval "$(shy init zsh --autosuggest)"
```

This command will add a zle widget for autosuggest, which can then be utilized with:

```sh
export ZSH_AUTOSUGGEST_STRATEGY=(shy_history)
```

If you feel confident in `shy`'s ability to integrate into your system, you can then stop zsh history collection with:

```
export HISTSIZE=1
```

### ZSH

Shy can automatically track your shell history by integrating with zsh. Add this to your `.zshrc`:

```sh
eval "$(shy init zsh)"
```


### FZF

For a better history search experience with fzf:

```sh
# Bind ctrl-r for history search
bindkey '^R' shy-fzf-history-widget
```

Place this after any fzf key bindings to ensure it overwrites other ctrl-r bindings.

### zsh-autosuggestions

Shy provides a custom strategy for zsh-autosuggestions, `shy_history`.

```sh
ZSH_AUTOSUGGEST_STRATEGY=(shy_history)
```

This can be placed at any point in your zsh setup.

### television

Television's history integration depends on the `history` command.  Shy aliases the `history` to `shy history` so no other steps are needed.

If you would like to add a channel to your command history with [television](https://github.com/alexpasmantier/television), a fuzzy finder TUI:

```bash
# Generate the television channel configuration
shy tv config > ~/.config/television/cable/shy-history.toml
```

Your shy history is now available through television with the command `tv shy-history`.  This
will provide a preview window of the command that provides all meta-data plus
the 5 commands run before and the 5 commands run after the command in that
session.

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
| `tv init` | N/A | N/A | Generate television channel configuration |


**Scope Types:**
- **ALL**: Searches entire command history across all sessions and directories
- **SESSION**: Current shell session (filtered by `source_app` and `source_pid`)
- **PWD**: Current working directory

**Duplicate Behavior:**
- **DUPS**: Returns all commands including duplicates
- **NO DUPS**: Filters out duplicate commands (keeps most recent)
- **NO SEQ DUPS**: Filters out consecutive duplicate commands
- **Single Result**: Returns only the most recent matching command
- **N/A**: Not applicable

### Insert Command

Insert a command into the history database:

```bash
shy insert --command "ls -la" --dir /home/user/project --status 0
```

## PERFORMANCE

The performance goal is for all commands to execute in under 20ms for databases with command counts up to 5 million.


| Use Case | Command | 10K | 1MIL | 5MIL |
|----------|---------|-----|------|------|
| up/arrow | `shy last-command`| 1.2ms | 2.6ms     |  2.9ms    |
| ctrl-r (fzf) | `shy fzf` | 1.5ms  | 1.5ms | 1.5ms  |
| ctrl-r (tv) | `shy history` | 1.6ms | 1.6ms | 1.6ms |
| autosuggest | `shy like-recent` | 0.6ms | 0.7ms | 0.8ms |
