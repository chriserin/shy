package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NormalizeDatabasePath normalizes a database path
// - Expands ~ to home directory
// - Resolves relative paths to absolute
// - Auto-appends .db extension if no extension present
// - Returns normalized absolute path
func NormalizeDatabasePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("database path cannot be empty")
	}

	// Expand tilde
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		if len(path) == 1 {
			path = home
		} else if path[1] == '/' || path[1] == filepath.Separator {
			path = filepath.Join(home, path[2:])
		} else {
			path = filepath.Join(home, path[1:])
		}
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Auto-append .db extension if no extension present
	ext := filepath.Ext(absPath)
	if ext == "" {
		absPath = absPath + ".db"
	}

	return absPath, nil
}

// ValidateDatabasePath validates that a database path is usable
// - Checks that parent directory exists or can be created
// - Checks write permissions
// - Ensures path is not a directory
func ValidateDatabasePath(path string) error {
	// Check if path already exists
	info, err := os.Stat(path)
	if err == nil {
		// Path exists - make sure it's not a directory
		if info.IsDir() {
			return fmt.Errorf("path is a directory, not a file: %s", path)
		}
		// File exists, check if writable by trying to open it
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			return fmt.Errorf("cannot write to existing database: %w", err)
		}
		file.Close()
		return nil
	}

	// Path doesn't exist - check parent directory
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	// Check parent directory exists and is writable
	parentDir := filepath.Dir(path)
	parentInfo, err := os.Stat(parentDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create parent directory
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("cannot create parent directory: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to stat parent directory: %w", err)
	}

	if !parentInfo.IsDir() {
		return fmt.Errorf("parent path is not a directory: %s", parentDir)
	}

	// Check if parent directory is writable by trying to create a temp file
	testFile := filepath.Join(parentDir, ".shy_write_test")
	file, err := os.Create(testFile)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "read-only") {
			return fmt.Errorf("permission denied: cannot write to %s", parentDir)
		}
		return fmt.Errorf("cannot write to parent directory: %w", err)
	}
	file.Close()
	os.Remove(testFile)

	return nil
}
