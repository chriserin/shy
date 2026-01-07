package cmd

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

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
	fcNoNum = false // Reset flag
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
	fcReverse = false // Reset flag
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

	// Then: the output should show error message
	assert.Error(t, err, "should return an error")
	assert.Contains(t, err.Error(), "shy: event not found: nonexistent")

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
	fcReverse = false // Reset flag
}
