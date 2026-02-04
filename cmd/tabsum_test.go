package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

func TestTabsumCommand(t *testing.T) {
	// Get user's home directory for path normalization tests
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	t.Run("multiple contexts sorted by duration", func(t *testing.T) {
		// Given: commands executed on 2026-01-14 in multiple contexts
		tempDir := t.TempDir()
		testDbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(testDbPath)
		require.NoError(t, err)

		// shy:yesterdays-summary - 8h 12m duration
		insertCommand(t, database, "git checkout -b", homeDir+"/shy", "yesterdays-summary", "2026-01-14 08:23:15")
		insertCommand(t, database, "git commit", homeDir+"/shy", "yesterdays-summary", "2026-01-14 16:35:42")

		// webapp:feature/auth - 5h 27m duration
		insertCommand(t, database, "npm install", homeDir+"/webapp", "feature/auth", "2026-01-14 10:15:20")
		insertCommand(t, database, "npm test", homeDir+"/webapp", "feature/auth", "2026-01-14 15:42:33")

		// shy:main - 2h 17m duration
		insertCommand(t, database, "git checkout main", homeDir+"/shy", "main", "2026-01-14 19:15:12")
		insertCommand(t, database, "git push", homeDir+"/shy", "main", "2026-01-14 21:32:45")

		// Close database before running command
		database.Close()

		// When: shy tabsum is run for 2026-01-14
		output := runTabsumWithDate(t, testDbPath, "2026-01-14")

		// Then: output should contain all expected strings
		assertContains(t, output, "Work Summary - 2026-01-14")
		assertContains(t, output, "Directory")
		assertContains(t, output, "Branch")
		assertContains(t, output, "Commands")
		assertContains(t, output, "Time Span")
		assertContains(t, output, "Duration")

		// Check contexts are sorted by duration (longest first)
		assertContains(t, output, "~/shy")
		assertContains(t, output, "yesterdays-summary")
		assertContains(t, output, "8h 12m")

		assertContains(t, output, "~/webapp")
		assertContains(t, output, "feature/auth")
		assertContains(t, output, "5h 27m")

		assertContains(t, output, "main")
		assertContains(t, output, "2h 17m")

		assertContains(t, output, "Total: 6 commands across 3 contexts")

		// Verify order: yesterdays-summary should come before main (longer duration)
		idxYesterday := strings.Index(output, "yesterdays-summary")
		idxMain := strings.Index(output, "main")
		assert.Less(t, idxYesterday, idxMain, "yesterdays-summary should appear before main")
	})

	t.Run("specific date with --date flag", func(t *testing.T) {
		// Given: commands on 2026-01-10
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		insertCommand(t, database, "go build", homeDir+"/shy", "main", "2026-01-10 09:00:00")
		insertCommand(t, database, "go test", homeDir+"/shy", "main", "2026-01-10 17:30:00")

		// When: shy tabsum --date 2026-01-10
		output := runTabsumWithDate(t, dbPath, "2026-01-10")

		// Then: output shows correct date
		assertContains(t, output, "Work Summary - 2026-01-10")
		assertContains(t, output, "~/shy")
		assertContains(t, output, "main")
		assertContains(t, output, "8h 30m")
		assertContains(t, output, "Total: 2 commands across 1 context")
	})

	t.Run("today's summary", func(t *testing.T) {
		// Given: commands executed today
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		today := time.Now()
		todayStr := today.Format("2006-01-02")
		todayMorning := time.Date(today.Year(), today.Month(), today.Day(), 10, 0, 0, 0, time.Local)
		todayLater := todayMorning.Add(1*time.Hour + 15*time.Minute)

		cmd1 := models.NewCommand("go build", homeDir+"/shy", 0)
		cmd1.Timestamp = todayMorning.Unix()
		cmd1.GitBranch = stringPtr("main")
		_, err = database.InsertCommand(cmd1)
		require.NoError(t, err)

		cmd2 := models.NewCommand("go test", homeDir+"/shy", 0)
		cmd2.Timestamp = todayLater.Unix()
		cmd2.GitBranch = stringPtr("main")
		_, err = database.InsertCommand(cmd2)
		require.NoError(t, err)

		// When: shy tabsum --date today
		output := runTabsumWithDate(t, dbPath, "today")

		// Then: output shows today's date
		assertContains(t, output, "Work Summary - "+todayStr)
		assertContains(t, output, "~/shy")
		assertContains(t, output, "main")
		assertContains(t, output, "1h 15m")
	})

	t.Run("no commands found", func(t *testing.T) {
		// Given: database with no commands on 2026-01-13
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// When: shy tabsum --date 2026-01-13
		output := runTabsumWithDate(t, dbPath, "2026-01-13")

		// Then: shows empty state message
		assertContains(t, output, "Work Summary - 2026-01-13")
		assertContains(t, output, "No commands found for this date.")
	})

	t.Run("non-git directory shows dash for branch", func(t *testing.T) {
		// Given: commands in non-git directory
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		insertCommandNoGit(t, database, "vim .bashrc", homeDir+"/dotfiles", "2026-01-14 14:10:15")
		insertCommandNoGit(t, database, "source", homeDir+"/dotfiles", "2026-01-14 14:35:22")

		// When: shy tabsum --date 2026-01-14
		output := runTabsumWithDate(t, dbPath, "2026-01-14")

		// Then: branch column shows "-"
		assertContains(t, output, "~/dotfiles")
		assertContains(t, output, "-")
		assertContains(t, output, "25m")
	})

	t.Run("mixed git and non-git directories", func(t *testing.T) {
		// Given: commands in both git and non-git directories
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		insertCommand(t, database, "go build", homeDir+"/shy", "main", "2026-01-14 08:00:00")
		insertCommand(t, database, "go test", homeDir+"/shy", "main", "2026-01-14 12:00:00")

		insertCommandNoGit(t, database, "vim .bashrc", homeDir+"/dotfiles", "2026-01-14 14:10:15")
		insertCommandNoGit(t, database, "source", homeDir+"/dotfiles", "2026-01-14 14:35:22")

		// When: shy tabsum
		output := runTabsumWithDate(t, dbPath, "2026-01-14")

		// Then: both contexts shown correctly
		assertContains(t, output, "~/shy")
		assertContains(t, output, "main")
		assertContains(t, output, "~/dotfiles")
		assertContains(t, output, "-")
	})

	t.Run("contexts sorted by duration longest first", func(t *testing.T) {
		// Given: three contexts with different durations
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// short - 5m
		insertCommand(t, database, "echo z", homeDir+"/short", "main", "2026-01-14 08:00:00")
		insertCommand(t, database, "echo z", homeDir+"/short", "main", "2026-01-14 08:05:00")

		// long - 9h
		insertCommand(t, database, "echo a", homeDir+"/long", "main", "2026-01-14 08:00:00")
		insertCommand(t, database, "echo a", homeDir+"/long", "main", "2026-01-14 17:00:00")

		// medium - 1h 30m
		insertCommand(t, database, "echo b", homeDir+"/medium", "main", "2026-01-14 10:00:00")
		insertCommand(t, database, "echo b", homeDir+"/medium", "main", "2026-01-14 11:30:00")

		// When: shy tabsum
		output := runTabsumWithDate(t, dbPath, "2026-01-14")

		// Then: contexts ordered by duration
		idxLong := strings.Index(output, "~/long")
		idxMedium := strings.Index(output, "~/medium")
		idxShort := strings.Index(output, "~/short")

		assert.Less(t, idxLong, idxMedium, "long should appear before medium")
		assert.Less(t, idxMedium, idxShort, "medium should appear before short")
	})

	t.Run("duration formatting for various time ranges", func(t *testing.T) {
		// Given: contexts with different duration lengths
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// 10h 30m
		insertCommand(t, database, "start", homeDir+"/long", "main", "2026-01-14 08:00:00")
		insertCommand(t, database, "end", homeDir+"/long", "main", "2026-01-14 18:30:00")

		// 45m
		insertCommand(t, database, "start", homeDir+"/medium", "dev", "2026-01-14 10:00:00")
		insertCommand(t, database, "end", homeDir+"/medium", "dev", "2026-01-14 10:45:00")

		// 45s
		insertCommand(t, database, "start", homeDir+"/short", "test", "2026-01-14 14:00:00")
		insertCommand(t, database, "end", homeDir+"/short", "test", "2026-01-14 14:00:45")

		// When: shy tabsum
		output := runTabsumWithDate(t, dbPath, "2026-01-14")

		// Then: durations formatted correctly
		assertContains(t, output, "10h 30m")
		assertContains(t, output, "45m")
		assertContains(t, output, "45s")
	})

	t.Run("single command shows zero duration", func(t *testing.T) {
		// Given: only one command in a context
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		insertCommand(t, database, "go build", homeDir+"/shy", "main", "2026-01-14 10:00:00")

		// When: shy tabsum
		output := runTabsumWithDate(t, dbPath, "2026-01-14")

		// Then: shows 0s duration
		assertContains(t, output, "0s")
		assertContains(t, output, "10:00 - 10:00")
	})

	t.Run("multiple branches in same directory are separate contexts", func(t *testing.T) {
		// Given: multiple branches in same working directory
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		insertCommand(t, database, "go build", homeDir+"/shy", "feature-a", "2026-01-14 08:00:00")
		insertCommand(t, database, "go test", homeDir+"/shy", "feature-a", "2026-01-14 12:00:00")

		insertCommand(t, database, "go build", homeDir+"/shy", "feature-b", "2026-01-14 13:00:00")
		insertCommand(t, database, "go test", homeDir+"/shy", "feature-b", "2026-01-14 14:00:00")

		insertCommand(t, database, "git merge", homeDir+"/shy", "main", "2026-01-14 16:00:00")
		insertCommand(t, database, "git push", homeDir+"/shy", "main", "2026-01-14 17:00:00")

		// When: shy tabsum
		output := runTabsumWithDate(t, dbPath, "2026-01-14")

		// Then: shows 3 separate contexts
		assertContains(t, output, "feature-a")
		assertContains(t, output, "feature-b")
		assertContains(t, output, "main")
		assertContains(t, output, "Total: 6 commands across 3 contexts")
	})

	t.Run("yesterday keyword", func(t *testing.T) {
		// Given: commands executed yesterday
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		yesterday := time.Now().AddDate(0, 0, -1)
		yesterdayStr := yesterday.Format("2006-01-02")
		yesterdayMorning := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 10, 0, 0, 0, time.Local)

		cmd := models.NewCommand("go build", homeDir+"/shy", 0)
		cmd.Timestamp = yesterdayMorning.Unix()
		cmd.GitBranch = stringPtr("main")
		_, err = database.InsertCommand(cmd)
		require.NoError(t, err)

		// When: shy tabsum --date yesterday
		output := runTabsumWithDate(t, dbPath, "yesterday")

		// Then: shows yesterday's date with "Yesterday's" label
		assertContains(t, output, "Work Summary - "+yesterdayStr)
	})

	t.Run("command count accuracy across contexts", func(t *testing.T) {
		// Given: different command counts in different contexts
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// shy:main - 3 commands
		for i := 0; i < 3; i++ {
			insertCommand(t, database, "echo", homeDir+"/shy", "main", "2026-01-14 08:00:00")
		}

		// shy:dev - 2 commands
		for i := 0; i < 2; i++ {
			insertCommand(t, database, "echo", homeDir+"/shy", "dev", "2026-01-14 11:00:00")
		}

		// webapp (no git) - 1 command
		insertCommandNoGit(t, database, "echo", homeDir+"/webapp", "2026-01-14 13:00:00")

		// When: shy tabsum
		output := runTabsumWithDate(t, dbPath, "2026-01-14")

		// Then: command counts are correct
		assertContains(t, output, "Total: 6 commands across 3 contexts")

		// Check individual counts are in output (even though formatting may vary)
		lines := strings.Split(output, "\n")
		var found3, found2, found1 bool
		for _, line := range lines {
			if strings.Contains(line, "main") && strings.Contains(line, "3") {
				found3 = true
			}
			if strings.Contains(line, "dev") && strings.Contains(line, "2") {
				found2 = true
			}
			if strings.Contains(line, "webapp") && strings.Contains(line, "1") {
				found1 = true
			}
		}
		assert.True(t, found3, "Should show 3 commands for main")
		assert.True(t, found2, "Should show 2 commands for dev")
		assert.True(t, found1, "Should show 1 command for webapp")
	})

	t.Run("sort by command count when durations equal", func(t *testing.T) {
		// Given: contexts with equal duration but different command counts
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// projA: 2 commands, 1h duration
		insertCommand(t, database, "echo a", homeDir+"/projA", "main", "2026-01-14 08:00:00")
		insertCommand(t, database, "echo b", homeDir+"/projA", "main", "2026-01-14 09:00:00")

		// projB: 4 commands, 1h duration
		insertCommand(t, database, "echo c", homeDir+"/projB", "main", "2026-01-14 10:00:00")
		for i := 0; i < 3; i++ {
			insertCommand(t, database, "echo d", homeDir+"/projB", "main", "2026-01-14 11:00:00")
		}

		// When: shy tabsum
		output := runTabsumWithDate(t, dbPath, "2026-01-14")

		// Then: projB appears before projA (more commands)
		idxProjB := strings.Index(output, "~/projB")
		idxProjA := strings.Index(output, "~/projA")
		assert.Less(t, idxProjB, idxProjA, "projB should appear before projA")
	})

	t.Run("sort by directory when duration and count equal", func(t *testing.T) {
		// Given: contexts with equal duration and command count
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// zebra: 2 commands, 1h duration
		insertCommand(t, database, "echo z", homeDir+"/zebra", "main", "2026-01-14 08:00:00")
		insertCommand(t, database, "echo z", homeDir+"/zebra", "main", "2026-01-14 09:00:00")

		// apple: 2 commands, 1h duration
		insertCommand(t, database, "echo a", homeDir+"/apple", "main", "2026-01-14 10:00:00")
		insertCommand(t, database, "echo a", homeDir+"/apple", "main", "2026-01-14 11:00:00")

		// When: shy tabsum
		output := runTabsumWithDate(t, dbPath, "2026-01-14")

		// Then: apple appears before zebra (alphabetical)
		idxApple := strings.Index(output, "~/apple")
		idxZebra := strings.Index(output, "~/zebra")
		assert.Less(t, idxApple, idxZebra, "apple should appear before zebra")
	})
}

// Helper functions
func insertCommand(t *testing.T, database *db.DB, cmdText, workingDir, gitBranch, timeStr string) {
	cmd := models.NewCommand(cmdText, workingDir, 0)
	cmd.Timestamp = parseTimeLocal(timeStr)
	cmd.GitBranch = stringPtr(gitBranch)
	_, err := database.InsertCommand(cmd)
	require.NoError(t, err)
}

func insertCommandNoGit(t *testing.T, database *db.DB, cmdText, workingDir, timeStr string) {
	cmd := models.NewCommand(cmdText, workingDir, 0)
	cmd.Timestamp = parseTimeLocal(timeStr)
	_, err := database.InsertCommand(cmd)
	require.NoError(t, err)
}

func runTabsumWithDate(t *testing.T, testDbPath string, date string) string {
	// Save original dbPath and restore after test
	originalDbPath := dbPath
	defer func() { dbPath = originalDbPath }()

	// Set global dbPath for the command to use
	dbPath = testDbPath

	// Create a new command instance to avoid state issues
	cmd := &cobra.Command{
		Use:   "tabsum",
		Short: "Show tabular summary of time spent in different contexts",
		Long:  "Display a tabular summary showing time spent in different working directories and git branches",
		RunE:  runTabsum,
	}

	cmd.Flags().StringVar(&tabsumDate, "date", "yesterday", "Date to summarize (yesterday, today, or YYYY-MM-DD)")

	// Set arguments
	cmd.SetArgs([]string{"--date", date})

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Run command
	err := cmd.Execute()
	require.NoError(t, err)

	return buf.String()
}

func assertContains(t *testing.T, output, substr string) {
	t.Helper()
	if !strings.Contains(output, substr) {
		t.Errorf("Output does not contain %q\nOutput:\n%s", substr, output)
	}
}

func parseTimeLocal(timeStr string) int64 {
	// Parse the time string and explicitly set the timezone to Local
	t, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		panic(err)
	}

	// Create a new time in the local timezone with the same date/time components
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	localTime := time.Date(year, month, day, hour, min, sec, 0, time.Local)

	return localTime.Unix()
}
