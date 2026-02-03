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

// TestScenario6_5_ListAllCommands tests that list-all shows all commands
// including duplicates
func TestScenario6_5_ListAllCommands(t *testing.T) {
	// Given: I have a database with 100 commands (10 unique command texts, each repeated 10 times)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
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
	// Given: I have commands with durations
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
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

	for _, tc := range testCases {
		cmd := &models.Command{
			CommandText: tc.command,
			WorkingDir:  tc.workingDir,
			ExitStatus:  0,
			Timestamp:   time.Now().Unix(),
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
	assert.Equal(t, "echo 5\necho 4\necho 3\n", output, "should have correct output")

	rootCmd.SetArgs(nil)
}
