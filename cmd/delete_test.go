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

func TestDeleteCommand_Basic(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		cmd := models.NewCommand("cmd", "/home/test", 0)
		cmd.Timestamp = int64(1704470400 + i)
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}
	database.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"delete", "2", "--db", dbPath})
	err = rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Deleted 1 command(s)")

	// Verify the command is gone
	database, err = db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	count, err := database.CountCommands()
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestDeleteCommand_MultipleIDs(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		cmd := models.NewCommand("cmd", "/home/test", 0)
		cmd.Timestamp = int64(1704470400 + i)
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}
	database.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"delete", "1", "2", "3", "--db", dbPath})
	err = rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Deleted 3 command(s)")
}

func TestDeleteCommand_InvalidID(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	database.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"delete", "abc", "--db", dbPath})
	err = rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid event ID")
}

func TestDeleteCommand_NonExistentID(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	database.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"delete", "999", "--db", dbPath})
	err = rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Deleted 0 command(s)")
}
