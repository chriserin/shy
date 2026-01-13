package cmd

import (
	"os"

	"github.com/chris/shy/internal/session"
	"github.com/spf13/cobra"
)

var dbPath string

var rootCmd = &cobra.Command{
	Use:     "shy",
	Short:   "Shell history tracker",
	Long:    "A command-line tool to track shell command history in SQLite",
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// If --db flag was explicitly set, use that
		if cmd.Flags().Changed("db") {
			return nil
		}

		// Otherwise, check session file for current database
		ppid := os.Getppid()
		sessionDB, err := session.GetCurrentDatabase(ppid)
		if err != nil {
			return err
		}

		// If session has a database, use it
		if sessionDB != "" {
			dbPath = sessionDB
		}
		// If sessionDB is empty, dbPath remains empty and db.New will use default

		return nil
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Database file path (default: ~/.local/share/shy/history.db)")
	// Version flag is automatically added by cobra when Version is set
	rootCmd.SetVersionTemplate("shy version {{.Version}}\n")
}
