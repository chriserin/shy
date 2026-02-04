package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/internal/session"
	"github.com/chris/shy/pkg/models"
)

// TestPushToNewDatabase tests pushing to a new database file
func TestPushToNewDatabase(t *testing.T) {
	testPID := 555555
	defer session.CleanupSession(testPID)

	tmpDir := t.TempDir()
	newDB := filepath.Join(tmpDir, "project_history.db")

	// Push to new database
	err := session.PushDatabase(testPID, newDB)
	require.NoError(t, err)

	// Verify current database is the new one
	currentDB, err := session.GetCurrentDatabase(testPID)
	require.NoError(t, err)
	assert.Equal(t, newDB, currentDB)

	// Verify database was created
	_, err = os.Stat(newDB)
	assert.NoError(t, err, "database should be created")

	// Verify we can open and use the new database
	database, err := db.NewForTesting(newDB)
	require.NoError(t, err)
	defer database.Close()

	cmd := &models.Command{
		CommandText: "echo 'command2'",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
		Timestamp:   1704470400,
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err)

	// Verify command is in new database by getting most recent event ID
	mostRecent, err := database.GetMostRecentEventID()
	require.NoError(t, err)
	assert.Equal(t, int64(1), mostRecent)
}

// TestAutoAppendDbExtension tests that .db extension is added when missing
func TestAutoAppendDbExtension(t *testing.T) {
	testPID := 555554
	defer session.CleanupSession(testPID)

	tmpDir := t.TempDir()
	pathWithoutExt := filepath.Join(tmpDir, "myproject")

	err := session.PushDatabase(testPID, pathWithoutExt)
	require.NoError(t, err)

	currentDB, err := session.GetCurrentDatabase(testPID)
	require.NoError(t, err)

	expectedPath := pathWithoutExt + ".db"
	assert.Equal(t, expectedPath, currentDB)

	// Verify database was created with .db extension
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err, "database should be created with .db extension")
}

// TestPreserveDbExtension tests that .db extension is preserved when present
func TestPreserveDbExtension(t *testing.T) {
	testPID := 555553
	defer session.CleanupSession(testPID)

	tmpDir := t.TempDir()
	pathWithExt := filepath.Join(tmpDir, "myproject.db")

	err := session.PushDatabase(testPID, pathWithExt)
	require.NoError(t, err)

	currentDB, err := session.GetCurrentDatabase(testPID)
	require.NoError(t, err)

	assert.Equal(t, pathWithExt, currentDB)
}

// TestMultipleNestedPushOperations tests nested push operations
func TestMultipleNestedPushOperations(t *testing.T) {
	testPID := 555552
	defer session.CleanupSession(testPID)

	tmpDir := t.TempDir()
	dbA := filepath.Join(tmpDir, "projectA.db")
	dbB := filepath.Join(tmpDir, "projectB.db")

	// Push first database
	err := session.PushDatabase(testPID, dbA)
	require.NoError(t, err)

	currentDB, err := session.GetCurrentDatabase(testPID)
	require.NoError(t, err)
	assert.Equal(t, dbA, currentDB)

	// Push second database
	err = session.PushDatabase(testPID, dbB)
	require.NoError(t, err)

	currentDB, err = session.GetCurrentDatabase(testPID)
	require.NoError(t, err)
	assert.Equal(t, dbB, currentDB)
}

// TestPopThroughNestedStack tests popping through a nested stack
func TestPopThroughNestedStack(t *testing.T) {
	testPID := 555551
	defer session.CleanupSession(testPID)

	tmpDir := t.TempDir()
	dbA := filepath.Join(tmpDir, "projectA.db")
	dbB := filepath.Join(tmpDir, "projectB.db")
	dbC := filepath.Join(tmpDir, "projectC.db")

	// Push three databases
	require.NoError(t, session.PushDatabase(testPID, dbA))
	require.NoError(t, session.PushDatabase(testPID, dbB))
	require.NoError(t, session.PushDatabase(testPID, dbC))

	// Pop first time - should return to dbB
	prevDB, err := session.PopDatabase(testPID)
	require.NoError(t, err)
	assert.Equal(t, dbB, prevDB)

	currentDB, err := session.GetCurrentDatabase(testPID)
	require.NoError(t, err)
	assert.Equal(t, dbB, currentDB)

	// Pop second time - should return to dbA
	prevDB, err = session.PopDatabase(testPID)
	require.NoError(t, err)
	assert.Equal(t, dbA, prevDB)

	currentDB, err = session.GetCurrentDatabase(testPID)
	require.NoError(t, err)
	assert.Equal(t, dbA, currentDB)

	// Pop third time - should return to default
	prevDB, err = session.PopDatabase(testPID)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(prevDB, "shy/history.db"))
}

// TestPopWhenNoSessionExists tests error when popping with no session
func TestPopWhenNoSessionExists(t *testing.T) {
	testPID := 555550
	defer session.CleanupSession(testPID)

	_, err := session.PopDatabase(testPID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no previous database")
}

// TestPopAtBottomOfStack tests error when popping at default database
func TestPopAtBottomOfStack(t *testing.T) {
	testPID := 555549
	defer session.CleanupSession(testPID)

	tmpDir := t.TempDir()
	db1 := filepath.Join(tmpDir, "test.db")

	// Push and pop back to default
	require.NoError(t, session.PushDatabase(testPID, db1))
	_, err := session.PopDatabase(testPID)
	require.NoError(t, err)

	// Try to pop again - should error
	_, err = session.PopDatabase(testPID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot pop default database")
}

// TestPushInvalidPath tests error handling for invalid paths
func TestPushInvalidPath(t *testing.T) {
	testPID := 555548
	defer session.CleanupSession(testPID)

	// Empty path
	err := session.PushDatabase(testPID, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// TestPushRestrictedPath tests error handling for restricted paths
func TestPushRestrictedPath(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	testPID := 555547
	defer session.CleanupSession(testPID)

	err := session.PushDatabase(testPID, "/root/restricted.db")
	require.Error(t, err)
}

// TestSessionIsolation tests that different PIDs have isolated sessions
func TestSessionIsolation(t *testing.T) {
	testPID1 := 444444
	testPID2 := 444445
	defer session.CleanupSession(testPID1)
	defer session.CleanupSession(testPID2)

	tmpDir := t.TempDir()
	dbA := filepath.Join(tmpDir, "sessionA.db")
	dbB := filepath.Join(tmpDir, "sessionB.db")

	// Push different databases in different sessions
	require.NoError(t, session.PushDatabase(testPID1, dbA))
	require.NoError(t, session.PushDatabase(testPID2, dbB))

	// Verify each session has its own database
	currentDB1, err := session.GetCurrentDatabase(testPID1)
	require.NoError(t, err)
	assert.Equal(t, dbA, currentDB1)

	currentDB2, err := session.GetCurrentDatabase(testPID2)
	require.NoError(t, err)
	assert.Equal(t, dbB, currentDB2)

	// Verify they're different
	assert.NotEqual(t, currentDB1, currentDB2)
}

// TestPushCreatesNewDatabase tests that push creates database if it doesn't exist
func TestPushCreatesNewDatabase(t *testing.T) {
	testPID := 444443
	defer session.CleanupSession(testPID)

	tmpDir := t.TempDir()
	newDB := filepath.Join(tmpDir, "new_db.db")

	// Verify database doesn't exist
	_, err := os.Stat(newDB)
	require.True(t, os.IsNotExist(err))

	// Push to new database
	err = session.PushDatabase(testPID, newDB)
	require.NoError(t, err)

	// Verify database was created
	_, err = os.Stat(newDB)
	require.NoError(t, err)

	// Verify it's a valid database
	database, err := db.NewForTesting(newDB)
	require.NoError(t, err)
	defer database.Close()
}

// TestPushToSameDatabaseTwice tests pushing to same path twice
func TestPushToSameDatabaseTwice(t *testing.T) {
	testPID := 444442
	defer session.CleanupSession(testPID)

	tmpDir := t.TempDir()
	db1 := filepath.Join(tmpDir, "test.db")

	// First push
	err := session.PushDatabase(testPID, db1)
	require.NoError(t, err)

	// Add a command to the database
	database, err := db.NewForTesting(db1)
	require.NoError(t, err)
	cmd := &models.Command{
		CommandText: "echo 'first context'",
		WorkingDir:  "/tmp",
		ExitStatus:  0,
		Timestamp:   1704470400,
	}
	_, err = database.InsertCommand(cmd)
	require.NoError(t, err)
	database.Close()

	// Pop back
	_, err = session.PopDatabase(testPID)
	require.NoError(t, err)

	// Push again to same path
	err = session.PushDatabase(testPID, db1)
	require.NoError(t, err)

	// Verify command still exists
	database, err = db.NewForTesting(db1)
	require.NoError(t, err)
	defer database.Close()

	mostRecent, err := database.GetMostRecentEventID()
	require.NoError(t, err)
	assert.Equal(t, int64(1), mostRecent, "command should still exist in database")
}

// TestRelativePath tests pushing with relative path
func TestRelativePath(t *testing.T) {
	testPID := 444441
	defer session.CleanupSession(testPID)

	// Create temp directory and change to it
	tmpDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWD)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Push with relative path
	err = session.PushDatabase(testPID, "./local_history")
	require.NoError(t, err)

	// Verify it was converted to absolute path
	currentDB, err := session.GetCurrentDatabase(testPID)
	require.NoError(t, err)

	expectedPath := filepath.Join(tmpDir, "local_history.db")
	assert.Equal(t, expectedPath, currentDB)
}

// TestCleanupSessionCommand tests the cleanup-session command
func TestCleanupSessionCommand(t *testing.T) {
	testPID := 333333
	defer session.CleanupSession(testPID)

	tmpDir := t.TempDir()
	db1 := filepath.Join(tmpDir, "test.db")

	// Create a session
	require.NoError(t, session.PushDatabase(testPID, db1))

	// Verify session file exists
	sessionFile := filepath.Join(os.Getenv("HOME"), ".cache", "shy", "sessions", "333333.txt")
	if os.Getenv("XDG_CACHE_HOME") != "" {
		sessionFile = filepath.Join(os.Getenv("XDG_CACHE_HOME"), "shy", "sessions", "333333.txt")
	}
	_, err := os.Stat(sessionFile)
	require.NoError(t, err, "session file should exist")

	// Clean up session
	err = session.CleanupSession(testPID)
	require.NoError(t, err)

	// Verify session file is deleted
	_, err = os.Stat(sessionFile)
	require.True(t, os.IsNotExist(err), "session file should be deleted")
}

// TestCleanupNonExistentSession tests cleanup of non-existent session
func TestCleanupNonExistentSession(t *testing.T) {
	testPID := 333332

	// Ensure session doesn't exist
	session.CleanupSession(testPID)

	// Should not error
	err := session.CleanupSession(testPID)
	require.NoError(t, err)
}
