package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupGitRepo creates a test git repository with a remote and branch
func setupGitRepo(t *testing.T, dir, remote, branch string) {
	t.Helper()

	// Determine initial branch name
	initBranch := branch
	if initBranch == "" {
		initBranch = "main"
	}

	// Initialize git repo with explicit branch name
	cmd := exec.Command("git", "init", "-b", initBranch)
	cmd.Dir = dir
	err := cmd.Run()
	require.NoError(t, err, "failed to init git repo")

	// Set user config for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err, "failed to set git user email")

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err, "failed to set git user name")

	// Add remote
	if remote != "" {
		cmd = exec.Command("git", "remote", "add", "origin", remote)
		cmd.Dir = dir
		err = cmd.Run()
		require.NoError(t, err, "failed to add git remote")
	}

	// Create initial commit
	readmeFile := filepath.Join(dir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test"), 0644)
	require.NoError(t, err, "failed to create README")

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err, "failed to git add")

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err, "failed to git commit")

}

// TestScenario8_AutoDetectGitContextFromWorkingDirectory tests auto-detection
// of git context from the working directory
func TestScenario8_AutoDetectGitContextFromWorkingDirectory(t *testing.T) {
	// Given: "/home/user/myproject" is a git repository with remote
	// "https://github.com/user/myproject.git" and the current branch is "main"
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "myproject")
	err := os.Mkdir(repoDir, 0755)
	require.NoError(t, err, "failed to create repo dir")

	setupGitRepo(t, repoDir, "https://github.com/user/myproject.git", "main")

	// When: I detect git context from the working directory
	gitCtx, err := DetectGitContext(repoDir)
	require.NoError(t, err, "failed to detect git context")

	// Then: the git context should be auto-detected
	require.NotNil(t, gitCtx, "git context should be detected")
	assert.Equal(t, "https://github.com/user/myproject.git", gitCtx.Repo, "git repo should match")
	assert.Equal(t, "main", gitCtx.Branch, "git branch should match")
}

// TestScenario9_AutoDetectGitContextFromSubdirectory tests auto-detection
// of git context from a subdirectory
func TestScenario9_AutoDetectGitContextFromSubdirectory(t *testing.T) {
	// Given: "/home/user/myproject/src/components" is inside a git repository
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "myproject")
	err := os.Mkdir(repoDir, 0755)
	require.NoError(t, err, "failed to create repo dir")

	setupGitRepo(t, repoDir, "https://github.com/user/myproject.git", "main")

	// Create subdirectory
	subDir := filepath.Join(repoDir, "src", "components")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err, "failed to create subdirectory")

	// When: I detect git context from the subdirectory
	gitCtx, err := DetectGitContext(subDir)
	require.NoError(t, err, "failed to detect git context")

	// Then: the git context should be auto-detected from the parent repository
	require.NotNil(t, gitCtx, "git context should be detected")
	assert.Equal(t, "https://github.com/user/myproject.git", gitCtx.Repo, "git repo should match")
	assert.Equal(t, "main", gitCtx.Branch, "git branch should match")
}

// TestDetectGitContext_NoGitRepo tests that DetectGitContext returns nil
// when not in a git repository
func TestDetectGitContext_NoGitRepo(t *testing.T) {
	// Given: a directory that is not a git repository
	tempDir := t.TempDir()

	// When: I detect git context
	gitCtx, err := DetectGitContext(tempDir)
	require.NoError(t, err, "should not error on non-git directory")

	// Then: git context should be nil
	assert.Nil(t, gitCtx, "git context should be nil for non-git directory")
}

// TestDetectGitContext_DifferentBranch tests detection with a custom branch
func TestDetectGitContext_DifferentBranch(t *testing.T) {
	// Given: a git repository on a feature branch
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "myproject")
	err := os.Mkdir(repoDir, 0755)
	require.NoError(t, err, "failed to create repo dir")

	setupGitRepo(t, repoDir, "https://github.com/user/myproject.git", "feature/test-branch")

	// When: I detect git context
	gitCtx, err := DetectGitContext(repoDir)
	require.NoError(t, err, "failed to detect git context")

	// Then: the branch should be detected correctly
	require.NotNil(t, gitCtx, "git context should be detected")
	assert.Equal(t, "feature/test-branch", gitCtx.Branch, "git branch should match")
}
