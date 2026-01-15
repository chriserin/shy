package summary

import (
	"github.com/chris/shy/pkg/models"
)

// ContextKey represents a unique context (directory/repository)
// Uses string values instead of pointers for proper map key comparison
type ContextKey struct {
	WorkingDir string
	GitRepo    string // Empty string if no git repo
}

// BranchKey represents a branch or "No branch" for non-git contexts
type BranchKey string

const NoBranch BranchKey = "No branch"

// GroupedCommands represents commands grouped by context and branch
type GroupedCommands struct {
	Contexts map[ContextKey]map[BranchKey][]models.Command
}

// GroupByContext groups commands by their context (working_dir/git_repo) and then by branch
// Returns a nested structure: Context -> Branch -> Commands
func GroupByContext(commands []models.Command) *GroupedCommands {
	grouped := &GroupedCommands{
		Contexts: make(map[ContextKey]map[BranchKey][]models.Command),
	}

	for _, cmd := range commands {
		// Create context key - convert pointer to value
		gitRepo := ""
		if cmd.GitRepo != nil {
			gitRepo = *cmd.GitRepo
		}
		contextKey := ContextKey{
			WorkingDir: cmd.WorkingDir,
			GitRepo:    gitRepo,
		}

		// Ensure context map exists
		if grouped.Contexts[contextKey] == nil {
			grouped.Contexts[contextKey] = make(map[BranchKey][]models.Command)
		}

		// Determine branch key
		var branchKey BranchKey
		if cmd.GitBranch != nil && *cmd.GitBranch != "" {
			branchKey = BranchKey(*cmd.GitBranch)
		} else {
			branchKey = NoBranch
		}

		// Append command to the appropriate context and branch
		grouped.Contexts[contextKey][branchKey] = append(
			grouped.Contexts[contextKey][branchKey],
			cmd,
		)
	}

	return grouped
}
