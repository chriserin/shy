package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

func resetListAllFlags() {
	listAllFormat = ""
	listAllSession = ""
	listAllCurrentSession = false
	listAllPwd = false
}

// TestScenario6_5_ListAllCommands tests that list-all shows all commands
// including duplicates
func TestScenario6_5_ListAllCommands(t *testing.T) {
	defer resetListAllFlags()
	// Given: I have a database with 100 commands (10 unique command texts, each repeated 10 times)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Insert 100 commands with 10 unique command texts
	for i := range 100 {
		cmd := &models.Command{
			CommandText: "echo test" + string(rune('0'+(i%10))),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy list-all"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list-all", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "list-all command should succeed")

	output := buf.String()

	// Then: the output should contain all 100 commands including duplicates
	// Count the number of lines (each command is one line)
	lines := bytes.Split([]byte(output), []byte("\n"))
	// Subtract 1 for the trailing newline
	commandCount := len(lines) - 1
	assert.Equal(t, 100, commandCount, "should display all 100 commands including duplicates")

	// Verify all unique command texts are present
	for i := 0; i < 10; i++ {
		expectedCmd := "echo test" + string(rune('0'+i))
		assert.Contains(t, output, expectedCmd, "should contain command: "+expectedCmd)
	}

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

func TestListAllWithCwd(t *testing.T) {
	defer resetListAllFlags()
	// Given: I have commands with durations
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	proj1 := filepath.Join(tempDir, "proj1")
	proj2 := filepath.Join(tempDir, "proj2")
	os.MkdirAll(proj1, 0755)
	os.MkdirAll(proj2, 0755)

	// Test cases from the spec
	testCases := []struct {
		command    string
		workingDir string
	}{
		{"echo 1", proj1},
		{"echo 2", proj1},
		{"echo 3", proj2},
		{"echo 4", proj2},
		{"echo 5", proj2},
		{"echo 6", proj1},
	}

	for i, tc := range testCases {
		cmd := &models.Command{
			CommandText: tc.command,
			WorkingDir:  tc.workingDir,
			ExitStatus:  0,
			Timestamp:   time.Now().Unix() + int64(i),
			Duration:    int64Ptr(1),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// Change to proj1 directory
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)

	err = os.Chdir(proj2)
	require.NoError(t, err, "failed to change directory")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list-all", "--pwd", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	require.Equal(t, 3, len(lines), "should have correct number of lines")
	assert.Equal(t, "echo 3\necho 4\necho 5\n", output, "should have correct output")

	rootCmd.SetArgs(nil)
}

// TestListAllWithSessionAppOnly tests filtering by app name only (without pid)
func TestListAllWithSessionAppOnly(t *testing.T) {
	defer resetListAllFlags()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert commands from zsh sessions (different pids)
	zshApp := "zsh"
	zshPid1 := int64(12345)
	zshPid2 := int64(67890)
	active := true

	zshCommands := []struct {
		text string
		pid  int64
		ts   int64
	}{
		{"zsh cmd 1", zshPid1, 1704470400},
		{"zsh cmd 2", zshPid1, 1704470401},
		{"zsh cmd 3", zshPid2, 1704470402},
	}

	for _, c := range zshCommands {
		cmd := &models.Command{
			CommandText:  c.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    c.ts,
			SourceApp:    &zshApp,
			SourcePid:    &c.pid,
			SourceActive: &active,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// Insert commands from bash session
	bashApp := "bash"
	bashPid := int64(11111)

	bashCommands := []struct {
		text string
		ts   int64
	}{
		{"bash cmd 1", 1704470403},
		{"bash cmd 2", 1704470404},
	}

	for _, c := range bashCommands {
		cmd := &models.Command{
			CommandText:  c.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    c.ts,
			SourceApp:    &bashApp,
			SourcePid:    &bashPid,
			SourceActive: &active,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy list-all --session zsh" (app only, no pid)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list-all", "--session", "zsh", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: should contain all zsh commands from both pids
	assert.Contains(t, output, "zsh cmd 1")
	assert.Contains(t, output, "zsh cmd 2")
	assert.Contains(t, output, "zsh cmd 3")

	// And: should NOT contain bash commands
	assert.NotContains(t, output, "bash cmd")

	// Count lines (should be 3 zsh commands)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 3, len(lines), "should have exactly 3 zsh commands")

	rootCmd.SetArgs(nil)
}

// TestListAllWithSessionAppAndPid tests filtering by both app and pid
func TestListAllWithSessionAppAndPid(t *testing.T) {
	defer resetListAllFlags()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert commands from two different zsh pids
	zshApp := "zsh"
	zshPid1 := int64(12345)
	zshPid2 := int64(67890)
	active := true

	commands := []struct {
		text string
		pid  int64
		ts   int64
	}{
		{"pid1 cmd 1", zshPid1, 1704470400},
		{"pid1 cmd 2", zshPid1, 1704470401},
		{"pid2 cmd 1", zshPid2, 1704470402},
		{"pid2 cmd 2", zshPid2, 1704470403},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText:  c.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    c.ts,
			SourceApp:    &zshApp,
			SourcePid:    &c.pid,
			SourceActive: &active,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy list-all --session zsh:12345" (specific pid)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list-all", "--session", "zsh:12345", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: should contain only pid1 commands
	assert.Contains(t, output, "pid1 cmd 1")
	assert.Contains(t, output, "pid1 cmd 2")

	// And: should NOT contain pid2 commands
	assert.NotContains(t, output, "pid2 cmd")

	// Count lines (should be 2 commands)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 2, len(lines), "should have exactly 2 commands from pid 12345")

	rootCmd.SetArgs(nil)
}
