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

// TestScenario4_GetMostRecentCommandStartingWithPrefix tests like-recent with matching prefix
func TestScenario4_GetMostRecentCommandStartingWithPrefix(t *testing.T) {
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
		{"git commit -m \"a\"", 1704470401},
		{"ls -la", 1704470402},
		{"git push", 1704470403},
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

	// When: I run "shy like-recent 'git'"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "git", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()

	// Then: the output should be "git push"
	assert.Equal(t, "git push\n", output, "should output the most recent git command")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestScenario5_LikeRecentWithNoMatches tests like-recent with no matching commands
func TestScenario5_LikeRecentWithNoMatches(t *testing.T) {
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
		{"ls -la", 1704470400},
		{"pwd", 1704470401},
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

	// When: I run "shy like-recent 'docker'"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "docker", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed even with no matches")

	output := buf.String()

	// Then: the output should be empty
	assert.Equal(t, "", output, "output should be empty when no matches found")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestScenario6_LikeRecentIsCaseSensitive tests case-sensitive matching
func TestScenario6_LikeRecentIsCaseSensitive(t *testing.T) {
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
		{"Git Status", 1704470400},
		{"git status", 1704470401},
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

	// When: I run "shy like-recent 'git'"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "git", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()

	// Then: the output should be "git status" (lowercase match)
	assert.Equal(t, "git status\n", output, "should match case-sensitively")
	assert.NotContains(t, output, "Git Status", "should not match different case")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestScenario7_LikeRecentIgnoresShyCommands tests filtering of shy commands
func TestScenario7_LikeRecentIgnoresShyCommands(t *testing.T) {
	// Given: I have a database with commands including shy commands
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
		{"shy list", 1704470401},
		{"git commit", 1704470402},
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

	// When: I run "shy like-recent 'git'"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "git", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()

	// Then: the output should be "git commit"
	assert.Equal(t, "git commit\n", output, "should output the most recent git command")
	assert.NotContains(t, output, "shy list", "should not include shy commands")

	// Also test that shy commands starting with 'shy' are filtered
	buf.Reset()
	rootCmd.SetArgs([]string{"like-recent", "shy", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output = buf.String()

	// Should be empty because shy commands are filtered out
	assert.Equal(t, "", output, "shy commands should be filtered out")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}
