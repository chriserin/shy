package models

import "time"

// Command represents a shell command entry in the history database
type Command struct {
	ID          int64
	Timestamp   int64
	ExitStatus  int
	CommandText string
	WorkingDir  string
	GitRepo     *string
	GitBranch   *string
	Duration    *int64 // Duration in milliseconds, null if not captured
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
