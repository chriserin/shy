package summary

import (
	"os"
	"path/filepath"
	"strings"

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

// The function is safe for both absolute and relative paths and normalises
// any Windows backslashes to forward slashes so the output is consistent
// across platforms.
func TildePath(path string) string {
	// Normalise the path first – this removes "..", ".", etc. and
	// gives us a consistent representation to compare against the home dir.
	abs, err := filepath.Abs(path)
	if err != nil {
		// If we can’t resolve an absolute path, fall back to the original.
		abs = path
	}

	// Get the user’s home directory in the same format as `abs`.
	home, err := os.UserHomeDir()
	if err != nil {
		// In the unlikely event we can’t determine the home dir,
		// just return the original path.
		return path
	}

	// Normalise the home directory string too, to match the normalised abs.
	home = filepath.Clean(home)

	// On Windows the home dir might contain backslashes. Convert both
	// strings to the same slash style so HasPrefix works as expected.
	// We use filepath.ToSlash which turns “C:\Users\me” → “C:/Users/me”.
	absSlash := filepath.ToSlash(abs)
	homeSlash := filepath.ToSlash(home)

	if !strings.HasPrefix(absSlash, homeSlash) {
		// No home‑directory prefix – nothing to replace.
		return path
	}

	// Compute the suffix after the home dir.
	// `len(homeSlash)` is the index where the suffix starts.
	suffix := absSlash[len(homeSlash):]

	// Join with "~".  If the original path was exactly the home dir,
	// suffix will be empty and we just return "~".
	return filepath.Join("~", suffix)
}
