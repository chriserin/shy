package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [event-ids...]",
	Short: "Delete commands from history by event ID",
	Long:  "Delete one or more commands from the history database by their event IDs",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	// Parse positional args as int64 event IDs
	ids := make([]int64, len(args))
	for i, arg := range args {
		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid event ID %q: %w", arg, err)
		}
		if id <= 0 {
			return fmt.Errorf("invalid event ID %q: must be a positive integer", arg)
		}
		ids[i] = id
	}

	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	count, err := database.DeleteCommands(ids)
	if err != nil {
		return fmt.Errorf("failed to delete commands: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Deleted %d command(s)\n", count)
	return nil
}
