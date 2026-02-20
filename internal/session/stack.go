package session

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chris/shy/internal/db"
)

// GetCurrentDatabase returns the current database path for a given PPID
// Reads from session file (line 1) if exists, otherwise returns default database
func GetCurrentDatabase(ppid int) (string, error) {
	lines, err := readSessionFile(ppid)
	if err != nil {
		return "", err
	}

	// If no session file or empty, return default database path
	if len(lines) == 0 {
		return "", nil // Empty string signals to use default
	}

	return lines[0], nil
}

// PushDatabase pushes current database to stack and switches to new database
// 1. Validates and normalizes new path
// 2. Creates new database if it doesn't exist
// 3. Reads existing session file (or starts new stack with default DB)
// 4. Prepends current DB to stack, writes new DB as line 1
func PushDatabase(ppid int, newPath string) error {
	// Normalize the new database path
	normalizedPath, err := NormalizeDatabasePath(newPath)
	if err != nil {
		return fmt.Errorf("invalid database path: %w", err)
	}

	// Validate the path
	if err := ValidateDatabasePath(normalizedPath); err != nil {
		return fmt.Errorf("cannot use database path: %w", err)
	}

	// Create the new database if it doesn't exist
	if err := ensureDatabaseExists(normalizedPath); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Read current session stack
	lines, err := readSessionFile(ppid)
	if err != nil {
		return fmt.Errorf("failed to read session: %w", err)
	}

	// If no session exists, we need to add the default database as the base
	if len(lines) == 0 {
		// Get default database path (empty string will be resolved by db.New)
		defaultPath, err := getDefaultDatabasePath()
		if err != nil {
			return fmt.Errorf("failed to get default database path: %w", err)
		}
		lines = []string{defaultPath}
	}

	// Prepend new database to the stack
	// Current line[0] becomes line[1], new path becomes line[0]
	newLines := append([]string{normalizedPath}, lines...)

	// Write updated session file
	if err := writeSessionFile(ppid, newLines); err != nil {
		return fmt.Errorf("failed to write session: %w", err)
	}

	return nil
}

// PopDatabase pops the current database from stack and returns to previous
// Returns error if stack has only 1 entry (can't pop the default database)
func PopDatabase(ppid int) (string, error) {
	// Read current session stack
	lines, err := readSessionFile(ppid)
	if err != nil {
		return "", fmt.Errorf("failed to read session: %w", err)
	}

	// Check if we have anything to pop
	if len(lines) == 0 {
		return "", fmt.Errorf("no previous database to pop to")
	}

	if len(lines) == 1 {
		return "", fmt.Errorf("cannot pop default database")
	}

	// Remove line 0, shift everything up
	newLines := lines[1:]

	// Write updated session file
	if err := writeSessionFile(ppid, newLines); err != nil {
		return "", fmt.Errorf("failed to write session: %w", err)
	}

	// Return the new current database (former line 1, now line 0)
	return newLines[0], nil
}

// ensureDatabaseExists creates a database file and initializes schema if needed
func ensureDatabaseExists(path string) error {
	// Open database — migrations run automatically
	database, err := db.New(path)
	if err != nil {
		return err
	}
	defer database.Close()

	return nil
}

// getDefaultDatabasePath returns the default database path
// This mimics the logic in db.New() for the default path
func getDefaultDatabasePath() (string, error) {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".local/share")
	}
	return filepath.Join(dataDir, "shy/history.db"), nil
}
