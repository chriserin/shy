package summary

import (
	"testing"

	"github.com/chris/shy/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGroupByContext_SingleRepoSingleBranch tests grouping commands from one repo and branch
func TestGroupByContext_SingleRepoSingleBranch(t *testing.T) {
	// Given: commands from single repository and branch
	repo := "github.com/chris/shy"
	branch := "main"
	commands := []models.Command{
		{
			ID:          1,
			Timestamp:   1736841600,
			CommandText: "git status",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
		},
		{
			ID:          2,
			Timestamp:   1736845200,
			CommandText: "go build",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
		},
	}

	// When: grouping by context
	grouped := GroupByContext(commands)

	// Then: should have one context
	require.Len(t, grouped.Contexts, 1)

	// And: context should have one branch
	contextKey := ContextKey{
		WorkingDir: "/home/user/projects/shy",
		GitRepo:    repo,
	}
	branches := grouped.Contexts[contextKey]
	require.Len(t, branches, 1)

	// And: branch should contain both commands
	branchCommands := branches[BranchKey("main")]
	assert.Len(t, branchCommands, 2)
	assert.Equal(t, "git status", branchCommands[0].CommandText)
	assert.Equal(t, "go build", branchCommands[1].CommandText)
}

// TestGroupByContext_MultipleRepos tests grouping commands from different repositories
func TestGroupByContext_MultipleRepos(t *testing.T) {
	// Given: commands from two different repositories
	repo1 := "github.com/chris/shy"
	repo2 := "github.com/user/app"
	branch := "main"
	commands := []models.Command{
		{
			CommandText: "go build",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo1,
			GitBranch:   &branch,
		},
		{
			CommandText: "npm install",
			WorkingDir:  "/home/user/projects/app",
			GitRepo:     &repo2,
			GitBranch:   &branch,
		},
		{
			CommandText: "go test",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo1,
			GitBranch:   &branch,
		},
	}

	// When: grouping by context
	grouped := GroupByContext(commands)

	// Then: should have two contexts
	require.Len(t, grouped.Contexts, 2)

	// And: first context should have go commands
	contextKey1 := ContextKey{
		WorkingDir: "/home/user/projects/shy",
		GitRepo:    repo1,
	}
	branches1 := grouped.Contexts[contextKey1]
	require.Len(t, branches1, 1)
	branchCommands1 := branches1[BranchKey("main")]
	assert.Len(t, branchCommands1, 2)
	assert.Equal(t, "go build", branchCommands1[0].CommandText)
	assert.Equal(t, "go test", branchCommands1[1].CommandText)

	// And: second context should have npm command
	contextKey2 := ContextKey{
		WorkingDir: "/home/user/projects/app",
		GitRepo:    repo2,
	}
	branches2 := grouped.Contexts[contextKey2]
	require.Len(t, branches2, 1)
	branchCommands2 := branches2[BranchKey("main")]
	assert.Len(t, branchCommands2, 1)
	assert.Equal(t, "npm install", branchCommands2[0].CommandText)
}

// TestGroupByContext_MultipleBranches tests grouping commands from different branches
func TestGroupByContext_MultipleBranches(t *testing.T) {
	// Given: commands from same repo but different branches
	repo := "github.com/chris/shy"
	mainBranch := "main"
	featureBranch := "feature-a"
	commands := []models.Command{
		{
			CommandText: "git checkout main",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &mainBranch,
		},
		{
			CommandText: "go test",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &mainBranch,
		},
		{
			CommandText: "git checkout -b feature-a",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &featureBranch,
		},
		{
			CommandText: "vim new.go",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &featureBranch,
		},
	}

	// When: grouping by context
	grouped := GroupByContext(commands)

	// Then: should have one context
	require.Len(t, grouped.Contexts, 1)

	// And: context should have two branches
	contextKey := ContextKey{
		WorkingDir: "/home/user/projects/shy",
		GitRepo:    repo,
	}
	branches := grouped.Contexts[contextKey]
	require.Len(t, branches, 2)

	// And: main branch should have 2 commands
	mainCommands := branches[BranchKey("main")]
	assert.Len(t, mainCommands, 2)
	assert.Equal(t, "git checkout main", mainCommands[0].CommandText)
	assert.Equal(t, "go test", mainCommands[1].CommandText)

	// And: feature branch should have 2 commands
	featureCommands := branches[BranchKey("feature-a")]
	assert.Len(t, featureCommands, 2)
	assert.Equal(t, "git checkout -b feature-a", featureCommands[0].CommandText)
	assert.Equal(t, "vim new.go", featureCommands[1].CommandText)
}

// TestGroupByContext_MixedGitAndNonGit tests grouping git and non-git directories
func TestGroupByContext_MixedGitAndNonGit(t *testing.T) {
	// Given: commands from git repo and non-git directory
	repo := "github.com/chris/shy"
	branch := "main"
	commands := []models.Command{
		{
			CommandText: "go build",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
		},
		{
			CommandText: "wget file.zip",
			WorkingDir:  "/home/user/downloads",
			GitRepo:     nil,
			GitBranch:   nil,
		},
		{
			CommandText: "unzip file.zip",
			WorkingDir:  "/home/user/downloads",
			GitRepo:     nil,
			GitBranch:   nil,
		},
	}

	// When: grouping by context
	grouped := GroupByContext(commands)

	// Then: should have two contexts
	require.Len(t, grouped.Contexts, 2)

	// And: git context should have one branch with one command
	gitContextKey := ContextKey{
		WorkingDir: "/home/user/projects/shy",
		GitRepo:    repo,
	}
	gitBranches := grouped.Contexts[gitContextKey]
	require.Len(t, gitBranches, 1)
	gitCommands := gitBranches[BranchKey("main")]
	assert.Len(t, gitCommands, 1)
	assert.Equal(t, "go build", gitCommands[0].CommandText)

	// And: non-git context should have "No branch" with two commands
	nonGitContextKey := ContextKey{
		WorkingDir: "/home/user/downloads",
		GitRepo:    "",
	}
	nonGitBranches := grouped.Contexts[nonGitContextKey]
	require.Len(t, nonGitBranches, 1)
	nonGitCommands := nonGitBranches[NoBranch]
	assert.Len(t, nonGitCommands, 2)
	assert.Equal(t, "wget file.zip", nonGitCommands[0].CommandText)
	assert.Equal(t, "unzip file.zip", nonGitCommands[1].CommandText)
}

// TestGroupByContext_NullGitBranch tests handling null git branch
func TestGroupByContext_NullGitBranch(t *testing.T) {
	// Given: commands with null git branch
	repo := "github.com/chris/shy"
	commands := []models.Command{
		{
			CommandText: "git init",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   nil,
		},
		{
			CommandText: "touch README.md",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   nil,
		},
	}

	// When: grouping by context
	grouped := GroupByContext(commands)

	// Then: should have one context
	require.Len(t, grouped.Contexts, 1)

	// And: context should use "No branch" for null branches
	contextKey := ContextKey{
		WorkingDir: "/home/user/projects/shy",
		GitRepo:    repo,
	}
	branches := grouped.Contexts[contextKey]
	require.Len(t, branches, 1)
	noBranchCommands := branches[NoBranch]
	assert.Len(t, noBranchCommands, 2)
	assert.Equal(t, "git init", noBranchCommands[0].CommandText)
	assert.Equal(t, "touch README.md", noBranchCommands[1].CommandText)
}

// TestGroupByContext_EmptyCommands tests handling empty command list
func TestGroupByContext_EmptyCommands(t *testing.T) {
	// Given: empty command list
	commands := []models.Command{}

	// When: grouping by context
	grouped := GroupByContext(commands)

	// Then: should have empty contexts map
	assert.Empty(t, grouped.Contexts)
}

// TestGroupByContext_EmptyBranch tests handling empty string branch
func TestGroupByContext_EmptyBranch(t *testing.T) {
	// Given: commands with empty string branch
	repo := "github.com/chris/shy"
	emptyBranch := ""
	commands := []models.Command{
		{
			CommandText: "git status",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &emptyBranch,
		},
	}

	// When: grouping by context
	grouped := GroupByContext(commands)

	// Then: should treat empty branch as "No branch"
	contextKey := ContextKey{
		WorkingDir: "/home/user/projects/shy",
		GitRepo:    repo,
	}
	branches := grouped.Contexts[contextKey]
	require.Len(t, branches, 1)
	noBranchCommands := branches[NoBranch]
	assert.Len(t, noBranchCommands, 1)
	assert.Equal(t, "git status", noBranchCommands[0].CommandText)
}
