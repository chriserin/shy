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
#   - Right Arrow: Complete with most recent matching command

# Ctrl-R: Interactive history search
# _shy_shell_history() {
#   local selected
#
#   # Check if fzf is available
#   if command -v fzf &>/dev/null; then
#     # Use fzf for interactive selection
#     selected=$(shy list-all --fmt=timestamp,status,pwd,cmd | fzf --tac --no-sort \
#       --preview 'echo {}' \
#       --preview-window=up:3:wrap \
#       --height=40% \
#       --bind 'ctrl-y:execute-silent(echo -n {4..} | pbcopy)' |
#       awk -F'\t' '{print $4}')
#   else
#     # Fallback to basic completion
#     selected=$(shy list-all | fzf --tac --no-sort --height=40%)
#   fi
#
#   if [[ -n "$selected" ]]; then
#     BUFFER="$selected"
#     CURSOR=$#BUFFER
#   fi
#
#   zle reset-prompt
# }
# zle -N _shy_shell_history
# bindkey '^R' _shy_shell_history

# Up Arrow: Cycle through command history
__shy_history_index=0

# Reset history index when a new line starts
_shy_reset_history_index() {
  __shy_history_index=0
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

alias history="shy history"
alias fc="shy fc"
alias r="shy fc -s"

_shy_bind_viins() {
  zvm_bindkey viins '^[[A' shy-up-line-or-history   # Standard up arrow
  zvm_bindkey viins '^[OA' shy-up-line-or-history   # Application mode up arrow
  zvm_bindkey viins '^[[B' shy-down-line-or-history # Standard down arrow
  zvm_bindkey viins '^[OB' shy-down-line-or-history # Application mode down arrow
  zvm_bindkey vicmd '^[[A' shy-up-line-or-history   # Standard up arrow
  zvm_bindkey vicmd '^[OA' shy-up-line-or-history   # Application mode up arrow
  zvm_bindkey vicmd '^[[B' shy-down-line-or-history # Standard down arrow
  zvm_bindkey vicmd '^[OB' shy-down-line-or-history # Application mode down arrow
}

zvm_after_init_commands+=(_shy_bind_viins)
