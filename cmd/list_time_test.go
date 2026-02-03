package cmd

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

// TestScenario7_ListWithFlagToday tests that --today flag shows only today's commands
func TestScenario7_ListWithFlagToday(t *testing.T) {
	// Given: I have a database with commands from various dates
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Get today at a fixed point for testing
	now := time.Date(2024, 1, 5, 12, 0, 0, 0, time.UTC)
	today := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	yesterday := time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC)
	twoDaysAgo := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

	commands := []struct {
		text      string
		timestamp time.Time
	}{
		{"cmd today 1", today.Add(2 * time.Hour)},
		{"cmd today 2", today.Add(10 * time.Hour)},
		{"cmd yesterday", yesterday.Add(10 * time.Hour)},
		{"cmd old", twoDaysAgo.Add(10 * time.Hour)},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp.Unix(),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// Mock "now" by using a specific date range
	// Since we can't easily mock time.Now(), we'll calculate the range manually
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	endOfToday := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()).Unix()

	// Query directly with the time range (simulating what --today would do)
	todayCommands, err := database.ListCommandsInRange(startOfToday, endOfToday, 0, "", 0, "")
	require.NoError(t, err, "failed to list commands in range")

	// Then: the output should contain today's commands
	var todayCmdTexts []string
	for _, cmd := range todayCommands {
		todayCmdTexts = append(todayCmdTexts, cmd.CommandText)
	}

	assert.Contains(t, todayCmdTexts, "cmd today 1", "should contain cmd today 1")
	assert.Contains(t, todayCmdTexts, "cmd today 2", "should contain cmd today 2")
	assert.NotContains(t, todayCmdTexts, "cmd yesterday", "should not contain cmd yesterday")
	assert.NotContains(t, todayCmdTexts, "cmd old", "should not contain cmd old")
}

// TestScenario8_ListWithFlagYesterday tests that --yesterday flag shows only yesterday's commands
func TestScenario8_ListWithFlagYesterday(t *testing.T) {
	// Given: I have a database with commands from various dates
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	now := time.Date(2024, 1, 5, 12, 0, 0, 0, time.UTC)
	today := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	yesterday := time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC)
	twoDaysAgo := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

	commands := []struct {
		text      string
		timestamp time.Time
	}{
		{"cmd today", today.Add(10 * time.Hour)},
		{"cmd yest 1", yesterday.Add(2 * time.Hour)},
		{"cmd yest 2", yesterday.Add(16 * time.Hour)},
		{"cmd old", twoDaysAgo.Add(10 * time.Hour)},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp.Unix(),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// Calculate yesterday's range
	yesterdayDate := now.AddDate(0, 0, -1)
	startOfYesterday := time.Date(yesterdayDate.Year(), yesterdayDate.Month(), yesterdayDate.Day(), 0, 0, 0, 0, yesterdayDate.Location()).Unix()
	endOfYesterday := time.Date(yesterdayDate.Year(), yesterdayDate.Month(), yesterdayDate.Day(), 23, 59, 59, 0, yesterdayDate.Location()).Unix()

	yesterdayCommands, err := database.ListCommandsInRange(startOfYesterday, endOfYesterday, 0, "", 0, "")
	require.NoError(t, err, "failed to list commands in range")

	var yesterdayCmdTexts []string
	for _, cmd := range yesterdayCommands {
		yesterdayCmdTexts = append(yesterdayCmdTexts, cmd.CommandText)
	}

	assert.Contains(t, yesterdayCmdTexts, "cmd yest 1", "should contain cmd yest 1")
	assert.Contains(t, yesterdayCmdTexts, "cmd yest 2", "should contain cmd yest 2")
	assert.NotContains(t, yesterdayCmdTexts, "cmd today", "should not contain cmd today")
	assert.NotContains(t, yesterdayCmdTexts, "cmd old", "should not contain cmd old")
}

// TestScenario9_ListWithFlagThisWeek tests that --this-week flag shows this week's commands
func TestScenario9_ListWithFlagThisWeek(t *testing.T) {
	// Given: I have a database with commands from various dates
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Friday, January 5, 2024 (week starting Monday, Jan 1)
	now := time.Date(2024, 1, 5, 12, 0, 0, 0, time.UTC)
	monday := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)     // Monday this week
	wednesday := time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC)  // Wednesday this week
	friday := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)     // Friday this week
	lastWeek := time.Date(2023, 12, 25, 10, 0, 0, 0, time.UTC) // Last week

	commands := []struct {
		text      string
		timestamp time.Time
	}{
		{"cmd monday", monday},
		{"cmd wednesday", wednesday},
		{"cmd friday", friday},
		{"cmd last week", lastWeek},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp.Unix(),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// Calculate this week's range (Monday to Sunday)
	weekday := int(now.Weekday())
	if weekday == 0 { // Sunday
		weekday = 7
	}
	daysFromMonday := weekday - 1
	thisMonday := now.AddDate(0, 0, -daysFromMonday)
	startOfWeek := time.Date(thisMonday.Year(), thisMonday.Month(), thisMonday.Day(), 0, 0, 0, 0, thisMonday.Location()).Unix()
	thisSunday := thisMonday.AddDate(0, 0, 6)
	endOfWeek := time.Date(thisSunday.Year(), thisSunday.Month(), thisSunday.Day(), 23, 59, 59, 0, thisSunday.Location()).Unix()

	weekCommands, err := database.ListCommandsInRange(startOfWeek, endOfWeek, 0, "", 0, "")
	require.NoError(t, err, "failed to list commands in range")

	var weekCmdTexts []string
	for _, cmd := range weekCommands {
		weekCmdTexts = append(weekCmdTexts, cmd.CommandText)
	}

	assert.Contains(t, weekCmdTexts, "cmd monday", "should contain cmd monday")
	assert.Contains(t, weekCmdTexts, "cmd wednesday", "should contain cmd wednesday")
	assert.Contains(t, weekCmdTexts, "cmd friday", "should contain cmd friday")
	assert.NotContains(t, weekCmdTexts, "cmd last week", "should not contain cmd last week")
}

// TestScenario10_ListWithFlagLastWeek tests that --last-week flag shows last week's commands
func TestScenario10_ListWithFlagLastWeek(t *testing.T) {
	// Given: I have a database with commands from various dates
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Friday, January 5, 2024
	now := time.Date(2024, 1, 5, 12, 0, 0, 0, time.UTC)
	thisWeek := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)     // Friday this week
	lastMonday := time.Date(2023, 12, 25, 10, 0, 0, 0, time.UTC) // Monday last week
	lastFriday := time.Date(2023, 12, 29, 10, 0, 0, 0, time.UTC) // Friday last week
	twoWeeks := time.Date(2023, 12, 18, 10, 0, 0, 0, time.UTC)   // Two weeks ago

	commands := []struct {
		text      string
		timestamp time.Time
	}{
		{"cmd this week", thisWeek},
		{"cmd last mon", lastMonday},
		{"cmd last fri", lastFriday},
		{"cmd two weeks", twoWeeks},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp.Unix(),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// Calculate last week's range
	weekday := int(now.Weekday())
	if weekday == 0 { // Sunday
		weekday = 7
	}
	daysFromMonday := weekday - 1
	thisMonday := now.AddDate(0, 0, -daysFromMonday)
	prevMonday := thisMonday.AddDate(0, 0, -7)
	startOfLastWeek := time.Date(prevMonday.Year(), prevMonday.Month(), prevMonday.Day(), 0, 0, 0, 0, prevMonday.Location()).Unix()
	prevSunday := prevMonday.AddDate(0, 0, 6)
	endOfLastWeek := time.Date(prevSunday.Year(), prevSunday.Month(), prevSunday.Day(), 23, 59, 59, 0, prevSunday.Location()).Unix()

	lastWeekCommands, err := database.ListCommandsInRange(startOfLastWeek, endOfLastWeek, 0, "", 0, "")
	require.NoError(t, err, "failed to list commands in range")

	var lastWeekCmdTexts []string
	for _, cmd := range lastWeekCommands {
		lastWeekCmdTexts = append(lastWeekCmdTexts, cmd.CommandText)
	}

	assert.Contains(t, lastWeekCmdTexts, "cmd last mon", "should contain cmd last mon")
	assert.Contains(t, lastWeekCmdTexts, "cmd last fri", "should contain cmd last fri")
	assert.NotContains(t, lastWeekCmdTexts, "cmd this week", "should not contain cmd this week")
	assert.NotContains(t, lastWeekCmdTexts, "cmd two weeks", "should not contain cmd two weeks")
}
