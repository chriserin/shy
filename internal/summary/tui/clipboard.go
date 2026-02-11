package tui

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// yankResultMsg is sent after a yank attempt completes.
type yankResultMsg struct {
	err error
}

// oscClipboard writes an OSC 52 escape sequence to set the system clipboard.
// It implements bubbletea's ExecCommand interface so it can be used with tea.Exec.
// When running inside tmux, the sequence is wrapped in a DCS passthrough.
type oscClipboard struct {
	text   string
	stdout io.Writer
}

func (o *oscClipboard) Run() error {
	encoded := base64.StdEncoding.EncodeToString([]byte(o.text))

	var seq string
	if os.Getenv("TMUX") != "" {
		// Wrap in tmux DCS passthrough: ESCs inside the payload are doubled.
		seq = fmt.Sprintf("\x1bPtmux;\x1b\x1b]52;c;%s\x07\x1b\\", encoded)
	} else {
		seq = fmt.Sprintf("\x1b]52;c;%s\x07", encoded)
	}

	_, err := io.WriteString(o.stdout, seq)
	return err
}

func (o *oscClipboard) SetStdin(_ io.Reader)  {}
func (o *oscClipboard) SetStdout(w io.Writer)  { o.stdout = w }
func (o *oscClipboard) SetStderr(_ io.Writer)  {}

// yankToClipboard returns a tea.Cmd that writes text to the clipboard via OSC 52.
func yankToClipboard(text string) tea.Cmd {
	return tea.Exec(&oscClipboard{text: text}, func(err error) tea.Msg {
		return yankResultMsg{err: err}
	})
}
