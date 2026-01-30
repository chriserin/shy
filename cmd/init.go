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

//go:embed integration_scripts/shy_autosuggest.zsh
var zshAutosuggestScript string

var (
	initRecord      bool
	initUse         bool
	initAutosuggest bool
)

var initCmd = &cobra.Command{
	Use:   "init [shell]",
	Short: "Generate shell integration script",
	Long:  "Generate shell integration script for command recording, history usage, and autosuggestions. By default, all three are included.",
	Args:  cobra.ExactArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initRecord, "record", false, "Enable command recording (preexec/precmd hooks)")
	initCmd.Flags().BoolVar(&initUse, "use", false, "Enable history usage (Ctrl-R, arrow keys)")
	initCmd.Flags().BoolVar(&initAutosuggest, "autosuggest", false, "Enable zsh-autosuggestions integration (strategy functions)")
}

func runInit(cmd *cobra.Command, args []string) error {
	shell := args[0]

	// Get flag values from the command
	record, _ := cmd.Flags().GetBool("record")
	use, _ := cmd.Flags().GetBool("use")
	autosuggest, _ := cmd.Flags().GetBool("autosuggest")

	// Default to all scripts if no flags specified (except fzf, which is opt-in)
	if !record && !use && !autosuggest {
		record = true
		use = true
		autosuggest = true
	}

	switch shell {
	case "zsh":
		var output strings.Builder
		needsNewline := false

		if record {
			output.WriteString(zshRecordScript)
			needsNewline = true
		}

		if use {
			if needsNewline {
				output.WriteString("\n")
			}
			output.WriteString(zshUseScript)
			needsNewline = true
		}

		if autosuggest {
			if needsNewline {
				output.WriteString("\n")
			}
			output.WriteString(zshAutosuggestScript)
			needsNewline = true
		}

		fmt.Fprint(cmd.OutOrStdout(), output.String())
		return nil
	default:
		return fmt.Errorf("unsupported shell: %s (supported: zsh)", shell)
	}
}
