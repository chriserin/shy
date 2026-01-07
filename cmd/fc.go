package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ncruces/go-strftime"
	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var (
	fcList         bool
	fcNoNum        bool
	fcReverse      bool
	fcLast         int
	fcShowTime     bool
	fcTimeISO      bool
	fcTimeUS       bool
	fcTimeEU       bool
	fcTimeCustom   string
	fcElapsedTime  bool
)

var fcCmd = &cobra.Command{
	Use:   "fc [flags] [first [last]]",
	Short: "Process command history (fc builtin)",
	Long:  "Process the command history list. With -l flag, lists commands. Without -l, edits and re-executes commands.",
	RunE:  runFc,
}

func init() {
	rootCmd.AddCommand(fcCmd)
	fcCmd.Flags().BoolVarP(&fcList, "list", "l", false, "List commands instead of editing")
	fcCmd.Flags().BoolVarP(&fcNoNum, "no-numbers", "n", false, "Suppress event numbers when listing")
	fcCmd.Flags().BoolVarP(&fcReverse, "reverse", "r", false, "Reverse order (oldest first)")
	fcCmd.Flags().IntVar(&fcLast, "last", 0, "Show last N commands (e.g., --last 10 instead of -- -10)")
	fcCmd.Flags().BoolVarP(&fcShowTime, "time", "d", false, "Display timestamps")
	fcCmd.Flags().BoolVarP(&fcTimeISO, "iso", "i", false, "Display timestamps in ISO8601 format (yyyy-mm-dd hh:mm)")
	fcCmd.Flags().BoolVarP(&fcTimeUS, "american", "f", false, "Display timestamps in US format (mm/dd/yy hh:mm)")
	fcCmd.Flags().BoolVarP(&fcTimeEU, "european", "E", false, "Display timestamps in European format (dd.mm.yyyy hh:mm)")
	fcCmd.Flags().StringVarP(&fcTimeCustom, "time-format", "t", "", "Custom timestamp format (strftime)")
	fcCmd.Flags().BoolVarP(&fcElapsedTime, "elapsed", "D", false, "Display elapsed time since command")
}

// formatTimestamp formats a Unix timestamp based on the active flags
func formatTimestamp(timestamp int64) string {
	t := time.Unix(timestamp, 0).UTC()

	// Custom format takes precedence
	if fcTimeCustom != "" {
		return strftime.Format(fcTimeCustom, t)
	}

	// Then check specific formats
	if fcTimeISO {
		return strftime.Format("%Y-%m-%d %H:%M", t)
	}
	if fcTimeUS {
		return strftime.Format("%m/%d/%y %H:%M", t)
	}
	if fcTimeEU {
		return strftime.Format("%d.%m.%Y %H:%M", t)
	}

	// Default format for -d flag
	if fcShowTime {
		return strftime.Format("%Y-%m-%d %H:%M:%S", t)
	}

	return ""
}

// formatDuration formats a duration in milliseconds to mm:ss format
// Returns "00:00" for null/missing duration or durations >= 1 hour
func formatDuration(durationMs *int64) string {
	if durationMs == nil {
		return "00:00"
	}

	totalSeconds := *durationMs / 1000

	// Don't show hours - return empty or 00:00 for >= 1 hour
	if totalSeconds >= 3600 {
		return "" // Empty string for >= 1 hour
	}

	minutes := totalSeconds / 60
	seconds := totalSeconds % 60

	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func runFc(cmd *cobra.Command, args []string) error {
	// For now, only implement -l (list) mode
	if !fcList {
		return fmt.Errorf("fc: editing mode not yet implemented, use -l flag")
	}

	// Open database
	database, err := db.New(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Parse range arguments
	first, last, err := parseHistoryRange(args, database)
	if err != nil {
		return err
	}

	// Get commands in range
	commands, err := database.GetCommandsByRange(first, last)
	if err != nil {
		return fmt.Errorf("failed to get commands: %w", err)
	}

	// Apply reverse flag
	if fcReverse {
		// Reverse the slice
		for i, j := 0, len(commands)-1; i < j; i, j = i+1, j-1 {
			commands[i], commands[j] = commands[j], commands[i]
		}
	}

	// Output commands
	for _, c := range commands {
		// Build output line
		var line string

		// Add event number (unless -n flag is set)
		if !fcNoNum {
			line = fmt.Sprintf("%5d", c.ID)
		}

		// Add timestamp if any time flag is set
		timeStr := formatTimestamp(c.Timestamp)
		if timeStr != "" {
			if line != "" {
				line += "  "
			}
			line += timeStr
		}

		// Add duration if -D flag is set
		if fcElapsedTime {
			durationStr := formatDuration(c.Duration)
			if line != "" {
				line += "  "
			}
			line += durationStr
		}

		// Add command text
		if line != "" {
			line += "  "
		}
		line += c.CommandText

		fmt.Fprintln(cmd.OutOrStdout(), line)
	}

	return nil
}

// parseHistoryRange parses the first and last arguments for history/fc commands
// Returns (first_id, last_id, error)
func parseHistoryRange(args []string, database *db.DB) (int64, int64, error) {
	// Get the most recent event ID
	mostRecent, err := database.GetMostRecentEventID()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get most recent event: %w", err)
	}

	// If no commands in database, return empty range
	if mostRecent == 0 {
		return 0, -1, nil // Invalid range that will return no results
	}

	var first, last int64

	// Handle --last flag (convenience for negative offset)
	if fcLast > 0 && len(args) == 0 {
		first = mostRecent - int64(fcLast) + 1
		if first < 1 {
			first = 1
		}
		last = mostRecent
		return first, last, nil
	}

	switch len(args) {
	case 0:
		// Default: last 16 events
		first = mostRecent - 15
		if first < 1 {
			first = 1
		}
		last = mostRecent

	case 1:
		// One argument
		arg := args[0]
		if num, err := strconv.ParseInt(arg, 10, 64); err == nil {
			// It's a number
			if num < 0 {
				// Negative: last N events
				first = mostRecent + num + 1
				if first < 1 {
					first = 1
				}
				last = mostRecent
			} else {
				// Positive: from N to most recent
				first = num
				last = mostRecent
			}
		} else {
			// It's a string: find most recent match
			matchID, err := database.FindMostRecentMatching(arg)
			if err != nil {
				return 0, 0, err
			}
			if matchID == 0 {
				return 0, 0, fmt.Errorf("shy: event not found: %s", arg)
			}
			first = matchID
			last = mostRecent
		}

	case 2:
		// Two arguments
		arg1 := args[0]
		arg2 := args[1]

		// Parse first argument
		if num, err := strconv.ParseInt(arg1, 10, 64); err == nil {
			// First is a number
			if num < 0 {
				first = mostRecent + num + 1
				if first < 1 {
					first = 1
				}
			} else {
				first = num
			}
		} else {
			// First is a string: find most recent match
			matchID, err := database.FindMostRecentMatching(arg1)
			if err != nil {
				return 0, 0, err
			}
			if matchID == 0 {
				return 0, 0, fmt.Errorf("shy: event not found: %s", arg1)
			}
			first = matchID
		}

		// Parse second argument
		if num, err := strconv.ParseInt(arg2, 10, 64); err == nil {
			// Second is a number
			if num < 0 {
				last = mostRecent + num + 1
				if last < 1 {
					last = 1
				}
			} else {
				last = num
			}
		} else {
			// Second is a string: find most recent match before first
			matchID, err := database.FindMostRecentMatchingBefore(arg2, mostRecent)
			if err != nil {
				return 0, 0, err
			}
			if matchID == 0 {
				return 0, 0, fmt.Errorf("shy: event not found: %s", arg2)
			}
			last = matchID
		}

	default:
		return 0, 0, fmt.Errorf("fc: too many arguments")
	}

	return first, last, nil
}
