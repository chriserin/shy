package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ncruces/go-strftime"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/internal/session"
	"github.com/chris/shy/pkg/models"
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
			cmd.SilenceUsage = true
			return err
		}

		if flags.help {
			cmd.Help()
			os.Exit(0)
		}

		cmd.SilenceUsage = true

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
		cmd.Flags().Set("write", flags.writeFile)
		cmd.Flags().Set("write-specified", fmt.Sprintf("%t", flags.writeSpecified))
		cmd.Flags().Set("append", flags.appendFile)
		cmd.Flags().Set("read", flags.readFile)
		cmd.Flags().Set("push", flags.pushDB)
		cmd.Flags().Set("pop", fmt.Sprintf("%t", flags.popDB))

		// Run fc with parsed arguments
		err = runFc(cmd, parsedArgs)

		// Reset flags after use
		resetFcFlags(cmd)

		return err
	},
}

// listModeFlags holds flags common to both fc -l and history commands
type listModeFlags struct {
	noNum      bool
	reverse    bool
	showTime   bool
	timeISO    bool
	timeUS     bool
	timeEU     bool
	timeCustom string
	elapsed    bool
	pattern    string
	internal   bool
	local      bool
}

// fcFlags holds parsed flag values for the fc command
type fcFlags struct {
	listModeFlags  // embed common flags
	list           bool
	editor         string // -e flag: specify editor to use
	quickExec      bool   // -s flag: re-execute without editing
	writeFile      string // -W flag: write history to file
	writeSpecified bool   // whether -W was specified (even without file)
	appendFile     string // -A flag: append history to file
	readFile       string // -R flag: read history from file
	pushDB         string // -p flag: push current database, start using new one
	popDB          bool   // -P flag: pop back to previous database
	help           bool
}

// parseListModeFlag attempts to parse a list-mode flag (common to fc -l and history)
// Returns: new index, whether the flag was handled, error
func parseListModeFlag(arg string, args []string, i int, flags *listModeFlags) (int, bool, error) {
	switch arg {
	case "-n", "--no-numbers":
		flags.noNum = true
		return i, true, nil
	case "-r", "--reverse":
		flags.reverse = true
		return i, true, nil
	case "-d", "--time":
		flags.showTime = true
		return i, true, nil
	case "-i", "--iso":
		flags.timeISO = true
		return i, true, nil
	case "-f", "--american":
		flags.timeUS = true
		return i, true, nil
	case "-E", "--european":
		flags.timeEU = true
		return i, true, nil
	case "-t", "--time-format":
		if i+1 >= len(args) {
			return i, true, fmt.Errorf("-t requires a format string")
		}
		flags.timeCustom = args[i+1]
		return i + 1, true, nil
	case "-D", "--elapsed":
		flags.elapsed = true
		return i, true, nil
	case "-m", "--match":
		if i+1 >= len(args) {
			return i, true, fmt.Errorf("-m requires a pattern")
		}
		flags.pattern = args[i+1]
		return i + 1, true, nil
	case "-I", "--internal":
		flags.internal = true
		return i, true, nil
	case "-L", "--local":
		flags.local = true
		return i, true, nil
	default:
		return i, false, nil
	}
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

			// Try parsing as a list-mode flag first
			newIndex, handled, err := parseListModeFlag(arg, args, i, &flags.listModeFlags)
			if err != nil {
				return nil, flags, nil, err
			}
			if handled {
				i = newIndex
				continue
			}

			// It's a flag specific to fc
			switch arg {
			case "-h", "--help":
				flags.help = true
			case "-l", "--list":
				flags.list = true
			case "-e", "--editor":
				if i+1 >= len(args) {
					return nil, flags, nil, fmt.Errorf("-e requires an editor path")
				}
				i++
				flags.editor = args[i]
			case "-s", "--quick-exec":
				flags.quickExec = true
			case "-W", "--write":
				flags.writeSpecified = true
				// -W can be specified without an argument (no-op case)
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					flags.writeFile = args[i]
				} else {
					flags.writeFile = "" // Empty string means -W without file
				}
			case "-A", "--append":
				if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
					return nil, flags, nil, fmt.Errorf("-A requires a file path")
				}
				i++
				flags.appendFile = args[i]
			case "-R", "--read":
				if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
					return nil, flags, nil, fmt.Errorf("-R requires a file path")
				}
				i++
				flags.readFile = args[i]
			case "-p", "--push":
				if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
					return nil, flags, nil, fmt.Errorf("-p requires a database path")
				}
				i++
				flags.pushDB = args[i]
			case "-P", "--pop":
				flags.popDB = true
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
				return nil, flags, nil, fmt.Errorf("shy fc: bad option: %s", arg)
			}
		} else {
			// Positional argument
			positional = append(positional, arg)
		}
	}

	// Validate mutually exclusive file operations
	fileOpCount := 0
	if flags.writeFile != "" {
		fileOpCount++
	}
	if flags.appendFile != "" {
		fileOpCount++
	}
	if flags.readFile != "" {
		fileOpCount++
	}
	if flags.pushDB != "" {
		fileOpCount++
	}
	if flags.popDB {
		fileOpCount++
	}
	if fileOpCount > 1 {
		return nil, flags, nil, fmt.Errorf("cannot use -W, -A, -R, -p, or -P together")
	}

	// Validate -s and -e are not used together
	if flags.quickExec && flags.editor != "" {
		return nil, flags, nil, fmt.Errorf("cannot use -s and -e together")
	}

	// Push/pop cannot be used with -l (list mode)
	if (flags.pushDB != "" || flags.popDB) && flags.list {
		return nil, flags, nil, fmt.Errorf("cannot use -p/-P with -l")
	}

	return positional, flags, parentFlags, nil
}

// addListModeFlags adds flags common to both fc -l and history commands
func addListModeFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("no-numbers", "n", false, "Suppress event numbers when listing")
	cmd.Flags().BoolP("reverse", "r", false, "Reverse order (oldest first)")
	cmd.Flags().BoolP("time", "d", false, "Display timestamps")
	cmd.Flags().BoolP("iso", "i", false, "Display timestamps in ISO8601 format (yyyy-mm-dd hh:mm)")
	cmd.Flags().BoolP("american", "f", false, "Display timestamps in US format (mm/dd/yy hh:mm)")
	cmd.Flags().BoolP("european", "E", false, "Display timestamps in European format (dd.mm.yyyy hh:mm)")
	cmd.Flags().StringP("time-format", "t", "", "Custom timestamp format (strftime)")
	cmd.Flags().BoolP("elapsed", "D", false, "Display elapsed time since command")
	cmd.Flags().StringP("match", "m", "", "Filter by glob pattern")
	cmd.Flags().BoolP("internal", "I", false, "Show only commands from current session")
	cmd.Flags().BoolP("local", "L", false, "Show only local commands (currently same as no filter)")
}

func init() {
	rootCmd.AddCommand(fcCmd)
	fcCmd.Flags().BoolP("list", "l", false, "List commands instead of editing")
	fcCmd.Flags().StringP("push", "p", "", "Push current database, start using new one")
	fcCmd.Flags().BoolP("pop", "P", false, "Pop back to previous database")
	addListModeFlags(fcCmd)
	fcCmd.Flags().StringP("editor", "e", "", "Specify editor to use")
	fcCmd.Flags().BoolP("quick-exec", "s", false, "Re-execute without editing")
	fcCmd.Flags().StringP("write", "W", "", "Write history to file")
	fcCmd.Flags().Bool("write-specified", false, "Internal: tracks if -W was specified")
	fcCmd.Flags().StringP("append", "A", "", "Append history to file")
	fcCmd.Flags().StringP("read", "R", "", "Read history from file")
	// Hide the internal write-specified flag from help
	fcCmd.Flags().MarkHidden("write-specified")
}

// resetFcFlags resets all fc flags to their default values (for testing)
func resetFcFlags(cmd *cobra.Command) {
	cmd.Flags().Set("list", "false")
	cmd.Flags().Set("no-numbers", "false")
	cmd.Flags().Set("reverse", "false")
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
	cmd.Flags().Set("write", "")
	cmd.Flags().Set("write-specified", "false")
	cmd.Flags().Set("append", "")
	cmd.Flags().Set("read", "")
	cmd.Flags().Set("push", "")
	cmd.Flags().Set("pop", "false")

	// Clear the "changed" status for all flags so they don't appear as modified
	cmd.Flags().Visit(func(f *pflag.Flag) {
		f.Changed = false
	})
}

// getSessionPid retrieves the current session PID from the SHY_SESSION_PID environment variable
func getSessionPid() (int64, error) {
	pidStr := os.Getenv("SHY_SESSION_PID")
	if pidStr == "" {
		return 0, fmt.Errorf("shy fc -I: SHY_SESSION_PID environment variable not set")
	}

	pid, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("shy fc -I: invalid SHY_SESSION_PID value: %w", err)
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
	fcWriteFile, _ := cmd.Flags().GetString("write")
	fcWriteSpecified, _ := cmd.Flags().GetBool("write-specified")
	fcAppendFile, _ := cmd.Flags().GetString("append")
	fcReadFile, _ := cmd.Flags().GetString("read")
	fcPushDB, _ := cmd.Flags().GetString("push")
	fcPopDB, _ := cmd.Flags().GetBool("pop")

	// Handle -W without file path (no-op)
	if fcWriteSpecified && fcWriteFile == "" {
		return nil
	}

	// Handle push/pop operations (don't need database open)
	if fcPushDB != "" {
		return runPushMode(fcPushDB)
	}
	if fcPopDB {
		return runPopMode()
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

	// Dispatch to appropriate mode handler based on flags
	switch {
	case fcReadFile != "":
		// -R: Import history from file
		return runReadMode(fcReadFile, database)

	case fcWriteFile != "" || fcAppendFile != "":
		// -W/-A: Export history to file
		return runWriteMode(cmd, args, database, fcWriteFile, fcAppendFile)

	case fcList:
		// -l: List history
		return runListMode(cmd, args, database)

	default:
		// Default: Edit and execute
		return runEditMode(cmd, args, database)
	}
}

// runReadMode handles -R flag: import history from a file
func runReadMode(filePath string, database *db.DB) error {
	return readHistoryFromFile(filePath, database)
}

// runWriteMode handles -W/-A flags: export history to a file
func runWriteMode(cmd *cobra.Command, args []string, database *db.DB, writeFile, appendFile string) error {
	// Get flags needed for write mode
	fcPattern, _ := cmd.Flags().GetString("match")
	fcInternal, _ := cmd.Flags().GetBool("internal")
	fcReverse, _ := cmd.Flags().GetBool("reverse")

	// Parse substitutions
	substitutions, remainingArgs, err := parseSubstitutions(args)
	if err != nil {
		return err
	}
	_ = substitutions // Not used in file operations

	// Parse range - file operations default to ALL commands if no range specified
	first, last, err := parseHistoryRangeForFileOp(remainingArgs, database)
	if err != nil {
		return err
	}

	// Get commands from database with filters
	commands, err := getCommandsWithFilters(database, first, last, fcPattern, fcInternal, true)
	if err != nil {
		return err
	}

	// Apply reverse if requested
	if fcReverse {
		reverseCommands(commands)
	}

	// Write or append to file
	filePath := writeFile
	isAppend := false
	if appendFile != "" {
		filePath = appendFile
		isAppend = true
	}

	if isAppend {
		return appendHistoryToFile(filePath, commands)
	}
	return writeHistoryToFile(filePath, commands)
}

// runListMode handles -l flag: list history commands
func runListMode(cmd *cobra.Command, args []string, database *db.DB) error {
	// Get flags needed for list mode
	fcNoNum, _ := cmd.Flags().GetBool("no-numbers")
	fcReverse, _ := cmd.Flags().GetBool("reverse")
	fcShowTime, _ := cmd.Flags().GetBool("time")
	fcTimeISO, _ := cmd.Flags().GetBool("iso")
	fcTimeUS, _ := cmd.Flags().GetBool("american")
	fcTimeEU, _ := cmd.Flags().GetBool("european")
	fcTimeCustom, _ := cmd.Flags().GetString("time-format")
	fcElapsedTime, _ := cmd.Flags().GetBool("elapsed")
	fcPattern, _ := cmd.Flags().GetString("match")
	fcInternal, _ := cmd.Flags().GetBool("internal")

	// Parse substitutions
	substitutions, remainingArgs, err := parseSubstitutions(args)
	if err != nil {
		return err
	}

	// Parse range - list mode defaults to last 16 commands
	first, last, err := parseHistoryRangeForList(remainingArgs, database)
	if err != nil {
		return err
	}

	// Get commands from database with filters
	commands, err := getCommandsWithFilters(database, first, last, fcPattern, fcInternal, false)
	if err != nil {
		return err
	}

	// Apply reverse if requested
	if fcReverse {
		reverseCommands(commands)
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

// runEditMode handles default mode: edit and execute commands
func runEditMode(cmd *cobra.Command, args []string, database *db.DB) error {
	// Get flags needed for edit mode
	fcPattern, _ := cmd.Flags().GetString("match")
	fcInternal, _ := cmd.Flags().GetBool("internal")
	fcEditor, _ := cmd.Flags().GetString("editor")
	fcQuickExec, _ := cmd.Flags().GetBool("quick-exec")

	// Parse substitutions
	substitutions, remainingArgs, err := parseSubstitutions(args)
	if err != nil {
		return err
	}

	// Parse range - edit mode defaults to last 1 command
	first, last, err := parseHistoryRangeForEdit(remainingArgs, database)
	if err != nil {
		return err
	}

	// Delegate to existing edit-and-execute handler
	return editAndExecuteMode(cmd, database, first, last, substitutions, fcPattern, fcInternal, fcEditor, fcQuickExec)
}

// parseHistoryRangeForFileOp parses range for file operations (defaults to ALL commands)
func parseHistoryRangeForFileOp(args []string, database *db.DB) (int64, int64, error) {
	// If no range specified, export ALL commands
	if len(args) == 0 {
		mostRecent, err := database.GetMostRecentEventID()
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get most recent event: %w", err)
		}
		if mostRecent == 0 {
			return 0, -1, nil // Empty database
		}
		return 1, mostRecent, nil
	}

	// Otherwise use normal list mode parsing
	return parseHistoryRange(args, database, true)
}

// parseHistoryRangeForList parses range for list mode (defaults to last 16)
func parseHistoryRangeForList(args []string, database *db.DB) (int64, int64, error) {
	return parseHistoryRange(args, database, true)
}

// parseHistoryRangeForEdit parses range for edit mode (defaults to last 1)
func parseHistoryRangeForEdit(args []string, database *db.DB) (int64, int64, error) {
	return parseHistoryRange(args, database, false)
}

// getCommandsWithFilters retrieves commands with optional pattern and session filtering
func getCommandsWithFilters(database *db.DB, first, last int64, pattern string, internal bool, allowEmpty bool) ([]models.Command, error) {
	var commands []models.Command
	var err error
	hasFilters := pattern != "" || internal

	if internal {
		// Get current session PID
		sessionPid, err := getSessionPid()
		if err != nil {
			return nil, err
		}

		if pattern != "" {
			// Both internal and pattern filtering
			likePattern := globToLike(pattern)
			commands, err = database.GetCommandsByRangeWithPatternInternal(first, last, sessionPid, likePattern)
		} else {
			// Internal filtering only
			commands, err = database.GetCommandsByRangeInternal(first, last, sessionPid)
		}
	} else if pattern != "" {
		// Pattern filtering only
		likePattern := globToLike(pattern)
		commands, err = database.GetCommandsByRangeWithPattern(first, last, likePattern)
	} else {
		// No filtering
		commands, err = database.GetCommandsByRange(first, last)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get commands: %w", err)
	}

	// Only error on empty results if we have filters and allowEmpty is false
	// Empty database with no filters is not an error
	if len(commands) == 0 && hasFilters && !allowEmpty {
		return nil, fmt.Errorf("shy fc: no matching events found")
	}

	return commands, nil
}

// reverseCommands reverses a slice of commands in place
func reverseCommands(commands []models.Command) {
	for i, j := 0, len(commands)-1; i < j; i, j = i+1, j-1 {
		commands[i], commands[j] = commands[j], commands[i]
	}
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
func parseHistoryRange(args []string, database *db.DB, listMode bool) (int64, int64, error) {
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
				return 0, 0, fmt.Errorf("shy fc: event not found: %s", arg)
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
				return 0, 0, fmt.Errorf("shy fc: event not found: %s", arg1)
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
				return 0, 0, fmt.Errorf("shy fc: event not found: %s", arg2)
			}
			last = matchID
		}

	default:
		return 0, 0, fmt.Errorf("shy fc: too many arguments")
	}

	return first, last, nil
}

// extendedHistoryRegex matches zsh extended history format: ": timestamp:duration;command"
var extendedHistoryRegex = regexp.MustCompile(`^:\s*(\d+):(\d+);(.*)$`)

// isExtendedFormat checks if a line is in zsh extended history format
func isExtendedFormat(line string) bool {
	return extendedHistoryRegex.MatchString(line)
}

// parseExtendedLine parses a line in zsh extended history format
// Returns: timestamp, duration (in ms), command, error
func parseExtendedLine(line string) (int64, int64, string, error) {
	matches := extendedHistoryRegex.FindStringSubmatch(line)
	if len(matches) != 4 {
		return 0, 0, "", fmt.Errorf("invalid extended history format")
	}

	timestamp, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, 0, "", fmt.Errorf("invalid timestamp: %w", err)
	}

	// Duration in extended format is in seconds, but we store in milliseconds
	durationSec, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return 0, 0, "", fmt.Errorf("invalid duration: %w", err)
	}
	durationMs := durationSec * 1000

	command := matches[3]
	return timestamp, durationMs, command, nil
}

// formatExtendedLine formats a command in zsh extended history format
// Duration should be in milliseconds, will be converted to seconds for output
func formatExtendedLine(timestamp int64, durationMs int64, command string) string {
	durationSec := durationMs / 1000
	return fmt.Sprintf(": %d:%d;%s\n", timestamp, durationSec, command)
}

// writeHistoryToFile writes commands to a file in zsh extended history format
func writeHistoryToFile(filePath string, commands []models.Command) error {
	// Create parent directories if they don't exist
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Open file for writing (overwrites if exists)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("fc: cannot write %s: %w", filePath, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, cmd := range commands {
		duration := int64(0)
		if cmd.Duration != nil {
			duration = *cmd.Duration
		}
		line := formatExtendedLine(cmd.Timestamp, duration, cmd.CommandText)

		if _, err := writer.WriteString(line); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	}

	return nil
}

// appendHistoryToFile appends commands to a file in zsh extended history format (creates if doesn't exist)
func appendHistoryToFile(filePath string, commands []models.Command) error {
	// Create parent directories if they don't exist
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Open file for appending (creates if doesn't exist)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file for append: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, cmd := range commands {
		duration := int64(0)
		if cmd.Duration != nil {
			duration = *cmd.Duration
		}
		line := formatExtendedLine(cmd.Timestamp, duration, cmd.CommandText)

		if _, err := writer.WriteString(line); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	}

	return nil
}

// readHistoryFromFile reads commands from a file and imports them into the database
// Automatically detects simple vs extended format
func readHistoryFromFile(filePath string, database *db.DB) error {
	// Check if file exists and is readable
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("fc: cannot read %s: no such file or directory", filePath)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("fc: cannot read %s: permission denied", filePath)
		}
		return fmt.Errorf("fc: cannot read %s: %w", filePath, err)
	}
	defer file.Close()

	// Get current working directory for imported commands
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "" // Fall back to empty string if we can't get cwd
	}

	scanner := bufio.NewScanner(file)
	currentTime := time.Now().Unix()

	for scanner.Scan() {
		line := scanner.Text()

		// Skip blank lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Skip comments (lines starting with #)
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Try to parse as extended format first
		if isExtendedFormat(line) {
			timestamp, durationMs, command, err := parseExtendedLine(line)
			if err != nil {
				// If parsing fails, treat as simple format
				cmd := &models.Command{
					CommandText: line,
					WorkingDir:  cwd,
					ExitStatus:  0,
					Timestamp:   currentTime,
				}
				_, err = database.InsertCommand(cmd)
				if err != nil {
					return fmt.Errorf("failed to insert command: %w", err)
				}
				continue
			}

			// Insert with parsed timestamp and duration
			cmd := &models.Command{
				CommandText: command,
				WorkingDir:  cwd,
				ExitStatus:  0,
				Timestamp:   timestamp,
				Duration:    &durationMs,
			}
			_, err = database.InsertCommand(cmd)
			if err != nil {
				return fmt.Errorf("failed to insert command: %w", err)
			}
		} else {
			// Simple format: just the command
			cmd := &models.Command{
				CommandText: line,
				WorkingDir:  cwd,
				ExitStatus:  0,
				Timestamp:   currentTime,
			}
			_, err = database.InsertCommand(cmd)
			if err != nil {
				return fmt.Errorf("failed to insert command: %w", err)
			}
		}

		// Increment timestamp for next command to maintain order
		currentTime++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}

// runPushMode handles -p flag: push current database to stack and switch to new one
func runPushMode(newPath string) error {
	ppid := os.Getppid()
	if err := session.PushDatabase(ppid, newPath); err != nil {
		return fmt.Errorf("failed to push database: %w", err)
	}
	return nil
}

// runPopMode handles -P flag: pop back to previous database
func runPopMode() error {
	ppid := os.Getppid()
	_, err := session.PopDatabase(ppid)
	if err != nil {
		return fmt.Errorf("failed to pop database: %w", err)
	}
	return nil
}
