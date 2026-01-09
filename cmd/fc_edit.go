package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/pkg/models"
	"github.com/spf13/cobra"
)

// Injectable function variables for testing
var (
	invokeEditorFunc = invokeEditorReal
	executeShellFunc = executeShellReal
)

// editAndExecuteMode orchestrates the edit-and-execute workflow
func editAndExecuteMode(cmd *cobra.Command, database *db.DB, first, last int64,
	substitutions []substitution, fcPattern string, fcInternal bool,
	fcEditor string, fcQuickExec bool) error {

	// 1. Validate range (backwards check)
	if first > last {
		fmt.Fprintf(os.Stderr, "Error: fc: history events can't be executed backwards, aborted\n")
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		osExit(1)
		return nil
	}

	// 2. Get commands from database (respect pattern/internal filters)
	var commands []models.Command
	var err error

	if fcInternal {
		sessionPid, err := getSessionPid()
		if err != nil {
			return err
		}
		if fcPattern != "" {
			likePattern := globToLike(fcPattern)
			commands, err = database.GetCommandsByRangeWithPatternInternal(first, last, sessionPid, likePattern)
		} else {
			commands, err = database.GetCommandsByRangeInternal(first, last, sessionPid)
		}
	} else if fcPattern != "" {
		likePattern := globToLike(fcPattern)
		commands, err = database.GetCommandsByRangeWithPattern(first, last, likePattern)
	} else {
		commands, err = database.GetCommandsByRange(first, last)
	}

	if err != nil {
		return fmt.Errorf("failed to get commands: %w", err)
	}

	// 3. Check if no commands found - use feat file error message
	if len(commands) == 0 {
		fmt.Fprintf(os.Stderr, "Error: fc: current history line would recurse endlessly, aborted\n")
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		osExit(1)
		return nil
	}

	// 4. Check for recursive fc (trying to execute fc itself)
	for _, c := range commands {
		trimmed := strings.TrimSpace(c.CommandText)
		// Check for various invocations: "fc", "shy fc", "./shy fc", "/path/to/shy fc", etc.
		if trimmed == "fc" || strings.HasPrefix(trimmed, "fc ") ||
			strings.HasPrefix(trimmed, "shy fc") || strings.Contains(trimmed, "/shy fc") {
			fmt.Fprintf(os.Stderr, "Error: fc: current history line would recurse endlessly, aborted\n")
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			osExit(1)
			return nil
		}
	}

	// 5. Apply old=new substitutions
	var commandTexts []string
	for _, c := range commands {
		text := c.CommandText
		if len(substitutions) > 0 {
			text = applySubstitutions(text, substitutions)
		}
		commandTexts = append(commandTexts, text)
	}

	// 6. If -s flag: execute directly, skip editor
	if fcQuickExec {
		return executeCommands(database, commandTexts)
	}

	// 7. Create temp file with commands (expand \n to actual newlines)
	tmpfile, err := createTempFileWithCommands(commandTexts)
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())

	// 8. Get editor and invoke
	editor, err := getEditor(fcEditor)
	if err != nil {
		return err
	}

	err = invokeEditor(editor, tmpfile.Name())
	if err != nil {
		// If editor exits with non-zero, exit silently with the same status code
		if exitErr, ok := err.(*exec.ExitError); ok {
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			osExit(exitErr.ExitCode())
			return nil
		}

		fmt.Fprintln(cmd.OutOrStdout(), err)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		osExit(1)
		// For other errors (e.g., editor not found), return the error
		return err
	}

	// 9. Parse edited file
	editedCommands, err := parseEditedFile(tmpfile.Name())
	if err != nil {
		return err
	}

	// 10. If empty, abort silently
	if len(editedCommands) == 0 {
		return nil
	}

	// 11. Execute edited commands
	return executeCommands(database, editedCommands)
}

// getEditor determines which editor to use
func getEditor(fcEditor string) (string, error) {
	// Priority order:
	// 1. -e flag value (if provided)
	// 3. $EDITOR environment variable
	// 4. Default to "vi"

	if fcEditor != "" {
		return fcEditor, nil
	}

	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor, nil
	}

	return "vi", nil
}

// createTempFileWithCommands creates a secure temp file with commands
// Expands \n to actual newlines for prettier editing
func createTempFileWithCommands(commandTexts []string) (*os.File, error) {
	// Create temp file with mode 0600 (secure permissions)
	tmpfile, err := os.CreateTemp("", "fc-*.sh")
	if err != nil {
		return nil, err
	}

	// Set secure permissions (user read/write only)
	if err := tmpfile.Chmod(0600); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return nil, err
	}

	// Write commands
	for _, cmdText := range commandTexts {
		// Expand \n escape sequences to actual newlines
		expandedText := strings.ReplaceAll(cmdText, "\\n", "\n")

		if _, err := tmpfile.WriteString(expandedText); err != nil {
			tmpfile.Close()
			os.Remove(tmpfile.Name())
			return nil, err
		}

		// Add newline after each command
		if _, err := tmpfile.WriteString("\n"); err != nil {
			tmpfile.Close()
			os.Remove(tmpfile.Name())
			return nil, err
		}
	}

	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		return nil, err
	}

	return tmpfile, nil
}

// invokeEditor launches the editor and waits for completion
func invokeEditor(editorPath string, filepath string) error {
	return invokeEditorFunc(editorPath, filepath)
}

// invokeEditorReal is the real implementation of invokeEditor
func invokeEditorReal(editorPath string, filepath string) error {
	// Use exec.Command with stdin/stdout/stderr connected to terminal
	// NOTE: We pass editor path as-is, don't parse shell arguments
	// User can specify `vim` or `/usr/bin/vim` but not `vim -u NONE`
	// (shell argument parsing is complex and error-prone)

	cmd := exec.Command(editorPath, filepath)

	// Connect to terminal for interactive editing
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	// Return error as-is (caller will check for exec.ExitError)
	return err
}

// parseEditedFile reads and parses edited commands from temp file
func parseEditedFile(filepath string) ([]string, error) {
	// Read temp file
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	// Split by newlines and treat each non-empty line as a command
	lines := strings.Split(string(content), "\n")

	var commands []string
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip comment lines
		if strings.HasPrefix(line, "#") {
			continue
		}

		commands = append(commands, line)
	}

	return commands, nil
}

// executeCommands executes shell commands and records to history
func executeCommands(database *db.DB, commands []string) error {
	// Execute each command in user's shell
	// Continue on errors (per user decision)
	// Add all executed commands to history as ONE newline-separated entry

	// Print all commands before executing (like zsh fc does)
	for _, cmdText := range commands {
		fmt.Println(cmdText)
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	for _, cmdText := range commands {
		err := executeShellFunc(shell, cmdText)
		if err != nil {
			// Continue executing other commands even if one fails
			// Do not print error here
		}
	}

	// Add executed commands to history as single newline-separated entry
	// (per user decision: "add to history as one newline separated command")
	combinedCommand := strings.Join(commands, "\n")

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	// Insert to database
	// Note: ExitStatus is 0 (we don't track individual command exits for fc)
	_, err = database.InsertCommand(&models.Command{
		Timestamp:   time.Now().Unix(),
		CommandText: combinedCommand,
		WorkingDir:  cwd,
		ExitStatus:  0,
	})

	if err != nil {
		// Non-fatal: print warning but don't fail
		fmt.Fprintf(os.Stderr, "fc: warning: failed to add to history: %v\n", err)
	}

	return nil
}

// executeShellReal is the real implementation of shell execution
func executeShellReal(shell, cmdText string) error {
	cmd := exec.Command(shell, "-c", cmdText)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
