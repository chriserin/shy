package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
)

var starCmd = &cobra.Command{
	Use:   "star",
	Short: "Manage starred commands",
	Long: `Star important commands for quick retrieval.

When run without a subcommand, stars the most recent command from the current session.

Use subcommands to add, remove, or list starred commands.`,
	RunE: runStarBare,
}

var starAddCmd = &cobra.Command{
	Use:   "add <event-id>",
	Short: "Star a command by event ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid event ID %q: %w", args[0], err)
		}
		if id <= 0 {
			return fmt.Errorf("invalid event ID %q: must be a positive integer", args[0])
		}

		database, err := db.New(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer database.Close()

		if err := database.StarCommand(id); err != nil {
			return fmt.Errorf("failed to star command: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Starred command %d\n", id)
		return nil
	},
}

var starRemoveCmd = &cobra.Command{
	Use:   "remove <event-id>",
	Short: "Unstar a command by event ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid event ID %q: %w", args[0], err)
		}
		if id <= 0 {
			return fmt.Errorf("invalid event ID %q: must be a positive integer", args[0])
		}

		database, err := db.New(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer database.Close()

		if err := database.UnstarCommand(id); err != nil {
			return fmt.Errorf("failed to unstar command: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Unstarred command %d\n", id)
		return nil
	},
}

var starListCmd = &cobra.Command{
	Use:   "list",
	Short: "List starred commands",
	RunE:  runStarList,
}

func init() {
	rootCmd.AddCommand(starCmd)
	starCmd.AddCommand(starAddCmd)
	starCmd.AddCommand(starRemoveCmd)
	starCmd.AddCommand(starListCmd)

	starListCmd.Flags().Bool("pwd", false, "Filter to current working directory")
	starListCmd.Flags().Bool("current-session", false, "Filter to current shell session")
}

func runStarBare(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	sourceApp, sourcePid, detected, err := detectCurrentSession()
	if err != nil {
		return fmt.Errorf("failed to detect session: %w", err)
	}
	if !detected {
		return fmt.Errorf("could not detect current session: SHY_SESSION_PID not set")
	}

	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Get the most recent command from this session
	commands, err := database.ListCommands(1, sourceApp, sourcePid, "")
	if err != nil {
		return fmt.Errorf("failed to list commands: %w", err)
	}
	if len(commands) == 0 {
		return fmt.Errorf("no commands found in current session")
	}

	// ListCommands returns oldest-first when limited, so the last item is most recent
	recent := commands[len(commands)-1]

	if err := database.StarCommand(recent.ID); err != nil {
		return fmt.Errorf("failed to star command: %w", err)
	}

	// Truncate command text for display
	text := recent.CommandText
	if len(text) > 60 {
		text = text[:57] + "..."
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Starred #%d: %s\n", recent.ID, text)
	return nil
}

func runStarList(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	var sourceApp string
	var sourcePid int64
	var cwd string

	currentSession, _ := cmd.Flags().GetBool("current-session")
	filterPwd, _ := cmd.Flags().GetBool("pwd")

	if currentSession {
		app, pid, err := parseSessionFilter(true, "")
		if err != nil {
			return err
		}
		sourceApp = app
		sourcePid = pid
	}

	if filterPwd {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		cwd = wd
	}

	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	commands, err := database.ListStarredCommands(sourceApp, sourcePid, cwd)
	if err != nil {
		return fmt.Errorf("failed to list starred commands: %w", err)
	}

	for _, c := range commands {
		fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\n", c.ID, c.CommandText)
	}

	return nil
}
