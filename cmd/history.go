package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// parseHistoryArgsAndFlags manually parses arguments to handle negative numbers correctly
// Returns: positional args, parent flags (as alternating flag/value pairs), error
func parseHistoryArgsAndFlags(args []string) ([]string, []string, error) {
	var positional []string
	var parentFlags []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if it's a flag
		if strings.HasPrefix(arg, "-") {
			// Check if it's a negative number (starts with - and rest are digits)
			if len(arg) > 1 && isNumeric(arg[1:]) {
				// It's a negative number, treat as positional argument
				positional = append(positional, arg)
				continue
			}

			// It's a flag
			switch arg {
			case "-n", "--no-numbers":
				fcNoNum = true
			case "-r", "--reverse":
				fcReverse = true
			case "--last":
				if i+1 >= len(args) {
					return nil, nil, fmt.Errorf("--last requires a value")
				}
				i++
				val, err := strconv.Atoi(args[i])
				if err != nil {
					return nil, nil, fmt.Errorf("--last value must be a number: %w", err)
				}
				fcLast = val
			case "--db":
				// Parent flag - save it to process later
				if i+1 < len(args) {
					parentFlags = append(parentFlags, "db", args[i+1])
					i++
				}
			case "--":
				// Everything after -- is positional
				positional = append(positional, args[i+1:]...)
				return positional, parentFlags, nil
			default:
				return nil, nil, fmt.Errorf("unknown flag: %s", arg)
			}
		} else {
			// Positional argument
			positional = append(positional, arg)
		}
	}

	return positional, parentFlags, nil
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

var historyCmd = &cobra.Command{
	Use:                "history [first [last]]",
	Short:              "Display command history (alias for fc -l)",
	Long:               "Display command history with event numbers. This is an alias for 'fc -l'.",
	DisableFlagParsing: true, // We'll parse flags manually to handle negative numbers
	RunE: func(cmd *cobra.Command, args []string) error {
		// Manually parse flags to handle negative numbers correctly
		parsedArgs, parentFlags, err := parseHistoryArgsAndFlags(args)
		if err != nil {
			return err
		}

		// Process parent/root flags manually (like --db)
		for i := 0; i < len(parentFlags); i += 2 {
			if i+1 < len(parentFlags) {
				cmd.Parent().PersistentFlags().Set(parentFlags[i], parentFlags[i+1])
			}
		}

		// Set fcList to true since history is equivalent to fc -l
		fcList = true
		// Run fc command with parsed arguments
		return runFc(cmd, parsedArgs)
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
	// Note: We use DisableFlagParsing and parse flags manually to support negative numbers like -10
	// Supported flags: -n/--no-numbers, -r/--reverse, --last N
}
