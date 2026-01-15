package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/internal/summary"
)

var (
	summaryDate        string
	summaryAllCommands bool
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show summary of shell command activity",
	Long:  "Display a structured summary of shell commands grouped by repository/directory and branch, with timelines",
	RunE:  runSummary,
}

func init() {
	rootCmd.AddCommand(summaryCmd)

	// Add flags
	summaryCmd.Flags().StringVar(&summaryDate, "date", "yesterday", "Date to summarize (yesterday, today, or YYYY-MM-DD)")
	summaryCmd.Flags().BoolVar(&summaryAllCommands, "all-commands", false, "Display all commands in each time bucket")
}

func runSummary(cmd *cobra.Command, args []string) error {
	// Parse date string to get start and end timestamps
	startTime, endTime, dateStr, err := parseDateRange(summaryDate)
	if err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}

	// Open database
	database, err := db.New(dbPath)
	if err != nil {
		// If database doesn't exist, show error
		if os.IsNotExist(err) {
			return fmt.Errorf("database not found at %s", dbPath)
		}
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Query commands for date range
	commands, err := database.GetCommandsByDateRange(startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to query commands: %w", err)
	}

	// Group commands by context (repo/dir and branch)
	grouped := summary.GroupByContext(commands)

	// Format and print summary
	opts := summary.FormatOptions{
		AllCommands: summaryAllCommands,
		Date:        dateStr,
	}
	output := summary.FormatSummary(grouped, opts)
	fmt.Fprint(cmd.OutOrStdout(), output)

	return nil
}

// parseDateRange parses a date string and returns Unix timestamp range (start, end)
// Supports: "yesterday", "today", "YYYY-MM-DD"
// Returns: startTime (inclusive), endTime (exclusive), dateStr (YYYY-MM-DD), error
func parseDateRange(dateStr string) (int64, int64, string, error) {
	var targetDate time.Time

	switch strings.ToLower(dateStr) {
	case "yesterday":
		targetDate = time.Now().AddDate(0, 0, -1)
	case "today":
		targetDate = time.Now()
	default:
		// Try parsing as YYYY-MM-DD
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return 0, 0, "", fmt.Errorf("date must be 'yesterday', 'today', or YYYY-MM-DD format")
		}
		targetDate = parsed
	}

	// Get start of day (00:00:00) in local timezone
	year, month, day := targetDate.Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, time.Local)

	// Get end of day (start of next day) in local timezone
	endOfDay := startOfDay.AddDate(0, 0, 1)

	// Convert to Unix timestamps
	startTime := startOfDay.Unix()
	endTime := endOfDay.Unix()

	// Format date string as YYYY-MM-DD
	formattedDate := startOfDay.Format("2006-01-02")

	return startTime, endTime, formattedDate, nil
}
