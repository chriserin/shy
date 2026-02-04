package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

// TestScenario22_ListOutputShowsGitBranch tests that --fmt=gb,cmd shows git branch
func TestScenario22_ListOutputShowsGitBranch(t *testing.T) {
	// Given: I have a database with a command with git context
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	gitRepo := "github.com/user/repo"
	gitBranch := "main"
	cmd := &models.Command{
		CommandText: "git commit -m 'test'",
		WorkingDir:  "/home/project",
		ExitStatus:  0,
		Timestamp:   1704470400,
		GitRepo:     &gitRepo,
		GitBranch:   &gitBranch,
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	// When: I run "shy list --fmt=gb,cmd"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "--fmt=gb,cmd", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "list command should succeed")

	output := buf.String()

	// Then: the output should show the git branch and the command
	assert.Contains(t, output, "main", "should show git branch")
	assert.Contains(t, output, "git commit -m 'test'", "should show command")
	assert.Contains(t, output, "main\tgit commit -m 'test'", "should be tab-separated")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}

// TestScenario23_ListOutputShowsGitRepo tests that --fmt=gr,cmd shows git repo
func TestScenario23_ListOutputShowsGitRepo(t *testing.T) {
	// Given: I have a database with a command with git context
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := db.NewForTesting(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	gitRepo := "github.com/user/repo"
	gitBranch := "main"
	cmd := &models.Command{
		CommandText: "git commit -m 'test'",
		WorkingDir:  "/home/project",
		ExitStatus:  0,
		Timestamp:   1704470400,
		GitRepo:     &gitRepo,
		GitBranch:   &gitBranch,
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	// When: I run "shy list --fmt=gr,cmd"
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "--fmt=gr,cmd", "--db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err, "list command should succeed")

	output := buf.String()

	// Then: the output should show the git repo and the command
	assert.Contains(t, output, "github.com/user/repo", "should show git repo")
	assert.Contains(t, output, "git commit -m 'test'", "should show command")
	assert.Contains(t, output, "github.com/user/repo\tgit commit -m 'test'", "should be tab-separated")

	// Reset command for next test
	rootCmd.SetArgs(nil)
}
