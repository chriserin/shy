package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ncruces/go-strftime"
	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

var (
	// osExit is a variable that can be overridden in tests
	osExit = os.Exit
)

var fcCmd = &cobra.Command{
	Use:                "fc [flags] [first [last]]",
	Short:              "Process command history (fc builtin)",
	Long:               "Process the command history list. With -l flag, lists commands. Without -l, edits and re-executes commands.",
	DisableFlagParsing: true, // We'll parse flags manually to handle negative numbers
	RunE: func(cmd *cobra.Command, args []string) error {
		// Manually parse flags to handle negative numbers correctly
		parsedArgs, flags, parentFlags, err := parseFcArgsAndFlags(args)
		if err != nil {
			return err
		}

		// Process parent/root flags manually (like --db)
		for i := 0; i < len(parentFlags); i += 2 {
			if i+1 < len(parentFlags) {
				cmd.Parent().PersistentFlags().Set(parentFlags[i], parentFlags[i+1])
			}
		}

		// Set flags on cmd so runFc can read them
		cmd.Flags().Set("list", fmt.Sprintf("%t", flags.list))
		cmd.Flags().Set("no-numbers", fmt.Sprintf("%t", flags.noNum))
		cmd.Flags().Set("reverse", fmt.Sprintf("%t", flags.reverse))
		cmd.Flags().Set("last", fmt.Sprintf("%d", flags.last))
		cmd.Flags().Set("time", fmt.Sprintf("%t", flags.showTime))
		cmd.Flags().Set("iso", fmt.Sprintf("%t", flags.timeISO))
		cmd.Flags().Set("american", fmt.Sprintf("%t", flags.timeUS))
		cmd.Flags().Set("european", fmt.Sprintf("%t", flags.timeEU))
		cmd.Flags().Set("time-format", flags.timeCustom)
		cmd.Flags().Set("elapsed", fmt.Sprintf("%t", flags.elapsed))
		cmd.Flags().Set("match", flags.pattern)
		cmd.Flags().Set("internal", fmt.Sprintf("%t", flags.internal))
		cmd.Flags().Set("local", fmt.Sprintf("%t", flags.local))
		cmd.Flags().Set("editor", flags.editor)
		cmd.Flags().Set("quick-exec", fmt.Sprintf("%t", flags.quickExec))

		// Run fc with parsed arguments
		err = runFc(cmd, parsedArgs)

		// Reset flags after use
		resetFcFlags(cmd)

		return err
	},
}

// fcFlags holds parsed flag values for the fc command
type fcFlags struct {
	list       bool
	noNum      bool
	reverse    bool
	last       int
	showTime   bool
	timeISO    bool
	timeUS     bool
	timeEU     bool
	timeCustom string
	elapsed    bool
	pattern    string
	internal   bool
	local      bool
	editor     string // -e flag: specify editor to use
	quickExec  bool   // -s flag: re-execute without editing
}

// parseFcArgsAndFlags manually parses arguments to handle negative numbers correctly
// Returns: positional args, parsed flags, parent flags (as alternating flag/value pairs), error
func parseFcArgsAndFlags(args []string) ([]string, fcFlags, []string, error) {
	var positional []string
	var parentFlags []string
	flags := fcFlags{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if it's a substitution pattern (contains '=' and not a flag value)
		// This must be checked before flag parsing to handle cases like "--verbose="
		if strings.Contains(arg, "=") && strings.HasPrefix(arg, "-") {
			// This could be a substitution like "--verbose=" or a flag like "--flag=value"
			// If there's no value after the flag name in a --flag=value pattern,
			// treat it as a substitution
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				// Check if this looks like a flag assignment (e.g., --db=path) vs substitution
				// Substitutions will have the pattern on the left side
				// Flags will have a flag name on the left side
				// For now, we'll treat anything with = as a potential substitution
				// and let the flag parser decide if it's a real flag

				// Actually, for flags we handle explicitly (like --db), they're processed separately
				// So we can treat things with = that start with - as substitutions
				positional = append(positional, arg)
				continue
			}
		}

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
			case "-l", "--list":
				flags.list = true
			case "-n", "--no-numbers":
				flags.noNum = true
			case "-r", "--reverse":
				flags.reverse = true
			case "--last":
				if i+1 >= len(args) {
					return nil, flags, nil, fmt.Errorf("--last requires a value")
				}
				i++
				val, err := strconv.Atoi(args[i])
				if err != nil {
					return nil, flags, nil, fmt.Errorf("--last value must be a number: %w", err)
				}
				flags.last = val
			case "-d", "--time":
				flags.showTime = true
			case "-i", "--iso":
				flags.timeISO = true
			case "-f", "--american":
				flags.timeUS = true
			case "-E", "--european":
				flags.timeEU = true
			case "-t", "--time-format":
				if i+1 >= len(args) {
					return nil, flags, nil, fmt.Errorf("-t requires a format string")
				}
				i++
				flags.timeCustom = args[i]
			case "-D", "--elapsed":
				flags.elapsed = true
			case "-m", "--match":
				if i+1 >= len(args) {
					return nil, flags, nil, fmt.Errorf("-m requires a pattern")
				}
				i++
				flags.pattern = args[i]
			case "-I", "--internal":
				flags.internal = true
			case "-L", "--local":
				flags.local = true
			case "-e", "--editor":
				if i+1 >= len(args) {
					return nil, flags, nil, fmt.Errorf("-e requires an editor path")
				}
				i++
				flags.editor = args[i]
			case "-s", "--quick-exec":
				flags.quickExec = true
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

func init() {
	rootCmd.AddCommand(fcCmd)
	fcCmd.Flags().BoolP("list", "l", false, "List commands instead of editing")
	fcCmd.Flags().BoolP("no-numbers", "n", false, "Suppress event numbers when listing")
	fcCmd.Flags().BoolP("reverse", "r", false, "Reverse order (oldest first)")
	fcCmd.Flags().Int("last", 0, "Show last N commands (e.g., --last 10 instead of -- -10)")
	fcCmd.Flags().BoolP("time", "d", false, "Display timestamps")
	fcCmd.Flags().BoolP("iso", "i", false, "Display timestamps in ISO8601 format (yyyy-mm-dd hh:mm)")
	fcCmd.Flags().BoolP("american", "f", false, "Display timestamps in US format (mm/dd/yy hh:mm)")
	fcCmd.Flags().BoolP("european", "E", false, "Display timestamps in European format (dd.mm.yyyy hh:mm)")
	fcCmd.Flags().StringP("time-format", "t", "", "Custom timestamp format (strftime)")
	fcCmd.Flags().BoolP("elapsed", "D", false, "Display elapsed time since command")
	fcCmd.Flags().StringP("match", "m", "", "Filter by glob pattern")
	fcCmd.Flags().BoolP("internal", "I", false, "Show only commands from current session")
	fcCmd.Flags().BoolP("local", "L", false, "Show only local commands (currently same as no filter)")
	fcCmd.Flags().StringP("editor", "e", "", "Specify editor to use")
	fcCmd.Flags().BoolP("quick-exec", "s", false, "Re-execute without editing")
}

// resetFcFlags resets all fc flags to their default values (for testing)
func resetFcFlags(cmd *cobra.Command) {
	cmd.Flags().Set("list", "false")
	cmd.Flags().Set("no-numbers", "false")
	cmd.Flags().Set("reverse", "false")
	cmd.Flags().Set("last", "0")
	cmd.Flags().Set("time", "false")
	cmd.Flags().Set("iso", "false")
	cmd.Flags().Set("american", "false")
	cmd.Flags().Set("european", "false")
	cmd.Flags().Set("time-format", "")
	cmd.Flags().Set("elapsed", "false")
	cmd.Flags().Set("match", "")
	cmd.Flags().Set("internal", "false")
	cmd.Flags().Set("local", "false")
	cmd.Flags().Set("editor", "")
	cmd.Flags().Set("quick-exec", "false")
}

// getSessionPid retrieves the current session PID from the SHY_SESSION_PID environment variable
func getSessionPid() (int64, error) {
	pidStr := os.Getenv("SHY_SESSION_PID")
	if pidStr == "" {
		return 0, fmt.Errorf("fc -I: SHY_SESSION_PID environment variable not set")
	}

	pid, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("fc -I: invalid SHY_SESSION_PID value: %w", err)
	}

	return pid, nil
}

// formatTimestamp formats a Unix timestamp based on the active flags
func formatTimestamp(timestamp int64, timeCustom string, timeISO, timeUS, timeEU, showTime bool) string {
	t := time.Unix(timestamp, 0).UTC()

	// Custom format takes precedence
	if timeCustom != "" {
		return strftime.Format(timeCustom, t)
	}

	// Then check specific formats
	if timeISO {
		return strftime.Format("%Y-%m-%d %H:%M", t)
	}
	if timeUS {
		return strftime.Format("%m/%d/%y %H:%M", t)
	}
	if timeEU {
		return strftime.Format("%d.%m.%Y %H:%M", t)
	}

	// Default format for -d flag
	if showTime {
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

// globToLike translates a glob pattern to SQL LIKE pattern
// Glob wildcards: * (zero or more chars), ? (exactly one char)
// SQL wildcards: % (zero or more chars), _ (exactly one char)
func globToLike(pattern string) string {
	// Escape existing SQL wildcards in the pattern
	escaped := strings.ReplaceAll(pattern, "\\", "\\\\") // Escape backslashes first
	escaped = strings.ReplaceAll(escaped, "%", "\\%")
	escaped = strings.ReplaceAll(escaped, "_", "\\_")

	// Translate glob wildcards to SQL wildcards
	escaped = strings.ReplaceAll(escaped, "*", "%")
	escaped = strings.ReplaceAll(escaped, "?", "_")

	return escaped
}

func runFc(cmd *cobra.Command, args []string) error {
	// Get all flag values
	fcList, _ := cmd.Flags().GetBool("list")
	fcNoNum, _ := cmd.Flags().GetBool("no-numbers")
	fcReverse, _ := cmd.Flags().GetBool("reverse")
	fcLast, _ := cmd.Flags().GetInt("last")
	fcShowTime, _ := cmd.Flags().GetBool("time")
	fcTimeISO, _ := cmd.Flags().GetBool("iso")
	fcTimeUS, _ := cmd.Flags().GetBool("american")
	fcTimeEU, _ := cmd.Flags().GetBool("european")
	fcTimeCustom, _ := cmd.Flags().GetString("time-format")
	fcElapsedTime, _ := cmd.Flags().GetBool("elapsed")
	fcPattern, _ := cmd.Flags().GetString("match")
	fcInternal, _ := cmd.Flags().GetBool("internal")
	fcLocal, _ := cmd.Flags().GetBool("local")
	fcEditor, _ := cmd.Flags().GetString("editor")
	fcQuickExec, _ := cmd.Flags().GetBool("quick-exec")

	// -L (local) flag is currently a no-op placeholder for future remote sync functionality
	_ = fcLocal

	// Validate flag combinations
	if fcQuickExec && fcEditor != "" {
		return fmt.Errorf("cannot use -s and -e together")
	}

	// Open database BEFORE branching to list/edit mode
	database, err := db.New(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Parse substitutions (old=new patterns)
	substitutions, remainingArgs, err := parseSubstitutions(args)
	if err != nil {
		return err
	}

	// Parse range arguments (after extracting substitutions)
	// Pass fcList to determine default behavior: edit mode defaults to last 1 command, list mode to last 16
	first, last, err := parseHistoryRange(remainingArgs, database, fcLast, fcList)
	if err != nil {
		// Print error cleanly without usage
		fmt.Fprintln(cmd.OutOrStderr(), "shy", err.Error())
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		osExit(1)
		return nil
	}

	// Branch based on mode
	if !fcList {
		// Edit-and-execute mode
		return editAndExecuteMode(cmd, database, first, last, substitutions, fcPattern, fcInternal, fcEditor, fcQuickExec)
	}

	// List mode continues below

	// Get commands in range, with optional pattern filtering and/or internal filtering
	var commands []models.Command
	if fcInternal {
		// Get current session PID from environment variable
		sessionPid, err := getSessionPid()
		if err != nil {
			return err
		}

		if fcPattern != "" {
			// Both internal and pattern filtering
			likePattern := globToLike(fcPattern)
			commands, err = database.GetCommandsByRangeWithPatternInternal(first, last, sessionPid, likePattern)
			if err != nil {
				return fmt.Errorf("failed to get commands: %w", err)
			}
		} else {
			// Internal filtering only
			commands, err = database.GetCommandsByRangeInternal(first, last, sessionPid)
			if err != nil {
				return fmt.Errorf("failed to get commands: %w", err)
			}
		}

		// Return exit code 1 if no matches found
		if len(commands) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "shy fc: no matching events found")
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			osExit(1)
			return nil
		}
	} else if fcPattern != "" {
		// Pattern filtering only
		likePattern := globToLike(fcPattern)
		commands, err = database.GetCommandsByRangeWithPattern(first, last, likePattern)
		if err != nil {
			return fmt.Errorf("failed to get commands: %w", err)
		}
		// Return exit code 1 if pattern finds no matches
		if len(commands) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "shy fc: no matching events found")
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			osExit(1)
			return nil
		}
	} else {
		// No filtering
		commands, err = database.GetCommandsByRange(first, last)
		if err != nil {
			return fmt.Errorf("failed to get commands: %w", err)
		}
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
		// Apply substitutions to command text
		commandText := c.CommandText
		if len(substitutions) > 0 {
			commandText = applySubstitutions(commandText, substitutions)
		}

		// Build output line
		var line string

		// Add event number (unless -n flag is set)
		if !fcNoNum {
			line = fmt.Sprintf("%5d", c.ID)
		}

		// Add timestamp if any time flag is set
		timeStr := formatTimestamp(c.Timestamp, fcTimeCustom, fcTimeISO, fcTimeUS, fcTimeEU, fcShowTime)
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

		// Add command text (with substitutions applied)
		if line != "" {
			line += "  "
		}
		line += commandText

		fmt.Fprintln(cmd.OutOrStdout(), line)
	}

	return nil
}

// substitution represents an old=new string substitution
type substitution struct {
	old string
	new string
}

// parseSubstitutions extracts old=new substitution patterns from args
// Substitutions MUST come before any range arguments
// Returns the substitutions and the remaining args
func parseSubstitutions(args []string) ([]substitution, []string, error) {
	var substitutions []substitution
	var remaining []string
	foundNonSubstitution := false

	for _, arg := range args {
		// Check if this is a substitution (contains '=')
		// Note: args starting with '-' that contain '=' were already filtered as substitutions
		// by parseFcArgsAndFlags, so we don't need to exclude them here
		isSubstitution := strings.Contains(arg, "=")

		if isSubstitution {
			// If we already found a non-substitution arg, substitutions are not allowed anymore
			if foundNonSubstitution {
				// This substitution came after range args, treat it as a regular arg
				remaining = append(remaining, arg)
			} else {
				// Valid substitution before range args
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) == 2 {
					substitutions = append(substitutions, substitution{
						old: parts[0],
						new: parts[1],
					})
				} else {
					return nil, nil, fmt.Errorf("invalid substitution format: %s", arg)
				}
			}
		} else {
			// Non-substitution argument (range arg)
			foundNonSubstitution = true
			remaining = append(remaining, arg)
		}
	}

	return substitutions, remaining, nil
}

// applySubstitutions applies all substitutions to a command text
func applySubstitutions(text string, subs []substitution) string {
	result := text
	for _, sub := range subs {
		result = strings.ReplaceAll(result, sub.old, sub.new)
	}
	return result
}

// parseHistoryRange parses the first and last arguments for history/fc commands
// Returns (first_id, last_id, error)
// listMode: true for list mode (fc -l), false for edit mode (fc without -l)
func parseHistoryRange(args []string, database *db.DB, lastN int, listMode bool) (int64, int64, error) {
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
	if lastN > 0 && len(args) == 0 {
		first = mostRecent - int64(lastN) + 1
		if first < 1 {
			first = 1
		}
		last = mostRecent
		return first, last, nil
	}

	switch len(args) {
	case 0:
		// Default behavior depends on mode:
		// - List mode (fc -l): last 16 events
		// - Edit mode (fc): last 1 event
		if listMode {
			first = mostRecent - 15
			if first < 1 {
				first = 1
			}
		} else {
			// Edit mode: just the last command
			first = mostRecent
		}
		last = mostRecent

	case 1:
		// One argument
		arg := args[0]
		if num, err := strconv.ParseInt(arg, 10, 64); err == nil {
			// It's a number
			if num < 0 {
				// Negative number: convert to event ID
				eventID := mostRecent + num + 1
				if eventID < 1 {
					eventID = 1
				}
				first = eventID
				// Behavior depends on mode
				if listMode {
					// List mode: from event to most recent
					last = mostRecent
				} else {
					// Edit mode: just edit that event
					last = eventID
				}
			} else {
				// Positive number: behavior depends on mode
				if listMode {
					// List mode: from event num to most recent
					first = num
					last = mostRecent
				} else {
					// Edit mode: just edit event num
					first = num
					last = num
				}
			}
		} else {
			// It's a string: find most recent match
			matchID, err := database.FindMostRecentMatching(arg)
			if err != nil {
				return 0, 0, err
			}
			if matchID == 0 {
				return 0, 0, fmt.Errorf("fc: event not found: %s", arg)
			}
			first = matchID
			if listMode {
				// List mode: from matched event to most recent
				last = mostRecent
			} else {
				// Edit mode: just edit the matched event
				last = matchID
			}
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
				return 0, 0, fmt.Errorf("fc: event not found: %s", arg1)
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
				return 0, 0, fmt.Errorf("fc: event not found: %s", arg2)
			}
			last = matchID
		}

	default:
		return 0, 0, fmt.Errorf("fc: too many arguments")
	}

	return first, last, nil
}
