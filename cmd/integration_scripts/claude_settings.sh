#!/bin/bash
# Merge shy hooks configuration into Claude Code settings

# Check if jq is installed
if ! command -v jq &> /dev/null; then
  echo "Error: jq is required but not installed." >&2
  echo "Install it with:" >&2
  echo "  - macOS: brew install jq" >&2
  echo "  - Ubuntu/Debian: sudo apt-get install jq" >&2
  echo "  - Fedora: sudo dnf install jq" >&2
  echo "  - Arch: sudo pacman -S jq" >&2
  exit 1
fi

SETTINGS_FILE=~/.claude/settings.json
if [ ! -f "$SETTINGS_FILE" ]; then
  mkdir -p ~/.claude
  echo '{}' > "$SETTINGS_FILE"
fi

# Check if PreToolUse hook already exists
PRE_HOOK_EXISTS=$(jq -r '
  .hooks.PreToolUse // [] |
  any(
    .matcher == "Bash" and
    (.hooks // [] | any(.command == "~/.claude/hooks/shy-start-timer.sh"))
  )
' "$SETTINGS_FILE")

if [ "$PRE_HOOK_EXISTS" = "false" ]; then
  # Create temporary file with new hook configuration
  TEMP_HOOKS=$(mktemp)

  # Add PreToolUse hook for starting timer
  cat > "$TEMP_HOOKS" << 'HOOKEOF'
{
  "matcher": "Bash",
  "hooks": [
    {
      "type": "command",
      "command": "~/.claude/hooks/shy-start-timer.sh",
      "timeout": 5
    }
  ]
}
HOOKEOF

  # Merge PreToolUse hook
  jq --argjson newhook "$(cat "$TEMP_HOOKS")" \
    '.hooks.PreToolUse = (.hooks.PreToolUse // []) + [$newhook]' \
    "$SETTINGS_FILE" > "${SETTINGS_FILE}.tmp" && mv "${SETTINGS_FILE}.tmp" "$SETTINGS_FILE"

  rm -f "$TEMP_HOOKS"
  echo "Added PreToolUse hook"
else
  echo "PreToolUse hook already exists, skipping"
fi

# Check if PostToolUse hook already exists
POST_HOOK_EXISTS=$(jq -r '
  .hooks.PostToolUse // [] |
  any(
    .matcher == "Bash" and
    (.hooks // [] | any(.command == "~/.claude/hooks/capture-to-shy.sh"))
  )
' "$SETTINGS_FILE")

if [ "$POST_HOOK_EXISTS" = "false" ]; then
  # Create temporary file with new hook configuration
  TEMP_HOOKS=$(mktemp)

  # Add PostToolUse hook for capturing commands
  cat > "$TEMP_HOOKS" << 'HOOKEOF'
{
  "matcher": "Bash",
  "hooks": [
    {
      "type": "command",
      "command": "~/.claude/hooks/capture-to-shy.sh",
      "timeout": 5
    }
  ]
}
HOOKEOF

  # Merge PostToolUse hook
  jq --argjson newhook "$(cat "$TEMP_HOOKS")" \
    '.hooks.PostToolUse = (.hooks.PostToolUse // []) + [$newhook]' \
    "$SETTINGS_FILE" > "${SETTINGS_FILE}.tmp" && mv "${SETTINGS_FILE}.tmp" "$SETTINGS_FILE"

  rm -f "$TEMP_HOOKS"
  echo "Added PostToolUse hook"
else
  echo "PostToolUse hook already exists, skipping"
fi

echo "Hooks configuration complete for $SETTINGS_FILE"
