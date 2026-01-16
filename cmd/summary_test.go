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

// resetSummaryFlags resets all summary flags to default values
func resetSummaryFlags() {
	summaryDate = "yesterday"
	summaryAllCommands = false
}

// TestSummary_Yesterday tests displaying yesterday's commands with --all-commands
func TestSummary_Yesterday(t *testing.T) {
	resetSummaryFlags()

	// Given: commands from yesterday
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Yesterday's date
	yesterday := time.Now().AddDate(0, 0, -1)
	year, month, day := yesterday.Date()

	repo := "github.com/chris/shy"
	branch := "main"

	commands := []struct {
		text    string
		hour    int
		minute  int
		working string
		repo    *string
		branch  *string
	}{
		{"git status", 8, 30, "/home/user/projects/shy", &repo, &branch},
		{"go build -o shy .", 9, 15, "/home/user/projects/shy", &repo, &branch},
		{"go test ./...", 14, 20, "/home/user/projects/shy", &repo, &branch},
	}

	for _, c := range commands {
		timestamp := time.Date(year, month, day, c.hour, c.minute, 0, 0, time.Local).Unix()
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  c.working,
			GitRepo:     c.repo,
			GitBranch:   c.branch,
			ExitStatus:  0,
			Timestamp:   timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy summary --yesterday --all-commands"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"summary", "--date", "yesterday", "--all-commands", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "summary should succeed")

	output := buf.String()
	expectedDate := yesterday.Format("2006-01-02")

	// Then: output should contain date header
	assert.Contains(t, output, fmt.Sprintf("Work Summary - %s", expectedDate))

	// And: output should contain context with directory:branch format
	assert.Contains(t, output, "/home/user/projects/shy:main")

	// And: output should contain hourly buckets with dashes
	assert.Contains(t, output, "8am ------------------------------")
	assert.Contains(t, output, "9am ------------------------------")
	assert.Contains(t, output, "2pm ------------------------------")

	// And: output should contain commands with minute-only timestamps
	assert.Contains(t, output, ":30  git status")
	assert.Contains(t, output, ":15  go build -o shy .")
	assert.Contains(t, output, ":20  go test ./...")

	// Reset
	rootCmd.SetArgs(nil)
}

// TestSummary_SpecificDate tests summary for specific date
func TestSummary_SpecificDate(t *testing.T) {
	resetSummaryFlags()

	// Given: commands from 2026-01-10
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	repo := "github.com/user/app"
	branch := "feature"

	commands := []struct {
		text   string
		hour   int
		minute int
	}{
		{"npm install", 10, 0},
		{"npm test", 10, 30},
	}

	for _, c := range commands {
		timestamp := time.Date(2026, 1, 10, c.hour, c.minute, 0, 0, time.Local).Unix()
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/user/projects/app",
			GitRepo:     &repo,
			GitBranch:   &branch,
			ExitStatus:  0,
			Timestamp:   timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy summary --date 2026-01-10 --all-commands"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"summary", "--date", "2026-01-10", "--all-commands", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "summary should succeed")

	output := buf.String()

	// Then: output should contain correct date
	assert.Contains(t, output, "Work Summary - 2026-01-10")

	// And: output should contain context with directory:branch format
	assert.Contains(t, output, "/home/user/projects/app:feature")

	// And: output should contain commands with minute-only timestamps
	assert.Contains(t, output, ":00  npm install")
	assert.Contains(t, output, ":30  npm test")

	// Reset
	rootCmd.SetArgs(nil)
}

// TestSummary_EmptyDay tests summary when no commands exist
func TestSummary_EmptyDay(t *testing.T) {
	resetSummaryFlags()

	// Given: empty database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// When: I run "shy summary --date 2026-01-14 --all-commands"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"summary", "--date", "2026-01-14", "--all-commands", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "summary should succeed")

	output := buf.String()

	// Then: output should show no commands message
	assert.Contains(t, output, "No commands found for 2026-01-14")

	// Reset
	rootCmd.SetArgs(nil)
}

// TestSummary_MultipleBranches tests summary with multiple branches
func TestSummary_MultipleBranches(t *testing.T) {
	resetSummaryFlags()

	// Given: commands from multiple branches
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	repo := "github.com/chris/shy"
	mainBranch := "main"
	featureBranch := "feature-a"

	commands := []struct {
		text   string
		hour   int
		branch *string
	}{
		{"git checkout main", 8, &mainBranch},
		{"go test ./...", 9, &mainBranch},
		{"git checkout -b feature-a", 10, &featureBranch},
		{"vim cmd/new.go", 11, &featureBranch},
	}

	for _, c := range commands {
		timestamp := time.Date(2026, 1, 14, c.hour, 0, 0, 0, time.Local).Unix()
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   c.branch,
			ExitStatus:  0,
			Timestamp:   timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy summary --date 2026-01-14 --all-commands"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"summary", "--date", "2026-01-14", "--all-commands", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "summary should succeed")

	output := buf.String()

	// Then: output should contain both branches in directory:branch format
	assert.Contains(t, output, "/home/user/projects/shy:feature-a")
	assert.Contains(t, output, "/home/user/projects/shy:main")

	// And: branches should be sorted alphabetically (feature-a before main)
	featureIndex := strings.Index(output, ":feature-a")
	mainIndex := strings.Index(output, ":main")
	assert.Less(t, featureIndex, mainIndex, "feature-a should appear before main")

	// And: statistics should show 2 branches
	assert.Contains(t, output, "Branches worked on: 2")

	// Reset
	rootCmd.SetArgs(nil)
}

// TestSummary_MixedGitAndNonGit tests mixed git and non-git directories
func TestSummary_MixedGitAndNonGit(t *testing.T) {
	resetSummaryFlags()

	// Given: commands from git repo and non-git directory
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	repo := "github.com/chris/shy"
	branch := "main"

	commands := []struct {
		text   string
		hour   int
		dir    string
		repo   *string
		branch *string
	}{
		{"go build", 9, "/home/user/projects/shy", &repo, &branch},
		{"wget file.zip", 10, "/home/user/downloads", nil, nil},
		{"unzip file.zip", 11, "/home/user/downloads", nil, nil},
	}

	for _, c := range commands {
		timestamp := time.Date(2026, 1, 14, c.hour, 0, 0, 0, time.Local).Unix()
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  c.dir,
			GitRepo:     c.repo,
			GitBranch:   c.branch,
			ExitStatus:  0,
			Timestamp:   timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy summary --date 2026-01-14 --all-commands"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"summary", "--date", "2026-01-14", "--all-commands", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "summary should succeed")

	output := buf.String()

	// Then: output should contain git context with directory:branch format
	assert.Contains(t, output, "/home/user/projects/shy:main")

	// And: output should contain non-git context with (non-git) label
	assert.Contains(t, output, "/home/user/downloads")

	// And: statistics should show mixed contexts
	assert.Contains(t, output, "Unique contexts: 2 (1 repos, 1 non-repo dir)")

	// Reset
	rootCmd.SetArgs(nil)
}

// TestSummary_AllTimePeriods tests commands distributed across all time periods
func TestSummary_AllTimePeriods(t *testing.T) {
	resetSummaryFlags()

	// Given: commands in all time periods
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	repo := "github.com/chris/shy"
	branch := "main"

	commands := []struct {
		text string
		hour int
	}{
		{"git commit -m \"late night\"", 2},
		{"git status", 8},
		{"go test ./...", 13},
		{"git push", 19},
	}

	for _, c := range commands {
		timestamp := time.Date(2026, 1, 14, c.hour, 0, 0, 0, time.Local).Unix()
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			ExitStatus:  0,
			Timestamp:   timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy summary --date 2026-01-14 --all-commands"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"summary", "--date", "2026-01-14", "--all-commands", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "summary should succeed")

	output := buf.String()

	// Then: output should contain hourly buckets
	// Commands at hours: 2 (2am), 8 (8am), 13 (1pm), 19 (7pm)
	assert.Contains(t, output, "2am")
	assert.Contains(t, output, "8am")
	assert.Contains(t, output, "1pm")
	assert.Contains(t, output, "7pm")

	// And: hours should be in chronological order
	hour2Index := strings.Index(output, "2am")
	hour8Index := strings.Index(output, "8am")
	hour13Index := strings.Index(output, "1pm")
	hour19Index := strings.Index(output, "7pm")

	assert.Less(t, hour2Index, hour8Index, "2am should appear before 8am")
	assert.Less(t, hour8Index, hour13Index, "8am should appear before 1pm")
	assert.Less(t, hour13Index, hour19Index, "1pm should appear before 7pm")

	// Reset
	rootCmd.SetArgs(nil)
}

// TestSummary_InvalidDate tests error handling for invalid date
func TestSummary_InvalidDate(t *testing.T) {
	resetSummaryFlags()

	// Given: empty database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// When: I run "shy summary --date invalid-date"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"summary", "--date", "invalid-date", "--db", dbPath})

	err = rootCmd.Execute()

	// Then: command should fail
	require.Error(t, err, "summary should fail with invalid date")
	assert.Contains(t, err.Error(), "invalid date format")

	// Reset
	rootCmd.SetArgs(nil)
}

// TestParseDateRange tests date parsing logic
func TestParseDateRange(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{"yesterday", "yesterday", false},
		{"today", "today", false},
		{"specific date", "2026-01-14", false},
		{"invalid format", "01-14-2026", true},
		{"invalid string", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime, endTime, dateStr, err := parseDateRange(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Greater(t, endTime, startTime, "end time should be after start time")
				assert.NotEmpty(t, dateStr, "date string should not be empty")

				// Verify the range is exactly 24 hours
				diff := endTime - startTime
				assert.Equal(t, int64(86400), diff, "range should be exactly 24 hours")
			}
		})
	}
}
