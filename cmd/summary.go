package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/summary/tui"
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Interactive summary of shell command activity",
	Long:  "Display an interactive summary of shell commands grouped by repository/directory and branch",
	RunE:  runSummary,
}

func init() {
	rootCmd.AddCommand(summaryCmd)
}

func runSummary(cmd *cobra.Command, args []string) error {
	model := tui.New(dbPath)
	defer model.Close()

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithReportFocus())
	if _, err := p.Run(); err != nil {
		if err.Error() == "quit" {
			return nil
		}
		return fmt.Errorf("failed to run summary: %w", err)
	}

	return nil
}

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}

// int64Ptr returns a pointer to the given int64
func int64Ptr(i int64) *int64 {
	return &i
}
