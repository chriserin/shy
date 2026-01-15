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
	tabsumDate string
)

var tabsumCmd = &cobra.Command{
	Use:   "tabsum",
	Short: "Show tabular summary of time spent in different contexts",
	Long:  "Display a tabular summary showing time spent in different working directories and git branches",
	RunE:  runTabsum,
}

func init() {
	rootCmd.AddCommand(tabsumCmd)

	// Add flags
	tabsumCmd.Flags().StringVar(&tabsumDate, "date", "yesterday", "Date to summarize (yesterday, today, or YYYY-MM-DD)")
}

func runTabsum(cmd *cobra.Command, args []string) error {
	// Parse date string to get start and end timestamps
	startTime, endTime, dateStr, isYesterday, err := parseTabsumDateRange(tabsumDate)
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

	// Query context summaries for date range
	summaries, err := database.GetContextSummary(startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to query context summaries: %w", err)
	}

	// Format and print tabular summary
	output := summary.FormatTable(summaries, dateStr, isYesterday)
	fmt.Fprint(cmd.OutOrStdout(), output)

	return nil
}

// parseTabsumDateRange parses a date string and returns Unix timestamp range (start, end)
// Supports: "yesterday", "today", "YYYY-MM-DD"
// Returns: startTime (inclusive), endTime (exclusive), dateStr (YYYY-MM-DD), isYesterday, error
func parseTabsumDateRange(dateStr string) (int64, int64, string, bool, error) {
	var targetDate time.Time
	var isYesterday bool

	switch strings.ToLower(dateStr) {
	case "yesterday":
		targetDate = time.Now().AddDate(0, 0, -1)
		isYesterday = true
	case "today":
		targetDate = time.Now()
		isYesterday = false
	default:
		// Try parsing as YYYY-MM-DD in local timezone
		parsed, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err != nil {
			return 0, 0, "", false, fmt.Errorf("date must be 'yesterday', 'today', or YYYY-MM-DD format")
		}
		targetDate = parsed
		isYesterday = false
	}

	// Get start of day (00:00:00) in local timezone
	year, month, day := targetDate.Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, time.Local)

	// Get end of day (next day at 00:00:00) for exclusive upper bound
	endOfDay := startOfDay.AddDate(0, 0, 1)

	// Format date string as YYYY-MM-DD
	formattedDate := startOfDay.Format("2006-01-02")

	return startOfDay.Unix(), endOfDay.Unix(), formattedDate, isYesterday, nil
}
