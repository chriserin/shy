package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// historyFlags holds parsed flag values for the history command
type historyFlags struct {
	listModeFlags // embed common flags (defined in fc.go)
}

// parseHistoryArgsAndFlags manually parses arguments to handle negative numbers correctly
// Returns: positional args, parsed flags, parent flags (as alternating flag/value pairs), error
func parseHistoryArgsAndFlags(args []string) ([]string, historyFlags, []string, error) {
	var positional []string
	var parentFlags []string
	flags := historyFlags{}

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

			// Try parsing as a list-mode flag first
			newIndex, handled, err := parseListModeFlag(arg, args, i, &flags.listModeFlags)
			if err != nil {
				return nil, flags, nil, err
			}
			if handled {
				i = newIndex
				continue
			}

			// It's a flag specific to history (or common flag not handled above)
			switch arg {
			case "--db":
				// Parent flag - save it to process later
				if i+1 < len(args) {
					parentFlags = append(parentFlags, "db", args[i+1])
					i++
				}
			case "--":
				// Everything after -- is positional
				positional = append(positional, args[i+1:]...)
				return positional, flags, parentFlags, nil
			default:
				return nil, flags, nil, fmt.Errorf("unknown flag: %s", arg)
			}
		} else {
			// Positional argument
			positional = append(positional, arg)
		}
	}

	return positional, flags, parentFlags, nil
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
		parsedArgs, flags, parentFlags, err := parseHistoryArgsAndFlags(args)
		if err != nil {
			return err
		}

		cmd.SilenceUsage = true

		// Process parent/root flags manually (like --db)
		for i := 0; i < len(parentFlags); i += 2 {
			if i+1 < len(parentFlags) {
				cmd.Parent().PersistentFlags().Set(parentFlags[i], parentFlags[i+1])
			}
		}

		// Set flags on fcCmd so runFc can read them
		fcCmd.Flags().Set("list", "true") // history is equivalent to fc -l
		fcCmd.Flags().Set("no-numbers", fmt.Sprintf("%t", flags.noNum))
		fcCmd.Flags().Set("reverse", fmt.Sprintf("%t", flags.reverse))
		fcCmd.Flags().Set("time", fmt.Sprintf("%t", flags.showTime))
		fcCmd.Flags().Set("iso", fmt.Sprintf("%t", flags.timeISO))
		fcCmd.Flags().Set("american", fmt.Sprintf("%t", flags.timeUS))
		fcCmd.Flags().Set("european", fmt.Sprintf("%t", flags.timeEU))
		fcCmd.Flags().Set("time-format", flags.timeCustom)
		fcCmd.Flags().Set("elapsed", fmt.Sprintf("%t", flags.elapsed))
		fcCmd.Flags().Set("match", flags.pattern)
		fcCmd.Flags().Set("internal", fmt.Sprintf("%t", flags.internal))
		fcCmd.Flags().Set("local", fmt.Sprintf("%t", flags.local))

		// Run fc command with parsed arguments
		// Pass fcCmd so it reads the flags we just set
		err = runFc(fcCmd, parsedArgs)

		// Reset fcCmd flags after use to avoid test pollution
		resetFcFlags(fcCmd)

		return err
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
	// Note: We use DisableFlagParsing and parse flags manually to support negative numbers like -10
	// But we still define flags here so they appear in --help output
	addListModeFlags(historyCmd)
}
