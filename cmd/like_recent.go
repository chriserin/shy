package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var (
	likeRecentPwd     bool
	likeRecentSession bool
	likeRecentExclude string
	likeRecentLimit   int
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

	// Add flags
	likeRecentCmd.Flags().BoolVar(&likeRecentPwd, "pwd", false, "Only match commands from current directory")
	likeRecentCmd.Flags().BoolVar(&likeRecentSession, "session", false, "Only match from current session (SHY_SESSION_PID)")
	likeRecentCmd.Flags().StringVar(&likeRecentExclude, "exclude", "", "Exclude commands matching pattern (glob)")
	likeRecentCmd.Flags().IntVar(&likeRecentLimit, "limit", 1, "Number of suggestions")
}

func runLikeRecent(cmd *cobra.Command, args []string) error {
	prefix := args[0]

	// If limit is 0, return empty results immediately
	if likeRecentLimit == 0 {
		return nil
	}

	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Build filter options
	opts := db.LikeRecentOptions{
		Prefix:  prefix,
		Exclude: likeRecentExclude,
	}

	// Add pwd filter if requested
	if likeRecentPwd {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		opts.WorkingDir = cwd
	}

	// Add session filter if requested
	if likeRecentSession {
		sourceApp, sourcePid, detected, err := detectCurrentSession()
		if err != nil {
			return fmt.Errorf("failed to detect current session: %w", err)
		}
		if detected {
			opts.SourceApp = sourceApp
			opts.SourcePid = sourcePid
		}
	}

	// Query database
	commands, err := database.LikeRecent(opts)
	if err != nil {
		return fmt.Errorf("failed to query commands: %w", err)
	}

	// Output commands (one per line)
	for _, cmdText := range commands {
		fmt.Fprintln(cmd.OutOrStdout(), cmdText)
	}

	return nil
}
