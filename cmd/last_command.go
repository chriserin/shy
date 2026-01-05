package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var lastCommandCmd = &cobra.Command{
	Use:   "last-command",
	Short: "Get the most recent command",
	Long:  "Display the most recent command from history (useful for shell integration)",
	RunE:  runLastCommand,
}

func init() {
	rootCmd.AddCommand(lastCommandCmd)
}

func runLastCommand(cmd *cobra.Command, args []string) error {
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

	// Get the most recent command (limit 1, will be ordered oldest to newest, so we get the last one)
	commands, err := database.ListCommands(1)
	if err != nil {
		return fmt.Errorf("failed to list commands: %w", err)
	}

	// If no commands, output nothing
	if len(commands) == 0 {
		return nil
	}

	// Output only the command text (no formatting)
	fmt.Fprintln(cmd.OutOrStdout(), commands[0].CommandText)

	return nil
}
