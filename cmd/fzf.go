package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var fzfCmd = &cobra.Command{
	Use:   "fzf",
	Short: "Output history in fzf-compatible format",
	Long: `Output command history in tab-separated, null-terminated format for fzf integration.

Format: event_number<TAB>command<NULL>
Commands are deduplicated (only most recent occurrence shown) and output in reverse chronological order.
All filtering is done interactively within fzf.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		// Open database
		database, err := db.New(dbPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer database.Close()

		// Get all commands with SQL-based deduplication
		entries, err := database.GetCommandsForFzf()
		if err != nil {
			return err
		}

		// Output: event_number<TAB>command<NULL>
		for _, entry := range entries {
			fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\000", entry.ID, entry.CommandText)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(fzfCmd)
}
