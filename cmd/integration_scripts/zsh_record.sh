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

# Export session PID for internal filtering (-I flag)
export SHY_SESSION_PID=$$

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
	# Capture start time in milliseconds (using date with %N for nanoseconds, falling back to seconds * 1000)
	if date +%N &>/dev/null 2>&1; then
		# GNU date supports nanoseconds
		__shy_cmd_start="$(date +%s%3N)"
	else
		# macOS/BSD date doesn't support %N, use seconds and append 000 for milliseconds
		__shy_cmd_start="$(($(date +%s) * 1000))"
	fi
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

	# Calculate duration if start time was captured
	local duration=""
	local timestamp=""
	if [[ -n "$__shy_cmd_start" ]]; then
		# Get current time in milliseconds
		local end_time
		if date +%N &>/dev/null 2>&1; then
			end_time="$(date +%s%3N)"
		else
			end_time="$(($(date +%s) * 1000))"
		fi

		# Calculate duration in milliseconds
		duration=$((end_time - __shy_cmd_start))

		# Timestamp for database (seconds since epoch)
		timestamp=$(((__shy_cmd_start + 500) / 1000))  # Round to nearest second
	fi

	sessionfile="$XDG_CACHE_HOME/shy/sessions/$SHY_SESSION_PID.txt"

	db=$(head -n 1 $sessionfile 2>/dev/null)

	# Build shy insert command
	local shy_args=(
		"insert"
		"--command" "$__shy_cmd"
		"--dir" "$__shy_cmd_dir"
		"--status" "$exit_status"
		"--db" "$db"
	)

	# Add timestamp if available
	if [[ -n "$timestamp" ]]; then
		shy_args+=("--timestamp" "$timestamp")
	fi

	# Add duration if calculated
	if [[ -n "$duration" ]] && [[ "$duration" -ge 0 ]]; then
		shy_args+=("--duration" "$duration")
	fi

	# Add source tracking fields
	shy_args+=("--source-app" "zsh")
	shy_args+=("--source-pid" "$$")

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

# Hook called when shell exits
__shy_session_close() {
	# Check if tracking is disabled
	if [[ -n "$SHY_DISABLE" ]]; then
		return 0
	fi

	# Build close-session command
	local shy_args=(
		"close-session"
		"--pid" "$$"
	)

	# Add custom database path if set
	if [[ -n "$SHY_DB_PATH" ]]; then
		shy_args+=("--db" "$SHY_DB_PATH")
	fi

	# Execute shy close-session in background
	# Use &! to disown the process so it continues even if shell exits
	(shy "${shy_args[@]}" 2>/dev/null) &!

	# Cleanup session file (for fc -p/-P stack management)
	(shy cleanup-session $$ 2>/dev/null) &!
}

# Register hooks using zsh's standard hook system
autoload -Uz add-zsh-hook
add-zsh-hook preexec __shy_preexec
add-zsh-hook precmd __shy_precmd
add-zsh-hook zshexit __shy_session_close
