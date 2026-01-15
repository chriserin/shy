package summary

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/chris/shy/pkg/models"
)

// FormatOptions contains options for formatting the summary
type FormatOptions struct {
	AllCommands bool   // Show all commands or just summary
	Date        string // Date being summarized (YYYY-MM-DD format)
}

// FormatSummary formats the grouped and bucketed commands into human-readable text
func FormatSummary(grouped *GroupedCommands, opts FormatOptions) string {
	var output strings.Builder

	// Header
	if strings.HasPrefix(opts.Date, time.Now().Format("2006-01-02")) {
		output.WriteString(fmt.Sprintf("Today's Work Summary - %s\n", opts.Date))
	} else {
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		if strings.HasPrefix(opts.Date, yesterday) {
			output.WriteString(fmt.Sprintf("Yesterday's Work Summary - %s\n", opts.Date))
		} else {
			output.WriteString(fmt.Sprintf("Work Summary - %s\n", opts.Date))
		}
	}
	output.WriteString("========================================\n\n")

	// Check if no commands
	if len(grouped.Contexts) == 0 {
		output.WriteString(fmt.Sprintf("No commands found for %s\n", opts.Date))
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

		// Context header
		if contextKey.GitRepo != "" {
			output.WriteString(fmt.Sprintf("%s (%s)\n", contextKey.WorkingDir, contextKey.GitRepo))
			repoCount++
		} else {
			output.WriteString(fmt.Sprintf("%s\n", contextKey.WorkingDir))
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
					output.WriteString("  No branch\n")
				} else {
					output.WriteString("  No git repository\n")
				}
			} else {
				output.WriteString(fmt.Sprintf("  Branch: %s\n", branchKey))
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
					output.WriteString(fmt.Sprintf("    %s (%s) - %d commands\n", period, firstTime, commandCount))
				} else {
					output.WriteString(fmt.Sprintf("    %s (%s - %s) - %d commands\n", period, firstTime, lastTime, commandCount))
				}

				// If --all-commands, show each command with timestamp
				if opts.AllCommands {
					for _, cmd := range bucket.Commands {
						timestamp := time.Unix(cmd.Timestamp, 0).Format("15:04:05")
						output.WriteString(fmt.Sprintf("      %s  %s\n", timestamp, cmd.CommandText))
					}
				}
			}
			output.WriteString("\n")
		}
	}

	// Summary statistics
	totalBranches = len(branchSet)
	output.WriteString("Summary Statistics:\n")
	output.WriteString(fmt.Sprintf("  Total commands: %d\n", totalCommands))

	// Format unique contexts string
	totalContexts := repoCount + nonRepoCount
	if repoCount > 0 && nonRepoCount > 0 {
		output.WriteString(fmt.Sprintf("  Unique contexts: %d (%d repos, %d non-repo dir)\n", totalContexts, repoCount, nonRepoCount))
	} else if repoCount > 0 {
		if repoCount == 1 {
			output.WriteString(fmt.Sprintf("  Unique contexts: %d (1 repo)\n", totalContexts))
		} else {
			output.WriteString(fmt.Sprintf("  Unique contexts: %d (%d repos)\n", totalContexts, repoCount))
		}
	} else {
		if nonRepoCount == 1 {
			output.WriteString(fmt.Sprintf("  Unique contexts: %d (1 non-repo dir)\n", totalContexts))
		} else {
			output.WriteString(fmt.Sprintf("  Unique contexts: %d (%d non-repo dirs)\n", totalContexts, nonRepoCount))
		}
	}

	if totalBranches > 0 {
		output.WriteString(fmt.Sprintf("  Branches worked on: %d\n", totalBranches))
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
