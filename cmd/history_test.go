package cmd

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

// Scenario 1: Display last 16 commands with event numbers (default behavior)
func TestScenario1_DisplayLast16CommandsWithEventNumbers(t *testing.T) {
	// Given: I have 50 commands in my history with event IDs 1 through 50
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert 50 commands
	for i := 1; i <= 50; i++ {
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show 16 commands
	assert.Equal(t, 16, len(lines), "should show 16 commands")

	// And: each command should be prefixed with its event number
	// And: the first line should show event 35
	assert.Contains(t, lines[0], "35", "first line should show event 35")
	assert.Contains(t, lines[0], "cmd35", "first line should show cmd35")

	// And: the last line should show event 50
	assert.Contains(t, lines[15], "50", "last line should show event 50")
	assert.Contains(t, lines[15], "cmd50", "last line should show cmd50")

	// And: the commands should be ordered oldest to newest (ascending event numbers)
	for i := 0; i < len(lines); i++ {
		expectedEventNum := 35 + i
		assert.Contains(t, lines[i], fmt.Sprintf("%d", expectedEventNum))
	}

	// Reset
	rootCmd.SetArgs(nil)
}

// Scenario 2: Display from specific event to most recent
func TestScenario2_DisplayFromSpecificEventToMostRecent(t *testing.T) {
	// Given: I have commands with event IDs 1 through 100
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for i := 1; i <= 100; i++ {
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history 80"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "80", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show 21 commands (80 through 100)
	assert.Equal(t, 21, len(lines), "should show 21 commands")

	// And: the first command should have event ID 80
	assert.Contains(t, lines[0], "80")
	assert.Contains(t, lines[0], "cmd80")

	// And: the last command should have event ID 100
	assert.Contains(t, lines[20], "100")
	assert.Contains(t, lines[20], "cmd100")

	rootCmd.SetArgs(nil)
}

// Scenario 3: Display range of commands by event number
func TestScenario3_DisplayRangeOfCommandsByEventNumber(t *testing.T) {
	// Given: I have commands with event IDs 1 through 100
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for i := 1; i <= 100; i++ {
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history 50 75"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "50", "75", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show 26 commands
	assert.Equal(t, 26, len(lines), "should show 26 commands")

	// And: the first command should have event ID 50
	assert.Contains(t, lines[0], "50")
	assert.Contains(t, lines[0], "cmd50")

	// And: the last command should have event ID 75
	assert.Contains(t, lines[25], "75")
	assert.Contains(t, lines[25], "cmd75")

	rootCmd.SetArgs(nil)
}

// Scenario 4: Display last N commands using negative offset
func TestScenario4_DisplayLastNCommandsUsingNegativeOffset(t *testing.T) {
	// Given: I have 100 commands in my history
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for i := 1; i <= 100; i++ {
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history -10"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "--db", dbPath, "--", "-10"})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show 10 commands
	assert.Equal(t, 10, len(lines), "should show 10 commands")

	// And: the first command should have event ID 91
	assert.Contains(t, lines[0], "91")

	// And: the last command should have event ID 100
	assert.Contains(t, lines[9], "100")

	rootCmd.SetArgs(nil)
}

// Scenario 5: Display from string match to most recent
func TestScenario5_DisplayFromStringMatchToMostRecent(t *testing.T) {
	// Given: I have commands with event IDs 1 through 100
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for i := 1; i <= 100; i++ {
		cmdText := fmt.Sprintf("cmd%d", i)
		// Add git commands at specific IDs
		if i == 45 {
			cmdText = "git status"
		} else if i == 67 {
			cmdText = "git commit"
		} else if i == 89 {
			cmdText = "git push"
		}

		cmd := &models.Command{
			CommandText: cmdText,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history git"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "git", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show commands from event 89 to 100
	assert.Equal(t, 12, len(lines), "should show 12 commands (89-100)")

	// And: the first command should be "git push"
	assert.Contains(t, lines[0], "89")
	assert.Contains(t, lines[0], "git push")

	rootCmd.SetArgs(nil)
}

// Scenario 6: Display from string match to specific event
func TestScenario6_DisplayFromStringMatchToSpecificEvent(t *testing.T) {
	// Given: I have commands with event IDs 1 through 100
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for i := 1; i <= 100; i++ {
		cmdText := fmt.Sprintf("cmd%d", i)
		if i == 45 {
			cmdText = "git status"
		} else if i == 67 {
			cmdText = "git commit"
		}

		cmd := &models.Command{
			CommandText: cmdText,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history git 70"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "git", "70", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show commands from event 67 to 70
	assert.Equal(t, 4, len(lines), "should show 4 commands (67-70)")

	// And: the first command should be "git commit"
	assert.Contains(t, lines[0], "67")
	assert.Contains(t, lines[0], "git commit")

	rootCmd.SetArgs(nil)
}

// Scenario 7: Display between two string matches
func TestScenario7_DisplayBetweenTwoStringMatches(t *testing.T) {
	// Given: I have commands with event IDs 1 through 100
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for i := 1; i <= 100; i++ {
		cmdText := fmt.Sprintf("cmd%d", i)
		if i == 30 {
			cmdText = "docker build"
		} else if i == 45 {
			cmdText = "git status"
		} else if i == 60 {
			cmdText = "docker run"
		} else if i == 75 {
			cmdText = "git commit"
		}

		cmd := &models.Command{
			CommandText: cmdText,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history docker git"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "docker", "git", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show commands from event 60 to 75
	assert.Equal(t, 16, len(lines), "should show 16 commands (60-75)")

	// And: the first command should be "docker run"
	assert.Contains(t, lines[0], "60")
	assert.Contains(t, lines[0], "docker run")

	// And: the last command should be "git commit"
	assert.Contains(t, lines[15], "75")
	assert.Contains(t, lines[15], "git commit")

	rootCmd.SetArgs(nil)
}

// Scenario 8: Event numbers are persistent and use database row ID
func TestScenario8_EventNumbersArePersistentAndUseRowID(t *testing.T) {
	// Given: I have commands with event IDs 1 through 10
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)

	for i := 1; i <= 10; i++ {
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}
	database.Close()

	// When: I insert a new command
	database, err = db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	cmd := &models.Command{
		CommandText: "cmd11",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   1704470411,
	}
	id, err := database.InsertCommand(cmd)
	require.NoError(t, err)

	// Then: the new command should have event ID 11
	assert.Equal(t, int64(11), id)

	// And: the event ID should match the database row ID
	retrieved, err := database.GetCommand(id)
	require.NoError(t, err)
	assert.Equal(t, id, retrieved.ID)
}

// Scenario 9: Suppress event numbers with -n flag
func TestScenario9_SuppressEventNumbersWithNFlag(t *testing.T) {
	// Given: I have 20 commands in my history
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for i := 1; i <= 20; i++ {
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history -n"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "-n", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show 16 commands without event numbers
	assert.Equal(t, 16, len(lines), "should show 16 commands")

	// And: each line should contain only the command text
	for _, line := range lines {
		// Should not have leading numbers
		assert.True(t, strings.HasPrefix(line, "cmd"), "line should start with 'cmd'")
		// Should not have extra spaces or formatting
		assert.False(t, strings.Contains(line, "  "), "should not have double spaces")
	}

	rootCmd.SetArgs(nil)
}

// Scenario 10: Reverse chronological order with -r flag
func TestScenario10_ReverseChronologicalOrderWithRFlag(t *testing.T) {
	// Given: I have commands with event IDs 1 through 20
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for i := 1; i <= 20; i++ {
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history -r"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "-r", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show 16 commands
	assert.Equal(t, 16, len(lines), "should show 16 commands")

	// And: the first command shown should have event ID 20
	assert.Contains(t, lines[0], "20")
	assert.Contains(t, lines[0], "cmd20")

	// And: the last command shown should have event ID 5
	assert.Contains(t, lines[15], "5")
	assert.Contains(t, lines[15], "cmd5")

	rootCmd.SetArgs(nil)
}

// Scenario 11: Empty history displays nothing
func TestScenario11_EmptyHistoryDisplaysNothing(t *testing.T) {
	// Given: I have an empty history database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// When: I run "shy history"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should be empty
	assert.Empty(t, output, "output should be empty")

	rootCmd.SetArgs(nil)
}

// Scenario 12: Range exceeds available history
func TestScenario12_RangeExceedsAvailableHistory(t *testing.T) {
	// Given: I have commands with event IDs 1 through 10
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for i := 1; i <= 10; i++ {
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history 5 20"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "5", "20", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show commands from event 5 to 10
	assert.Equal(t, 6, len(lines), "should show 6 commands (5-10)")

	// And: no error should occur (verified by require.NoError above)

	rootCmd.SetArgs(nil)
}

// Scenario 13: Single event number shows from that event to most recent
func TestScenario13_SingleEventNumberShowsFromEventToMostRecent(t *testing.T) {
	// Given: I have commands with event IDs 1 through 100
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for i := 1; i <= 100; i++ {
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history 100"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "100", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show exactly 1 command
	assert.Equal(t, 1, len(lines), "should show exactly 1 command")

	// And: that command should have event ID 100
	assert.Contains(t, lines[0], "100")
	assert.Contains(t, lines[0], "cmd100")

	rootCmd.SetArgs(nil)
}

// Scenario 14: String not found outputs error message
func TestScenario14_StringNotFoundOutputsErrorMessage(t *testing.T) {
	// Given: I have commands with event IDs 1 through 100
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for i := 1; i <= 100; i++ {
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history nonexistent"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"history", "nonexistent", "--db", dbPath})

	err = rootCmd.Execute()

	// And: the output should show error message
	assert.Contains(t, buf.String(), "shy fc: event not found: nonexistent")

	rootCmd.SetArgs(nil)
}

// Scenario 15: Combine negative offset with reverse flag
func TestScenario15_CombineNegativeOffsetWithReverseFlag(t *testing.T) {
	// Given: I have commands with event IDs 1 through 50
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for i := 1; i <= 50; i++ {
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history -10 -r"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "-r", "--db", dbPath, "--", "-10"})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show 10 commands
	assert.Equal(t, 10, len(lines), "should show 10 commands")

	// And: the first command should have event ID 50
	assert.Contains(t, lines[0], "50")
	assert.Contains(t, lines[0], "cmd50")

	// And: the last command should have event ID 41
	assert.Contains(t, lines[9], "41")
	assert.Contains(t, lines[9], "cmd41")

	rootCmd.SetArgs(nil)
}

// Phase 2: Timestamp Formatting Tests

// Scenario 16: Display timestamps with -d flag
func TestScenario16_DisplayTimestampsWithDFlag(t *testing.T) {
	// Given: I have a command with event ID 100 run at "2024-01-15 14:30:00"
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Parse the timestamp "2024-01-15 14:30:00" in UTC
	testTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2024-01-15 14:30:00", time.UTC)
	require.NoError(t, err)

	// Insert 100 commands, with the last one at the specific time
	for i := 1; i <= 100; i++ {
		var timestamp int64
		if i == 100 {
			timestamp = testTime.Unix()
		} else {
			timestamp = int64(1704470400 + i)
		}
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history 100 -d"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "100", "-d", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should show the event number, timestamp, and command
	assert.Contains(t, output, "100", "should show event number 100")
	assert.Contains(t, output, "cmd100", "should show command text")
	// And: the timestamp should be in a readable format (default: YYYY-MM-DD HH:MM:SS)
	assert.Contains(t, output, "2024-01-15 14:30:00", "should show timestamp in default format")

	rootCmd.SetArgs(nil)
}

// Scenario 17: Display timestamps in ISO8601 format
func TestScenario17_DisplayTimestampsISO8601(t *testing.T) {
	// Given: I have a command with event ID 100 run at "2024-01-15 14:30:00"
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	testTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2024-01-15 14:30:00", time.UTC)
	require.NoError(t, err)

	cmd := &models.Command{
		CommandText: "test command",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   testTime.Unix(),
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err)

	// When: I run "shy history 1 -i"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "1", "-i", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should contain "2024-01-15 14:30"
	assert.Contains(t, output, "2024-01-15 14:30", "should show timestamp in ISO8601 format")

	rootCmd.SetArgs(nil)
}

// Scenario 18: Display timestamps in US format
func TestScenario18_DisplayTimestampsUSFormat(t *testing.T) {
	// Given: I have a command with event ID 100 run at "2024-01-15 14:30:00"
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	testTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2024-01-15 14:30:00", time.UTC)
	require.NoError(t, err)

	cmd := &models.Command{
		CommandText: "test command",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   testTime.Unix(),
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err)

	// When: I run "shy history 1 -f"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "1", "-f", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should contain "01/15/24 14:30"
	assert.Contains(t, output, "01/15/24 14:30", "should show timestamp in US format")

	rootCmd.SetArgs(nil)
}

// Scenario 19: Display timestamps in European format
func TestScenario19_DisplayTimestampsEuropeanFormat(t *testing.T) {
	// Given: I have a command with event ID 100 run at "2024-01-15 14:30:00"
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	testTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2024-01-15 14:30:00", time.UTC)
	require.NoError(t, err)

	cmd := &models.Command{
		CommandText: "test command",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   testTime.Unix(),
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err)

	// When: I run "shy history 1 -E"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "1", "-E", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should contain "15.01.2024 14:30"
	assert.Contains(t, output, "15.01.2024 14:30", "should show timestamp in European format")

	rootCmd.SetArgs(nil)
}

// Scenario 20: Display timestamps with custom format
func TestScenario20_DisplayTimestampsCustomFormat(t *testing.T) {
	// Given: I have a command with event ID 100 run at "2024-01-15 14:30:00"
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	testTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2024-01-15 14:30:00", time.UTC)
	require.NoError(t, err)

	cmd := &models.Command{
		CommandText: "test command",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   testTime.Unix(),
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err)

	// When: I run "shy history 1 -t '%Y-%m-%d %H:%M:%S'"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "1", "-t", "%Y-%m-%d %H:%M:%S", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should contain "2024-01-15 14:30:00"
	assert.Contains(t, output, "2024-01-15 14:30:00", "should show timestamp in custom format")

	rootCmd.SetArgs(nil)
}

// Scenario 21: Display duration with -D flag
func TestScenario21_DisplayDuration(t *testing.T) {
	// Given: I have a command with a 2.5 minute duration
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Create a command with 2 minutes 30 seconds duration
	duration := int64(150000) // 150 seconds = 2m 30s

	cmd := &models.Command{
		CommandText: "test command",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   time.Now().Unix(),
		Duration:    &duration,
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err)

	// When: I run "shy history -1 -D"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "-D", "--db", dbPath, "--", "-1"})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should show duration in mm:ss format
	assert.Contains(t, output, "02:30", "should show duration as 02:30")

	rootCmd.SetArgs(nil)
}

// Scenario 22: Combine timestamp format with duration
func TestScenario22_CombineTimestampAndDuration(t *testing.T) {
	// Given: I have a command run at a specific time with duration
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Create a timestamp and duration
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	duration := int64(30000) // 30 seconds

	cmd := &models.Command{
		CommandText: "test command",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   oneHourAgo.Unix(),
		Duration:    &duration,
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err)

	// When: I run "shy history -1 -i -D"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "-i", "-D", "--db", dbPath, "--", "-1"})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should show both ISO timestamp and duration
	// Check for ISO format pattern (YYYY-MM-DD HH:MM)
	assert.Regexp(t, `\d{4}-\d{2}-\d{2} \d{2}:\d{2}`, output, "should show ISO timestamp")
	// Check for duration in mm:ss format
	assert.Contains(t, output, "00:30", "should show duration")

	rootCmd.SetArgs(nil)
}

// Phase 2: Duration Display Tests

// Scenario 9: Display duration with -D flag in history
func TestDurationScenario9_DisplayDurationWithDFlag(t *testing.T) {
	// Given: I have commands with durations in the database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert commands with various durations
	durations := []int64{1500, 45000, 120000} // 1.5s, 45s, 2m
	for i, dur := range durations {
		d := dur
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i+1),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
			Duration:    &d,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history -D"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "-D", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: the output should show duration for each command
	assert.Equal(t, 3, len(lines), "should show 3 commands")

	// And: durations should be in human-readable format 'mm:ss'
	assert.Contains(t, lines[0], "00:01", "1.5s should show as 00:01")
	assert.Contains(t, lines[1], "00:45", "45s should show as 00:45")
	assert.Contains(t, lines[2], "02:00", "120s should show as 02:00")

	rootCmd.SetArgs(nil)
}

// Scenario 11: Display duration in human-readable format
func TestDurationScenario11_DisplayDurationInHumanReadableFormat(t *testing.T) {
	// Given: I have commands with various durations
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Test cases: duration in ms, expected output
	testCases := []struct {
		durationMs int64
		expected   string
		desc       string
	}{
		{500, "00:00", "under 1 second should show 00:00"},
		{45000, "00:45", "45 seconds should show 00:45"},
		{150000, "02:30", "2 minutes 30 seconds should show 02:30"},
		{3600000, "", "1 hour should show empty string"},
		{7200000, "", "2 hours should show empty string"},
	}

	for i, tc := range testCases {
		d := tc.durationMs
		cmd := &models.Command{
			CommandText: fmt.Sprintf("cmd%d", i+1),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
			Duration:    &d,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy history -D"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "-D", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: verify each format
	for i, tc := range testCases {
		if tc.expected != "" {
			assert.Contains(t, lines[i], tc.expected, tc.desc)
		} else {
			// For empty duration (>= 1 hour), the line should still be present but without duration shown
			// It will just have the event number and command text
			assert.Contains(t, lines[i], fmt.Sprintf("cmd%d", i+1), "command should still be shown")
		}
	}

	rootCmd.SetArgs(nil)
}

// Scenario 12: Display duration alongside timestamps
func TestDurationScenario12_DisplayDurationAlongsideTimestamps(t *testing.T) {
	// Given: I have a command with timestamp and duration
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	testTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2024-01-15 14:30:00", time.UTC)
	require.NoError(t, err)

	duration := int64(125000) // 2m 5s
	cmd := &models.Command{
		CommandText: "test command",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   testTime.Unix(),
		Duration:    &duration,
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err)

	// When: I run "shy history -i -D"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "1", "-i", "-D", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should show: event number, timestamp, duration, command text
	assert.Contains(t, output, "1", "should show event number")
	assert.Contains(t, output, "2024-01-15 14:30", "should show timestamp")
	assert.Contains(t, output, "02:05", "should show duration")
	assert.Contains(t, output, "test command", "should show command text")

	// And: all fields should be properly aligned (separated by spaces)
	assert.Regexp(t, `\s+1\s+2024-01-15 14:30\s+02:05\s+test command`, output)

	rootCmd.SetArgs(nil)
}

// Scenario 13: Commands without duration show 00:00
func TestDurationScenario13_CommandsWithoutDurationShow0000(t *testing.T) {
	// Given: I have some commands with duration and some without
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Command with duration
	duration := int64(45000) // 45s
	cmd1 := &models.Command{
		CommandText: "cmd_with_duration",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   int64(1704470400),
		Duration:    &duration,
	}
	_, err = database.InsertCommand(cmd1)
	require.NoError(t, err)

	// Command without duration (nil)
	cmd2 := &models.Command{
		CommandText: "cmd_without_duration",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   int64(1704470401),
		Duration:    nil,
	}
	_, err = database.InsertCommand(cmd2)
	require.NoError(t, err)

	// When: I run "shy history -D"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "-D", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: commands with duration should show the duration
	assert.Contains(t, lines[0], "00:45", "command with duration should show 00:45")

	// And: commands without duration should show "00:00"
	assert.Contains(t, lines[1], "00:00", "command without duration should show 00:00")

	rootCmd.SetArgs(nil)
}

// Test that history supports -m flag for pattern matching (like fc -l -m)
func TestHistory_PatternMatchingFlag(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert various commands
	commands := []string{
		"git status",
		"git commit",
		"ls -la",
		"git push",
		"npm test",
	}

	for _, cmdText := range commands {
		cmd := &models.Command{
			CommandText: cmdText,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// Run: shy history -m 'git*'
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "-m", "git*", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should contain git commands
	assert.Contains(t, output, "git status")
	assert.Contains(t, output, "git commit")
	assert.Contains(t, output, "git push")

	// Should not contain non-git commands
	assert.NotContains(t, output, "ls -la")
	assert.NotContains(t, output, "npm test")

	rootCmd.SetArgs(nil)
}

// Test that history supports -I flag for internal/session filtering (like fc -l -I)
func TestHistory_InternalFlag(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert commands with different session PIDs
	currentPID := int64(12345)
	otherPID := int64(67890)
	sourceActive := true

	commands := []struct {
		text string
		pid  int64
	}{
		{"git status", currentPID},
		{"npm install", otherPID},
		{"git commit", currentPID},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText:  cmd.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    int64(1704470400),
			SourcePid:    &cmd.pid,
			SourceActive: &sourceActive,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// Set environment variable for current session
	t.Setenv("SHY_SESSION_PID", "12345")

	// Run: shy history -I
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"history", "-I", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should contain only current session commands
	assert.Contains(t, output, "git status")
	assert.Contains(t, output, "git commit")

	// Should not contain other session commands
	assert.NotContains(t, output, "npm install")

	rootCmd.SetArgs(nil)
}
