package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var (
	lastCommandOffset         int
	lastCommandSession        string
	lastCommandCurrentSession bool
)

var lastCommandCmd = &cobra.Command{
	Use:   "last-command",
	Short: "Get the most recent command",
	Long:  "Display the most recent command from history (useful for shell integration)",
	RunE:  runLastCommand,
}

func init() {
	rootCmd.AddCommand(lastCommandCmd)
	lastCommandCmd.Flags().IntVarP(&lastCommandOffset, "offset", "n", 1, "Which command to return (1=most recent, 2=second most recent, etc.)")
	lastCommandCmd.Flags().StringVar(&lastCommandSession, "session", "", "Filter by session (format: app:pid, e.g., zsh:12345)")
	lastCommandCmd.Flags().BoolVar(&lastCommandCurrentSession, "current-session", false, "Filter by current session (auto-detect from environment)")
}

func runLastCommand(cmd *cobra.Command, args []string) error {
	// Validate that n is at least 1
	if lastCommandOffset < 1 {
		return nil
	}

	// Open database
	database, err := db.New(dbPath)
	if err != nil {
		// If database doesn't exist, return empty (no error)
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Parse session filter if provided
	var sourceApp string
	var sourcePid int64
	if lastCommandCurrentSession {
		// Auto-detect current session from environment
		var detected bool
		sourceApp, sourcePid, detected, err = detectCurrentSession()
		if err != nil {
			return fmt.Errorf("failed to detect current session: %w", err)
		}
		if !detected {
			return fmt.Errorf("could not auto-detect session: SHY_SESSION_PID not set")
		}
	} else if lastCommandSession != "" {
		// Parse provided session string
		parts := strings.Split(lastCommandSession, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid session format: expected 'app:pid' (e.g., zsh:12345)")
		}
		sourceApp = parts[0]
		_, err := fmt.Sscanf(parts[1], "%d", &sourcePid)
		if err != nil {
			return fmt.Errorf("invalid session PID: %s", parts[1])
		}
		if sourcePid <= 0 {
			return fmt.Errorf("invalid session PID: must be positive")
		}
	}

	// Get current working directory for union with session results
	workingDir, err := os.Getwd()
	if err != nil {
		// If we can't get working directory, just use empty string (no directory filter)
		workingDir = ""
	}

	// Get the Nth most recent command without consecutive duplicates
	// offset is 0-indexed: 0=most recent, 1=second most recent, etc.
	// If session is specified, results union with current directory after session is exhausted
	offset := lastCommandOffset - 1
	command, err := database.GetRecentCommandsWithoutConsecutiveDuplicates(offset, sourceApp, sourcePid, workingDir)
	if err != nil {
		return fmt.Errorf("failed to get command: %w", err)
	}

	// If no command found at this offset, output nothing
	if command == nil {
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), command.CommandText)

	return nil
}
