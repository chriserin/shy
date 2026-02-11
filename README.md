# shy

[![license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/chriserin/shy/main/LICENSE)
[![Tests](https://github.com/chriserin/shy/actions/workflows/test.yml/badge.svg)](https://github.com/chriserin/shy/actions/workflows/test.yml)

An opinionated, performant, and metadata rich command history tool.

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Scopes](#scopes)
- [Duplicates](#duplicates)
- [Integration](#integration)
  - [Progressive Integration](#progressive-integration)
  - [ZSH](#zsh)
  - [FZF](#fzf)
  - [zsh-autosuggestions](#zsh-autosuggestions)
  - [television](#television)
  - [zsh-vim-mode (zvm)](#zsh-vim-mode-zvm)
  - [claude code](#claude-code)
- [Configuration](#configuration)
- [Commands](#commands)
- [Performance](#performance)

## Features

- **Metadata**: start time, duration, working directory, git repo, git branch, session
- **Integrations**: zsh, zvm, television, fzf, zsh-autosuggestion, claude code
- **Summarization**: Reports to help summarize a previous period's activity
- **Opinionated Command Duplication Behaviour**
- **Opinionated Context Scoping**

## Installation

Install from the [releases page](https://github.com/chriserin/shy/releases) or install with mise `mise install github:chriserin/shy`.

## QUICK START

Full, integration, just go for it:

```sh
shy init-db # create the database
shy fc -R $HISTFILE # read data into the database from your history file
```

Somewhere in your dot files:

```sh
eval "$(shy init)"
bindkey '^R' shy-shell-history # if you use fzf or native history search
export ZSH_AUTOSUGGEST_STRATEGY=(shy_history) # if you use autosuggest
zvm_after_init_commands+=(_shy_bind_viins) # if you use zvm
zvm_after_init_commands+=(_shy_bind_viins_ctrl_r) # if you use zvm + fzf or native search
export HISTSIZE=1 # fully minimize the zsh history system
```

If you just want to collect commands in parallel with your current zsh setup:

```sh
eval "$(shy init --record)" # anywhere in your dotfiles
```

## Scopes

There are 3 different scopes for `shy`. The current session, the current
working directory, and all. These scopes support 3 different use cases:

1. up/down arrow: Traverse previous commands of the **current session**.
2. autosuggest: Suggest previous commands that have been used in the **current working directory**.
3. ctrl-r: Search through **all** previous commands

### Compared to ZSH

ZSH has a number of [options](https://zsh.sourceforge.io/Doc/Release/Options.html) that allow you to customize when a
command is saved to the history file and when a command is
read from the history file, but you have to choose between having access to a
command from another session and having a session history that is linear and
cohesive. `shy` is opinionated about which use case should have access to
which command scope in a prioritized manner.

## Duplicates

There is a tension between collection duplicates and not collecting duplicates.

1. **DUPS**: Command Context: Useful when re-executing commands in order with
   the "accept-line-and-down-history" widget, or understanding how a command
   was in context with other commands.
2. **NO DUPS**: Searching: It is not useful to find 10, 20, or 100 of the same
   command when searching through commands
3. **NO SEQ DUPS**: Traversing: When moving through previous commands with the
   up or down errors, it's not useful to traverse the same command more than
   once.

### Compared to ZSH

It is not possible in zsh to support all 3 use cases. Duplicates in zsh are
managed at the command time - by not storing duplicates - not read time, so you
have to choose one of the scenarios supported by the HIST_IGNORE_ALL_DUPS or
HIST_IGNORE_DUPS options.

In `shy`, all commands are stored, enabling the application to be intentional
about when duplicates are presented or not according to each use case.

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

```sh
export HISTSIZE=1
```

### ZSH

Shy can automatically track your shell history by integrating with zsh. Add this to your `.zshrc`:

```sh
eval "$(shy init zsh)"
```

Additionally, bind ctrl-r to the shy shell history widget to search the shy history database.

```sh
bindkey '^R' shy-shell-history
```

### FZF

To use fzf as the fuzzy finder for your shy search history:

```sh
bindkey '^R' shy-shell-history
```

Place this after any fzf key bindings to ensure it overwrites other ctrl-r bindings.

### zsh-autosuggestions

Shy provides a custom strategy for zsh-autosuggestions, `shy_history`.

```sh
ZSH_AUTOSUGGEST_STRATEGY=(shy_history)
```

This can be placed at any point in your zsh setup.

### television

Television's history integration depends on the `history` command. Shy aliases the `history` to `shy history` so no other steps are needed.

If you would like to add a channel to your command history with [television](https://github.com/alexpasmantier/television), a fuzzy finder TUI:

```bash
# Generate the television channel configuration
shy tv config > ~/.config/television/cable/shy-history.toml
```

Your shy history is now available through television with the command `tv shy-history`. This
will provide a preview window of the command that provides all meta-data plus
the 5 commands run before and the 5 commands run after the command in that
session.

### zsh-vim-mode (zvm)

To utilize the `shy-shell-history` widget when using [zvm](https://github.com/jeffreytse/zsh-vi-mode), add an init function to the `zvm_after_init_commands`.

```sh
zvm_after_init_commands+=(_shy_bind_viins_ctrl_r)
```

### claude code

Store the commands that claude code runs with it's Bash tool:

```sh
shy claude-init
```

This will create hook functions located in the ~/.claude/hooks/capture-to-shy.sh file and add an entry in the ~/.claude/settings.json file.

## Configuration

Control tracking behavior with environment variables:

```bash
# disable tracking
SHY_DISABLE=1

# insert to a db of your choosing
SHY_DB_PATH=/path/to/custom.db
```

## Commands

### Command Overview

| Command          | Default Scope | Duplicates    | Description                                                                                   |
| ---------------- | ------------- | ------------- | --------------------------------------------------------------------------------------------- |
| `fc` / `history` | ALL           | NO DUPS       | List or edit command history (use `--internal` for session, `--local` for pwd)                |
| `list`           | ALL           | DUPS          | List recent commands (use `--session` or `--current-session` to filter)                       |
| `list-all`       | ALL           | DUPS          | List all commands (use `--session` or `--current-session` to filter)                          |
| `last-command`   | SESSION + PWD | NO SEQ DUPS   | Get most recent command (unions session with current directory, skips consecutive duplicates) |
| `fzf`            | ALL           | NO DUPS       | Output history for fzf integration (SQL-based deduplication)                                  |
| `like-recent`    | ALL           | Single Result | Find most recent command with prefix (use `--pwd` or `--session` to filter)                   |
| `summary`        | ALL           | N/A           | Show aggregated activity report (use `--source-app` to filter)                                |
| `insert`         | N/A           | N/A           | Manually insert a command into the database                                                   |
| `close-session`  | N/A           | N/A           | Mark a session as inactive                                                                    |
| `init`           | N/A           | N/A           | Generate shell integration scripts                                                            |
| `tv init`        | N/A           | N/A           | Generate television channel configuration                                                     |

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

| Use Case     | Command            | 10K   | 1MIL  | 5MIL  |
| ------------ | ------------------ | ----- | ----- | ----- |
| up/arrow     | `shy last-command` | 1.2ms | 2.6ms | 2.9ms |
| ctrl-r (fzf) | `shy fzf`          | 1.5ms | 1.5ms | 1.5ms |
| ctrl-r (tv)  | `shy history`      | 1.6ms | 1.6ms | 1.6ms |
| autosuggest  | `shy like-recent`  | 0.6ms | 0.7ms | 0.8ms |
