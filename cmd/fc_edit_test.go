package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

// commandCapture is a helper for capturing executed commands in tests
type commandCapture struct {
	commands []string
	mu       sync.Mutex
}

// setupMockEditor injects a function that modifies the file
// Returns a cleanup function that should be deferred
func setupMockEditor(t *testing.T, modifyFunc func(content string) string) func() {
	oldEditor := invokeEditorFunc
	invokeEditorFunc = func(editorPath, filepath string) error {
		// Read file
		content, err := os.ReadFile(filepath)
		if err != nil {
			return err
		}

		// Apply modification
		modified := modifyFunc(string(content))

		// Write back
		return os.WriteFile(filepath, []byte(modified), 0600)
	}

	// Return cleanup function
	return func() {
		invokeEditorFunc = oldEditor
	}
}

// setupCommandCapture injects a function that captures commands instead of executing them
// Returns the capture struct and a cleanup function that should be deferred
func setupCommandCapture(t *testing.T) (*commandCapture, func()) {
	capture := &commandCapture{
		commands: []string{},
	}
	oldExec := executeShellFunc

	executeShellFunc = func(shell, cmdText string) error {
		capture.mu.Lock()
		defer capture.mu.Unlock()
		capture.commands = append(capture.commands, cmdText)
		return nil
	}

	return capture, func() {
		executeShellFunc = oldExec
	}
}

// Test scenarios for edit-and-execute mode

// TestScenario_EditAndExecuteSingleCommand tests basic edit and execute
func TestScenario_EditAndExecuteSingleCommand(t *testing.T) {
	defer resetFcFlags(fcCmd)

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert test command
	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567891,
		CommandText: "git status",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	// Setup mock editor that changes "status" to "log"
	cleanupEditor := setupMockEditor(t, func(content string) string {
		return strings.Replace(content, "status", "log", 1)
	})
	defer cleanupEditor()

	// Setup command capture
	capture, cleanupExec := setupCommandCapture(t)
	defer cleanupExec()

	// Run fc with edit mode (no -l flag, so edit mode is triggered)
	rootCmd.SetArgs([]string{"fc", "--db", dbPath, "1"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify command was executed
	assert.Equal(t, []string{"git log"}, capture.commands)

	// Verify command was added to history
	commands, err := database.GetCommandsByRange(1, 100)
	require.NoError(t, err)
	assert.Equal(t, 2, len(commands)) // Original + executed
	assert.Equal(t, "git log", commands[1].CommandText)

	rootCmd.SetArgs(nil)
}

// TestScenario_QuickExecuteWithoutEditing tests -s flag
func TestScenario_QuickExecuteWithoutEditing(t *testing.T) {
	defer resetFcFlags(fcCmd)

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert test command
	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567891,
		CommandText: "git status",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	// Setup command capture (no editor needed for -s)
	capture, cleanupExec := setupCommandCapture(t)
	defer cleanupExec()

	// Run fc with -s flag
	rootCmd.SetArgs([]string{"fc", "-s", "--db", dbPath, "1"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify command was executed without editing
	assert.Equal(t, []string{"git status"}, capture.commands)

	rootCmd.SetArgs(nil)
}

// TestScenario_QuickExecuteWithMultipleSubstitutions tests -s with old=new
func TestScenario_QuickExecuteWithMultipleSubstitutions(t *testing.T) {
	defer resetFcFlags(fcCmd)

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert test command
	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567892,
		CommandText: "npm install lodash",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	// Setup command capture
	capture, cleanupExec := setupCommandCapture(t)
	defer cleanupExec()

	// Run fc with -s and substitutions
	rootCmd.SetArgs([]string{"fc", "-s", "npm=yarn", "install=add", "--db", dbPath, "1"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify substitutions were applied
	assert.Equal(t, []string{"yarn add lodash"}, capture.commands)

	rootCmd.SetArgs(nil)
}

// TestScenario_CannotCombineSWithE tests flag conflict validation
func TestScenario_CannotCombineSWithE(t *testing.T) {
	defer resetFcFlags(fcCmd)

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Run fc with both -s and -e
	rootCmd.SetArgs([]string{"fc", "-s", "-e", "nano", "--db", dbPath, "1"})
	err = rootCmd.Execute()

	// Should error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use -s and -e together")

	rootCmd.SetArgs(nil)
}

// TestScenario_InvalidRange tests backwards range error
func TestScenario_InvalidRange(t *testing.T) {
	defer resetFcFlags(fcCmd)

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert test commands
	for i := 100; i <= 105; i++ {
		_, err = database.InsertCommand(&models.Command{
			Timestamp:   int64(1234567890 + i),
			CommandText: "test command",
			WorkingDir:  "/tmp",
			ExitStatus:  0,
		})
		require.NoError(t, err)
	}

	// Setup command capture
	capture, cleanupExec := setupCommandCapture(t)
	defer cleanupExec()

	// Run fc with backwards range (5 to 1)
	rootCmd.SetArgs([]string{"fc", "-s", "--db", dbPath, "5", "1"})
	err = rootCmd.Execute()

	// Should error with specific message
	require.Error(t, err)
	assert.Contains(t, err.Error(), "history events can't be executed backwards")

	// No commands should have been executed
	assert.Equal(t, 0, len(capture.commands))

	rootCmd.SetArgs(nil)
}

// TestScenario_EditWithSingleSubstitution tests substitution before editing
func TestScenario_EditWithSingleSubstitution(t *testing.T) {
	defer resetFcFlags(fcCmd)

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert test command
	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567890,
		CommandText: "echo \"hello world\"",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	// Setup mock editor that doesn't change anything (just verify substitution was applied)
	var editorContent string
	cleanupEditor := setupMockEditor(t, func(content string) string {
		editorContent = content
		return content
	})
	defer cleanupEditor()

	// Setup command capture
	capture, cleanupExec := setupCommandCapture(t)
	defer cleanupExec()

	// Run fc with substitution
	rootCmd.SetArgs([]string{"fc", "echo=printf", "--db", dbPath, "1"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify substitution was applied before editing
	assert.Contains(t, editorContent, "printf \"hello world\"")
	assert.NotContains(t, editorContent, "echo \"hello world\"")

	// Verify command was executed with substitution
	assert.Equal(t, []string{"printf \"hello world\""}, capture.commands)

	rootCmd.SetArgs(nil)
}

// TestScenario_TempFileSecurePermissions tests file permissions
func TestScenario_TempFileSecurePermissions(t *testing.T) {
	// Test the createTempFileWithCommands function directly
	commands := []string{"echo hello", "echo world"}

	tmpfile, err := createTempFileWithCommands(commands)
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	// Check file permissions
	info, err := os.Stat(tmpfile.Name())
	require.NoError(t, err)

	// Mode should be 0600 (user read/write only)
	mode := info.Mode()
	assert.Equal(t, os.FileMode(0600), mode.Perm())
}

// TestScenario_NoArgsEditsLastCommand tests that "fc" with no args edits the last command
func TestScenario_NoArgsEditsLastCommand(t *testing.T) {
	defer resetFcFlags(fcCmd)

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert multiple test commands
	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567890,
		CommandText: "git status",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567891,
		CommandText: "ls -la",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567892,
		CommandText: "echo hello",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	// Setup mock editor to verify only the last command is edited
	var editorContent string
	cleanupEditor := setupMockEditor(t, func(content string) string {
		editorContent = content
		// Change "hello" to "goodbye"
		return strings.Replace(content, "hello", "goodbye", 1)
	})
	defer cleanupEditor()

	// Setup command capture
	capture, cleanupExec := setupCommandCapture(t)
	defer cleanupExec()

	// Run fc with no arguments
	rootCmd.SetArgs([]string{"fc", "--db", dbPath})
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify only the last command was in the editor
	assert.Equal(t, "echo hello\n", editorContent)

	// Verify the modified command was executed
	assert.Equal(t, []string{"echo goodbye"}, capture.commands)

	rootCmd.SetArgs(nil)
}

// TestScenario_EditSingleEventByID tests that "fc 2" edits just event 2
func TestScenario_EditSingleEventByID(t *testing.T) {
	defer resetFcFlags(fcCmd)

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert multiple test commands
	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567890,
		CommandText: "git status",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567891,
		CommandText: "ls -la",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567892,
		CommandText: "echo hello",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	// Setup mock editor to verify only event 2 is edited
	var editorContent string
	cleanupEditor := setupMockEditor(t, func(content string) string {
		editorContent = content
		// Change "ls" to "dir"
		return strings.Replace(content, "ls", "dir", 1)
	})
	defer cleanupEditor()

	// Setup command capture
	capture, cleanupExec := setupCommandCapture(t)
	defer cleanupExec()

	// Run fc with event ID 2
	rootCmd.SetArgs([]string{"fc", "--db", dbPath, "2"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify only command 2 was in the editor (ls -la)
	assert.Equal(t, "ls -la\n", editorContent)

	// Verify the modified command was executed
	assert.Equal(t, []string{"dir -la"}, capture.commands)

	rootCmd.SetArgs(nil)
}

// TestScenario_EditSingleEventByString tests that "fc git" edits just the matched event
func TestScenario_EditSingleEventByString(t *testing.T) {
	defer resetFcFlags(fcCmd)

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert multiple test commands
	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567890,
		CommandText: "git status",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567891,
		CommandText: "ls -la",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567892,
		CommandText: "git commit -m 'test'",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	// Setup mock editor to verify only the most recent "git" command is edited
	var editorContent string
	cleanupEditor := setupMockEditor(t, func(content string) string {
		editorContent = content
		// Change "commit" to "push"
		return strings.Replace(content, "commit", "push", 1)
	})
	defer cleanupEditor()

	// Setup command capture
	capture, cleanupExec := setupCommandCapture(t)
	defer cleanupExec()

	// Run fc with string match "git"
	rootCmd.SetArgs([]string{"fc", "--db", dbPath, "git"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify only the most recent git command was in the editor
	assert.Equal(t, "git commit -m 'test'\n", editorContent)

	// Verify the modified command was executed
	assert.Equal(t, []string{"git push -m 'test'"}, capture.commands)

	rootCmd.SetArgs(nil)
}

// TestScenario_EditorExitsWithError tests that fc exits silently with editor's exit code
func TestScenario_EditorExitsWithError(t *testing.T) {
	defer resetFcFlags(fcCmd)

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert test command
	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567891,
		CommandText: "git status",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	// Override osExit to capture the exit code
	exitCode := -1
	oldOsExit := osExit
	osExit = func(code int) {
		exitCode = code
	}
	defer func() { osExit = oldOsExit }()

	// Override the editor function to return an ExitError with status 1
	oldEditor := invokeEditorFunc
	invokeEditorFunc = func(editorPath, filepath string) error {
		// Simulate editor exiting with status 1 (e.g., vim :cq)
		// Use a command that always exits with 1
		cmd := exec.Command("false") // 'false' command always exits with 1
		err := cmd.Run()             // This will be an *exec.ExitError
		return err
	}
	defer func() {
		invokeEditorFunc = oldEditor
	}()

	// Setup command capture
	capture, cleanupExec := setupCommandCapture(t)
	defer cleanupExec()

	// Run fc - error may be nil because osExit is called instead
	rootCmd.SetArgs([]string{"fc", "--db", dbPath, "1"})
	_ = rootCmd.Execute()

	// Verify fc exited with code 1 (same as editor)
	assert.Equal(t, 1, exitCode, "fc should exit with same code as editor")

	// No commands should have been executed
	assert.Equal(t, 0, len(capture.commands))

	rootCmd.SetArgs(nil)
}

// TestScenario_EditSingleEventByNegativeNumber tests that "fc -2" edits just the 2nd-to-last command
func TestScenario_EditSingleEventByNegativeNumber(t *testing.T) {
	defer resetFcFlags(fcCmd)

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert multiple test commands
	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567890,
		CommandText: "git status",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567891,
		CommandText: "ls -la",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567892,
		CommandText: "echo hello",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	// Setup mock editor to verify only the 2nd-to-last command is edited
	var editorContent string
	cleanupEditor := setupMockEditor(t, func(content string) string {
		editorContent = content
		// Change "ls" to "dir"
		return strings.Replace(content, "ls", "dir", 1)
	})
	defer cleanupEditor()

	// Setup command capture
	capture, cleanupExec := setupCommandCapture(t)
	defer cleanupExec()

	// Run fc with -2 (2nd-to-last command)
	rootCmd.SetArgs([]string{"fc", "--db", dbPath, "-2"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify only the 2nd-to-last command was in the editor (ls -la, which is event 2)
	assert.Equal(t, "ls -la\n", editorContent)

	// Verify the modified command was executed
	assert.Equal(t, []string{"dir -la"}, capture.commands)

	rootCmd.SetArgs(nil)
}

// TestScenario_EditRangeWithNegativeNumbers tests that "fc -2 -1" edits last 2 commands
func TestScenario_EditRangeWithNegativeNumbers(t *testing.T) {
	defer resetFcFlags(fcCmd)

	// Setup database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert multiple test commands
	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567890,
		CommandText: "git status",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567891,
		CommandText: "ls -la",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	_, err = database.InsertCommand(&models.Command{
		Timestamp:   1234567892,
		CommandText: "echo hello",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
	})
	require.NoError(t, err)

	// Setup mock editor to verify both commands are edited
	var editorContent string
	cleanupEditor := setupMockEditor(t, func(content string) string {
		editorContent = content
		// Just return as-is
		return content
	})
	defer cleanupEditor()

	// Setup command capture
	capture, cleanupExec := setupCommandCapture(t)
	defer cleanupExec()

	// Run fc with -2 -1 (last 2 commands: events 2 and 3)
	rootCmd.SetArgs([]string{"fc", "--db", dbPath, "-2", "-1"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify both commands were in the editor
	assert.Contains(t, editorContent, "ls -la")
	assert.Contains(t, editorContent, "echo hello")

	// Verify both commands were executed
	assert.Equal(t, 2, len(capture.commands))
	assert.Equal(t, "ls -la", capture.commands[0])
	assert.Equal(t, "echo hello", capture.commands[1])

	rootCmd.SetArgs(nil)
}
