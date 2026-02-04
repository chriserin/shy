package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

// setupGitRepo creates a test git repository
func setupGitRepo(t *testing.T, dir, remote, branch string) {
	t.Helper()

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	require.NoError(t, cmd.Run(), "failed to init git repo")

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	require.NoError(t, cmd.Run(), "failed to set git user email")

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	require.NoError(t, cmd.Run(), "failed to set git user name")

	if remote != "" {
		cmd = exec.Command("git", "remote", "add", "origin", remote)
		cmd.Dir = dir
		require.NoError(t, cmd.Run(), "failed to add git remote")
	}

	readmeFile := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readmeFile, []byte("# Test"), 0644), "failed to create README")

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = dir
	require.NoError(t, cmd.Run(), "failed to git add")

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	require.NoError(t, cmd.Run(), "failed to git commit")

	if branch != "" && branch != "master" && branch != "main" {
		cmd = exec.Command("git", "checkout", "-b", branch)
		cmd.Dir = dir
		require.NoError(t, cmd.Run(), "failed to checkout branch")
	}
}

// TestScenario10_ExplicitGitContextOverridesAutoDetection tests that explicit
// git context parameters override auto-detection
func TestScenario10_ExplicitGitContextOverridesAutoDetection(t *testing.T) {
	// Given: the shy database exists and we're in a git repository
	tempDir := t.TempDir()
	dbPath_val := filepath.Join(tempDir, "history.db")

	repoDir := filepath.Join(tempDir, "myproject")
	require.NoError(t, os.Mkdir(repoDir, 0755), "failed to create repo dir")
	setupGitRepo(t, repoDir, "https://github.com/user/myproject.git", "main")

	// When: I insert with explicit git context that differs from auto-detection
	database, err := db.NewForTesting(dbPath_val)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	cmd := models.NewCommand("ls", repoDir, 0)
	overrideRepo := "override-repo"
	overrideBranch := "override-branch"
	cmd.GitRepo = &overrideRepo
	cmd.GitBranch = &overrideBranch

	id, err := database.InsertCommand(cmd)
	require.NoError(t, err, "insert should succeed")

	// Then: the record should have the explicit git context, not the auto-detected one
	retrievedCmd, err := database.GetCommand(id)
	require.NoError(t, err, "failed to get command")

	require.NotNil(t, retrievedCmd.GitRepo, "git repo should not be NULL")
	assert.Equal(t, "override-repo", *retrievedCmd.GitRepo, "git repo should be overridden")
	require.NotNil(t, retrievedCmd.GitBranch, "git branch should not be NULL")
	assert.Equal(t, "override-branch", *retrievedCmd.GitBranch, "git branch should be overridden")
}

// TestScenario11_InsertCommandWithTimestampOverride tests inserting a command
// with a custom timestamp
func TestScenario11_InsertCommandWithTimestampOverride(t *testing.T) {
	// Given: the shy database exists
	tempDir := t.TempDir()
	dbPath_val := filepath.Join(tempDir, "history.db")
	database, err := db.NewForTesting(dbPath_val)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// When: I insert with a timestamp override
	cmd := models.NewCommand("ls", "/home/user", 0)
	cmd.Timestamp = 1704067200

	id, err := database.InsertCommand(cmd)
	require.NoError(t, err, "insert should succeed")

	// Then: the record should have the custom timestamp
	retrievedCmd, err := database.GetCommand(id)
	require.NoError(t, err, "failed to get command")
	assert.Equal(t, int64(1704067200), retrievedCmd.Timestamp, "timestamp should match override")
}

// TestScenario12_InsertFailsWithMissingRequiredParameters tests that the insert
// command fails when required parameters are missing
func TestScenario12_InsertFailsWithMissingRequiredParameters(t *testing.T) {
	// Given: the shy database exists
	tempDir := t.TempDir()
	dbPath_val := filepath.Join(tempDir, "history.db")

	// Create database first
	database, err := db.NewForTesting(dbPath_val)
	require.NoError(t, err, "failed to create database")
	database.Close()

	// Reset flags
	command = ""
	dir = ""
	status = 0
	gitRepo = ""
	gitBranch = ""
	timestamp = 0

	// When: I run shy insert without the --dir parameter
	command = "ls"
	dir = "" // Missing required parameter
	status = 0
	dbPath = dbPath_val

	err = runInsert(nil, nil)

	// Then: the command should fail with an error
	// Note: Since we're testing the RunE function directly, we need to check
	// that dir is empty. In a real CLI scenario, Cobra would enforce the
	// required flag before RunE is called.
	assert.Error(t, err, "should error when dir is empty")
}

// TestScenario14_DatabaseHandlesConcurrentInserts tests that the database
// can handle concurrent insert operations
func TestScenario14_DatabaseHandlesConcurrentInserts(t *testing.T) {
	// Given: the shy database exists
	tempDir := t.TempDir()
	dbPath_val := filepath.Join(tempDir, "history.db")

	// Create database and table first to avoid concurrent table creation
	{
		database, err := db.NewForTesting(dbPath_val)
		require.NoError(t, err, "failed to create database")
		database.Close()
	}

	// When: I run two shy insert commands simultaneously
	var wg sync.WaitGroup
	errors := make([]error, 2)
	ids := make([]int64, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Each goroutine uses its own database connection
			database, err := db.NewForTesting(dbPath_val)
			if err != nil {
				errors[index] = err
				return
			}
			defer database.Close()

			// Insert a command using the models package
			cmd := &models.Command{
				Timestamp:   1704067200 + int64(index),
				ExitStatus:  0,
				CommandText: "test command " + string(rune('A'+index)),
				WorkingDir:  "/home/user",
			}
			id, err := database.InsertCommand(cmd)
			ids[index] = id
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Then: both commands should succeed
	for i, err := range errors {
		assert.NoError(t, err, "insert %d should succeed", i)
	}

	// And: both records should be in the database
	database, err := db.NewForTesting(dbPath_val)
	require.NoError(t, err, "failed to open database")
	defer database.Close()

	count, err := database.CountCommands()
	require.NoError(t, err, "failed to count commands")
	assert.Equal(t, 2, count, "should have 2 commands in database")

	// And: no database locking errors should occur (verified by no errors above)
	// And: both records should have unique ids
	cmd1, err := database.GetCommand(ids[0])
	require.NoError(t, err, "failed to get command 1")
	cmd2, err := database.GetCommand(ids[1])
	require.NoError(t, err, "failed to get command 2")
	assert.NotEqual(t, cmd1.ID, cmd2.ID, "commands should have unique IDs")
}

// Scenario 31: Database captures session source on insert
func TestScenario31_DatabaseCapturesSessionSourceOnInsert(t *testing.T) {
	// Given: shy is integrated with zsh
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// When: I run a command "echo test" with pid 12345
	sourceApp := "zsh"
	sourcePid := int64(12345)
	sourceActive := true

	cmd := &models.Command{
		CommandText:  "echo test",
		WorkingDir:   "/home/test",
		ExitStatus:   0,
		Timestamp:    1704470400,
		SourceApp:    &sourceApp,
		SourcePid:    &sourcePid,
		SourceActive: &sourceActive,
	}
	id, err := database.InsertCommand(cmd)
	require.NoError(t, err)

	// Then: the database should record source_app as "zsh"
	retrievedCmd, err := database.GetCommand(id)
	require.NoError(t, err)
	require.NotNil(t, retrievedCmd.SourceApp)
	assert.Equal(t, "zsh", *retrievedCmd.SourceApp)

	// And: the database should record source_pid as 12345
	require.NotNil(t, retrievedCmd.SourcePid)
	assert.Equal(t, int64(12345), *retrievedCmd.SourcePid)

	// And: source_active should be true
	require.NotNil(t, retrievedCmd.SourceActive)
	assert.True(t, *retrievedCmd.SourceActive)
}

// Test insert command with source tracking flags
func TestInsertCommandWithSourceTracking(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	// Initialize database first
	initDB, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	initDB.Close()

	// Run insert command with source tracking flags
	rootCmd.SetArgs([]string{
		"insert",
		"--command", "git status",
		"--dir", tempDir,
		"--status", "0",
		"--source-app", "zsh",
		"--source-pid", "54321",
		"--db", dbPath,
	})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Verify the command was inserted with source tracking
	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	cmd, err := database.GetCommand(1)
	require.NoError(t, err)

	assert.Equal(t, "git status", cmd.CommandText)
	require.NotNil(t, cmd.SourceApp)
	assert.Equal(t, "zsh", *cmd.SourceApp)
	require.NotNil(t, cmd.SourcePid)
	assert.Equal(t, int64(54321), *cmd.SourcePid)
	require.NotNil(t, cmd.SourceActive)
	assert.True(t, *cmd.SourceActive)

	rootCmd.SetArgs(nil)
}

// Scenario 33: Bash session tracking works independently
func TestScenario33_BashSessionTrackingWorksIndependently(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Given: I have zsh session with pid 11111
	zshApp := "zsh"
	zshPid := int64(11111)
	zshActive := true

	// And: I have bash session with pid 22222
	bashApp := "bash"
	bashPid := int64(22222)
	bashActive := true

	// When: I run command "zsh-cmd" in zsh session
	zshCmd := &models.Command{
		CommandText:  "zsh-cmd",
		WorkingDir:   "/home/test",
		ExitStatus:   0,
		Timestamp:    1704470400,
		SourceApp:    &zshApp,
		SourcePid:    &zshPid,
		SourceActive: &zshActive,
	}
	_, err = database.InsertCommand(zshCmd)
	require.NoError(t, err)

	// And: I run command "bash-cmd" in bash session
	bashCmd := &models.Command{
		CommandText:  "bash-cmd",
		WorkingDir:   "/home/test",
		ExitStatus:   0,
		Timestamp:    1704470401,
		SourceApp:    &bashApp,
		SourcePid:    &bashPid,
		SourceActive: &bashActive,
	}
	_, err = database.InsertCommand(bashCmd)
	require.NoError(t, err)

	// Then: "zsh-cmd" should have source "zsh:11111"
	cmd1, err := database.GetCommand(1)
	require.NoError(t, err)
	require.NotNil(t, cmd1.SourceApp)
	assert.Equal(t, "zsh", *cmd1.SourceApp)
	require.NotNil(t, cmd1.SourcePid)
	assert.Equal(t, int64(11111), *cmd1.SourcePid)

	// And: "bash-cmd" should have source "bash:22222"
	cmd2, err := database.GetCommand(2)
	require.NoError(t, err)
	require.NotNil(t, cmd2.SourceApp)
	assert.Equal(t, "bash", *cmd2.SourceApp)
	require.NotNil(t, cmd2.SourcePid)
	assert.Equal(t, int64(22222), *cmd2.SourcePid)

	// When: I run "shy fc -l -I" in zsh session (PID 11111)
	// Then: I should only see "zsh-cmd"
	commands, err := database.GetCommandsByRangeInternal(1, 100, 11111)
	require.NoError(t, err)
	assert.Len(t, commands, 1)
	assert.Equal(t, "zsh-cmd", commands[0].CommandText)
}

// TestInsertCommandWithLeadingSpace tests that commands with leading space are not inserted
func TestInsertCommandWithLeadingSpace(t *testing.T) {
	t.Run("command with leading space is not inserted", func(t *testing.T) {
		// Given: a database exists
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")

		// When: I insert a command with a leading space
		rootCmd.SetArgs([]string{
			"insert",
			"--command", " secret command", // Leading space
			"--dir", tempDir,
			"--status", "0",
			"--db", dbPath,
		})

		err := rootCmd.Execute()

		// Then: the command should succeed (exit code 0)
		require.NoError(t, err, "insert should succeed even with leading space")

		// And: the database should be empty (command was not inserted)
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		count, err := database.CountCommands()
		require.NoError(t, err)
		assert.Equal(t, 0, count, "database should be empty")

		rootCmd.SetArgs(nil)
	})

	t.Run("command without leading space is inserted normally", func(t *testing.T) {
		// Given: a database exists
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")

		// Initialize database first
		initDB, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		initDB.Close()

		// When: I insert a command without a leading space
		rootCmd.SetArgs([]string{
			"insert",
			"--command", "normal command",
			"--dir", tempDir,
			"--status", "0",
			"--db", dbPath,
		})

		err = rootCmd.Execute()
		require.NoError(t, err)

		// Then: the command should be inserted
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		count, err := database.CountCommands()
		require.NoError(t, err)
		assert.Equal(t, 1, count, "database should have 1 command")

		cmd, err := database.GetCommand(1)
		require.NoError(t, err)
		assert.Equal(t, "normal command", cmd.CommandText)

		rootCmd.SetArgs(nil)
	})

	t.Run("command with multiple leading spaces is not inserted", func(t *testing.T) {
		// Given: a database exists
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")

		// When: I insert a command with multiple leading spaces
		rootCmd.SetArgs([]string{
			"insert",
			"--command", "   command with multiple spaces",
			"--dir", tempDir,
			"--status", "0",
			"--db", dbPath,
		})

		err := rootCmd.Execute()
		require.NoError(t, err, "insert should succeed")

		// Then: the database should be empty
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		count, err := database.CountCommands()
		require.NoError(t, err)
		assert.Equal(t, 0, count, "database should be empty")

		rootCmd.SetArgs(nil)
	})

	t.Run("command with trailing space is inserted and trimmed", func(t *testing.T) {
		// Given: a database exists
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "history.db")

		// Initialize database first
		initDB, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		initDB.Close()

		// When: I insert a command with trailing space (but no leading space)
		rootCmd.SetArgs([]string{
			"insert",
			"--command", "command with trailing space ",
			"--dir", tempDir,
			"--status", "0",
			"--db", dbPath,
		})

		err = rootCmd.Execute()
		require.NoError(t, err)

		// Then: the command should be inserted (trailing space doesn't prevent insertion)
		database, err := db.NewForTesting(dbPath)
		require.NoError(t, err)
		defer database.Close()

		count, err := database.CountCommands()
		require.NoError(t, err)
		assert.Equal(t, 1, count, "database should have 1 command")

		cmd, err := database.GetCommand(1)
		require.NoError(t, err)
		assert.Equal(t, "command with trailing space", cmd.CommandText)

		rootCmd.SetArgs(nil)
	})
}
