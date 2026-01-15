package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionCommand tests the version command
func TestVersionCommand(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	require.NoError(t, err, "version command should succeed")

	output := buf.String()
	assert.Contains(t, output, "shy version", "should contain 'shy version'")
	assert.Contains(t, output, ShyVersion, "should contain version number")

	rootCmd.SetArgs(nil)
}

// TestVersionFlag tests the --version flag
func TestVersionFlag(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--version"})

	err := rootCmd.Execute()
	require.NoError(t, err, "version flag should succeed")

	output := buf.String()
	assert.Contains(t, output, "shy version", "should contain 'shy version'")
	assert.Contains(t, output, ShyVersion, "should contain version number")

	rootCmd.SetArgs(nil)
}

// TestVersionShortFlag tests the -v flag
func TestVersionShortFlag(t *testing.T) {
	flag := rootCmd.Flags().Lookup("version")
	require.NotNil(t, flag, "-v flag should be registered")
	assert.Equal(t, "v", flag.Shorthand, "shorthand should be -v")
}
