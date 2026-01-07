package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var (
	fcList    bool
	fcNoNum   bool
	fcReverse bool
	fcLast    int
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
		if fcNoNum {
			fmt.Fprintln(cmd.OutOrStdout(), c.CommandText)
		} else {
			// Format with event number - right-aligned, matching zsh format
			fmt.Fprintf(cmd.OutOrStdout(), "%5d  %s\n", c.ID, c.CommandText)
		}
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
