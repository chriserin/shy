package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
			git_branch TEXT,
			source_app TEXT,
			source_pid INTEGER,
			source_active INTEGER DEFAULT 1
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
	// Retry on database locked errors since concurrent connections may race to enable WAL
	var walErr error
	for i := 0; i < 5; i++ {
		_, walErr = conn.Exec("PRAGMA journal_mode=WAL")
		if walErr == nil {
			break
		}
		// Check if it's a database locked error
		if strings.Contains(walErr.Error(), "database is locked") {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		// Other errors are not retryable
		break
	}
	if walErr != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", walErr)
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

	// Check if migration is needed
	needsMigration := false
	hasDuration := false
	hasSourceApp := false
	hasSourcePid := false
	hasSourceActive := false

	for _, col := range columns {
		switch col.name {
		case "duration":
			hasDuration = true
		case "source_app":
			hasSourceApp = true
		case "source_pid":
			hasSourcePid = true
		case "source_active":
			hasSourceActive = true
		}
	}

	// Expected columns: id, timestamp, exit_status, duration, command_text, working_dir, git_repo, git_branch, source_app, source_pid, source_active (11 total)
	if len(columns) < 11 {
		needsMigration = true
	} else if len(columns) >= 4 && columns[3].name != "duration" {
		// Check if duration is in correct position (column 3, 0-indexed)
		needsMigration = true
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
			git_branch TEXT,
			source_app TEXT,
			source_pid INTEGER,
			source_active INTEGER DEFAULT 1
		);
	`
	if _, err := db.conn.Exec(createNewTableSQL); err != nil {
		return fmt.Errorf("failed to create new table: %w", err)
	}

	// Copy data from old table to new table
	// Default values: duration=0, source_app=NULL, source_pid=NULL, source_active=1
	var copyDataSQL string
	if hasDuration && hasSourceApp && hasSourcePid && hasSourceActive {
		// All columns exist
		copyDataSQL = `
			INSERT INTO commands_new (id, timestamp, exit_status, duration, command_text, working_dir, git_repo, git_branch, source_app, source_pid, source_active)
			SELECT id, timestamp, exit_status, COALESCE(duration, 0), command_text, working_dir, git_repo, git_branch, source_app, source_pid, COALESCE(source_active, 1)
			FROM commands;
		`
	} else if hasDuration {
		// Duration exists, but source columns don't
		copyDataSQL = `
			INSERT INTO commands_new (id, timestamp, exit_status, duration, command_text, working_dir, git_repo, git_branch, source_app, source_pid, source_active)
			SELECT id, timestamp, exit_status, COALESCE(duration, 0), command_text, working_dir, git_repo, git_branch, NULL, NULL, 1
			FROM commands;
		`
	} else {
		// Duration doesn't exist
		copyDataSQL = `
			INSERT INTO commands_new (id, timestamp, exit_status, duration, command_text, working_dir, git_repo, git_branch, source_app, source_pid, source_active)
			SELECT id, timestamp, exit_status, 0, command_text, working_dir, git_repo, git_branch, NULL, NULL, 1
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

	// Convert source_active bool pointer to integer for SQLite
	var sourceActive interface{}
	if cmd.SourceActive != nil {
		if *cmd.SourceActive {
			sourceActive = 1
		} else {
			sourceActive = 0
		}
	} else {
		sourceActive = nil
	}

	result, err := db.conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration, source_app, source_pid, source_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cmd.Timestamp,
		cmd.ExitStatus,
		cmd.CommandText,
		cmd.WorkingDir,
		cmd.GitRepo,
		cmd.GitBranch,
		duration,
		cmd.SourceApp,
		cmd.SourcePid,
		sourceActive,
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
	var sourceActive *int64
	err := db.conn.QueryRow(`
		SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration, source_app, source_pid, source_active
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
		&cmd.SourceApp,
		&cmd.SourcePid,
		&sourceActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get command: %w", err)
	}

	// Convert source_active from integer to bool pointer
	if sourceActive != nil {
		active := *sourceActive != 0
		cmd.SourceActive = &active
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
		SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration, source_app, source_pid, source_active
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
		var sourceActive *int64
		if err := rows.Scan(
			&cmd.ID,
			&cmd.Timestamp,
			&cmd.ExitStatus,
			&cmd.CommandText,
			&cmd.WorkingDir,
			&cmd.GitRepo,
			&cmd.GitBranch,
			&cmd.Duration,
			&cmd.SourceApp,
			&cmd.SourcePid,
			&sourceActive,
		); err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		// Convert source_active from integer to bool pointer
		if sourceActive != nil {
			active := *sourceActive != 0
			cmd.SourceActive = &active
		}
		commands = append(commands, cmd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating commands: %w", err)
	}

	return commands, nil
}

// GetCommandsByRangeWithPattern retrieves commands by event ID range (inclusive) that match a pattern
// Returns commands ordered by ID ascending
// The pattern uses glob syntax (* for any chars, ? for single char) and is translated to SQL LIKE
func (db *DB) GetCommandsByRangeWithPattern(first, last int64, pattern string) ([]models.Command, error) {
	// Handle invalid range
	if first > last {
		return []models.Command{}, nil
	}

	query := `
		SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration, source_app, source_pid, source_active
		FROM commands
		WHERE id >= ? AND id <= ?
		AND command_text LIKE ? ESCAPE '\'
		ORDER BY id ASC`

	rows, err := db.conn.Query(query, first, last, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands by range with pattern: %w", err)
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		var cmd models.Command
		var sourceActive *int64
		if err := rows.Scan(
			&cmd.ID,
			&cmd.Timestamp,
			&cmd.ExitStatus,
			&cmd.CommandText,
			&cmd.WorkingDir,
			&cmd.GitRepo,
			&cmd.GitBranch,
			&cmd.Duration,
			&cmd.SourceApp,
			&cmd.SourcePid,
			&sourceActive,
		); err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		// Convert source_active from integer to bool pointer
		if sourceActive != nil {
			active := *sourceActive != 0
			cmd.SourceActive = &active
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

// GetCommandsByRangeInternal retrieves commands by event ID range (inclusive) filtered by session
// Only returns commands from the active session with the given PID
// Returns commands ordered by ID ascending
func (db *DB) GetCommandsByRangeInternal(first, last, sessionPid int64) ([]models.Command, error) {
	// Handle invalid range
	if first > last {
		return []models.Command{}, nil
	}

	query := `
		SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration, source_app, source_pid, source_active
		FROM commands
		WHERE id >= ? AND id <= ?
		AND source_pid = ?
		AND source_active = 1
		ORDER BY id ASC`

	rows, err := db.conn.Query(query, first, last, sessionPid)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands by range (internal): %w", err)
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		var cmd models.Command
		var sourceActive *int64
		if err := rows.Scan(
			&cmd.ID,
			&cmd.Timestamp,
			&cmd.ExitStatus,
			&cmd.CommandText,
			&cmd.WorkingDir,
			&cmd.GitRepo,
			&cmd.GitBranch,
			&cmd.Duration,
			&cmd.SourceApp,
			&cmd.SourcePid,
			&sourceActive,
		); err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		// Convert source_active from integer to bool pointer
		if sourceActive != nil {
			active := *sourceActive != 0
			cmd.SourceActive = &active
		}
		commands = append(commands, cmd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating commands: %w", err)
	}

	return commands, nil
}

// GetCommandsByRangeWithPatternInternal retrieves commands by event ID range (inclusive) that match a pattern
// and are from the active session with the given PID
// Returns commands ordered by ID ascending
// The pattern uses glob syntax (* for any chars, ? for single char) and is translated to SQL LIKE
func (db *DB) GetCommandsByRangeWithPatternInternal(first, last, sessionPid int64, pattern string) ([]models.Command, error) {
	// Handle invalid range
	if first > last {
		return []models.Command{}, nil
	}

	query := `
		SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration, source_app, source_pid, source_active
		FROM commands
		WHERE id >= ? AND id <= ?
		AND command_text LIKE ? ESCAPE '\'
		AND source_pid = ?
		AND source_active = 1
		ORDER BY id ASC`

	rows, err := db.conn.Query(query, first, last, pattern, sessionPid)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands by range with pattern (internal): %w", err)
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		var cmd models.Command
		var sourceActive *int64
		if err := rows.Scan(
			&cmd.ID,
			&cmd.Timestamp,
			&cmd.ExitStatus,
			&cmd.CommandText,
			&cmd.WorkingDir,
			&cmd.GitRepo,
			&cmd.GitBranch,
			&cmd.Duration,
			&cmd.SourceApp,
			&cmd.SourcePid,
			&sourceActive,
		); err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		// Convert source_active from integer to bool pointer
		if sourceActive != nil {
			active := *sourceActive != 0
			cmd.SourceActive = &active
		}
		commands = append(commands, cmd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating commands: %w", err)
	}

	return commands, nil
}

// CloseSession marks all active commands from a session as inactive
// Returns the number of commands updated
func (db *DB) CloseSession(sessionPid int64) (int64, error) {
	result, err := db.conn.Exec(`
		UPDATE commands
		SET source_active = 0
		WHERE source_pid = ? AND source_active = 1`,
		sessionPid,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to close session: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return count, nil
}

// LikeRecentOptions contains options for LikeRecent query
type LikeRecentOptions struct {
	Prefix     string
	Limit      int
	IncludeShy bool
	Exclude    string
	WorkingDir string
	SessionPID string
}

// LikeRecent finds commands matching a prefix with various filters
func (db *DB) LikeRecent(opts LikeRecentOptions) ([]string, error) {
	// Build SQL query
	query := `
		SELECT command_text
		FROM commands
		WHERE command_text LIKE ? || '%'
	`
	args := []interface{}{opts.Prefix}

	// Exclude shy commands by default
	if !opts.IncludeShy {
		query += ` AND command_text NOT LIKE 'shy %' AND command_text != 'shy'`
	}

	// Add exclude pattern filter
	if opts.Exclude != "" {
		query += ` AND command_text NOT GLOB ?`
		args = append(args, opts.Exclude)
	}

	// Add working directory filter
	if opts.WorkingDir != "" {
		query += ` AND working_dir = ?`
		args = append(args, opts.WorkingDir)
	}

	// Add session PID filter
	if opts.SessionPID != "" {
		query += ` AND source_pid = ?`
		args = append(args, opts.SessionPID)
	}

	// Order by timestamp descending and limit results
	query += ` ORDER BY timestamp DESC`
	if opts.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, opts.Limit)
	}

	// Execute query
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query commands: %w", err)
	}
	defer rows.Close()

	// Collect results
	var results []string
	for rows.Next() {
		var cmdText string
		if err := rows.Scan(&cmdText); err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		results = append(results, cmdText)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating commands: %w", err)
	}

	return results, nil
}

// LikeRecentAfterOptions contains options for LikeRecentAfter query
type LikeRecentAfterOptions struct {
	Prefix     string
	PrevCmd    string
	Limit      int
	IncludeShy bool
	Exclude    string
}

// LikeRecentAfter finds commands matching a prefix that came after a specific previous command
func (db *DB) LikeRecentAfter(opts LikeRecentAfterOptions) ([]string, error) {
	// Build SQL query with CTE to find matching commands
	query := `
		WITH recent_matches AS (
			SELECT c.id, c.command_text, c.timestamp
			FROM commands c
			WHERE c.command_text LIKE ? || '%'
	`
	args := []interface{}{opts.Prefix}

	// Exclude shy commands by default
	if !opts.IncludeShy {
		query += ` AND c.command_text NOT LIKE 'shy %' AND c.command_text != 'shy'`
	}

	// Add exclude pattern filter
	if opts.Exclude != "" {
		query += ` AND c.command_text NOT GLOB ?`
		args = append(args, opts.Exclude)
	}

	// Order by timestamp descending and limit to 200 recent matches for context search
	query += `
			ORDER BY c.timestamp DESC
			LIMIT 200
		)
		SELECT rm.command_text
		FROM recent_matches rm
		JOIN commands prev ON prev.id = rm.id - 1
		WHERE prev.command_text = ?
		ORDER BY rm.timestamp DESC
	`
	args = append(args, opts.PrevCmd)

	// Add limit if specified
	if opts.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, opts.Limit)
	}

	// Execute query
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query commands: %w", err)
	}
	defer rows.Close()

	// Collect results
	var results []string
	for rows.Next() {
		var cmdText string
		if err := rows.Scan(&cmdText); err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		results = append(results, cmdText)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating commands: %w", err)
	}

	return results, nil
}
