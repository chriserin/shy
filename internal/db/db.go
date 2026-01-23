package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	_ "modernc.org/sqlite"

	"github.com/chris/shy/internal/summary"
	"github.com/chris/shy/pkg/models"
)

const (
	defaultDBPath  = "~/.local/share/shy/history.db"
	CreateTableSQL = `
		CREATE TABLE IF NOT EXISTS commands (
			id INTEGER PRIMARY KEY %s,
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

	dbType := DbType()

	_, err := os.Stat(dbPath)
	dbExists := err == nil

	// Open database connection
	conn, err := sql.Open(dbType, dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if !dbExists {
		autinc := "AUTOINCREMENT"
		// Create table
		if dbType == "duckdb" {
			seqSql := "CREATE SEQUENCE id_sequence START 1;"
			if _, err := conn.Exec(seqSql); err != nil {
				conn.Close()
				return nil, fmt.Errorf("failed to create table: %w", err)
			}
			autinc = "DEFAULT nextval('id_sequence')"
		}
		if _, err := conn.Exec(fmt.Sprintf(CreateTableSQL, autinc)); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Enable WAL mode for better concurrency
	// Retry on database locked errors since concurrent connections may race to enable WAL
	if dbType == "sqlite" {

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
	}

	// Run migrations
	db := &DB{
		conn: conn,
		path: dbPath,
	}
	if dbType == "sqlite" {
		if err := db.migrate(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}
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
	if DbType() == "duckdb" {
		return nil
	}
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

	// Check if all required indexes exist
	if !needsMigration {
		indexRows, err := db.conn.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='commands'")
		if err != nil {
			return fmt.Errorf("failed to get index info: %w", err)
		}

		existingIndexes := make(map[string]bool)
		for indexRows.Next() {
			var name string
			if err := indexRows.Scan(&name); err != nil {
				indexRows.Close()
				return fmt.Errorf("failed to scan index name: %w", err)
			}
			existingIndexes[name] = true
		}
		indexRows.Close()

		// Required indexes
		requiredIndexes := []string{
			"idx_timestamp_desc",
			"idx_command_text_like",
			"idx_command_text",
			"idx_source_desc",
		}

		for _, idx := range requiredIndexes {
			if !existingIndexes[idx] {
				needsMigration = true
				break
			}
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

	// Create indexes
	if _, err := db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_timestamp_desc ON commands (timestamp DESC)"); err != nil {
		return fmt.Errorf("failed to create timestamp index: %w", err)
	}
	if _, err := db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_command_text_like ON commands (command_text COLLATE NOCASE)"); err != nil {
		return fmt.Errorf("failed to create command_text_like index: %w", err)
	}
	if _, err := db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_command_text ON commands (command_text, id)"); err != nil {
		return fmt.Errorf("failed to create command_text composite index: %w", err)
	}
	if _, err := db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_source_desc ON commands (source_pid, source_app, timestamp DESC) WHERE source_active = 1"); err != nil {
		return fmt.Errorf("failed to create source_desc partial index: %w", err)
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

	if DbType() == "sqlite" {
		id, err := result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get last insert ID: %w", err)
		}
		return id, nil
	} else if DbType() == "duckdb" {
		var currval int64
		result := db.conn.QueryRow("select currval('id_sequence')")
		result.Scan(&currval)
		return currval, nil
	}

	return 0, fmt.Errorf("No return possible")
}

// GetCommand retrieves a command by ID
func (db *DB) GetCommand(id int64) (*models.Command, error) {
	cmd := &models.Command{}
	var sourceActive *int64
	err := db.conn.QueryRow(`
		SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration, source_app, source_pid, source_active
		FROM commands where id = ?`, id,
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

// GetCommandsByDateRange retrieves commands within a Unix timestamp range (inclusive start, exclusive end)
// Returns commands ordered by timestamp ascending
func (db *DB) GetCommandsByDateRange(startTime, endTime int64, sourceApp *string) ([]models.Command, error) {
	query := `
		SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration, source_app, source_pid, source_active
		FROM commands
		WHERE timestamp >= ? AND timestamp < ?
		AND source_app == ?
		ORDER BY timestamp ASC`

	rows, err := db.conn.Query(query, startTime, endTime, sourceApp)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands by date range: %w", err)
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
func (db *DB) GetTableSchema() ([]map[string]any, error) {
	rows, err := db.conn.Query("PRAGMA table_info(commands)")
	if err != nil {
		return nil, fmt.Errorf("failed to get table schema: %w", err)
	}
	defer rows.Close()

	var schema []map[string]any
	for rows.Next() {
		var name, colType string
		var placeholder any
		if err := rows.Scan(&placeholder, &name, &colType, &placeholder, &placeholder, &placeholder); err != nil {
			return nil, fmt.Errorf("failed to scan schema row: %w", err)
		}

		schema = append(schema, map[string]any{
			"name": name,
			"type": colType,
		})
	}

	return schema, nil
}

// ListCommands retrieves commands ordered by timestamp ascending (oldest first)
// When a limit is applied, it returns the N most recent commands, but still ordered oldest-to-newest
// If limit is 0, all commands are returned
// If sourceApp and sourcePid are provided, only active commands from that session are returned
func (db *DB) ListCommands(limit int, sourceApp string, sourcePid int64) ([]models.Command, error) {
	return db.ListCommandsInRange(0, 0, limit, sourceApp, sourcePid)
}

// ListCommandsInRange retrieves commands within a timestamp range, ordered by timestamp ascending
// If startTime is 0, no lower bound is applied
// If endTime is 0, no upper bound is applied
// If limit is 0, all matching commands are returned
// If sourceApp and sourcePid are provided, only active commands from that session are returned
func (db *DB) ListCommandsInRange(startTime, endTime int64, limit int, sourceApp string, sourcePid int64) ([]models.Command, error) {
	var query string
	var whereClauses []string

	// Build WHERE clause for time range
	if startTime > 0 && endTime > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp >= %d AND timestamp <= %d", startTime, endTime))
	} else if startTime > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp >= %d", startTime))
	} else if endTime > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp <= %d", endTime))
	}

	// Add session filter if provided
	if sourceApp != "" && sourcePid > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("source_app = '%s' AND source_pid = %d AND source_active = 1",
			strings.ReplaceAll(sourceApp, "'", "''"), sourcePid))
	}

	// Combine WHERE clauses
	var whereClause string
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
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
		// Get all commands in the range
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

// GetRecentCommandsWithoutConsecutiveDuplicates retrieves recent commands without consecutive duplicates
// Commands are ordered by timestamp descending (most recent first), then consecutive duplicates are removed
// If sourceApp and sourcePid are provided, only active commands from that session are returned
// If workingDir is also provided along with session filters, results are unioned with directory commands (session first, then directory)
// Returns up to 'limit' commands after deduplication
// Runs session and directory queries in parallel, only using directory results if session results < limit
// Only populates ID and CommandText fields in the returned Command structs
func (db *DB) GetRecentCommandsWithoutConsecutiveDuplicates(limit int, sourceApp string, sourcePid int64, workingDir string) ([]models.Command, error) {
	if sourceApp == "" || sourcePid == 0 {
		return []models.Command{}, fmt.Errorf("both sourceApp and sourcePid must be provided for session filtering")
	}

	// Calculate fetch limit: enough to handle duplicates but not too many
	// Use 10x multiplier with a minimum of 100 and maximum of 10,000
	fetchLimit := limit * 10
	if fetchLimit < 100 {
		fetchLimit = 100
	}
	if fetchLimit > 10000 {
		fetchLimit = 10000
	}

	// Session commands query
	sessionQuery := fmt.Sprintf(`
		WITH recent_subset AS (
			SELECT timestamp, command_text
			FROM commands
			WHERE source_app = '%s' AND source_pid = %d AND source_active = 1
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
		LIMIT %d`,
		strings.ReplaceAll(sourceApp, "'", "''"), sourcePid, fetchLimit, limit)

	// Execute session query
	type queryResult struct {
		commands []models.Command
		err      error
	}

	sessionChan := make(chan queryResult, 1)
	directoryChan := make(chan queryResult, 1)

	// Run session query
	go func() {
		rows, err := db.conn.Query(sessionQuery)
		if err != nil {
			sessionChan <- queryResult{nil, fmt.Errorf("failed to query session commands: %w", err)}
			return
		}
		defer rows.Close()

		var commands []models.Command
		for rows.Next() {
			var cmd models.Command
			if err := rows.Scan(&cmd.CommandText); err != nil {
				sessionChan <- queryResult{nil, fmt.Errorf("failed to scan session command: %w", err)}
				return
			}
			commands = append(commands, cmd)
		}

		if err := rows.Err(); err != nil {
			sessionChan <- queryResult{nil, fmt.Errorf("error iterating session commands: %w", err)}
			return
		}

		sessionChan <- queryResult{commands, nil}
	}()

	// Run directory query if workingDir is provided
	if workingDir != "" {
		directoryQuery := fmt.Sprintf(`
			WITH recent_subset AS (
				SELECT timestamp, command_text
				FROM commands
				WHERE working_dir = '%s'
				ORDER BY timestamp DESC, id DESC
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
			LIMIT %d`,
			strings.ReplaceAll(workingDir, "'", "''"), fetchLimit, limit)

		go func() {
			rows, err := db.conn.Query(directoryQuery)
			if err != nil {
				directoryChan <- queryResult{nil, fmt.Errorf("failed to query directory commands: %w", err)}
				return
			}
			defer rows.Close()

			var commands []models.Command
			for rows.Next() {
				var cmd models.Command
				if err := rows.Scan(&cmd.CommandText); err != nil {
					directoryChan <- queryResult{nil, fmt.Errorf("failed to scan directory command: %w", err)}
					return
				}
				commands = append(commands, cmd)
			}

			if err := rows.Err(); err != nil {
				directoryChan <- queryResult{nil, fmt.Errorf("error iterating directory commands: %w", err)}
				return
			}

			directoryChan <- queryResult{commands, nil}
		}()
	} else {
		// No directory query needed
		directoryChan <- queryResult{nil, nil}
	}

	// Wait for session results
	sessionResult := <-sessionChan
	if sessionResult.err != nil {
		return nil, sessionResult.err
	}

	// If session commands meet the limit, return them
	if len(sessionResult.commands) >= limit {
		return sessionResult.commands[:limit], nil
	}

	// Wait for directory results
	directoryResult := <-directoryChan
	if directoryResult.err != nil {
		return nil, directoryResult.err
	}

	// If no directory results, return session results
	if directoryResult.commands == nil {
		return sessionResult.commands, nil
	}

	// Combine results: session commands first, then directory commands up to limit
	combined := append([]models.Command{}, sessionResult.commands...)
	needed := limit - len(combined)
	if needed > 0 {
		if needed > len(directoryResult.commands) {
			needed = len(directoryResult.commands)
		}
		combined = append(combined, directoryResult.commands[:needed]...)
	}

	return combined, nil
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
		FROM commands where id in (select max(id) from commands where id >= ? AND id <= ? group by command_text)
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
		WHERE id IN (
			SELECT max(id)
			FROM commands
			WHERE id >= ? AND id <= ?
			AND command_text LIKE ? ESCAPE '\'
			GROUP BY command_text
		)
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
		WHERE id IN (
			SELECT max(id)
			FROM commands
			WHERE id >= ? AND id <= ?
			AND source_pid = ?
			AND source_active = 1
			GROUP BY command_text
		)
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
		WHERE id IN (
			SELECT max(id)
			FROM commands
			WHERE id >= ? AND id <= ?
			AND command_text LIKE ? ESCAPE '\'
			AND source_pid = ?
			AND source_active = 1
			GROUP BY command_text
		)
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
	IncludeShy bool
	Exclude    string
	WorkingDir string
	SourceApp  string
	SourcePid  int64
}

// LikeRecent finds commands matching a prefix with various filters
// Runs three parallel queries (session, working dir, whole history) and returns first non-empty result
// Query priority: session (with workingdir if provided) > working dir only > whole history
func (db *DB) LikeRecent(opts LikeRecentOptions) ([]string, error) {
	type queryResult struct {
		results []string
		err     error
	}

	// Build base WHERE clause for common filters (prefix, IncludeShy, Exclude)
	baseWhere := "command_text LIKE ?"
	baseArgs := []any{opts.Prefix + "%"}

	// Exclude shy commands by default
	if !opts.IncludeShy {
		baseWhere += " AND command_text NOT LIKE 'shy %' AND command_text != 'shy'"
	}

	// Create channels for results
	sessionChan := make(chan queryResult, 1)
	workingDirChan := make(chan queryResult, 1)
	historyChan := make(chan queryResult, 1)

	// Helper function to execute query
	executeQuery := func(whereClauses string, args []any, ch chan<- queryResult) {
		query := "SELECT command_text FROM commands WHERE " + whereClauses + " ORDER BY timestamp DESC LIMIT 1"

		rows, err := db.conn.Query(query, args...)
		if err != nil {
			ch <- queryResult{nil, fmt.Errorf("failed to query commands: %w", err)}
			return
		}
		defer rows.Close()

		var results []string
		for rows.Next() {
			var cmdText string
			if err := rows.Scan(&cmdText); err != nil {
				ch <- queryResult{nil, fmt.Errorf("failed to scan command: %w", err)}
				return
			}
			results = append(results, cmdText)
		}

		if err := rows.Err(); err != nil {
			ch <- queryResult{nil, fmt.Errorf("error iterating commands: %w", err)}
			return
		}

		ch <- queryResult{results, nil}
	}

	// 1. Session query (if SourceApp and SourcePid provided)
	// Includes working dir filter if also provided
	if opts.SourceApp != "" && opts.SourcePid > 0 {
		go func() {
			sessionWhere := baseWhere + " AND source_app = ? AND source_pid = ? AND source_active = 1"
			sessionArgs := append(append([]any{}, baseArgs...), opts.SourceApp, opts.SourcePid)

			// Add working dir to session query if provided
			if opts.WorkingDir != "" {
				sessionWhere += " AND working_dir = ?"
				sessionArgs = append(sessionArgs, opts.WorkingDir)
			}

			executeQuery(sessionWhere, sessionArgs, sessionChan)
		}()
	} else {
		sessionChan <- queryResult{nil, nil}
	}

	// 2. Working directory query (if WorkingDir provided and no session filter)
	// Only runs if session query didn't run
	if opts.WorkingDir != "" && (opts.SourceApp == "" || opts.SourcePid == 0) {
		go func() {
			workingDirWhere := baseWhere + " AND working_dir = ?"
			workingDirArgs := append(append([]any{}, baseArgs...), opts.WorkingDir)
			executeQuery(workingDirWhere, workingDirArgs, workingDirChan)
		}()
	} else {
		workingDirChan <- queryResult{nil, nil}
	}

	// 3. Whole history query (always run)
	go func() {
		executeQuery(baseWhere, baseArgs, historyChan)
	}()

	// Check results in order: session, working dir, history
	// Return first non-empty result
	sessionResult := <-sessionChan
	if sessionResult.err != nil {
		return nil, sessionResult.err
	}
	if len(sessionResult.results) > 0 {
		return sessionResult.results, nil
	}

	workingDirResult := <-workingDirChan
	if workingDirResult.err != nil {
		return nil, workingDirResult.err
	}
	if len(workingDirResult.results) > 0 {
		return workingDirResult.results, nil
	}

	historyResult := <-historyChan
	if historyResult.err != nil {
		return nil, historyResult.err
	}

	return historyResult.results, nil
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

// GetCommandsForFzf retrieves commands optimized for fzf integration
// Calls fn for each deduplicated entry in reverse chronological order (most recent first)
// Deduplicates by command_text, keeping only the most recent occurrence (max id)
func (db *DB) GetCommandsForFzf(fn func(id int64, cmdText string) error) error {
	// Use the same max(id) deduplication pattern as other functions in this file
	query := `
		SELECT id, command_text
		FROM commands
		WHERE id IN (
			SELECT max(id)
			FROM commands
			GROUP BY command_text
		)
		ORDER BY id DESC`

	rows, err := db.conn.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query commands for fzf: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var cmdText string
		if err := rows.Scan(&id, &cmdText); err != nil {
			return fmt.Errorf("failed to scan fzf entry: %w", err)
		}

		// Call the provided function for each entry
		if err := fn(id, cmdText); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating fzf entries: %w", err)
	}

	return nil
}

// GetCommandWithContext returns a command along with surrounding commands from the same session
// Returns (beforeCommands, targetCommand, afterCommands, error)
// beforeCommands are in chronological order (oldest first)
// afterCommands are in chronological order (oldest first)
func (db *DB) GetCommandWithContext(id int64, contextSize int) ([]models.Command, *models.Command, []models.Command, error) {
	// First, get the target command
	targetCmd, err := db.GetCommand(id)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get target command: %w", err)
	}

	// If there's no source_pid, we can't filter by session
	var beforeCommands []models.Command
	var afterCommands []models.Command

	if targetCmd.SourcePid == nil {
		// No session info, just get commands by ID
		beforeCommands, err = db.getCommandsByIDRange(id-int64(contextSize), id-1)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get before commands: %w", err)
		}

		afterCommands, err = db.getCommandsByIDRange(id+1, id+int64(contextSize))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get after commands: %w", err)
		}
	} else {
		// Get commands from the same session
		sessionPid := *targetCmd.SourcePid

		// Get commands before (same session, ID < target, ordered by ID DESC, limit contextSize)
		beforeQuery := `
			SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration, source_app, source_pid, source_active
			FROM commands
			WHERE id < ? AND source_pid = ?
			ORDER BY id DESC
			LIMIT ?`

		rows, err := db.conn.Query(beforeQuery, id, sessionPid, contextSize)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to query before commands: %w", err)
		}

		beforeCommands, err = db.scanCommands(rows)
		rows.Close()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to scan before commands: %w", err)
		}

		// Reverse beforeCommands to get chronological order (oldest first)
		for i, j := 0, len(beforeCommands)-1; i < j; i, j = i+1, j-1 {
			beforeCommands[i], beforeCommands[j] = beforeCommands[j], beforeCommands[i]
		}

		// Get commands after (same session, ID > target, ordered by ID ASC, limit contextSize)
		afterQuery := `
			SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration, source_app, source_pid, source_active
			FROM commands
			WHERE id > ? AND source_pid = ?
			ORDER BY id ASC
			LIMIT ?`

		rows, err = db.conn.Query(afterQuery, id, sessionPid, contextSize)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to query after commands: %w", err)
		}

		afterCommands, err = db.scanCommands(rows)
		rows.Close()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to scan after commands: %w", err)
		}
	}

	return beforeCommands, targetCmd, afterCommands, nil
}

// getCommandsByIDRange gets commands within an ID range (inclusive)
func (db *DB) getCommandsByIDRange(startID, endID int64) ([]models.Command, error) {
	if startID > endID {
		return []models.Command{}, nil
	}

	query := `
		SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration, source_app, source_pid, source_active
		FROM commands
		WHERE id >= ? AND id <= ?
		ORDER BY id ASC`

	rows, err := db.conn.Query(query, startID, endID)
	if err != nil {
		return nil, fmt.Errorf("failed to query commands by ID range: %w", err)
	}
	defer rows.Close()

	return db.scanCommands(rows)
}

// scanCommands is a helper to scan multiple command rows
func (db *DB) scanCommands(rows *sql.Rows) ([]models.Command, error) {
	var commands []models.Command

	for rows.Next() {
		var cmd models.Command
		var sourceActive *int64

		err := rows.Scan(
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

// GetContextSummary returns aggregated command summaries grouped by context
// A context is defined by (working_dir, git_branch)
func (db *DB) GetContextSummary(startTime, endTime int64) ([]summary.ContextSummary, error) {
	query := `
		SELECT
			working_dir,
			COALESCE(git_branch, ''),
			COUNT(*) as command_count,
			MIN(timestamp) as first_time,
			MAX(timestamp) as last_time
		FROM commands
		WHERE timestamp >= ? AND timestamp < ?
		GROUP BY working_dir, COALESCE(git_branch, '')
		ORDER BY (MAX(timestamp) - MIN(timestamp)) DESC, COUNT(*) DESC, working_dir ASC
	`

	rows, err := db.conn.Query(query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query context summary: %w", err)
	}
	defer rows.Close()

	var summaries []summary.ContextSummary

	for rows.Next() {
		var sum summary.ContextSummary
		var gitBranch sql.NullString

		err := rows.Scan(
			&sum.WorkingDir,
			&gitBranch,
			&sum.CommandCount,
			&sum.FirstTime,
			&sum.LastTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan context summary: %w", err)
		}

		// Convert NULL git_branch to nil pointer
		if gitBranch.Valid && gitBranch.String != "" {
			sum.GitBranch = &gitBranch.String
		}

		summaries = append(summaries, sum)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating context summaries: %w", err)
	}

	return summaries, nil
}

func DbType() string {
	if os.Getenv("SHY_DB_TYPE") == "duckdb" {
		return "duckdb"
	} else {
		return "sqlite"
	}
}
