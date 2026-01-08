package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/chris/shy/pkg/models"
)

const (
	defaultDBPath  = "~/.local/share/shy/history.db"
	createTableSQL = `
		CREATE TABLE IF NOT EXISTS commands (
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
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
	path string
}

// New creates a new database connection and initializes the schema
func New(dbPath string) (*DB, error) {
	// Expand tilde in path or use default
	if dbPath == "" || dbPath == defaultDBPath {
		// Use XDG_DATA_HOME if set, otherwise fallback to ~/.local/share
		dataDir := os.Getenv("XDG_DATA_HOME")
		if dataDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get user home directory: %w", err)
			}
			dataDir = filepath.Join(home, ".local/share")
		}
		dbPath = filepath.Join(dataDir, "shy/history.db")
	} else if dbPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		dbPath = filepath.Join(home, dbPath[1:])
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout for handling concurrent writes
	if _, err := conn.Exec("PRAGMA busy_timeout=5000"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Create table
	if _, err := conn.Exec(createTableSQL); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	// Run migrations
	db := &DB{
		conn: conn,
		path: dbPath,
	}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// migrate runs database migrations
//
// Migration Strategy: Table Recreation with Exclusive Lock
//
// This function uses a table recreation approach rather than ALTER TABLE because:
// 1. SQLite's ALTER TABLE is limited and doesn't support column reordering
// 2. Column order matters for storage efficiency (fixed-size before variable-length)
// 3. Table recreation allows complete schema control for optimal alignment
//
// Locking Strategy: BEGIN EXCLUSIVE
//
// We use an exclusive transaction lock rather than a deferred insert queue because:
// 1. Scale: Maximum ~1M rows (~200MB) migrates in < 1 second
// 2. Simplicity: No complex state management or race conditions
// 3. Reliability: busy_timeout=5000ms means concurrent operations automatically retry
// 4. Consistency: Guaranteed no data loss or split-brain scenarios
//
// During migration:
// - All writes are blocked and will wait (up to busy_timeout)
// - WAL mode allows reads to continue on existing data
// - Migration completes quickly, blocked operations succeed automatically
//
// Alternative approaches (deferred insert queues, migration state tables) add significant
// complexity and are only justified for multi-hour migrations or high-throughput systems.
func (db *DB) migrate() error {
	// Check current schema
	rows, err := db.conn.Query("PRAGMA table_info(commands)")
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}

	type columnInfo struct {
		cid  int
		name string
	}
	var columns []columnInfo
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan column info: %w", err)
		}
		columns = append(columns, columnInfo{cid: cid, name: name})
	}
	rows.Close()

	// Check if migration is needed and if duration column exists
	needsMigration := false
	hasDuration := false

	for _, col := range columns {
		if col.name == "duration" {
			hasDuration = true
			break
		}
	}

	if len(columns) < 8 {
		// Missing duration column
		needsMigration = true
	} else {
		// Check if duration is in correct position (column 3, 0-indexed)
		if len(columns) >= 4 && columns[3].name != "duration" {
			needsMigration = true
		}
	}

	if !needsMigration {
		return nil
	}

	// Perform table recreation migration with exclusive lock
	// Use BEGIN EXCLUSIVE to immediately lock the database and block all other writes
	if _, err := db.conn.Exec("BEGIN EXCLUSIVE"); err != nil {
		return fmt.Errorf("failed to begin exclusive transaction: %w", err)
	}

	// Create a pseudo-transaction wrapper for defer cleanup
	committed := false
	defer func() {
		if !committed {
			db.conn.Exec("ROLLBACK")
		}
	}()

	// Create new table with correct schema
	createNewTableSQL := `
		CREATE TABLE commands_new (
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
	if _, err := db.conn.Exec(createNewTableSQL); err != nil {
		return fmt.Errorf("failed to create new table: %w", err)
	}

	// Copy data from old table to new table (duration defaults to 0 for old records)
	var copyDataSQL string
	if hasDuration {
		// Duration column exists, use COALESCE
		copyDataSQL = `
			INSERT INTO commands_new (id, timestamp, exit_status, duration, command_text, working_dir, git_repo, git_branch)
			SELECT id, timestamp, exit_status, COALESCE(duration, 0), command_text, working_dir, git_repo, git_branch
			FROM commands;
		`
	} else {
		// Duration column doesn't exist, use constant 0
		copyDataSQL = `
			INSERT INTO commands_new (id, timestamp, exit_status, duration, command_text, working_dir, git_repo, git_branch)
			SELECT id, timestamp, exit_status, 0, command_text, working_dir, git_repo, git_branch
			FROM commands;
		`
	}
	if _, err := db.conn.Exec(copyDataSQL); err != nil {
		return fmt.Errorf("failed to copy data to new table: %w", err)
	}

	// Drop old table
	if _, err := db.conn.Exec("DROP TABLE commands"); err != nil {
		return fmt.Errorf("failed to drop old table: %w", err)
	}

	// Rename new table to original name
	if _, err := db.conn.Exec("ALTER TABLE commands_new RENAME TO commands"); err != nil {
		return fmt.Errorf("failed to rename new table: %w", err)
	}

	// Commit transaction
	if _, err := db.conn.Exec("COMMIT"); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}
	committed = true

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Path returns the database file path
func (db *DB) Path() string {
	return db.path
}

// InsertCommand inserts a new command into the database
func (db *DB) InsertCommand(cmd *models.Command) (int64, error) {
	// Convert nil duration to 0
	duration := int64(0)
	if cmd.Duration != nil {
		duration = *cmd.Duration
	}

	result, err := db.conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		cmd.Timestamp,
		cmd.ExitStatus,
		cmd.CommandText,
		cmd.WorkingDir,
		cmd.GitRepo,
		cmd.GitBranch,
		duration,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert command: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return id, nil
}

// GetCommand retrieves a command by ID
func (db *DB) GetCommand(id int64) (*models.Command, error) {
	cmd := &models.Command{}
	err := db.conn.QueryRow(`
		SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration
		FROM commands WHERE id = ?`,
		id,
	).Scan(
		&cmd.ID,
		&cmd.Timestamp,
		&cmd.ExitStatus,
		&cmd.CommandText,
		&cmd.WorkingDir,
		&cmd.GitRepo,
		&cmd.GitBranch,
		&cmd.Duration,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get command: %w", err)
	}

	return cmd, nil
}

// CountCommands returns the total number of commands in the database
func (db *DB) CountCommands() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM commands").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count commands: %w", err)
	}
	return count, nil
}

// TableExists checks if the commands table exists
func (db *DB) TableExists() (bool, error) {
	var name string
	err := db.conn.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name='commands'`).Scan(&name)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}
	return true, nil
}

// GetTableSchema returns the schema of the commands table
func (db *DB) GetTableSchema() ([]map[string]interface{}, error) {
	rows, err := db.conn.Query("PRAGMA table_info(commands)")
	if err != nil {
		return nil, fmt.Errorf("failed to get table schema: %w", err)
	}
	defer rows.Close()

	var schema []map[string]interface{}
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}

		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("failed to scan schema row: %w", err)
		}

		schema = append(schema, map[string]interface{}{
			"name": name,
			"type": colType,
		})
	}

	return schema, nil
}

// ListCommands retrieves commands ordered by timestamp ascending (oldest first)
// When a limit is applied, it returns the N most recent commands, but still ordered oldest-to-newest
// If limit is 0, all commands are returned
func (db *DB) ListCommands(limit int) ([]models.Command, error) {
	return db.ListCommandsInRange(0, 0, limit)
}

// ListCommandsInRange retrieves commands within a timestamp range, ordered by timestamp ascending
// If startTime is 0, no lower bound is applied
// If endTime is 0, no upper bound is applied
// If limit is 0, all matching commands are returned
func (db *DB) ListCommandsInRange(startTime, endTime int64, limit int) ([]models.Command, error) {
	var query string
	var whereClause string

	// Build WHERE clause for time range
	if startTime > 0 && endTime > 0 {
		whereClause = fmt.Sprintf("WHERE timestamp >= %d AND timestamp <= %d", startTime, endTime)
	} else if startTime > 0 {
		whereClause = fmt.Sprintf("WHERE timestamp >= %d", startTime)
	} else if endTime > 0 {
		whereClause = fmt.Sprintf("WHERE timestamp <= %d", endTime)
	}

	if limit > 0 {
		// Get the N most recent commands in the range, then order them oldest-to-newest
		query = fmt.Sprintf(`
			SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration
			FROM (
				SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration
				FROM commands
				%s
				ORDER BY timestamp DESC
				LIMIT %d
			)
			ORDER BY timestamp ASC`, whereClause, limit)
	} else {
		query = fmt.Sprintf(`
			SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration
			FROM commands
			%s
			ORDER BY timestamp ASC`, whereClause)
	}

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list commands: %w", err)
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		var cmd models.Command
		if err := rows.Scan(
			&cmd.ID,
			&cmd.Timestamp,
			&cmd.ExitStatus,
			&cmd.CommandText,
			&cmd.WorkingDir,
			&cmd.GitRepo,
			&cmd.GitBranch,
			&cmd.Duration,
		); err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		commands = append(commands, cmd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating commands: %w", err)
	}

	return commands, nil
}

// GetMostRecentEventID returns the ID of the most recent command
// Returns 0 if no commands exist
func (db *DB) GetMostRecentEventID() (int64, error) {
	var id sql.NullInt64
	err := db.conn.QueryRow("SELECT MAX(id) FROM commands").Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to get most recent event ID: %w", err)
	}
	if !id.Valid {
		// No commands in database
		return 0, nil
	}
	return id.Int64, nil
}

// GetCommandsByRange retrieves commands by event ID range (inclusive)
// Returns commands ordered by ID ascending
func (db *DB) GetCommandsByRange(first, last int64) ([]models.Command, error) {
	// Handle invalid range
	if first > last {
		return []models.Command{}, nil
	}

	query := `
		SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration
		FROM commands
		WHERE id >= ? AND id <= ?
		ORDER BY id ASC`

	rows, err := db.conn.Query(query, first, last)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands by range: %w", err)
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		var cmd models.Command
		if err := rows.Scan(
			&cmd.ID,
			&cmd.Timestamp,
			&cmd.ExitStatus,
			&cmd.CommandText,
			&cmd.WorkingDir,
			&cmd.GitRepo,
			&cmd.GitBranch,
			&cmd.Duration,
		); err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		commands = append(commands, cmd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating commands: %w", err)
	}

	return commands, nil
}

// FindMostRecentMatching finds the most recent command that starts with the given prefix
// Returns the event ID, or 0 if not found
func (db *DB) FindMostRecentMatching(prefix string) (int64, error) {
	var id int64
	err := db.conn.QueryRow(`
		SELECT id FROM commands
		WHERE command_text LIKE ?
		ORDER BY id DESC
		LIMIT 1`,
		prefix+"%",
	).Scan(&id)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to find matching command: %w", err)
	}

	return id, nil
}

// FindMostRecentMatchingBefore finds the most recent command that starts with the given prefix
// and has an ID <= beforeID
// Returns the event ID, or 0 if not found
func (db *DB) FindMostRecentMatchingBefore(prefix string, beforeID int64) (int64, error) {
	var id int64
	err := db.conn.QueryRow(`
		SELECT id FROM commands
		WHERE command_text LIKE ? AND id <= ?
		ORDER BY id DESC
		LIMIT 1`,
		prefix+"%",
		beforeID,
	).Scan(&id)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to find matching command: %w", err)
	}

	return id, nil
}
