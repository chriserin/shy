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
