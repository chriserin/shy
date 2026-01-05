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
#   - Up Arrow: Navigate to previous command
#   - Right Arrow: Complete with most recent matching command

# Ctrl-R: Interactive history search
shy_ctrl_r() {
	local selected

	# Check if fzf is available
	if command -v fzf &> /dev/null; then
		# Use fzf for interactive selection
		selected=$(shy list-all --fmt=timestamp,status,pwd,cmd | fzf --tac --no-sort \
			--preview 'echo {}' \
			--preview-window=up:3:wrap \
			--height=40% \
			--bind 'ctrl-y:execute-silent(echo -n {4..} | pbcopy)' \
			| awk -F'\t' '{print $4}')
	else
		# Fallback to basic completion
		selected=$(shy list-all | fzf --tac --no-sort --height=40%)
	fi

	if [[ -n "$selected" ]]; then
		BUFFER="$selected"
		CURSOR=$#BUFFER
	fi

	zle reset-prompt
}
zle -N shy_ctrl_r
bindkey '^R' shy_ctrl_r

# Up Arrow: Get last command
shy_up_arrow() {
	local last_cmd=$(shy last-command)

	if [[ -n "$last_cmd" ]]; then
		BUFFER="$last_cmd"
		CURSOR=$#BUFFER
	fi

	zle reset-prompt
}
zle -N shy_up_arrow
bindkey '^[[A' shy_up_arrow

# Right Arrow: Complete with like-recent
shy_right_arrow() {
	# Only complete if we have text and cursor is at end
	if [[ -n "$BUFFER" ]] && [[ $CURSOR -eq $#BUFFER ]]; then
		local completion=$(shy like-recent "$BUFFER")

		if [[ -n "$completion" ]]; then
			BUFFER="$completion"
			CURSOR=$#BUFFER
		fi
	else
		# Default right arrow behavior (move cursor)
		zle forward-char
	fi
}
zle -N shy_right_arrow
bindkey '^[[C' shy_right_arrow
