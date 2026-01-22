# shy - Autosuggest Integration for Zsh
#
# This script provides shy-based strategy functions for zsh-autosuggestions plugin.
#
# Installation:
#   Add to your ~/.zshrc after loading zsh-autosuggestions:
#     source "$(shy init zsh --autosuggest)"
#     ZSH_AUTOSUGGEST_STRATEGY=(shy_history shy_match_prev_cmd)
#
# Available Strategies:
#   - shy_history: Simple prefix matching (replaces default history strategy)
#   - shy_match_prev_cmd: Context-aware matching based on previous command
#   - shy_pwd: Directory-specific suggestions
#   - shy_session: Session-specific suggestions
#
# Configuration:
#   ZSH_AUTOSUGGEST_HISTORY_IGNORE - Pattern to exclude from suggestions (same as default)

# Strategy 1: Simple history matching using shy
_zsh_autosuggest_strategy_shy_history() {
	emulate -L zsh

	local prefix="$1"

	# Build shy command arguments
	local shy_args=("like-recent" "$prefix")

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

# Strategy 2: Context-aware matching using shy (matches previous command)
_zsh_autosuggest_strategy_shy_match_prev_cmd() {
	emulate -L zsh

	local prefix="$1"

	# Get previous command from shy
	local prev_cmd
	prev_cmd=$(shy last-command --current-session -n 2 2>/dev/null)

	# If we can't get previous command, fall back to empty suggestion
	if [[ -z $prev_cmd ]]; then
		typeset -g suggestion=""
		return
	fi

	# Build shy command arguments
	local shy_args=(
		"like-recent-after"
		"$prefix"
		--prev "$prev_cmd"
	)

	# Add exclude pattern if set
	if [[ -n $ZSH_AUTOSUGGEST_HISTORY_IGNORE ]]; then
		shy_args+=(--exclude "$ZSH_AUTOSUGGEST_HISTORY_IGNORE")
	fi

	# Query shy for contextual suggestion (suppress errors)
	local result
	result=$(shy "${shy_args[@]}" 2>/dev/null)

	# Set global suggestion variable (required by zsh-autosuggestions)
	typeset -g suggestion="$result"
}

# Strategy 3: Directory-aware suggestions
_zsh_autosuggest_strategy_shy_pwd() {
	emulate -L zsh

	local prefix="$1"

	# Build shy command with pwd filter
	local shy_args=(
		"like-recent"
		"$prefix"
		--pwd
	)

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

# Strategy 4: Session-aware suggestions
_zsh_autosuggest_strategy_shy_session() {
	emulate -L zsh

	local prefix="$1"

	# Build shy command with session filter
	local shy_args=(
		"like-recent"
		"$prefix"
		--session
	)

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
