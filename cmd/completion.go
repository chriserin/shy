package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

func init() {
	// Register custom completions after all commands are initialized
	cobra.OnInitialize(registerCompletions)
}

func registerCompletions() {
	// --db flag: complete with .db files
	rootCmd.RegisterFlagCompletionFunc("db", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"db"}, cobra.ShellCompDirectiveFilterFileExt
	})

	// init command: complete with supported shells
	initCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{"zsh"}, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// list command completions
	registerListCompletions(listCmd)

	// list-all command completions
	registerListCompletions(listAllCmd)

	// summary command completions
	registerSummaryCompletions(summaryCmd)

	// last-command completions
	registerLastCommandCompletions(lastCommandCmd)

	// like-recent completions
	registerLikeRecentCompletions(likeRecentCmd)
}

// getSourceAppsFromDB queries the database for unique source apps
func getSourceAppsFromDB() []string {
	database, err := db.New(dbPath)
	if err != nil {
		return nil
	}
	defer database.Close()

	apps, err := database.GetUniqueSourceApps()
	if err != nil {
		return nil
	}
	return apps
}

// completeSourceApp returns completions for --source-app and --session flags
func completeSourceApp(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	apps := getSourceAppsFromDB()
	if len(apps) == 0 {
		// Fallback to common shells if database is empty/unavailable
		return []string{
			"zsh\tzsh shell",
			"bash\tbash shell",
		}, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, app := range apps {
		completions = append(completions, fmt.Sprintf("%s\t%s shell", app, app))
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func registerListCompletions(cmd *cobra.Command) {
	// --session flag: complete with source apps from database
	cmd.RegisterFlagCompletionFunc("session", completeSourceApp)

	// --fmt flag: complete with column names
	cmd.RegisterFlagCompletionFunc("fmt", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Parse already-specified columns
		specified := make(map[string]bool)
		if toComplete != "" {
			parts := strings.Split(toComplete, ",")
			for _, p := range parts[:len(parts)-1] {
				specified[strings.TrimSpace(p)] = true
			}
		}

		columns := []string{
			"timestamp\tCommand timestamp",
			"status\tExit status",
			"pwd\tWorking directory",
			"cmd\tCommand text",
			"gb\tGit branch",
			"gr\tGit repository",
			"durs\tDuration in seconds",
			"durms\tDuration in milliseconds",
		}

		// Filter out already specified columns
		var available []string
		for _, col := range columns {
			name := strings.Split(col, "\t")[0]
			if !specified[name] {
				available = append(available, col)
			}
		}

		return available, cobra.ShellCompDirectiveNoFileComp
	})
}

func registerSummaryCompletions(cmd *cobra.Command) {
	// --date flag
	cmd.RegisterFlagCompletionFunc("date", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"yesterday\tYesterday's activity",
			"today\tToday's activity",
		}, cobra.ShellCompDirectiveNoFileComp
	})

	// --bucket flag
	cmd.RegisterFlagCompletionFunc("bucket", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"hour\tGroup by hour",
			"period\tGroup by time period (morning, afternoon, evening)",
			"day\tGroup by day",
			"week\tGroup by week",
		}, cobra.ShellCompDirectiveNoFileComp
	})

	// --source-app flag: complete with source apps from database
	cmd.RegisterFlagCompletionFunc("source-app", completeSourceApp)
}

func registerLastCommandCompletions(cmd *cobra.Command) {
	// --session flag: complete with source apps from database
	cmd.RegisterFlagCompletionFunc("session", completeSourceApp)
}

func registerLikeRecentCompletions(cmd *cobra.Command) {
	// No specific completions needed - prefix is free-form text
}
