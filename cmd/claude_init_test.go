package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClaudeInitCommand tests the basic claude-init command execution
func TestClaudeInitCommand(t *testing.T) {
	// Given: I want to set up Claude Code integration
	// When: I run "shy claude-init"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err, "claude-init command should succeed")

	output := buf.String()

	// Then: the command should output installation instructions
	assert.NotEmpty(t, output, "output should not be empty")
	assert.Contains(t, output, "Claude Code Integration for shy", "should have title")
	assert.Contains(t, output, "Follow these steps", "should have instructions")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestClaudeInitOutputStructure tests that the output has the correct structure
func TestClaudeInitOutputStructure(t *testing.T) {
	// Given: I run the claude-init command
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: it should have numbered steps
	assert.Contains(t, output, "# 1. Create the hooks directory", "should have step 1")
	assert.Contains(t, output, "# 2. Create the pre-hook script", "should have step 2")
	assert.Contains(t, output, "# 3. Create the post-hook script", "should have step 3")
	assert.Contains(t, output, "# 4. Make the scripts executable", "should have step 4")
	assert.Contains(t, output, "# 5. Merge hooks into Claude Code settings", "should have step 5")
	assert.Contains(t, output, "# 6. Reload Claude Code or restart the session", "should have step 6")

	// And: it should have a test section
	assert.Contains(t, output, "# Test the integration:", "should have test section")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitCreatesHooksDirectory tests mkdir command is included
func TestClaudeInitCreatesHooksDirectory(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should include mkdir command for hooks directory
	assert.Contains(t, output, "mkdir -p ~/.claude/hooks", "should create hooks directory")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitIncludesPreHookScript tests pre-hook script generation
func TestClaudeInitIncludesPreHookScript(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should create pre-hook script with heredoc
	assert.Contains(t, output, "cat > ~/.claude/hooks/shy-start-timer.sh << 'EOF'", "should create pre-hook file")

	// Pre-hook script should contain expected content
	assert.Contains(t, output, "# Claude Code Pre-Tool Use Hook for shy", "should have pre-hook header")
	assert.Contains(t, output, "INPUT=$(cat)", "should read JSON from stdin")
	assert.Contains(t, output, "COMMAND=$(echo \"$INPUT\" | jq -r '.tool_input.command')", "should extract command")
	assert.Contains(t, output, "START_TIME=$(date +%s)", "should capture start time")
	assert.Contains(t, output, "echo \"$START_TIME\" > \"/tmp/shy-timer-${CMD_ID}\"", "should store start time")
	assert.Contains(t, output, "echo \"$CMD_ID\" > /tmp/shy-last-cmd-id", "should store command ID")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitIncludesPostHookScript tests post-hook script generation
func TestClaudeInitIncludesPostHookScript(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should create post-hook script with heredoc
	assert.Contains(t, output, "cat > ~/.claude/hooks/capture-to-shy.sh << 'EOF'", "should create post-hook file")

	// Post-hook script should contain expected content
	assert.Contains(t, output, "# Claude Code Post-Tool Use Hook for shy", "should have post-hook header")
	assert.Contains(t, output, "INPUT=$(cat)", "should read JSON from stdin")
	assert.Contains(t, output, "COMMAND=$(echo \"$INPUT\" | jq -r '.tool_input.command')", "should extract command")
	assert.Contains(t, output, "EXIT_CODE=$(echo \"$INPUT\" | jq -r '.tool_response.exit_code // .tool_response.return_code // 0')", "should extract exit code")

	// Should use correct flag names (not the old ones)
	assert.Contains(t, output, "--status $EXIT_CODE", "should use --status flag")
	assert.Contains(t, output, "--dir", "should use --dir flag")
	assert.Contains(t, output, "$WORKING_DIR", "should use WORKING_DIR variable")
	assert.NotContains(t, output, "--exit-status", "should not use old --exit-status flag")
	assert.NotContains(t, output, "--working-dir", "should not use old --working-dir flag")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitIncludesShyInsertCommand tests the shy insert command structure
func TestClaudeInitIncludesShyInsertCommand(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Post-hook should build a shy insert command with all required fields
	assert.Contains(t, output, "shy insert", "should call shy insert")
	assert.Contains(t, output, "--command", "should include --command flag")
	assert.Contains(t, output, "ESCAPED_COMMAND=\"${COMMAND", "should escape COMMAND variable")
	assert.Contains(t, output, "$ESCAPED_COMMAND", "should use ESCAPED_COMMAND variable")
	assert.Contains(t, output, "--status $EXIT_CODE", "should include exit status")
	assert.Contains(t, output, "--dir", "should include --dir flag")
	assert.Contains(t, output, "$WORKING_DIR", "should use WORKING_DIR variable")
	assert.Contains(t, output, "--source-app claude-code", "should set source app to claude-code")
	assert.Contains(t, output, "--source-pid $SOURCE_PID", "should include source PID")

	// Should optionally include duration and git context
	assert.Contains(t, output, "--duration $DURATION", "should include duration when available")
	assert.Contains(t, output, "--git-repo", "should include git repo when available")
	assert.Contains(t, output, "$GIT_REPO", "should use GIT_REPO variable")
	assert.Contains(t, output, "--git-branch", "should include git branch when available")
	assert.Contains(t, output, "$GIT_BRANCH", "should use GIT_BRANCH variable")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitIncludesGitContextDetection tests git context detection
func TestClaudeInitIncludesGitContextDetection(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should detect git context
	assert.Contains(t, output, "if git rev-parse --git-dir", "should check if in git repo")
	assert.Contains(t, output, "git config --get remote.origin.url", "should get git repo URL")
	assert.Contains(t, output, "git rev-parse --abbrev-ref HEAD", "should get git branch")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitIncludesDurationCalculation tests duration calculation
func TestClaudeInitIncludesDurationCalculation(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should calculate duration from pre-hook timer
	assert.Contains(t, output, "CMD_ID=$(cat /tmp/shy-last-cmd-id", "should read command ID")
	assert.Contains(t, output, "/tmp/shy-timer-${CMD_ID}", "should check for timer file")
	assert.Contains(t, output, "START_TIME=$(cat \"/tmp/shy-timer-${CMD_ID}\")", "should read start time")
	assert.Contains(t, output, "DURATION=$((END_TIME - START_TIME))", "should calculate duration")

	// Should clean up temp files
	assert.Contains(t, output, "rm -f \"/tmp/shy-timer-${CMD_ID}\"", "should remove timer file")
	assert.Contains(t, output, "rm -f /tmp/shy-last-cmd-id", "should remove cmd ID file")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitIncludesSettingsScript tests the settings merge script
func TestClaudeInitIncludesSettingsScript(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should include the settings script content
	assert.Contains(t, output, "# 5. Merge hooks into Claude Code settings", "should have settings section")

	// Settings script should check for jq
	assert.Contains(t, output, "if ! command -v jq", "should check for jq")
	assert.Contains(t, output, "jq is required but not installed", "should warn about jq")

	// Should create settings file if not exists
	assert.Contains(t, output, "SETTINGS_FILE=~/.claude/settings.json", "should define settings file path")
	assert.Contains(t, output, "if [ ! -f \"$SETTINGS_FILE\" ]", "should check if settings exists")
	assert.Contains(t, output, "echo '{}' > \"$SETTINGS_FILE\"", "should create empty settings")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitConfiguresPreToolUseHook tests PreToolUse hook configuration
func TestClaudeInitConfiguresPreToolUseHook(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should add PreToolUse hook for Bash matcher
	assert.Contains(t, output, "PreToolUse", "should configure PreToolUse hook")
	assert.Contains(t, output, "\"matcher\": \"Bash\"", "should match Bash tool")
	assert.Contains(t, output, "~/.claude/hooks/shy-start-timer.sh", "should use pre-hook script")
	assert.Contains(t, output, "\"timeout\": 5", "should set timeout")

	// Should check if hook already exists
	assert.Contains(t, output, "PRE_HOOK_EXISTS=", "should check if pre-hook exists")
	assert.Contains(t, output, "Added PreToolUse hook", "should confirm addition")
	assert.Contains(t, output, "PreToolUse hook already exists, skipping", "should skip if exists")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitConfiguresPostToolUseHook tests PostToolUse hook configuration
func TestClaudeInitConfiguresPostToolUseHook(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should add PostToolUse hook for Bash matcher
	assert.Contains(t, output, "PostToolUse", "should configure PostToolUse hook")
	assert.Contains(t, output, "\"matcher\": \"Bash\"", "should match Bash tool")
	assert.Contains(t, output, "~/.claude/hooks/capture-to-shy.sh", "should use post-hook script")
	assert.Contains(t, output, "\"timeout\": 5", "should set timeout")

	// Should check if hook already exists
	assert.Contains(t, output, "POST_HOOK_EXISTS=", "should check if post-hook exists")
	assert.Contains(t, output, "Added PostToolUse hook", "should confirm addition")
	assert.Contains(t, output, "PostToolUse hook already exists, skipping", "should skip if exists")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitMakesScriptsExecutable tests chmod commands
func TestClaudeInitMakesScriptsExecutable(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should make both scripts executable
	assert.Contains(t, output, "chmod +x ~/.claude/hooks/shy-start-timer.sh", "should make pre-hook executable")
	assert.Contains(t, output, "chmod +x ~/.claude/hooks/capture-to-shy.sh", "should make post-hook executable")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitIncludesTestInstructions tests the test instructions
func TestClaudeInitIncludesTestInstructions(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should include test instructions
	assert.Contains(t, output, "# Test the integration:", "should have test section")
	assert.Contains(t, output, "shy fc -l -10", "should suggest checking history")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitSessionIDHandling tests session ID to PID conversion
func TestClaudeInitSessionIDHandling(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should extract session ID from JSON
	assert.Contains(t, output, "SESSION_ID=$(echo \"$INPUT\" | jq -r '.session_id // \"\"')", "should extract session ID")

	// Should convert session ID to numeric PID
	assert.Contains(t, output, "if [ -n \"$SESSION_ID\" ]", "should check if session ID exists")
	assert.Contains(t, output, "SOURCE_PID=$(echo \"$SESSION_ID\" | md5sum | tr -d 'a-z-' | cut -c1-5)", "should hash session ID")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitErrorHandling tests error handling in scripts
func TestClaudeInitErrorHandling(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Pre-hook should exit cleanly
	preHookSection := extractScriptSection(output, "shy-start-timer.sh")
	assert.Contains(t, preHookSection, "exit 0", "pre-hook should exit 0")

	// Post-hook should silence errors to avoid breaking Claude
	postHookSection := extractScriptSection(output, "capture-to-shy.sh")
	assert.Contains(t, postHookSection, "2>/dev/null", "post-hook should silence errors")
	assert.Contains(t, postHookSection, "exit 0", "post-hook should always exit 0")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitScriptSeparation tests that scripts are properly separated
func TestClaudeInitScriptSeparation(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"claude-init"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should have EOF markers for heredocs
	eofCount := strings.Count(output, "EOF")
	assert.GreaterOrEqual(t, eofCount, 4, "should have at least 4 EOF markers (2 per script)")

	// Each script should be in its own heredoc
	assert.Contains(t, output, "<< 'EOF'\n#!/bin/bash\n# Claude Code Pre-Tool Use Hook", "pre-hook should be in heredoc")
	assert.Contains(t, output, "<< 'EOF'\n#!/bin/bash\n# Claude Code Post-Tool Use Hook", "post-hook should be in heredoc")

	rootCmd.SetArgs(nil)
}

// TestClaudeInitCommandDescription tests command help text
func TestClaudeInitCommandDescription(t *testing.T) {
	// The command should be registered with rootCmd
	cmd, _, err := rootCmd.Find([]string{"claude-init"})
	require.NoError(t, err, "claude-init command should be registered")
	require.NotNil(t, cmd, "command should not be nil")

	// Check short description
	assert.Contains(t, cmd.Short, "Claude Code integration", "should mention Claude Code")

	// Check long description
	assert.Contains(t, cmd.Long, "Claude Code hook scripts", "should explain hooks")
	assert.Contains(t, cmd.Long, "capture", "should mention capturing commands")
}

// Helper function to extract a script section from the output
func extractScriptSection(output, scriptName string) string {
	startMarker := "cat > ~/.claude/hooks/" + scriptName + " << 'EOF'"
	startIdx := strings.Index(output, startMarker)
	if startIdx == -1 {
		return ""
	}

	// Find the closing EOF
	endMarker := "EOF"
	endIdx := strings.Index(output[startIdx+len(startMarker):], endMarker)
	if endIdx == -1 {
		return ""
	}

	return output[startIdx : startIdx+len(startMarker)+endIdx]
}
