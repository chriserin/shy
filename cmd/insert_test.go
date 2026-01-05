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
	database, err := db.New(dbPath_val)
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
	database, err := db.New(dbPath_val)
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
	database, err := db.New(dbPath_val)
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
		database, err := db.New(dbPath_val)
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
			database, err := db.New(dbPath_val)
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
	database, err := db.New(dbPath_val)
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
