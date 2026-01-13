package cmd

import (
	"fmt"
	"strconv"

	"github.com/chris/shy/internal/session"
	"github.com/spf13/cobra"
)

var cleanupSessionCmd = &cobra.Command{
	Use:   "cleanup-session <pid>",
	Short: "Clean up session file for a given PID",
	Long:  "Removes the session file associated with a shell process ID. Called by zshexit hook.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse PID from argument
		pid, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid PID: %s", args[0])
		}

		// Clean up the session
		if err := session.CleanupSession(pid); err != nil {
			return fmt.Errorf("failed to cleanup session: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanupSessionCmd)
}
