package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var likeRecentCmd = &cobra.Command{
	Use:   "like-recent <prefix>",
	Short: "Get most recent command starting with prefix",
	Long:  "Find the most recent command that starts with the given prefix (useful for shell completion)",
	Args:  cobra.ExactArgs(1),
	RunE:  runLikeRecent,
}

func init() {
	rootCmd.AddCommand(likeRecentCmd)
}

func runLikeRecent(cmd *cobra.Command, args []string) error {
	prefix := args[0]

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

	// Get all commands (we need to search through them)
	// Use a reasonable limit to avoid loading too much data
	commands, err := database.ListCommands(10000)
	if err != nil {
		return fmt.Errorf("failed to list commands: %w", err)
	}

	// Find the most recent command matching the prefix
	// Commands are ordered oldest to newest, so iterate backwards
	for i := len(commands) - 1; i >= 0; i-- {
		cmdText := commands[i].CommandText

		// Filter out shy commands
		if strings.HasPrefix(cmdText, "shy ") || cmdText == "shy" {
			continue
		}

		// Check if it starts with the prefix (case-sensitive)
		if strings.HasPrefix(cmdText, prefix) {
			// Output only the command text (no formatting)
			fmt.Fprintln(cmd.OutOrStdout(), cmdText)
			return nil
		}
	}

	// No match found - output nothing
	return nil
}
