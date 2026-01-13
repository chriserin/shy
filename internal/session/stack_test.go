package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetCurrentDatabase(t *testing.T) {
	testPID := 888888
	defer CleanupSession(testPID)

	t.Run("no session file returns empty string", func(t *testing.T) {
		CleanupSession(testPID)

		current, err := GetCurrentDatabase(testPID)
		if err != nil {
			t.Errorf("GetCurrentDatabase() error = %v", err)
		}
		if current != "" {
			t.Errorf("GetCurrentDatabase() = %q, want empty string", current)
		}
	})

	t.Run("returns first line of session file", func(t *testing.T) {
		lines := []string{
			"/tmp/current.db",
			"/tmp/previous.db",
			"/home/user/.local/share/shy/history.db",
		}
		err := writeSessionFile(testPID, lines)
		if err != nil {
			t.Fatalf("writeSessionFile() error = %v", err)
		}

		current, err := GetCurrentDatabase(testPID)
		if err != nil {
			t.Errorf("GetCurrentDatabase() error = %v", err)
		}
		if current != lines[0] {
			t.Errorf("GetCurrentDatabase() = %q, want %q", current, lines[0])
		}
	})
}

func TestPushDatabase(t *testing.T) {
	testPID := 777777
	defer CleanupSession(testPID)

	// Create temp directory for test databases
	tmpDir := t.TempDir()

	t.Run("push to new database from empty stack", func(t *testing.T) {
		CleanupSession(testPID)

		newDB := filepath.Join(tmpDir, "test1.db")
		err := PushDatabase(testPID, newDB)
		if err != nil {
			t.Fatalf("PushDatabase() error = %v", err)
		}

		// Check session file has 2 entries: new DB and default DB
		lines, err := readSessionFile(testPID)
		if err != nil {
			t.Fatalf("readSessionFile() error = %v", err)
		}

		if len(lines) != 2 {
			t.Errorf("session file has %d lines, want 2", len(lines))
		}

		if lines[0] != newDB {
			t.Errorf("line 0 = %q, want %q", lines[0], newDB)
		}

		// Line 1 should be default database path
		if !strings.HasSuffix(lines[1], "shy/history.db") {
			t.Errorf("line 1 = %q, want path ending with shy/history.db", lines[1])
		}

		// Database should be created
		if _, err := os.Stat(newDB); os.IsNotExist(err) {
			t.Errorf("database not created at %s", newDB)
		}
	})

	t.Run("push auto-appends .db extension", func(t *testing.T) {
		CleanupSession(testPID)

		newDB := filepath.Join(tmpDir, "test_no_ext")
		err := PushDatabase(testPID, newDB)
		if err != nil {
			t.Fatalf("PushDatabase() error = %v", err)
		}

		lines, err := readSessionFile(testPID)
		if err != nil {
			t.Fatalf("readSessionFile() error = %v", err)
		}

		expected := newDB + ".db"
		if lines[0] != expected {
			t.Errorf("line 0 = %q, want %q", lines[0], expected)
		}

		// Database should be created with .db extension
		if _, err := os.Stat(expected); os.IsNotExist(err) {
			t.Errorf("database not created at %s", expected)
		}
	})

	t.Run("nested push operations", func(t *testing.T) {
		CleanupSession(testPID)

		db1 := filepath.Join(tmpDir, "projectA.db")
		db2 := filepath.Join(tmpDir, "projectB.db")
		db3 := filepath.Join(tmpDir, "projectC.db")

		// Push first database
		err := PushDatabase(testPID, db1)
		if err != nil {
			t.Fatalf("PushDatabase(db1) error = %v", err)
		}

		// Push second database
		err = PushDatabase(testPID, db2)
		if err != nil {
			t.Fatalf("PushDatabase(db2) error = %v", err)
		}

		// Push third database
		err = PushDatabase(testPID, db3)
		if err != nil {
			t.Fatalf("PushDatabase(db3) error = %v", err)
		}

		// Check session file structure
		lines, err := readSessionFile(testPID)
		if err != nil {
			t.Fatalf("readSessionFile() error = %v", err)
		}

		if len(lines) != 4 {
			t.Fatalf("session file has %d lines, want 4", len(lines))
		}

		if lines[0] != db3 {
			t.Errorf("line 0 = %q, want %q", lines[0], db3)
		}
		if lines[1] != db2 {
			t.Errorf("line 1 = %q, want %q", lines[1], db2)
		}
		if lines[2] != db1 {
			t.Errorf("line 2 = %q, want %q", lines[2], db1)
		}
	})

	t.Run("push to same path twice", func(t *testing.T) {
		CleanupSession(testPID)

		db := filepath.Join(tmpDir, "same.db")

		// First push
		err := PushDatabase(testPID, db)
		if err != nil {
			t.Fatalf("PushDatabase() first error = %v", err)
		}

		// Pop back
		_, err = PopDatabase(testPID)
		if err != nil {
			t.Fatalf("PopDatabase() error = %v", err)
		}

		// Push again to same path
		err = PushDatabase(testPID, db)
		if err != nil {
			t.Fatalf("PushDatabase() second error = %v", err)
		}

		// Should succeed and database should still exist
		if _, err := os.Stat(db); os.IsNotExist(err) {
			t.Errorf("database not found at %s", db)
		}
	})

	t.Run("push with invalid path returns error", func(t *testing.T) {
		CleanupSession(testPID)

		err := PushDatabase(testPID, "")
		if err == nil {
			t.Error("PushDatabase() with empty path should error")
		}
	})

	t.Run("push to restricted path returns error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping test when running as root")
		}

		CleanupSession(testPID)

		err := PushDatabase(testPID, "/root/restricted.db")
		if err == nil {
			t.Error("PushDatabase() to restricted path should error")
		}
	})
}

func TestPopDatabase(t *testing.T) {
	testPID := 666666
	defer CleanupSession(testPID)

	tmpDir := t.TempDir()

	t.Run("pop with no session returns error", func(t *testing.T) {
		CleanupSession(testPID)

		_, err := PopDatabase(testPID)
		if err == nil {
			t.Error("PopDatabase() with no session should error")
		}
		if !strings.Contains(err.Error(), "no previous database") {
			t.Errorf("PopDatabase() error = %v, want error about no previous database", err)
		}
	})

	t.Run("pop with only default database returns error", func(t *testing.T) {
		CleanupSession(testPID)

		// Create session with only default database
		defaultDB, _ := getDefaultDatabasePath()
		lines := []string{defaultDB}
		err := writeSessionFile(testPID, lines)
		if err != nil {
			t.Fatalf("writeSessionFile() error = %v", err)
		}

		_, err = PopDatabase(testPID)
		if err == nil {
			t.Error("PopDatabase() with only default should error")
		}
		if !strings.Contains(err.Error(), "cannot pop default database") {
			t.Errorf("PopDatabase() error = %v, want error about popping default", err)
		}
	})

	t.Run("pop returns to previous database", func(t *testing.T) {
		CleanupSession(testPID)

		db := filepath.Join(tmpDir, "temp.db")

		// Push a database
		err := PushDatabase(testPID, db)
		if err != nil {
			t.Fatalf("PushDatabase() error = %v", err)
		}

		// Pop back
		prevDB, err := PopDatabase(testPID)
		if err != nil {
			t.Fatalf("PopDatabase() error = %v", err)
		}

		// Should return default database
		if !strings.HasSuffix(prevDB, "shy/history.db") {
			t.Errorf("PopDatabase() = %q, want path ending with shy/history.db", prevDB)
		}

		// Session file should have only 1 entry now
		lines, err := readSessionFile(testPID)
		if err != nil {
			t.Fatalf("readSessionFile() error = %v", err)
		}

		if len(lines) != 1 {
			t.Errorf("session file has %d lines, want 1", len(lines))
		}
	})

	t.Run("pop through nested stack", func(t *testing.T) {
		CleanupSession(testPID)

		db1 := filepath.Join(tmpDir, "db1.db")
		db2 := filepath.Join(tmpDir, "db2.db")
		db3 := filepath.Join(tmpDir, "db3.db")

		// Push three databases
		PushDatabase(testPID, db1)
		PushDatabase(testPID, db2)
		PushDatabase(testPID, db3)

		// Pop first time - should return db2
		prevDB, err := PopDatabase(testPID)
		if err != nil {
			t.Fatalf("PopDatabase() first error = %v", err)
		}
		if prevDB != db2 {
			t.Errorf("PopDatabase() first = %q, want %q", prevDB, db2)
		}

		// Pop second time - should return db1
		prevDB, err = PopDatabase(testPID)
		if err != nil {
			t.Fatalf("PopDatabase() second error = %v", err)
		}
		if prevDB != db1 {
			t.Errorf("PopDatabase() second = %q, want %q", prevDB, db1)
		}

		// Pop third time - should return default
		prevDB, err = PopDatabase(testPID)
		if err != nil {
			t.Fatalf("PopDatabase() third error = %v", err)
		}
		if !strings.HasSuffix(prevDB, "shy/history.db") {
			t.Errorf("PopDatabase() third = %q, want path ending with shy/history.db", prevDB)
		}

		// Pop fourth time - should error
		_, err = PopDatabase(testPID)
		if err == nil {
			t.Error("PopDatabase() fourth should error")
		}
	})

	t.Run("multiple pops progressively shrink session file", func(t *testing.T) {
		CleanupSession(testPID)

		db1 := filepath.Join(tmpDir, "prog1.db")
		db2 := filepath.Join(tmpDir, "prog2.db")

		PushDatabase(testPID, db1)
		PushDatabase(testPID, db2)

		// Should have 3 lines
		lines, _ := readSessionFile(testPID)
		if len(lines) != 3 {
			t.Errorf("after 2 pushes: %d lines, want 3", len(lines))
		}

		PopDatabase(testPID)

		// Should have 2 lines
		lines, _ = readSessionFile(testPID)
		if len(lines) != 2 {
			t.Errorf("after 1 pop: %d lines, want 2", len(lines))
		}

		PopDatabase(testPID)

		// Should have 1 line
		lines, _ = readSessionFile(testPID)
		if len(lines) != 1 {
			t.Errorf("after 2 pops: %d lines, want 1", len(lines))
		}
	})
}

func TestGetDefaultDatabasePath(t *testing.T) {
	path, err := getDefaultDatabasePath()
	if err != nil {
		t.Fatalf("getDefaultDatabasePath() error = %v", err)
	}

	if !strings.HasSuffix(path, "shy/history.db") {
		t.Errorf("getDefaultDatabasePath() = %q, want path ending with shy/history.db", path)
	}

	if !filepath.IsAbs(path) {
		t.Errorf("getDefaultDatabasePath() = %q, want absolute path", path)
	}
}

func TestGetDefaultDatabasePath_XDGDataHome(t *testing.T) {
	// Save original value
	originalXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Set custom XDG_DATA_HOME
	customData := "/custom/data"
	os.Setenv("XDG_DATA_HOME", customData)

	path, err := getDefaultDatabasePath()
	if err != nil {
		t.Fatalf("getDefaultDatabasePath() error = %v", err)
	}

	expectedPrefix := filepath.Join(customData, "shy")
	if !strings.HasPrefix(path, expectedPrefix) {
		t.Errorf("getDefaultDatabasePath() = %q, want path starting with %q", path, expectedPrefix)
	}
}
