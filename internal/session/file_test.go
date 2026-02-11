package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadWriteSessionFile(t *testing.T) {
	// Use a unique PID that won't conflict with real sessions
	testPID := 999999

	// Clean up any existing session file
	defer CleanupSession(testPID)

	t.Run("read non-existent file returns empty slice", func(t *testing.T) {
		CleanupSession(testPID) // Ensure it doesn't exist
		lines, err := readSessionFile(testPID)
		if err != nil {
			t.Errorf("readSessionFile() unexpected error = %v", err)
		}
		if len(lines) != 0 {
			t.Errorf("readSessionFile() = %v, want empty slice", lines)
		}
	})

	t.Run("write and read session file", func(t *testing.T) {
		testLines := []string{
			"/tmp/current.db",
			"/tmp/previous.db",
			"/home/user/.local/share/shy/history.db",
		}

		err := writeSessionFile(testPID, testLines)
		if err != nil {
			t.Fatalf("writeSessionFile() error = %v", err)
		}

		lines, err := readSessionFile(testPID)
		if err != nil {
			t.Fatalf("readSessionFile() error = %v", err)
		}

		if len(lines) != len(testLines) {
			t.Errorf("readSessionFile() got %d lines, want %d", len(lines), len(testLines))
		}

		for i, line := range lines {
			if line != testLines[i] {
				t.Errorf("readSessionFile() line %d = %q, want %q", i, line, testLines[i])
			}
		}
	})

	t.Run("write empty lines", func(t *testing.T) {
		err := writeSessionFile(testPID, []string{})
		if err != nil {
			t.Fatalf("writeSessionFile() error = %v", err)
		}

		lines, err := readSessionFile(testPID)
		if err != nil {
			t.Fatalf("readSessionFile() error = %v", err)
		}

		if len(lines) != 0 {
			t.Errorf("readSessionFile() = %v, want empty slice", lines)
		}
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		// Write initial content
		initial := []string{"/tmp/first.db"}
		err := writeSessionFile(testPID, initial)
		if err != nil {
			t.Fatalf("writeSessionFile() initial error = %v", err)
		}

		// Overwrite with new content
		updated := []string{"/tmp/second.db", "/tmp/third.db"}
		err = writeSessionFile(testPID, updated)
		if err != nil {
			t.Fatalf("writeSessionFile() update error = %v", err)
		}

		lines, err := readSessionFile(testPID)
		if err != nil {
			t.Fatalf("readSessionFile() error = %v", err)
		}

		if len(lines) != len(updated) {
			t.Errorf("readSessionFile() got %d lines, want %d", len(lines), len(updated))
		}

		for i, line := range lines {
			if line != updated[i] {
				t.Errorf("readSessionFile() line %d = %q, want %q", i, line, updated[i])
			}
		}
	})

	t.Run("read skips empty lines", func(t *testing.T) {
		path := getSessionFilePath(testPID)
		os.MkdirAll(filepath.Dir(path), 0755)

		// Write file with empty lines manually
		content := "/tmp/first.db\n\n/tmp/second.db\n\n\n/tmp/third.db\n"
		err := os.WriteFile(path, []byte(content), 0600)
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		lines, err := readSessionFile(testPID)
		if err != nil {
			t.Fatalf("readSessionFile() error = %v", err)
		}

		expected := []string{"/tmp/first.db", "/tmp/second.db", "/tmp/third.db"}
		if len(lines) != len(expected) {
			t.Errorf("readSessionFile() got %d lines, want %d", len(lines), len(expected))
		}
	})
}

func TestCleanupSession(t *testing.T) {
	testPID := 999998

	t.Run("cleanup removes session file", func(t *testing.T) {
		// Create a session file
		lines := []string{"/tmp/test.db"}
		err := writeSessionFile(testPID, lines)
		if err != nil {
			t.Fatalf("writeSessionFile() error = %v", err)
		}

		// Verify it exists
		path := getSessionFilePath(testPID)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatal("Session file was not created")
		}

		// Clean up
		err = CleanupSession(testPID)
		if err != nil {
			t.Errorf("CleanupSession() error = %v", err)
		}

		// Verify it's gone
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("Session file was not removed")
		}
	})

	t.Run("cleanup non-existent file succeeds", func(t *testing.T) {
		// Ensure file doesn't exist
		CleanupSession(testPID)

		// Should not error
		err := CleanupSession(testPID)
		if err != nil {
			t.Errorf("CleanupSession() error = %v, want nil", err)
		}
	})
}

func TestGetSessionFilePath(t *testing.T) {
	testPID := 12345

	path := getSessionFilePath(testPID)

	// Should contain the PID
	expectedSuffix := "12345.txt"
	if filepath.Base(path) != expectedSuffix {
		t.Errorf("getSessionFilePath() = %v, want path ending with %v", path, expectedSuffix)
	}

	// Should be in cache directory
	if !filepath.IsAbs(path) {
		t.Errorf("getSessionFilePath() = %v, want absolute path", path)
	}

	// Should contain 'shy/sessions'
	if !contains(path, "shy") || !contains(path, "sessions") {
		t.Errorf("getSessionFilePath() = %v, want path containing 'shy/sessions'", path)
	}
}

func TestGetSessionFilePath_XDGCacheHome(t *testing.T) {
	// Save original value
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)

	// Set custom XDG_CACHE_HOME
	customCache := "/custom/cache"
	os.Setenv("XDG_CACHE_HOME", customCache)

	path := getSessionFilePath(12345)

	expectedPrefix := filepath.Join(customCache, "shy", "sessions")
	if !contains(path, expectedPrefix) {
		t.Errorf("getSessionFilePath() = %v, want path containing %v", path, expectedPrefix)
	}
}

func contains(s, substr string) bool {
	return (s == substr || len(s) >= len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
