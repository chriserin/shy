package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
)

func init() {
	// Force color output for TV preview (television supports colors)
	lipgloss.SetColorProfile(termenv.TrueColor)
}

// Styles for TV preview
var (
	labelStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // Cyan
	valueStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("255")) // White
	eventStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Orange
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))  // Green
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))   // Red
	gitStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("141")) // Purple
	contextStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("242")) // Gray
	separatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Dark gray
)

var tvCmd = &cobra.Command{
	Use:   "tv",
	Short: "Television integration commands",
	Long: `Commands for integrating shy with television (https://github.com/alexpasmantier/television).

Use 'shy tv config' to generate a television channel configuration.`,
}

var tvConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Output television channel configuration",
	Long: `Output a TOML configuration for television that allows browsing shy command history.

To use this with television, add it to your cable channels:
  shy tv config > ~/.config/television/cable/shy.toml

Then in television, select the "shy" channel to browse your command history.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		// Output the TOML configuration
		fmt.Fprint(cmd.OutOrStdout(), getTVConfig())
		return nil
	},
}

var tvListCmd = &cobra.Command{
	Use:   "list",
	Short: "Output history in television-compatible format",
	Long: `Output command history in tab-separated format for television integration.

Format: event_number<TAB>command<NEWLINE>
Commands are deduplicated (only most recent occurrence shown) and output in reverse chronological order.`,
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

		// Stream commands with SQL-based deduplication
		// Output: event_number<TAB>command<NEWLINE>
		err = database.GetCommandsForFzf(func(id int64, cmdText string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\n", id, cmdText)
			return err
		})

		return err
	},
}

var tvPreviewCmd = &cobra.Command{
	Use:   "preview <event-number>",
	Short: "Display command details with surrounding context",
	Long: `Display detailed information about a command including all metadata and surrounding commands from the same session.

This command is designed to be used as a preview command in television channel configurations.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		// Parse event number
		eventNum, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid event number: %w", err)
		}

		// Open database
		database, err := db.New(dbPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("database not found")
			}
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer database.Close()

		// Get command with context (5 commands before and after)
		beforeCmds, targetCmd, afterCmds, err := database.GetCommandWithContext(eventNum, 5)
		if err != nil {
			return fmt.Errorf("failed to get command context: %w", err)
		}

		// Display the output
		displayCommandWithContext(cmd.OutOrStdout(), beforeCmds, targetCmd, afterCmds)

		return nil
	},
}

func getTVConfig() string {
	return `[metadata]
name = "shy-history"
description = "Browse shell command history from shy database"
requirements = ["shy"]

[source]
command = "shy tv list"
display = "{split:\t:1}"
output = "{split:\t:0}"

[preview]
command = "shy tv preview '{split:\t:0}'"
`
}

func displayCommandWithContext(w interface{ Write([]byte) (int, error) }, beforeCmds []models.Command, targetCmd *models.Command, afterCmds []models.Command) {
	// Display target command metadata first
	displayDetailedCommand(w, targetCmd)

	// Separator before context commands
	if len(beforeCmds) > 0 || len(afterCmds) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, separatorStyle.Render("  ─────────────────────────────────────────────────────────────────────"))
		fmt.Fprintln(w)
	}

	// Display commands before (if any)
	if len(beforeCmds) > 0 {
		for _, cmd := range beforeCmds {
			displaySimpleCommand(w, &cmd)
		}
	}

	// Display this command in simple format (highlighted)
	fmt.Fprintf(w, "  %s %s\n",
		eventStyle.Render(fmt.Sprintf("%d", targetCmd.ID)),
		valueStyle.Render(targetCmd.CommandText))

	// Display commands after (if any)
	if len(afterCmds) > 0 {
		for _, cmd := range afterCmds {
			displaySimpleCommand(w, &cmd)
		}
	}
}

func displayDetailedCommand(w interface{ Write([]byte) (int, error) }, cmd *models.Command) {
	// Format timestamp
	timestamp := time.Unix(cmd.Timestamp, 0)
	timeStr := timestamp.Format("2006-01-02 15:04:05")

	// Event number
	fmt.Fprintf(w, "  %s %s\n",
		labelStyle.Render("Event:"),
		eventStyle.Render(fmt.Sprintf("%d", cmd.ID)))

	// Command text
	fmt.Fprintf(w, "  %s %s\n",
		labelStyle.Render("Command:"),
		valueStyle.Render(cmd.CommandText))

	// Exit status (colored based on success/failure)
	var statusStr string
	if cmd.ExitStatus == 0 {
		statusStr = successStyle.Render(fmt.Sprintf("%d", cmd.ExitStatus))
	} else {
		statusStr = errorStyle.Render(fmt.Sprintf("%d", cmd.ExitStatus))
	}
	fmt.Fprintf(w, "  %s %s\n",
		labelStyle.Render("Exit Status:"),
		statusStr)

	// Timestamp
	fmt.Fprintf(w, "  %s %s\n",
		labelStyle.Render("Timestamp:"),
		valueStyle.Render(timeStr))

	// Working directory
	fmt.Fprintf(w, "  %s %s\n",
		labelStyle.Render("Working Dir:"),
		valueStyle.Render(cmd.WorkingDir))

	// Duration
	if cmd.Duration != nil {
		fmt.Fprintf(w, "  %s %s\n",
			labelStyle.Render("Duration:"),
			valueStyle.Render(formatDurationHuman(cmd.Duration)))
	} else {
		fmt.Fprintf(w, "  %s %s\n",
			labelStyle.Render("Duration:"),
			contextStyle.Render("(not recorded)"))
	}

	// Git info (if present)
	if cmd.GitRepo != nil {
		fmt.Fprintf(w, "  %s %s\n",
			labelStyle.Render("Git Repo:"),
			gitStyle.Render(*cmd.GitRepo))
	}

	if cmd.GitBranch != nil {
		fmt.Fprintf(w, "  %s %s\n",
			labelStyle.Render("Git Branch:"),
			gitStyle.Render(*cmd.GitBranch))
	}

	// Session info (if present)
	if cmd.SourceApp != nil && cmd.SourcePid != nil {
		sessionStr := fmt.Sprintf("%s:%d", *cmd.SourceApp, *cmd.SourcePid)
		// Add :X if session is not active
		if cmd.SourceActive != nil && !*cmd.SourceActive {
			sessionStr += ":X"
		}
		fmt.Fprintf(w, "  %s %s\n",
			labelStyle.Render("Session:"),
			valueStyle.Render(sessionStr))
	}
}

func displaySimpleCommand(w interface{ Write([]byte) (int, error) }, cmd *models.Command) {
	fmt.Fprintf(w, "  %s %s\n",
		contextStyle.Render(fmt.Sprintf("%d", cmd.ID)),
		contextStyle.Render(cmd.CommandText))
}

func formatDurationHuman(durationMs *int64) string {
	if durationMs == nil {
		return "0s"
	}

	millis := *durationMs
	d := time.Duration(millis) * time.Millisecond

	if d < time.Second {
		return fmt.Sprintf("%dms", millis)
	}

	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func init() {
	rootCmd.AddCommand(tvCmd)
	tvCmd.AddCommand(tvConfigCmd)
	tvCmd.AddCommand(tvListCmd)
	tvCmd.AddCommand(tvPreviewCmd)
}
