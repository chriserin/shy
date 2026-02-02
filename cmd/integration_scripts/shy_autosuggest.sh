# shy - Autosuggest Integration for Zsh
#
# This script provides shy-based strategy functions for zsh-autosuggestions plugin.
#
# Installation:
#   Add to your ~/.zshrc after loading zsh-autosuggestions:
#     source "$(shy init zsh --autosuggest)"
#     ZSH_AUTOSUGGEST_STRATEGY=(shy_history)
#
# Available Strategies:
#   - shy_history: Simple prefix matching with session and pwd filters
#
# Configuration:
#   ZSH_AUTOSUGGEST_HISTORY_IGNORE - Pattern to exclude from suggestions (same as default)

# Strategy: Simple history matching using shy
_zsh_autosuggest_strategy_shy_history() {
  emulate -L zsh

  local prefix="$1"

  # Build shy command arguments
  local shy_args=("like-recent" "--session" "--pwd" "$prefix")

  # Add exclude pattern if set
  if [[ -n $ZSH_AUTOSUGGEST_HISTORY_IGNORE ]]; then
    shy_args+=(--exclude "$ZSH_AUTOSUGGEST_HISTORY_IGNORE")
  fi

  # Query shy for suggestion (suppress errors)
  local result
  result=$(shy "${shy_args[@]}" 2>/dev/null)

  # Set global suggestion variable (required by zsh-autosuggestions)
  typeset -g suggestion="$result"
}
