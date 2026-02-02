package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScenario8_InitWithRecordFlag tests that --record only enables recording
func TestScenario8_InitWithRecordFlag(t *testing.T) {
	// Given: I am using zsh as my shell
	// When: I run "shy init zsh --record"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"init", "zsh", "--record"})

	err := rootCmd.Execute()
	require.NoError(t, err, "init command should succeed")

	output := buf.String()

	// Then: the output should define __shy_preexec
	assert.Contains(t, output, "__shy_preexec()", "should define __shy_preexec")

	// And: the output should define __shy_precmd
	assert.Contains(t, output, "__shy_precmd()", "should define __shy_precmd")

	// And: the output should NOT include history lookup functions
	assert.NotContains(t, output, "shy_ctrl_r", "should not include shy_ctrl_r")
	assert.NotContains(t, output, "shy_up_arrow", "should not include shy_up_arrow")
	assert.NotContains(t, output, "shy_right_arrow", "should not include shy_right_arrow")

	// Reset command and flags for next test
	rootCmd.SetArgs(nil)
	initCmd.Flags().Set("record", "false")
	initCmd.Flags().Set("use", "false")
}

// TestScenario9_InitWithUseFlag tests that --use only enables history usage
func TestScenario9_InitWithUseFlag(t *testing.T) {
	// Given: I am using zsh as my shell
	// When: I run "shy init zsh --use"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"init", "zsh", "--use"})

	err := rootCmd.Execute()
	require.NoError(t, err, "init command should succeed")

	output := buf.String()

	// Then: the output should include Ctrl-R binding for history search
	assert.Contains(t, output, "bindkey '^R'", "should bind Ctrl-R")
	assert.Contains(t, output, "shy fzf", "should use shy fzf for history")

	// And: the output should include up arrow binding
	assert.Contains(t, output, "bindkey '^[[A'", "should bind up arrow (standard)")
	assert.Contains(t, output, "bindkey '^[OA'", "should bind up arrow (application mode)")
	assert.Contains(t, output, "shy last-command", "should use shy last-command")

	// And: the output should include down arrow binding
	assert.Contains(t, output, "bindkey '^[[B'", "should bind down arrow (standard)")
	assert.Contains(t, output, "bindkey '^[OB'", "should bind down arrow (application mode)")

	// And: the output should NOT include recording hooks
	assert.NotContains(t, output, "__shy_preexec", "should not include __shy_preexec")
	assert.NotContains(t, output, "__shy_precmd", "should not include __shy_precmd")
	assert.NotContains(t, output, "add-zsh-hook preexec", "should not add preexec hook")
	assert.NotContains(t, output, "add-zsh-hook precmd", "should not add precmd hook")

	// Reset command and flags for next test
	rootCmd.SetArgs(nil)
	initCmd.Flags().Set("record", "false")
	initCmd.Flags().Set("use", "false")
}

// TestScenario10_InitWithBothFlags tests that both flags enable everything
func TestScenario10_InitWithBothFlags(t *testing.T) {
	// Given: I am using zsh as my shell
	// When: I run "shy init zsh --record --use"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"init", "zsh", "--record", "--use"})

	err := rootCmd.Execute()
	require.NoError(t, err, "init command should succeed")

	output := buf.String()

	// Then: the output should include recording hooks
	assert.Contains(t, output, "__shy_preexec", "should include __shy_preexec")
	assert.Contains(t, output, "__shy_precmd", "should include __shy_precmd")
	assert.Contains(t, output, "add-zsh-hook preexec", "should add preexec hook")
	assert.Contains(t, output, "add-zsh-hook precmd", "should add precmd hook")

	// And: the output should include history usage bindings
	assert.Contains(t, output, "bindkey '^R'", "should bind Ctrl-R")
	assert.Contains(t, output, "bindkey '^[[A'", "should bind up arrow")

	// And: both shy commands should be present
	assert.Contains(t, output, "shy insert", "should use shy insert for recording")
	assert.Contains(t, output, "shy fzf", "should use shy fzf for history")
	assert.Contains(t, output, "shy last-command", "should use shy last-command")

	// Reset command and flags for next test
	rootCmd.SetArgs(nil)
	initCmd.Flags().Set("record", "false")
	initCmd.Flags().Set("use", "false")
}

// TestScenario11_InitWithoutFlagsDefaultsToRecord tests default behavior
func TestScenario11_InitWithoutFlagsDefaultsToRecord(t *testing.T) {
	// Given: I am using zsh as my shell
	// When: I run "shy init zsh"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"init", "zsh"})

	err := rootCmd.Execute()
	require.NoError(t, err, "init command should succeed")

	output := buf.String()

	// Then: the output should include all three scripts by default
	// Recording script
	assert.Contains(t, output, "__shy_preexec()", "should define __shy_preexec")
	assert.Contains(t, output, "__shy_precmd()", "should define __shy_precmd")

	// Usage script
	assert.Contains(t, output, "_shy_shell_history", "should include history lookup function")
	assert.Contains(t, output, "_shy_up_line_or_history", "should include up arrow handler")
	assert.Contains(t, output, "_shy_down_line_or_history", "should include down arrow handler")

	// Autosuggest script
	assert.Contains(t, output, "_zsh_autosuggest_strategy_shy_history", "should include shy_history strategy")

	// Reset command and flags for next test
	rootCmd.SetArgs(nil)
	initCmd.Flags().Set("record", "false")
	initCmd.Flags().Set("use", "false")
	initCmd.Flags().Set("autosuggest", "false")
	initCmd.Flags().Set("use", "false")
}
