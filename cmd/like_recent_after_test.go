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

// resetLikeRecentAfterFlags resets all like-recent-after flags to default values
func resetLikeRecentAfterFlags() {
	likeRecentAfterPrev = ""
	likeRecentAfterLimit = 1
	likeRecentAfterExclude = ""
	likeRecentAfterIncludeShy = false
	// Reset cobra flag state by re-initializing the command flags
	likeRecentAfterCmd.Flags().Set("prev", "")
	likeRecentAfterCmd.Flags().Set("limit", "1")
	likeRecentAfterCmd.Flags().Set("exclude", "")
	likeRecentAfterCmd.Flags().Set("include-shy", "false")
}

// TestLikeRecentAfterWithMatchingContext tests basic context-aware matching
func TestLikeRecentAfterWithMatchingContext(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"git commit -m \"initial\"", 1704470400},
		{"git push origin main", 1704470401},
		{"git commit -m \"fix bug\"", 1704470402},
		{"git push origin feature", 1704470403},
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

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent-after", "git p", "--prev", "git commit -m \"initial\"", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent-after should succeed")

	output := buf.String()
	assert.Equal(t, "git push origin main\n", output, "should match command after specified previous command")

	rootCmd.SetArgs(nil)
	resetLikeRecentAfterFlags()
}

// TestLikeRecentAfterWithDifferentContext tests different contexts
func TestLikeRecentAfterWithDifferentContext(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"git commit -m \"initial\"", 1704470400},
		{"git push origin main", 1704470401},
		{"git commit -m \"fix bug\"", 1704470402},
		{"git push origin feature", 1704470403},
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

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent-after", "git p", "--prev", "git commit -m \"fix bug\"", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent-after should succeed")

	output := buf.String()
	assert.Equal(t, "git push origin feature\n", output, "should match command after different context")

	rootCmd.SetArgs(nil)
	resetLikeRecentAfterFlags()
}

// TestLikeRecentAfterWithNoContextMatch tests when context doesn't match
func TestLikeRecentAfterWithNoContextMatch(t *testing.T) {
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
		{"git commit", 1704470401},
		{"npm install", 1704470402},
		{"npm test", 1704470403},
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

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent-after", "npm", "--prev", "git status", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent-after should succeed")

	output := buf.String()
	assert.Equal(t, "", output, "should return empty when context doesn't match (git status not immediately before npm commands)")

	rootCmd.SetArgs(nil)
	resetLikeRecentAfterFlags()
}

// TestLikeRecentAfterExcludesShyByDefault tests shy command filtering
func TestLikeRecentAfterExcludesShyByDefault(t *testing.T) {
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
		{"shy fc -l", 1704470401},
		{"git status", 1704470402},
		{"git commit", 1704470403},
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

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent-after", "shy", "--prev", "git status", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent-after should succeed")

	output := buf.String()
	assert.Equal(t, "", output, "should exclude shy commands by default")

	rootCmd.SetArgs(nil)
	resetLikeRecentAfterFlags()
}

// TestLikeRecentAfterWithIncludeShy tests --include-shy flag
func TestLikeRecentAfterWithIncludeShy(t *testing.T) {
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
		{"shy fc -l", 1704470401},
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

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent-after", "shy", "--prev", "git status", "--include-shy", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent-after should succeed")

	output := buf.String()
	assert.Equal(t, "shy fc -l\n", output, "should include shy commands with flag")

	rootCmd.SetArgs(nil)
	resetLikeRecentAfterFlags()
}

// TestLikeRecentAfterWithLimit tests --limit flag
func TestLikeRecentAfterWithLimit(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"git commit", 1704470400},
		{"git push origin main", 1704470401},
		{"git commit", 1704470402},
		{"git push origin dev", 1704470403},
		{"git commit", 1704470404},
		{"git push origin test", 1704470405},
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

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent-after", "git push", "--prev", "git commit", "--limit", "2", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent-after should succeed")

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	assert.Len(t, lines, 2, "should return exactly 2 results")
	assert.Equal(t, "git push origin test", string(lines[0]), "first should be most recent")
	assert.Equal(t, "git push origin dev", string(lines[1]))

	rootCmd.SetArgs(nil)
	resetLikeRecentAfterFlags()
}

// TestLikeRecentAfterWithoutPrevFlag tests error when --prev is missing
func TestLikeRecentAfterWithoutPrevFlag(t *testing.T) {
	// Reset flags to ensure prev is not set from previous test
	resetLikeRecentAfterFlags()
	// Also explicitly clear the cobra flag to reset Changed() state
	likeRecentAfterCmd.Flags().Lookup("prev").Changed = false

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	database.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"like-recent-after", "git", "--db", dbPath})

	err = rootCmd.Execute()
	require.Error(t, err, "like-recent-after should fail without --prev flag")
	assert.Contains(t, err.Error(), "prev", "error should mention prev flag")

	rootCmd.SetArgs(nil)
	resetLikeRecentAfterFlags()
}

// TestLikeRecentAfterWithEmptyPrev tests behavior with empty --prev
func TestLikeRecentAfterWithEmptyPrev(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	cmd := &models.Command{
		CommandText: "git status",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   1704470400,
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent-after", "git", "--prev", "", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent-after should succeed with empty prev")

	output := buf.String()
	assert.Equal(t, "", output, "should return empty with empty prev")

	rootCmd.SetArgs(nil)
	resetLikeRecentAfterFlags()
}

// TestLikeRecentAfterWithMissingDatabase tests graceful handling of missing database
func TestLikeRecentAfterWithMissingDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "nonexistent.db")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent-after", "git", "--prev", "git commit", "--db", dbPath})

	err := rootCmd.Execute()
	require.NoError(t, err, "like-recent-after should succeed with missing database")

	output := buf.String()
	assert.Equal(t, "", output, "should return empty with missing database")

	rootCmd.SetArgs(nil)
	resetLikeRecentAfterFlags()
}

// TestLikeRecentAfterWithEmptyDatabase tests behavior with empty database
func TestLikeRecentAfterWithEmptyDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	database.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent-after", "git", "--prev", "git commit", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent-after should succeed with empty database")

	output := buf.String()
	assert.Equal(t, "", output, "should return empty with empty database")

	rootCmd.SetArgs(nil)
	resetLikeRecentAfterFlags()
}

// TestLikeRecentAfterWithFirstCommand tests behavior when matching first command
func TestLikeRecentAfterWithFirstCommand(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	cmd := &models.Command{
		CommandText: "git status",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   1704470400,
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent-after", "git", "--prev", "anything", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent-after should succeed")

	output := buf.String()
	assert.Equal(t, "", output, "should return empty for first command")

	rootCmd.SetArgs(nil)
	resetLikeRecentAfterFlags()
}

// TestLikeRecentAfterWithSpecialCharacters tests special character handling
func TestLikeRecentAfterWithSpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"git commit -m \"fix: bug\"", 1704470400},
		{"echo \"test string\"", 1704470401},
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

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent-after", "echo", "--prev", "git commit -m \"fix: bug\"", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent-after should succeed")

	output := buf.String()
	assert.Equal(t, "echo \"test string\"\n", output, "should handle special characters")

	rootCmd.SetArgs(nil)
	resetLikeRecentAfterFlags()
}
