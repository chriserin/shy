# shy - Shell History Tracker Integration for Zsh
#
# This script enables automatic command tracking in zsh.
#
# Installation:
#   Add the following line to your ~/.zshrc:
#     eval "$(shy init zsh)"
#
# Configuration:
#   SHY_DISABLE=1    - Temporarily disable command tracking
#   SHY_DB_PATH      - Custom database path (default: $XDG_DATA_HOME/shy/history.db or ~/.local/share/shy/history.db)
#
# Troubleshooting:
#   Errors are logged to: $XDG_DATA_HOME/shy/error.log or ~/.local/share/shy/error.log
#
# Uninstall:
#   Remove the eval line from your ~/.zshrc and restart your shell

# Store command for tracking
__shy_cmd=""
__shy_cmd_dir=""
__shy_cmd_start=""

# Hook called before command execution
__shy_preexec() {
	# Check if tracking is disabled
	if [[ -n "$SHY_DISABLE" ]]; then
		return 0
	fi

	# Store the command and context
	__shy_cmd="$1"
	__shy_cmd_dir="$PWD"
	__shy_cmd_start="$(date +%s)"
}

# Hook called after command execution
__shy_precmd() {
	local exit_status=$?

	# Check if tracking is disabled
	if [[ -n "$SHY_DISABLE" ]]; then
		return 0
	fi

	# Only track if we have a command
	if [[ -z "$__shy_cmd" ]]; then
		return 0
	fi

	# Determine log directory (use XDG_DATA_HOME if set, otherwise fallback)
	local shy_data_dir="${XDG_DATA_HOME:-$HOME/.local/share}/shy"
	local shy_error_log="$shy_data_dir/error.log"

	# Build shy insert command
	local shy_args=(
		"insert"
		"--command" "$__shy_cmd"
		"--dir" "$__shy_cmd_dir"
		"--status" "$exit_status"
	)

	# Add timestamp if available
	if [[ -n "$__shy_cmd_start" ]]; then
		shy_args+=("--timestamp" "$__shy_cmd_start")
	fi

	# Add custom database path if set
	if [[ -n "$SHY_DB_PATH" ]]; then
		shy_args+=("--db" "$SHY_DB_PATH")
	fi

	# Execute shy insert in background, log errors if they occur
	(
		local error_output
		if ! error_output=$(shy "${shy_args[@]}" 2>&1); then
			# Ensure log directory exists
			mkdir -p "$shy_data_dir" 2>/dev/null
			# Log the error with timestamp and error message
			{
				echo "[$(date '+%Y-%m-%d %H:%M:%S')] Error executing shy insert"
				echo "  Command: $__shy_cmd"
				echo "  Working dir: $__shy_cmd_dir"
				echo "  Error output: $error_output"
				echo ""
			} >> "$shy_error_log" 2>/dev/null
		fi
	) &!

	# Clear stored command
	__shy_cmd=""
	__shy_cmd_dir=""
	__shy_cmd_start=""
}

# Register hooks using zsh's standard hook system
autoload -Uz add-zsh-hook
add-zsh-hook preexec __shy_preexec
add-zsh-hook precmd __shy_precmd
