package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

// TestScenario1_ListRecentCommands tests that list shows all commands
// ordered by timestamp descending
func TestScenario1_ListRecentCommands(t *testing.T) {
	defer resetListFlags()
	// Given: I have a database with commands
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Insert test commands with specific timestamps
	commands := []struct {
		text      string
		dir       string
		status    int
		timestamp int64
	}{
		{"git status", "/home/test", 0, 1704470400},
		{"ls -la", "/home/test", 0, 1704470401},
		{"npm test", "/home/proj", 1, 1704470402},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  c.dir,
			ExitStatus:  c.status,
			Timestamp:   c.timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy list"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "list command should succeed")

	output := buf.String()

	// Then: the output should contain all 3 commands
	assert.Contains(t, output, "git status", "output should contain 'git status'")
	assert.Contains(t, output, "ls -la", "output should contain 'ls -la'")
	assert.Contains(t, output, "npm test", "output should contain 'npm test'")

	// And: the commands should be ordered by timestamp ascending (oldest first, newest last)
	// git status (1704470400) should appear before ls -la (1704470401)
	// ls -la should appear before npm test (1704470402)
	gitIdx := bytes.Index([]byte(output), []byte("git status"))
	lsIdx := bytes.Index([]byte(output), []byte("ls -la"))
	npmIdx := bytes.Index([]byte(output), []byte("npm test"))

	assert.True(t, gitIdx < lsIdx, "git status should appear before ls -la")
	assert.True(t, lsIdx < npmIdx, "ls -la should appear before npm test")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestScenario2_ListWithLimitFlag tests that list respects the -n limit flag
func TestScenario2_ListWithLimitFlag(t *testing.T) {
	defer resetListFlags()
	// Given: I have a database with 10 commands
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Insert 10 commands with incrementing timestamps
	for i := 0; i < 10; i++ {
		cmd := &models.Command{
			CommandText: "echo test" + string(rune('0'+i)),
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   int64(1704470400 + i),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy list -n 8"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "-n", "8", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "list command should succeed")

	output := buf.String()

	// Then: the output should contain 8 commands
	// Count the number of command entries by counting newlines
	lines := bytes.Split([]byte(output), []byte("\n"))
	// Subtract 1 for the trailing newline
	commandCount := len(lines) - 1
	assert.Equal(t, 8, commandCount, "should display exactly 8 commands")

	// And: the commands should be the 8 most recent (but ordered oldest-to-newest)
	// Oldest shown is test2 (timestamp 1704470402), most recent is test9 (timestamp 1704470409)
	assert.Contains(t, output, "echo test2", "should contain test2 (oldest of the 8)")
	assert.Contains(t, output, "echo test8", "should contain test8")
	assert.Contains(t, output, "echo test9", "should contain test9 (most recent)")

	// Should NOT contain the 2 oldest commands overall
	assert.NotContains(t, output, "echo test0", "should not contain test0 (too old)")
	assert.NotContains(t, output, "echo test1", "should not contain test1 (too old)")

	// Verify ordering: test2 should appear before test9
	test2Idx := bytes.Index([]byte(output), []byte("echo test2"))
	test9Idx := bytes.Index([]byte(output), []byte("echo test9"))
	assert.True(t, test2Idx < test9Idx, "test2 should appear before test9")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestScenario3_ListShowsCommandMetadataWithFmtFlag tests that list --fmt
// shows requested columns in tab-separated format
func TestScenario3_ListShowsCommandMetadataWithFmtFlag(t *testing.T) {
	defer resetListFlags()
	// Given: I have a database with a command "git commit -m 'test'" at "/home/project" with exit status 0
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	cmd := &models.Command{
		CommandText: "git commit -m 'test'",
		WorkingDir:  "/home/project",
		ExitStatus:  0,
		Timestamp:   1704470400, // 2024-01-05 11:00:00
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	// When: I run "shy list --fmt=timestamp,status,pwd,cmd"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "--fmt=timestamp,status,pwd,cmd", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "list command should succeed")

	output := buf.String()

	// Then: the output should show the command text
	assert.Contains(t, output, "git commit -m 'test'", "should show command text")

	// And: the output should show the working directory
	assert.Contains(t, output, "/home/project", "should show working directory")

	// And: the output should show the exit status
	assert.Contains(t, output, "0", "should show exit status")

	// And: the output should show a readable timestamp
	assert.Contains(t, output, "2024-01-05 11:00:00", "should show readable timestamp")

	// And: the columns should be tab separated
	assert.Contains(t, output, "\t", "should have tab-separated columns")

	// Verify the exact format: timestamp\tstatus\tpwd\tcmd
	expectedLine := "2024-01-05 11:00:00\t0\t/home/project\tgit commit -m 'test'"
	assert.Contains(t, output, expectedLine, "should match expected tab-separated format")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// Phase 3: Duration in list command

// Scenario 17a: List command includes duration in seconds in format string
func TestDurationScenario17a_ListCommandIncludesDurationInSeconds(t *testing.T) {
	defer resetListFlags()
	// Given: I have commands with durations
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Test cases from the spec
	testCases := []struct {
		command    string
		durationMs int64
		expected   string
	}{
		{"sleep 2", 2028, "2s"},
		{"sleep 2.7", 2528, "2s"},             // Rounds down
		{"echo hello", 500, "0s"},             // Under 1 second
		{"sleep 72", 72028, "1m12s"},          // 72 seconds
		{"minsleep 72", 4320028, "1h12m0s"},   // 72 minutes
		{"hrsleep 28", 100800028, "1d4h0m0s"}, // 28 hours
	}

	for _, tc := range testCases {
		dur := tc.durationMs
		cmd := &models.Command{
			CommandText: tc.command,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   time.Now().Unix(),
			Duration:    &dur,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy list --fmt 'cmd,durs'"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "--fmt", "cmd,durs", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: each line should show command text and duration
	require.Equal(t, len(testCases), len(lines), "should have correct number of lines")

	// And: duration should be in human-readable format
	for i, tc := range testCases {
		assert.Contains(t, lines[i], tc.command, "line %d should contain command", i)
		assert.Contains(t, lines[i], tc.expected, "line %d should contain duration %s", i, tc.expected)
	}

	rootCmd.SetArgs(nil)
}

// Scenario 17b: List command includes duration in milliseconds in format string
func TestDurationScenario17b_ListCommandIncludesDurationInMilliseconds(t *testing.T) {
	defer resetListFlags()
	// Given: I have commands with durations
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Test cases from the spec
	testCases := []struct {
		command    string
		durationMs int64
		expected   string
	}{
		{"sleep 2", 2028, "2s28ms"},
		{"sleep 2.7", 2528, "2s528ms"},            // With milliseconds
		{"echo hello", 500, "500ms"},              // Just milliseconds
		{"sleep 72", 72028, "1m12s28ms"},          // 72 seconds + 28ms
		{"minsleep 72", 4320028, "1h12m0s28ms"},   // 72 minutes + 28ms
		{"hrsleep 28", 100800028, "1d4h0m0s28ms"}, // 28 hours + 28ms
	}

	for _, tc := range testCases {
		dur := tc.durationMs
		cmd := &models.Command{
			CommandText: tc.command,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   time.Now().Unix(),
			Duration:    &dur,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy list --fmt 'cmd,durms'"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "--fmt", "cmd,durms", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Then: each line should show command text and duration with milliseconds
	require.Equal(t, len(testCases), len(lines), "should have correct number of lines")

	// And: duration should be in human-readable format with milliseconds
	for i, tc := range testCases {
		assert.Contains(t, lines[i], tc.command, "line %d should contain command", i)
		assert.Contains(t, lines[i], tc.expected, "line %d should contain duration %s", i, tc.expected)
	}

	rootCmd.SetArgs(nil)
}

func TestListWithCwd(t *testing.T) {
	defer resetListFlags()
	// Given: I have commands with durations
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	proj1 := filepath.Join(tempDir, "proj1")
	proj2 := filepath.Join(tempDir, "proj2")
	os.MkdirAll(proj1, 0755)
	os.MkdirAll(proj2, 0755)

	// Test cases from the spec
	testCases := []struct {
		command    string
		workingDir string
	}{
		{"echo 1", proj1},
		{"echo 2", proj1},
		{"echo 3", proj2},
		{"echo 4", proj2},
		{"echo 5", proj2},
		{"echo 6", proj1},
	}

	for _, tc := range testCases {
		cmd := &models.Command{
			CommandText: tc.command,
			WorkingDir:  tc.workingDir,
			ExitStatus:  0,
			Timestamp:   time.Now().Unix(),
			Duration:    int64Ptr(1),
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// Change to proj1 directory
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)

	err = os.Chdir(proj2)
	require.NoError(t, err, "failed to change directory")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "--pwd", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	require.Equal(t, 3, len(lines), "should have correct number of lines")
	assert.Equal(t, "echo 3\necho 4\necho 5\n", output, "should have correct output")

	rootCmd.SetArgs(nil)
}

func resetListFlags() {
	listLimit = 20
	listFormat = ""
	listToday = false
	listYesterday = false
	listThisWeek = false
	listLastWeek = false
	listSession = ""
	listCurrentSession = false
	listPwd = false
}

// TestListWithSessionAppOnly tests filtering by app name only (without pid)
func TestListWithSessionAppOnly(t *testing.T) {
	defer resetListFlags()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert commands from zsh sessions (different pids)
	zshApp := "zsh"
	zshPid1 := int64(12345)
	zshPid2 := int64(67890)
	active := true

	zshCommands := []struct {
		text string
		pid  int64
		ts   int64
	}{
		{"zsh cmd 1", zshPid1, 1704470400},
		{"zsh cmd 2", zshPid1, 1704470401},
		{"zsh cmd 3", zshPid2, 1704470402},
	}

	for _, c := range zshCommands {
		cmd := &models.Command{
			CommandText:  c.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    c.ts,
			SourceApp:    &zshApp,
			SourcePid:    &c.pid,
			SourceActive: &active,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// Insert commands from bash session
	bashApp := "bash"
	bashPid := int64(11111)

	bashCommands := []struct {
		text string
		ts   int64
	}{
		{"bash cmd 1", 1704470403},
		{"bash cmd 2", 1704470404},
	}

	for _, c := range bashCommands {
		cmd := &models.Command{
			CommandText:  c.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    c.ts,
			SourceApp:    &bashApp,
			SourcePid:    &bashPid,
			SourceActive: &active,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy list --session zsh" (app only, no pid)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "--session", "zsh", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: should contain all zsh commands from both pids
	assert.Contains(t, output, "zsh cmd 1")
	assert.Contains(t, output, "zsh cmd 2")
	assert.Contains(t, output, "zsh cmd 3")

	// And: should NOT contain bash commands
	assert.NotContains(t, output, "bash cmd")

	// Count lines (should be 3 zsh commands)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 3, len(lines), "should have exactly 3 zsh commands")

	rootCmd.SetArgs(nil)
}

// TestListWithSessionAppAndPid tests filtering by both app and pid
func TestListWithSessionAppAndPid(t *testing.T) {
	defer resetListFlags()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Insert commands from two different zsh pids
	zshApp := "zsh"
	zshPid1 := int64(12345)
	zshPid2 := int64(67890)
	active := true

	commands := []struct {
		text string
		pid  int64
		ts   int64
	}{
		{"pid1 cmd 1", zshPid1, 1704470400},
		{"pid1 cmd 2", zshPid1, 1704470401},
		{"pid2 cmd 1", zshPid2, 1704470402},
		{"pid2 cmd 2", zshPid2, 1704470403},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText:  c.text,
			WorkingDir:   "/home/test",
			ExitStatus:   0,
			Timestamp:    c.ts,
			SourceApp:    &zshApp,
			SourcePid:    &c.pid,
			SourceActive: &active,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: I run "shy list --session zsh:12345" (specific pid)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "--session", "zsh:12345", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()

	// Then: should contain only pid1 commands
	assert.Contains(t, output, "pid1 cmd 1")
	assert.Contains(t, output, "pid1 cmd 2")

	// And: should NOT contain pid2 commands
	assert.NotContains(t, output, "pid2 cmd")

	// Count lines (should be 2 commands)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 2, len(lines), "should have exactly 2 commands from pid 12345")

	rootCmd.SetArgs(nil)
}
