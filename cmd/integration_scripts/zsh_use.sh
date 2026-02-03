# shy - Shell History Usage Integration for Zsh
#
# This script enables interactive history lookup and completion using shy.
#
# Installation:
#   Add the following line to your ~/.zshrc:
#     eval "$(shy init zsh --use)"
#
# Features:
#   - Ctrl-R: Interactive history search
#   - Up Arrow: Navigate backward through command history (older commands)
#   - Down Arrow: Navigate forward through command history (newer commands)
#   - Ctrl-O: Accept line and show next (newer) history entry (for replaying sequences)
#   - Right Arrow: Complete with most recent matching command

# Ctrl-R: Interactive history search
_shy_isearch() {
  local search="" char match

  while true; do
    zle -R "bck-i-search: $search"
    read -k char

    case $char in
    $'\r' | $'\n') break ;;                  # Enter - accept
    $'\x7f' | $'\b') search="${search%?}" ;; # Backspace
    $'\x03' | $'\x07')
      BUFFER=""
      break
      ;;                                         # Ctrl-C/G - cancel
    $'\e') read -k -t 0.01 && read -k -t 0.01 ;; # Discard escape sequences
    [[:print:]]) search+="$char" ;;              # Only accept printable chars
    esac

    if [[ -n "$search" ]]; then
      match=$(shy fc -l -n -1000 | grep -F "$search" | tail -1)
      [[ -n "$match" ]] && BUFFER="$match"
    fi
  done

  CURSOR=$#BUFFER
  zle reset-prompt
}

_shy_shell_history() {
  if command -v fzf &>/dev/null; then
    local selected event_num

    # shy fzf outputs: event_number<TAB>command<NULL>
    # --read0: read null-terminated input (handles commands with newlines)
    # -n2..: match from field 2 onwards (skip event number)
    selected=$(shy fzf | fzf --read0 -n2.. --scheme=history --height=40%)

    if [[ -n "$selected" ]]; then
      # Extract event number (first field before tab)
      event_num="${selected%%$'\t'*}"
      # Look up exact command by event number (handles newlines/tabs safely)
      selected=$(shy fc -l -n "$event_num" "$event_num" 2>/dev/null)
    fi

    if [[ -n "$selected" ]]; then
      BUFFER="$selected"
      CURSOR=$#BUFFER
    fi

    zle reset-prompt
  else
    _shy_isearch
  fi
}
zle -N shy-shell-history _shy_shell_history

# Up Arrow: Cycle through command history
__shy_history_index=0
__shy_replay_index=0  # Persists across prompts for accept-line-and-down-history

# Reset history index when a new line starts
_shy_reset_history_index() {
  # Check if we're in replay mode (accept-line-and-down-history)
  if [[ $__shy_replay_index -gt 0 ]]; then
    __shy_history_index=$__shy_replay_index
    __shy_replay_index=0
    # Pre-fill the buffer with the next command in sequence
    local cmd=$(shy last-command --current-session -n $__shy_history_index)
    if [[ -n "$cmd" ]]; then
      BUFFER="$cmd"
      CURSOR=$#BUFFER
    fi
  else
    __shy_history_index=0
  fi
}
zle -N shy-reset-history-index _shy_reset_history_index

# Register the reset hook (add-zle-hook-widget allows multiple hooks without clobbering)
autoload -Uz add-zle-hook-widget 2>/dev/null
if typeset -f add-zle-hook-widget >/dev/null; then
  add-zle-hook-widget zle-line-init shy-reset-history-index
fi

_shy_up_line_or_history() {
  ((__shy_history_index++))

  # Get the command at the current index (1-based: 1=most recent, 2=second most recent, etc.)
  local cmd=$(shy last-command --current-session -n $__shy_history_index)

  if [[ -n "$cmd" ]]; then
    BUFFER="$cmd"
    CURSOR=$#BUFFER
  else
    # No command at this index, we've gone too far back
    # Decrement to stay at the last valid command
    ((__shy_history_index--))
  fi

  zle reset-prompt
}
zle -N shy-up-line-or-history _shy_up_line_or_history
bindkey '^[[A' shy-up-line-or-history # Standard up arrow
bindkey '^[OA' shy-up-line-or-history # Application mode up arrow

# Down Arrow: Cycle forward through command history (towards more recent)
_shy_down_line_or_history() {
  if [[ $__shy_history_index -gt 0 ]]; then
    ((__shy_history_index--))
  else
    BUFFER=""
    CURSOR=0
    zle reset-prompt
    return
  fi

  # Get the command at the current index (1-based: 1=most recent, 2=second most recent, etc.)
  local cmd=$(shy last-command --current-session -n $__shy_history_index)

  if [[ -n "$cmd" ]]; then
    BUFFER="$cmd"
    CURSOR=$#BUFFER
  else
    BUFFER=""
    CURSOR=0
  fi

  zle reset-prompt
}
zle -N shy-down-line-or-history _shy_down_line_or_history
bindkey '^[[B' shy-down-line-or-history # Standard down arrow
bindkey '^[OB' shy-down-line-or-history # Application mode down arrow

# Ctrl-O: Accept line and move to next (newer) history entry
# Useful for replaying a sequence of commands
_shy_accept_line_and_down_history() {
  # If we're navigating history, set up to show the next newer command
  # Note: After the command executes and is recorded, all indices shift up by 1
  # So the "next" command (old index-1) will be at the same numeric index
  if [[ $__shy_history_index -gt 1 ]]; then
    __shy_replay_index=$__shy_history_index
  else
    __shy_replay_index=0
  fi

  # Accept the current line (execute it)
  # The zle-line-init hook will pick up __shy_replay_index and pre-fill the buffer
  zle accept-line
}
zle -N shy-accept-line-and-down-history _shy_accept_line_and_down_history
bindkey '^O' shy-accept-line-and-down-history

alias history="shy history"
alias fc="shy fc"
alias r="shy fc -s"

_shy_bind_viins() {
  zvm_bindkey viins '^[[A' shy-up-line-or-history   # Standard up arrow
  zvm_bindkey viins '^[OA' shy-up-line-or-history   # Application mode up arrow
  zvm_bindkey viins '^[[B' shy-down-line-or-history # Standard down arrow
  zvm_bindkey viins '^[OB' shy-down-line-or-history # Application mode down arrow
  zvm_bindkey viins '^O' shy-accept-line-and-down-history # Accept and show next
  zvm_bindkey vicmd '^[[A' shy-up-line-or-history   # Standard up arrow
  zvm_bindkey vicmd '^[OA' shy-up-line-or-history   # Application mode up arrow
  zvm_bindkey vicmd '^[[B' shy-down-line-or-history # Standard down arrow
  zvm_bindkey vicmd '^[OB' shy-down-line-or-history # Application mode down arrow
  zvm_bindkey vicmd '^O' shy-accept-line-and-down-history # Accept and show next
}

zvm_after_init_commands+=(_shy_bind_viins)

_shy_bind_viins_ctrl_r() {
  zvm_bindkey viins '^R' shy-shell-history
  zvm_bindkey vicmd '^R' shy-shell-history
}

# Uncomment for ctrl-r
# zvm_after_init_commands+=(_shy_bind_viins_ctrl_r)
# bindkey '^R' shy-shell-history
