package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var (
	listAllFormat string
)

var listAllCmd = &cobra.Command{
	Use:   "list-all",
	Short: "List all commands",
	Long:  "Display all commands from history, ordered oldest to newest",
	RunE:  runListAll,
}

func init() {
	rootCmd.AddCommand(listAllCmd)
	listAllCmd.Flags().StringVar(&listAllFormat, "fmt", "", "Format output with comma-separated columns (timestamp,status,pwd,cmd)")
}

func runListAll(cmd *cobra.Command, args []string) error {
	// Open database
	database, err := db.New(dbPath)
	if err != nil {
		// Check if it's a "file doesn't exist" error
		if os.IsNotExist(err) || (err.Error() != "" && os.IsNotExist(fmt.Errorf("%w", err))) {
			return fmt.Errorf("database doesn't exist (run a command first or use 'shy insert' to add history)")
		}
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// List all commands (no limit)
	commands, err := database.ListCommands(0)
	if err != nil {
		return fmt.Errorf("failed to list commands: %w", err)
	}

	// Handle empty result
	if len(commands) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No commands found")
		return nil
	}

	// Display commands
	if listAllFormat != "" {
		// Format output based on specified columns
		columns := strings.Split(listAllFormat, ",")
		for _, c := range commands {
			var parts []string
			for _, col := range columns {
				col = strings.TrimSpace(col)
				switch col {
				case "timestamp":
					parts = append(parts, time.Unix(c.Timestamp, 0).Format("2006-01-02 15:04:05"))
				case "status":
					parts = append(parts, fmt.Sprintf("%d", c.ExitStatus))
				case "pwd":
					parts = append(parts, c.WorkingDir)
				case "cmd":
					parts = append(parts, c.CommandText)
				case "gb": // git branch
					if c.GitBranch != nil && *c.GitBranch != "" {
						parts = append(parts, *c.GitBranch)
					} else {
						parts = append(parts, "")
					}
				case "gr": // git repo
					if c.GitRepo != nil && *c.GitRepo != "" {
						parts = append(parts, *c.GitRepo)
					} else {
						parts = append(parts, "")
					}
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", strings.Join(parts, "\t"))
		}
	} else {
		// Default: one per line, just the command text
		for _, c := range commands {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", c.CommandText)
		}
	}

	return nil
}
