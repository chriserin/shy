package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/internal/git"
	"github.com/chris/shy/pkg/models"
)

var (
	command    string
	dir        string
	status     int
	gitRepo    string
	gitBranch  string
	timestamp  int64
	duration   int64
)

var insertCmd = &cobra.Command{
	Use:   "insert",
	Short: "Insert a command into the history database",
	Long:  "Insert a command with metadata into the shell history database",
	RunE:  runInsert,
}

func init() {
	rootCmd.AddCommand(insertCmd)

	insertCmd.Flags().StringVar(&command, "command", "", "Command text (required)")
	insertCmd.Flags().StringVar(&dir, "dir", "", "Working directory (required)")
	insertCmd.Flags().IntVar(&status, "status", 0, "Exit status (default: 0)")
	insertCmd.Flags().StringVar(&gitRepo, "git-repo", "", "Git repository URL")
	insertCmd.Flags().StringVar(&gitBranch, "git-branch", "", "Git branch name")
	insertCmd.Flags().Int64Var(&timestamp, "timestamp", 0, "Unix timestamp (default: current time)")
	insertCmd.Flags().Int64Var(&duration, "duration", 0, "Command duration in milliseconds")

	insertCmd.MarkFlagRequired("command")
	insertCmd.MarkFlagRequired("dir")
}

func runInsert(cmd *cobra.Command, args []string) error {
	// Validate required parameters
	if command == "" {
		return fmt.Errorf("--command is required")
	}
	if dir == "" {
		return fmt.Errorf("--dir is required")
	}

	// Open database
	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Create command model
	cmdModel := models.NewCommand(command, dir, status)

	// Override timestamp if provided
	if timestamp != 0 {
		cmdModel.Timestamp = timestamp
	}

	// Set duration if provided
	if duration > 0 {
		cmdModel.Duration = &duration
	}

	// Handle git context
	var finalGitRepo *string
	var finalGitBranch *string

	// If explicit git context provided, use it
	if gitRepo != "" || gitBranch != "" {
		if gitRepo != "" {
			finalGitRepo = &gitRepo
		}
		if gitBranch != "" {
			finalGitBranch = &gitBranch
		}
	} else {
		// Auto-detect git context
		gitCtx, err := git.DetectGitContext(dir)
		if err == nil && gitCtx != nil {
			if gitCtx.Repo != "" {
				finalGitRepo = &gitCtx.Repo
			}
			if gitCtx.Branch != "" {
				finalGitBranch = &gitCtx.Branch
			}
		}
	}

	cmdModel.GitRepo = finalGitRepo
	cmdModel.GitBranch = finalGitBranch

	// Insert command
	id, err := database.InsertCommand(cmdModel)
	if err != nil {
		return fmt.Errorf("failed to insert command: %w", err)
	}

	fmt.Printf("Inserted command with ID: %d\n", id)
	return nil
}
