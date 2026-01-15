package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
	"github.com/stretchr/testify/require"
)

func TestTVConfig(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer
	tvConfigCmd.SetOut(&buf)
	tvConfigCmd.SetErr(&buf)

	// Execute the command
	err := tvConfigCmd.RunE(tvConfigCmd, []string{})
	require.NoError(t, err)

	output := buf.String()

	// Verify TOML structure
	require.Contains(t, output, "[metadata]")
	require.Contains(t, output, `name = "shy-history"`)
	require.Contains(t, output, `description = "Browse shell command history from shy database"`)
	require.Contains(t, output, `requirements = ["shy"]`)

	require.Contains(t, output, "[source]")
	require.Contains(t, output, `command = "shy tv list"`)
	require.Contains(t, output, `display = "{split:\t:1}"`)
	require.Contains(t, output, `output = "{split:\t:0}"`)

	require.Contains(t, output, "[preview]")
	require.Contains(t, output, `command = "shy tv preview '{split:\t:0}'"`)
}

func TestTVList(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert test commands
	timestamp := time.Now().Unix()
	commands := []struct {
		text string
		id   int64
	}{
		{"ls -la", 0},
		{"cd /tmp", 0},
		{"echo hello", 0},
		{"ls -la", 0}, // Duplicate - should be deduplicated
	}

	for i, cmd := range commands {
		command := &models.Command{
			Timestamp:   timestamp + int64(i),
			ExitStatus:  0,
			CommandText: cmd.text,
			WorkingDir:  "/home/test",
		}
		id, err := database.InsertCommand(command)
		require.NoError(t, err)
		commands[i].id = id
	}

	// Create a buffer to capture output
	var buf bytes.Buffer

	// Test the underlying function
	err = database.GetCommandsForFzf(func(id int64, cmdText string) error {
		_, err := buf.WriteString(strings.Join([]string{
			string(rune(id + '0')),
			cmdText,
		}, "\t") + "\n")
		return err
	})
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have 3 unique commands (deduplicated)
	require.Equal(t, 3, len(lines))

	// Verify format: event_number<TAB>command
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		require.Equal(t, 2, len(parts), "Each line should have exactly 2 tab-separated fields")
		require.NotEmpty(t, parts[0], "Event number should not be empty")
		require.NotEmpty(t, parts[1], "Command text should not be empty")
	}
}

func TestTVPreview(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert test commands in the same session
	timestamp := time.Now().Unix()
	sessionPid := int64(12345)
	sourceApp := "zsh"
	duration := int64(100)
	gitRepo := "git@github.com:user/repo.git"
	gitBranch := "main"

	commands := []string{
		"command 1",
		"command 2",
		"command 3", // Target
		"command 4",
		"command 5",
	}

	var targetID int64
	for i, cmdText := range commands {
		command := &models.Command{
			Timestamp:   timestamp + int64(i),
			ExitStatus:  0,
			CommandText: cmdText,
			WorkingDir:  "/home/test",
			Duration:    &duration,
			GitRepo:     &gitRepo,
			GitBranch:   &gitBranch,
			SourceApp:   &sourceApp,
			SourcePid:   &sessionPid,
		}
		id, err := database.InsertCommand(command)
		require.NoError(t, err)
		if i == 2 { // Middle command
			targetID = id
		}
	}

	// Get command with context
	beforeCmds, targetCmd, afterCmds, err := database.GetCommandWithContext(targetID, 5)
	require.NoError(t, err)

	// Create a buffer to capture output
	var buf bytes.Buffer
	displayCommandWithContext(&buf, beforeCmds, targetCmd, afterCmds)

	output := buf.String()

	// Verify metadata section is present and first
	require.Contains(t, output, "Event:")
	require.Contains(t, output, "Command:")
	require.Contains(t, output, "Exit Status:")
	require.Contains(t, output, "Timestamp:")
	require.Contains(t, output, "Working Dir:")
	require.Contains(t, output, "Duration:")
	require.Contains(t, output, "Git Repo:")
	require.Contains(t, output, "Git Branch:")

	// Verify metadata appears before command lists
	metadataIdx := strings.Index(output, "Event:")
	require.Greater(t, metadataIdx, -1)

	// Verify Session field is present (format: source:pid or source:pid:X)
	require.Contains(t, output, "Session:")

	// Verify old source fields are NOT present (replaced by Session)
	require.NotContains(t, output, "Source App:")
	require.NotContains(t, output, "Source PID:")
	require.NotContains(t, output, "Active:")

	// Verify commands before are listed
	require.Contains(t, output, "command 1")
	require.Contains(t, output, "command 2")

	// Verify target command is listed
	require.Contains(t, output, "command 3")

	// Verify commands after are listed
	require.Contains(t, output, "command 4")
	require.Contains(t, output, "command 5")

	// Verify simple format for before/after commands (event_number command_text)
	// Note: Output now includes ANSI color codes, so we check for the pattern with colors
	lines := strings.Split(output, "\n")
	var foundSimpleFormat bool
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Look for lines that contain context commands (command 1-5)
		// These will be in simple format with color codes
		if (strings.Contains(trimmed, "command 1") ||
			strings.Contains(trimmed, "command 2") ||
			strings.Contains(trimmed, "command 4") ||
			strings.Contains(trimmed, "command 5")) &&
			!strings.Contains(trimmed, "Exit Status") &&
			!strings.Contains(trimmed, "Timestamp") {
			foundSimpleFormat = true
			break
		}
	}
	require.True(t, foundSimpleFormat, "Should find simple format lines for context commands")
}

func TestTVPreviewNoContext(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert single command
	timestamp := time.Now().Unix()
	command := &models.Command{
		Timestamp:   timestamp,
		ExitStatus:  0,
		CommandText: "single command",
		WorkingDir:  "/home/test",
	}
	id, err := database.InsertCommand(command)
	require.NoError(t, err)

	// Get command with context
	beforeCmds, targetCmd, afterCmds, err := database.GetCommandWithContext(id, 5)
	require.NoError(t, err)
	require.Empty(t, beforeCmds)
	require.Empty(t, afterCmds)

	// Create a buffer to capture output
	var buf bytes.Buffer
	displayCommandWithContext(&buf, beforeCmds, targetCmd, afterCmds)

	output := buf.String()

	// Verify metadata is still present
	require.Contains(t, output, "Event:")
	require.Contains(t, output, "Command:")
	require.Contains(t, output, "single command")

	// Should only show the command once in simple format
	count := strings.Count(output, "single command")
	require.Equal(t, 2, count, "Command should appear twice: once in metadata, once in simple list")
}

func TestDisplaySimpleCommand(t *testing.T) {
	var buf bytes.Buffer
	cmd := &models.Command{
		ID:          42,
		CommandText: "test command",
	}

	displaySimpleCommand(&buf, cmd)

	output := buf.String()
	// Output now includes ANSI color codes
	require.Contains(t, output, "42")
	require.Contains(t, output, "test command")
	// Verify it has the gray color code (242)
	require.Contains(t, output, "38;5;242")
}

func TestFormatDurationHuman(t *testing.T) {
	tests := []struct {
		name     string
		millis   *int64
		expected string
	}{
		{
			name:     "nil duration",
			millis:   nil,
			expected: "0s",
		},
		{
			name:     "milliseconds",
			millis:   ptr(int64(500)),
			expected: "500ms",
		},
		{
			name:     "seconds",
			millis:   ptr(int64(5000)),
			expected: "5s",
		},
		{
			name:     "minutes and seconds",
			millis:   ptr(int64(125000)),
			expected: "2m 5s",
		},
		{
			name:     "hours, minutes and seconds",
			millis:   ptr(int64(7325000)),
			expected: "2h 2m 5s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDurationHuman(tt.millis)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestTVPreviewInactiveSession(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert command with inactive session
	timestamp := time.Now().Unix()
	sessionPid := int64(99999)
	sourceApp := "bash"
	inactive := false
	duration := int64(500)

	command := &models.Command{
		Timestamp:    timestamp,
		ExitStatus:   1,
		CommandText:  "test command",
		WorkingDir:   "/tmp",
		Duration:     &duration,
		SourceApp:    &sourceApp,
		SourcePid:    &sessionPid,
		SourceActive: &inactive,
	}
	id, err := database.InsertCommand(command)
	require.NoError(t, err)

	// Get command with context
	beforeCmds, targetCmd, afterCmds, err := database.GetCommandWithContext(id, 5)
	require.NoError(t, err)

	// Create a buffer to capture output
	var buf bytes.Buffer
	displayCommandWithContext(&buf, beforeCmds, targetCmd, afterCmds)

	output := buf.String()

	// Verify Session field includes :X for inactive sessions
	require.Contains(t, output, "Session:")
	require.Contains(t, output, "bash:99999:X")
}

func ptr(i int64) *int64 {
	return &i
}
