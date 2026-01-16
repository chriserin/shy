#!/bin/bash
# Claude Code Pre-Tool Use Hook for shy
# This script captures the start time of bash commands executed by Claude Code

# Read JSON from stdin
INPUT=$(cat)

# Extract command
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command')

# Create a unique ID for this command (hash of command + timestamp)
CMD_ID=$(echo "${COMMAND}${RANDOM}" | md5sum | cut -d' ' -f1)

# Store start time in temp file
START_TIME=$(date +%s)
echo "$START_TIME" > "/tmp/shy-timer-${CMD_ID}"

# Store the command ID for the post hook to find
echo "$CMD_ID" > /tmp/shy-last-cmd-id

exit 0
