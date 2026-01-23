package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/chris/shy/internal/summary"
	"github.com/chris/shy/pkg/models"
)

// ZDB wraps the zombiezen SQLite database connection
type ZDB struct {
	conn *sqlite.Conn
	path string
}

// NewZ creates a new database connection using zombiezen.com/go/sqlite and initializes the schema
func NewZ(dbPath string) (*ZDB, error) {
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

	// Open connection
	conn, err := sqlite.OpenConn(dbPath, sqlite.OpenReadWrite|sqlite.OpenCreate|sqlite.OpenWAL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode and set busy timeout
	if err := sqlitex.ExecuteTransient(conn, "PRAGMA journal_mode=WAL", nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	if err := sqlitex.ExecuteTransient(conn, "PRAGMA busy_timeout=5000", nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Create table
	if err := sqlitex.ExecuteScript(conn, createTableSQL, nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	// Create ZDB instance
	zdb := &ZDB{
		conn: conn,
		path: dbPath,
	}

	// Run migrations
	if err := zdb.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return zdb, nil
}

// migrate runs database migrations for ZDB
func (zdb *ZDB) migrate() error {
	// Check current schema
	var columns []struct {
		cid  int
		name string
	}

	err := sqlitex.Execute(zdb.conn, "PRAGMA table_info(commands)", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			columns = append(columns, struct {
				cid  int
				name string
			}{
				cid:  stmt.ColumnInt(0),
				name: stmt.ColumnText(1),
			})
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}

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

	// Expected columns: 11 total
	if len(columns) < 11 {
		needsMigration = true
	} else if len(columns) >= 4 && columns[3].name != "duration" {
		needsMigration = true
	}

	if !needsMigration {
		return nil
	}

	// Perform table recreation migration with exclusive lock
	defer sqlitex.Save(zdb.conn)(&err)

	if err := sqlitex.ExecuteTransient(zdb.conn, "BEGIN EXCLUSIVE", nil); err != nil {
		return fmt.Errorf("failed to begin exclusive transaction: %w", err)
	}

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
	if err := sqlitex.ExecuteScript(zdb.conn, createNewTableSQL, nil); err != nil {
		return fmt.Errorf("failed to create new table: %w", err)
	}

	// Copy data from old table to new table
	var copyDataSQL string
	if hasDuration && hasSourceApp && hasSourcePid && hasSourceActive {
		copyDataSQL = `
			INSERT INTO commands_new (id, timestamp, exit_status, duration, command_text, working_dir, git_repo, git_branch, source_app, source_pid, source_active)
			SELECT id, timestamp, exit_status, COALESCE(duration, 0), command_text, working_dir, git_repo, git_branch, source_app, source_pid, COALESCE(source_active, 1)
			FROM commands;
		`
	} else if hasDuration {
		copyDataSQL = `
			INSERT INTO commands_new (id, timestamp, exit_status, duration, command_text, working_dir, git_repo, git_branch, source_app, source_pid, source_active)
			SELECT id, timestamp, exit_status, COALESCE(duration, 0), command_text, working_dir, git_repo, git_branch, NULL, NULL, 1
			FROM commands;
		`
	} else {
		copyDataSQL = `
			INSERT INTO commands_new (id, timestamp, exit_status, duration, command_text, working_dir, git_repo, git_branch, source_app, source_pid, source_active)
			SELECT id, timestamp, exit_status, 0, command_text, working_dir, git_repo, git_branch, NULL, NULL, 1
			FROM commands;
		`
	}
	if err := sqlitex.ExecuteScript(zdb.conn, copyDataSQL, nil); err != nil {
		return fmt.Errorf("failed to copy data to new table: %w", err)
	}

	// Drop old table
	if err := sqlitex.ExecuteTransient(zdb.conn, "DROP TABLE commands", nil); err != nil {
		return fmt.Errorf("failed to drop old table: %w", err)
	}

	// Rename new table to original name
	if err := sqlitex.ExecuteTransient(zdb.conn, "ALTER TABLE commands_new RENAME TO commands", nil); err != nil {
		return fmt.Errorf("failed to rename new table: %w", err)
	}

	// Create indexes
	if err := sqlitex.ExecuteTransient(zdb.conn, "CREATE INDEX IF NOT EXISTS idx_timestamp_desc ON commands (timestamp DESC)", nil); err != nil {
		return fmt.Errorf("failed to create timestamp index: %w", err)
	}
	if err := sqlitex.ExecuteTransient(zdb.conn, "CREATE INDEX IF NOT EXISTS idx_command_text_like ON commands (command_text COLLATE NOCASE)", nil); err != nil {
		return fmt.Errorf("failed to create command_text_like index: %w", err)
	}
	if err := sqlitex.ExecuteTransient(zdb.conn, "CREATE INDEX IF NOT EXISTS idx_command_text ON commands (command_text, id)", nil); err != nil {
		return fmt.Errorf("failed to create command_text composite index: %w", err)
	}
	if err := sqlitex.ExecuteTransient(zdb.conn, "CREATE INDEX IF NOT EXISTS idx_source_desc ON commands (source_pid, source_app, timestamp DESC) WHERE source_active = 1", nil); err != nil {
		return fmt.Errorf("failed to create source_desc partial index: %w", err)
	}

	// Commit transaction
	if err := sqlitex.ExecuteTransient(zdb.conn, "COMMIT", nil); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	return nil
}

// Close closes the database connection
func (zdb *ZDB) Close() error {
	return zdb.conn.Close()
}

// Path returns the database file path
func (zdb *ZDB) Path() string {
	return zdb.path
}

// InsertCommand inserts a command into the database
func (zdb *ZDB) InsertCommand(cmd *models.Command) (int64, error) {
	query := `
		INSERT INTO commands (
			timestamp, exit_status, duration, command_text, working_dir,
			git_repo, git_branch, source_app, source_pid, source_active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt := zdb.conn.Prep(query)
	stmt.BindInt64(1, cmd.Timestamp)
	stmt.BindInt64(2, int64(cmd.ExitStatus))
	if cmd.Duration != nil {
		stmt.BindInt64(3, *cmd.Duration)
	} else {
		stmt.BindInt64(3, 0)
	}
	stmt.BindText(4, cmd.CommandText)
	stmt.BindText(5, cmd.WorkingDir)

	if cmd.GitRepo != nil {
		stmt.BindText(6, *cmd.GitRepo)
	} else {
		stmt.BindNull(6)
	}
	if cmd.GitBranch != nil {
		stmt.BindText(7, *cmd.GitBranch)
	} else {
		stmt.BindNull(7)
	}
	if cmd.SourceApp != nil {
		stmt.BindText(8, *cmd.SourceApp)
	} else {
		stmt.BindNull(8)
	}
	if cmd.SourcePid != nil {
		stmt.BindInt64(9, *cmd.SourcePid)
	} else {
		stmt.BindNull(9)
	}
	if cmd.SourceActive != nil {
		if *cmd.SourceActive {
			stmt.BindInt64(10, 1)
		} else {
			stmt.BindInt64(10, 0)
		}
	} else {
		stmt.BindInt64(10, 1)
	}

	_, err := stmt.Step()
	if err != nil {
		stmt.Finalize()
		return 0, fmt.Errorf("failed to insert command: %w", err)
	}
	stmt.Finalize()

	return zdb.conn.LastInsertRowID(), nil
}

// GetCommand retrieves a command by ID
func (zdb *ZDB) GetCommand(id int64) (*models.Command, error) {
	query := `
		SELECT id, timestamp, exit_status, duration, command_text, working_dir,
		       git_repo, git_branch, source_app, source_pid, source_active
		FROM commands
		WHERE id = ?
	`

	stmt := zdb.conn.Prep(query)
	defer stmt.Finalize()
	stmt.BindInt64(1, id)

	hasRow, err := stmt.Step()
	if err != nil {
		return nil, fmt.Errorf("failed to query command: %w", err)
	}
	if !hasRow {
		return nil, fmt.Errorf("command not found")
	}

	return zdb.scanCommand(stmt), nil
}

// CountCommands returns the total number of commands
func (zdb *ZDB) CountCommands() (int, error) {
	stmt := zdb.conn.Prep("SELECT COUNT(*) FROM commands")
	defer stmt.Finalize()

	hasRow, err := stmt.Step()
	if err != nil {
		return 0, fmt.Errorf("failed to count commands: %w", err)
	}
	if !hasRow {
		return 0, nil
	}

	return stmt.ColumnInt(0), nil
}

// GetCommandsByDateRange retrieves commands within a timestamp range
func (zdb *ZDB) GetCommandsByDateRange(startTime, endTime int64, sourceApp *string) ([]models.Command, error) {
	query := `
		SELECT id, timestamp, exit_status, duration, command_text, working_dir,
		       git_repo, git_branch, source_app, source_pid, source_active
		FROM commands
		WHERE timestamp >= ? AND timestamp < ?
	`

	if sourceApp != nil {
		query += " AND source_app = ?"
	}

	query += " ORDER BY timestamp ASC"

	stmt := zdb.conn.Prep(query)
	defer stmt.Finalize()

	stmt.BindInt64(1, startTime)
	stmt.BindInt64(2, endTime)
	if sourceApp != nil {
		stmt.BindText(3, *sourceApp)
	}

	var commands []models.Command
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("failed to query commands: %w", err)
		}
		if !hasRow {
			break
		}
		commands = append(commands, *zdb.scanCommand(stmt))
	}

	return commands, nil
}

// TableExists checks if the commands table exists
func (zdb *ZDB) TableExists() (bool, error) {
	stmt := zdb.conn.Prep("SELECT name FROM sqlite_master WHERE type='table' AND name='commands'")
	defer stmt.Finalize()

	hasRow, err := stmt.Step()
	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}

	return hasRow, nil
}

// GetTableSchema retrieves the schema of the commands table
func (zdb *ZDB) GetTableSchema() ([]map[string]interface{}, error) {
	var schema []map[string]interface{}

	err := sqlitex.Execute(zdb.conn, "PRAGMA table_info(commands)", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			schema = append(schema, map[string]interface{}{
				"cid":        stmt.ColumnInt(0),
				"name":       stmt.ColumnText(1),
				"type":       stmt.ColumnText(2),
				"notnull":    stmt.ColumnInt(3),
				"dflt_value": stmt.ColumnText(4),
				"pk":         stmt.ColumnInt(5),
			})
			return nil
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get table schema: %w", err)
	}

	return schema, nil
}

// ListCommands retrieves recent commands with optional session filtering
func (zdb *ZDB) ListCommands(limit int, sourceApp string, sourcePid int64) ([]models.Command, error) {
	return zdb.ListCommandsInRange(0, 0, limit, sourceApp, sourcePid)
}

// ListCommandsInRange retrieves commands within a timestamp range
func (zdb *ZDB) ListCommandsInRange(startTime, endTime int64, limit int, sourceApp string, sourcePid int64) ([]models.Command, error) {
	var whereClause string
	var binds []interface{}

	// Build WHERE clause for time range
	var whereParts []string
	if startTime > 0 && endTime > 0 {
		whereParts = append(whereParts, "timestamp >= ? AND timestamp <= ?")
		binds = append(binds, startTime, endTime)
	} else if startTime > 0 {
		whereParts = append(whereParts, "timestamp >= ?")
		binds = append(binds, startTime)
	} else if endTime > 0 {
		whereParts = append(whereParts, "timestamp <= ?")
		binds = append(binds, endTime)
	}

	// Add session filter if provided
	if sourceApp != "" && sourcePid > 0 {
		whereParts = append(whereParts, "source_app = ? AND source_pid = ? AND source_active = 1")
		binds = append(binds, sourceApp, sourcePid)
	}

	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	var query string
	if limit > 0 {
		// Get the N most recent commands in the range, then order them oldest-to-newest
		query = fmt.Sprintf(`
			SELECT id, timestamp, exit_status, duration, command_text, working_dir,
			       git_repo, git_branch, source_app, source_pid, source_active
			FROM (
				SELECT id, timestamp, exit_status, duration, command_text, working_dir,
				       git_repo, git_branch, source_app, source_pid, source_active
				FROM commands
				%s
				ORDER BY timestamp DESC
				LIMIT ?
			)
			ORDER BY timestamp ASC`, whereClause)
		binds = append(binds, limit)
	} else {
		// Get all commands in the range
		query = fmt.Sprintf(`
			SELECT id, timestamp, exit_status, duration, command_text, working_dir,
			       git_repo, git_branch, source_app, source_pid, source_active
			FROM commands
			%s
			ORDER BY timestamp ASC`, whereClause)
	}

	stmt := zdb.conn.Prep(query)
	defer stmt.Finalize()

	for i, bind := range binds {
		switch v := bind.(type) {
		case int64:
			stmt.BindInt64(i+1, v)
		case int:
			stmt.BindInt64(i+1, int64(v))
		case string:
			stmt.BindText(i+1, v)
		}
	}

	var commands []models.Command
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("failed to query commands: %w", err)
		}
		if !hasRow {
			break
		}
		commands = append(commands, *zdb.scanCommand(stmt))
	}

	return commands, nil
}

// GetRecentCommandsWithoutConsecutiveDuplicates retrieves recent commands excluding consecutive duplicates
func (zdb *ZDB) GetRecentCommandsWithoutConsecutiveDuplicates(limit int, sourceApp string, sourcePid int64, workingDir string) ([]models.Command, error) {
	if sourceApp == "" || sourcePid == 0 {
		return []models.Command{}, fmt.Errorf("both sourceApp and sourcePid must be provided for session filtering")
	}

	fetchLimit := limit * 2

	// Session query
	sessionQuery := fmt.Sprintf(`
		WITH recent_subset AS (
			SELECT timestamp, command_text
			FROM commands
			WHERE source_app = ? AND source_pid = ? AND source_active = 1
			ORDER BY timestamp DESC
			LIMIT %d
		),
		deduped AS (
			SELECT
				command_text,
				LAG(command_text) OVER (ORDER BY timestamp DESC) AS prev_command_text
			FROM recent_subset
		)
		SELECT command_text
		FROM deduped
		WHERE command_text != prev_command_text OR prev_command_text IS NULL
		LIMIT %d`, fetchLimit, limit)

	stmt := zdb.conn.Prep(sessionQuery)
	defer stmt.Finalize()

	stmt.BindText(1, sourceApp)
	stmt.BindInt64(2, sourcePid)

	var sessionResults []models.Command
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("session query failed: %w", err)
		}
		if !hasRow {
			break
		}
		sessionResults = append(sessionResults, models.Command{
			CommandText: stmt.ColumnText(0),
		})
	}

	// If we have enough session results, return them
	if len(sessionResults) >= limit {
		return sessionResults, nil
	}

	// Need more results from directory query
	remaining := limit - len(sessionResults)
	directoryQuery := fmt.Sprintf(`
		WITH recent_subset AS (
			SELECT timestamp, command_text
			FROM commands
			WHERE working_dir = ?
			ORDER BY timestamp DESC
			LIMIT %d
		),
		deduped AS (
			SELECT
				command_text,
				LAG(command_text) OVER (ORDER BY timestamp DESC) AS prev_command_text
			FROM recent_subset
		)
		SELECT command_text
		FROM deduped
		WHERE command_text != prev_command_text OR prev_command_text IS NULL
		LIMIT %d`, fetchLimit, remaining)

	stmt = zdb.conn.Prep(directoryQuery)
	defer stmt.Finalize()

	stmt.BindText(1, workingDir)

	var dirResults []models.Command
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("directory query failed: %w", err)
		}
		if !hasRow {
			break
		}
		dirResults = append(dirResults, models.Command{
			CommandText: stmt.ColumnText(0),
		})
	}

	// Combine results
	return append(sessionResults, dirResults...), nil
}

// GetMostRecentEventID returns the ID of the most recent command
func (zdb *ZDB) GetMostRecentEventID() (int64, error) {
	stmt := zdb.conn.Prep("SELECT COALESCE(MAX(id), 0) FROM commands")
	defer stmt.Finalize()

	hasRow, err := stmt.Step()
	if err != nil {
		return 0, fmt.Errorf("failed to get most recent event ID: %w", err)
	}
	if !hasRow {
		return 0, nil
	}

	return stmt.ColumnInt64(0), nil
}

// GetCommandsByRange retrieves commands within an ID range with deduplication
func (zdb *ZDB) GetCommandsByRange(first, last int64) ([]models.Command, error) {
	// Determine order
	ascending := first <= last
	var minID, maxID int64
	if ascending {
		minID, maxID = first, last
	} else {
		minID, maxID = last, first
	}

	query := `
		SELECT id, timestamp, exit_status, duration, command_text, working_dir,
		       git_repo, git_branch, source_app, source_pid, source_active
		FROM commands
		WHERE id IN (
			SELECT max(id)
			FROM commands
			WHERE id >= ? AND id <= ?
			GROUP BY command_text
		)
		ORDER BY id
	`

	if !ascending {
		query += " DESC"
	}

	stmt := zdb.conn.Prep(query)
	defer stmt.Finalize()

	stmt.BindInt64(1, minID)
	stmt.BindInt64(2, maxID)

	var commands []models.Command
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("failed to query commands: %w", err)
		}
		if !hasRow {
			break
		}
		commands = append(commands, *zdb.scanCommand(stmt))
	}

	return commands, nil
}

// GetCommandsByRangeWithPattern retrieves commands within an ID range matching a pattern
func (zdb *ZDB) GetCommandsByRangeWithPattern(first, last int64, pattern string) ([]models.Command, error) {
	ascending := first <= last
	var minID, maxID int64
	if ascending {
		minID, maxID = first, last
	} else {
		minID, maxID = last, first
	}

	query := `
		SELECT id, timestamp, exit_status, duration, command_text, working_dir,
		       git_repo, git_branch, source_app, source_pid, source_active
		FROM commands
		WHERE id IN (
			SELECT max(id)
			FROM commands
			WHERE id >= ? AND id <= ?
			  AND command_text LIKE ?
			GROUP BY command_text
		)
		ORDER BY id
	`

	if !ascending {
		query += " DESC"
	}

	stmt := zdb.conn.Prep(query)
	defer stmt.Finalize()

	stmt.BindInt64(1, minID)
	stmt.BindInt64(2, maxID)
	stmt.BindText(3, pattern)

	var commands []models.Command
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("failed to query commands: %w", err)
		}
		if !hasRow {
			break
		}
		commands = append(commands, *zdb.scanCommand(stmt))
	}

	return commands, nil
}

// FindMostRecentMatching finds the most recent command ID matching a prefix
func (zdb *ZDB) FindMostRecentMatching(prefix string) (int64, error) {
	query := `
		SELECT id
		FROM commands
		WHERE command_text LIKE ?
		ORDER BY id DESC
		LIMIT 1
	`

	stmt := zdb.conn.Prep(query)
	defer stmt.Finalize()

	stmt.BindText(1, prefix+"%")

	hasRow, err := stmt.Step()
	if err != nil {
		return 0, fmt.Errorf("failed to find matching command: %w", err)
	}
	if !hasRow {
		return 0, fmt.Errorf("no matching command found")
	}

	return stmt.ColumnInt64(0), nil
}

// FindMostRecentMatchingBefore finds the most recent command ID matching a prefix before a given ID
func (zdb *ZDB) FindMostRecentMatchingBefore(prefix string, beforeID int64) (int64, error) {
	query := `
		SELECT id
		FROM commands
		WHERE command_text LIKE ? AND id < ?
		ORDER BY id DESC
		LIMIT 1
	`

	stmt := zdb.conn.Prep(query)
	defer stmt.Finalize()

	stmt.BindText(1, prefix+"%")
	stmt.BindInt64(2, beforeID)

	hasRow, err := stmt.Step()
	if err != nil {
		return 0, fmt.Errorf("failed to find matching command: %w", err)
	}
	if !hasRow {
		return 0, fmt.Errorf("no matching command found")
	}

	return stmt.ColumnInt64(0), nil
}

// GetCommandsByRangeInternal retrieves commands for a specific session with deduplication
func (zdb *ZDB) GetCommandsByRangeInternal(first, last, sessionPid int64) ([]models.Command, error) {
	ascending := first <= last
	var minID, maxID int64
	if ascending {
		minID, maxID = first, last
	} else {
		minID, maxID = last, first
	}

	query := `
		SELECT id, timestamp, exit_status, duration, command_text, working_dir,
		       git_repo, git_branch, source_app, source_pid, source_active
		FROM commands
		WHERE id IN (
			SELECT max(id)
			FROM commands
			WHERE id >= ? AND id <= ?
			  AND source_pid = ?
			  AND source_active = 1
			GROUP BY command_text
		)
		ORDER BY id
	`

	if !ascending {
		query += " DESC"
	}

	stmt := zdb.conn.Prep(query)
	defer stmt.Finalize()

	stmt.BindInt64(1, minID)
	stmt.BindInt64(2, maxID)
	stmt.BindInt64(3, sessionPid)

	var commands []models.Command
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("failed to query commands: %w", err)
		}
		if !hasRow {
			break
		}
		commands = append(commands, *zdb.scanCommand(stmt))
	}

	return commands, nil
}

// GetCommandsByRangeWithPatternInternal retrieves commands for a session matching a pattern
func (zdb *ZDB) GetCommandsByRangeWithPatternInternal(first, last, sessionPid int64, pattern string) ([]models.Command, error) {
	ascending := first <= last
	var minID, maxID int64
	if ascending {
		minID, maxID = first, last
	} else {
		minID, maxID = last, first
	}

	query := `
		SELECT id, timestamp, exit_status, duration, command_text, working_dir,
		       git_repo, git_branch, source_app, source_pid, source_active
		FROM commands
		WHERE id IN (
			SELECT max(id)
			FROM commands
			WHERE id >= ? AND id <= ?
			  AND source_pid = ?
			  AND source_active = 1
			  AND command_text LIKE ?
			GROUP BY command_text
		)
		ORDER BY id
	`

	if !ascending {
		query += " DESC"
	}

	stmt := zdb.conn.Prep(query)
	defer stmt.Finalize()

	stmt.BindInt64(1, minID)
	stmt.BindInt64(2, maxID)
	stmt.BindInt64(3, sessionPid)
	stmt.BindText(4, pattern)

	var commands []models.Command
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("failed to query commands: %w", err)
		}
		if !hasRow {
			break
		}
		commands = append(commands, *zdb.scanCommand(stmt))
	}

	return commands, nil
}

// CloseSession marks all commands from a session as inactive
func (zdb *ZDB) CloseSession(sessionPid int64) (int64, error) {
	stmt := zdb.conn.Prep("UPDATE commands SET source_active = 0 WHERE source_pid = ? AND source_active = 1")
	defer stmt.Finalize()

	stmt.BindInt64(1, sessionPid)

	if _, err := stmt.Step(); err != nil {
		return 0, fmt.Errorf("failed to close session: %w", err)
	}

	return int64(zdb.conn.Changes()), nil
}

// LikeRecent finds commands matching a prefix with various filters
func (zdb *ZDB) LikeRecent(opts LikeRecentOptions) ([]string, error) {
	// Build base WHERE clause for common filters
	baseWhere := "command_text LIKE ?"
	var baseArgs []interface{}
	baseArgs = append(baseArgs, opts.Prefix+"%")

	if !opts.IncludeShy {
		baseWhere += " AND command_text NOT LIKE 'shy %' AND command_text != 'shy'"
	}
	if opts.Exclude != "" {
		baseWhere += " AND command_text NOT GLOB ?"
		baseArgs = append(baseArgs, opts.Exclude)
	}

	limit := opts.Limit
	if limit == 0 {
		limit = 1
	}

	// Helper to execute query
	executeQuery := func(whereClause string, args []interface{}) ([]string, error) {
		query := fmt.Sprintf(`
			SELECT DISTINCT command_text
			FROM commands
			WHERE %s
			ORDER BY timestamp DESC
			LIMIT %d
		`, whereClause, limit)

		stmt := zdb.conn.Prep(query)
		defer stmt.Finalize()

		for i, arg := range args {
			switch v := arg.(type) {
			case string:
				stmt.BindText(i+1, v)
			case int64:
				stmt.BindInt64(i+1, v)
			}
		}

		var results []string
		for {
			hasRow, err := stmt.Step()
			if err != nil {
				return nil, err
			}
			if !hasRow {
				break
			}
			results = append(results, stmt.ColumnText(0))
		}

		return results, nil
	}

	// 1. Try session query first (if SourceApp and SourcePid provided)
	if opts.SourceApp != "" && opts.SourcePid > 0 {
		sessionWhere := baseWhere + " AND source_app = ? AND source_pid = ? AND source_active = 1"
		sessionArgs := append(append([]interface{}{}, baseArgs...), opts.SourceApp, opts.SourcePid)

		if opts.WorkingDir != "" {
			sessionWhere += " AND working_dir = ?"
			sessionArgs = append(sessionArgs, opts.WorkingDir)
		}

		results, err := executeQuery(sessionWhere, sessionArgs)
		if err != nil {
			return nil, err
		}
		if len(results) > 0 {
			return results, nil // Early return
		}
	}

	// 2. Try working directory query (only if no session filter)
	if opts.WorkingDir != "" && (opts.SourceApp == "" || opts.SourcePid == 0) {
		workingDirWhere := baseWhere + " AND working_dir = ?"
		workingDirArgs := append(append([]interface{}{}, baseArgs...), opts.WorkingDir)

		results, err := executeQuery(workingDirWhere, workingDirArgs)
		if err != nil {
			return nil, err
		}
		if len(results) > 0 {
			return results, nil // Early return
		}
	}

	// 3. Fall back to whole history query
	return executeQuery(baseWhere, baseArgs)
}

// LikeRecentAfter finds commands matching a prefix that came after a specific previous command
func (zdb *ZDB) LikeRecentAfter(opts LikeRecentAfterOptions) ([]string, error) {
	// Build SQL query with CTE to find matching commands
	query := `
		WITH matched_commands AS (
			SELECT
				c1.command_text as current_cmd,
				c2.command_text as prev_cmd
			FROM commands c1
			JOIN commands c2 ON c2.id = (
				SELECT id FROM commands
				WHERE id < c1.id
				ORDER BY id DESC
				LIMIT 1
			)
			WHERE c1.command_text LIKE ?
	`

	limit := opts.Limit
	if limit == 0 {
		limit = 1
	}

	var binds []interface{}
	binds = append(binds, opts.Prefix+"%")

	// Add filters
	if !opts.IncludeShy {
		query += " AND c1.command_text NOT LIKE 'shy %' AND c1.command_text != 'shy'"
	}
	if opts.Exclude != "" {
		query += " AND c1.command_text NOT GLOB ?"
		binds = append(binds, opts.Exclude)
	}

	query += fmt.Sprintf(`
		)
		SELECT DISTINCT current_cmd
		FROM matched_commands
		WHERE prev_cmd = ?
		LIMIT %d
	`, limit)

	binds = append(binds, opts.PrevCmd)

	stmt := zdb.conn.Prep(query)
	defer stmt.Finalize()

	for i, bind := range binds {
		switch v := bind.(type) {
		case string:
			stmt.BindText(i+1, v)
		case int64:
			stmt.BindInt64(i+1, v)
		}
	}

	var results []string
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("failed to query commands: %w", err)
		}
		if !hasRow {
			break
		}
		results = append(results, stmt.ColumnText(0))
	}

	return results, nil
}

// GetCommandsForFzf iterates over unique commands and calls a function for each
func (zdb *ZDB) GetCommandsForFzf(fn func(id int64, cmdText string) error) error {
	query := `
		SELECT id, command_text
		FROM commands
		WHERE id IN (
			SELECT max(id)
			FROM commands
			GROUP BY command_text
		)
		ORDER BY id DESC
	`

	err := sqlitex.Execute(zdb.conn, query, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			return fn(stmt.ColumnInt64(0), stmt.ColumnText(1))
		},
	})

	return err
}

// GetCommandWithContext retrieves a command with surrounding context
func (zdb *ZDB) GetCommandWithContext(id int64, contextSize int) ([]models.Command, *models.Command, []models.Command, error) {
	// Get the target command
	target, err := zdb.GetCommand(id)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get target command: %w", err)
	}

	// Get before context
	beforeQuery := `
		SELECT id, timestamp, exit_status, duration, command_text, working_dir,
		       git_repo, git_branch, source_app, source_pid, source_active
		FROM commands
		WHERE id < ?
		ORDER BY id DESC
		LIMIT ?
	`

	stmt := zdb.conn.Prep(beforeQuery)
	defer stmt.Finalize()

	stmt.BindInt64(1, id)
	stmt.BindInt64(2, int64(contextSize))

	var before []models.Command
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to query before context: %w", err)
		}
		if !hasRow {
			break
		}
		before = append(before, *zdb.scanCommand(stmt))
	}

	// Reverse before slice
	for i, j := 0, len(before)-1; i < j; i, j = i+1, j-1 {
		before[i], before[j] = before[j], before[i]
	}

	// Get after context
	afterQuery := `
		SELECT id, timestamp, exit_status, duration, command_text, working_dir,
		       git_repo, git_branch, source_app, source_pid, source_active
		FROM commands
		WHERE id > ?
		ORDER BY id ASC
		LIMIT ?
	`

	stmt = zdb.conn.Prep(afterQuery)
	defer stmt.Finalize()

	stmt.BindInt64(1, id)
	stmt.BindInt64(2, int64(contextSize))

	var after []models.Command
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to query after context: %w", err)
		}
		if !hasRow {
			break
		}
		after = append(after, *zdb.scanCommand(stmt))
	}

	return before, target, after, nil
}

// GetContextSummary retrieves context summaries for a time range
func (zdb *ZDB) GetContextSummary(startTime, endTime int64) ([]summary.ContextSummary, error) {
	query := `
		SELECT
			working_dir,
			git_branch,
			COUNT(*) as command_count,
			MIN(timestamp) as first_time,
			MAX(timestamp) as last_time
		FROM commands
		WHERE timestamp >= ? AND timestamp < ?
		GROUP BY working_dir, git_branch
		ORDER BY (MAX(timestamp) - MIN(timestamp)) DESC, command_count DESC
	`

	stmt := zdb.conn.Prep(query)
	defer stmt.Finalize()

	stmt.BindInt64(1, startTime)
	stmt.BindInt64(2, endTime)

	var summaries []summary.ContextSummary
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("failed to query context summary: %w", err)
		}
		if !hasRow {
			break
		}

		sum := summary.ContextSummary{
			WorkingDir:   stmt.ColumnText(0),
			CommandCount: stmt.ColumnInt(2),
			FirstTime:    stmt.ColumnInt64(3),
			LastTime:     stmt.ColumnInt64(4),
		}

		// Handle nullable git_branch
		if stmt.ColumnType(1) != sqlite.TypeNull {
			branch := stmt.ColumnText(1)
			sum.GitBranch = &branch
		}

		summaries = append(summaries, sum)
	}

	return summaries, nil
}

// scanCommand scans a row into a Command struct
func (zdb *ZDB) scanCommand(stmt *sqlite.Stmt) *models.Command {
	cmd := &models.Command{
		ID:          stmt.ColumnInt64(0),
		Timestamp:   stmt.ColumnInt64(1),
		ExitStatus:  int(stmt.ColumnInt64(2)),
		CommandText: stmt.ColumnText(4),
		WorkingDir:  stmt.ColumnText(5),
	}

	// Duration
	if stmt.ColumnType(3) != sqlite.TypeNull {
		duration := stmt.ColumnInt64(3)
		cmd.Duration = &duration
	}

	// GitRepo
	if stmt.ColumnType(6) != sqlite.TypeNull {
		repo := stmt.ColumnText(6)
		cmd.GitRepo = &repo
	}

	// GitBranch
	if stmt.ColumnType(7) != sqlite.TypeNull {
		branch := stmt.ColumnText(7)
		cmd.GitBranch = &branch
	}

	// SourceApp
	if stmt.ColumnType(8) != sqlite.TypeNull {
		app := stmt.ColumnText(8)
		cmd.SourceApp = &app
	}

	// SourcePid
	if stmt.ColumnType(9) != sqlite.TypeNull {
		pid := stmt.ColumnInt64(9)
		cmd.SourcePid = &pid
	}

	// SourceActive
	if stmt.ColumnType(10) != sqlite.TypeNull {
		active := stmt.ColumnInt64(10) == 1
		cmd.SourceActive = &active
	}

	return cmd
}
