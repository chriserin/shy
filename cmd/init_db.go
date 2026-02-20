package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var initDBCmd = &cobra.Command{
	Use:   "init-db",
	Short: "Initialize the database schema",
	Long:  "Creates the shy database and initializes the schema. Safe to run multiple times - will not overwrite existing data.",
	RunE:  runInitDB,
}

func init() {
	rootCmd.AddCommand(initDBCmd)
}

func runInitDB(cmd *cobra.Command, args []string) error {
	dbPath, _ := cmd.Flags().GetString("db")

	// Open with SkipSchemaCheck, then call InitSchema to detect new vs existing
	database, err := db.NewWithOptions(dbPath, db.Options{SkipSchemaCheck: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	created, err := database.InitSchema()
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	if created {
		fmt.Fprintf(cmd.OutOrStdout(), "Database initialized: %s\n", database.Path())
	}
	// Silent if already initialized (for idempotent shell init)

	return nil
}
