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

// Scenario 32: Session close marks all internal commands
func TestScenario32_SessionCloseMarksAllInternalCommands(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Setup test data
	sourceApp1 := "zsh"
	sourcePid1 := int64(12345)
	sourceActive1 := true

	sourceApp2 := "zsh"
	sourcePid2 := int64(67890)
	sourceActive2 := true

	commands := []struct {
		text   string
		app    *string
		pid    *int64
		active *bool
	}{
		{"cmd1", &sourceApp1, &sourcePid1, &sourceActive1},
		{"cmd2", &sourceApp1, &sourcePid1, &sourceActive1},
		{"cmd3", &sourceApp2, &sourcePid2, &sourceActive2},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText:  cmd.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    1704470400,
			SourceApp:    cmd.app,
			SourcePid:    cmd.pid,
			SourceActive: cmd.active,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// When the session with pid 12345 closes
	count, err := database.CloseSession(12345)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "should have closed 2 commands")

	// Then commands from session 12345 should have source_active as false
	cmd1, err := database.GetCommand(1)
	require.NoError(t, err)
	require.NotNil(t, cmd1.SourceActive)
	assert.False(t, *cmd1.SourceActive)

	cmd2, err := database.GetCommand(2)
	require.NoError(t, err)
	require.NotNil(t, cmd2.SourceActive)
	assert.False(t, *cmd2.SourceActive)

	// And command from session 67890 should still have source_active as true
	cmd3, err := database.GetCommand(3)
	require.NoError(t, err)
	require.NotNil(t, cmd3.SourceActive)
	assert.True(t, *cmd3.SourceActive)
}

// Test close-session command
func TestCloseSessionCommand(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Setup test data
	sourceApp := "zsh"
	sourcePid := int64(12345)
	sourceActive := true

	commands := []struct {
		text string
	}{
		{"cmd1"},
		{"cmd2"},
		{"cmd3"},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText:  cmd.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    1704470400,
			SourceApp:    &sourceApp,
			SourcePid:    &sourcePid,
			SourceActive: &sourceActive,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// Run close-session command
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"close-session", "--pid", "12345", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify all commands are now inactive
	for i := 1; i <= 3; i++ {
		cmd, err := database.GetCommand(int64(i))
		require.NoError(t, err)
		require.NotNil(t, cmd.SourceActive)
		assert.False(t, *cmd.SourceActive, "command %d should be inactive", i)
	}

	rootCmd.SetArgs(nil)
}

// Test close-session with no matching session
func TestCloseSessionCommand_NoMatchingSession(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Run close-session command on non-existent session
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"close-session", "--pid", "99999", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err) // Should succeed even if no commands found

	rootCmd.SetArgs(nil)
}

// Test close-session idempotency
func TestCloseSessionCommand_Idempotent(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Setup test data
	sourceApp := "zsh"
	sourcePid := int64(12345)
	sourceActive := true

	c := &models.Command{
		CommandText:  "test-cmd",
		WorkingDir:   "/home/test",
		ExitStatus:   0,
		Timestamp:    1704470400,
		SourceApp:    &sourceApp,
		SourcePid:    &sourcePid,
		SourceActive: &sourceActive,
	}
	_, err = database.InsertCommand(c)
	require.NoError(t, err)

	// Close session first time
	count, err := database.CloseSession(12345)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Close session second time - should affect 0 rows
	count, err = database.CloseSession(12345)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "second close should affect 0 rows")
}
