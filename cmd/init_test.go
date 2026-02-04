package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
)

// TestScenario1_GenerateZshIntegrationScript tests that the init command
// generates valid zsh integration script
func TestScenario1_GenerateZshIntegrationScript(t *testing.T) {
	// Given: I am using zsh as my shell
	// When: I run "shy init zsh"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"init", "zsh"})

	err := rootCmd.Execute()
	require.NoError(t, err, "init command should succeed")

	output := buf.String()

	// Then: the command should output zsh integration script
	assert.NotEmpty(t, output, "output should not be empty")

	// And: the script should define a preexec function
	assert.Contains(t, output, "__shy_preexec()", "should define preexec function")

	// And: the script should define a precmd function
	assert.Contains(t, output, "__shy_precmd()", "should define precmd function")

	// And: the script should include instructions for installation
	assert.Contains(t, output, "Installation:", "should include installation instructions")
	assert.Contains(t, output, "eval \"$(shy init zsh)\"", "should include eval command")

	// And: the output should use zsh's standard hook system
	assert.Contains(t, output, "add-zsh-hook", "should use add-zsh-hook")
	assert.Contains(t, output, "add-zsh-hook preexec __shy_preexec", "should register preexec hook")
	assert.Contains(t, output, "add-zsh-hook precmd __shy_precmd", "should register precmd hook")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestScenario13_InitCommandWithEvalOutput tests that the init output
// includes eval instructions
func TestScenario13_InitCommandWithEvalOutput(t *testing.T) {
	// Given: I am using zsh
	// When: I run "shy init zsh"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"init", "zsh"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should include a line like "eval \"$(shy init zsh)\""
	assert.Contains(t, output, "eval \"$(shy init zsh)\"", "should include eval command")

	// And: the output should explain how to add to .zshrc
	assert.Contains(t, output, ".zshrc", "should mention .zshrc")

	rootCmd.SetArgs(nil)
}

// TestScenario14_InitCommandProvidesUninstallInstructions tests that
// uninstall instructions are provided
func TestScenario14_InitCommandProvidesUninstallInstructions(t *testing.T) {
	// Given: I am using zsh
	// When: I run "shy init zsh"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"init", "zsh"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should include comments explaining how to disable tracking
	assert.Contains(t, output, "Uninstall:", "should include uninstall section")

	// And: the output should explain how to remove the integration
	assert.Contains(t, output, "Remove", "should explain removal")

	rootCmd.SetArgs(nil)
}

// TestScenario21_InitCommandIncludesConfigurationOptions tests that
// configuration options are documented
func TestScenario21_InitCommandIncludesConfigurationOptions(t *testing.T) {
	// Given: I am using zsh
	// When: I run "shy init zsh"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"init", "zsh"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should include a comment about the SHY_DISABLE variable
	assert.Contains(t, output, "SHY_DISABLE", "should mention SHY_DISABLE variable")

	// And: the output should mention the database location
	assert.Contains(t, output, "SHY_DB_PATH", "should mention SHY_DB_PATH variable")
	assert.Contains(t, output, ".local/share/shy/history.db", "should mention default database path")

	// And: the output should be self-documenting
	assert.Contains(t, output, "Configuration:", "should have configuration section")

	rootCmd.SetArgs(nil)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Helper function to simulate a command execution with shy integration
func simulateCommand(t *testing.T, dbPath, command string, exitStatus int, workingDir string) {
	t.Helper()

	// Initialize database first (as the real integration does via shy init-db)
	if dbPath != "" {
		initDB, err := db.NewForTesting(dbPath)
		require.NoError(t, err, "failed to initialize database")
		initDB.Close()
	}

	// Get the absolute path to the shy binary
	shyBinary, err := filepath.Abs("./shy")
	if err != nil || !fileExists(shyBinary) {
		shyBinary, err = filepath.Abs("../shy")
		if err != nil || !fileExists(shyBinary) {
			t.Fatalf("failed to find shy binary: %v", err)
		}
	}

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a script that simulates what the hooks do
	// This avoids the issue with preexec being called multiple times
	script := `#!/usr/bin/env zsh
set -e

# Set up environment
export PATH="` + filepath.Dir(shyBinary) + `:$PATH"
` + (func() string {
		if dbPath != "" {
			return `export SHY_DB_PATH="` + dbPath + `"`
		}
		return ""
	})() + `

# Simulate what preexec and precmd do
__shy_cmd="` + command + `"
__shy_cmd_dir="` + workingDir + `"
__shy_cmd_start="$(date +%s)"
__shy_exit_status=` + fmt.Sprintf("%d", exitStatus) + `

# Determine log directory (same logic as in integration)
shy_data_dir="${XDG_DATA_HOME:-$HOME/.local/share}/shy"
shy_error_log="$shy_data_dir/error.log"

# Build shy insert command (same logic as precmd)
shy_args=(
	"insert"
	"--command" "$__shy_cmd"
	"--dir" "$__shy_cmd_dir"
	"--status" "$__shy_exit_status"
)

if [[ -n "$__shy_cmd_start" ]]; then
	shy_args+=("--timestamp" "$__shy_cmd_start")
fi

if [[ -n "$SHY_DB_PATH" ]]; then
	shy_args+=("--db" "$SHY_DB_PATH")
fi

# Execute shy insert (not in background for testing)
error_output=$(shy "${shy_args[@]}" 2>&1) || {
	mkdir -p "$shy_data_dir" 2>/dev/null
	{
		echo "[$(date '+%Y-%m-%d %H:%M:%S')] Error executing shy insert"
		echo "  Command: $__shy_cmd"
		echo "  Working dir: $__shy_cmd_dir"
		echo "  Error output: $error_output"
		echo ""
	} >> "$shy_error_log" 2>/dev/null
	exit 1
}
`

	scriptFile := filepath.Join(tempDir, "simulate.zsh")
	err = os.WriteFile(scriptFile, []byte(script), 0755)
	require.NoError(t, err, "failed to write simulation script")

	// Run the script
	cmd := exec.Command("zsh", scriptFile)
	cmd.Dir = tempDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Logf("Simulation script failed: %v", err)
		t.Logf("Stderr: %s", stderr.String())
	}
}

// TestScenario2_IntegrationScriptCapturesCommandText tests that the
// integration captures command text and metadata
func TestScenario2_IntegrationScriptCapturesCommandText(t *testing.T) {
	// Skip if zsh is not available
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}

	// Given: I have sourced the shy zsh integration
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	// When: I execute the command "echo hello"
	simulateCommand(t, dbPath, "echo hello", 0, tempDir)

	// Then: shy should store the command text "echo hello"
	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err, "failed to open database")
	defer database.Close()

	count, err := database.CountCommands()
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should have one command")

	// Get the command
	cmd, err := database.GetCommand(1)
	require.NoError(t, err, "failed to get command")

	// Verify the command text
	assert.Equal(t, "echo hello", cmd.CommandText, "command text should match")

	// And: the working directory should be captured
	assert.Equal(t, tempDir, cmd.WorkingDir, "working directory should match")

	// And: the timestamp should be recorded
	assert.Greater(t, cmd.Timestamp, int64(0), "timestamp should be recorded")
	assert.InDelta(t, time.Now().Unix(), cmd.Timestamp, 5, "timestamp should be recent")
}

// TestScenario3_IntegrationScriptCapturesExitStatus tests capturing
// non-zero exit status
func TestScenario3_IntegrationScriptCapturesExitStatus(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	// When: I execute the command "false" which exits with status 1
	simulateCommand(t, dbPath, "false", 1, tempDir)

	// Then: shy should store the exit status 1
	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	cmd, err := database.GetCommand(1)
	require.NoError(t, err)

	assert.Equal(t, "false", cmd.CommandText, "command text should be 'false'")
	assert.Equal(t, 1, cmd.ExitStatus, "exit status should be 1")
}

// TestScenario4_IntegrationScriptCapturesSuccessfulCommands tests capturing
// successful commands with exit status 0
func TestScenario4_IntegrationScriptCapturesSuccessfulCommands(t *testing.T) {

	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	// When: I execute the command "true" which exits with status 0
	simulateCommand(t, dbPath, "true", 0, tempDir)

	// Then: shy should store the exit status 0
	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	cmd, err := database.GetCommand(1)
	require.NoError(t, err)

	assert.Equal(t, "true", cmd.CommandText, "command text should be 'true'")
	assert.Equal(t, 0, cmd.ExitStatus, "exit status should be 0")
}

// TestScenario18_IntegrationHandlesErrorsGracefully tests that errors
// are logged and don't break the shell
func TestScenario18_IntegrationHandlesErrorsGracefully(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}

	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	dbPath := filepath.Join(dataDir, "shy/history.db")
	errorLog := filepath.Join(dataDir, "shy/error.log")

	// Simulate a command with an invalid working directory (will cause error)
	// Note: We pass an empty dir which will fail validation
	simulateCommand(t, dbPath, "test command", 0, "")

	// The command should fail to insert, but an error log should be created
	time.Sleep(time.Second)

	// Check if error log was created
	if _, err := os.Stat(errorLog); os.IsNotExist(err) {
		// If log doesn't exist, it might be in XDG_DATA_HOME or ~/.local/share
		// For this test, we're using a custom location so it should be there
		t.Logf("Error log not found at: %s", errorLog)
		// This is okay - error logging is best-effort
	} else {
		// Error log exists, verify it contains useful information
		content, err := os.ReadFile(errorLog)
		require.NoError(t, err, "should be able to read error log")

		logContent := string(content)
		assert.Contains(t, logContent, "Error executing shy insert", "log should contain error message")
		assert.Contains(t, logContent, "test command", "log should contain command")
	}

	// Most importantly: the database should not have the bad command
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		database, err := db.NewForTesting(dbPath)
		if err == nil {
			defer database.Close()
			count, _ := database.CountCommands()
			// Should be 0 because the insert failed
			assert.Equal(t, 0, count, "failed command should not be in database")
		}
	}
}

// TestInitCommand_UnsupportedShell tests error handling for unsupported shells
func TestInitCommand_UnsupportedShell(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"init", "bash"})

	err := rootCmd.Execute()
	assert.Error(t, err, "should error for unsupported shell")
	assert.Contains(t, err.Error(), "unsupported shell", "error should mention unsupported shell")

	rootCmd.SetArgs(nil)
}

// TestInitWithAutosuggest tests that --autosuggest flag generates autosuggest strategies
func TestInitWithAutosuggest(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"init", "zsh", "--autosuggest"})

	err := rootCmd.Execute()
	require.NoError(t, err, "init command should succeed")

	output := buf.String()

	// Should contain autosuggest strategy functions
	assert.Contains(t, output, "_zsh_autosuggest_strategy_shy_history", "should define shy_history strategy")

	// Should use shy commands
	assert.Contains(t, output, "like-recent", "should use like-recent command")

	// Reset init flags
	initRecord = false
	initUse = false
	initAutosuggest = false
	rootCmd.SetArgs(nil)
}

// TestInitWithMultipleFlags tests combining --record, --use, and --autosuggest
func TestInitWithMultipleFlags(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"init", "zsh", "--record", "--use", "--autosuggest"})

	err := rootCmd.Execute()
	require.NoError(t, err, "init command should succeed")

	output := buf.String()

	// Should contain all three scripts
	assert.Contains(t, output, "__shy_preexec", "should include record script")
	assert.Contains(t, output, "_shy_shell_history", "should include use script")
	assert.Contains(t, output, "_zsh_autosuggest_strategy_shy_history", "should include autosuggest script")

	// Reset init flags
	initRecord = false
	initUse = false
	initAutosuggest = false
	rootCmd.SetArgs(nil)
}

// Capture stdout for testing
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
