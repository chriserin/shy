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
	assert.Contains(t, view, "Feb 4")
	assert.Contains(t, view, "◆")
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
	assert.Contains(t, view, "Feb 3")
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
	assert.Contains(t, view, "Feb 4")
	assert.Contains(t, view, "◆")
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
	assert.Contains(t, view, "Feb 5")
	assert.Contains(t, view, "★")
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
	assert.Contains(t, view, "Feb 5")
	assert.Contains(t, view, "★")
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
	assert.Contains(t, view, "Feb 4")
	assert.Contains(t, view, "◆")
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
	assert.Contains(t, view, "Feb 4")
	assert.Contains(t, view, "◆")
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
	assert.Contains(t, view, "Feb 1")
	// Header should not contain relative date labels
	header := strings.Split(view, "\n")[0]
	assert.NotContains(t, header, "◆")
	assert.NotContains(t, header, "★")
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
	assert.Contains(t, view, "Feb 4")
	assert.Contains(t, view, "◆")

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
	assert.Contains(t, view, "◆ Wednesday Feb 4")
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

// TestDetailReturnToSummaryWithDash tests returning to summary view with '-'
func TestDetailReturnToSummaryWithDash(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Should be main (most commands)
	assert.Equal(t, 0, model.SelectedIdx())

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	pressKey(model, '-')
	assert.Equal(t, SummaryView, model.ViewState())

	view := model.View()
	assert.NotContains(t, view, "Work Summary")
	assert.Contains(t, view, "●")
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

	pressKey(model, '-')
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
	assert.NotContains(t, view, "[j/k]")
	assert.NotContains(t, view, "[Esc]")
}

// TestDetailStatusBar tests footer bar content in detail view
func TestDetailStatusBar(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase2Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	view := model.View()
	// Footer shows mode indicator, no key hints
	assert.Contains(t, view, "All")
	assert.NotContains(t, view, "[j/k]")
	assert.NotContains(t, view, "[Esc]")
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

	// Footer bar should be on last non-empty line
	lastLine := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastLine = lines[i]
			break
		}
	}
	assert.Contains(t, lastLine, "All")

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

	// Footer bar still visible (last non-empty line)
	lastLine := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastLine = lines[i]
			break
		}
	}
	assert.Contains(t, lastLine, "All")
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

// === Phase 3 Test Helpers ===

// phase3Commands returns the background data set from the phase 3 feat file
func phase3Commands(date time.Time) []models.Command {
	return []models.Command{
		// ~/projects/shy:main — 8 commands (3x "go build", 2x "go test ./... -v", 1x "./shy summary", 1x "git commit", 1x "go test ./cmd -v")
		makeCommandWithText(date, 8, 15, "go build -o shy .", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 8, 22, "./shy summary", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 8, 30, "go test ./... -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 9, 0, "go build -o shy .", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 9, 5, `git commit -m "feat"`, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 9, 15, "go build -o shy .", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 9, 30, "go test ./... -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(date, 14, 20, "go test ./cmd -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		// ~/downloads — 2 commands (both unique)
		makeCommandWithText(date, 8, 0, "curl -O example.com", "/home/user/downloads", nil, nil),
		makeCommandWithText(date, 8, 10, "tar xzf archive.gz", "/home/user/downloads", nil, nil),
	}
}

// === Phase 3 Tests ===

// TestDefaultAllModeShowsTotalCounts tests default mode shows total command counts
func TestDefaultAllModeShowsTotalCounts(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	assert.Equal(t, AllMode, model.DisplayMode())
	view := model.View()
	assert.Contains(t, view, "8 commands")
	assert.Contains(t, view, "2 commands")
}

// TestUniqueModeShowsUniqueCounts tests unique mode shows count of once-only commands
func TestUniqueModeShowsUniqueCounts(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressKey(model, 'u')

	assert.Equal(t, UniqueMode, model.DisplayMode())
	view := model.View()
	assert.Contains(t, view, "3 commands")
	assert.Contains(t, view, "2 commands")
}

// TestAllModeReturnsTotalCounts tests pressing 'a' returns to all mode
func TestAllModeReturnsTotalCounts(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressKey(model, 'u')
	assert.Equal(t, UniqueMode, model.DisplayMode())

	pressKey(model, 'a')
	assert.Equal(t, AllMode, model.DisplayMode())
	view := model.View()
	assert.Contains(t, view, "8 commands")
	assert.Contains(t, view, "2 commands")
}

// TestPressingUWhileInUniqueModeStays tests pressing 'u' while already in unique stays
func TestPressingUWhileInUniqueModeStays(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressKey(model, 'u')
	assert.Equal(t, UniqueMode, model.DisplayMode())

	pressKey(model, 'u')
	assert.Equal(t, UniqueMode, model.DisplayMode())
}

// TestModePersistsEnterLeaveDetail tests mode persists when entering and leaving detail
func TestModePersistsEnterLeaveDetail(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressKey(model, 'u')
	assert.Equal(t, UniqueMode, model.DisplayMode())

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, UniqueMode, model.DisplayMode())

	pressKey(model, '-')
	assert.Equal(t, SummaryView, model.ViewState())
	assert.Equal(t, UniqueMode, model.DisplayMode())
	view := model.View()
	assert.Contains(t, view, "3 commands")
}

// TestModePersistsAcrossDateNav tests mode persists across date navigation
func TestModePersistsAcrossDateNav(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressKey(model, 'u')
	assert.Equal(t, UniqueMode, model.DisplayMode())

	pressKey(model, 'h')
	assert.Equal(t, UniqueMode, model.DisplayMode())
}

// TestUniqueModeFiltersInDetail tests unique mode filters commands in detail view
func TestUniqueModeFiltersInDetail(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Enter detail for shy:main (first context, most commands)
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	pressKey(model, 'u')
	assert.Equal(t, UniqueMode, model.DisplayMode())

	assert.Equal(t, 3, len(model.DetailCommands()))
	assert.Equal(t, 3, len(model.DetailBuckets()))

	view := model.View()
	assert.Contains(t, view, "8am")
	assert.Contains(t, view, "9am")
	assert.Contains(t, view, "2pm")
	assert.Contains(t, view, "./shy summary")
	assert.Contains(t, view, `git commit`)
	assert.Contains(t, view, "go test ./cmd -v")
	assert.NotContains(t, view, "go build")
}

// TestChangingModeResetsSelection tests changing mode resets selection to first
func TestChangingModeResetsSelection(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	// Navigate to 5th command (index 4)
	for i := 0; i < 4; i++ {
		pressKey(model, 'j')
	}
	assert.Equal(t, 4, model.DetailCmdIdx())

	pressKey(model, 'u')
	assert.Equal(t, 0, model.DetailCmdIdx())
}

// TestStatusBarShowsModeIndicator tests footer bar shows active display mode
func TestStatusBarShowsModeIndicator(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	view := model.View()
	// Default mode is All
	assert.Contains(t, view, "All")
}

// TestStatusBarShowsActiveMode tests footer bar shows switched mode
func TestStatusBarShowsActiveMode(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressKey(model, 'u')

	view := model.View()
	assert.Contains(t, view, "Uniq")
}

// TestStatusBarModeInDetail tests footer bar shows mode in detail view
func TestStatusBarModeInDetail(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)

	view := model.View()
	assert.Contains(t, view, "All")
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
	assert.Contains(t, view, "No commands found in")
	assert.Contains(t, view, ctxName)
}

// === Phase 4a Test Helpers ===

func int64Ptr(v int64) *int64 {
	return &v
}

// makeCommandFull creates a command with full metadata including pid and duration
func makeCommandFull(date time.Time, hour, minute int, text, workingDir string, gitRepo, gitBranch *string, exitStatus int, durationMs *int64, pid *int64) models.Command {
	timestamp := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.Local)
	app := "zsh"
	return models.Command{
		CommandText: text,
		WorkingDir:  workingDir,
		GitRepo:     gitRepo,
		GitBranch:   gitBranch,
		ExitStatus:  exitStatus,
		Timestamp:   timestamp.Unix(),
		Duration:    durationMs,
		SourcePid:   pid,
		SourceApp:   &app,
	}
}

// phase4aCommands returns commands with full metadata for phase 4a tests
func phase4aCommands(date time.Time) []models.Command {
	pid := int64Ptr(10001)
	return []models.Command{
		makeCommandFull(date, 8, 15, "go build -o shy .", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(1200), pid),
		makeCommandFull(date, 8, 22, "./shy summary", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(250), pid),
		makeCommandFull(date, 8, 30, "go test ./... -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 1, int64Ptr(8200), pid),
		makeCommandFull(date, 9, 0, `git commit -m "feat: add summary"`, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(500), pid),
		makeCommandFull(date, 9, 5, "go build -o shy .", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(1050), pid),
		makeCommandFull(date, 9, 30, "git push", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(2100), pid),
		makeCommandFull(date, 14, 20, "shy summary", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(310), pid),
		makeCommandFull(date, 14, 25, "go test ./cmd -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(3200), pid),
	}
}

// enterCommandDetailDirect simulates entering command detail view by directly setting state
// This avoids needing DB access for GetCommandWithContext
func enterCommandDetailDirect(model *Model, target *models.Command, before, after []models.Command) {
	var all []models.Command
	all = append(all, before...)
	if target != nil {
		all = append(all, *target)
	}
	all = append(all, after...)
	model.cmdDetailAll = all
	model.cmdDetailIdx = len(before) // point at target
	model.cmdDetailStartIdx = model.cmdDetailIdx
	model.viewState = CommandDetailView
}

// === Phase 4a Tests ===

// TestCmdDetailEnterFromContextDetail tests entering command detail view via Enter
func TestCmdDetailEnterFromContextDetail(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase4aCommands(yesterday))
	model := initModel(t, dbPath, today)

	// Enter context detail view
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	// Press Enter on first command to enter command detail
	pressEnter(model)
	assert.Equal(t, CommandDetailView, model.ViewState())

	view := model.View()
	assert.Contains(t, view, "Event:")
}

// TestCmdDetailShowsMetadata tests that command detail view shows metadata fields
func TestCmdDetailShowsMetadata(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday)
	target := cmds[0] // "go build -o shy ." at 8:15
	target.ID = 1

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	enterCommandDetailDirect(model, &target, nil, cmds[1:3])

	view := model.View()
	assert.Contains(t, view, "Command:")
	assert.Contains(t, view, "go build -o shy .")
	assert.Contains(t, view, "Timestamp:")
	assert.Contains(t, view, "Working Dir:")
	assert.Contains(t, view, "Duration:")
	assert.Contains(t, view, "Git Repo:")
	assert.Contains(t, view, "github.com/chris/shy")
	assert.Contains(t, view, "Git Branch:")
	assert.Contains(t, view, "main")
}

// TestCmdDetailExitStatusSuccess tests success indicator
func TestCmdDetailExitStatusSuccess(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday)
	target := cmds[0] // exit status 0
	target.ID = 1

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	enterCommandDetailDirect(model, &target, nil, nil)

	view := model.View()
	assert.Contains(t, view, "0 \u2713")
}

// TestCmdDetailExitStatusFailure tests failure indicator
func TestCmdDetailExitStatusFailure(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday)
	target := cmds[2] // "go test ./... -v" with exit status 1
	target.ID = 3

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	enterCommandDetailDirect(model, &target, nil, nil)

	view := model.View()
	assert.Contains(t, view, "1 \u2717")
}

// TestCmdDetailDurationHumanReadable tests human-readable duration
func TestCmdDetailDurationHumanReadable(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday)
	target := cmds[2] // duration 8200ms = 8s
	target.ID = 3

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	enterCommandDetailDirect(model, &target, nil, nil)

	view := model.View()
	assert.Contains(t, view, "Duration:")
	assert.Contains(t, view, "8s")
}

// TestCmdDetailMissingDuration tests missing duration shows em dash
func TestCmdDetailMissingDuration(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	target := makeCommandWithText(yesterday, 9, 0, "test cmd", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"))
	target.ID = 1
	target.Duration = nil

	cmds := phase4aCommands(yesterday)
	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	enterCommandDetailDirect(model, &target, nil, nil)

	view := model.View()
	assert.Contains(t, view, "Duration:")
	assert.Contains(t, view, "\u2014")
}

// TestCmdDetailSessionContext tests session context display
func TestCmdDetailSessionContext(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday)
	target := cmds[3] // git commit
	target.ID = 4
	before := []models.Command{cmds[1], cmds[2]}
	before[0].ID = 2
	before[1].ID = 3
	after := []models.Command{cmds[4]}
	after[0].ID = 5

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	enterCommandDetailDirect(model, &target, before, after)

	view := model.View()
	assert.Contains(t, view, "Context (same session):")
	assert.Contains(t, view, "./shy summary")
	assert.Contains(t, view, "go test ./... -v")
	assert.Contains(t, view, `git commit -m "feat: add summary"`)
	assert.Contains(t, view, "go build -o shy .")
}

// TestCmdDetailCurrentCommandHighlighted tests the target is highlighted
func TestCmdDetailCurrentCommandHighlighted(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday)
	target := cmds[3]
	target.ID = 4
	before := []models.Command{cmds[2]}
	before[0].ID = 3
	after := []models.Command{cmds[4]}
	after[0].ID = 5

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	enterCommandDetailDirect(model, &target, before, after)

	view := model.View()
	assert.Contains(t, view, "▶")
}

// TestCmdDetailNavigateDown tests navigating down in command detail
func TestCmdDetailNavigateDown(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday)
	target := cmds[3] // git commit at index 3
	target.ID = 4
	before := []models.Command{cmds[2]}
	before[0].ID = 3
	after := []models.Command{cmds[4]}
	after[0].ID = 5

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	enterCommandDetailDirect(model, &target, before, after)

	assert.Equal(t, `git commit -m "feat: add summary"`, model.CmdDetailTarget().CommandText)

	// j reloads context centered on the next command
	pressKey(model, 'j')
	assert.Equal(t, "go build -o shy .", model.CmdDetailTarget().CommandText)
}

// TestCmdDetailNavigateUp tests navigating up in command detail
func TestCmdDetailNavigateUp(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday)
	target := cmds[3]
	target.ID = 4
	before := []models.Command{cmds[2]}
	before[0].ID = 3
	after := []models.Command{cmds[4]}
	after[0].ID = 5

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	enterCommandDetailDirect(model, &target, before, after)

	assert.Equal(t, `git commit -m "feat: add summary"`, model.CmdDetailTarget().CommandText)

	// k reloads context centered on the previous command
	pressKey(model, 'k')
	assert.Equal(t, "go test ./... -v", model.CmdDetailTarget().CommandText)
}

// TestCmdDetailNavigationStopsAtBoundary tests navigation stops at session boundaries
func TestCmdDetailNavigationStopsAtBoundary(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday) // 8 commands, same PID
	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	// Enter via the full flow so context is loaded from DB
	pressEnter(model) // → ContextDetailView
	assert.Equal(t, ContextDetailView, model.ViewState())
	pressEnter(model) // → CommandDetailView on first command
	assert.Equal(t, CommandDetailView, model.ViewState())

	firstCmdText := cmds[0].CommandText
	assert.Equal(t, firstCmdText, model.CmdDetailTarget().CommandText)

	// Try to go up past the session start — should stay on first command
	pressKey(model, 'k')
	assert.Equal(t, firstCmdText, model.CmdDetailTarget().CommandText)

	// Navigate all the way to the last command
	lastCmdText := cmds[len(cmds)-1].CommandText
	for i := 0; i < len(cmds); i++ {
		pressKey(model, 'j')
	}
	assert.Equal(t, lastCmdText, model.CmdDetailTarget().CommandText)

	// Try to go down past the session end — should stay on last command
	pressKey(model, 'j')
	assert.Equal(t, lastCmdText, model.CmdDetailTarget().CommandText)
}

// TestCmdDetailNavigateAllCommands walks through every command in the session
// via j/k navigation, verifying the target command text at each position.
func TestCmdDetailNavigateAllCommands(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday) // 8 commands, same PID
	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	allTexts := make([]string, len(cmds))
	for i, c := range cmds {
		allTexts[i] = c.CommandText
	}

	// Enter command detail on the 3rd command (ID 3)
	target := cmds[2]
	target.ID = 3
	before := []models.Command{cmds[0], cmds[1]}
	before[0].ID = 1
	before[1].ID = 2
	after := []models.Command{cmds[3], cmds[4]}
	after[0].ID = 4
	after[1].ID = 5
	enterCommandDetailDirect(model, &target, before, after)

	assert.Equal(t, allTexts[2], model.CmdDetailTarget().CommandText)

	// Navigate up to the very first command
	pressKey(model, 'k')
	assert.Equal(t, allTexts[1], model.CmdDetailTarget().CommandText)

	pressKey(model, 'k')
	assert.Equal(t, allTexts[0], model.CmdDetailTarget().CommandText)

	// Clamped at top
	pressKey(model, 'k')
	assert.Equal(t, allTexts[0], model.CmdDetailTarget().CommandText)

	// Navigate down through every command in the session to the last
	for i := 1; i < len(allTexts); i++ {
		pressKey(model, 'j')
		assert.Equal(t, allTexts[i], model.CmdDetailTarget().CommandText, "command text after pressing j %d times", i)
	}

	// Clamped at bottom
	pressKey(model, 'j')
	assert.Equal(t, allTexts[len(allTexts)-1], model.CmdDetailTarget().CommandText)

	// Verify the view renders the currently selected command's metadata
	view := model.View()
	assert.Contains(t, view, "Event:")
	assert.Contains(t, view, allTexts[len(allTexts)-1])
}

// TestCmdDetailReturnWithDash tests returning to context detail with '-'
func TestCmdDetailReturnWithDash(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase4aCommands(yesterday))
	model := initModel(t, dbPath, today)

	// Enter context detail
	pressEnter(model)

	// Enter command detail
	target := model.DetailCommands()[0]
	enterCommandDetailDirect(model, &target, nil, nil)
	assert.Equal(t, CommandDetailView, model.ViewState())

	// Press '-' to return
	pressKey(model, '-')
	assert.Equal(t, ContextDetailView, model.ViewState())
}

// TestCmdDetailQuit tests quitting from command detail view
func TestCmdDetailQuit(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday)
	target := cmds[0]
	target.ID = 1

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	enterCommandDetailDirect(model, &target, nil, nil)

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok, "expected tea.QuitMsg")
}

// TestCmdDetailStatusBar tests footer bar content in command detail view
func TestCmdDetailStatusBar(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday)
	target := cmds[0]
	target.ID = 1

	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	enterCommandDetailDirect(model, &target, nil, nil)

	view := model.View()
	// Footer bar has no key hints in command detail view
	assert.NotContains(t, view, "[j/k]")
	assert.NotContains(t, view, "[Esc]")
}

// TestCmdDetailContextFillsHeight tests that total context scales with terminal height
func TestCmdDetailContextFillsHeight(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday)
	dbPath := setupTestDB(t, cmds)

	tests := []struct {
		name      string
		height    int
		wantTotal int
	}{
		{"tall terminal", 40, 24},      // 40 - 16 = 24
		{"short terminal", 20, 4},      // 20 - 16 = 4
		{"medium terminal", 30, 14},    // 30 - 16 = 14
		{"very short terminal", 15, 1}, // 15 - 16 < 1 → 1
		{"zero height fallback", 0, 10}, // fallback
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := initModel(t, dbPath, today)
			model.Update(tea.WindowSizeMsg{Width: 80, Height: tt.height})

			assert.Equal(t, tt.wantTotal, model.CmdDetailTotalContext())
		})
	}
}

// TestCmdDetailContextCountMatchesHeight tests that the actual number of context
// commands loaded matches what fits the terminal height
func TestCmdDetailContextCountMatchesHeight(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	// All 8 commands share the same PID, so context lookups return session neighbors
	cmds := phase4aCommands(yesterday)
	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	// Set height to 40 → totalContext = 24 → but only 8 cmds in DB
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

	// Enter context detail, then command detail via the full flow
	pressEnter(model) // → ContextDetailView
	assert.Equal(t, ContextDetailView, model.ViewState())

	pressEnter(model) // → CommandDetailView (first command)
	assert.Equal(t, CommandDetailView, model.ViewState())

	// With 8 commands in DB and target at index 0, we expect:
	// 0 before + 1 target + 7 after = 8 total (limited by DB, not totalContext)
	allCmds := model.CmdDetailAll()
	assert.Equal(t, len(cmds), len(allCmds))

	// Now test with height 20 → totalContext = 4 → at most 4 context + 1 target = 5
	model2 := initModel(t, dbPath, today)
	model2.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	pressEnter(model2) // → ContextDetailView
	pressEnter(model2) // → CommandDetailView

	allCmds2 := model2.CmdDetailAll()
	// totalContext=4, target is first cmd (index 0), so 0 before + 1 target + 4 after = 5
	assert.LessOrEqual(t, len(allCmds2), 5)
	assert.GreaterOrEqual(t, len(allCmds2), 1) // at least the target
}

// TestCmdDetailContextBalancesWhenNearEdge tests that when near the start/end
// of a session, surplus context from the short side fills the other side
func TestCmdDetailContextBalancesWhenNearEdge(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday) // 8 commands, same PID
	dbPath := setupTestDB(t, cmds)

	// height 30 → totalContext = 14 → balanced would be 7 before + 7 after
	// But at the first command there are 0 before, so all 14 should go to after
	model := initModel(t, dbPath, today)
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	pressEnter(model)
	pressEnter(model) // first command

	allCmds := model.CmdDetailAll()
	// 0 before + 1 target + 7 after = 8 (limited by 8 cmds in DB, not 12)
	assert.Equal(t, len(cmds), len(allCmds), "near start: should show all 8 commands")
	assert.Equal(t, cmds[0].CommandText, model.CmdDetailTarget().CommandText)

	// Navigate to the last command
	for i := 0; i < len(cmds)-1; i++ {
		pressKey(model, 'j')
	}
	assert.Equal(t, cmds[len(cmds)-1].CommandText, model.CmdDetailTarget().CommandText)

	// At the last command: 0 after, so all 14 should go to before
	allCmds = model.CmdDetailAll()
	assert.Equal(t, len(cmds), len(allCmds), "near end: should show all 8 commands")
}

// TestCmdDetailResizeReloadsContext tests that resizing the terminal reloads
// the context to fill the new available space
func TestCmdDetailResizeReloadsContext(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	cmds := phase4aCommands(yesterday) // 8 commands, same PID
	dbPath := setupTestDB(t, cmds)
	model := initModel(t, dbPath, today)

	// Start small: height 18 → totalContext = 2
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 18})
	pressEnter(model) // → ContextDetailView
	pressEnter(model) // → CommandDetailView on first command

	assert.Equal(t, CommandDetailView, model.ViewState())
	smallCount := len(model.CmdDetailAll())
	// totalContext=2, target at start → 0 before + 1 target + 2 after = 3
	assert.LessOrEqual(t, smallCount, 3)

	// Resize larger: height 40 → totalContext = 24
	_, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	assert.NotNil(t, cmd, "resize in CommandDetailView should trigger reload")
	msg := cmd()
	model.Update(msg)

	assert.Equal(t, CommandDetailView, model.ViewState())
	largeCount := len(model.CmdDetailAll())
	// All 8 commands fit within totalContext=24
	assert.Equal(t, len(cmds), largeCount)
	assert.Greater(t, largeCount, smallCount)
}

// === Phase 4b Test Helpers ===

// typeString sends each rune as a KeyMsg to simulate typing
func typeString(model *Model, s string) {
	for _, r := range s {
		model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

// pressBackspace simulates pressing backspace
func pressBackspace(model *Model) {
	model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
}

// pressSlash simulates pressing '/' to open filter
func pressSlash(model *Model) {
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
}

// === Phase 4b Tests ===

// TestFilterOpenWithSlash tests opening filter with '/'
func TestFilterOpenWithSlash(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	assert.False(t, model.FilterActive())

	pressSlash(model)
	assert.True(t, model.FilterActive())

	view := model.View()
	assert.Contains(t, view, "Filter:")
}

// TestFilterTypingUpdatesText tests typing updates filter text
func TestFilterTypingUpdatesText(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressSlash(model)
	typeString(model, "go test")

	assert.Equal(t, "go test", model.FilterText())
	view := model.View()
	assert.Contains(t, view, "Filter: go test")
}

// TestFilterLiveApplicationSummary tests filter applies live in summary view
func TestFilterLiveApplicationSummary(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressSlash(model)
	typeString(model, "go build")

	view := model.View()
	// shy:main has 3x "go build" commands
	assert.Contains(t, view, "3 commands")
	// downloads has 0 matching
	assert.Contains(t, view, "0 commands")
}

// TestFilterLiveApplicationDetail tests filter applies live in detail view
func TestFilterLiveApplicationDetail(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Enter detail for shy:main
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 8, len(model.DetailCommands()))

	// Open filter and type
	pressSlash(model)
	typeString(model, "go build")

	// Should show only "go build" commands
	assert.Equal(t, 3, len(model.DetailCommands()))
}

// TestFilterSubmitWithEnter tests submitting filter with Enter
func TestFilterSubmitWithEnter(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressSlash(model)
	typeString(model, "git")
	assert.True(t, model.FilterActive())

	pressEnter(model)
	assert.False(t, model.FilterActive())
	assert.Equal(t, "git", model.FilterText())
}

// TestFilterCancelWithEscRestoresPrevious tests Esc restores previous filter
func TestFilterCancelWithEscRestoresPrevious(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Set initial filter
	pressSlash(model)
	typeString(model, "build")
	pressEnter(model)
	assert.Equal(t, "build", model.FilterText())

	// Open again and type something different
	pressSlash(model)
	typeString(model, "git")
	assert.Equal(t, "buildgit", model.FilterText()) // appended

	// Cancel with Esc — should restore "build"
	pressEsc(model)
	assert.False(t, model.FilterActive())
	assert.Equal(t, "build", model.FilterText())
}

// TestFilterCancelWithEscNoPrior tests Esc with no prior filter clears all
func TestFilterCancelWithEscNoPrior(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressSlash(model)
	typeString(model, "git")

	pressEsc(model)
	assert.False(t, model.FilterActive())
	assert.Equal(t, "", model.FilterText())
}

// TestFilterClearWithEmptySubmit tests clearing filter by submitting empty
func TestFilterClearWithEmptySubmit(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Set filter
	pressSlash(model)
	typeString(model, "go test")
	pressEnter(model)
	assert.Equal(t, "go test", model.FilterText())

	// Open again, clear text, submit
	pressSlash(model)
	// Remove all chars with backspace
	for i := 0; i < len("go test"); i++ {
		pressBackspace(model)
	}
	pressEnter(model)
	assert.Equal(t, "", model.FilterText())

	view := model.View()
	assert.Contains(t, view, "8 commands")
}

// TestFilterBackspaceRemovesChars tests backspace removes characters
func TestFilterBackspaceRemovesChars(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressSlash(model)
	typeString(model, "go test")
	assert.Equal(t, "go test", model.FilterText())

	pressBackspace(model)
	assert.Equal(t, "go tes", model.FilterText())
}

// TestFilterBackspaceOnEmptyCloses tests backspace on empty closes filter
func TestFilterBackspaceOnEmptyCloses(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressSlash(model)
	assert.True(t, model.FilterActive())
	assert.Equal(t, "", model.FilterText())

	pressBackspace(model)
	assert.False(t, model.FilterActive())
}

// TestFilterAndUniqueModeCombine tests filter with unique mode
func TestFilterAndUniqueModeCombine(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Set filter to "go"
	pressSlash(model)
	typeString(model, "go")
	pressEnter(model)

	// Set unique mode
	pressKey(model, 'u')

	view := model.View()
	// "go test ./cmd -v" is the only unique "go" command (count=1)
	// "go build" (3x) and "go test ./... -v" (2x) excluded
	assert.Contains(t, view, "1 command")
}

// TestFilterPersistsAcrossViews tests filter persists when entering/leaving detail
func TestFilterPersistsAcrossViews(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Set filter
	pressSlash(model)
	typeString(model, "build")
	pressEnter(model)
	assert.Equal(t, "build", model.FilterText())

	// Enter detail view
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, "build", model.FilterText())

	// Should only show "go build" commands
	assert.Equal(t, 3, len(model.DetailCommands()))

	// Return
	pressKey(model, '-')
	assert.Equal(t, "build", model.FilterText())
}

// TestFilterPersistsAcrossDates tests filter persists when navigating dates
func TestFilterPersistsAcrossDates(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Set filter
	pressSlash(model)
	typeString(model, "git")
	pressEnter(model)
	assert.Equal(t, "git", model.FilterText())

	pressKey(model, 'h')
	assert.Equal(t, "git", model.FilterText())
}

// TestFilterIndicatorInStatusBar tests filter indicator in status bar
func TestFilterIndicatorInStatusBar(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	// Set filter
	pressSlash(model)
	typeString(model, "go test")
	pressEnter(model)

	view := model.View()
	assert.Contains(t, view, "/go test")
}

// TestFilterSlashInDetailView tests opening filter in detail view
func TestFilterSlashInDetailView(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	pressSlash(model)
	assert.True(t, model.FilterActive())
}

// TestFilterQuitWhileFilterOpen tests quitting while filter is open
func TestFilterQuitWhileFilterOpen(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, phase3Commands(yesterday))
	model := initModel(t, dbPath, today)

	pressSlash(model)
	assert.True(t, model.FilterActive())

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok, "expected tea.QuitMsg")
}

// === Phase 4c Test Helpers ===

// phase4Commands returns multi-day data matching the feat file background
func phase4Commands() []models.Command {
	pid := int64Ptr(10001)
	pid2 := int64Ptr(10002)
	pid3 := int64Ptr(10003)

	mon := time.Date(2026, 2, 3, 0, 0, 0, 0, time.Local) // Monday
	tue := time.Date(2026, 2, 4, 0, 0, 0, 0, time.Local) // Tuesday
	wed := time.Date(2026, 2, 5, 0, 0, 0, 0, time.Local) // Wednesday

	return []models.Command{
		// Mon Feb 3 - ~/projects/shy:main (3 commands)
		makeCommandFull(mon, 9, 15, "go build -o shy .", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(1200), pid),
		makeCommandFull(mon, 9, 30, "go test ./... -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(4500), pid),
		makeCommandFull(mon, 14, 20, "shy summary", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(300), pid),

		// Tue Feb 4 - ~/projects/shy:main (8 commands)
		makeCommandFull(tue, 8, 15, "go build -o shy .", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(1100), pid),
		makeCommandFull(tue, 8, 22, "./shy summary", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(250), pid),
		makeCommandFull(tue, 8, 30, "go test ./... -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 1, int64Ptr(8200), pid),
		makeCommandFull(tue, 9, 0, `git commit -m "feat: add summary"`, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(500), pid),
		makeCommandFull(tue, 9, 5, "go build -o shy .", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(1050), pid),
		makeCommandFull(tue, 9, 30, "git push", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(2100), pid),
		makeCommandFull(tue, 14, 20, "shy summary", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(310), pid),
		makeCommandFull(tue, 14, 25, "go test ./cmd -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(3200), pid),

		// Tue Feb 4 - ~/projects/shy:bugfix (3 commands)
		makeCommandFull(tue, 8, 0, "git checkout bugfix", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix"), 0, int64Ptr(150), pid2),
		makeCommandFull(tue, 8, 10, "go test ./... -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix"), 0, int64Ptr(5100), pid2),
		makeCommandFull(tue, 8, 45, "git diff", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix"), 0, int64Ptr(80), pid2),

		// Tue Feb 4 - ~/downloads (2 commands)
		makeCommandFull(tue, 10, 0, "curl -O https://example.com/archive.tar.gz", "/home/user/downloads", nil, nil, 0, int64Ptr(15000), pid3),
		makeCommandFull(tue, 10, 5, "tar xzf archive.tar.gz", "/home/user/downloads", nil, nil, 0, int64Ptr(900), pid3),

		// Wed Feb 5 - ~/projects/shy:main (2 commands)
		makeCommandFull(wed, 10, 0, "go build -o shy .", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(1000), pid),
		makeCommandFull(wed, 10, 30, "go test ./... -v", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main"), 0, int64Ptr(4000), pid),
	}
}

// pressBracketRight presses ']'
func pressBracketRight(model *Model) {
	pressKey(model, ']')
}

// pressBracketLeft presses '['
func pressBracketLeft(model *Model) {
	pressKey(model, '[')
}

// === Phase 4c Tests ===

// TestDefaultPeriodIsDay tests the default period is Day
func TestDefaultPeriodIsDay(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	assert.Equal(t, DayPeriod, model.Period())
	view := model.View()
	assert.Contains(t, view, "Day")
}

// TestPeriodCycleDayToWeek tests ] cycles Day to Week
func TestPeriodCycleDayToWeek(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	assert.Equal(t, DayPeriod, model.Period())

	pressBracketRight(model)
	assert.Equal(t, WeekPeriod, model.Period())
	view := model.View()
	assert.Contains(t, view, "Week")
	assert.Contains(t, view, "Week of")
}

// TestPeriodCycleWeekToMonth tests ] cycles Week to Month
func TestPeriodCycleWeekToMonth(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week
	pressBracketRight(model) // Week → Month
	assert.Equal(t, MonthPeriod, model.Period())
	view := model.View()
	assert.Contains(t, view, "Month")
	assert.Contains(t, view, "February 2026")
}

// TestPeriodCycleMonthClamped tests ] at Month does nothing
func TestPeriodCycleMonthClamped(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week
	pressBracketRight(model) // Week → Month
	pressBracketRight(model) // Month → still Month
	assert.Equal(t, MonthPeriod, model.Period())
}

// TestPeriodCycleMonthToWeek tests [ cycles Month to Week
func TestPeriodCycleMonthToWeek(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week
	pressBracketRight(model) // Week → Month
	pressBracketLeft(model)  // Month → Week
	assert.Equal(t, WeekPeriod, model.Period())
}

// TestPeriodCycleWeekToDay tests [ cycles Week to Day
func TestPeriodCycleWeekToDay(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week
	pressBracketLeft(model)  // Week → Day
	assert.Equal(t, DayPeriod, model.Period())
}

// TestPeriodCycleDayClamped tests [ at Day does nothing
func TestPeriodCycleDayClamped(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketLeft(model) // Day → still Day
	assert.Equal(t, DayPeriod, model.Period())
}

// TestWeekViewAggregatedCounts tests week view shows aggregated command counts
func TestWeekViewAggregatedCounts(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week
	assert.Equal(t, WeekPeriod, model.Period())

	view := model.View()
	// shy:main has 3(Mon) + 8(Tue) + 2(Wed) = 13 commands for the week
	assert.Contains(t, view, "13 commands")
}

// TestWeekDateNavMovesByWeek tests h/l moves by 7 days in week view
func TestWeekDateNavMovesByWeek(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week

	// Note currentDate before
	dateBefore := model.CurrentDate()

	pressKey(model, 'h') // go back 1 week
	dateAfter := model.CurrentDate()

	diff := dateBefore.Sub(dateAfter)
	assert.Equal(t, 7*24*time.Hour, diff)

	view := model.View()
	assert.Contains(t, view, "Week of Jan 26")
}

// TestMonthDateNavMovesByMonth tests h/l moves by month
func TestMonthDateNavMovesByMonth(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week
	pressBracketRight(model) // Week → Month

	pressKey(model, 'h') // go back 1 month
	view := model.View()
	assert.Contains(t, view, "January 2026")
}

// TestWeekDetailBucketsByDay tests week detail view buckets by day
func TestWeekDetailBucketsByDay(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week

	// Enter detail for shy:main (should be first, most commands)
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	view := model.View()
	// Should bucket by day (Feb 3 2026 = Tuesday)
	assert.Contains(t, view, "Tue Feb 3")
	assert.Contains(t, view, "Wed Feb 4")
	assert.Contains(t, view, "Thu Feb 5")
}

// TestWeekDetailShowsFullTimestamps tests week detail shows full timestamps
func TestWeekDetailShowsFullTimestamps(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model)
	pressEnter(model)

	view := model.View()
	// Should show full time like "9:15 AM" instead of just ":15"
	assert.Contains(t, view, "AM")
}

// TestMonthDetailBucketsByWeek tests month detail view buckets by week
func TestMonthDetailBucketsByWeek(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week
	pressBracketRight(model) // Week → Month

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	view := model.View()
	assert.Contains(t, view, "Week of Feb 2") // Feb 3-5 are in ISO week 6, Monday is Feb 2
}

// TestHeaderWeekFormat tests header format in week view
func TestHeaderWeekFormat(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model)

	view := model.View()
	assert.Contains(t, view, "Week of Feb 2")
	assert.Contains(t, view, "Week")
}

// TestHeaderMonthFormat tests header format in month view
func TestHeaderMonthFormat(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model)
	pressBracketRight(model)

	view := model.View()
	assert.Contains(t, view, "February 2026")
	assert.Contains(t, view, "Month")
}

// TestHeaderDayPeriodIndicator tests header shows Day indicator
func TestHeaderDayPeriodIndicator(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	view := model.View()
	assert.Contains(t, view, "Day")
}

// TestCannotNavigatePastCurrentWeek tests cannot navigate past current week
func TestCannotNavigatePastCurrentWeek(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week

	dateBefore := model.CurrentDate()
	pressKey(model, 'l') // Try to navigate forward
	dateAfter := model.CurrentDate()

	// Should not have moved (we're already in current week)
	assert.Equal(t, dateBefore.Format("2006-01-02"), dateAfter.Format("2006-01-02"))
}

// TestCannotNavigatePastCurrentMonth tests cannot navigate past current month
func TestCannotNavigatePastCurrentMonth(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week
	pressBracketRight(model) // Week → Month

	dateBefore := model.CurrentDate()
	pressKey(model, 'l') // Try to navigate forward
	dateAfter := model.CurrentDate()

	assert.Equal(t, dateBefore.Format("2006-01-02"), dateAfter.Format("2006-01-02"))
}

// TestPeriodSwitchInDetailResetsSelection tests period switch resets selection
func TestPeriodSwitchInDetailResetsSelection(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	// Navigate to Tue Feb 4
	pressKey(model, 'h')
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	// Navigate to 5th command
	for i := 0; i < 4; i++ {
		pressKey(model, 'j')
	}
	assert.Equal(t, 4, model.DetailCmdIdx())

	// Switch to week period
	pressBracketRight(model)
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 0, model.DetailCmdIdx())
}

// TestPeriodSwitchPreservesDisplayMode tests display mode is preserved
func TestPeriodSwitchPreservesDisplayMode(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressKey(model, 'u') // unique mode
	assert.Equal(t, UniqueMode, model.DisplayMode())

	pressBracketRight(model) // Day → Week
	assert.Equal(t, UniqueMode, model.DisplayMode())
}

// TestPeriodSwitchPreservesContext tests context is preserved in detail view
func TestPeriodSwitchPreservesContext(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	// Navigate to Tue and enter detail for shy:main
	pressKey(model, 'h')
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	view := model.View()
	assert.Contains(t, view, "main")

	// Switch to week
	pressBracketRight(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	view = model.View()
	assert.Contains(t, view, "main")
}

// TestAnchorDatePreservedWeekToDay tests Day→Week→Day restores anchor date
func TestAnchorDatePreservedWeekToDay(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	// Navigate to Tue Feb 4
	pressKey(model, 'h')
	assert.Contains(t, model.CurrentDate().Format("2006-01-02"), "2026-02-04")

	// Day → Week
	pressBracketRight(model)
	assert.Equal(t, WeekPeriod, model.Period())

	// Week → Day (should restore Feb 4)
	pressBracketLeft(model)
	assert.Equal(t, DayPeriod, model.Period())
	assert.Equal(t, "2026-02-04", model.CurrentDate().Format("2006-01-02"))
}

// TestPeriodKeysBothViews tests ] and [ work in both summary and detail views
func TestPeriodKeysBothViews(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	// Summary view
	pressBracketRight(model)
	assert.Equal(t, WeekPeriod, model.Period())
	pressBracketRight(model)
	assert.Equal(t, MonthPeriod, model.Period())
	pressBracketLeft(model)
	assert.Equal(t, WeekPeriod, model.Period())
	pressBracketLeft(model)
	assert.Equal(t, DayPeriod, model.Period())
}

// TestUniqueModeInWeekView tests unique mode works in week view
func TestUniqueModeInWeekView(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week
	pressKey(model, 'u')     // unique mode

	// In week view, shy:main has 13 total commands.
	// "go build -o shy ." appears 4x, "go test ./... -v" 3x, "shy summary" 2x
	// Unique: "./shy summary"(1), git commit(1), git push(1), "go test ./cmd -v"(1) = 4
	view := model.View()
	assert.Contains(t, view, "4 commands")
}

// TestFilterWorksInWeekView tests filter works in week view
func TestFilterWorksInWeekView(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	pressBracketRight(model) // Day → Week

	// Set filter to "build"
	pressSlash(model)
	typeString(model, "build")
	pressEnter(model)

	view := model.View()
	// shy:main has 4 "go build" commands across the week
	assert.Contains(t, view, "4 commands")
	// downloads has 0 matching
	assert.Contains(t, view, "0 commands")
}

// TestFilterPersistsWhenSwitchingPeriods tests filter persists when switching periods
func TestFilterPersistsWhenSwitchingPeriods(t *testing.T) {
	today := time.Date(2026, 2, 6, 12, 0, 0, 0, time.Local)

	dbPath := setupTestDB(t, phase4Commands())
	model := initModel(t, dbPath, today)

	// Set filter
	pressSlash(model)
	typeString(model, "git")
	pressEnter(model)
	assert.Equal(t, "git", model.FilterText())

	// Switch to week
	pressBracketRight(model)
	assert.Equal(t, "git", model.FilterText())

	// Switch to month
	pressBracketRight(model)
	assert.Equal(t, "git", model.FilterText())
}

// === Empty State Navigation Hints Tests ===

// TestEmptyDetailShowsContextHints tests H/L context hints in empty detail view.
// This is synchronous and needs no DB peek.
func TestEmptyDetailShowsContextHints(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	// Three contexts: shy:main (most commands), shy:bugfix, downloads
	// "go build" is duplicated in shy:main so UniqueMode filters it to 0.
	commands := []models.Command{
		makeCommandWithText(yesterday, 9, 0, "go build", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(yesterday, 9, 5, "go build", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(yesterday, 9, 10, "go build", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(yesterday, 10, 0, "git status", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("bugfix")),
		makeCommandWithText(yesterday, 11, 0, "ls -la", "/home/user/downloads", nil, nil),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	// Context 0 = shy:main (3 commands), then bugfix (1), downloads (1)
	assert.Equal(t, 3, len(model.Contexts()))

	// Enter detail for first context (shy:main)
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 3, len(model.DetailCommands()))

	// Switch to UniqueMode — "go build" appears 3 times, so 0 unique commands remain
	pressKey(model, 'u')
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 0, len(model.DetailCommands()))

	// The view should show all four hint keys (H, L, h, l)
	view := model.View()
	assert.Contains(t, view, "H")
	assert.Contains(t, view, "L")
	assert.Contains(t, view, "h")
	assert.Contains(t, view, "l")

	// L hint should show the next context's count
	assert.Contains(t, view, "command")
}

// TestEmptyDetailShowsPeriodHints tests h/l period hints when detail is empty
// due to filter. This requires DB peek to resolve adjacent periods.
func TestEmptyDetailShowsPeriodHints(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := yesterday.AddDate(0, 0, -1)

	// Commands on two different days for the same context
	commands := []models.Command{
		makeCommandWithText(dayBefore, 9, 0, "go build", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(dayBefore, 10, 0, "go test", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(dayBefore, 11, 0, "go build", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(yesterday, 9, 0, "git status", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	// Enter detail for the context
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 1, len(model.DetailCommands()))

	// Apply a filter that matches nothing on yesterday but matches on dayBefore
	pressSlash(model)
	typeString(model, "go build")
	pressEnter(model)

	assert.Equal(t, 0, len(model.DetailCommands()))

	// Period peek data should be loaded
	assert.NotNil(t, model.EmptyPrevPeriod(), "should have prev period peek data")
	assert.Equal(t, 2, model.EmptyPrevPeriod().Count(), "dayBefore has 2 'go build' commands (unique filtered out from 3, but mode is All)")

	// View should show the h hint with the date label
	view := model.View()
	assert.Contains(t, view, "h")
	assert.Contains(t, view, "2 commands")
}

// TestEmptyDetailNoForwardPeekAtCurrentPeriod tests that 'l' hint is not shown
// when we are at the current period.
func TestEmptyDetailNoForwardPeekAtCurrentPeriod(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)

	// Only commands today in a single context
	commands := []models.Command{
		makeCommandWithText(today, 9, 0, "go build", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(today, 9, 5, "go build", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	// Navigate to today
	pressKey(model, 't')
	pressEnter(model)

	// Switch to UniqueMode — "go build" is duplicated, so 0 commands
	pressKey(model, 'u')
	assert.Equal(t, 0, len(model.DetailCommands()))

	// Should not have a next period peek (we're at current period)
	assert.Nil(t, model.EmptyNextPeriod(), "should not peek forward from current period")
}

// TestEmptyDetailOrphanedContextLGoesToFirst tests that L navigates to the
// first available context when the current context is not in the period's list.
func TestEmptyDetailOrphanedContextLGoesToFirst(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := yesterday.AddDate(0, 0, -1)

	// shy:main exists only on dayBefore, downloads exists on both days
	commands := []models.Command{
		makeCommandWithText(dayBefore, 9, 0, "go build", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(dayBefore, 10, 0, "ls -la", "/home/user/downloads", nil, nil),
		makeCommandWithText(yesterday, 11, 0, "cat file.txt", "/home/user/downloads", nil, nil),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	// Navigate to dayBefore (h twice from yesterday)
	pressKey(model, 'h')
	assert.Equal(t, 2, len(model.Contexts()))

	// Enter shy:main (should be at index 0 — it has 1 command, same as downloads)
	// Find shy:main's index
	var mainIdx int
	for i, ctx := range model.Contexts() {
		if ctx.Branch == "main" {
			mainIdx = i
			break
		}
	}
	// Navigate to shy:main and enter detail
	for i := 0; i < mainIdx; i++ {
		pressKey(model, 'j')
	}
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 1, len(model.DetailCommands()))

	// Navigate forward to yesterday — shy:main doesn't exist there
	pressKey(model, 'l')
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 0, len(model.DetailCommands()), "orphaned context has no commands")

	// The empty state should show the L hint with downloads context name
	view := model.View()
	assert.Contains(t, view, "downloads", "L hint should show first available context name")

	// Press L — should jump to the first available context (downloads)
	pressShiftKey(model, 'L')
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 0, model.SelectedIdx(), "L should select first context")
	assert.True(t, len(model.DetailCommands()) > 0, "should have commands after L")
}

// TestEmptyDetailOrphanedContextHGoesToLast tests that H navigates to the
// last available context when the current context is not in the period's list.
func TestEmptyDetailOrphanedContextHGoesToLast(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := yesterday.AddDate(0, 0, -1)

	// shy:main exists only on dayBefore; downloads and projects exist on yesterday
	commands := []models.Command{
		makeCommandWithText(dayBefore, 9, 0, "go build", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(yesterday, 10, 0, "ls -la", "/home/user/downloads", nil, nil),
		makeCommandWithText(yesterday, 11, 0, "cat file.txt", "/home/user/projects/other", nil, nil),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	// Navigate to dayBefore
	pressKey(model, 'h')

	// Enter shy:main
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	// Navigate forward to yesterday — shy:main doesn't exist there
	pressKey(model, 'l')
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 0, len(model.DetailCommands()))

	// The empty state should show the H hint with the last context's name
	lastCtx := model.Contexts()[len(model.Contexts())-1]
	lastName := formatContextName(lastCtx.Key, lastCtx.Branch)
	view := model.View()
	assert.Contains(t, view, lastName, "H hint should show last available context name")

	// Press H — should jump to the last available context
	pressShiftKey(model, 'H')
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, len(model.Contexts())-1, model.SelectedIdx(), "H should select last context")
	assert.True(t, len(model.DetailCommands()) > 0, "should have commands after H")
}

// TestEmptyDetailFilterDoesNotNavigate tests that typing in the filter bar
// on the empty detail screen does not trigger navigation keys like 't'.
func TestEmptyDetailFilterDoesNotNavigate(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	// One context with duplicate commands so UniqueMode empties it
	commands := []models.Command{
		makeCommandWithText(yesterday, 9, 0, "go test", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(yesterday, 9, 5, "go test", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(yesterday, 9, 10, "go test", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	// Switch to UniqueMode — all commands are duplicated, so 0 remain
	pressKey(model, 'u')
	assert.Equal(t, 0, len(model.DetailCommands()))

	// Open filter and type "t" — should add to filter, not navigate
	pressSlash(model)
	assert.True(t, model.FilterActive())

	typeString(model, "t")
	assert.Equal(t, "t", model.FilterText(), "filter should contain 't'")
	assert.Equal(t, ContextDetailView, model.ViewState(), "should stay in detail view")
	assert.True(t, model.FilterActive(), "filter should still be active")
}

// TestEmptyDetailFilterOrphanedDoesNotSwitchContext tests that typing in the
// filter bar on an orphaned empty detail screen does not switch to a different context.
func TestEmptyDetailFilterOrphanedDoesNotSwitchContext(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := yesterday.AddDate(0, 0, -1)

	// shy:main exists only on dayBefore, downloads exists on yesterday
	commands := []models.Command{
		makeCommandWithText(dayBefore, 9, 0, "go build", "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
		makeCommandWithText(yesterday, 10, 0, "ls -la", "/home/user/downloads", nil, nil),
	}

	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)

	// Navigate to dayBefore
	pressKey(model, 'h')

	// Enter shy:main
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 1, len(model.DetailCommands()))

	// Navigate forward to yesterday — shy:main doesn't exist there (orphaned)
	pressKey(model, 'l')
	assert.Equal(t, ContextDetailView, model.ViewState())
	assert.Equal(t, 0, len(model.DetailCommands()))

	// Remember the context key before filtering
	headerBefore := model.View()
	assert.Contains(t, headerBefore, "shy", "header should still show the orphaned context")

	// Open filter and type "t" — should NOT switch to downloads
	pressSlash(model)
	assert.True(t, model.FilterActive())

	typeString(model, "ls")
	assert.Equal(t, "ls", model.FilterText())
	assert.Equal(t, ContextDetailView, model.ViewState())

	// The header should still reference the orphaned context
	headerAfter := model.View()
	assert.Contains(t, headerAfter, "shy:main", "header should still show orphaned context after filtering")
	assert.Equal(t, 0, len(model.DetailCommands()), "should still show empty commands")
}

// TestHelpViewFromSummary tests pressing ? in SummaryView enters HelpView and ? returns
func TestHelpViewFromSummary(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	})
	model := initModel(t, dbPath, today)

	assert.Equal(t, SummaryView, model.ViewState())

	// Press ? to enter help
	pressKey(model, '?')
	assert.Equal(t, HelpView, model.ViewState())
	assert.Equal(t, SummaryView, model.HelpPreviousView())

	// Press ? again to return
	pressKey(model, '?')
	assert.Equal(t, SummaryView, model.ViewState())
}

// TestHelpViewFromContextDetail tests pressing ? in ContextDetailView
func TestHelpViewFromContextDetail(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	})
	model := initModel(t, dbPath, today)

	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	pressKey(model, '?')
	assert.Equal(t, HelpView, model.ViewState())
	assert.Equal(t, ContextDetailView, model.HelpPreviousView())

	// Esc returns to previous view
	pressEsc(model)
	assert.Equal(t, ContextDetailView, model.ViewState())
}

// TestHelpViewFromCommandDetail tests pressing ? in CommandDetailView
func TestHelpViewFromCommandDetail(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	commands := phase2Commands(yesterday)
	dbPath := setupTestDB(t, commands)
	model := initModel(t, dbPath, today)
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Enter context detail, then command detail
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())
	pressEnter(model)
	assert.Equal(t, CommandDetailView, model.ViewState())

	pressKey(model, '?')
	assert.Equal(t, HelpView, model.ViewState())
	assert.Equal(t, CommandDetailView, model.HelpPreviousView())

	pressKey(model, '?')
	assert.Equal(t, CommandDetailView, model.ViewState())
}

// TestHelpViewEscReturns tests esc in HelpView returns to previous view
func TestHelpViewEscReturns(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	})
	model := initModel(t, dbPath, today)

	pressKey(model, '?')
	assert.Equal(t, HelpView, model.ViewState())

	pressEsc(model)
	assert.Equal(t, SummaryView, model.ViewState())
}

// TestHelpViewRendersBindings tests that help view renders keybindings for source view
func TestHelpViewRendersBindings(t *testing.T) {
	today := time.Date(2026, 2, 5, 12, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)

	dbPath := setupTestDB(t, []models.Command{
		makeCommand(yesterday, 9, "/home/user/projects/shy", strPtr("github.com/chris/shy"), strPtr("main")),
	})
	model := initModel(t, dbPath, today)
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Help from summary view
	pressKey(model, '?')
	view := model.View()
	assert.Contains(t, view, "Help")
	assert.Contains(t, view, "Navigate down")
	assert.Contains(t, view, "Open context")
	assert.Contains(t, view, "Quit")
	// Should NOT contain context detail bindings
	assert.NotContains(t, view, "Back to summary")

	// Return and go to detail view
	pressKey(model, '?')
	pressEnter(model)
	assert.Equal(t, ContextDetailView, model.ViewState())

	pressKey(model, '?')
	view = model.View()
	assert.Contains(t, view, "View command detail")
	assert.Contains(t, view, "Back to summary")
	assert.Contains(t, view, "Previous context")
}
