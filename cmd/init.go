package cmd

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed integration_scripts/zsh_record.sh
var zshRecordScript string

//go:embed integration_scripts/zsh_use.sh
var zshUseScript string

var (
	initRecord bool
	initUse    bool
)

var initCmd = &cobra.Command{
	Use:   "init [shell]",
	Short: "Generate shell integration script",
	Long:  "Generate shell integration script for automatic command tracking and/or history usage",
	Args:  cobra.ExactArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initRecord, "record", false, "Enable command recording (preexec/precmd hooks)")
	initCmd.Flags().BoolVar(&initUse, "use", false, "Enable history usage (Ctrl-R, arrow keys)")
}

func runInit(cmd *cobra.Command, args []string) error {
	shell := args[0]

	// Get flag values from the command
	record, _ := cmd.Flags().GetBool("record")
	use, _ := cmd.Flags().GetBool("use")

	// Default to --record if no flags specified
	if !record && !use {
		record = true
	}

	switch shell {
	case "zsh":
		var output strings.Builder

		if record {
			output.WriteString(zshRecordScript)
		}

		if use {
			if record {
				output.WriteString("\n")
			}
			output.WriteString(zshUseScript)
		}

		fmt.Fprint(cmd.OutOrStdout(), output.String())
		return nil
	default:
		return fmt.Errorf("unsupported shell: %s (supported: zsh)", shell)
	}
}
