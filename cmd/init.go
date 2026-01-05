package cmd

import (
	_ "embed"
	"fmt"

	"github.com/spf13/cobra"
)

//go:embed integration_scripts/zsh.sh
var zshIntegrationScript string

var initCmd = &cobra.Command{
	Use:   "init [shell]",
	Short: "Generate shell integration script",
	Long:  "Generate shell integration script for automatic command tracking",
	Args:  cobra.ExactArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	shell := args[0]

	switch shell {
	case "zsh":
		fmt.Fprint(cmd.OutOrStdout(), zshIntegrationScript)
		return nil
	default:
		return fmt.Errorf("unsupported shell: %s (supported: zsh)", shell)
	}
}
