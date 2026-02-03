package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// detectCurrentSession detects the current shell session from environment variables
// Returns (sourceApp, sourcePid, detected, error)
func detectCurrentSession() (string, int64, bool, error) {
	// Check for SHY_SESSION_PID environment variable
	sessionPidStr := os.Getenv("SHY_SESSION_PID")
	if sessionPidStr == "" {
		return "", 0, false, nil
	}

	// Parse the PID
	sessionPid, err := strconv.ParseInt(sessionPidStr, 10, 64)
	if err != nil {
		return "", 0, false, fmt.Errorf("invalid SHY_SESSION_PID value: %s", sessionPidStr)
	}

	// Detect shell from SHELL environment variable
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return "", 0, false, fmt.Errorf("SHELL environment variable not set")
	}

	// Extract shell name from path (e.g., /bin/zsh -> zsh)
	shellName := filepath.Base(shellPath)

	return shellName, sessionPid, true, nil
}

// parseSessionFilter parses session filter from flags.
// If currentSession is true, auto-detects from environment.
// If session is non-empty, parses as "app:pid" or just "app" format.
// Returns (sourceApp, sourcePid, error). sourcePid is 0 if not specified.
func parseSessionFilter(currentSession bool, session string) (string, int64, error) {
	if currentSession {
		sourceApp, sourcePid, detected, err := detectCurrentSession()
		if err != nil {
			return "", 0, fmt.Errorf("failed to detect current session: %w", err)
		}
		if !detected {
			return "", 0, fmt.Errorf("could not auto-detect session: SHY_SESSION_PID not set")
		}
		return sourceApp, sourcePid, nil
	}

	if session != "" {
		parts := strings.Split(session, ":")
		if len(parts) == 1 {
			// Just app name, no pid
			return parts[0], 0, nil
		}
		if len(parts) == 2 {
			sourceApp := parts[0]
			var sourcePid int64
			_, err := fmt.Sscanf(parts[1], "%d", &sourcePid)
			if err != nil {
				return "", 0, fmt.Errorf("invalid session PID: %s", parts[1])
			}
			if sourcePid <= 0 {
				return "", 0, fmt.Errorf("invalid session PID: must be positive")
			}
			return sourceApp, sourcePid, nil
		}
		return "", 0, fmt.Errorf("invalid session format: expected 'app' or 'app:pid' (e.g., zsh or zsh:12345)")
	}

	return "", 0, nil
}
