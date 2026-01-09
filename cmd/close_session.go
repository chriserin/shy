package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var (
	sessionPid int64
)

var closeSessionCmd = &cobra.Command{
	Use:   "close-session",
	Short: "Mark a shell session as closed",
	Long:  "Mark all commands from a shell session as inactive (source_active=0)",
	RunE:  runCloseSession,
}

func init() {
	rootCmd.AddCommand(closeSessionCmd)

	closeSessionCmd.Flags().Int64Var(&sessionPid, "pid", 0, "Shell session PID (required)")
	closeSessionCmd.MarkFlagRequired("pid")
}

func runCloseSession(cmd *cobra.Command, args []string) error {
	// Validate required parameters
	if sessionPid <= 0 {
		return fmt.Errorf("--pid is required and must be positive")
	}

	// Open database
	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Close the session
	count, err := database.CloseSession(sessionPid)
	if err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	// Silently succeed (no output) - this is typically called from shell hooks
	_ = count // Avoid unused variable error
	return nil
}
