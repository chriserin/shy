package models

import (
	"strings"
	"time"
)

// Command represents a shell command entry in the history database
type Command struct {
	ID           int64
	Timestamp    int64
	ExitStatus   int
	CommandText  string
	WorkingDir   string
	GitRepo      *string
	GitBranch    *string
	Duration     *int64  // Duration in milliseconds, null if not captured
	SourceApp    *string // Shell application (e.g., "zsh", "bash"), null if not tracked
	SourcePid    *int64  // Process ID of the shell session, null if not tracked
	SourceActive *bool   // Whether the shell session is still active, null if not tracked
}

func (c *Command) TrimCommandText() {
	c.CommandText = strings.Trim(c.CommandText, "\n ")
}

// NewCommand creates a new Command with the current timestamp
func NewCommand(commandText, workingDir string, exitStatus int) *Command {
	return &Command{
		Timestamp:   time.Now().Unix(),
		ExitStatus:  exitStatus,
		CommandText: commandText,
		WorkingDir:  workingDir,
	}
}
