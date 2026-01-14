# shy fzf history widget
# Replacement for fzf's default history widget that uses shy as the data source
#
# Installation:
#   1. Ensure fzf is installed and its key-bindings are loaded
#   2. Source this file:
#      source <(shy init zsh --fzf)  # or wherever this file is located
#   3. Bind the widget (overwrites fzf's default Ctrl-R):
#      bindkey '^R' shy-fzf-history-widget
#
# Features:
#   - Uses shy database instead of zsh history array
#   - Supports multi-select (select multiple commands with Tab)
#   - Commands are automatically deduplicated (most recent occurrence)
#   - All filtering done interactively in fzf
#   - Supports all fzf options via FZF_CTRL_R_OPTS
#
# Environment variables:
#   FZF_CTRL_R_OPTS - Additional options to pass to fzf

shy-fzf-history-widget() {
	local selected
	setopt localoptions noglobsubst noposixbuiltins pipefail no_aliases no_glob no_ksharrays extendedglob 2>/dev/null

	# Get history from shy and pipe to fzf
	# shy fzf outputs: event_number<TAB>command<NULL>...
	# fzf options:
	#   -n2..,..           - Match from field 2 onwards (skip event number)
	#   --scheme=history   - Use history matching scheme
	#   --bind=ctrl-r      - Toggle sort order with Ctrl-R
	#   --wrap-sign        - Wrap indicator for long lines
	#   --highlight-line   - Highlight current line
	#   --multi            - Allow selecting multiple entries with Tab
	#   --read0            - Read null-terminated input
	#   --query            - Pre-fill with current buffer
	selected="$(shy fzf 2>/dev/null | \
		FZF_DEFAULT_OPTS=$(__fzf_defaults "" "-n2..,.. --scheme=history --bind=ctrl-r:toggle-sort --wrap-sign '\t↳ ' --highlight-line --multi ${FZF_CTRL_R_OPTS-} --query=${(qqq)LBUFFER} --read0") \
		FZF_DEFAULT_OPTS_FILE='' $(__fzfcmd))"

	local ret=$?
	local -a cmds
	local -a mbegin mend match

	if [ -n "$selected" ]; then
		# Parse selected output: each line is "event_number<TAB>command"
		for line in ${(ps:\n:)selected}; do
			# Match pattern: number<TAB>anything
			if [[ $line == (#b)(<->)(#B)$'\t'* ]]; then
				# Extract event number
				local event_num="${match[1]}"

				# Get exact command text by looking up event number in shy
				# Use -n to suppress event number in output
				local cmd_text
				cmd_text=$(shy fc -l -n "$event_num" "$event_num" 2>/dev/null)

				# Add to commands array if we got something
				if [[ -n "$cmd_text" ]]; then
					# Strip leading/trailing whitespace
					cmd_text="${cmd_text#"${cmd_text%%[![:space:]]*}"}"
					cmd_text="${cmd_text%"${cmd_text##*[![:space:]]}"}"
					cmds+=("${cmd_text}")
				fi
			fi
		done

		# Build BUFFER from selected commands
		if (( ${#cmds[@]} )); then
			# Join multiple commands with newlines, strip trailing newlines
			BUFFER="${(pj:\n:)${(@)cmds%%$'\n'#}}"
			CURSOR=${#BUFFER}
		fi
	fi

	zle reset-prompt
	return $ret
}

# Register as a zle widget
zle -N shy-fzf-history-widget

# Optional: Auto-bind to Ctrl-R (uncomment to enable)
# This will override fzf's default Ctrl-R binding
# bindkey '^R' shy-fzf-history-widget
