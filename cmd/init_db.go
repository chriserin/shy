package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var initDBCmd = &cobra.Command{
	Use:   "init-db",
	Short: "Initialize the shy database",
	Long:  "Create the shy SQLite database and tables if they don't exist",
	RunE:  runInitDB,
}

func init() {
	rootCmd.AddCommand(initDBCmd)
}

func runInitDB(cmd *cobra.Command, args []string) error {
	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	fmt.Printf("Database initialized at: %s\n", database.Path())
	return nil
}
