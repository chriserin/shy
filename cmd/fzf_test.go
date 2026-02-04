package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

func TestFzfCommand(t *testing.T) {
	// Setup database with test commands
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert test commands (some duplicates)
	commands := []string{
		"git status",
		"ls -la",
		"git commit -m 'test'",
		"git status", // duplicate
		"pwd",
		"ls -la", // duplicate
		"echo hello",
	}

	for i, cmdText := range commands {
		cmd := &models.Command{
			CommandText: cmdText,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// Run fzf command
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fzf", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Parse output (null-separated entries)
	entries := strings.Split(output, "\000")

	// Should have deduplicated entries (remove empty last entry from split)
	if entries[len(entries)-1] == "" {
		entries = entries[:len(entries)-1]
	}

	// Should have 5 unique commands (7 total - 2 duplicates)
	assert.Len(t, entries, 5, "Should have 5 unique commands after deduplication")

	// Each entry should be tab-separated: event_number<TAB>command
	for _, entry := range entries {
		parts := strings.SplitN(entry, "\t", 2)
		assert.Len(t, parts, 2, "Entry should have exactly 2 parts: '%s'", entry)

		// First part should be a number (event ID)
		assert.Regexp(t, `^\d+$`, parts[0], "First part should be event number")

		// Second part should be non-empty command text
		assert.NotEmpty(t, parts[1], "Command text should not be empty")
	}

	// Most recent commands should appear first (reverse chronological)
	// Check that "echo hello" (most recent, ID 7) appears first
	firstEntry := entries[0]
	assert.Contains(t, firstEntry, "echo hello", "Most recent command should be first")

	// Deduplication keeps the most recent occurrence:
	// - git status appears at ID 1 and 4 -> keep ID 4
	// - ls -la appears at ID 2 and 6 -> keep ID 6
	// So order should be: ID 7 (echo), 6 (ls), 4 (git status), 3 (git commit), 5 (pwd)
	// Note: We verify commands are deduplicated, not exact order

	// Verify each unique command appears exactly once
	commandCounts := make(map[string]int)
	for _, entry := range entries {
		parts := strings.SplitN(entry, "\t", 2)
		if len(parts) == 2 {
			commandCounts[parts[1]]++
		}
	}

	for cmd, count := range commandCounts {
		assert.Equal(t, 1, count, "Command '%s' should appear exactly once", cmd)
	}

	rootCmd.SetArgs(nil)
}

func TestFzfCommand_EmptyDatabase(t *testing.T) {
	// Setup empty database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Run fzf command
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fzf", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should be empty
	assert.Empty(t, output, "Output should be empty for empty database")

	rootCmd.SetArgs(nil)
}

