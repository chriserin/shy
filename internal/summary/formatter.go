package summary

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/chris/shy/pkg/models"
)

// Styles for summary output (matching television theme palette from cmd/tv.go)
var (
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true) // bright-magenta
	contextStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))            // white
	branchStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))            // bright-blue
	periodStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true) // bright-yellow (for period name)
	timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // bright-yellow (for command timestamps)
	commandStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))            // white
	statLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))            // bright-blue
	statValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))            // bright-green
	separatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // bright-black
)

// FormatOptions contains options for formatting the summary
type FormatOptions struct {
	AllCommands   bool // Show all commands or just summary
	UniqCommands  bool
	MultiCommands bool
	Date          string // Date being summarized (YYYY-MM-DD format)
	NoColor       bool   // Disable color output
	BucketSize    BucketSize
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
	title := fmt.Sprintf("Work Summary - %s", opts.Date)
	separator := renderStyle(separatorStyle, strings.Repeat("=", max(40-(ansi.StringWidth(title)/2), 0)), opts.NoColor)
	output.WriteString(fmt.Sprintf("\n%s %s %s\n\n", separator, renderStyle(headerStyle, title, opts.NoColor), separator))

	// Check if no commands
	if len(grouped.Contexts) == 0 {
		output.WriteString(renderStyle(statLabelStyle, fmt.Sprintf("No commands found for %s", opts.Date), opts.NoColor) + "\n")
		return output.String()
	}

	// Sort contexts by working directory
	contexts := sortContexts(grouped.Contexts)

	// Format each context
	for _, contextKey := range contexts {
		branches := grouped.Contexts[contextKey]

		workingDir := TildePath(contextKey.WorkingDir)

		// Sort branches alphabetically
		sortedBranches := sortBranches(branches)

		// Format each branch
		for _, branchKey := range sortedBranches {
			commands := branches[branchKey]

			// Context header with directory:branch on one line
			var contextLine string
			if contextKey.GitRepo != "" {

				if branchKey == NoBranch {
					contextLine = fmt.Sprintf("%s:%s",
						renderStyle(contextStyle, workingDir, opts.NoColor),
						renderStyle(branchStyle, "No branch", opts.NoColor),
					)
				} else {
					contextLine = fmt.Sprintf("%s:%s",
						renderStyle(contextStyle, workingDir, opts.NoColor),
						renderStyle(branchStyle, string(branchKey), opts.NoColor),
					)
				}
			} else {
				contextLine = fmt.Sprintf("%s",
					renderStyle(contextStyle, workingDir, opts.NoColor))
			}

			output.WriteString(contextLine + "\n")

			// Bucket commands by hour
			buckets := BucketBy(commands, opts.BucketSize)

			// Format each hour in chronological order
			bucketIDs := GetOrderedBuckets(buckets)
			for _, bucketID := range bucketIDs {
				bucket := buckets[bucketID]
				bucketLabel := bucket.FormatLabel()

				// Hour separator line with dashes
				fmt.Fprintf(&output, "  %s %s\n",
					renderStyle(periodStyle, bucketLabel, opts.NoColor),
					renderStyle(separatorStyle, strings.Repeat("-", 80-2-len(bucketLabel)), opts.NoColor))

				// If --all-commands, show each command with timestamp
				if opts.AllCommands {
					for _, cmd := range bucket.Commands {
						timestamp := formatTimestamp(cmd, opts.BucketSize)

						fmt.Fprintf(&output, "    %s  %s\n",
							renderStyle(timestampStyle, timestamp, opts.NoColor),
							renderStyle(commandStyle, truncate(cmd.CommandText), opts.NoColor))
					}
				}

				if opts.MultiCommands {
					// sort map keys by value
					commandsSorted := make([]string, 0, len(bucket.CommandCounts))
					for cmd := range bucket.CommandCounts {
						commandsSorted = append(commandsSorted, cmd)
					}
					sort.Slice(commandsSorted, func(i, j int) bool {
						return bucket.CommandCounts[commandsSorted[i]] > bucket.CommandCounts[commandsSorted[j]]
					})
					for _, cmd := range commandsSorted {
						count := bucket.CommandCounts[cmd]
						if count > 1 {
							fmt.Fprintf(&output, "    %s %s\n",
								renderStyle(statValueStyle.Width(4).Align(lipgloss.Left), fmt.Sprintf("⟳ %d", count), opts.NoColor),
								renderStyle(commandStyle, truncate(cmd), opts.NoColor))
						}
					}
				}

				if opts.UniqCommands {
					for _, cmd := range bucket.Commands {
						if bucket.CommandCounts[cmd.CommandText] == 1 {
							timestamp := formatTimestamp(cmd, opts.BucketSize)

							fmt.Fprintf(&output, "    %s  %s\n",
								renderStyle(timestampStyle, timestamp, opts.NoColor),
								renderStyle(commandStyle, truncate(cmd.CommandText), opts.NoColor))
						}
					}
				}

				if !opts.AllCommands && !opts.UniqCommands && !opts.MultiCommands {
					// If no detailed options, just show total commands in the hour
					fmt.Fprintf(&output, "    %s commands\n",
						renderStyle(commandStyle, fmt.Sprintf("%d", len(bucket.Commands)), opts.NoColor))
				}
				output.WriteString("\n")
			}
		}
	}

	return output.String()
}

func truncate(s string) string {
	if strings.Split(s, "\n"); len(strings.Split(s, "\n")) > 1 {
		return strings.Split(s, "\n")[0] + " ↵"
	}
	return s
}

func formatTimestamp(cmd models.Command, bucketSize BucketSize) string {
	t := time.Unix(cmd.Timestamp, 0)
	switch bucketSize {
	case Hourly:
		return t.Format(":04")
	case Periodically:
		return t.Format("03:04 PM")
	case Daily:
		return t.Format("03:04 PM")
	case Weekly:
		return t.Format("Jan 2 03:04 PM")
	}
	return t.Format("2006-01-02 15:04:05")
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
