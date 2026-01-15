package summary

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/chris/shy/pkg/models"
)

// Styles for summary output (matching television theme palette from cmd/tv.go)
var (
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true) // bright-magenta
	contextStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))            // white
	gitStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // bright-magenta
	branchStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))            // bright-blue
	periodStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true) // bright-yellow (for period name)
	timeRangeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))            // bright-black/gray (for time range in heading)
	timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))            // bright-yellow (for command timestamps)
	commandStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))            // white
	statLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))            // bright-blue
	statValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))            // bright-green
	separatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // bright-black
)

// FormatOptions contains options for formatting the summary
type FormatOptions struct {
	AllCommands bool   // Show all commands or just summary
	Date        string // Date being summarized (YYYY-MM-DD format)
	NoColor     bool   // Disable color output
}

// Helper function to render with or without colors
func renderStyle(style lipgloss.Style, text string, noColor bool) string {
	if noColor {
		return text
	}
	return style.Render(text)
}

// FormatSummary formats the grouped and bucketed commands into human-readable text
func FormatSummary(grouped *GroupedCommands, opts FormatOptions) string {
	var output strings.Builder

	// Header
	var title string
	if strings.HasPrefix(opts.Date, time.Now().Format("2006-01-02")) {
		title = fmt.Sprintf("Today's Work Summary - %s", opts.Date)
	} else {
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		if strings.HasPrefix(opts.Date, yesterday) {
			title = fmt.Sprintf("Yesterday's Work Summary - %s", opts.Date)
		} else {
			title = fmt.Sprintf("Work Summary - %s", opts.Date)
		}
	}
	output.WriteString(renderStyle(headerStyle, title, opts.NoColor) + "\n")
	output.WriteString(renderStyle(separatorStyle, "========================================", opts.NoColor) + "\n\n")

	// Check if no commands
	if len(grouped.Contexts) == 0 {
		output.WriteString(renderStyle(statLabelStyle, fmt.Sprintf("No commands found for %s", opts.Date), opts.NoColor) + "\n")
		return output.String()
	}

	// Sort contexts by working directory
	contexts := sortContexts(grouped.Contexts)

	// Statistics trackers
	totalCommands := 0
	totalBranches := 0
	branchSet := make(map[string]bool)
	repoCount := 0
	nonRepoCount := 0

	// Format each context
	for _, contextKey := range contexts {
		branches := grouped.Contexts[contextKey]

		workingDir := TildePath(contextKey.WorkingDir)
		// Context header
		if contextKey.GitRepo != "" {
			repo := strings.Replace(contextKey.GitRepo, "git@github.com", "GH", 1)
			output.WriteString(fmt.Sprintf("%s (%s)\n",
				renderStyle(contextStyle, workingDir, opts.NoColor),
				renderStyle(gitStyle, repo, opts.NoColor)))
			repoCount++
		} else {
			output.WriteString(renderStyle(contextStyle, workingDir, opts.NoColor) + "\n")
			nonRepoCount++
		}

		// Sort branches alphabetically
		sortedBranches := sortBranches(branches)

		// Format each branch
		for _, branchKey := range sortedBranches {
			commands := branches[branchKey]

			// Track branch statistics
			if branchKey != NoBranch {
				branchSet[string(branchKey)] = true
			}
			totalCommands += len(commands)

			// Branch header
			if branchKey == NoBranch {
				if contextKey.GitRepo != "" {
					output.WriteString("  " + renderStyle(branchStyle, "No branch", opts.NoColor) + "\n")
				} else {
					output.WriteString("  " + renderStyle(branchStyle, "No git repository", opts.NoColor) + "\n")
				}
			} else {
				output.WriteString(fmt.Sprintf("  %s: %s\n",
					renderStyle(statLabelStyle, "Branch", opts.NoColor),
					renderStyle(branchStyle, string(branchKey), opts.NoColor)))
			}

			// Bucket commands by time period
			buckets := BucketByTimePeriod(commands)

			// Format each time period in chronological order
			periods := GetOrderedPeriods()
			for _, period := range periods {
				bucket := buckets[period]
				if bucket == nil {
					continue
				}

				// Format time range
				firstTime := time.Unix(bucket.FirstTime, 0).Format("3:04pm")
				lastTime := time.Unix(bucket.LastTime, 0).Format("3:04pm")
				commandCount := len(bucket.Commands)

				if commandCount == 1 {
					output.WriteString(fmt.Sprintf("    %s (%s) - %s\n",
						renderStyle(periodStyle, string(period), opts.NoColor),
						renderStyle(timeRangeStyle, firstTime, opts.NoColor),
						renderStyle(statValueStyle, fmt.Sprintf("%d commands", commandCount), opts.NoColor)))
				} else {
					output.WriteString(fmt.Sprintf("    %s (%s) - %s\n",
						renderStyle(periodStyle, string(period), opts.NoColor),
						renderStyle(timeRangeStyle, firstTime+" - "+lastTime, opts.NoColor),
						renderStyle(statValueStyle, fmt.Sprintf("%d commands", commandCount), opts.NoColor)))
				}

				// If --all-commands, show each command with timestamp
				if opts.AllCommands {
					for _, cmd := range bucket.Commands {
						timestamp := time.Unix(cmd.Timestamp, 0).Format("3:04pm")
						output.WriteString(fmt.Sprintf("      %s  %s\n",
							renderStyle(timestampStyle, timestamp, opts.NoColor),
							renderStyle(commandStyle, cmd.CommandText, opts.NoColor)))
					}
				}
				output.WriteString("\n")
			}
		}
	}

	// Summary statistics
	totalBranches = len(branchSet)
	output.WriteString(renderStyle(headerStyle, "Summary Statistics:", opts.NoColor) + "\n")
	output.WriteString(fmt.Sprintf("  %s: %s\n",
		renderStyle(statLabelStyle, "Total commands", opts.NoColor),
		renderStyle(statValueStyle, fmt.Sprintf("%d", totalCommands), opts.NoColor)))

	// Format unique contexts string
	totalContexts := repoCount + nonRepoCount
	if repoCount > 0 && nonRepoCount > 0 {
		output.WriteString(fmt.Sprintf("  %s: %s (%d repos, %d non-repo dir)\n",
			renderStyle(statLabelStyle, "Unique contexts", opts.NoColor),
			renderStyle(statValueStyle, fmt.Sprintf("%d", totalContexts), opts.NoColor),
			repoCount, nonRepoCount))
	} else if repoCount > 0 {
		if repoCount == 1 {
			output.WriteString(fmt.Sprintf("  %s: %s (1 repo)\n",
				renderStyle(statLabelStyle, "Unique contexts", opts.NoColor),
				renderStyle(statValueStyle, fmt.Sprintf("%d", totalContexts), opts.NoColor)))
		} else {
			output.WriteString(fmt.Sprintf("  %s: %s (%d repos)\n",
				renderStyle(statLabelStyle, "Unique contexts", opts.NoColor),
				renderStyle(statValueStyle, fmt.Sprintf("%d", totalContexts), opts.NoColor),
				repoCount))
		}
	} else {
		if nonRepoCount == 1 {
			output.WriteString(fmt.Sprintf("  %s: %s (1 non-repo dir)\n",
				renderStyle(statLabelStyle, "Unique contexts", opts.NoColor),
				renderStyle(statValueStyle, fmt.Sprintf("%d", totalContexts), opts.NoColor)))
		} else {
			output.WriteString(fmt.Sprintf("  %s: %s (%d non-repo dirs)\n",
				renderStyle(statLabelStyle, "Unique contexts", opts.NoColor),
				renderStyle(statValueStyle, fmt.Sprintf("%d", totalContexts), opts.NoColor),
				nonRepoCount))
		}
	}

	if totalBranches > 0 {
		output.WriteString(fmt.Sprintf("  %s: %s\n",
			renderStyle(statLabelStyle, "Branches worked on", opts.NoColor),
			renderStyle(statValueStyle, fmt.Sprintf("%d", totalBranches), opts.NoColor)))
	}

	return output.String()
}

// sortContexts returns context keys sorted alphabetically by working directory
func sortContexts(contexts map[ContextKey]map[BranchKey][]models.Command) []ContextKey {
	keys := make([]ContextKey, 0, len(contexts))
	for k := range contexts {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].WorkingDir < keys[j].WorkingDir
	})

	return keys
}

// sortBranches returns branch keys sorted alphabetically (with "No branch" last)
func sortBranches(branches map[BranchKey][]models.Command) []BranchKey {
	keys := make([]BranchKey, 0, len(branches))
	for k := range branches {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		// Put "No branch" at the end
		if keys[i] == NoBranch {
			return false
		}
		if keys[j] == NoBranch {
			return true
		}
		return keys[i] < keys[j]
	})

	return keys
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
