package cmd

import (
	_ "embed"
	"fmt"

	"github.com/spf13/cobra"
)

//go:embed integration_scripts/claude_pre_hook.sh
var claudePreHookScript string

//go:embed integration_scripts/claude_post_hook.sh
var claudePostHookScript string

//go:embed integration_scripts/claude_settings.sh
var claudeSettingsScript string

var claudeInitCmd = &cobra.Command{
	Use:   "claude-init",
	Short: "Generate Claude Code integration scripts for command capture",
	Long: `Generate Claude Code hook scripts to automatically capture bash commands
executed by Claude Code into your shy history database.

This command outputs the necessary hook scripts and configuration
instructions for setting up Claude Code integration.`,
	RunE: runClaudeInit,
}

func init() {
	rootCmd.AddCommand(claudeInitCmd)
}

func runClaudeInit(cmd *cobra.Command, args []string) error {
	var output string

	output += "# Claude Code Integration for shy\n"
	output += "# Follow these steps to set up command capture:\n\n"
	output += "# 1. Create the hooks directory\n"
	output += "mkdir -p ~/.claude/hooks\n\n"
	output += "# 2. Create the pre-hook script\n"
	output += "cat > ~/.claude/hooks/shy-start-timer.sh << 'EOF'\n"
	output += claudePreHookScript
	output += "EOF\n\n"
	output += "# 3. Create the post-hook script\n"
	output += "cat > ~/.claude/hooks/capture-to-shy.sh << 'EOF'\n"
	output += claudePostHookScript
	output += "EOF\n\n"
	output += "# 4. Make the scripts executable\n"
	output += "chmod +x ~/.claude/hooks/shy-start-timer.sh\n"
	output += "chmod +x ~/.claude/hooks/capture-to-shy.sh\n\n"
	output += "# 5. Merge hooks into Claude Code settings\n"
	output += claudeSettingsScript
	output += "\n"
	output += "# 6. Reload Claude Code or restart the session\n\n"
	output += "# Test the integration:\n"
	output += "# Run a bash command in Claude Code, then check:\n"
	output += "# shy fc -l -10\n"

	fmt.Fprint(cmd.OutOrStdout(), output)
	return nil
}
