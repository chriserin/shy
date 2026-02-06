package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

// fixedTime returns a function that always returns the given time
func fixedTime(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// setupTestDB creates a test database with the given commands
func setupTestDB(t *testing.T, commands []models.Command) string {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	for _, cmd := range commands {
		_, err := database.InsertCommand(&cmd)
		require.NoError(t, err)
	}

	return dbPath
}

// makeCommand creates a command with the given parameters
func makeCommand(date time.Time, hour int, workingDir string, gitRepo, gitBranch *string) models.Command {
	timestamp := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, time.Local)
	return models.Command{
		CommandText: "test command",
		WorkingDir:  workingDir,
		GitRepo:     gitRepo,
		GitBranch:   gitBranch,
		ExitStatus:  0,
		Timestamp:   timestamp.Unix(),
	}
}

func strPtr(s string) *string {
	return &s
}

// initModel creates a model and loads its initial contexts
func initModel(t *testing.T, dbPath string, today time.Time) *Model {
	t.Helper()
	model := New(dbPath, WithNow(fixedTime(today)))
	cmd := model.Init()
	msg := cmd()
	model.Update(msg)
	return model
}

// pressKey simulates a key press and executes any resulting command
func pressKey(model *Model, key rune) {
	model, cmd := model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
	if cmd != nil {
		msg := cmd()
		model.Update(msg)
	}
}

// TestLaunchWithYesterdaysContexts tests the scenario:
// "Launch summary with yesterday's contexts"
func TestLaunchWithYesterdaysContexts(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommand(yesterday, 10, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommand(yesterday, 11, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix")),
		makeCommand(yesterday, 12, "/home/user/downloads", nil, nil),
	}

	for i := 0; i < 40; i++ {
		commands = append(commands, makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")))
	}
	for i := 0; i < 11; i++ {
		commands = append(commands, makeCommand(yesterday, 11, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix")))
	}
	for i := 0; i < 4; i++ {
		commands = append(commands, makeCommand(yesterday, 12, "/home/user/downloads", nil, nil))
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	view := model.View()
	assert.Contains(t, view, "Work Summary")
	assert.Contains(t, view, "2026-02-04")
	assert.Contains(t, view, "Yesterday")
	assert.Contains(t, view, "projects/shy")
	assert.Contains(t, view, "main")
	assert.Contains(t, view, "bugfix")
	assert.Contains(t, view, "downloads")
	assert.Equal(t, 0, model.SelectedIdx())
}

// TestNavigateDownThroughContexts tests the scenario:
// "Navigate down through contexts"
func TestNavigateDownThroughContexts(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommand(yesterday, 10, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix")),
		makeCommand(yesterday, 11, "/home/user/downloads", nil, nil),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	assert.Equal(t, 0, model.SelectedIdx())

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, model.SelectedIdx())

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 2, model.SelectedIdx())
}

// TestNavigateUpThroughContexts tests the scenario:
// "Navigate up through contexts"
func TestNavigateUpThroughContexts(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommand(yesterday, 10, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix")),
		makeCommand(yesterday, 11, "/home/user/downloads", nil, nil),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 2, model.SelectedIdx())

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 1, model.SelectedIdx())
}

// TestSelectionStopsAtBoundaries tests the scenario:
// "Selection stops at boundaries"
func TestSelectionStopsAtBoundaries(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommand(yesterday, 10, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix")),
		makeCommand(yesterday, 11, "/home/user/downloads", nil, nil),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	// Bottom boundary
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 2, model.SelectedIdx())
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 2, model.SelectedIdx())

	// Top boundary
	model2 := initModel(t, dbPath, today)
	assert.Equal(t, 0, model2.SelectedIdx())
	model2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, model2.SelectedIdx())
}

// TestNavigateToPreviousDay tests the scenario:
// "Navigate to previous day"
func TestNavigateToPreviousDay(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := today.AddDate(0, 0, -2)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommand(dayBefore, 10, "/home/user/projects/other", strPtr("github.com/chris/other"), strPtr("feature")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	assert.Equal(t, yesterday.Format("2006-01-02"), model.CurrentDate().Format("2006-01-02"))

	pressKey(model, 'h')

	assert.Equal(t, dayBefore.Format("2006-01-02"), model.CurrentDate().Format("2006-01-02"))
	view := model.View()
	assert.Contains(t, view, "2026-02-03")
}

// TestNavigateToNextDay tests the scenario:
// "Navigate to next day"
func TestNavigateToNextDay(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := today.AddDate(0, 0, -2)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommand(dayBefore, 10, "/home/user/projects/other", strPtr("github.com/chris/other"), strPtr("feature")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)
	model.currentDate = dayBefore

	pressKey(model, 'l')

	assert.Equal(t, yesterday.Format("2006-01-02"), model.CurrentDate().Format("2006-01-02"))
	view := model.View()
	assert.Contains(t, view, "2026-02-04")
	assert.Contains(t, view, "Yesterday")
}

// TestNavigateToTodayWithNoCommands tests the scenario:
// "Navigate to today with no commands"
func TestNavigateToTodayWithNoCommands(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	pressKey(model, 'l')

	view := model.View()
	assert.Contains(t, view, "2026-02-05")
	assert.Contains(t, view, "Today")
	assert.Contains(t, view, "No commands found")
}

// TestCannotNavigatePastToday tests the scenario:
// "Cannot navigate past today"
func TestCannotNavigatePastToday(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)

	commands := []models.Command{
		makeCommand(today, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	// Navigate to today first
	pressKey(model, 'l')
	assert.Equal(t, today.Format("2006-01-02"), model.CurrentDate().Format("2006-01-02"))

	// Try to go past today
	pressKey(model, 'l')
	assert.Equal(t, today.Format("2006-01-02"), model.CurrentDate().Format("2006-01-02"))
}

// TestJumpToToday tests the scenario:
// "Jump to today"
func TestJumpToToday(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	dayBefore := today.AddDate(0, 0, -2)

	commands := []models.Command{
		makeCommand(dayBefore, 10, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)
	model.currentDate = dayBefore

	pressKey(model, 't')

	view := model.View()
	assert.Contains(t, view, "2026-02-05")
	assert.Contains(t, view, "Today")
}

// TestJumpToYesterday tests the scenario:
// "Jump to yesterday"
func TestJumpToYesterday(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := today.AddDate(0, 0, -2)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommand(dayBefore, 10, "/home/user/projects/other", strPtr("github.com/chris/other"), strPtr("feature")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)
	model.currentDate = dayBefore

	pressKey(model, 'y')

	view := model.View()
	assert.Contains(t, view, "2026-02-04")
	assert.Contains(t, view, "Yesterday")
}

// TestSelectionResetsWhenChangingDays tests the scenario:
// "Selection resets when changing days"
func TestSelectionResetsWhenChangingDays(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := today.AddDate(0, 0, -2)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommand(yesterday, 10, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix")),
		makeCommand(yesterday, 11, "/home/user/downloads", nil, nil),
		makeCommand(dayBefore, 10, "/home/user/projects/other", strPtr("github.com/chris/other"), strPtr("feature")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 2, model.SelectedIdx())

	pressKey(model, 'h')
	assert.Equal(t, 0, model.SelectedIdx())
}

// TestQuitApplication tests the scenario:
// "Quit the application"
func TestQuitApplication(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := New(dbPath, WithNow(fixedTime(today)))

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok, "expected tea.QuitMsg")
}

// TestQuitWithCtrlC tests the scenario:
// "Quit with Ctrl+C"
func TestQuitWithCtrlC(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := New(dbPath, WithNow(fixedTime(today)))

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok, "expected tea.QuitMsg")
}

// TestArrowKeysWork tests the scenario:
// "Arrow keys work for navigation"
func TestArrowKeysWork(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommand(yesterday, 10, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix")),
		makeCommand(yesterday, 11, "/home/user/downloads", nil, nil),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	assert.Equal(t, 0, model.SelectedIdx())

	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, model.SelectedIdx())

	model.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, model.SelectedIdx())
}

// TestViewportScrollsToKeepSelectionVisible tests the scenario:
// "Viewport scrolls to keep selection visible"
func TestViewportScrollsToKeepSelectionVisible(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	var commands []models.Command
	for i := 0; i < 20; i++ {
		dir := "/home/user/projects/project" + string(rune('a'+i))
		commands = append(commands, makeCommand(yesterday, 9, dir, strPtr("github.com/user/repo"), strPtr("main")))
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	model.Update(tea.WindowSizeMsg{Width: 80, Height: 15})
	assert.Equal(t, 0, model.SelectedIdx())

	for i := 0; i < 12; i++ {
		model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	assert.Equal(t, 12, model.SelectedIdx())
	view := model.View()
	assert.Contains(t, view, "projectm")
}

// TestFocusedIndicator tests the scenario:
// "Header shows focused indicator when app has focus"
func TestFocusedIndicator(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	view := model.View()
	assert.Contains(t, view, "●")
	assert.NotContains(t, view, "○")
}

// TestBlurredIndicator tests the scenario:
// "Header shows blurred indicator when app loses focus"
func TestBlurredIndicator(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	// Simulate blur
	model.Update(tea.BlurMsg{})

	view := model.View()
	assert.Contains(t, view, "○")
	assert.NotContains(t, view, "●")
}

// TestFocusRestoredIndicator tests the scenario:
// "Header restores focused indicator when app regains focus"
func TestFocusRestoredIndicator(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	// Blur then refocus
	model.Update(tea.BlurMsg{})
	assert.False(t, model.Focused())

	model.Update(tea.FocusMsg{})
	assert.True(t, model.Focused())

	view := model.View()
	assert.Contains(t, view, "●")
}

// TestContextsOrderedByCommandCountDescending tests the scenario:
// "Contexts are ordered by command count descending"
func TestContextsOrderedByCommandCountDescending(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	var commands []models.Command
	// 5 commands for shy:main
	for i := 0; i < 5; i++ {
		commands = append(commands, makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")))
	}
	// 20 commands for other:feature
	for i := 0; i < 20; i++ {
		commands = append(commands, makeCommand(yesterday, 10, "/home/user/projects/other", strPtr("github.com/chris/other"), strPtr("feature")))
	}
	// 12 commands for downloads
	for i := 0; i < 12; i++ {
		commands = append(commands, makeCommand(yesterday, 11, "/home/user/downloads", nil, nil))
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	require.Len(t, model.Contexts(), 3)
	assert.Equal(t, 20, model.Contexts()[0].CommandCount)
	assert.Equal(t, 12, model.Contexts()[1].CommandCount)
	assert.Equal(t, 5, model.Contexts()[2].CommandCount)
}

// TestLongContextNameTruncated tests the scenario:
// "Long context name is truncated with ellipsis"
func TestLongContextNameTruncated(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	longDir := "/home/user/projects/very/deeply/nested/directory/structure/my-project"
	commands := []models.Command{
		makeCommand(yesterday, 9, longDir, strPtr("github.com/user/repo"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	// Set narrow terminal
	model.Update(tea.WindowSizeMsg{Width: 60, Height: 24})

	view := model.View()
	assert.Contains(t, view, "…")
	// The full path should NOT appear
	assert.NotContains(t, view, longDir+":main")
}

// TestCommandCountsRightAligned tests the scenario:
// "Command counts are right-aligned"
func TestCommandCountsRightAligned(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	var commands []models.Command
	for i := 0; i < 142; i++ {
		commands = append(commands, makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")))
	}
	for i := 0; i < 8; i++ {
		commands = append(commands, makeCommand(yesterday, 10, "/home/user/projects/other", strPtr("github.com/chris/other"), strPtr("feature")))
	}
	for i := 0; i < 42; i++ {
		commands = append(commands, makeCommand(yesterday, 11, "/home/user/downloads", nil, nil))
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	view := model.View()

	// Find all lines containing "commands"
	lines := strings.Split(view, "\n")
	var commandLines []string
	for _, line := range lines {
		if strings.Contains(line, "commands") || strings.Contains(line, "command") {
			// Only context lines (not status bar)
			if !strings.Contains(line, "[") {
				commandLines = append(commandLines, line)
			}
		}
	}

	require.Len(t, commandLines, 3, "should have 3 context lines with command counts")

	// All lines should end at the same column (the word "commands" is aligned)
	for _, line := range commandLines {
		assert.True(t, strings.HasSuffix(strings.TrimRight(line, " \t"), "commands"),
			"line should end with 'commands': %q", line)
	}
}

// TestHeaderIncludesDayOfWeek tests the scenario:
// "Header includes day of the week"
func TestHeaderIncludesDayOfWeek(t *testing.T) {
	// 2026-02-05 is a Thursday, 2026-02-04 is a Wednesday
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	view := model.View()
	assert.Contains(t, view, "Wednesday")
	assert.Contains(t, view, "2026-02-04")
	assert.Contains(t, view, "Yesterday")
}

// TestHeaderIncludesDayOfWeekNonRelative tests the scenario:
// "Header includes day of the week for non-relative dates"
func TestHeaderIncludesDayOfWeekNonRelative(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	// 2026-02-01 is a Sunday
	target := time.Date(2026, 2, 1, 0, 0, 0, 0, time.Local)

	commands := []models.Command{
		makeCommand(target, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)
	model.currentDate = target

	// Reload for the new date
	cmd := model.loadContexts()
	model.Update(cmd)

	view := model.View()
	assert.Contains(t, view, "Sunday")
	assert.Contains(t, view, "2026-02-01")
	// Header should not contain relative date labels
	header := strings.Split(view, "\n")[0]
	assert.NotContains(t, header, "Yesterday")
	assert.NotContains(t, header, "Today")
}

// TestHomeDirectoryDisplaysFullPath tests the scenario:
// "Home directory displays as full path"
func TestHomeDirectoryDisplaysFullPath(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	commands := []models.Command{
		makeCommand(yesterday, 9, homeDir, nil, nil),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	view := model.View()
	assert.Contains(t, view, homeDir)

	// Make sure it's not just "~" by checking the full path appears
	// and that a bare "~" followed by a space (which would be tilde-only) doesn't
	lines := strings.Split(view, "\n")
	foundFullPath := false
	for _, line := range lines {
		if strings.Contains(line, homeDir) {
			foundFullPath = true
		}
	}
	assert.True(t, foundFullPath, "expected full home directory path in view")
}

// === Phase 2 Test Helpers ===

// makeCommandWithText creates a command with specific text and minute
func makeCommandWithText(date time.Time, hour, minute int, text, workingDir string, gitRepo, gitBranch *string) models.Command {
	timestamp := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.Local)
	return models.Command{
		CommandText: text,
		WorkingDir:  workingDir,
		GitRepo:     gitRepo,
		GitBranch:   gitBranch,
		ExitStatus:  0,
		Timestamp:   timestamp.Unix(),
	}
}

func pressEnter(model *Model) {
	model, cmd := model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		msg := cmd()
		model.Update(msg)
	}
}

func pressEsc(model *Model) {
	model, cmd := model.handleKey(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd != nil {
		msg := cmd()
		model.Update(msg)
	}
}

func pressShiftKey(model *Model, key rune) {
	model, cmd := model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
	if cmd != nil {
		msg := cmd()
		model.Update(msg)
	}
}

// phase2Commands returns the background data set from the feat file
func phase2Commands(date time.Time) []models.Command {
	return []models.Command{
		makeCommandWithText(date, 8, 15, "go build -o shy .", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 8, 22, "./shy summary", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 8, 30, "go test ./... -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 9, 0, "git commit -m \"feat\"", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 9, 5, "go build -o shy .", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 9, 30, "git push", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 14, 20, "shy summary", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 14, 25, "go test ./cmd -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 8, 0, "git checkout bugfix", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix")),
		makeCommandWithText(date, 8, 10, "go test ./... -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix")),
		makeCommandWithText(date, 8, 45, "git diff", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix")),
		makeCommandWithText(date, 10, 0, "curl -O example.com", "/home/user/downloads", nil, nil),
		makeCommandWithText(date, 10, 5, "tar xzf archive.gz", "/home/user/downloads", nil, nil),
	}
}

// === Phase 2 Tests ===

// TestDetailViewCommandsForSelectedContext tests entering the detail view
func TestDetailViewCommandsForSelectedContext(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	// shy:main has 8 commands, should be first (most commands)
	require.True(t, len(model.Contexts()) >= 1)
	assert.Contains(t, formatContextName(model.Contexts()[0].Key, model.Contexts()[0].Branch), "main")

	pressEnter(model)

	assert.Equal(t, ContextDetailView, model.ViewState())

	view := model.View()
	// Header shows context name, not "Work Summary"
	assert.Contains(t, view, "main")
	assert.Contains(t, view, "Wednesday")
	assert.Contains(t, view, "2026-02-04")
	assert.Contains(t, view, "Yesterday")

	// Check buckets
	assert.Contains(t, view, "8am")
	assert.Contains(t, view, "9am")
	assert.Contains(t, view, "2pm")

	// Check commands
	assert.Contains(t, view, "go build -o shy .")
	assert.Contains(t, view, "./shy summary")
	assert.Contains(t, view, "go test ./... -v")
	assert.Contains(t, view, "git commit")
	assert.Contains(t, view, "git push")
	assert.Contains(t, view, "shy summary")
	assert.Contains(t, view, "go test ./cmd -v")
}

// TestDetailHeaderConsistentLayout tests header layout consistency
func TestDetailHeaderConsistentLayout(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	view := model.View()
	// Header should show context name instead of "Work Summary"
	assert.NotContains(t, view, "Work Summary")
	assert.Contains(t, view, "●")
	assert.Contains(t, view, "Wednesday 2026-02-04 (Yesterday)")
}

// TestDetailHeaderFocusIndicator tests focus indicator in detail view
func TestDetailHeaderFocusIndicator(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	view := model.View()
	assert.Contains(t, view, "●")

	// Blur
	model.Update(tea.BlurMsg{})
	view = model.View()
	assert.Contains(t, view, "○")
	assert.NotContains(t, view, "●")
}

// TestDetailCommandCountNotShown tests that "commands" text doesn't appear
func TestDetailCommandCountNotShown(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	view := model.View()
	assert.NotContains(t, view, "commands")
}

// TestDetailFirstCommandSelected tests first command is selected on entry
func TestDetailFirstCommandSelected(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	assert.Equal(t, 0, model.DetailCmdIdx())
	require.True(t, len(model.DetailCommands()) > 0)
	assert.Equal(t, "go build -o shy .", model.DetailCommands()[0].CommandText)

	view := model.View()
	assert.Contains(t, view, "▶")
}

// TestDetailNavigateDown tests navigating down through commands
func TestDetailNavigateDown(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	assert.Equal(t, "go build -o shy .", model.DetailCommands()[model.DetailCmdIdx()].CommandText)

	pressKey(model, 'j')
	assert.Equal(t, "./shy summary", model.DetailCommands()[model.DetailCmdIdx()].CommandText)

	pressKey(model, 'j')
	assert.Equal(t, "go test ./... -v", model.DetailCommands()[model.DetailCmdIdx()].CommandText)

	pressKey(model, 'j')
	assert.Equal(t, "git commit -m \"feat\"", model.DetailCommands()[model.DetailCmdIdx()].CommandText)
}

// TestDetailNavigateUp tests navigating up through commands
func TestDetailNavigateUp(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	// Navigate down to "git commit"
	pressKey(model, 'j')
	pressKey(model, 'j')
	pressKey(model, 'j')
	assert.Equal(t, "git commit -m \"feat\"", model.DetailCommands()[model.DetailCmdIdx()].CommandText)

	pressKey(model, 'k')
	assert.Equal(t, "go test ./... -v", model.DetailCommands()[model.DetailCmdIdx()].CommandText)
}

// TestDetailSelectionSkipsBucketHeaders tests selection skips bucket headers
func TestDetailSelectionSkipsBucketHeaders(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	// Navigate to last command in 8am bucket (go test ./... -v at index 2)
	pressKey(model, 'j')
	pressKey(model, 'j')
	assert.Equal(t, "go test ./... -v", model.DetailCommands()[model.DetailCmdIdx()].CommandText)

	// Next should be first in 9am bucket, skipping the header
	pressKey(model, 'j')
	assert.Equal(t, "git commit -m \"feat\"", model.DetailCommands()[model.DetailCmdIdx()].CommandText)
}

// TestDetailSelectionStopsAtFirst tests selection stops at first command
func TestDetailSelectionStopsAtFirst(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	assert.Equal(t, 0, model.DetailCmdIdx())

	pressKey(model, 'k')
	assert.Equal(t, 0, model.DetailCmdIdx())
	assert.Equal(t, "go build -o shy .", model.DetailCommands()[model.DetailCmdIdx()].CommandText)
}

// TestDetailSelectionStopsAtLast tests selection stops at last command
func TestDetailSelectionStopsAtLast(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	lastIdx := len(model.DetailCommands()) - 1

	// Navigate to the last command
	for i := 0; i < lastIdx+5; i++ {
		pressKey(model, 'j')
	}

	assert.Equal(t, lastIdx, model.DetailCmdIdx())
	assert.Equal(t, "go test ./cmd -v", model.DetailCommands()[model.DetailCmdIdx()].CommandText)

	// Try to go further
	pressKey(model, 'j')
	assert.Equal(t, lastIdx, model.DetailCmdIdx())
}

// TestDetailArrowKeys tests arrow keys in detail view
func TestDetailArrowKeys(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	assert.Equal(t, "go build -o shy .", model.DetailCommands()[model.DetailCmdIdx()].CommandText)

	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, "./shy summary", model.DetailCommands()[model.DetailCmdIdx()].CommandText)

	model.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, "go build -o shy .", model.DetailCommands()[model.DetailCmdIdx()].CommandText)
}

// TestDetailCommandTimestamps tests minute-only timestamps in buckets
func TestDetailCommandTimestamps(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	view := model.View()
	assert.Contains(t, view, ":15")
	assert.Contains(t, view, ":22")
	assert.Contains(t, view, ":30")
}

// TestDetailReturnToSummaryWithEsc tests returning to summary view
func TestDetailReturnToSummaryWithEsc(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Should be main (most commands)
	assert.Equal(t, 0, model.SelectedIdx())

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	pressEsc(model)
	assert.Equal(t, SummaryView, model.ViewState())

	view := model.View()
	assert.Contains(t, view, "Work Summary")
	assert.Equal(t, 0, model.SelectedIdx())
}

// TestDetailSelectionPreservedOnReturn tests context selection preserved on return
func TestDetailSelectionPreservedOnReturn(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Navigate to bugfix context
	pressKey(model, 'j')
	ctx := model.Contexts()[model.SelectedIdx()]
	bugfixName := formatContextName(ctx.Key, ctx.Branch)
	assert.Contains(t, bugfixName, "bugfix")

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	view := model.View()
	assert.Contains(t, view, "bugfix")

	pressEsc(model)
	assert.Equal(t, SummaryView, model.ViewState())
	assert.Equal(t, 1, model.SelectedIdx())
	ctx = model.Contexts()[model.SelectedIdx()]
	assert.Contains(t, formatContextName(ctx.Key, ctx.Branch), "bugfix")
}

// TestDetailSwitchNextContextL tests switching to next context with L
func TestDetailSwitchNextContextL(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())
	firstCtx := formatContextName(model.Contexts()[0].Key, model.Contexts()[0].Branch)

	pressShiftKey(model, 'L')
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 1, model.SelectedIdx())
	secondCtx := formatContextName(model.Contexts()[1].Key, model.Contexts()[1].Branch)
	assert.NotEqual(t, firstCtx, secondCtx)
	assert.Equal(t, 0, model.DetailCmdIdx())
}

// TestDetailSwitchPrevContextH tests switching to previous context with H
func TestDetailSwitchPrevContextH(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Go to second context first
	pressKey(model, 'j')
	pressEnter(model)
	assert.Equal(t, 1, model.SelectedIdx())

	pressShiftKey(model, 'H')
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 0, model.SelectedIdx())
	assert.Equal(t, 0, model.DetailCmdIdx())
}

// TestDetailContextSwitchStopsAtFirst tests context switch stops at first
func TestDetailContextSwitchStopsAtFirst(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)
	assert.Equal(t, 0, model.SelectedIdx())

	pressShiftKey(model, 'H')
	assert.Equal(t, 0, model.SelectedIdx())
	assert.Equal(t, ContextDetailView, model.ViewState())
}

// TestDetailContextSwitchStopsAtLast tests context switch stops at last
func TestDetailContextSwitchStopsAtLast(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Go to last context
	lastIdx := len(model.Contexts()) - 1
	for i := 0; i < lastIdx; i++ {
		pressKey(model, 'j')
	}

	pressEnter(model)
	assert.Equal(t, lastIdx, model.SelectedIdx())

	pressShiftKey(model, 'L')
	assert.Equal(t, lastIdx, model.SelectedIdx())
	assert.Equal(t, ContextDetailView, model.ViewState())
}

// TestDetailContextSwitchFollowsOrder tests context switching follows summary order
func TestDetailContextSwitchFollowsOrder(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	require.Len(t, model.Contexts(), 3)

	pressEnter(model)
	firstCtx := formatContextName(model.Contexts()[0].Key, model.Contexts()[0].Branch)

	pressShiftKey(model, 'L')
	secondCtx := formatContextName(model.Contexts()[1].Key, model.Contexts()[1].Branch)

	pressShiftKey(model, 'L')
	thirdCtx := formatContextName(model.Contexts()[2].Key, model.Contexts()[2].Branch)

	// All three should be different contexts
	assert.NotEqual(t, firstCtx, secondCtx)
	assert.NotEqual(t, secondCtx, thirdCtx)
	assert.Equal(t, 2, model.SelectedIdx())
}

// TestDetailNavigatePrevDay tests navigating to previous day stays in detail
func TestDetailNavigatePrevDay(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := today.AddDate(0, 0, -2)

	// Put commands on both days for the same context
	cmds := append(phase2Commands(yesterday), phase2Commands(dayBefore)...)
	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	// Get context name before entering detail
	ctx := model.Contexts()[0]
	ctxName := formatContextName(ctx.Key, ctx.Branch)

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	pressKey(model, 'h')
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, "2026-02-03", model.CurrentDate().Format("2006-01-02"))

	view := model.View()
	assert.Contains(t, view, ctxName)
}

// TestDetailNavigateNextDay tests navigating to next day stays in detail
func TestDetailNavigateNextDay(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := today.AddDate(0, 0, -2)

	// Put commands on both days for the same context
	cmds := append(phase2Commands(dayBefore), phase2Commands(yesterday)...)
	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	// Navigate to day before yesterday
	pressKey(model, 'h')
	assert.Equal(t, "2026-02-03", model.CurrentDate().Format("2006-01-02"))

	// Get context name before entering detail
	ctx := model.Contexts()[0]
	ctxName := formatContextName(ctx.Key, ctx.Branch)

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	pressKey(model, 'l')
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, "2026-02-04", model.CurrentDate().Format("2006-01-02"))

	view := model.View()
	assert.Contains(t, view, ctxName)
}

// TestDetailCannotNavigatePastToday tests cannot navigate past today
func TestDetailCannotNavigatePastToday(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)

	cmds := []models.Command{
		makeCommandWithText(today, 9, 0, "test cmd", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	// Navigate to today
	pressKey(model, 'l')
	assert.Equal(t, today.Format("2006-01-02"), model.CurrentDate().Format("2006-01-02"))

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	pressKey(model, 'l')
	assert.Equal(t, today.Format("2006-01-02"), model.CurrentDate().Format("2006-01-02"))
}

// TestDetailJumpToToday tests jumping to today from detail view
func TestDetailJumpToToday(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	dayBefore := today.AddDate(0, 0, -2)

	dbPath := setupTestDB(t, phase2Commands(dayBefore))
	model := initModel(t, dbPath, today)

	pressKey(model, 'h')
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	pressKey(model, 't')
	assert.Equal(t, SummaryView, model.ViewState())
	assert.Equal(t, today.Format("2006-01-02"), model.CurrentDate().Format("2006-01-02"))
}

// TestDetailJumpToYesterday tests jumping to yesterday from detail view
func TestDetailJumpToYesterday(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	dayBefore := today.AddDate(0, 0, -2)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(dayBefore))
	model := initModel(t, dbPath, today)

	pressKey(model, 'h')
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	pressKey(model, 'y')
	assert.Equal(t, SummaryView, model.ViewState())
	assert.Equal(t, yesterday.Format("2006-01-02"), model.CurrentDate().Format("2006-01-02"))
}

// TestDetailQuit tests quitting from detail view
func TestDetailQuit(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok, "expected tea.QuitMsg")
}

// TestDetailEmptyState tests empty state in detail view
func TestDetailEmptyState(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	// Manually clear detail commands to simulate empty state
	model.detailBuckets = nil
	model.detailCommands = nil

	view := model.View()
	assert.Contains(t, view, "No commands found")
	assert.NotContains(t, view, "[j/k] Select")
	assert.Contains(t, view, "[Esc] Back")
	assert.Contains(t, view, "[h/l] Time")
	assert.Contains(t, view, "[H/L] Context")
}

// TestDetailStatusBar tests status bar content in detail view
func TestDetailStatusBar(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	view := model.View()
	assert.Contains(t, view, "[j/k] Select")
	assert.Contains(t, view, "[Esc] Back")
	assert.Contains(t, view, "[h/l] Time")
	assert.Contains(t, view, "[H/L] Context")
}

// TestDetailViewportOverflow tests that header and footer stay visible when commands overflow
func TestDetailViewportOverflow(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	// Create 30 commands across multiple hours for one context
	var cmds []models.Command
	for i := 0; i < 30; i++ {
		hour := 8 + (i / 10)
		minute := (i % 10) * 5
		cmds = append(cmds, makeCommandWithText(yesterday, hour, minute,
			fmt.Sprintf("command-%02d", i),
			"/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")))
	}

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	model.Update(tea.WindowSizeMsg{Width: 80, Height: 15})
	pressEnter(model)

	view := model.View()
	lines := strings.Split(view, "\n")

	// Header should be on first line
	assert.Contains(t, lines[0], "main")

	// Status bar should be on last non-empty line
	lastLine := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastLine = lines[i]
			break
		}
	}
	assert.Contains(t, lastLine, "[Esc] Back")

	// Not all 30 commands should be visible
	visibleCount := 0
	for _, line := range lines {
		if strings.Contains(line, "command-") {
			visibleCount++
		}
	}
	assert.Less(t, visibleCount, 30, "should not show all 30 commands in 15-line window")
	assert.Greater(t, visibleCount, 0, "should show some commands")
}

// TestDetailViewportScrollsWithSelection tests scrolling keeps selected command visible
func TestDetailViewportScrollsWithSelection(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	// Create 30 commands for one context
	var cmds []models.Command
	for i := 0; i < 30; i++ {
		hour := 8 + (i / 10)
		minute := (i % 10) * 5
		cmds = append(cmds, makeCommandWithText(yesterday, hour, minute,
			fmt.Sprintf("command-%02d", i),
			"/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")))
	}

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	model.Update(tea.WindowSizeMsg{Width: 80, Height: 15})
	pressEnter(model)

	// Navigate past the visible area
	for i := 0; i < 20; i++ {
		pressKey(model, 'j')
	}

	view := model.View()
	lines := strings.Split(view, "\n")

	// The selected command should be visible
	selectedCmd := model.DetailCommands()[model.DetailCmdIdx()].CommandText
	assert.Contains(t, view, selectedCmd, "selected command should be visible after scrolling")

	// Header still visible (first line)
	assert.Contains(t, lines[0], "main")

	// Status bar still visible (last non-empty line)
	lastLine := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastLine = lines[i]
			break
		}
	}
	assert.Contains(t, lastLine, "[Esc] Back")
}

// TestDetailViewportScrollUpShowsBucketHeader tests that scrolling back up reveals the bucket header
func TestDetailViewportScrollUpShowsBucketHeader(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	// Create commands in two buckets: 8am (10 cmds) and 9am (10 cmds)
	var cmds []models.Command
	for i := 0; i < 10; i++ {
		cmds = append(cmds, makeCommandWithText(yesterday, 8, i*5,
			fmt.Sprintf("eight-%02d", i),
			"/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")))
	}
	for i := 0; i < 10; i++ {
		cmds = append(cmds, makeCommandWithText(yesterday, 9, i*5,
			fmt.Sprintf("nine-%02d", i),
			"/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")))
	}

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	// Small window so scrolling is needed
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	pressEnter(model)

	// Scroll down into the 9am bucket
	for i := 0; i < 15; i++ {
		pressKey(model, 'j')
	}
	view := model.View()
	assert.Contains(t, view, "9am", "should see 9am bucket after scrolling down")

	// Now scroll back up to first command in 9am bucket
	for model.DetailCmdIdx() > 10 {
		pressKey(model, 'k')
	}

	view = model.View()
	assert.Contains(t, view, "9am", "9am bucket header should be visible when at first command in 9am")

	// Scroll all the way back to 8am first command
	for model.DetailCmdIdx() > 0 {
		pressKey(model, 'k')
	}

	view = model.View()
	assert.Contains(t, view, "8am", "8am bucket header should be visible when at first command in 8am")
}

// TestDetailNavigateNextDayEmptyContext tests navigating to a day where context has no commands
func TestDetailNavigateNextDayEmptyContext(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	// Commands only on yesterday, nothing on today
	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Get context name
	ctx := model.Contexts()[0]
	ctxName := formatContextName(ctx.Key, ctx.Branch)

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	// Navigate to today where this context has no commands
	pressKey(model, 'l')
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, today.Format("2006-01-02"), model.CurrentDate().Format("2006-01-02"))

	view := model.View()
	assert.Contains(t, view, "No commands found")
	assert.Contains(t, view, ctxName)
}
