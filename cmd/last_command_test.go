package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

// TestScenario1_GetLastCommand tests that last-command returns the most recent command
func TestScenario1_GetLastCommand(t *testing.T) {
	// Given: I have a database with commands
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"git status", 1704470400},
		{"ls -la", 1704470401},
		{"npm test", 1704470402},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy last-command"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"last-command", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "last-command should succeed")

	output := buf.String()

	// Then: the output should be "npm test"
	assert.Equal(t, "npm test\n", output, "should output the most recent command")

	// And: only the command text should be output (no formatting)
	// Verify there's no extra formatting (just the command and newline)
	assert.NotContains(t, output, "timestamp", "should not contain formatting")
	assert.NotContains(t, output, "status", "should not contain formatting")
	assert.NotContains(t, output, "/home/test", "should not contain working directory")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestScenario2_LastCommandWithNoHistory tests empty database handling
func TestScenario2_LastCommandWithNoHistory(t *testing.T) {
	// Given: I have an empty database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// When: I run "shy last-command"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"last-command", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "last-command should succeed even with empty database")

	output := buf.String()

	// Then: the output should be empty
	assert.Equal(t, "", output, "output should be empty for empty database")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestScenario3_LastCommandWithOffset tests the -n flag for cycling through history
func TestScenario3_LastCommandWithOffset(t *testing.T) {
	// Given: I have a database with commands
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"cmd1", 1704470400},
		{"cmd2", 1704470401},
		{"cmd3", 1704470402},
		{"cmd4", 1704470403},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy last-command -n 1" (most recent)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"last-command", "-n", "1", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "last-command should succeed")
	assert.Equal(t, "cmd4\n", buf.String(), "n=1 should return most recent")

	// When: I run "shy last-command -n 2" (second most recent)
	buf.Reset()
	rootCmd.SetArgs([]string{"last-command", "-n", "2", "--db", dbPath})
	err = rootCmd.Execute()
	require.NoError(t, err, "last-command should succeed")
	assert.Equal(t, "cmd3\n", buf.String(), "n=2 should return second most recent")

	// When: I run "shy last-command -n 3" (third most recent)
	buf.Reset()
	rootCmd.SetArgs([]string{"last-command", "-n", "3", "--db", dbPath})
	err = rootCmd.Execute()
	require.NoError(t, err, "last-command should succeed")
	assert.Equal(t, "cmd2\n", buf.String(), "n=3 should return third most recent")

	// When: I run "shy last-command -n 4" (fourth most recent)
	buf.Reset()
	rootCmd.SetArgs([]string{"last-command", "-n", "4", "--db", dbPath})
	err = rootCmd.Execute()
	require.NoError(t, err, "last-command should succeed")
	assert.Equal(t, "cmd1\n", buf.String(), "n=4 should return fourth most recent")

	// When: I run "shy last-command -n 5" (beyond available history)
	buf.Reset()
	rootCmd.SetArgs([]string{"last-command", "-n", "5", "--db", dbPath})
	err = rootCmd.Execute()
	require.NoError(t, err, "last-command should succeed")
	assert.Equal(t, "", buf.String(), "n beyond history should return empty")

	// Reset command for next test
	rootCmd.SetArgs(nil)
	lastCommandCmd.Flags().Set("offset", "1")
}

// TestScenario4_LastCommandWithSessionFilter tests the --session flag for session filtering
func TestScenario4_LastCommandWithSessionFilter(t *testing.T) {
	// Given: I have a database with commands from different sessions
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Insert commands from session 1 (zsh:12345)
	sourceApp1 := "zsh"
	sourcePid1 := int64(12345)
	sourceActive1 := true

	cmd1 := &models.Command{
		CommandText:  "session1 cmd1",
		WorkingDir:   "/home/test",
		ExitStatus:   0,
		Timestamp:    1704470400,
		SourceApp:    &sourceApp1,
		SourcePid:    &sourcePid1,
		SourceActive: &sourceActive1,
	}
	_, err = database.InsertCommand(cmd1)
	require.NoError(t, err, "failed to insert command")

	cmd2 := &models.Command{
		CommandText:  "session1 cmd2",
		WorkingDir:   "/home/test",
		ExitStatus:   0,
		Timestamp:    1704470401,
		SourceApp:    &sourceApp1,
		SourcePid:    &sourcePid1,
		SourceActive: &sourceActive1,
	}
	_, err = database.InsertCommand(cmd2)
	require.NoError(t, err, "failed to insert command")

	// Insert commands from session 2 (bash:67890)
	sourceApp2 := "bash"
	sourcePid2 := int64(67890)
	sourceActive2 := true

	cmd3 := &models.Command{
		CommandText:  "session2 cmd1",
		WorkingDir:   "/home/test",
		ExitStatus:   0,
		Timestamp:    1704470402,
		SourceApp:    &sourceApp2,
		SourcePid:    &sourcePid2,
		SourceActive: &sourceActive2,
	}
	_, err = database.InsertCommand(cmd3)
	require.NoError(t, err, "failed to insert command")

	cmd4 := &models.Command{
		CommandText:  "session2 cmd2",
		WorkingDir:   "/home/test",
		ExitStatus:   0,
		Timestamp:    1704470403,
		SourceApp:    &sourceApp2,
		SourcePid:    &sourcePid2,
		SourceActive: &sourceActive2,
	}
	_, err = database.InsertCommand(cmd4)
	require.NoError(t, err, "failed to insert command")

	// When: I run "shy last-command --session zsh:12345"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"last-command", "--session", "zsh:12345", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "last-command should succeed")

	output := buf.String()

	// Then: the output should be the most recent command from session 1
	assert.Equal(t, "session1 cmd2\n", output, "should return most recent command from zsh:12345 session")

	// When: I run "shy last-command --session bash:67890"
	buf.Reset()
	rootCmd.SetArgs([]string{"last-command", "--session", "bash:67890", "--db", dbPath})
	err = rootCmd.Execute()
	require.NoError(t, err, "last-command should succeed")

	output = buf.String()

	// Then: the output should be the most recent command from session 2
	assert.Equal(t, "session2 cmd2\n", output, "should return most recent command from bash:67890 session")

	// When: I run "shy last-command --session zsh:12345 -n 2"
	buf.Reset()
	rootCmd.SetArgs([]string{"last-command", "--session", "zsh:12345", "-n", "2", "--db", dbPath})
	err = rootCmd.Execute()
	require.NoError(t, err, "last-command should succeed")

	output = buf.String()

	// Then: the output should be the second most recent command from session 1
	assert.Equal(t, "session1 cmd1\n", output, "should return second most recent command from zsh:12345 session")

	// Reset command for next test
	rootCmd.SetArgs(nil)
	lastCommandCmd.Flags().Set("offset", "1")
	lastCommandCmd.Flags().Set("session", "")
}

// TestScenario5_LastCommandWithSessionAutoDetect tests the --session flag with auto-detection
func TestScenario5_LastCommandWithSessionAutoDetect(t *testing.T) {
	// Given: I have a database with commands from different sessions
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Insert commands from session 1 (zsh:99999)
	sourceApp1 := "zsh"
	sourcePid1 := int64(99999)
	sourceActive1 := true

	cmd1 := &models.Command{
		CommandText:  "auto session cmd1",
		WorkingDir:   "/home/test",
		ExitStatus:   0,
		Timestamp:    1704470400,
		SourceApp:    &sourceApp1,
		SourcePid:    &sourcePid1,
		SourceActive: &sourceActive1,
	}
	_, err = database.InsertCommand(cmd1)
	require.NoError(t, err, "failed to insert command")

	cmd2 := &models.Command{
		CommandText:  "auto session cmd2",
		WorkingDir:   "/home/test",
		ExitStatus:   0,
		Timestamp:    1704470401,
		SourceApp:    &sourceApp1,
		SourcePid:    &sourcePid1,
		SourceActive: &sourceActive1,
	}
	_, err = database.InsertCommand(cmd2)
	require.NoError(t, err, "failed to insert command")

	// Insert commands from another session (bash:88888)
	sourceApp2 := "bash"
	sourcePid2 := int64(88888)
	sourceActive2 := true

	cmd3 := &models.Command{
		CommandText:  "other session cmd",
		WorkingDir:   "/home/test",
		ExitStatus:   0,
		Timestamp:    1704470402,
		SourceApp:    &sourceApp2,
		SourcePid:    &sourcePid2,
		SourceActive: &sourceActive2,
	}
	_, err = database.InsertCommand(cmd3)
	require.NoError(t, err, "failed to insert command")

	// Set up environment variables to simulate current session
	os.Setenv("SHY_SESSION_PID", "99999")
	os.Setenv("SHELL", "/bin/zsh")
	defer os.Unsetenv("SHY_SESSION_PID")
	defer os.Unsetenv("SHELL")

	// When: I run "shy last-command --current-session" (to trigger auto-detect)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"last-command", "--db", dbPath, "--current-session"})

	err = rootCmd.Execute()
	require.NoError(t, err, "last-command should succeed")

	output := buf.String()

	// Then: the output should be the most recent command from the auto-detected session
	assert.Equal(t, "auto session cmd2\n", output, "should return most recent command from auto-detected session")

	// Reset command for next test
	rootCmd.SetArgs(nil)
	lastCommandCmd.Flags().Set("offset", "1")
	lastCommandCmd.Flags().Set("session", "")
	lastCommandCmd.Flags().Set("current-session", "false")
}
