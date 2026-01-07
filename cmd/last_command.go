package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var (
	lastCommandOffset int
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

	// Get N most recent commands (where N is 1-based: 1=most recent, 2=second most recent, etc.)
	// ListCommands returns them ordered oldest-to-newest, so the first one
	// is the Nth most recent command
	limit := lastCommandOffset
	commands, err := database.ListCommands(limit)
	if err != nil {
		return fmt.Errorf("failed to list commands: %w", err)
	}

	// If we don't have enough commands, output nothing
	if len(commands) < limit {
		return nil
	}

	// Output the oldest of the retrieved commands (which is the Nth most recent)
	// commands[0] is the oldest of the set we retrieved
	fmt.Fprintln(cmd.OutOrStdout(), commands[0].CommandText)

	return nil
}
