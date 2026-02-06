package tui

import (
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
