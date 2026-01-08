package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

var (
	listLimit     int
	listFormat    string
	listToday     bool
	listYesterday bool
	listThisWeek  bool
	listLastWeek  bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent commands",
	Long:  "Display recent commands from history, ordered by most recent first",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().IntVarP(&listLimit, "limit", "n", 20, "Maximum number of commands to display")
	listCmd.Flags().StringVar(&listFormat, "fmt", "", "Format output with comma-separated columns (timestamp,status,pwd,cmd,gb,gr,durs,durms)")
	listCmd.Flags().BoolVar(&listToday, "today", false, "Show only commands from today")
	listCmd.Flags().BoolVar(&listYesterday, "yesterday", false, "Show only commands from yesterday")
	listCmd.Flags().BoolVar(&listThisWeek, "this-week", false, "Show only commands from this week")
	listCmd.Flags().BoolVar(&listLastWeek, "last-week", false, "Show only commands from last week")
}

// formatDurationSeconds formats duration in milliseconds to human-readable format without milliseconds
// Examples: "0s", "2s", "1m12s", "1h12m0s", "1d4h0m0s"
func formatDurationSeconds(durationMs *int64) string {
	if durationMs == nil {
		return "0s"
	}

	totalMs := *durationMs
	totalSeconds := totalMs / 1000

	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	var result string
	if days > 0 {
		result = fmt.Sprintf("%dd%dh%dm%ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		result = fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	} else if minutes > 0 {
		result = fmt.Sprintf("%dm%ds", minutes, seconds)
	} else {
		result = fmt.Sprintf("%ds", seconds)
	}

	return result
}

// formatDurationMilliseconds formats duration in milliseconds to human-readable format with milliseconds
// Examples: "500ms", "2s28ms", "1m12s0ms", "1h12m0s28ms", "1d4h0m0s28ms"
func formatDurationMilliseconds(durationMs *int64) string {
	if durationMs == nil {
		return "0ms"
	}

	totalMs := *durationMs
	totalSeconds := totalMs / 1000
	ms := totalMs % 1000

	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	var result string
	if days > 0 {
		result = fmt.Sprintf("%dd%dh%dm%ds%dms", days, hours, minutes, seconds, ms)
	} else if hours > 0 {
		result = fmt.Sprintf("%dh%dm%ds%dms", hours, minutes, seconds, ms)
	} else if minutes > 0 {
		result = fmt.Sprintf("%dm%ds%dms", minutes, seconds, ms)
	} else if seconds > 0 {
		result = fmt.Sprintf("%ds%dms", seconds, ms)
	} else {
		result = fmt.Sprintf("%dms", ms)
	}

	return result
}

func runList(cmd *cobra.Command, args []string) error {
	// Open database
	database, err := db.New(dbPath)
	if err != nil {
		// Check if it's a "file doesn't exist" error
		if os.IsNotExist(err) || (err.Error() != "" && os.IsNotExist(fmt.Errorf("%w", err))) {
			return fmt.Errorf("database doesn't exist (run a command first or use 'shy insert' to add history)")
		}
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Calculate time range based on flags
	var startTime, endTime int64
	now := time.Now()

	if listToday {
		// Start of today (00:00:00)
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
		// End of today (23:59:59)
		endTime = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()).Unix()
	} else if listYesterday {
		yesterday := now.AddDate(0, 0, -1)
		// Start of yesterday (00:00:00)
		startTime = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location()).Unix()
		// End of yesterday (23:59:59)
		endTime = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 0, yesterday.Location()).Unix()
	} else if listThisWeek {
		// Find the most recent Monday (start of this week)
		weekday := int(now.Weekday())
		if weekday == 0 { // Sunday
			weekday = 7
		}
		daysFromMonday := weekday - 1
		monday := now.AddDate(0, 0, -daysFromMonday)
		// Start of Monday (00:00:00)
		startTime = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location()).Unix()
		// End of Sunday (23:59:59)
		sunday := monday.AddDate(0, 0, 6)
		endTime = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 23, 59, 59, 0, sunday.Location()).Unix()
	} else if listLastWeek {
		// Find last week's Monday
		weekday := int(now.Weekday())
		if weekday == 0 { // Sunday
			weekday = 7
		}
		daysFromMonday := weekday - 1
		thisMonday := now.AddDate(0, 0, -daysFromMonday)
		lastMonday := thisMonday.AddDate(0, 0, -7)
		// Start of last Monday (00:00:00)
		startTime = time.Date(lastMonday.Year(), lastMonday.Month(), lastMonday.Day(), 0, 0, 0, 0, lastMonday.Location()).Unix()
		// End of last Sunday (23:59:59)
		lastSunday := lastMonday.AddDate(0, 0, 6)
		endTime = time.Date(lastSunday.Year(), lastSunday.Month(), lastSunday.Day(), 23, 59, 59, 0, lastSunday.Location()).Unix()
	}

	// List commands
	var commands []models.Command
	if startTime > 0 || endTime > 0 {
		commands, err = database.ListCommandsInRange(startTime, endTime, listLimit)
	} else {
		commands, err = database.ListCommands(listLimit)
	}
	if err != nil {
		return fmt.Errorf("failed to list commands: %w", err)
	}

	// Handle empty result
	if len(commands) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No commands found")
		return nil
	}

	// Display commands
	if listFormat != "" {
		// Format output based on specified columns
		columns := strings.Split(listFormat, ",")
		for _, c := range commands {
			var parts []string
			for _, col := range columns {
				col = strings.TrimSpace(col)
				switch col {
				case "timestamp":
					parts = append(parts, time.Unix(c.Timestamp, 0).Format("2006-01-02 15:04:05"))
				case "status":
					parts = append(parts, fmt.Sprintf("%d", c.ExitStatus))
				case "pwd":
					parts = append(parts, c.WorkingDir)
				case "cmd":
					parts = append(parts, c.CommandText)
				case "gb": // git branch
					if c.GitBranch != nil && *c.GitBranch != "" {
						parts = append(parts, *c.GitBranch)
					} else {
						parts = append(parts, "")
					}
				case "gr": // git repo
					if c.GitRepo != nil && *c.GitRepo != "" {
						parts = append(parts, *c.GitRepo)
					} else {
						parts = append(parts, "")
					}
				case "durs": // duration in seconds format
					parts = append(parts, formatDurationSeconds(c.Duration))
				case "durms": // duration in milliseconds format
					parts = append(parts, formatDurationMilliseconds(c.Duration))
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", strings.Join(parts, "\t"))
		}
	} else {
		// Default: one per line, just the command text
		for _, c := range commands {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", c.CommandText)
		}
	}

	return nil
}
