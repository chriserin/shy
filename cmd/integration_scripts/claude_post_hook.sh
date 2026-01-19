#!/bin/bash
# Claude Code Post-Tool Use Hook for shy
# This script captures bash commands executed by Claude Code into shy database

# Read JSON from stdin
INPUT=$(cat)

# Extract fields
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command')
EXIT_CODE=$(echo "$INPUT" | jq -r '.tool_response.exit_code // .tool_response.return_code // 0')
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // ""')
END_TIME=$(date +%s)

# Convert session_id to numeric PID (hash it and take first 5 digits)
if [ -n "$SESSION_ID" ]; then
  SOURCE_PID=$(echo "$SESSION_ID" | md5sum | tr -d 'a-z-' | cut -c1-5)
else
  SOURCE_PID=$$
fi

# Get the command ID from pre-hook
CMD_ID=$(cat /tmp/shy-last-cmd-id 2>/dev/null || echo "")

# Calculate duration if we have timing info
DURATION=""
if [ -n "$CMD_ID" ] && [ -f "/tmp/shy-timer-${CMD_ID}" ]; then
  START_TIME=$(cat "/tmp/shy-timer-${CMD_ID}")
  DURATION=$((END_TIME - START_TIME))
  # Clean up temp files
  rm -f "/tmp/shy-timer-${CMD_ID}"
  rm -f /tmp/shy-last-cmd-id
fi

# Get working directory from JSON (Claude provides this as 'cwd')
WORKING_DIR=$(echo "$INPUT" | jq -r '.cwd // .working_directory // ""')

# Get git context (run in the working directory)
GIT_REPO=""
GIT_BRANCH=""
if [ -n "$WORKING_DIR" ] && [ -d "$WORKING_DIR" ]; then
  cd "$WORKING_DIR" 2>/dev/null
  if git rev-parse --git-dir >/dev/null 2>&1; then
    GIT_REPO=$(git config --get remote.origin.url 2>/dev/null || echo "")
    GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
  fi
fi

# Build shy insert command
# Escape double quotes in the command so they survive the eval
ESCAPED_COMMAND="${COMMAND//\"/\\\"}"
INSERT_CMD="shy insert --command \"$ESCAPED_COMMAND\" --status $EXIT_CODE --dir \"$WORKING_DIR\" --source-app claude-code --source-pid $SOURCE_PID"

# Add duration if available
if [ -n "$DURATION" ]; then
  INSERT_CMD="$INSERT_CMD --duration $DURATION"
fi

# Add git context if available
if [ -n "$GIT_REPO" ]; then
  INSERT_CMD="$INSERT_CMD --git-repo \"$GIT_REPO\""
fi
if [ -n "$GIT_BRANCH" ]; then
  INSERT_CMD="$INSERT_CMD --git-branch \"$GIT_BRANCH\""
fi

# Execute
eval "$INSERT_CMD" 2>/dev/null

exit 0
