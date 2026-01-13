package session

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

// readSessionFile reads all lines from a session file
// Returns empty slice if file doesn't exist (not an error)
func readSessionFile(ppid int) ([]string, error) {
	path := getSessionFilePath(ppid)

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // Not an error, just no session yet
		}
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" { // Skip empty lines
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	return lines, nil
}

// writeSessionFile writes lines to a session file atomically
// Creates parent directory if needed
func writeSessionFile(ppid int, lines []string) error {
	path := getSessionFilePath(ppid)

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Write to temporary file first
	tmpPath := path + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create temporary session file: %w", err)
	}

	// Write all lines
	for _, line := range lines {
		if _, err := fmt.Fprintln(file, line); err != nil {
			file.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("failed to write to session file: %w", err)
		}
	}

	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close session file: %w", err)
	}

	// Atomically rename temporary file to actual file
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename session file: %w", err)
	}

	return nil
}

// getSessionFilePath returns the path to the session file for a given PPID
// Uses XDG_CACHE_HOME if set, otherwise ~/.cache
func getSessionFilePath(ppid int) string {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback to /tmp if we can't get home directory
			return filepath.Join("/tmp", "shy", "sessions", fmt.Sprintf("%d.txt", ppid))
		}
		cacheDir = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheDir, "shy", "sessions", fmt.Sprintf("%d.txt", ppid))
}

// CleanupSession deletes the session file for a given PID
// Fails silently if file doesn't exist
func CleanupSession(pid int) error {
	path := getSessionFilePath(pid)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove session file: %w", err)
	}
	return nil
}
