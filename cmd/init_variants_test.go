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

	// Then: the output should define shy_ctrl_r for Ctrl-R binding
	assert.Contains(t, output, "shy_ctrl_r()", "should define shy_ctrl_r")
	assert.Contains(t, output, "bindkey '^R' shy_ctrl_r", "should bind Ctrl-R")

	// And: the output should define shy_up_arrow for up arrow binding
	assert.Contains(t, output, "shy_up_arrow()", "should define shy_up_arrow")
	assert.Contains(t, output, "bindkey '^[[A' shy_up_arrow", "should bind up arrow")

	// And: the output should define shy_right_arrow for right arrow completion
	assert.Contains(t, output, "shy_right_arrow()", "should define shy_right_arrow")
	assert.Contains(t, output, "bindkey '^[[C' shy_right_arrow", "should bind right arrow")

	// And: the output should NOT include __shy_preexec
	assert.NotContains(t, output, "__shy_preexec()", "should not include __shy_preexec")

	// And: the output should NOT include __shy_precmd
	assert.NotContains(t, output, "__shy_precmd()", "should not include __shy_precmd")

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

	// Then: the output should define __shy_preexec
	assert.Contains(t, output, "__shy_preexec()", "should define __shy_preexec")

	// And: the output should define __shy_precmd
	assert.Contains(t, output, "__shy_precmd()", "should define __shy_precmd")

	// And: the output should define shy_ctrl_r
	assert.Contains(t, output, "shy_ctrl_r()", "should define shy_ctrl_r")

	// And: the output should define shy_up_arrow
	assert.Contains(t, output, "shy_up_arrow()", "should define shy_up_arrow")

	// And: the output should define shy_right_arrow
	assert.Contains(t, output, "shy_right_arrow()", "should define shy_right_arrow")

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
