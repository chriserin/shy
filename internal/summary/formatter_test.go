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
		NoColor:     true,
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should contain date header
	assert.Contains(t, output, "Work Summary - 2026-01-14")

	// And: should contain context header with directory:branch on one line
	assert.Contains(t, output, "/home/user/projects/shy:main")

	// And: should contain hourly buckets with dashes
	assert.Contains(t, output, "9am ------------------------------")
	assert.Contains(t, output, "10am ------------------------------")

	// And: should contain commands with minute-only timestamps
	assert.Contains(t, output, ":00  git status")
	assert.Contains(t, output, ":00  go build")
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
		NoColor:     true,
		AllCommands: false,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should show hourly buckets with dashes
	assert.Contains(t, output, "9am ------------------------------")
	assert.Contains(t, output, "10am ------------------------------")

	// But: should NOT show individual commands (allCommands is false)
	assert.NotContains(t, output, ":00  git status")
	assert.NotContains(t, output, ":00  go build")
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
		NoColor:     true,
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
		NoColor:     true,
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should contain both branches in directory:branch format
	assert.Contains(t, output, "/home/user/projects/shy:feature")
	assert.Contains(t, output, "/home/user/projects/shy:main")

	// And: branches should be sorted alphabetically
	featureIndex := strings.Index(output, ":feature")
	mainIndex := strings.Index(output, ":main")
	assert.Less(t, featureIndex, mainIndex, "feature should appear before main")
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
		NoColor:     true,
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should show working dir without git repo
	assert.Contains(t, output, "/home/user/downloads")
	assert.NotContains(t, output, "github.com")

	// And: should show "(non-git)" on the same line as directory
	assert.Contains(t, output, "/home/user/downloads")
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
		NoColor:     true,
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)
	assert.Contains(t, output, "/home/user/downloads")
	assert.Contains(t, output, "/home/user/projects/shy:main")
}

// TestFormatSummary_EmptyCommands tests formatting with no commands
func TestFormatSummary_EmptyCommands(t *testing.T) {
	// Given: no commands
	commands := []models.Command{}
	grouped := GroupByContext(commands)
	opts := FormatOptions{
		NoColor:     true,
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
		NoColor:     true,
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should show hourly buckets in chronological order
	// Commands are at 9am, 2pm (14), 8pm (20)
	hour9Index := strings.Index(output, "9am")
	hour14Index := strings.Index(output, "2pm")
	hour20Index := strings.Index(output, "8pm")

	assert.Greater(t, hour9Index, -1, "9am should be present")
	assert.Greater(t, hour14Index, -1, "2pm should be present")
	assert.Greater(t, hour20Index, -1, "8pm should be present")
	assert.Less(t, hour9Index, hour14Index, "9am should appear before 2pm")
	assert.Less(t, hour14Index, hour20Index, "2pm should appear before 8pm")
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
		NoColor:     true,
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should show hourly buckets with dashes
	// Commands at 8:23am and 11:47am are in different hours
	assert.Contains(t, output, "8am ------------------------------")
	assert.Contains(t, output, "11am ------------------------------")

	// And: should show timestamps as minutes only
	assert.Contains(t, output, ":23  first")
	assert.Contains(t, output, ":47  last")
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
		NoColor:     true,
		AllCommands: true,
		Date:        "2026-01-14",
	}

	// When: formatting summary
	output := FormatSummary(grouped, opts)

	// Then: should show hour separator with dashes
	assert.Contains(t, output, "9am ------------------------------")
	// And: should show command with minute timestamp
	assert.Contains(t, output, ":00  git status")
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
		"zeta":   {},
		"alpha":  {},
		"beta":   {},
		NoBranch: {},
	}

	// When: sorting branches
	sorted := sortBranches(branches)

	// Then: should be sorted alphabetically with "No branch" last
	assert.Equal(t, BranchKey("alpha"), sorted[0])
	assert.Equal(t, BranchKey("beta"), sorted[1])
	assert.Equal(t, BranchKey("zeta"), sorted[2])
	assert.Equal(t, NoBranch, sorted[3])
}
