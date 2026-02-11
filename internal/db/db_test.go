package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/pkg/models"
)

// TestScenario1_DatabaseInitializationOnFirstInsert tests that the database
// is created with the correct schema when inserting the first command
func TestScenario1_DatabaseInitializationOnFirstInsert(t *testing.T) {
	// Given: no existing shy database exists
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	// Verify database doesn't exist yet
	_, err := os.Stat(dbPath)
	require.True(t, os.IsNotExist(err), "database should not exist yet")

	// When: I run "shy insert --command 'ls -la' --dir '/home/user/projects' --status 0"
	database, err := NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	cmd := models.NewCommand("ls -la", "/home/user/projects", 0)
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	// Then: a SQLite database should be created
	_, err = os.Stat(dbPath)
	require.NoError(t, err, "database file should exist")
	assert.Equal(t, dbPath, database.Path(), "database path should match")

	// And: the database should contain a "commands" table
	exists, err := database.TableExists()
	require.NoError(t, err, "failed to check table existence")
	assert.True(t, exists, "commands table should exist")

	// And: the "commands" table should have the correct columns
	schema, err := database.GetTableSchema()
	require.NoError(t, err, "failed to get table schema")

	expectedColumns := []struct {
		name string
		typ  string
	}{
		{"id", "INTEGER"},
		{"timestamp", "INTEGER"},
		{"exit_status", "INTEGER"},
		{"duration", "INTEGER"},
		{"command_text", "TEXT"},
		{"working_dir_id", "INTEGER"},
		{"git_context_id", "INTEGER"},
		{"source_id", "INTEGER"},
		{"is_duplicate", "INTEGER"},
	}

	require.Len(t, schema, len(expectedColumns), "should have correct number of columns")

	for i, expected := range expectedColumns {
		assert.Equal(t, expected.name, schema[i]["name"], "column %d name should match", i)
		assert.Equal(t, expected.typ, schema[i]["type"], "column %d type should match", i)
	}

	// And: a new record should be inserted with the provided values
	count, err := database.CountCommands()
	require.NoError(t, err, "failed to count commands")
	assert.Equal(t, 1, count, "should have one command in database")
}

// TestScenario2_InsertSimpleCommandWithoutGitContext tests inserting a command
// without git context
func TestScenario2_InsertSimpleCommandWithoutGitContext(t *testing.T) {
	// Given: the shy database exists
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// When: I run "shy insert --command 'ls -la' --dir '/home/user/projects'"
	cmd := models.NewCommand("ls -la", "/home/user/projects", 0)
	id, err := database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	// Then: a new record should be inserted into the commands table
	retrievedCmd, err := database.GetCommand(id)
	require.NoError(t, err, "failed to retrieve command")

	// And: the record should have command_text "ls -la"
	assert.Equal(t, "ls -la", retrievedCmd.CommandText, "command text should match")

	// And: the record should have working_dir "/home/user/projects"
	assert.Equal(t, "/home/user/projects", retrievedCmd.WorkingDir, "working dir should match")

	// And: the record should have exit_status 0
	assert.Equal(t, 0, retrievedCmd.ExitStatus, "exit status should be 0")

	// And: the record should have git_repo NULL
	assert.Nil(t, retrievedCmd.GitRepo, "git repo should be NULL")

	// And: the record should have git_branch NULL
	assert.Nil(t, retrievedCmd.GitBranch, "git branch should be NULL")

	// And: the timestamp should be within 1 second of the current time
	now := cmd.Timestamp
	assert.InDelta(t, now, retrievedCmd.Timestamp, 1, "timestamp should be within 1 second")
}

// TestScenario3_InsertCommandWithGitContext tests inserting a command
// with explicit git context
func TestScenario3_InsertCommandWithGitContext(t *testing.T) {
	// Given: the shy database exists
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// When: I run "shy insert --command 'git status' --dir '/home/user/myproject'
	//       --git-repo 'https://github.com/user/myproject.git' --git-branch 'feature/new-feature'"
	cmd := models.NewCommand("git status", "/home/user/myproject", 0)
	repo := "https://github.com/user/myproject.git"
	branch := "feature/new-feature"
	cmd.GitRepo = &repo
	cmd.GitBranch = &branch

	id, err := database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	// Then: a new record should be inserted into the commands table
	retrievedCmd, err := database.GetCommand(id)
	require.NoError(t, err, "failed to retrieve command")

	// And: the record should have the correct values
	assert.Equal(t, "git status", retrievedCmd.CommandText, "command text should match")
	assert.Equal(t, "/home/user/myproject", retrievedCmd.WorkingDir, "working dir should match")
	assert.Equal(t, 0, retrievedCmd.ExitStatus, "exit status should be 0")
	require.NotNil(t, retrievedCmd.GitRepo, "git repo should not be NULL")
	assert.Equal(t, "https://github.com/user/myproject.git", *retrievedCmd.GitRepo, "git repo should match")
	require.NotNil(t, retrievedCmd.GitBranch, "git branch should not be NULL")
	assert.Equal(t, "feature/new-feature", *retrievedCmd.GitBranch, "git branch should match")
}

// TestScenario4_InsertCommandWithNonZeroExitStatus tests inserting a command
// with a non-zero exit status
func TestScenario4_InsertCommandWithNonZeroExitStatus(t *testing.T) {
	// Given: the shy database exists
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// When: I run "shy insert --command 'grep nonexistent file.txt' --dir '/home/user/projects' --status 1"
	cmd := models.NewCommand("grep nonexistent file.txt", "/home/user/projects", 1)
	id, err := database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	// Then: a new record should be inserted into the commands table
	retrievedCmd, err := database.GetCommand(id)
	require.NoError(t, err, "failed to retrieve command")

	// And: the record should have command_text "grep nonexistent file.txt"
	assert.Equal(t, "grep nonexistent file.txt", retrievedCmd.CommandText, "command text should match")

	// And: the record should have exit_status 1
	assert.Equal(t, 1, retrievedCmd.ExitStatus, "exit status should be 1")
}

// TestScenario5_InsertMultipleCommandsSequentially tests inserting multiple
// commands in sequence
func TestScenario5_InsertMultipleCommandsSequentially(t *testing.T) {
	// Given: the shy database exists and has 0 command records
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	count, err := database.CountCommands()
	require.NoError(t, err, "failed to count commands")
	require.Equal(t, 0, count, "database should start empty")

	// When: I run the following commands
	commands := []struct {
		text   string
		status int
	}{
		{"pwd", 0},
		{"echo test", 0},
		{"false", 1},
		{"true", 0},
	}

	var ids []int64
	for _, cmdData := range commands {
		cmd := models.NewCommand(cmdData.text, "/home/user", cmdData.status)
		id, err := database.InsertCommand(cmd)
		require.NoError(t, err, "failed to insert command %s", cmdData.text)
		ids = append(ids, id)
	}

	// Then: the database should have 4 command records
	count, err = database.CountCommands()
	require.NoError(t, err, "failed to count commands")
	assert.Equal(t, 4, count, "should have 4 commands in database")

	// And: each record should have a unique id
	uniqueIDs := make(map[int64]bool)
	for _, id := range ids {
		assert.False(t, uniqueIDs[id], "ID %d should be unique", id)
		uniqueIDs[id] = true
	}
	assert.Len(t, uniqueIDs, 4, "should have 4 unique IDs")

	// And: the records should be ordered by timestamp ascending
	for i := 1; i < len(ids); i++ {
		cmd1, err := database.GetCommand(ids[i-1])
		require.NoError(t, err, "failed to get command %d", ids[i-1])
		cmd2, err := database.GetCommand(ids[i])
		require.NoError(t, err, "failed to get command %d", ids[i])
		assert.LessOrEqual(t, cmd1.Timestamp, cmd2.Timestamp, "commands should be ordered by timestamp")
	}
}

// TestScenario6_InsertCommandWithSpecialCharacters tests that special characters
// are properly handled
func TestScenario6_InsertCommandWithSpecialCharacters(t *testing.T) {
	// Given: the shy database exists
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// When: I run "shy insert --command 'echo \"Hello \\\"World\\\"\"' --dir '/home/user' --status 0"
	cmdText := `echo "Hello \"World\""`
	cmd := models.NewCommand(cmdText, "/home/user", 0)
	id, err := database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	// Then: a new record should be inserted into the commands table
	retrievedCmd, err := database.GetCommand(id)
	require.NoError(t, err, "failed to retrieve command")

	// And: the record should have command_text with special characters preserved
	assert.Equal(t, cmdText, retrievedCmd.CommandText, "special characters should be preserved")
}

// TestScenario7_InsertVeryLongCommand tests that very long commands are
// captured completely
func TestScenario7_InsertVeryLongCommand(t *testing.T) {
	// Given: the shy database exists
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// When: I run "shy insert" with a command text that is 1000 characters long
	longCmd := ""
	for i := 0; i < 1000; i++ {
		longCmd += "a"
	}

	cmd := models.NewCommand(longCmd, "/home/user", 0)
	id, err := database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	// Then: a new record should be inserted into the commands table
	retrievedCmd, err := database.GetCommand(id)
	require.NoError(t, err, "failed to retrieve command")

	// And: the record should have the complete command_text with all 1000 characters
	assert.Len(t, retrievedCmd.CommandText, 1000, "command text should be 1000 characters")
	assert.Equal(t, longCmd, retrievedCmd.CommandText, "command text should match exactly")
}

// TestGetCommandsByDateRange tests retrieving commands within a timestamp range
func TestGetCommandsByDateRange(t *testing.T) {
	// Given: the shy database exists with commands at different timestamps
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Insert commands with timestamps representing 2026-01-14
	pid := int64Ptr(12345)

	// Morning: 8:00 AM (timestamp: 1736841600)
	cmd1 := models.NewCommand("git status", "/home/user/projects/shy", 0)
	cmd1.Timestamp = 1736841600
	cmd1.SourceApp = stringPtr("zsh")
	cmd1.SourcePid = pid
	_, err = database.InsertCommand(cmd1)
	require.NoError(t, err, "failed to insert command 1")

	// Morning: 9:00 AM (timestamp: 1736845200)
	cmd2 := models.NewCommand("go build", "/home/user/projects/shy", 0)
	cmd2.Timestamp = 1736845200
	cmd2.SourceApp = stringPtr("zsh")
	cmd2.SourcePid = pid
	_, err = database.InsertCommand(cmd2)
	require.NoError(t, err, "failed to insert command 2")

	// Afternoon: 2:00 PM (timestamp: 1736863200)
	cmd3 := models.NewCommand("go test", "/home/user/projects/shy", 0)
	cmd3.Timestamp = 1736863200
	cmd3.SourceApp = stringPtr("zsh")
	cmd3.SourcePid = pid
	_, err = database.InsertCommand(cmd3)
	require.NoError(t, err, "failed to insert command 3")

	// Next day: 2026-01-15 8:00 AM (timestamp: 1736928000)
	cmd4 := models.NewCommand("git pull", "/home/user/projects/shy", 0)
	cmd4.SourceApp = stringPtr("zsh")
	cmd4.SourcePid = pid
	cmd4.Timestamp = 1736928000
	_, err = database.InsertCommand(cmd4)
	require.NoError(t, err, "failed to insert command 4")

	// When: I query commands for 2026-01-14 (start: 1736812800, end: 1736899200)
	startOfDay := int64(1736812800) // 2026-01-14 00:00:00 UTC
	endOfDay := int64(1736899200)   // 2026-01-15 00:00:00 UTC
	commands, err := database.GetCommandsByDateRange(startOfDay, endOfDay, stringPtr("zsh"))
	require.NoError(t, err, "failed to get commands by date range")

	// Then: should return only the 3 commands from 2026-01-14
	assert.Len(t, commands, 3, "should have 3 commands from 2026-01-14")

	// And: commands should be ordered by timestamp ascending
	assert.Equal(t, "git status", commands[0].CommandText)
	assert.Equal(t, int64(1736841600), commands[0].Timestamp)

	assert.Equal(t, "go build", commands[1].CommandText)
	assert.Equal(t, int64(1736845200), commands[1].Timestamp)

	assert.Equal(t, "go test", commands[2].CommandText)
	assert.Equal(t, int64(1736863200), commands[2].Timestamp)
}

// TestGetCommandsByDateRange_EmptyResult tests querying when no commands exist in range
func TestGetCommandsByDateRange_EmptyResult(t *testing.T) {
	// Given: the shy database exists with no commands
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// When: I query commands for any date range
	startOfDay := int64(1736812800)
	endOfDay := int64(1736899200)
	commands, err := database.GetCommandsByDateRange(startOfDay, endOfDay, nil)
	require.NoError(t, err, "failed to get commands by date range")

	// Then: should return empty slice
	assert.Empty(t, commands, "should return empty slice when no commands in range")
}

// TestGetCommandsByDateRange_BoundaryConditions tests exact boundary matching
func TestGetCommandsByDateRange_BoundaryConditions(t *testing.T) {
	// Given: the shy database exists with commands at boundary times
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Insert command exactly at start time
	cmdStart := models.NewCommand("at start", "/home/user", 0)
	cmdStart.Timestamp = 1736812800 // Exactly 00:00:00
	_, err = database.InsertCommand(cmdStart)
	require.NoError(t, err, "failed to insert command at start")

	// Insert command exactly at end time
	cmdEnd := models.NewCommand("at end", "/home/user", 0)
	cmdEnd.Timestamp = 1736899200 // Exactly 00:00:00 next day
	_, err = database.InsertCommand(cmdEnd)
	require.NoError(t, err, "failed to insert command at end")

	// Insert command one second before end
	cmdBeforeEnd := models.NewCommand("before end", "/home/user", 0)
	cmdBeforeEnd.Timestamp = 1736899199 // 23:59:59
	_, err = database.InsertCommand(cmdBeforeEnd)
	require.NoError(t, err, "failed to insert command before end")

	// When: I query with range [start, end)
	commands, err := database.GetCommandsByDateRange(1736812800, 1736899200, nil)
	require.NoError(t, err, "failed to get commands by date range")

	// Then: should include start time (inclusive) but exclude end time (exclusive)
	assert.Len(t, commands, 2, "should include start and before-end, but not end")
	assert.Equal(t, "at start", commands[0].CommandText)
	assert.Equal(t, "before end", commands[1].CommandText)
}

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

// TestGetCommandsByRange tests that GetCommandsByRange returns unique commands
func TestGetCommandsByRange(t *testing.T) {
	t.Run("returns unique commands by deduplication", func(t *testing.T) {
		// Given: database with duplicate commands
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert commands with duplicates
		commands := []string{
			"echo hello",
			"ls -la",
			"echo hello", // duplicate
			"pwd",
			"ls -la",     // duplicate
			"echo hello", // duplicate
		}

		for _, cmdText := range commands {
			cmd := models.NewCommand(cmdText, "/home/user", 0)
			_, err := database.InsertCommand(cmd)
			require.NoError(t, err)
		}

		// When: GetCommandsByRange is called
		results, err := database.GetCommandsByRange(1, 6)
		require.NoError(t, err)

		// Then: should return only unique commands (most recent occurrence)
		assert.Len(t, results, 3, "should return 3 unique commands")

		// And: should contain the unique command texts
		cmdTexts := make(map[string]bool)
		for _, cmd := range results {
			cmdTexts[cmd.CommandText] = true
		}
		assert.True(t, cmdTexts["echo hello"])
		assert.True(t, cmdTexts["ls -la"])
		assert.True(t, cmdTexts["pwd"])

		// And: should return the most recent ID for each unique command
		// "echo hello" appears at IDs 1, 3, 6 - should get ID 6
		// "ls -la" appears at IDs 2, 5 - should get ID 5
		// "pwd" appears at ID 4 - should get ID 4
		idMap := make(map[string]int64)
		for _, cmd := range results {
			idMap[cmd.CommandText] = cmd.ID
		}
		assert.Equal(t, int64(6), idMap["echo hello"], "should get most recent 'echo hello'")
		assert.Equal(t, int64(5), idMap["ls -la"], "should get most recent 'ls -la'")
		assert.Equal(t, int64(4), idMap["pwd"], "should get 'pwd'")
	})

	t.Run("handles empty range", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// When: called with invalid range (first > last)
		results, err := database.GetCommandsByRange(10, 5)
		require.NoError(t, err)

		// Then: should return empty slice
		assert.Empty(t, results)
	})

	t.Run("handles range with no duplicates", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert unique commands
		for i := 1; i <= 5; i++ {
			cmd := models.NewCommand("command"+string(rune('0'+i)), "/home/user", 0)
			_, err := database.InsertCommand(cmd)
			require.NoError(t, err)
		}

		// When: GetCommandsByRange is called
		results, err := database.GetCommandsByRange(1, 5)
		require.NoError(t, err)

		// Then: should return all 5 commands
		assert.Len(t, results, 5)
	})
}

// TestGetCommandsByRangeWithPattern tests pattern matching with deduplication
func TestGetCommandsByRangeWithPattern(t *testing.T) {
	t.Run("returns unique commands matching pattern", func(t *testing.T) {
		// Given: database with duplicate commands
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert commands
		commands := []string{
			"git status",
			"git commit -m 'test'",
			"git status", // duplicate
			"ls -la",
			"git push",
			"git status", // duplicate
		}

		for _, cmdText := range commands {
			cmd := models.NewCommand(cmdText, "/home/user", 0)
			_, err := database.InsertCommand(cmd)
			require.NoError(t, err)
		}

		// When: GetCommandsByRangeWithPattern is called with pattern "git%"
		results, err := database.GetCommandsByRangeWithPattern(1, 6, "git%")
		require.NoError(t, err)

		// Then: should return only unique git commands
		assert.Len(t, results, 3, "should return 3 unique git commands")

		// And: should contain the unique git command texts
		cmdTexts := make(map[string]bool)
		for _, cmd := range results {
			cmdTexts[cmd.CommandText] = true
		}
		assert.True(t, cmdTexts["git status"])
		assert.True(t, cmdTexts["git commit -m 'test'"])
		assert.True(t, cmdTexts["git push"])

		// And: should get most recent ID for "git status" (ID 6)
		for _, cmd := range results {
			if cmd.CommandText == "git status" {
				assert.Equal(t, int64(6), cmd.ID, "should get most recent 'git status'")
			}
		}
	})

	t.Run("handles pattern with no matches", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		cmd := models.NewCommand("echo hello", "/home/user", 0)
		_, err = database.InsertCommand(cmd)
		require.NoError(t, err)

		// When: pattern doesn't match any commands
		results, err := database.GetCommandsByRangeWithPattern(1, 1, "git%")
		require.NoError(t, err)

		// Then: should return empty slice
		assert.Empty(t, results)
	})
}

// TestGetCommandsByRangeInternal tests session filtering with deduplication
func TestGetCommandsByRangeInternal(t *testing.T) {
	t.Run("returns unique commands for specific session", func(t *testing.T) {
		// Given: database with commands from multiple sessions
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert commands from session 1000
		for _, cmdText := range []string{"echo session1", "ls", "echo session1"} {
			cmd := models.NewCommand(cmdText, "/home/user", 0)
			cmd.SourceApp = stringPtr("zsh")
			cmd.SourcePid = int64Ptr(1000)
			cmd.SourceActive = boolPtr(true)
			_, err := database.InsertCommand(cmd)
			require.NoError(t, err)
		}

		// Insert commands from session 2000
		for _, cmdText := range []string{"echo session2", "pwd", "echo session2"} {
			cmd := models.NewCommand(cmdText, "/home/user", 0)
			cmd.SourceApp = stringPtr("zsh")
			cmd.SourcePid = int64Ptr(2000)
			cmd.SourceActive = boolPtr(true)
			_, err := database.InsertCommand(cmd)
			require.NoError(t, err)
		}

		// When: GetCommandsByRangeInternal is called for session 1000
		results, err := database.GetCommandsByRangeInternal(1, 6, 1000)
		require.NoError(t, err)

		// Then: should return only unique commands from session 1000
		assert.Len(t, results, 2, "should return 2 unique commands from session 1000")

		// And: should contain the correct commands
		cmdTexts := make(map[string]int)
		for _, cmd := range results {
			cmdTexts[cmd.CommandText]++
		}
		assert.Equal(t, 1, cmdTexts["echo session1"], "should have one 'echo session1'")
		assert.Equal(t, 1, cmdTexts["ls"], "should have one 'ls'")
		assert.Equal(t, 0, cmdTexts["echo session2"], "should not have 'echo session2'")
	})

	t.Run("respects source_active flag", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert active command
		cmd1 := models.NewCommand("echo active", "/home/user", 0)
		cmd1.SourceApp = stringPtr("zsh")
		cmd1.SourcePid = int64Ptr(1000)
		cmd1.SourceActive = boolPtr(true)
		_, err = database.InsertCommand(cmd1)
		require.NoError(t, err)

		// Insert inactive command
		cmd2 := models.NewCommand("echo inactive", "/home/user", 0)
		cmd2.SourceApp = stringPtr("zsh")
		cmd2.SourcePid = int64Ptr(1000)
		cmd2.SourceActive = boolPtr(false)
		_, err = database.InsertCommand(cmd2)
		require.NoError(t, err)

		// When: GetCommandsByRangeInternal is called
		results, err := database.GetCommandsByRangeInternal(1, 2, 1000)
		require.NoError(t, err)

		// Then: should only return active command
		assert.Len(t, results, 1)
		assert.Equal(t, "echo active", results[0].CommandText)
	})
}

// TestGetCommandsByRangeWithPatternInternal tests pattern + session filtering with deduplication
func TestGetCommandsByRangeWithPatternInternal(t *testing.T) {
	t.Run("returns unique commands matching pattern for session", func(t *testing.T) {
		// Given: database with commands from multiple sessions
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Session 1000: git commands
		for _, cmdText := range []string{"git status", "ls", "git commit", "git status"} {
			cmd := models.NewCommand(cmdText, "/home/user", 0)
			cmd.SourceApp = stringPtr("zsh")
			cmd.SourcePid = int64Ptr(1000)
			cmd.SourceActive = boolPtr(true)
			_, err := database.InsertCommand(cmd)
			require.NoError(t, err)
		}

		// Session 2000: different commands
		cmd := models.NewCommand("git push", "/home/user", 0)
		cmd.SourceApp = stringPtr("zsh")
		cmd.SourcePid = int64Ptr(2000)
		cmd.SourceActive = boolPtr(true)
		_, err = database.InsertCommand(cmd)
		require.NoError(t, err)

		// When: GetCommandsByRangeWithPatternInternal is called for session 1000 with pattern "git%"
		results, err := database.GetCommandsByRangeWithPatternInternal(1, 5, 1000, "git%")
		require.NoError(t, err)

		// Then: should return unique git commands from session 1000 only
		assert.Len(t, results, 2, "should return 2 unique git commands from session 1000")

		cmdTexts := make(map[string]bool)
		for _, cmd := range results {
			cmdTexts[cmd.CommandText] = true
		}
		assert.True(t, cmdTexts["git status"])
		assert.True(t, cmdTexts["git commit"])
		assert.False(t, cmdTexts["git push"], "should not include git push from session 2000")
	})
}

// TestListCommandsInRange tests timestamp-based listing including duplicates
func TestListCommandsInRange(t *testing.T) {
	t.Run("returns all commands in timestamp range including duplicates", func(t *testing.T) {
		// Given: database with duplicate commands at different times
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert commands with timestamps
		commands := []struct {
			text      string
			timestamp int64
		}{
			{"echo hello", 1000},
			{"ls -la", 2000},
			{"echo hello", 3000}, // duplicate
			{"pwd", 4000},
			{"ls -la", 5000},     // duplicate
			{"echo hello", 6000}, // duplicate
		}

		for _, cmdData := range commands {
			cmd := models.NewCommand(cmdData.text, "/home/user", 0)
			cmd.Timestamp = cmdData.timestamp
			_, err := database.InsertCommand(cmd)
			require.NoError(t, err)
		}

		// When: ListCommandsInRange is called
		results, err := database.ListCommandsInRange(1000, 6000, 0, "", 0, "")
		require.NoError(t, err)

		// Then: should return all commands including duplicates
		assert.Len(t, results, 6, "should return 6 commands including duplicates")

		// And: commands should be ordered by timestamp
		assert.Equal(t, "echo hello", results[0].CommandText)
		assert.Equal(t, int64(1000), results[0].Timestamp)
		assert.Equal(t, "ls -la", results[1].CommandText)
		assert.Equal(t, int64(2000), results[1].Timestamp)
		assert.Equal(t, "echo hello", results[2].CommandText)
		assert.Equal(t, int64(3000), results[2].Timestamp)
	})

	t.Run("returns all commands with limit including duplicates", func(t *testing.T) {
		// Given: database with many duplicate commands
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert 10 commands: 5 unique, each repeated twice
		for i := 0; i < 10; i++ {
			cmd := models.NewCommand("cmd"+string(rune('0'+(i%5))), "/home/user", 0)
			cmd.Timestamp = int64(1000 + i*1000)
			_, err := database.InsertCommand(cmd)
			require.NoError(t, err)
		}

		// When: ListCommandsInRange is called with limit 10
		results, err := database.ListCommandsInRange(0, 0, 10, "", 0, "")
		require.NoError(t, err)

		// Then: should return all 10 commands including duplicates
		assert.Equal(t, 10, len(results), "should return all 10 commands")

		// And: commands should be ordered by timestamp
		for i := 0; i < 10; i++ {
			assert.Equal(t, int64(1000+i*1000), results[i].Timestamp, "command %d should have correct timestamp", i)
		}
	})

	t.Run("handles timestamp boundaries", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert commands outside and inside range
		cmd1 := models.NewCommand("before", "/home/user", 0)
		cmd1.Timestamp = 500
		_, err = database.InsertCommand(cmd1)
		require.NoError(t, err)

		cmd2 := models.NewCommand("inside", "/home/user", 0)
		cmd2.Timestamp = 1500
		_, err = database.InsertCommand(cmd2)
		require.NoError(t, err)

		cmd3 := models.NewCommand("after", "/home/user", 0)
		cmd3.Timestamp = 2500
		_, err = database.InsertCommand(cmd3)
		require.NoError(t, err)

		// When: ListCommandsInRange is called with specific range
		results, err := database.ListCommandsInRange(1000, 2000, 0, "", 0, "")
		require.NoError(t, err)

		// Then: should only return commands in range
		assert.Len(t, results, 1)
		assert.Equal(t, "inside", results[0].CommandText)
	})
}

// TestGetRecentCommandsWithoutConsecutiveDuplicates_UnionWithDirectory tests that
// session results are unioned with directory results when both filters are provided
func TestGetRecentCommandsWithoutConsecutiveDuplicates_UnionWithDirectory(t *testing.T) {
	// Given: database with commands from different sessions and directories
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Session commands (zsh:12345) in /session-dir
	sourceApp1 := "zsh"
	sourcePid1 := int64(12345)
	sourceActive1 := true

	sessionCommands := []struct {
		text      string
		timestamp int64
	}{
		{"session 1", 100},
		{"session 2", 200},
		{"session 3", 300},
	}

	for _, c := range sessionCommands {
		cmd := &models.Command{
			CommandText:  c.text,
			WorkingDir:   "/session-dir",
			ExitStatus:   0,
			Timestamp:    c.timestamp,
			SourceApp:    &sourceApp1,
			SourcePid:    &sourcePid1,
			SourceActive: &sourceActive1,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// Directory commands (bash:99999) in /target-dir
	sourceApp2 := "bash"
	sourcePid2 := int64(99999)
	sourceActive2 := true

	dirCommands := []struct {
		text      string
		timestamp int64
	}{
		{"dir 1", 400},
		{"dir 2", 500},
		{"dir 3", 600},
		{"dir 4", 700},
		{"dir 5", 800},
	}

	for _, c := range dirCommands {
		cmd := &models.Command{
			CommandText:  c.text,
			WorkingDir:   "/target-dir",
			ExitStatus:   0,
			Timestamp:    c.timestamp,
			SourceApp:    &sourceApp2,
			SourcePid:    &sourcePid2,
			SourceActive: &sourceActive2,
		}
		_, err := database.InsertCommand(cmd)
		require.NoError(t, err)
	}

	// When: GetRecentCommandsWithoutConsecutiveDuplicates called with session filter and working dir
	// Collect results for offsets 0-5
	var results []models.Command
	for offset := 0; offset < 6; offset++ {
		result, err := database.GetRecentCommandsWithoutConsecutiveDuplicates(offset, "zsh", 12345, "/target-dir")
		require.NoError(t, err)
		require.NotNil(t, result, "offset %d should return a command", offset)
		results = append(results, *result)
	}

	// Then: should return session command first (most recent matching source_id), then directory commands
	// Note: With normalized schema and no unique constraint on sources, each command gets its own source_id.
	// The source lookup returns only the most recent source_id, so only the last session command matches.
	require.Len(t, results, 6, "should return 1 session + 5 dir commands")

	// Session command (only the most recent one matches the looked-up source_id)
	assert.Equal(t, "session 3", results[0].CommandText, "1st should be most recent session command")

	// Directory commands (reverse chronological)
	assert.Equal(t, "session 2", results[1].CommandText, "2nd should be most recent dir command")
	assert.Equal(t, "session 1", results[2].CommandText, "3rd should be second dir command")
	assert.Equal(t, "dir 5", results[3].CommandText, "4th should be third dir command")
	assert.Equal(t, "dir 4", results[4].CommandText, "5th should be fourth dir command")
	assert.Equal(t, "dir 3", results[5].CommandText, "6th should be fifth dir command")
}

// TestGetUniqueSourceApps tests retrieval of unique source app names
func TestGetUniqueSourceApps(t *testing.T) {
	t.Run("returns unique apps sorted alphabetically", func(t *testing.T) {
		// Given: database with commands from different source apps
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert commands from different apps
		apps := []string{"zsh", "bash", "zsh", "fish", "bash"}
		for i, app := range apps {
			sourceApp := app
			sourcePid := int64(1000 + i)
			sourceActive := true
			cmd := &models.Command{
				CommandText:  "test command",
				WorkingDir:   "/tmp",
				ExitStatus:   0,
				Timestamp:    int64(1000 + i),
				SourceApp:    &sourceApp,
				SourcePid:    &sourcePid,
				SourceActive: &sourceActive,
			}
			_, err := database.InsertCommand(cmd)
			require.NoError(t, err)
		}

		// When: GetUniqueSourceApps is called
		result, err := database.GetUniqueSourceApps()

		// Then: should return unique apps sorted alphabetically
		require.NoError(t, err)
		assert.Equal(t, []string{"bash", "fish", "zsh"}, result)
	})

	t.Run("returns empty slice for empty database", func(t *testing.T) {
		// Given: empty database
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// When: GetUniqueSourceApps is called
		result, err := database.GetUniqueSourceApps()

		// Then: should return empty slice
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns apps even without source info", func(t *testing.T) {
		// Given: database with commands without source info
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")
		database, err := NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert command without source info
		cmd := &models.Command{
			CommandText: "test command",
			WorkingDir:  "/tmp",
			ExitStatus:  0,
			Timestamp:   1000,
		}
		_, err = database.InsertCommand(cmd)
		require.NoError(t, err)

		// When: GetUniqueSourceApps is called
		result, err := database.GetUniqueSourceApps()

		// Then: should return empty since no sources exist
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}
