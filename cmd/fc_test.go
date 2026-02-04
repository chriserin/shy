package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

// Scenario 1: Filter commands with simple wildcard prefix match
func TestPatternScenario1_FilterCommandsWithSimpleWildcardPrefixMatch(t *testing.T) {
	defer resetFcFlags(fcCmd) // Reset flags after test

	// Given: I have the following commands in history
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	commands := []struct {
		id   int64
		text string
	}{
		{100, "git status"},
		{101, "git commit"},
		{102, "ls -la"},
		{103, "git push"},
		{104, "npm test"},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText: cmd.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + cmd.id),
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// When: I run "shy fc -l -m 'git*'"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "-m", "git*", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: the output should contain git commands in order
	assert.Contains(t, output, "git status")
	assert.Contains(t, output, "git commit")
	assert.Contains(t, output, "git push")

	// And: should not contain non-git commands
	assert.NotContains(t, output, "ls -la")
	assert.NotContains(t, output, "npm test")

	// Verify ordering (IDs will be auto-generated as 1, 2, 3, 4, 5)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, 3)
	assert.Contains(t, lines[0], "1")
	assert.Contains(t, lines[0], "git status")
	assert.Contains(t, lines[1], "2")
	assert.Contains(t, lines[1], "git commit")
	assert.Contains(t, lines[2], "4")
	assert.Contains(t, lines[2], "git push")

	rootCmd.SetArgs(nil)
}

// Scenario 2: Filter commands with wildcard suffix match
func TestPatternScenario2_FilterCommandsWithWildcardSuffixMatch(t *testing.T) {
	defer resetFcFlags(fcCmd) // Reset flags after test

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	commands := []struct {
		id   int64
		text string
	}{
		{200, "test-unit"},
		{201, "npm test"},
		{202, "pytest"},
		{203, "cargo test"},
		{204, "ls"},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText: cmd.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + cmd.id),
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "-m", "*test", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	assert.Contains(t, output, "npm test")
	assert.Contains(t, output, "pytest")
	assert.Contains(t, output, "cargo test")
	assert.NotContains(t, output, "test-unit")
	assert.NotContains(t, output, "ls")

	rootCmd.SetArgs(nil)
}

// Scenario 3: Filter commands with wildcard in middle
func TestPatternScenario3_FilterCommandsWithWildcardInMiddle(t *testing.T) {
	defer resetFcFlags(fcCmd) // Reset flags after test

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	commands := []struct {
		id   int64
		text string
	}{
		{300, "git commit -m \"fix\""},
		{301, "npm install"},
		{302, "git commit -m \"feature\""},
		{303, "git status"},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText: cmd.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + cmd.id),
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "-m", "git*commit*", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	assert.Contains(t, output, "git commit -m \"fix\"")
	assert.Contains(t, output, "git commit -m \"feature\"")
	assert.NotContains(t, output, "npm install")
	assert.NotContains(t, output, "git status")

	rootCmd.SetArgs(nil)
}

// Scenario 4: Filter commands with single character wildcard
func TestPatternScenario4_FilterCommandsWithSingleCharacterWildcard(t *testing.T) {
	defer resetFcFlags(fcCmd) // Reset flags after test

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	commands := []struct {
		id   int64
		text string
	}{
		{400, "git"},
		{401, "cat"},
		{402, "cut"},
		{403, "apt"},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText: cmd.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + cmd.id),
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "-m", "?at", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	assert.Len(t, lines, 1)
	assert.Contains(t, output, "cat")
	assert.NotContains(t, output, "git")
	assert.NotContains(t, output, "cut")
	assert.NotContains(t, output, "apt")

	rootCmd.SetArgs(nil)
}

// Scenario 5: Filter commands with multiple wildcards
func TestPatternScenario5_FilterCommandsWithMultipleWildcards(t *testing.T) {
	defer resetFcFlags(fcCmd) // Reset flags after test

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	commands := []struct {
		id   int64
		text string
	}{
		{500, "docker build -t app"},
		{501, "podman build -t service"},
		{502, "docker run app"},
		{503, "npm build"},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText: cmd.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + cmd.id),
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "-m", "*build*", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	assert.Contains(t, output, "docker build -t app")
	assert.Contains(t, output, "podman build -t service")
	assert.Contains(t, output, "npm build")
	assert.NotContains(t, output, "docker run app")

	rootCmd.SetArgs(nil)
}

// Scenario 6: Pattern filter with no matches returns exit code 1
func TestPatternScenario6_PatternFilterWithNoMatches(t *testing.T) {
	defer resetFcFlags(fcCmd) // Reset flags after test

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	commands := []struct {
		id   int64
		text string
	}{
		{600, "ls"},
		{601, "pwd"},
		{602, "echo test"},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText: cmd.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + cmd.id),
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	rootCmd.SetArgs([]string{"fc", "-l", "-m", "git*", "--db", dbPath})

	err = rootCmd.Execute()
	if err != nil {
		fmt.Println("Filter no matches Error:", err)
	}

	// Verify error message is printed
	assert.Contains(t, err.Error(), "shy fc: no matching events found")

	rootCmd.SetArgs(nil)
}

// Scenario 7: Pattern filter with range
func TestPatternScenario7_PatternFilterWithRange(t *testing.T) {
	defer resetFcFlags(fcCmd) // Reset flags after test

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	commands := []struct {
		id   int64
		text string
	}{
		{700, "git status"},
		{701, "git commit"},
		{702, "ls"},
		{703, "git push"},
		{704, "git pull"},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText: cmd.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + cmd.id),
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	// IDs are auto-generated as 1-5, so range 1-4 covers first 4 commands
	rootCmd.SetArgs([]string{"fc", "-l", "1", "4", "-m", "git*", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should match git status (1), git commit (2), git push (4)
	// Should NOT match git pull (5) - outside range
	assert.Contains(t, output, "git status")
	assert.Contains(t, output, "git commit")
	assert.Contains(t, output, "git push")
	assert.NotContains(t, output, "git pull")
	assert.NotContains(t, output, "ls")

	rootCmd.SetArgs(nil)
}

// Scenario 9: Pattern filter with SQL special characters
func TestPatternScenario9_PatternFilterWithSQLSpecialCharacters(t *testing.T) {
	defer resetFcFlags(fcCmd) // Reset flags after test

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	commands := []struct {
		id   int64
		text string
	}{
		{900, "test_file"},
		{901, "test-file"},
		{902, "testfile"},
		{903, "test file"},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText: cmd.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + cmd.id),
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "-m", "test?file", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// ? matches any single character
	assert.Len(t, lines, 3)
	assert.Contains(t, output, "test_file")
	assert.Contains(t, output, "test-file")
	assert.Contains(t, output, "test file")
	assert.NotContains(t, output, "testfile")

	rootCmd.SetArgs(nil)
}

// Scenario 10: Pattern filter combined with other flags
func TestPatternScenario10_PatternFilterCombinedWithOtherFlags(t *testing.T) {
	defer resetFcFlags(fcCmd) // Reset flags after test

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	commands := []struct {
		id        int64
		text      string
		timestamp int64
	}{
		{1000, "git status", 1704470400},
		{1001, "git commit", 1704470401},
		{1002, "git push", 1704470402},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText: cmd.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   cmd.timestamp,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-n", "-r", "-l", "-m", "git*", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should not contain event numbers
	for _, line := range lines {
		// Check that lines don't start with numbers followed by spaces
		// (which would indicate event numbers)
		trimmed := strings.TrimSpace(line)
		assert.True(t, strings.HasPrefix(trimmed, "git"))
	}

	// Should be in reverse order
	assert.Contains(t, lines[0], "git push")
	assert.Contains(t, lines[1], "git commit")
	assert.Contains(t, lines[2], "git status")

	rootCmd.SetArgs(nil)
}

// Test globToLike function
func TestGlobToLike(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple prefix", "git*", "git%"},
		{"simple suffix", "*test", "%test"},
		{"middle wildcard", "git*commit", "git%commit"},
		{"single char", "?at", "_at"},
		{"multiple wildcards", "*build*", "%build%"},
		{"escape percent", "test%value", "test\\%value"},
		{"escape underscore", "test_file", "test\\_file"},
		{"mixed escapes", "test_%value*", "test\\_\\%value%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := globToLike(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Scenario 11: Show only internal commands from current session
func TestInternalScenario11_ShowOnlyInternalCommandsFromCurrentSession(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Setup test data
	sourceApp1 := "zsh"
	sourcePid1 := int64(12345)
	sourceActive1 := true

	sourceApp2 := "zsh"
	sourcePid2 := int64(67890)
	sourceActive2 := true

	commands := []struct {
		id     int64
		text   string
		app    *string
		pid    *int64
		active *bool
	}{
		{1100, "ls", &sourceApp1, &sourcePid1, &sourceActive1},
		{1101, "pwd", &sourceApp1, &sourcePid1, &sourceActive1},
		{1102, "echo test", &sourceApp2, &sourcePid2, &sourceActive2},
		{1103, "git status", &sourceApp1, &sourcePid1, &sourceActive1},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText:  cmd.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    int64(1704470400 + cmd.id),
			SourceApp:    cmd.app,
			SourcePid:    cmd.pid,
			SourceActive: cmd.active,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// Set current session PID
	os.Setenv("SHY_SESSION_PID", "12345")
	defer os.Unsetenv("SHY_SESSION_PID")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "-I", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should contain commands from session 12345
	assert.Contains(t, output, "ls")
	assert.Contains(t, output, "pwd")
	assert.Contains(t, output, "git status")

	// Should not contain commands from session 67890
	assert.NotContains(t, output, "echo test")

	rootCmd.SetArgs(nil)
}

// Scenario 12: Internal filter shows no commands from closed sessions
func TestInternalScenario12_InternalFilterShowsNoCommandsFromClosedSessions(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Setup test data
	sourceApp1 := "zsh"
	sourcePid1 := int64(12345)
	sourceActive1 := false // Closed session

	sourceApp2 := "zsh"
	sourcePid2 := int64(67890)
	sourceActive2 := true // Active session

	commands := []struct {
		id     int64
		text   string
		app    *string
		pid    *int64
		active *bool
	}{
		{1200, "ls", &sourceApp1, &sourcePid1, &sourceActive1},
		{1201, "pwd", &sourceApp1, &sourcePid1, &sourceActive1},
		{1202, "echo test", &sourceApp2, &sourcePid2, &sourceActive2},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText:  cmd.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    int64(1704470400 + cmd.id),
			SourceApp:    cmd.app,
			SourcePid:    cmd.pid,
			SourceActive: cmd.active,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// Set current session PID
	os.Setenv("SHY_SESSION_PID", "67890")
	defer os.Unsetenv("SHY_SESSION_PID")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "-I", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should only contain "echo test"
	assert.Len(t, lines, 1)
	assert.Contains(t, output, "echo test")

	// Should not contain commands from closed session 12345
	assert.NotContains(t, output, "ls")
	assert.NotContains(t, output, "pwd")

	rootCmd.SetArgs(nil)
}

// Scenario 13: Internal filter with range
func TestInternalScenario13_InternalFilterWithRange(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Setup test data
	sourceApp1 := "zsh"
	sourcePid1 := int64(11111)
	sourceActive1 := true

	sourceApp2 := "zsh"
	sourcePid2 := int64(22222)
	sourceActive2 := true

	commands := []struct {
		id     int64
		text   string
		app    *string
		pid    *int64
		active *bool
	}{
		{1300, "cmd1", &sourceApp1, &sourcePid1, &sourceActive1},
		{1301, "cmd2", &sourceApp2, &sourcePid2, &sourceActive2},
		{1302, "cmd3", &sourceApp2, &sourcePid2, &sourceActive2},
		{1303, "cmd4", &sourceApp1, &sourcePid1, &sourceActive1},
		{1304, "cmd5", &sourceApp2, &sourcePid2, &sourceActive2},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText:  cmd.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    int64(1704470400 + cmd.id),
			SourceApp:    cmd.app,
			SourcePid:    cmd.pid,
			SourceActive: cmd.active,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// Set current session PID
	os.Setenv("SHY_SESSION_PID", "22222")
	defer os.Unsetenv("SHY_SESSION_PID")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	// IDs are auto-generated as 1-5, so range 2-5 covers commands 2-5
	rootCmd.SetArgs([]string{"fc", "-l", "2", "5", "-I", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should contain cmd2, cmd3, cmd5 from session 22222 in range 2-5
	assert.Contains(t, output, "cmd2")
	assert.Contains(t, output, "cmd3")
	assert.Contains(t, output, "cmd5")

	// Should not contain cmd1 or cmd4 (from different session)
	assert.NotContains(t, output, "cmd1")
	assert.NotContains(t, output, "cmd4")

	rootCmd.SetArgs(nil)
}

// Scenario 14: Internal filter combined with pattern filter
func TestInternalScenario14_InternalFilterCombinedWithPatternFilter(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Setup test data
	sourceApp1 := "zsh"
	sourcePid1 := int64(12345)
	sourceActive1 := true

	sourceApp2 := "zsh"
	sourcePid2 := int64(67890)
	sourceActive2 := true

	commands := []struct {
		id     int64
		text   string
		app    *string
		pid    *int64
		active *bool
	}{
		{1400, "git status", &sourceApp1, &sourcePid1, &sourceActive1},
		{1401, "git commit", &sourceApp2, &sourcePid2, &sourceActive2},
		{1402, "git push", &sourceApp1, &sourcePid1, &sourceActive1},
		{1403, "ls", &sourceApp1, &sourcePid1, &sourceActive1},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText:  cmd.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    int64(1704470400 + cmd.id),
			SourceApp:    cmd.app,
			SourcePid:    cmd.pid,
			SourceActive: cmd.active,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// Set current session PID
	os.Setenv("SHY_SESSION_PID", "12345")
	defer os.Unsetenv("SHY_SESSION_PID")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "-I", "-m", "git*", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should contain git commands from session 12345
	assert.Contains(t, output, "git status")
	assert.Contains(t, output, "git push")

	// Should not contain git commit (from different session)
	assert.NotContains(t, output, "git commit")
	// Should not contain ls (doesn't match pattern)
	assert.NotContains(t, output, "ls")

	rootCmd.SetArgs(nil)
}

// Scenario 15: Internal filter with different shell sources
func TestInternalScenario15_InternalFilterWithDifferentShellSources(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Setup test data
	sourceAppBash := "bash"
	sourcePidBash := int64(11111)
	sourceActiveBash := true

	sourceAppZsh := "zsh"
	sourcePidZsh := int64(22222)
	sourceActiveZsh := true

	commands := []struct {
		id     int64
		text   string
		app    *string
		pid    *int64
		active *bool
	}{
		{1500, "bash-cmd", &sourceAppBash, &sourcePidBash, &sourceActiveBash},
		{1501, "zsh-cmd1", &sourceAppZsh, &sourcePidZsh, &sourceActiveZsh},
		{1502, "zsh-cmd2", &sourceAppZsh, &sourcePidZsh, &sourceActiveZsh},
		{1503, "bash-cmd2", &sourceAppBash, &sourcePidBash, &sourceActiveBash},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText:  cmd.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    int64(1704470400 + cmd.id),
			SourceApp:    cmd.app,
			SourcePid:    cmd.pid,
			SourceActive: cmd.active,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// Set current session PID (zsh session)
	os.Setenv("SHY_SESSION_PID", "22222")
	defer os.Unsetenv("SHY_SESSION_PID")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "-I", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Should contain zsh commands only
	assert.Contains(t, output, "zsh-cmd1")
	assert.Contains(t, output, "zsh-cmd2")

	// Should not contain bash commands
	assert.NotContains(t, output, "bash-cmd")
	assert.NotContains(t, output, "bash-cmd2")

	rootCmd.SetArgs(nil)
}

// Scenario 16: Internal filter when session is closed
func TestInternalScenario16_InternalFilterWhenSessionIsClosed(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Setup test data with closed session
	sourceApp1 := "zsh"
	sourcePid1 := int64(12345)
	sourceActive1 := false // Closed session

	commands := []struct {
		id     int64
		text   string
		app    *string
		pid    *int64
		active *bool
	}{
		{1600, "old-cmd", &sourceApp1, &sourcePid1, &sourceActive1},
		{1601, "old-cmd2", &sourceApp1, &sourcePid1, &sourceActive1},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText:  cmd.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    int64(1704470400 + cmd.id),
			SourceApp:    cmd.app,
			SourcePid:    cmd.pid,
			SourceActive: cmd.active,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// Now start a new session with the same PID (PID reuse)
	sourceApp2 := "zsh"
	sourcePid2 := int64(12345)
	sourceActive2 := true

	newCmd := &models.Command{
		CommandText:  "new-cmd",
		WorkingDir:   "/home/test",
		ExitStatus:   0,
		Timestamp:    int64(1704470400 + 1602),
		SourceApp:    &sourceApp2,
		SourcePid:    &sourcePid2,
		SourceActive: &sourceActive2,
	}
	_, err = database.InsertCommand(newCmd)
	require.NoError(t, err)

	// Set current session PID (new session with reused PID)
	os.Setenv("SHY_SESSION_PID", "12345")
	defer os.Unsetenv("SHY_SESSION_PID")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "-I", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should only contain new-cmd
	assert.Len(t, lines, 1)
	assert.Contains(t, output, "new-cmd")

	// Should not contain old commands from closed session
	assert.NotContains(t, output, "old-cmd")
	assert.NotContains(t, output, "old-cmd2")

	rootCmd.SetArgs(nil)
}

// Scenario 17: Local filter behaves identically to no filter
func TestLocalScenario17_LocalFilterBehavesIdenticallyToNoFilter(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Setup test data
	sourceApp1 := "zsh"
	sourcePid1 := int64(12345)
	sourceActive1 := true

	sourceApp2 := "zsh"
	sourcePid2 := int64(67890)
	sourceActive2 := true

	sourceApp3 := "bash"
	sourcePid3 := int64(11111)
	sourceActive3 := true

	commands := []struct {
		text   string
		app    *string
		pid    *int64
		active *bool
	}{
		{"ls", &sourceApp1, &sourcePid1, &sourceActive1},
		{"pwd", &sourceApp2, &sourcePid2, &sourceActive2},
		{"echo test", &sourceApp3, &sourcePid3, &sourceActive3},
	}

	for _, cmd := range commands {
		c := &models.Command{
			CommandText:  cmd.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    int64(1704470400),
			SourceApp:    cmd.app,
			SourcePid:    cmd.pid,
			SourceActive: cmd.active,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// Run "shy fc -l" without -L flag
	var bufNoFlag bytes.Buffer
	rootCmd.SetOut(&bufNoFlag)
	rootCmd.SetArgs([]string{"fc", "-l", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	outputNoFlag := bufNoFlag.String()

	// Run "shy fc -l -L" with -L flag
	var bufWithL bytes.Buffer
	rootCmd.SetOut(&bufWithL)
	rootCmd.SetArgs([]string{"fc", "-l", "-L", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	outputWithL := bufWithL.String()

	// Then: outputs should be identical
	assert.Equal(t, outputNoFlag, outputWithL, "-L flag should produce identical output to no flag")

	rootCmd.SetArgs(nil)
}

// Scenario 18: Local filter with range produces same results as without
func TestLocalScenario18_LocalFilterWithRangeProducesSameResults(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Setup test data
	commands := []string{"cmd1", "cmd2", "cmd3", "cmd4"}

	for _, cmdText := range commands {
		c := &models.Command{
			CommandText: cmdText,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400),
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// Run "shy fc -l 2 4" without -L flag (IDs 2-4)
	var bufNoFlag bytes.Buffer
	rootCmd.SetOut(&bufNoFlag)
	rootCmd.SetArgs([]string{"fc", "-l", "2", "4", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	outputNoFlag := bufNoFlag.String()

	// Run "shy fc -l 2 4 -L" with -L flag
	var bufWithL bytes.Buffer
	rootCmd.SetOut(&bufWithL)
	rootCmd.SetArgs([]string{"fc", "-l", "2", "4", "-L", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	outputWithL := bufWithL.String()

	// Then: outputs should be identical
	assert.Equal(t, outputNoFlag, outputWithL, "-L flag with range should produce identical output to no flag")

	// Verify the output contains the expected commands
	assert.Contains(t, outputWithL, "cmd2")
	assert.Contains(t, outputWithL, "cmd3")
	assert.Contains(t, outputWithL, "cmd4")
	assert.NotContains(t, outputWithL, "cmd1")

	rootCmd.SetArgs(nil)
}

// Scenario: No substitution as part of range
func TestSubstitutionNoSubstitutionAsPartOfRange(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	c := &models.Command{
		CommandText: "git status",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   int64(1704470400),
	}
	_, err = database.InsertCommand(c)
	require.NoError(t, err)

	// When: I run "shy fc -l 1 git=svn"
	// git=svn appears as second range arg, should be treated as event ID
	rootCmd.SetArgs([]string{"fc", "-l", "1", "git=svn", "--db", dbPath})

	err = rootCmd.Execute()
	assert.Contains(t, err.Error(), "shy fc: event not found: git=svn")

	rootCmd.SetArgs(nil)
}

// Scenario: No substitution after range
func TestSubstitutionNoSubstitutionAfterRange(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	c := &models.Command{
		CommandText: "git status",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   int64(1704470400),
	}
	_, err = database.InsertCommand(c)
	require.NoError(t, err)

	// When: I run "shy fc -l 1 1 git=svn"
	// git=svn appears after both range args
	rootCmd.SetArgs([]string{"fc", "-l", "1", "1", "git=svn", "--db", dbPath})

	err = rootCmd.Execute()
	// Error may be nil because osExit is called instead of returning an error

	// And: should print error message
	assert.Contains(t, err.Error(), "shy fc: too many arguments")

	rootCmd.SetArgs(nil)
}

// Scenario 19: Simple string substitution in list output
func TestSubstitutionScenario19_SimpleStringSubstitution(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	commands := []string{
		"git status",
		"git commit -m foo",
		"git push origin",
	}

	for _, cmdText := range commands {
		c := &models.Command{
			CommandText: cmdText,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400),
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// When: I run "shy fc -l git=svn 1 3"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "git=svn", "1", "3", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: output should have "git" replaced with "svn"
	assert.Contains(t, output, "svn status")
	assert.Contains(t, output, "svn commit -m foo")
	assert.Contains(t, output, "svn push origin")
	assert.NotContains(t, output, "git status")
	assert.NotContains(t, output, "git commit")
	assert.NotContains(t, output, "git push")

	rootCmd.SetArgs(nil)
}

// Scenario 20: Multiple substitutions applied to same command
func TestSubstitutionScenario20_MultipleSubstitutions(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	commands := []string{
		"echo hello world",
		"test hello world",
	}

	for _, cmdText := range commands {
		c := &models.Command{
			CommandText: cmdText,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400),
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// When: I run "shy fc -l hello=goodbye world=universe 1 2"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "hello=goodbye", "world=universe", "1", "2", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: both substitutions should be applied
	assert.Contains(t, output, "echo goodbye universe")
	assert.Contains(t, output, "test goodbye universe")
	assert.NotContains(t, output, "hello")
	assert.NotContains(t, output, "world")

	rootCmd.SetArgs(nil)
}

// Scenario 22: Substitution does not modify database
func TestSubstitutionScenario22_DoesNotModifyDatabase(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	c := &models.Command{
		CommandText: "git status",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   int64(1704470400),
	}
	id, err := database.InsertCommand(c)
	require.NoError(t, err)

	// When: I run "shy fc -l git=svn 1"
	var buf1 bytes.Buffer
	rootCmd.SetOut(&buf1)
	rootCmd.SetArgs([]string{"fc", "-l", "git=svn", "1", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Then: output should show "svn status"
	assert.Contains(t, buf1.String(), "svn status")

	// When: I run "shy fc -l 1" without substitution
	var buf2 bytes.Buffer
	rootCmd.SetOut(&buf2)
	rootCmd.SetArgs([]string{"fc", "-l", "1", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Then: output should show original "git status"
	assert.Contains(t, buf2.String(), "git status")

	// Verify database still has original command
	cmd, err := database.GetCommand(id)
	require.NoError(t, err)
	assert.Equal(t, "git status", cmd.CommandText)

	rootCmd.SetArgs(nil)
}

// Scenario 25: Substitution order matters (all occurrences replaced)
func TestSubstitutionScenario25_AllOccurrencesReplaced(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	c := &models.Command{
		CommandText: "test test test",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   int64(1704470400),
	}
	_, err = database.InsertCommand(c)
	require.NoError(t, err)

	// When: I run "shy fc -l test=foo 1"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "test=foo", "1", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Then: all occurrences should be replaced
	output := buf.String()
	assert.Contains(t, output, "foo foo foo")
	assert.NotContains(t, output, "test")

	rootCmd.SetArgs(nil)
}

// Scenario 26: Empty substitution removes text
func TestSubstitutionScenario26_EmptySubstitutionRemovesText(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	c := &models.Command{
		CommandText: "git --verbose",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   int64(1704470400),
	}
	_, err = database.InsertCommand(c)
	require.NoError(t, err)

	// When: I run "shy fc -l --verbose= 1" (empty replacement)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "--verbose=", "1", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Then: --verbose should be removed
	output := buf.String()
	assert.Contains(t, output, "git ")
	assert.NotContains(t, output, "verbose")

	rootCmd.SetArgs(nil)
}

// Scenario 30: Multiple old=new pairs with overlapping patterns
func TestSubstitutionScenario30_MultipleSubstitutionsWithOverlapping(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	c := &models.Command{
		CommandText: "git status",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   int64(1704470400),
	}
	_, err = database.InsertCommand(c)
	require.NoError(t, err)

	// When: I run "shy fc -l git=svn status=stat 1"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-l", "git=svn", "status=stat", "1", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Then: both substitutions should be applied
	output := buf.String()
	assert.Contains(t, output, "svn stat")
	assert.NotContains(t, output, "git")
	assert.NotContains(t, output, "status")

	rootCmd.SetArgs(nil)
}

// ========== FILE OPERATIONS TESTS ==========

// Test -W: Write history to file (always in extended format)
func TestFileOp_WriteToFile(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	outputFile := filepath.Join(tempDir, "export.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert test commands
	commands := []string{
		"test 2",
		"test 3",
		"test 4",
		"test 5",
		"test 6",
		"test 7",
		"test 8",
		"test 9",
		"test 10",
		"test 11",
		"test 12",
		"test 13",
		"test 14",
		"test 15",
		"test 16",
		"test 17",
		"test 18",
		"echo \"hello world\"",
		"git status",
		"npm install lodash",
	}

	timestamp := int64(1234567890)
	for _, cmd := range commands {
		c := &models.Command{
			CommandText: cmd,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   timestamp,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
		timestamp++
	}

	// Run: shy fc -W /tmp/export.txt
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-W", outputFile, "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify file was created
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	// Check content (always extended format)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Len(t, lines, 20)

	for i, command := range commands {
		expectedLine := fmt.Sprintf(": %d:0;%s", 1234567890+int64(i), command)
		assert.Equal(t, expectedLine, lines[i])
	}

	rootCmd.SetArgs(nil)
}

// Test -W: Write history in extended format
func TestFileOp_WriteExtendedFormat(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	outputFile := filepath.Join(tempDir, "export.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert test commands with timestamps and durations
	testData := []struct {
		text      string
		timestamp int64
		duration  int64
	}{
		{"echo \"hello world\"", 1234567890, 500},
		{"git status", 1234567891, 1200},
		{"npm install lodash", 1234567892, 45000},
	}

	for _, td := range testData {
		c := &models.Command{
			CommandText: td.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   td.timestamp,
			Duration:    &td.duration,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
	}

	// Run: shy fc -W /tmp/export.txt
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-W", outputFile, "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify file was created
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	// Check extended format (always used now)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, ": 1234567890:0;echo \"hello world\"", lines[0])
	assert.Equal(t, ": 1234567891:1;git status", lines[1])
	assert.Equal(t, ": 1234567892:45;npm install lodash", lines[2])

	rootCmd.SetArgs(nil)
}

// Test -W without arguments (no-op)
func TestFileOp_WriteWithoutFile_NoOp(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Run: shy fc -W (no file argument)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-W", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Should succeed with no output
	assert.Empty(t, buf.String())

	rootCmd.SetArgs(nil)
}

// Test -W: Create parent directories
func TestFileOp_WriteCreatesParentDirs(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	outputFile := filepath.Join(tempDir, "nested", "deep", "export.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	c := &models.Command{
		CommandText: "echo test",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   int64(1234567890),
	}
	_, err = database.InsertCommand(c)
	require.NoError(t, err)

	// Run: shy fc -W /tmp/nested/deep/export.txt
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-W", outputFile, "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify directories were created
	_, err = os.Stat(filepath.Join(tempDir, "nested", "deep"))
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(outputFile)
	require.NoError(t, err)

	rootCmd.SetArgs(nil)
}

// Test -A: Append to existing file
func TestFileOp_AppendToExistingFile(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	outputFile := filepath.Join(tempDir, "existing.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Create existing file with content (in extended format)
	existingContent := ": 1234567800:100;previous command 1\n: 1234567801:200;previous command 2\n"
	err = os.WriteFile(outputFile, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Insert test commands
	timestamp := int64(1234567890)
	commands := []string{"echo test", "git status"}
	for _, cmd := range commands {
		c := &models.Command{
			CommandText: cmd,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   timestamp,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
		timestamp++
	}

	// Run: shy fc -A /tmp/existing.txt
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-A", outputFile, "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify file content (all in extended format)
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Len(t, lines, 4)
	assert.Equal(t, ": 1234567800:100;previous command 1", lines[0])
	assert.Equal(t, ": 1234567801:200;previous command 2", lines[1])
	assert.Equal(t, ": 1234567890:0;echo test", lines[2])
	assert.Equal(t, ": 1234567891:0;git status", lines[3])

	rootCmd.SetArgs(nil)
}

// Test -A: Append to non-existent file (creates new)
func TestFileOp_AppendToNewFile(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	outputFile := filepath.Join(tempDir, "new_append.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	c := &models.Command{
		CommandText: "echo test",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   int64(1234567890),
	}
	_, err = database.InsertCommand(c)
	require.NoError(t, err)

	// Run: shy fc -A /tmp/new_append.txt
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-A", outputFile, "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify file was created (in extended format)
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, ": 1234567890:0;echo test\n", string(content))

	rootCmd.SetArgs(nil)
}

// Test -R: Read simple format
func TestFileOp_ReadSimpleFormat(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	inputFile := filepath.Join(tempDir, "import.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Create import file
	content := "echo \"imported 1\"\ngit clone repo\nmake build\n"
	err = os.WriteFile(inputFile, []byte(content), 0600)
	require.NoError(t, err)

	// Run: shy fc -R /tmp/import.txt
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-R", inputFile, "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify commands were imported
	commands, err := database.GetCommandsByRange(1, 100)
	require.NoError(t, err)
	require.Len(t, commands, 3)
	assert.Equal(t, "echo \"imported 1\"", commands[0].CommandText)
	assert.Equal(t, "git clone repo", commands[1].CommandText)
	assert.Equal(t, "make build", commands[2].CommandText)

	rootCmd.SetArgs(nil)
}

// Test -R: Read extended format
func TestFileOp_ReadExtendedFormat(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	inputFile := filepath.Join(tempDir, "import.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Create import file with extended format
	content := ": 1600000000:1500;ls -la\n: 1600000001:2000;cd /tmp\n: 1600000002:500;pwd\n"
	err = os.WriteFile(inputFile, []byte(content), 0600)
	require.NoError(t, err)

	// Run: shy fc -R /tmp/import.txt
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-R", inputFile, "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify commands were imported with timestamps and durations
	commands, err := database.GetCommandsByRangeFull(1, 100)
	require.NoError(t, err)
	require.Len(t, commands, 3)

	assert.Equal(t, "ls -la", commands[0].CommandText)
	assert.Equal(t, int64(1600000000), commands[0].Timestamp)
	assert.NotNil(t, commands[0].Duration)
	assert.Equal(t, int64(1500000), *commands[0].Duration)

	assert.Equal(t, "cd /tmp", commands[1].CommandText)
	assert.Equal(t, int64(1600000001), commands[1].Timestamp)

	assert.Equal(t, "pwd", commands[2].CommandText)
	assert.Equal(t, int64(1600000002), commands[2].Timestamp)

	rootCmd.SetArgs(nil)
}

// Test -R: Read mixed format file
func TestFileOp_ReadMixedFormat(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	inputFile := filepath.Join(tempDir, "mixed.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Create import file with mixed formats
	content := "echo \"simple format\"\n: 1600000000:1000;echo \"extended format\"\npwd\n"
	err = os.WriteFile(inputFile, []byte(content), 0600)
	require.NoError(t, err)

	// Run: shy fc -R /tmp/mixed.txt
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-R", inputFile, "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify all commands were imported
	commands, err := database.GetCommandsByRangeFull(1, 100)
	require.NoError(t, err)
	require.Len(t, commands, 3)

	assert.Equal(t, "echo \"simple format\"", commands[0].CommandText)
	assert.Equal(t, "echo \"extended format\"", commands[1].CommandText)
	assert.Equal(t, int64(1600000000), commands[1].Timestamp)
	assert.Equal(t, "pwd", commands[2].CommandText)

	rootCmd.SetArgs(nil)
}

// Test -R: Skip blank lines and comments
func TestFileOp_ReadSkipsBlankLinesAndComments(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	inputFile := filepath.Join(tempDir, "with_blanks.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Create import file with blank lines and comments
	content := "# This is a comment\necho \"line 1\"\n\n# Another comment\necho \"line 2\"\n\n\necho \"line 3\"\n"
	err = os.WriteFile(inputFile, []byte(content), 0600)
	require.NoError(t, err)

	// Run: shy fc -R /tmp/with_blanks.txt
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-R", inputFile, "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify only commands were imported (no comments or blank lines)
	commands, err := database.GetCommandsByRange(1, 100)
	require.NoError(t, err)
	require.Len(t, commands, 3)
	assert.Equal(t, "echo \"line 1\"", commands[0].CommandText)
	assert.Equal(t, "echo \"line 2\"", commands[1].CommandText)
	assert.Equal(t, "echo \"line 3\"", commands[2].CommandText)

	rootCmd.SetArgs(nil)
}

// Test -R: Non-existent file returns error
func TestFileOp_ReadNonExistentFile_Error(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	inputFile := filepath.Join(tempDir, "does_not_exist.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Run: shy fc -R /tmp/does_not_exist.txt
	rootCmd.SetArgs([]string{"fc", "-R", inputFile, "--db", dbPath})

	err = rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")

	rootCmd.SetArgs(nil)
}

// Test -W with pattern filter
func TestFileOp_WriteWithPatternFilter(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	outputFile := filepath.Join(tempDir, "filtered.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert test commands
	timestamp := int64(1234567890)
	commands := []string{"git status", "npm install", "git commit", "echo test"}
	for _, cmd := range commands {
		c := &models.Command{
			CommandText: cmd,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   timestamp,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
		timestamp++
	}

	// Run: shy fc -W /tmp/filtered.txt -m 'git*'
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-W", outputFile, "-m", "git*", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify only git commands were written (in extended format)
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Len(t, lines, 2)
	assert.Equal(t, ": 1234567890:0;git status", lines[0])
	assert.Equal(t, ": 1234567892:0;git commit", lines[1])

	rootCmd.SetArgs(nil)
}

// Test -W with -I (internal/session filter)
func TestFileOp_WriteWithInternalFilter(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	outputFile := filepath.Join(tempDir, "session.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert commands with different session PIDs
	currentPID := int64(12345)
	otherPID := int64(67890)

	timestamp := int64(1234567890)
	commands := []struct {
		text string
		pid  int64
	}{
		{"git status", currentPID},
		{"npm install", otherPID},
		{"git commit", currentPID},
	}

	sourceActive := true
	for _, cmd := range commands {
		c := &models.Command{
			CommandText:  cmd.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    timestamp,
			SourceApp:    stringPtr("zsh"),
			SourcePid:    &cmd.pid,
			SourceActive: &sourceActive,
		}
		_, err := database.InsertCommand(c)
		require.NoError(t, err)
		timestamp++
	}

	// Set SHY_SESSION_PID environment variable
	os.Setenv("SHY_SESSION_PID", "12345")
	defer os.Unsetenv("SHY_SESSION_PID")

	// Run: shy fc -W /tmp/session.txt -I
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"fc", "-W", outputFile, "-I", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify only current session commands were written (in extended format)
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Len(t, lines, 2)
	assert.Equal(t, ": 1234567890:0;git status", lines[0])
	assert.Equal(t, ": 1234567892:0;git commit", lines[1])

	rootCmd.SetArgs(nil)
}

// Test that using multiple file operation flags together returns an error
func TestFileOp_MutuallyExclusiveFlags(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	testFile := filepath.Join(tempDir, "test.txt")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Test -W and -A together
	t.Run("WriteAndAppend", func(t *testing.T) {
		var buf bytes.Buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)
		rootCmd.SetArgs([]string{"fc", "-W", testFile, "-A", testFile, "--db", dbPath})

		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot use -W, -A, -R, -p, or -P together")

		rootCmd.SetArgs(nil)
	})

	// Test -W and -R together
	t.Run("WriteAndRead", func(t *testing.T) {
		var buf bytes.Buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)
		rootCmd.SetArgs([]string{"fc", "-W", testFile, "-R", testFile, "--db", dbPath})

		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot use -W, -A, -R, -p, or -P together")

		rootCmd.SetArgs(nil)
	})

	// Test -A and -R together
	t.Run("AppendAndRead", func(t *testing.T) {
		var buf bytes.Buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)
		rootCmd.SetArgs([]string{"fc", "-A", testFile, "-R", testFile, "--db", dbPath})

		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot use -W, -A, -R, -p, or -P together")

		rootCmd.SetArgs(nil)
	})

	// Test all three together
	t.Run("AllThreeTogether", func(t *testing.T) {
		var buf bytes.Buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)
		rootCmd.SetArgs([]string{"fc", "-W", testFile, "-A", testFile, "-R", testFile, "--db", dbPath})

		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot use -W, -A, -R, -p, or -P together")

		rootCmd.SetArgs(nil)
	})
}

// Test that using -s and -e together returns an error
func TestFc_QuickExecAndEditorMutuallyExclusive(t *testing.T) {
	defer resetFcFlags(fcCmd)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert a command
	c := &models.Command{
		CommandText: "echo test",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   1234567890,
	}
	_, err = database.InsertCommand(c)
	require.NoError(t, err)

	// Try to use -s and -e together
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"fc", "-s", "-e", "nano", "1", "--db", dbPath})

	err = rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use -s and -e together")

	rootCmd.SetArgs(nil)
}
