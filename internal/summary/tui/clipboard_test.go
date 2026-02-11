package tui

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOscClipboard(t *testing.T) {
	text := "echo hello"
	encoded := base64.StdEncoding.EncodeToString([]byte(text))

	t.Run("plain OSC 52", func(t *testing.T) {
		t.Setenv("TMUX", "")

		var buf bytes.Buffer
		o := &oscClipboard{text: text}
		o.SetStdout(&buf)

		err := o.Run()
		require.NoError(t, err)

		want := "\x1b]52;c;" + encoded + "\x07"
		assert.Equal(t, want, buf.String())
	})

	t.Run("tmux DCS passthrough", func(t *testing.T) {
		t.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")

		var buf bytes.Buffer
		o := &oscClipboard{text: text}
		o.SetStdout(&buf)

		err := o.Run()
		require.NoError(t, err)

		want := "\x1bPtmux;\x1b\x1b]52;c;" + encoded + "\x07\x1b\\"
		assert.Equal(t, want, buf.String())
	})

	t.Run("encodes multi-byte text", func(t *testing.T) {
		t.Setenv("TMUX", "")

		multiByteText := "git log --oneline | head -5"
		multiEncoded := base64.StdEncoding.EncodeToString([]byte(multiByteText))

		var buf bytes.Buffer
		o := &oscClipboard{text: multiByteText}
		o.SetStdout(&buf)

		err := o.Run()
		require.NoError(t, err)

		want := "\x1b]52;c;" + multiEncoded + "\x07"
		assert.Equal(t, want, buf.String())
	})
}
