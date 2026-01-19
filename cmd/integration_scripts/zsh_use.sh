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
__shy_history_index=1

_shy_up_line_or_history() {
  # If the last widget was also _shy_up_line_or_history (or an autosuggestion widget), keep cycling
  # Otherwise, reset to 1 (most recent)
  if [[ "$LASTWIDGET" != "_shy_up_line_or_history" && "$LASTWIDGET" != autosuggest-* ]]; then
    __shy_history_index=1
  else
    # We're continuing to navigate backward, increment to get older command
    ((__shy_history_index++))
  fi

  # Get the command at the current index (1-based: 1=most recent, 2=second most recent, etc.)
  local cmd=$(shy last-command -n $__shy_history_index)

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
zle -N _shy_up_line_or_history
bindkey '^[[A' _shy_up_line_or_history # Standard up arrow
bindkey '^[OA' _shy_up_line_or_history # Application mode up arrow

# Down Arrow: Cycle forward through command history (towards more recent)
_shy_down_line_or_history() {
  # If the last widget was also _shy_down_line_or_history (or an autosuggestion widget), keep cycling
  # Otherwise, use current history index from up/down arrow navigation
  if [[ "$LASTWIDGET" != "_shy_down_line_or_history" && "$LASTWIDGET" != autosuggest-* && "$LASTWIDGET" != "_shy_up_line_or_history" ]]; then
    __shy_history_index=1
  else
    # We're continuing to navigate forward, decrement to get more recent command
    if [[ $__shy_history_index -gt 1 ]]; then
      ((__shy_history_index--))
    else
      # Already at most recent (index 1), clear the buffer
      BUFFER=""
      CURSOR=0
      zle reset-prompt
      return
    fi
  fi

  # Get the command at the current index (1-based: 1=most recent, 2=second most recent, etc.)
  local cmd=$(shy last-command -n $__shy_history_index)

  if [[ -n "$cmd" ]]; then
    BUFFER="$cmd"
    CURSOR=$#BUFFER
  fi

  zle reset-prompt
}
zle -N _shy_down_line_or_history
bindkey '^[[B' _shy_down_line_or_history # Standard down arrow
bindkey '^[OB' _shy_down_line_or_history # Application mode down arrow

alias history="shy history"
alias fc="shy fc"
alias r="shy fc -s"

_shy_bind_viins() {
  zvm_bindkey viins '^[[A' _shy_up_line_or_history   # Standard up arrow
  zvm_bindkey viins '^[OA' _shy_up_line_or_history   # Application mode up arrow
  zvm_bindkey viins '^[[B' _shy_down_line_or_history # Standard down arrow
  zvm_bindkey viins '^[OB' _shy_down_line_or_history # Application mode down arrow
  zvm_bindkey vicmd '^[[A' _shy_up_line_or_history   # Standard up arrow
  zvm_bindkey vicmd '^[OA' _shy_up_line_or_history   # Application mode up arrow
  zvm_bindkey vicmd '^[[B' _shy_down_line_or_history # Standard down arrow
  zvm_bindkey vicmd '^[OB' _shy_down_line_or_history # Application mode down arrow
}

zvm_after_init_commands+=(_shy_bind_viins)
