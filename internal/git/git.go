package git

import (
	"os/exec"
	"strings"
)

// GitContext holds git repository information
type GitContext struct {
	Repo   string
	Branch string
}

// DetectGitContext attempts to detect git repository information from a directory
func DetectGitContext(dir string) (*GitContext, error) {
	// Check if we're in a git repository
	if !isGitRepo(dir) {
		return nil, nil
	}

	repo, err := getGitRemote(dir)
	if err != nil {
		return nil, err
	}

	branch, err := getGitBranch(dir)
	if err != nil {
		return nil, err
	}

	return &GitContext{
		Repo:   repo,
		Branch: branch,
	}, nil
}

// isGitRepo checks if the directory is within a git repository
func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	return cmd.Run() == nil
}

// getGitRemote gets the remote URL for the repository
func getGitRemote(dir string) (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		// If no remote, return empty string
		return "", nil
	}
	return strings.TrimSpace(string(output)), nil
}

// getGitBranch gets the current branch name
func getGitBranch(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
