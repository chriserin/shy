package summary

import (
	"strings"
	"testing"
	"time"

	"github.com/chris/shy/pkg/models"
	"github.com/stretchr/testify/assert"
)

// TestFormatSummary_SingleContext tests formatting with one context
func TestFormatSummary_SingleContext(t *testing.T) {
	// Given: commands from single repository and branch
	repo := "github.com/chris/shy"
	branch := "main"
	commands := []models.Command{
		{
			CommandText: "git status",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "go build",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 10, 0, 0, 0, time.Local).Unix(),
		},
	}

	grouped := GroupByContext(commands)
	opts := FormatOptions{
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should contain date header
	assert.Contains(t, output, "Work Summary - 2026-01-14")

	// And: should contain context header
	assert.Contains(t, output, "/home/user/projects/shy (github.com/chris/shy)")

	// And: should contain branch
	assert.Contains(t, output, "Branch: main")

	// And: should contain time period
	assert.Contains(t, output, "Morning")

	// And: should contain commands with timestamps
	assert.Contains(t, output, "9:00am  git status")
	assert.Contains(t, output, "10:00am  go build")

	// And: should contain statistics
	assert.Contains(t, output, "Total commands: 2")
	assert.Contains(t, output, "Unique contexts: 1 (1 repo)")
	assert.Contains(t, output, "Branches worked on: 1")
}

// TestFormatSummary_WithoutAllCommands tests summary without showing all commands
func TestFormatSummary_WithoutAllCommands(t *testing.T) {
	// Given: commands from single repository
	repo := "github.com/chris/shy"
	branch := "main"
	commands := []models.Command{
		{
			CommandText: "git status",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "go build",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 10, 0, 0, 0, time.Local).Unix(),
		},
	}

	grouped := GroupByContext(commands)
	opts := FormatOptions{
		AllCommands: false,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should show time period with count
	assert.Contains(t, output, "Morning")
	assert.Contains(t, output, "2 commands")

	// But: should NOT show individual commands
	assert.NotContains(t, output, "9:00am  git status")
	assert.NotContains(t, output, "10:00am  go build")
}

// TestFormatSummary_MultipleContexts tests formatting with multiple contexts
func TestFormatSummary_MultipleContexts(t *testing.T) {
	// Given: commands from two repositories
	repo1 := "github.com/chris/shy"
	repo2 := "github.com/user/app"
	branch := "main"
	commands := []models.Command{
		{
			CommandText: "go build",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo1,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "npm install",
			WorkingDir:  "/home/user/projects/app",
			GitRepo:     &repo2,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 10, 0, 0, 0, time.Local).Unix(),
		},
	}

	grouped := GroupByContext(commands)
	opts := FormatOptions{
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should contain both contexts
	assert.Contains(t, output, "/home/user/projects/app")
	assert.Contains(t, output, "/home/user/projects/shy")

	// And: contexts should be sorted alphabetically
	appIndex := strings.Index(output, "/home/user/projects/app")
	shyIndex := strings.Index(output, "/home/user/projects/shy")
	assert.Less(t, appIndex, shyIndex, "app should appear before shy")

	// And: statistics should show 2 repos
	assert.Contains(t, output, "Unique contexts: 2 (2 repos)")
}

// TestFormatSummary_MultipleBranches tests formatting with multiple branches
func TestFormatSummary_MultipleBranches(t *testing.T) {
	// Given: commands from multiple branches
	repo := "github.com/chris/shy"
	mainBranch := "main"
	featureBranch := "feature"
	commands := []models.Command{
		{
			CommandText: "git checkout main",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &mainBranch,
			Timestamp:   time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "git checkout feature",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &featureBranch,
			Timestamp:   time.Date(2026, 1, 14, 10, 0, 0, 0, time.Local).Unix(),
		},
	}

	grouped := GroupByContext(commands)
	opts := FormatOptions{
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should contain both branches
	assert.Contains(t, output, "Branch: feature")
	assert.Contains(t, output, "Branch: main")

	// And: branches should be sorted alphabetically
	featureIndex := strings.Index(output, "Branch: feature")
	mainIndex := strings.Index(output, "Branch: main")
	assert.Less(t, featureIndex, mainIndex, "feature should appear before main")

	// And: statistics should show 2 branches
	assert.Contains(t, output, "Branches worked on: 2")
}

// TestFormatSummary_NonGitDirectory tests formatting non-git directory
func TestFormatSummary_NonGitDirectory(t *testing.T) {
	// Given: commands from non-git directory
	commands := []models.Command{
		{
			CommandText: "wget file.zip",
			WorkingDir:  "/home/user/downloads",
			GitRepo:     nil,
			GitBranch:   nil,
			Timestamp:   time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix(),
		},
	}

	grouped := GroupByContext(commands)
	opts := FormatOptions{
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should show working dir without git repo
	assert.Contains(t, output, "/home/user/downloads")
	assert.NotContains(t, output, "github.com")

	// And: should show "No git repository"
	assert.Contains(t, output, "No git repository")

	// And: statistics should show non-repo dir
	assert.Contains(t, output, "Unique contexts: 1 (1 non-repo dir)")
}

// TestFormatSummary_MixedGitAndNonGit tests mixed git and non-git contexts
func TestFormatSummary_MixedGitAndNonGit(t *testing.T) {
	// Given: commands from git repo and non-git directory
	repo := "github.com/chris/shy"
	branch := "main"
	commands := []models.Command{
		{
			CommandText: "go build",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "wget file",
			WorkingDir:  "/home/user/downloads",
			GitRepo:     nil,
			GitBranch:   nil,
			Timestamp:   time.Date(2026, 1, 14, 10, 0, 0, 0, time.Local).Unix(),
		},
	}

	grouped := GroupByContext(commands)
	opts := FormatOptions{
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: statistics should show both types
	assert.Contains(t, output, "Unique contexts: 2 (1 repos, 1 non-repo dir)")
}

// TestFormatSummary_EmptyCommands tests formatting with no commands
func TestFormatSummary_EmptyCommands(t *testing.T) {
	// Given: no commands
	commands := []models.Command{}
	grouped := GroupByContext(commands)
	opts := FormatOptions{
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should show no commands message
	assert.Contains(t, output, "No commands found for 2026-01-14")
}

// TestFormatSummary_AllTimePeriods tests formatting with commands in all periods
func TestFormatSummary_AllTimePeriods(t *testing.T) {
	// Given: commands in all time periods
	repo := "github.com/chris/shy"
	branch := "main"
	commands := []models.Command{
		{
			CommandText: "night cmd",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 2, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "morning cmd",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "afternoon cmd",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 14, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "evening cmd",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 20, 0, 0, 0, time.Local).Unix(),
		},
	}

	grouped := GroupByContext(commands)
	opts := FormatOptions{
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should show all time periods in chronological order
	nightIndex := strings.Index(output, "Night")
	morningIndex := strings.Index(output, "Morning")
	afternoonIndex := strings.Index(output, "Afternoon")
	eveningIndex := strings.Index(output, "Evening")

	assert.Less(t, nightIndex, morningIndex)
	assert.Less(t, morningIndex, afternoonIndex)
	assert.Less(t, afternoonIndex, eveningIndex)
}

// TestFormatSummary_TimeRangeFormatting tests time range formatting
func TestFormatSummary_TimeRangeFormatting(t *testing.T) {
	// Given: commands with specific timestamps
	repo := "github.com/chris/shy"
	branch := "main"
	commands := []models.Command{
		{
			CommandText: "first",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 8, 23, 15, 0, time.Local).Unix(),
		},
		{
			CommandText: "last",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 11, 47, 30, 0, time.Local).Unix(),
		},
	}

	grouped := GroupByContext(commands)
	opts := FormatOptions{
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should show time range in 12-hour format
	assert.Contains(t, output, "Morning (8:23am - 11:47am)")

	// And: should show timestamps in 12-hour format without seconds
	assert.Contains(t, output, "8:23am  first")
	assert.Contains(t, output, "11:47am  last")
}

// TestFormatSummary_SingleCommandTimeRange tests single command time display
func TestFormatSummary_SingleCommandTimeRange(t *testing.T) {
	// Given: single command
	repo := "github.com/chris/shy"
	branch := "main"
	commands := []models.Command{
		{
			CommandText: "git status",
			WorkingDir:  "/home/user/projects/shy",
			GitRepo:     &repo,
			GitBranch:   &branch,
			Timestamp:   time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix(),
		},
	}

	grouped := GroupByContext(commands)
	opts := FormatOptions{
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should show single time (not range) for single command
	assert.Contains(t, output, "Morning (9:00am) - 1 commands")
	assert.NotContains(t, output, "9:00am - 9:00am")
}

// TestSortContexts tests context sorting
func TestSortContexts(t *testing.T) {
	// Given: contexts with different working directories
	repo1 := "github.com/user/zzz"
	repo2 := "github.com/user/aaa"
	contexts := map[ContextKey]map[BranchKey][]models.Command{
		{WorkingDir: "/home/user/zzz", GitRepo: repo1}: {},
		{WorkingDir: "/home/user/aaa", GitRepo: repo2}: {},
		{WorkingDir: "/home/user/mmm", GitRepo: ""}:    {},
	}

	// When: sorting contexts
	sorted := sortContexts(contexts)

	// Then: should be sorted alphabetically by working dir
	assert.Equal(t, "/home/user/aaa", sorted[0].WorkingDir)
	assert.Equal(t, "/home/user/mmm", sorted[1].WorkingDir)
	assert.Equal(t, "/home/user/zzz", sorted[2].WorkingDir)
}

// TestSortBranches tests branch sorting
func TestSortBranches(t *testing.T) {
	// Given: branches including "No branch"
	branches := map[BranchKey][]models.Command{
		"zeta":    {},
		"alpha":   {},
		"beta":    {},
		NoBranch:  {},
	}

	// When: sorting branches
	sorted := sortBranches(branches)

	// Then: should be sorted alphabetically with "No branch" last
	assert.Equal(t, BranchKey("alpha"), sorted[0])
	assert.Equal(t, BranchKey("beta"), sorted[1])
	assert.Equal(t, BranchKey("zeta"), sorted[2])
	assert.Equal(t, NoBranch, sorted[3])
}
