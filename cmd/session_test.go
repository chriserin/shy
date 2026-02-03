package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSessionFilter(t *testing.T) {
	tests := []struct {
		name           string
		currentSession bool
		session        string
		wantApp        string
		wantPid        int64
		wantErr        bool
		errContains    string
	}{
		{
			name:    "empty session returns empty",
			session: "",
			wantApp: "",
			wantPid: 0,
			wantErr: false,
		},
		{
			name:    "app:pid format",
			session: "zsh:12345",
			wantApp: "zsh",
			wantPid: 12345,
			wantErr: false,
		},
		{
			name:    "app only format",
			session: "zsh",
			wantApp: "zsh",
			wantPid: 0,
			wantErr: false,
		},
		{
			name:    "bash app only",
			session: "bash",
			wantApp: "bash",
			wantPid: 0,
			wantErr: false,
		},
		{
			name:    "bash:pid format",
			session: "bash:67890",
			wantApp: "bash",
			wantPid: 67890,
			wantErr: false,
		},
		{
			name:        "invalid pid",
			session:     "zsh:abc",
			wantErr:     true,
			errContains: "invalid session PID",
		},
		{
			name:        "negative pid",
			session:     "zsh:-123",
			wantErr:     true,
			errContains: "invalid session PID: must be positive",
		},
		{
			name:        "zero pid",
			session:     "zsh:0",
			wantErr:     true,
			errContains: "invalid session PID: must be positive",
		},
		{
			name:        "too many colons",
			session:     "zsh:123:456",
			wantErr:     true,
			errContains: "invalid session format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, pid, err := parseSessionFilter(tt.currentSession, tt.session)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantApp, app)
			assert.Equal(t, tt.wantPid, pid)
		})
	}
}

func TestParseSessionFilter_CurrentSession(t *testing.T) {
	// Set up environment for current session detection
	os.Setenv("SHY_SESSION_PID", "54321")
	os.Setenv("SHELL", "/bin/zsh")
	defer os.Unsetenv("SHY_SESSION_PID")
	defer os.Unsetenv("SHELL")

	app, pid, err := parseSessionFilter(true, "")
	require.NoError(t, err)
	assert.Equal(t, "zsh", app)
	assert.Equal(t, int64(54321), pid)
}

func TestParseSessionFilter_CurrentSessionNotSet(t *testing.T) {
	// Ensure environment variables are not set
	os.Unsetenv("SHY_SESSION_PID")

	_, _, err := parseSessionFilter(true, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SHY_SESSION_PID not set")
}
