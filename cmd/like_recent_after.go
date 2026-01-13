package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var (
	likeRecentAfterPrev       string
	likeRecentAfterLimit      int
	likeRecentAfterExclude    string
	likeRecentAfterIncludeShy bool
)

var likeRecentAfterCmd = &cobra.Command{
	Use:   "like-recent-after <prefix>",
	Short: "Get command matching prefix that came after a specific previous command",
	Long: `Find commands matching a prefix that historically came after a specific previous command.
This provides context-aware suggestions based on command sequences.

Example:
  shy like-recent-after "git pu" --prev "git commit -m 'fix'"
  # Suggests "git push" if it typically follows git commit`,
	Args: cobra.ExactArgs(1),
	RunE: runLikeRecentAfter,
}

func init() {
	rootCmd.AddCommand(likeRecentAfterCmd)

	// Add flags
	likeRecentAfterCmd.Flags().StringVar(&likeRecentAfterPrev, "prev", "", "Previous command to match context (required)")
	likeRecentAfterCmd.Flags().IntVar(&likeRecentAfterLimit, "limit", 1, "Number of suggestions")
	likeRecentAfterCmd.Flags().StringVar(&likeRecentAfterExclude, "exclude", "", "Exclude commands matching pattern (glob)")
	likeRecentAfterCmd.Flags().BoolVar(&likeRecentAfterIncludeShy, "include-shy", false, "Include shy commands in results")

	// Note: We handle the required check manually in runLikeRecentAfter using cmd.Flags().Changed("prev")
	// to avoid test state pollution issues with MarkFlagRequired
}

func runLikeRecentAfter(cmd *cobra.Command, args []string) error {
	prefix := args[0]

	// Check if prev flag was actually provided (required flag check)
	if !cmd.Flags().Changed("prev") {
		return fmt.Errorf("required flag \"prev\" not set")
	}

	// If prev is empty (but was provided), return empty results (no error)
	if likeRecentAfterPrev == "" {
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

	// Build filter options
	opts := db.LikeRecentAfterOptions{
		Prefix:     prefix,
		PrevCmd:    likeRecentAfterPrev,
		Limit:      likeRecentAfterLimit,
		IncludeShy: likeRecentAfterIncludeShy,
		Exclude:    likeRecentAfterExclude,
	}

	// Query database
	commands, err := database.LikeRecentAfter(opts)
	if err != nil {
		return fmt.Errorf("failed to query commands: %w", err)
	}

	// Output commands (one per line)
	for _, cmdText := range commands {
		fmt.Fprintln(cmd.OutOrStdout(), cmdText)
	}

	return nil
}
