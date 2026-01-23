package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chris/shy/pkg/models"
)

// TestMigrate_NoMigrationNeeded tests that migrate is a no-op when schema is current
func TestMigrate_NoMigrationNeeded(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	// Given: a database with the current schema (created by New())
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := New(dbPath)
	require.NoError(t, err, "failed to create database")
	defer database.Close()

	// Insert a test command
	cmd := models.NewCommand("test command", "/home/user", 0)
	id, err := database.InsertCommand(cmd)
	require.NoError(t, err, "failed to insert command")

	// When: migrate is called again
	err = database.migrate()

	// Then: no error should occur
	require.NoError(t, err, "migrate should succeed")

	// And: the data should still be accessible
	retrievedCmd, err := database.GetCommand(id)
	require.NoError(t, err, "should still be able to retrieve command")
	assert.Equal(t, "test command", retrievedCmd.CommandText)
}

// TestMigrate_FromLegacySchemaWithoutDuration tests migration from original schema
func TestMigrate_FromLegacySchemaWithoutDuration(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	// Given: a database with the legacy schema (no duration, no source columns)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	// Create database with old schema manually
	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	// Create old schema (without duration and source columns)
	oldSchema := `
		CREATE TABLE commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
			command_text TEXT NOT NULL,
			working_dir TEXT NOT NULL,
			git_repo TEXT,
			git_branch TEXT
		);
	`
	_, err = conn.Exec(oldSchema)
	require.NoError(t, err, "failed to create old schema")

	// Insert test data
	_, err = conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, command_text, working_dir, git_repo, git_branch)
		VALUES (1000, 0, 'ls -la', '/home/user', 'https://github.com/user/repo.git', 'main')
	`)
	require.NoError(t, err, "failed to insert test data")

	_, err = conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, command_text, working_dir, git_repo, git_branch)
		VALUES (2000, 1, 'false', '/home/user', NULL, NULL)
	`)
	require.NoError(t, err, "failed to insert test data")

	conn.Close()

	// When: we open the database (which triggers migration)
	database, err := New(dbPath)
	require.NoError(t, err, "failed to open and migrate database")
	defer database.Close()

	// Then: the table should have the new schema
	schema, err := database.GetTableSchema()
	require.NoError(t, err)
	assert.Len(t, schema, 11, "should have 11 columns")

	// Verify column names in correct order
	expectedColumns := []string{"id", "timestamp", "exit_status", "duration", "command_text", "working_dir", "git_repo", "git_branch", "source_app", "source_pid", "source_active"}
	for i, expected := range expectedColumns {
		assert.Equal(t, expected, schema[i]["name"], "column %d should be %s", i, expected)
	}

	// And: the old data should be preserved with default values
	count, err := database.CountCommands()
	require.NoError(t, err)
	assert.Equal(t, 2, count, "should have 2 commands")

	cmd1, err := database.GetCommand(1)
	require.NoError(t, err)
	assert.Equal(t, "ls -la", cmd1.CommandText)
	assert.Equal(t, int64(1000), cmd1.Timestamp)
	assert.Equal(t, 0, cmd1.ExitStatus)
	assert.NotNil(t, cmd1.Duration)
	assert.Equal(t, int64(0), *cmd1.Duration, "duration should default to 0")
	assert.Nil(t, cmd1.SourceApp, "source_app should be NULL")
	assert.Nil(t, cmd1.SourcePid, "source_pid should be NULL")
	assert.NotNil(t, cmd1.SourceActive)
	assert.True(t, *cmd1.SourceActive, "source_active should default to true")
	assert.NotNil(t, cmd1.GitRepo)
	assert.Equal(t, "https://github.com/user/repo.git", *cmd1.GitRepo)

	cmd2, err := database.GetCommand(2)
	require.NoError(t, err)
	assert.Equal(t, "false", cmd2.CommandText)
	assert.Equal(t, 1, cmd2.ExitStatus)
	assert.Nil(t, cmd2.GitRepo)
	assert.Nil(t, cmd2.GitBranch)
}

// TestMigrate_FromSchemaWithDurationButNoSource tests migration from intermediate schema
func TestMigrate_FromSchemaWithDurationButNoSource(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	// Given: a database with duration but no source columns
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	// Create schema with duration but no source columns
	schema := `
		CREATE TABLE commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
			duration INTEGER NOT NULL,
			command_text TEXT NOT NULL,
			working_dir TEXT NOT NULL,
			git_repo TEXT,
			git_branch TEXT
		);
	`
	_, err = conn.Exec(schema)
	require.NoError(t, err)

	// Insert test data with duration
	_, err = conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, duration, command_text, working_dir, git_repo, git_branch)
		VALUES (1000, 0, 1234, 'go test', '/home/user/project', 'https://github.com/user/project.git', 'feature')
	`)
	require.NoError(t, err)

	conn.Close()

	// When: we open the database (triggers migration)
	database, err := New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Then: source columns should be added with defaults
	cmd, err := database.GetCommand(1)
	require.NoError(t, err)
	assert.Equal(t, "go test", cmd.CommandText)
	assert.NotNil(t, cmd.Duration)
	assert.Equal(t, int64(1234), *cmd.Duration, "duration should be preserved")
	assert.Nil(t, cmd.SourceApp, "source_app should default to NULL")
	assert.Nil(t, cmd.SourcePid, "source_pid should default to NULL")
	assert.NotNil(t, cmd.SourceActive)
	assert.True(t, *cmd.SourceActive, "source_active should default to true")
}

// TestMigrate_FromSchemaWithWrongColumnOrder tests migration when columns are out of order
func TestMigrate_FromSchemaWithWrongColumnOrder(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	// Given: a database with all columns but duration in wrong position
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	// Create schema with duration at the end (wrong position)
	schema := `
		CREATE TABLE commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
			command_text TEXT NOT NULL,
			working_dir TEXT NOT NULL,
			git_repo TEXT,
			git_branch TEXT,
			source_app TEXT,
			source_pid INTEGER,
			source_active INTEGER DEFAULT 1,
			duration INTEGER NOT NULL
		);
	`
	_, err = conn.Exec(schema)
	require.NoError(t, err)

	// Insert test data
	_, err = conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, command_text, working_dir, git_repo, git_branch, source_app, source_pid, source_active, duration)
		VALUES (1000, 0, 'test', '/home/user', NULL, NULL, 'zsh', 12345, 1, 5678)
	`)
	require.NoError(t, err)

	conn.Close()

	// When: we open the database (triggers migration)
	database, err := New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Then: columns should be reordered correctly
	schema2, err := database.GetTableSchema()
	require.NoError(t, err)
	assert.Equal(t, "duration", schema2[3]["name"], "duration should be at position 3")

	// And: data should be preserved
	cmd, err := database.GetCommand(1)
	require.NoError(t, err)
	assert.Equal(t, "test", cmd.CommandText)
	assert.NotNil(t, cmd.Duration)
	assert.Equal(t, int64(5678), *cmd.Duration)
	assert.NotNil(t, cmd.SourceApp)
	assert.Equal(t, "zsh", *cmd.SourceApp)
	assert.NotNil(t, cmd.SourcePid)
	assert.Equal(t, int64(12345), *cmd.SourcePid)
}

// TestMigrate_PreservesMultipleRecords tests that migration preserves all data
func TestMigrate_PreservesMultipleRecords(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	// Given: a database with old schema and multiple records
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	// Create old schema
	oldSchema := `
		CREATE TABLE commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
			command_text TEXT NOT NULL,
			working_dir TEXT NOT NULL,
			git_repo TEXT,
			git_branch TEXT
		);
	`
	_, err = conn.Exec(oldSchema)
	require.NoError(t, err)

	// Insert multiple test records
	testCommands := []struct {
		timestamp int64
		status    int
		text      string
		dir       string
	}{
		{1000, 0, "ls -la", "/home/user"},
		{2000, 0, "git status", "/home/user/project"},
		{3000, 1, "false", "/home/user"},
		{4000, 0, "echo hello", "/tmp"},
		{5000, 0, "pwd", "/home/user/project"},
	}

	for _, cmd := range testCommands {
		_, err = conn.Exec(`
			INSERT INTO commands (timestamp, exit_status, command_text, working_dir)
			VALUES (?, ?, ?, ?)
		`, cmd.timestamp, cmd.status, cmd.text, cmd.dir)
		require.NoError(t, err)
	}

	conn.Close()

	// When: we open the database (triggers migration)
	database, err := New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Then: all records should be preserved
	count, err := database.CountCommands()
	require.NoError(t, err)
	assert.Equal(t, len(testCommands), count, "all records should be preserved")

	// Verify each record
	for i, expected := range testCommands {
		cmd, err := database.GetCommand(int64(i + 1))
		require.NoError(t, err, "should retrieve command %d", i+1)
		assert.Equal(t, expected.text, cmd.CommandText)
		assert.Equal(t, expected.timestamp, cmd.Timestamp)
		assert.Equal(t, expected.status, cmd.ExitStatus)
		assert.Equal(t, expected.dir, cmd.WorkingDir)
	}
}

// TestMigrate_CreatesAllIndexes tests that migration creates all required indexes
func TestMigrate_CreatesAllIndexes(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	// Given: a database with old schema
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	// Create old schema without indexes
	oldSchema := `
		CREATE TABLE commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
			command_text TEXT NOT NULL,
			working_dir TEXT NOT NULL,
			git_repo TEXT,
			git_branch TEXT
		);
	`
	_, err = conn.Exec(oldSchema)
	require.NoError(t, err)

	conn.Close()

	// When: we open the database (triggers migration)
	database, err := New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Then: all indexes should be created
	rows, err := database.conn.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='commands'")
	require.NoError(t, err)
	defer rows.Close()

	indexes := make(map[string]bool)
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		indexes[name] = true
	}

	// Verify all expected indexes exist
	expectedIndexes := []string{
		"idx_timestamp_desc",
		"idx_command_text_like",
		"idx_command_text",
		"idx_source_desc",
	}

	for _, idx := range expectedIndexes {
		assert.True(t, indexes[idx], "index %s should be created", idx)
	}
}

// TestMigrate_Idempotent tests that running migrate multiple times is safe
func TestMigrate_Idempotent(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	// Given: a database that has already been migrated
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	database, err := New(dbPath)
	require.NoError(t, err)

	// Insert test data
	cmd := models.NewCommand("test", "/home/user", 0)
	id, err := database.InsertCommand(cmd)
	require.NoError(t, err)

	// When: we run migrate again multiple times
	for i := 0; i < 3; i++ {
		err = database.migrate()
		require.NoError(t, err, "migrate should succeed on iteration %d", i)
	}

	// Then: data should still be intact
	retrievedCmd, err := database.GetCommand(id)
	require.NoError(t, err)
	assert.Equal(t, "test", retrievedCmd.CommandText)

	count, err := database.CountCommands()
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should still have exactly one command")

	database.Close()
}

// TestMigrate_PreservesNullValues tests that NULL values are preserved correctly
func TestMigrate_PreservesNullValues(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	// Given: a database with old schema and NULL values
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	oldSchema := `
		CREATE TABLE commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
			command_text TEXT NOT NULL,
			working_dir TEXT NOT NULL,
			git_repo TEXT,
			git_branch TEXT
		);
	`
	_, err = conn.Exec(oldSchema)
	require.NoError(t, err)

	// Insert with NULL git_repo and git_branch
	_, err = conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, command_text, working_dir, git_repo, git_branch)
		VALUES (1000, 0, 'test', '/home/user', NULL, NULL)
	`)
	require.NoError(t, err)

	// Insert with non-NULL values
	_, err = conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, command_text, working_dir, git_repo, git_branch)
		VALUES (2000, 0, 'test2', '/home/user', 'https://github.com/test/repo.git', 'main')
	`)
	require.NoError(t, err)

	conn.Close()

	// When: we open the database (triggers migration)
	database, err := New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Then: NULL values should remain NULL
	cmd1, err := database.GetCommand(1)
	require.NoError(t, err)
	assert.Nil(t, cmd1.GitRepo, "git_repo should be NULL")
	assert.Nil(t, cmd1.GitBranch, "git_branch should be NULL")

	// And: non-NULL values should be preserved
	cmd2, err := database.GetCommand(2)
	require.NoError(t, err)
	assert.NotNil(t, cmd2.GitRepo)
	assert.Equal(t, "https://github.com/test/repo.git", *cmd2.GitRepo)
	assert.NotNil(t, cmd2.GitBranch)
	assert.Equal(t, "main", *cmd2.GitBranch)
}

// TestMigrate_HandlesSpecialCharacters tests that special characters are preserved
func TestMigrate_HandlesSpecialCharacters(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	// Given: a database with special characters in commands
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	oldSchema := `
		CREATE TABLE commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
			command_text TEXT NOT NULL,
			working_dir TEXT NOT NULL,
			git_repo TEXT,
			git_branch TEXT
		);
	`
	_, err = conn.Exec(oldSchema)
	require.NoError(t, err)

	// Insert commands with special characters
	specialCommands := []string{
		`echo "Hello 'World'"`,
		`grep '\$HOME' file.txt`,
		`sed 's/\\/\\\\/g' input.txt`,
		`command with 中文`,
		`emoji test 🚀`,
	}

	for _, cmdText := range specialCommands {
		_, err = conn.Exec(`
			INSERT INTO commands (timestamp, exit_status, command_text, working_dir)
			VALUES (?, 0, ?, '/home/user')
		`, int64(1000), cmdText)
		require.NoError(t, err)
	}

	conn.Close()

	// When: we open the database (triggers migration)
	database, err := New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Then: all special characters should be preserved
	for i, expected := range specialCommands {
		cmd, err := database.GetCommand(int64(i + 1))
		require.NoError(t, err)
		assert.Equal(t, expected, cmd.CommandText, "special characters should be preserved for command %d", i+1)
	}
}

// TestMigrate_WithDurationNull tests migration handles NULL duration values
func TestMigrate_WithDurationNull(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	// Given: a database with duration column that has NULL values
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	schema := `
		CREATE TABLE commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
			duration INTEGER,
			command_text TEXT NOT NULL,
			working_dir TEXT NOT NULL,
			git_repo TEXT,
			git_branch TEXT
		);
	`
	_, err = conn.Exec(schema)
	require.NoError(t, err)

	// Insert with NULL duration
	_, err = conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, duration, command_text, working_dir)
		VALUES (1000, 0, NULL, 'test', '/home/user')
	`)
	require.NoError(t, err)

	conn.Close()

	// When: we open the database (triggers migration)
	database, err := New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Then: NULL duration should be converted to 0
	cmd, err := database.GetCommand(1)
	require.NoError(t, err)
	assert.NotNil(t, cmd.Duration)
	assert.Equal(t, int64(0), *cmd.Duration, "NULL duration should become 0")
}

// TestMigrate_MissingIndexes tests that migration runs when indexes are missing
func TestMigrate_MissingIndexes(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	// Given: a database with correct schema but missing indexes
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	// Create schema with correct columns but no indexes
	schema := `
		CREATE TABLE commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
			duration INTEGER NOT NULL,
			command_text TEXT NOT NULL,
			working_dir TEXT NOT NULL,
			git_repo TEXT,
			git_branch TEXT,
			source_app TEXT,
			source_pid INTEGER,
			source_active INTEGER DEFAULT 1
		);
	`
	_, err = conn.Exec(schema)
	require.NoError(t, err)

	// Insert test data
	_, err = conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, duration, command_text, working_dir, source_app, source_pid, source_active)
		VALUES (1000, 0, 0, 'test', '/home/user', 'zsh', 12345, 1)
	`)
	require.NoError(t, err)

	// Verify no indexes exist
	rows, err := conn.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='commands'")
	require.NoError(t, err)
	indexCount := 0
	for rows.Next() {
		indexCount++
	}
	rows.Close()
	assert.Equal(t, 0, indexCount, "should start with no indexes")

	conn.Close()

	// When: we open the database (triggers migration)
	database, err := New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Then: all indexes should be created
	rows, err = database.conn.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='commands'")
	require.NoError(t, err)
	defer rows.Close()

	indexes := make(map[string]bool)
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		indexes[name] = true
	}

	// Verify all required indexes exist
	expectedIndexes := []string{
		"idx_timestamp_desc",
		"idx_command_text_like",
		"idx_command_text",
		"idx_source_desc",
	}

	for _, idx := range expectedIndexes {
		assert.True(t, indexes[idx], "index %s should be created", idx)
	}

	// And: data should still be intact
	cmd, err := database.GetCommand(1)
	require.NoError(t, err)
	assert.Equal(t, "test", cmd.CommandText)
}

// TestMigrate_PartialIndexes tests that migration runs when some indexes are missing
func TestMigrate_PartialIndexes(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	// Given: a database with correct schema and only some indexes
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	// Create schema with correct columns
	schema := `
		CREATE TABLE commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
			duration INTEGER NOT NULL,
			command_text TEXT NOT NULL,
			working_dir TEXT NOT NULL,
			git_repo TEXT,
			git_branch TEXT,
			source_app TEXT,
			source_pid INTEGER,
			source_active INTEGER DEFAULT 1
		);
	`
	_, err = conn.Exec(schema)
	require.NoError(t, err)

	// Create only some indexes (missing idx_command_text and idx_source_desc)
	_, err = conn.Exec("CREATE INDEX idx_timestamp_desc ON commands (timestamp DESC)")
	require.NoError(t, err)

	_, err = conn.Exec("CREATE INDEX idx_command_text_like ON commands (command_text COLLATE NOCASE)")
	require.NoError(t, err)

	// Insert test data
	_, err = conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, duration, command_text, working_dir)
		VALUES (1000, 0, 0, 'test', '/home/user')
	`)
	require.NoError(t, err)

	conn.Close()

	// When: we open the database (triggers migration)
	database, err := New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Then: missing indexes should be created
	rows, err := database.conn.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='commands'")
	require.NoError(t, err)
	defer rows.Close()

	indexes := make(map[string]bool)
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		indexes[name] = true
	}

	// Verify all required indexes now exist
	assert.True(t, indexes["idx_timestamp_desc"], "existing index should remain")
	assert.True(t, indexes["idx_command_text_like"], "existing index should remain")
	assert.True(t, indexes["idx_command_text"], "missing index should be created")
	assert.True(t, indexes["idx_source_desc"], "missing index should be created")

	// And: data should still be intact
	count, err := database.CountCommands()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// TestMigrate_LargeDataset tests migration performance with many records
func TestMigrate_LargeDataset(t *testing.T) {
	if DbType() == "duckdb" {
		return
	}
	if testing.Short() {
		t.Skip("skipping large dataset test in short mode")
	}

	// Given: a database with many records
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "history.db")

	conn, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	oldSchema := `
		CREATE TABLE commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
			command_text TEXT NOT NULL,
			working_dir TEXT NOT NULL,
			git_repo TEXT,
			git_branch TEXT
		);
	`
	_, err = conn.Exec(oldSchema)
	require.NoError(t, err)

	// Insert 10,000 records
	tx, err := conn.Begin()
	require.NoError(t, err)

	stmt, err := tx.Prepare(`
		INSERT INTO commands (timestamp, exit_status, command_text, working_dir)
		VALUES (?, 0, ?, '/home/user')
	`)
	require.NoError(t, err)

	for i := 0; i < 10000; i++ {
		_, err = stmt.Exec(int64(i), "test command "+string(rune(i%26+'a')))
		require.NoError(t, err)
	}

	stmt.Close()
	err = tx.Commit()
	require.NoError(t, err)

	conn.Close()

	// When: we open the database (triggers migration)
	database, err := New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Then: all records should be preserved
	count, err := database.CountCommands()
	require.NoError(t, err)
	assert.Equal(t, 10000, count, "all 10,000 records should be preserved")

	// Spot check a few records
	cmd1, err := database.GetCommand(1)
	require.NoError(t, err)
	assert.NotNil(t, cmd1.Duration)
	assert.Equal(t, int64(0), *cmd1.Duration)

	cmd5000, err := database.GetCommand(5000)
	require.NoError(t, err)
	assert.Equal(t, int64(4999), cmd5000.Timestamp)
}
