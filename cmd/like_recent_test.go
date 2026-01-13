package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

// resetLikeRecentFlags resets all like-recent flags to default values
func resetLikeRecentFlags() {
	likeRecentPwd = false
	likeRecentSession = false
	likeRecentExclude = ""
	likeRecentLimit = 1
	likeRecentIncludeShy = false
}

// TestScenario4_GetMostRecentCommandStartingWithPrefix tests like-recent with matching prefix
func TestScenario4_GetMostRecentCommandStartingWithPrefix(t *testing.T) {
	// Given: I have a database with commands
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"git status", 1704470400},
		{"git commit -m \"a\"", 1704470401},
		{"ls -la", 1704470402},
		{"git push", 1704470403},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy like-recent 'git'"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "git", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()

	// Then: the output should be "git push"
	assert.Equal(t, "git push\n", output, "should output the most recent git command")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestScenario5_LikeRecentWithNoMatches tests like-recent with no matching commands
func TestScenario5_LikeRecentWithNoMatches(t *testing.T) {
	// Given: I have a database with commands
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"ls -la", 1704470400},
		{"pwd", 1704470401},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy like-recent 'docker'"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "docker", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed even with no matches")

	output := buf.String()

	// Then: the output should be empty
	assert.Equal(t, "", output, "output should be empty when no matches found")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestScenario6_LikeRecentIsCaseSensitive tests case-sensitive matching
func TestScenario6_LikeRecentIsCaseSensitive(t *testing.T) {
	// Given: I have a database with commands
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"Git Status", 1704470400},
		{"git status", 1704470401},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy like-recent 'git'"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "git", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()

	// Then: the output should be "git status" (lowercase match)
	assert.Equal(t, "git status\n", output, "should match case-sensitively")
	assert.NotContains(t, output, "Git Status", "should not match different case")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestScenario7_LikeRecentIgnoresShyCommands tests filtering of shy commands
func TestScenario7_LikeRecentIgnoresShyCommands(t *testing.T) {
	// Given: I have a database with commands including shy commands
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"git status", 1704470400},
		{"shy list", 1704470401},
		{"git commit", 1704470402},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// When: I run "shy like-recent 'git'"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "git", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()

	// Then: the output should be "git commit"
	assert.Equal(t, "git commit\n", output, "should output the most recent git command")
	assert.NotContains(t, output, "shy list", "should not include shy commands")

	// Also test that shy commands starting with 'shy' are filtered
	buf.Reset()
	rootCmd.SetArgs([]string{"like-recent", "shy", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output = buf.String()

	// Should be empty because shy commands are filtered out
	assert.Equal(t, "", output, "shy commands should be filtered out")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestLikeRecentWithIncludeShy tests --include-shy flag
func TestLikeRecentWithIncludeShy(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"shy list", 1704470400},
		{"shy list-all", 1704470401},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "shy", "--include-shy", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()
	assert.Equal(t, "shy list-all\n", output, "should include shy commands")

	rootCmd.SetArgs(nil)
	resetLikeRecentFlags()
}

// TestLikeRecentWithLimit tests --limit flag
func TestLikeRecentWithLimit(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"git status", 1704470400},
		{"git add .", 1704470401},
		{"git commit -m \"msg\"", 1704470402},
		{"git push", 1704470403},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "git", "--limit", "3", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	assert.Len(t, lines, 3, "should return exactly 3 results")
	assert.Equal(t, "git push", string(lines[0]), "first should be most recent")
	assert.Equal(t, "git commit -m \"msg\"", string(lines[1]))
	assert.Equal(t, "git add .", string(lines[2]))

	rootCmd.SetArgs(nil)
	resetLikeRecentFlags()
}

// TestLikeRecentWithPwd tests --pwd flag
func TestLikeRecentWithPwd(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	// Create test directories
	proj1 := filepath.Join(tempDir, "proj1")
	proj2 := filepath.Join(tempDir, "proj2")
	os.MkdirAll(proj1, 0755)
	os.MkdirAll(proj2, 0755)

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
		dir       string
	}{
		{"npm install", 1704470400, proj1},
		{"npm test", 1704470401, proj2},
		{"npm build", 1704470402, proj1},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  c.dir,
			ExitStatus:  0,
			Timestamp:   c.timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// Change to proj1 directory
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)

	err = os.Chdir(proj1)
	require.NoError(t, err, "failed to change directory")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "npm", "--pwd", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()
	assert.Equal(t, "npm build\n", output, "should only match commands from proj1")

	rootCmd.SetArgs(nil)
	resetLikeRecentFlags()
}

// TestLikeRecentWithSession tests --session flag
func TestLikeRecentWithSession(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
		pid       int64
	}{
		{"git push origin main", 1704470400, 12345},
		{"git push origin feature", 1704470401, 12346},
		{"git push origin dev", 1704470402, 12345},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp,
			SourcePid:   &c.pid,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// Set session PID
	os.Setenv("SHY_SESSION_PID", "12345")
	defer os.Unsetenv("SHY_SESSION_PID")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "git push", "--session", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()
	assert.Equal(t, "git push origin dev\n", output, "should only match commands from session 12345")
	assert.NotContains(t, output, "feature", "should not include commands from other sessions")

	rootCmd.SetArgs(nil)
	resetLikeRecentFlags()
}

// TestLikeRecentWithExclude tests --exclude flag
func TestLikeRecentWithExclude(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"git pull origin main", 1704470400},
		{"git push origin main", 1704470401},
		{"git push origin dev", 1704470402},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "git", "--exclude", "git pull*", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()
	assert.Equal(t, "git push origin dev\n", output, "should exclude git pull commands")

	rootCmd.SetArgs(nil)
	resetLikeRecentFlags()
}

// TestLikeRecentWithMultipleFilters tests combining multiple filters
func TestLikeRecentWithMultipleFilters(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	// Create test directories
	proj1 := filepath.Join(tempDir, "proj1")
	proj2 := filepath.Join(tempDir, "proj2")
	os.MkdirAll(proj1, 0755)
	os.MkdirAll(proj2, 0755)

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
		dir       string
		pid       int64
	}{
		{"git pull", 1704470400, proj1, 12345},
		{"git push origin main", 1704470401, proj1, 12345},
		{"git push origin dev", 1704470402, proj2, 12345},
		{"git push origin test", 1704470403, proj1, 12346},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  c.dir,
			ExitStatus:  0,
			Timestamp:   c.timestamp,
			SourcePid:   &c.pid,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	// Set up environment
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	err = os.Chdir(proj1)
	require.NoError(t, err, "failed to change directory")
	os.Setenv("SHY_SESSION_PID", "12345")
	defer os.Unsetenv("SHY_SESSION_PID")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "git", "--pwd", "--session", "--exclude", "git pull*", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()
	assert.Equal(t, "git push origin main\n", output, "should apply all filters")

	rootCmd.SetArgs(nil)
	resetLikeRecentFlags()
}

// TestLikeRecentWithSpecialCharacters tests commands with special characters
func TestLikeRecentWithSpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	commands := []struct {
		text      string
		timestamp int64
	}{
		{"echo \"hello world\"", 1704470400},
		{"grep \"pattern\" file.txt", 1704470401},
		{"sed 's/old/new/'", 1704470402},
	}

	for _, c := range commands {
		cmd := &models.Command{
			CommandText: c.text,
			WorkingDir:  "/home/test",
			ExitStatus:  0,
			Timestamp:   c.timestamp,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command")
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "echo \"", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()
	assert.Equal(t, "echo \"hello world\"\n", output, "should handle special characters")

	rootCmd.SetArgs(nil)
}

// TestLikeRecentWithEmptyDatabase tests behavior with empty database
func TestLikeRecentWithEmptyDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	database.Close()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "git", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed with empty database")

	output := buf.String()
	assert.Equal(t, "", output, "should return empty output")

	rootCmd.SetArgs(nil)
}

// TestLikeRecentWithLimitZero tests --limit 0
func TestLikeRecentWithLimitZero(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	cmd := &models.Command{
		CommandText: "git status",
		WorkingDir:  "/home/test",
		ExitStatus:  0,
		Timestamp:   1704470400,
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"like-recent", "git", "--limit", "0", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "like-recent should succeed")

	output := buf.String()
	assert.Equal(t, "", output, "should return empty output with limit 0")

	rootCmd.SetArgs(nil)
	resetLikeRecentFlags()
}
