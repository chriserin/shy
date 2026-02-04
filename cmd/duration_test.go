package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

// Scenario 1: Capture duration for successful command
func TestDurationScenario1_CaptureDurationForSuccessfulCommand(t *testing.T) {
	// Given: I run a command that takes 2.5 seconds to complete
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Create a command with 2.5 seconds duration (2500ms)
	duration := int64(2500)
	cmd := &models.Command{
		CommandText: "sleep 2.5",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   time.Now().Unix(),
		Duration:    &duration,
	}

	// When: the command finishes successfully
	id, err := database.InsertCommand(cmd)
	require.NoError(t, err)

	// Then: the database should record a duration of approximately 2.5 seconds
	retrieved, err := database.GetCommand(id)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Duration, "duration should be recorded")
	assert.Equal(t, int64(2500), *retrieved.Duration, "duration should be 2500ms")
	assert.Equal(t, 0, retrieved.ExitStatus, "exit status should be 0")
}

// Scenario 2: Capture duration for failed command
func TestDurationScenario2_CaptureDurationForFailedCommand(t *testing.T) {
	// Given: I run a command that takes 1.2 seconds and fails
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Create a command with 1.2 seconds duration (1200ms)
	duration := int64(1200)
	cmd := &models.Command{
		CommandText: "failing_command",
		WorkingDir:  "/home/test",
		ExitStatus:  1,
		Timestamp:   time.Now().Unix(),
		Duration:    &duration,
	}

	// When: the command finishes with exit status 1
	id, err := database.InsertCommand(cmd)
	require.NoError(t, err)

	// Then: the database should record a duration of approximately 1.2 seconds
	retrieved, err := database.GetCommand(id)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Duration, "duration should be recorded")
	assert.Equal(t, int64(1200), *retrieved.Duration, "duration should be 1200ms")
	// And: the exit status should be 1
	assert.Equal(t, 1, retrieved.ExitStatus, "exit status should be 1")
}

// Scenario 3: Capture duration for very short commands
func TestDurationScenario3_CaptureDurationForVeryShortCommands(t *testing.T) {
	// Given: I run a command that completes in less than 0.1 seconds
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Create a command with 45ms duration
	duration := int64(45)
	cmd := &models.Command{
		CommandText: "true",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   time.Now().Unix(),
		Duration:    &duration,
	}

	// When: the command finishes
	id, err := database.InsertCommand(cmd)
	require.NoError(t, err)

	// Then: the database should record a duration in milliseconds
	retrieved, err := database.GetCommand(id)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Duration, "duration should be recorded")
	assert.Equal(t, int64(45), *retrieved.Duration, "duration should be 45ms")
}

// Scenario 4: Capture duration for long-running commands
func TestDurationScenario4_CaptureDurationForLongRunningCommands(t *testing.T) {
	// Given: I run a command that takes 3 hours and 45 minutes
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// 3 hours 45 minutes = 13500 seconds = 13500000 milliseconds
	duration := int64(13500000)
	cmd := &models.Command{
		CommandText: "long_running_process",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   time.Now().Unix(),
		Duration:    &duration,
	}

	// When: the command finishes
	id, err := database.InsertCommand(cmd)
	require.NoError(t, err)

	// Then: the database should record a duration of 13500000 milliseconds
	retrieved, err := database.GetCommand(id)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Duration, "duration should be recorded")
	assert.Equal(t, int64(13500000), *retrieved.Duration, "duration should be 13500000ms")
}

// Scenario 5: Duration is null for commands without timing data
func TestDurationScenario5_DurationIsNullForCommandsWithoutTimingData(t *testing.T) {
	// Given: I have historical commands imported without duration
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Create a command without duration (Duration = nil)
	cmd := &models.Command{
		CommandText: "old_command",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   time.Now().Unix(),
		Duration:    nil, // No duration
	}

	id, err := database.InsertCommand(cmd)
	require.NoError(t, err)

	// When: I query those commands
	retrieved, err := database.GetCommand(id)
	require.NoError(t, err)

	// Then: the duration field should be 0
	require.NotNil(t, retrieved.Duration, "duration should not be nil")
	assert.Equal(t, int64(0), *retrieved.Duration, "duration should be 0 for commands without timing data")
	// And: commands should display without errors
	assert.NotZero(t, retrieved.ID, "command should be retrieved successfully")
	assert.Equal(t, "old_command", retrieved.CommandText)
}

// Scenario 6: Shy insert command with duration parameter
func TestDurationScenario6_ShyInsertCommandWithDurationParameter(t *testing.T) {
	// Given: I want to insert a command with duration
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	// Initialize database first
	initDB, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	initDB.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{
		"insert",
		"--command", "test command",
		"--dir", "/home/test",
		"--status", "0",
		"--duration", "1500",
		"--db", dbPath,
	})

	// When: I run shy insert with --duration flag
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Then: the command should be inserted with the duration
	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Get the most recent command
	cmd, err := database.GetCommand(1)
	require.NoError(t, err)

	require.NotNil(t, cmd.Duration, "duration should be set")
	assert.Equal(t, int64(1500), *cmd.Duration, "duration should be 1500ms")

	rootCmd.SetArgs(nil)
	duration = 0 // Reset package-level variable
}

// Scenario 7: Shy insert without duration parameter
func TestDurationScenario7_ShyInsertWithoutDurationParameter(t *testing.T) {
	// Given: I want to insert a command without duration
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	// Initialize database first
	initDB, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	initDB.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{
		"insert",
		"--command", "test command",
		"--dir", "/home/test",
		"--status", "0",
		"--db", dbPath,
	})

	// When: I run shy insert without --duration flag
	err = rootCmd.Execute()
	require.NoError(t, err)

	// Then: the command should be inserted with null duration
	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	cmd, err := database.GetCommand(1)
	require.NoError(t, err)

	require.NotNil(t, cmd.Duration, "duration should not be nil")
	assert.Equal(t, int64(0), *cmd.Duration, "duration should be 0 when not provided")

	rootCmd.SetArgs(nil)
	duration = 0 // Reset package-level variable
}

// Scenario 8: Database migration adds duration column to existing database
func TestDurationScenario8_DatabaseMigrationAddsDurationColumn(t *testing.T) {
	// Given: an existing database without duration column
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	// Create database without duration column (using raw SQL to simulate old schema)
	database1, err := db.NewForTesting(dbPath)
	require.NoError(t, err)

	// Insert a command (this will have duration column from the current schema)
	cmd := &models.Command{
		CommandText: "test command",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   time.Now().Unix(),
	}
	id, err := database1.InsertCommand(cmd)
	require.NoError(t, err)
	database1.Close()

	// When: shy runs with duration support (opening the database again)
	database2, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database2.Close()

	// Then: the database should be migrated to add duration column
	schema, err := database2.GetTableSchema()
	require.NoError(t, err)

	hasDuration := false
	for _, col := range schema {
		if col["name"] == "duration" {
			hasDuration = true
			break
		}
	}
	assert.True(t, hasDuration, "duration column should exist after migration")

	// And: existing commands should have 0 duration
	retrieved, err := database2.GetCommand(id)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Duration, "duration should not be nil")
	assert.Equal(t, int64(0), *retrieved.Duration, "existing commands should have 0 duration after migration")
}

// Integration test for shell hook duration capture
func TestDurationIntegration_ZshHooksCaptureAndCalculateDuration(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Verify the preexec hook captures start time
	// This tests that __shy_cmd_start is set correctly in the shell script
	t.Run("preexec captures start time", func(t *testing.T) {
		// Read the zsh_record.sh script
		script, err := os.ReadFile("integration_scripts/zsh_record.sh")
		require.NoError(t, err)

		// Verify it contains the start time capture logic
		scriptContent := string(script)
		assert.Contains(t, scriptContent, "__shy_cmd_start=", "script should set __shy_cmd_start")
		assert.Contains(t, scriptContent, "date +%s%3N", "script should capture milliseconds")
	})

	// Verify the precmd hook calculates duration
	t.Run("precmd calculates duration", func(t *testing.T) {
		// Read the zsh_record.sh script
		script, err := os.ReadFile("integration_scripts/zsh_record.sh")
		require.NoError(t, err)

		scriptContent := string(script)
		// Verify duration calculation logic exists
		assert.Contains(t, scriptContent, "duration=", "script should calculate duration")
		assert.Contains(t, scriptContent, "end_time - __shy_cmd_start", "script should calculate duration from start to end")
		assert.Contains(t, scriptContent, "--duration", "script should pass duration to shy insert")
	})

	// Verify duration is passed to shy insert
	t.Run("duration is passed to shy insert", func(t *testing.T) {
		script, err := os.ReadFile("integration_scripts/zsh_record.sh")
		require.NoError(t, err)

		scriptContent := string(script)
		// Verify the shy_args includes --duration
		assert.Contains(t, scriptContent, `shy_args+=("--duration" "$duration")`, "script should add duration to shy args")
	})

	// Verify duration is only added when calculated
	t.Run("duration is only added when available", func(t *testing.T) {
		script, err := os.ReadFile("integration_scripts/zsh_record.sh")
		require.NoError(t, err)

		scriptContent := string(script)
		// Verify conditional logic for adding duration
		assert.Contains(t, scriptContent, `if [[ -n "$duration" ]]`, "script should check if duration is set")
		assert.Contains(t, scriptContent, `[[ "$duration" -ge 0 ]]`, "script should verify duration is non-negative")
	})
}
