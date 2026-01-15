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
	database, err := New(dbPath)
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
		{"working_dir", "TEXT"},
		{"git_repo", "TEXT"},
		{"git_branch", "TEXT"},
		{"source_app", "TEXT"},
		{"source_pid", "INTEGER"},
		{"source_active", "INTEGER"},
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
	database, err := New(dbPath)
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
	database, err := New(dbPath)
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
	database, err := New(dbPath)
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
	database, err := New(dbPath)
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
	database, err := New(dbPath)
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
	database, err := New(dbPath)
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
	database, err := New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Insert commands with timestamps representing 2026-01-14
	// Morning: 8:00 AM (timestamp: 1736841600)
	cmd1 := models.NewCommand("git status", "/home/user/projects/shy", 0)
	cmd1.Timestamp = 1736841600
	_, err = database.InsertCommand(cmd1)
	require.NoError(t, err, "failed to insert command 1")

	// Morning: 9:00 AM (timestamp: 1736845200)
	cmd2 := models.NewCommand("go build", "/home/user/projects/shy", 0)
	cmd2.Timestamp = 1736845200
	_, err = database.InsertCommand(cmd2)
	require.NoError(t, err, "failed to insert command 2")

	// Afternoon: 2:00 PM (timestamp: 1736863200)
	cmd3 := models.NewCommand("go test", "/home/user/projects/shy", 0)
	cmd3.Timestamp = 1736863200
	_, err = database.InsertCommand(cmd3)
	require.NoError(t, err, "failed to insert command 3")

	// Next day: 2026-01-15 8:00 AM (timestamp: 1736928000)
	cmd4 := models.NewCommand("git pull", "/home/user/projects/shy", 0)
	cmd4.Timestamp = 1736928000
	_, err = database.InsertCommand(cmd4)
	require.NoError(t, err, "failed to insert command 4")

	// When: I query commands for 2026-01-14 (start: 1736812800, end: 1736899200)
	startOfDay := int64(1736812800) // 2026-01-14 00:00:00 UTC
	endOfDay := int64(1736899200)   // 2026-01-15 00:00:00 UTC
	commands, err := database.GetCommandsByDateRange(startOfDay, endOfDay)
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
	database, err := New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// When: I query commands for any date range
	startOfDay := int64(1736812800)
	endOfDay := int64(1736899200)
	commands, err := database.GetCommandsByDateRange(startOfDay, endOfDay)
	require.NoError(t, err, "failed to get commands by date range")

	// Then: should return empty slice
	assert.Empty(t, commands, "should return empty slice when no commands in range")
}

// TestGetCommandsByDateRange_BoundaryConditions tests exact boundary matching
func TestGetCommandsByDateRange_BoundaryConditions(t *testing.T) {
	// Given: the shy database exists with commands at boundary times
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")
	database, err := New(dbPath)
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
	commands, err := database.GetCommandsByDateRange(1736812800, 1736899200)
	require.NoError(t, err, "failed to get commands by date range")

	// Then: should include start time (inclusive) but exclude end time (exclusive)
	assert.Len(t, commands, 2, "should include start and before-end, but not end")
	assert.Equal(t, "at start", commands[0].CommandText)
	assert.Equal(t, "before end", commands[1].CommandText)
}

// TestGetContextSummary tests the GetContextSummary method
func TestGetContextSummary(t *testing.T) {
	t.Run("multiple contexts sorted by duration", func(t *testing.T) {
		// Given: database with commands in multiple contexts
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := New(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert commands in shy repo (main branch) - 9h duration
		shyMain1 := models.NewCommand("go build", "/home/user/shy", 0)
		shyMain1.Timestamp = 1736838000 // 2026-01-14 08:00:00 UTC
		shyMain1.GitBranch = stringPtr("main")
		_, err = database.InsertCommand(shyMain1)
		require.NoError(t, err)

		shyMain2 := models.NewCommand("go test", "/home/user/shy", 0)
		shyMain2.Timestamp = 1736870400 // 2026-01-14 17:00:00 UTC
		shyMain2.GitBranch = stringPtr("main")
		_, err = database.InsertCommand(shyMain2)
		require.NoError(t, err)

		// Insert commands in webapp repo (dev branch) - 1.5h duration
		webapp1 := models.NewCommand("npm start", "/home/user/webapp", 0)
		webapp1.Timestamp = 1736845200 // 2026-01-14 10:00:00 UTC
		webapp1.GitBranch = stringPtr("dev")
		_, err = database.InsertCommand(webapp1)
		require.NoError(t, err)

		webapp2 := models.NewCommand("npm test", "/home/user/webapp", 0)
		webapp2.Timestamp = 1736850600 // 2026-01-14 11:30:00 UTC
		webapp2.GitBranch = stringPtr("dev")
		_, err = database.InsertCommand(webapp2)
		require.NoError(t, err)

		// When: GetContextSummary is called for the date range
		startTime := int64(1736812800) // 2026-01-14 00:00:00 UTC
		endTime := int64(1736899200)   // 2026-01-15 00:00:00 UTC
		summaries, err := database.GetContextSummary(startTime, endTime)

		// Then: should return summaries sorted by duration
		require.NoError(t, err)
		require.Len(t, summaries, 2)

		// First context (longest duration)
		assert.Equal(t, "/home/user/shy", summaries[0].WorkingDir)
		assert.Equal(t, "main", *summaries[0].GitBranch)
		assert.Equal(t, 2, summaries[0].CommandCount)
		assert.Equal(t, int64(1736838000), summaries[0].FirstTime)
		assert.Equal(t, int64(1736870400), summaries[0].LastTime)
		assert.Equal(t, int64(32400), summaries[0].LastTime-summaries[0].FirstTime) // 9h

		// Second context (shorter duration)
		assert.Equal(t, "/home/user/webapp", summaries[1].WorkingDir)
		assert.Equal(t, "dev", *summaries[1].GitBranch)
		assert.Equal(t, 2, summaries[1].CommandCount)
	})

	t.Run("non-git directory", func(t *testing.T) {
		// Given: database with commands in non-git directory
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := New(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert commands without git context
		cmd1 := models.NewCommand("vim .bashrc", "/home/user/dotfiles", 0)
		cmd1.Timestamp = 1736856615 // 2026-01-14 14:10:15 UTC
		_, err = database.InsertCommand(cmd1)
		require.NoError(t, err)

		cmd2 := models.NewCommand("source .bashrc", "/home/user/dotfiles", 0)
		cmd2.Timestamp = 1736858122 // 2026-01-14 14:35:22 UTC
		_, err = database.InsertCommand(cmd2)
		require.NoError(t, err)

		// When: GetContextSummary is called
		startTime := int64(1736812800) // 2026-01-14 00:00:00 UTC
		endTime := int64(1736899200)   // 2026-01-15 00:00:00 UTC
		summaries, err := database.GetContextSummary(startTime, endTime)

		// Then: should return summary with nil git branch
		require.NoError(t, err)
		require.Len(t, summaries, 1)

		assert.Equal(t, "/home/user/dotfiles", summaries[0].WorkingDir)
		assert.Nil(t, summaries[0].GitBranch)
		assert.Equal(t, 2, summaries[0].CommandCount)
	})

	t.Run("empty result", func(t *testing.T) {
		// Given: database with no commands in date range
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := New(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// When: GetContextSummary is called for empty date range
		startTime := int64(1736812800) // 2026-01-14 00:00:00 UTC
		endTime := int64(1736899200)   // 2026-01-15 00:00:00 UTC
		summaries, err := database.GetContextSummary(startTime, endTime)

		// Then: should return empty slice
		require.NoError(t, err)
		assert.Empty(t, summaries)
	})

	t.Run("multiple branches in same directory", func(t *testing.T) {
		// Given: database with multiple branches in same working directory
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := New(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Insert commands on feature-a branch
		featureA1 := models.NewCommand("go build", "/home/user/shy", 0)
		featureA1.Timestamp = 1736838000 // 2026-01-14 08:00:00 UTC
		featureA1.GitBranch = stringPtr("feature-a")
		_, err = database.InsertCommand(featureA1)
		require.NoError(t, err)

		featureA2 := models.NewCommand("go test", "/home/user/shy", 0)
		featureA2.Timestamp = 1736852400 // 2026-01-14 12:00:00 UTC
		featureA2.GitBranch = stringPtr("feature-a")
		_, err = database.InsertCommand(featureA2)
		require.NoError(t, err)

		// Insert commands on main branch
		main1 := models.NewCommand("git merge", "/home/user/shy", 0)
		main1.Timestamp = 1736866800 // 2026-01-14 16:00:00 UTC
		main1.GitBranch = stringPtr("main")
		_, err = database.InsertCommand(main1)
		require.NoError(t, err)

		main2 := models.NewCommand("git push", "/home/user/shy", 0)
		main2.Timestamp = 1736870400 // 2026-01-14 17:00:00 UTC
		main2.GitBranch = stringPtr("main")
		_, err = database.InsertCommand(main2)
		require.NoError(t, err)

		// When: GetContextSummary is called
		startTime := int64(1736812800) // 2026-01-14 00:00:00 UTC
		endTime := int64(1736899200)   // 2026-01-15 00:00:00 UTC
		summaries, err := database.GetContextSummary(startTime, endTime)

		// Then: should return separate contexts for each branch
		require.NoError(t, err)
		require.Len(t, summaries, 2)

		// Both contexts have same working directory but different branches
		assert.Equal(t, "/home/user/shy", summaries[0].WorkingDir)
		assert.Equal(t, "/home/user/shy", summaries[1].WorkingDir)
		assert.NotEqual(t, summaries[0].GitBranch, summaries[1].GitBranch)
	})

	t.Run("sort by command count when duration equal", func(t *testing.T) {
		// Given: database with contexts having equal duration but different command counts
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")
		database, err := New(dbPath)
		require.NoError(t, err)
		defer database.Close()

		// Context A: 2 commands, 1h duration
		cmdA1 := models.NewCommand("echo a", "/home/user/projA", 0)
		cmdA1.Timestamp = 1736838000 // 08:00:00
		cmdA1.GitBranch = stringPtr("main")
		_, err = database.InsertCommand(cmdA1)
		require.NoError(t, err)

		cmdA2 := models.NewCommand("echo b", "/home/user/projA", 0)
		cmdA2.Timestamp = 1736841600 // 09:00:00
		cmdA2.GitBranch = stringPtr("main")
		_, err = database.InsertCommand(cmdA2)
		require.NoError(t, err)

		// Context B: 4 commands, 1h duration
		cmdB1 := models.NewCommand("echo c", "/home/user/projB", 0)
		cmdB1.Timestamp = 1736845200 // 10:00:00
		cmdB1.GitBranch = stringPtr("main")
		_, err = database.InsertCommand(cmdB1)
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			cmdBN := models.NewCommand("echo d", "/home/user/projB", 0)
			cmdBN.Timestamp = 1736848800 // 11:00:00
			cmdBN.GitBranch = stringPtr("main")
			_, err = database.InsertCommand(cmdBN)
			require.NoError(t, err)
		}

		// When: GetContextSummary is called
		startTime := int64(1736812800)
		endTime := int64(1736899200)
		summaries, err := database.GetContextSummary(startTime, endTime)

		// Then: projB should come first (more commands)
		require.NoError(t, err)
		require.Len(t, summaries, 2)
		assert.Equal(t, "/home/user/projB", summaries[0].WorkingDir)
		assert.Equal(t, 4, summaries[0].CommandCount)
		assert.Equal(t, "/home/user/projA", summaries[1].WorkingDir)
		assert.Equal(t, 2, summaries[1].CommandCount)
	})
}

func stringPtr(s string) *string {
	return &s
}
