package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

func TestStarAdd(t *testing.T) {
	tempDir := t.TempDir()
	testDBPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewForTesting(testDBPath)
	require.NoError(t, err)

	cmd := models.NewCommand("echo hello", "/home/test", 0)
	cmd.Timestamp = 1704470400
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err)
	database.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"star", "add", "1", "--db", testDBPath})
	err = rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Starred command 1")

	// Verify the command is starred
	database, err = db.New(testDBPath)
	require.NoError(t, err)
	defer database.Close()

	starred, err := database.IsStarred(1)
	require.NoError(t, err)
	assert.True(t, starred)
}

func TestStarRemove(t *testing.T) {
	tempDir := t.TempDir()
	testDBPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewForTesting(testDBPath)
	require.NoError(t, err)

	cmd := models.NewCommand("echo hello", "/home/test", 0)
	cmd.Timestamp = 1704470400
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err)

	err = database.StarCommand(1)
	require.NoError(t, err)
	database.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"star", "remove", "1", "--db", testDBPath})
	err = rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Unstarred command 1")

	// Verify the command is unstarred
	database, err = db.New(testDBPath)
	require.NoError(t, err)
	defer database.Close()

	starred, err := database.IsStarred(1)
	require.NoError(t, err)
	assert.False(t, starred)
}

func TestStarList(t *testing.T) {
	tempDir := t.TempDir()
	testDBPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewForTesting(testDBPath)
	require.NoError(t, err)

	// Insert and star 2 commands
	for i := 0; i < 3; i++ {
		cmd := models.NewCommand("cmd"+string(rune('0'+i)), "/home/test", 0)
		cmd.Timestamp = int64(1704470400 + i)
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	err = database.StarCommand(1)
	require.NoError(t, err)
	err = database.StarCommand(3)
	require.NoError(t, err)
	database.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"star", "list", "--db", testDBPath})
	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "cmd0")
	assert.Contains(t, output, "cmd2")
	assert.NotContains(t, output, "cmd1")
}

func TestStarListPwd(t *testing.T) {
	tempDir := t.TempDir()
	testDBPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewForTesting(testDBPath)
	require.NoError(t, err)

	cmd1 := models.NewCommand("cmd1", "/home/test/dir1", 0)
	cmd1.Timestamp = 1704470400
	_, err = database.InsertCommand(cmd1)
	require.NoError(t, err)

	cmd2 := models.NewCommand("cmd2", "/home/test/dir2", 0)
	cmd2.Timestamp = 1704470401
	_, err = database.InsertCommand(cmd2)
	require.NoError(t, err)

	err = database.StarCommand(1)
	require.NoError(t, err)
	err = database.StarCommand(2)
	require.NoError(t, err)
	database.Close()

	// Use --pwd; the test's cwd is tempDir which doesn't match either dir,
	// so we should get 0 results
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"star", "list", "--pwd", "--db", testDBPath})
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Neither dir1 nor dir2 matches the test's CWD, so output should be empty
	assert.Empty(t, buf.String())
}

func TestStarBare(t *testing.T) {
	tempDir := t.TempDir()
	testDBPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewForTesting(testDBPath)
	require.NoError(t, err)

	// Insert commands with session info
	app := "zsh"
	pid := int64(12345)
	active := true

	cmd1 := models.NewCommand("first cmd", "/home/test", 0)
	cmd1.Timestamp = 1704470400
	cmd1.SourceApp = &app
	cmd1.SourcePid = &pid
	cmd1.SourceActive = &active
	_, err = database.InsertCommand(cmd1)
	require.NoError(t, err)

	cmd2 := models.NewCommand("second cmd", "/home/test", 0)
	cmd2.Timestamp = 1704470401
	cmd2.SourceApp = &app
	cmd2.SourcePid = &pid
	cmd2.SourceActive = &active
	_, err = database.InsertCommand(cmd2)
	require.NoError(t, err)
	database.Close()

	// Set environment variables for session detection
	t.Setenv("SHY_SESSION_PID", "12345")
	t.Setenv("SHELL", "/bin/zsh")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"star", "--db", testDBPath})
	err = rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Starred #2")

	// Verify the most recent command is starred
	database, err = db.New(testDBPath)
	require.NoError(t, err)
	defer database.Close()

	starred, err := database.IsStarred(2)
	require.NoError(t, err)
	assert.True(t, starred)
}

func TestStarAddInvalidID(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"star", "add", "abc"})
	err := rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid event ID")
}

func TestStarListCurrentSession(t *testing.T) {
	tempDir := t.TempDir()
	testDBPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewForTesting(testDBPath)
	require.NoError(t, err)

	app := "zsh"
	pid1 := int64(1001)
	pid2 := int64(2002)
	active := true

	// Session 1 command
	cmd1 := models.NewCommand("session1 cmd", "/home/test", 0)
	cmd1.Timestamp = 1704470400
	cmd1.SourceApp = &app
	cmd1.SourcePid = &pid1
	cmd1.SourceActive = &active
	_, err = database.InsertCommand(cmd1)
	require.NoError(t, err)

	// Session 2 command
	cmd2 := models.NewCommand("session2 cmd", "/home/test", 0)
	cmd2.Timestamp = 1704470401
	cmd2.SourceApp = &app
	cmd2.SourcePid = &pid2
	cmd2.SourceActive = &active
	_, err = database.InsertCommand(cmd2)
	require.NoError(t, err)

	// Star both
	err = database.StarCommand(1)
	require.NoError(t, err)
	err = database.StarCommand(2)
	require.NoError(t, err)
	database.Close()

	// Set environment for session 1
	t.Setenv("SHY_SESSION_PID", "1001")
	t.Setenv("SHELL", "/bin/zsh")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"star", "list", "--current-session", "--pwd=false", "--db", testDBPath})
	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "session1 cmd")
	assert.NotContains(t, output, "session2 cmd")
}
