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
	defaultDBPath = "~/.local/share/shy/history.db"

	CreateWorkingDirsTableSQL = `
		CREATE TABLE IF NOT EXISTS working_dirs (
			id INTEGER PRIMARY KEY %s,
			path TEXT NOT NULL UNIQUE
		);
	`

	CreateGitContextsTableSQL = `
		CREATE TABLE IF NOT EXISTS git_contexts (
			id INTEGER PRIMARY KEY %s,
			repo TEXT,
			branch TEXT,
			UNIQUE(repo, branch)
		);
	`

	CreateSourcesTableSQL = `
		CREATE TABLE IF NOT EXISTS sources (
			id INTEGER PRIMARY KEY %s,
			app TEXT NOT NULL,
			pid INTEGER NOT NULL,
			active INTEGER DEFAULT 1,
			UNIQUE(app, pid, active)
		);
	`

	CreateTableSQL = `
		CREATE TABLE IF NOT EXISTS commands (
			id INTEGER PRIMARY KEY %s,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
			duration INTEGER NOT NULL,
			command_text TEXT NOT NULL,
			working_dir_id INTEGER NOT NULL REFERENCES working_dirs(id),
			git_context_id INTEGER REFERENCES git_contexts(id),
			source_id INTEGER REFERENCES sources(id)
		);
	`

	// CreateIndexesSQL creates performance indexes for common query patterns
	CreateIndexesSQL = `
		-- Index for session lookup (source_id + timestamp DESC)
		CREATE INDEX IF NOT EXISTS idx_source_timestamp ON commands (source_id, timestamp DESC);

		-- Index for working_dir lookup (working_dir_id + timestamp DESC)
		CREATE INDEX IF NOT EXISTS idx_working_dir_timestamp ON commands (working_dir_id, timestamp DESC);

		-- Index for full history (timestamp DESC)
		CREATE INDEX IF NOT EXISTS idx_timestamp_desc ON commands (timestamp DESC);

		-- Index for source lookup query
		CREATE INDEX IF NOT EXISTS idx_sources_app_pid_active ON sources (app, pid, active);

		-- Index for working_dir lookup query
		CREATE INDEX IF NOT EXISTS idx_working_dirs_path ON working_dirs (path);

		-- Index for GetCommandsForFzf deduplication (GROUP BY command_text, max(id))
		CREATE INDEX IF NOT EXISTS idx_command_text_id ON commands (command_text, id DESC);
	`
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
	path string
}

// Options configures database connection behavior
type Options struct {
	// ReadOnly opens the database without creating tables.
	// Use this for existing databases to avoid write locks.
	ReadOnly bool
}

// New creates a new database connection and initializes the schema
func New(dbPath string) (*DB, error) {
	return NewWithOptions(dbPath, Options{})
}

// NewWithOptions creates a new database connection with configurable options
func NewWithOptions(dbPath string, opts Options) (*DB, error) {
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

	// Open database connection
	conn, err := sql.Open(dbType, dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables unless ReadOnly mode
	if !opts.ReadOnly {
		autinc := "AUTOINCREMENT"
		if dbType == "duckdb" {
			// Check if sequence already exists
			var seqExists int
			conn.QueryRow("SELECT COUNT(*) FROM duckdb_sequences() WHERE sequence_name = 'id_sequence'").Scan(&seqExists)
			if seqExists == 0 {
				seqSql := "CREATE SEQUENCE id_sequence START 1;"
				if _, err := conn.Exec(seqSql); err != nil {
					conn.Close()
					return nil, fmt.Errorf("failed to create sequence: %w", err)
				}
			}
			autinc = "DEFAULT nextval('id_sequence')"
		}
		// Create lookup tables first (commands table has foreign keys to them)
		if _, err := conn.Exec(fmt.Sprintf(CreateWorkingDirsTableSQL, autinc)); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create working_dirs table: %w", err)
		}
		if _, err := conn.Exec(fmt.Sprintf(CreateGitContextsTableSQL, autinc)); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create git_contexts table: %w", err)
		}
		if _, err := conn.Exec(fmt.Sprintf(CreateSourcesTableSQL, autinc)); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create sources table: %w", err)
		}
		if _, err := conn.Exec(fmt.Sprintf(CreateTableSQL, autinc)); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create commands table: %w", err)
		}
		if _, err := conn.Exec(CreateIndexesSQL); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create indexes: %w", err)
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

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Path returns the database file path
func (db *DB) Path() string {
	return db.path
}

// getOrCreateWorkingDir returns the ID for a working directory, creating it if needed
func (db *DB) getOrCreateWorkingDir(path string) (int64, error) {
	// Try to get existing
	var id int64
	err := db.conn.QueryRow("SELECT id FROM working_dirs WHERE path = ?", path).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query working_dir: %w", err)
	}

	// Insert new
	result, err := db.conn.Exec("INSERT INTO working_dirs (path) VALUES (?)", path)
	if err != nil {
		// Handle race condition - another connection may have inserted
		err2 := db.conn.QueryRow("SELECT id FROM working_dirs WHERE path = ?", path).Scan(&id)
		if err2 == nil {
			return id, nil
		}
		return 0, fmt.Errorf("failed to insert working_dir: %w", err)
	}

	return result.LastInsertId()
}

// getOrCreateGitContext returns the ID for a git context, creating it if needed
// Returns nil if both repo and branch are nil
func (db *DB) getOrCreateGitContext(repo, branch *string) (*int64, error) {
	if repo == nil && branch == nil {
		return nil, nil
	}

	// Try to get existing
	var id int64
	err := db.conn.QueryRow(
		"SELECT id FROM git_contexts WHERE repo IS ? AND branch IS ?",
		repo, branch,
	).Scan(&id)
	if err == nil {
		return &id, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query git_context: %w", err)
	}

	// Insert new
	result, err := db.conn.Exec(
		"INSERT INTO git_contexts (repo, branch) VALUES (?, ?)",
		repo, branch,
	)
	if err != nil {
		// Handle race condition
		err2 := db.conn.QueryRow(
			"SELECT id FROM git_contexts WHERE repo IS ? AND branch IS ?",
			repo, branch,
		).Scan(&id)
		if err2 == nil {
			return &id, nil
		}
		return nil, fmt.Errorf("failed to insert git_context: %w", err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// getOrCreateSource returns the ID for a source, creating it if needed
// Returns nil if both app and pid are nil
func (db *DB) getOrCreateSource(app *string, pid *int64, active *bool) (*int64, error) {
	if app == nil || pid == nil {
		return nil, nil
	}

	// Convert active bool to int
	activeInt := 1
	if active != nil && !*active {
		activeInt = 0
	}

	// Try to get existing
	var id int64
	err := db.conn.QueryRow(
		"SELECT id FROM sources WHERE app = ? AND pid = ? AND active = ?",
		*app, *pid, activeInt,
	).Scan(&id)
	if err == nil {
		return &id, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query source: %w", err)
	}

	// Insert new
	result, err := db.conn.Exec(
		"INSERT INTO sources (app, pid, active) VALUES (?, ?, ?)",
		*app, *pid, activeInt,
	)
	if err != nil {
		// Handle race condition - another connection may have inserted
		err2 := db.conn.QueryRow(
			"SELECT id FROM sources WHERE app = ? AND pid = ? AND active = ?",
			*app, *pid, activeInt,
		).Scan(&id)
		if err2 == nil {
			return &id, nil
		}
		return nil, fmt.Errorf("failed to insert source: %w", err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// InsertCommand inserts a new command into the database
func (db *DB) InsertCommand(cmd *models.Command) (int64, error) {
	// Convert nil duration to 0
	duration := int64(0)
	if cmd.Duration != nil {
		duration = *cmd.Duration
	}

	// Get or create lookup table records
	workingDirID, err := db.getOrCreateWorkingDir(cmd.WorkingDir)
	if err != nil {
		return 0, fmt.Errorf("failed to get working_dir_id: %w", err)
	}

	gitContextID, err := db.getOrCreateGitContext(cmd.GitRepo, cmd.GitBranch)
	if err != nil {
		return 0, fmt.Errorf("failed to get git_context_id: %w", err)
	}

	sourceID, err := db.getOrCreateSource(cmd.SourceApp, cmd.SourcePid, cmd.SourceActive)
	if err != nil {
		return 0, fmt.Errorf("failed to get source_id: %w", err)
	}

	result, err := db.conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, duration, command_text, working_dir_id, git_context_id, source_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		cmd.Timestamp,
		cmd.ExitStatus,
		duration,
		cmd.CommandText,
		workingDirID,
		gitContextID,
		sourceID,
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

// commandSelectColumns is the common SELECT clause for denormalized command queries
const commandSelectColumns = `
	c.id, c.timestamp, c.exit_status, c.duration, c.command_text,
	w.path,
	g.repo, g.branch,
	s.app, s.pid, s.active
`

// commandFromJoins is the common FROM/JOIN clause for denormalized command queries
const commandFromJoins = `
	FROM commands c
	JOIN working_dirs w ON c.working_dir_id = w.id
	LEFT JOIN git_contexts g ON c.git_context_id = g.id
	LEFT JOIN sources s ON c.source_id = s.id
`

// scanCommand scans a row into a Command struct (used with commandSelectColumns)
func scanCommand(scanner interface{ Scan(...any) error }) (*models.Command, error) {
	cmd := &models.Command{}
	var sourceActive *int64
	err := scanner.Scan(
		&cmd.ID,
		&cmd.Timestamp,
		&cmd.ExitStatus,
		&cmd.Duration,
		&cmd.CommandText,
		&cmd.WorkingDir,
		&cmd.GitRepo,
		&cmd.GitBranch,
		&cmd.SourceApp,
		&cmd.SourcePid,
		&sourceActive,
	)
	if err != nil {
		return nil, err
	}

	// Convert source_active from integer to bool pointer
	if sourceActive != nil {
		active := *sourceActive != 0
		cmd.SourceActive = &active
	}

	return cmd, nil
}

// GetCommand retrieves a command by ID
func (db *DB) GetCommand(id int64) (*models.Command, error) {
	query := "SELECT " + commandSelectColumns + commandFromJoins + " WHERE c.id = ?"
	cmd, err := scanCommand(db.conn.QueryRow(query, id))
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

// GetCommandsByDateRange retrieves commands within a Unix timestamp range (inclusive start, exclusive end)
// Returns commands ordered by timestamp ascending
func (db *DB) GetCommandsByDateRange(startTime, endTime int64, sourceApp *string) ([]models.Command, error) {
	var query string
	var args []any

	if sourceApp != nil {
		query = "SELECT " + commandSelectColumns + commandFromJoins + `
			WHERE c.timestamp >= ? AND c.timestamp < ?
			AND s.app = ?
			ORDER BY c.timestamp ASC`
		args = []any{startTime, endTime, *sourceApp}
	} else {
		query = "SELECT " + commandSelectColumns + commandFromJoins + `
			WHERE c.timestamp >= ? AND c.timestamp < ?
			ORDER BY c.timestamp ASC`
		args = []any{startTime, endTime}
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands by date range: %w", err)
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		cmd, err := scanCommand(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		commands = append(commands, *cmd)
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
		whereClauses = append(whereClauses, fmt.Sprintf("c.timestamp >= %d AND c.timestamp <= %d", startTime, endTime))
	} else if startTime > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("c.timestamp >= %d", startTime))
	} else if endTime > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("c.timestamp <= %d", endTime))
	}

	// Add session filter if provided
	if sourceApp != "" && sourcePid > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("s.app = '%s' AND s.pid = %d AND s.active = 1",
			strings.ReplaceAll(sourceApp, "'", "''"), sourcePid))
	}

	// Combine WHERE clauses
	var whereClause string
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	selectCols := commandSelectColumns
	fromJoins := commandFromJoins

	if limit > 0 {
		// Get the N most recent commands in the range, then order them oldest-to-newest
		query = fmt.Sprintf(`
			SELECT * FROM (
				SELECT %s %s
				%s
				ORDER BY c.timestamp DESC
				LIMIT %d
			)
			ORDER BY timestamp ASC`, selectCols, fromJoins, whereClause, limit)
	} else {
		// Get all commands in the range
		query = fmt.Sprintf(`
			SELECT %s %s
			%s
			ORDER BY c.timestamp ASC`, selectCols, fromJoins, whereClause)
	}

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list commands: %w", err)
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		cmd, err := scanCommand(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		commands = append(commands, *cmd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating commands: %w", err)
	}

	return commands, nil
}

// GetRecentCommandsWithoutConsecutiveDuplicates retrieves recent commands without consecutive duplicates
// Commands are gathered from session, working_dir, and full history contexts, sorted by context priority
// (session=1, working_dir=2, history=3) then by timestamp DESC, and consecutive duplicates are removed using LAG
// Returns up to 'limit' commands after deduplication
// Only populates CommandText field in the returned Command structs
func (db *DB) GetRecentCommandsWithoutConsecutiveDuplicates(limit int, sourceApp string, sourcePid int64, workingDir string) ([]models.Command, error) {
	if sourceApp == "" || sourcePid == 0 {
		return []models.Command{}, fmt.Errorf("both sourceApp and sourcePid must be provided for session filtering")
	}

	// Calculate fetch limit per priority bucket
	// Use 3x multiplier with a minimum of 50 to handle duplicates
	bucketLimit := limit * 3
	if bucketLimit < 50 {
		bucketLimit = 50
	}

	// Look up source ID for this session (single record)
	var sourceID sql.NullInt64
	err := db.conn.QueryRow(
		"SELECT id FROM sources WHERE app = ? AND pid = ? AND active = 1 ORDER BY id DESC LIMIT 1",
		sourceApp, sourcePid,
	).Scan(&sourceID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query source ID: %w", err)
	}

	// Look up working_dir ID
	var workingDirID sql.NullInt64
	err = db.conn.QueryRow(
		"SELECT id FROM working_dirs WHERE path = ?",
		workingDir,
	).Scan(&workingDirID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query working_dir ID: %w", err)
	}

	// Build UNION ALL query with separate indexed queries for each priority
	// Each branch can use its own index efficiently
	// Priority 1: session (source_id match)
	// Priority 2: working_dir match (excluding session matches)
	// Priority 3: everything else (excluding session and working_dir matches)
	// Note: SQLite requires wrapping in subqueries to use ORDER BY/LIMIT in UNION branches
	query := `
		WITH ranked AS (
			-- Priority 1: session commands
			SELECT * FROM (
				SELECT timestamp, command_text, 1 as priority
				FROM commands
				WHERE source_id = ?
				ORDER BY timestamp DESC
				LIMIT ?
			)

			UNION ALL

			-- Priority 2: working_dir commands (not in session)
			SELECT * FROM (
				SELECT timestamp, command_text, 2 as priority
				FROM commands
				WHERE working_dir_id = ?
				  AND (source_id IS NULL OR source_id != ?)
				ORDER BY timestamp DESC
				LIMIT ?
			)

			UNION ALL

			-- Priority 3: full history (not in session or working_dir)
			SELECT * FROM (
				SELECT timestamp, command_text, 3 as priority
				FROM commands
				WHERE (source_id IS NULL OR source_id != ?)
				  AND (working_dir_id IS NULL OR working_dir_id != ?)
				ORDER BY timestamp DESC
				LIMIT ?
			)
		),
		deduped AS (
			SELECT
				command_text,
				LAG(command_text) OVER (ORDER BY priority, timestamp DESC) AS prev_command_text
			FROM ranked
		)
		SELECT command_text
		FROM deduped
		WHERE command_text != prev_command_text OR prev_command_text IS NULL
		LIMIT ?`

	args := []any{
		sourceID, bucketLimit, // Priority 1
		workingDirID, sourceID, bucketLimit, // Priority 2
		sourceID, workingDirID, bucketLimit, // Priority 3
		limit, // Final limit
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query commands: %w", err)
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		var cmd models.Command
		if err := rows.Scan(&cmd.CommandText); err != nil {
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

	query := `SELECT ` + commandSelectColumns + commandFromJoins + `
		WHERE c.id IN (SELECT max(id) FROM commands WHERE id >= ? AND id <= ? GROUP BY command_text)
		ORDER BY c.id ASC`

	rows, err := db.conn.Query(query, first, last)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands by range: %w", err)
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		cmd, err := scanCommand(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		commands = append(commands, *cmd)
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

	query := `SELECT ` + commandSelectColumns + commandFromJoins + `
		WHERE c.id IN (
			SELECT max(id)
			FROM commands
			WHERE id >= ? AND id <= ?
			AND command_text LIKE ? ESCAPE '\'
			GROUP BY command_text
		)
		ORDER BY c.id ASC`

	rows, err := db.conn.Query(query, first, last, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands by range with pattern: %w", err)
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		cmd, err := scanCommand(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		commands = append(commands, *cmd)
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

	query := `SELECT ` + commandSelectColumns + commandFromJoins + `
		WHERE c.id IN (
			SELECT max(c2.id)
			FROM commands c2
			JOIN sources s2 ON c2.source_id = s2.id
			WHERE c2.id >= ? AND c2.id <= ?
			AND s2.pid = ?
			AND s2.active = 1
			GROUP BY c2.command_text
		)
		ORDER BY c.id ASC`

	rows, err := db.conn.Query(query, first, last, sessionPid)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands by range (internal): %w", err)
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		cmd, err := scanCommand(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		commands = append(commands, *cmd)
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

	query := `SELECT ` + commandSelectColumns + commandFromJoins + `
		WHERE c.id IN (
			SELECT max(c2.id)
			FROM commands c2
			JOIN sources s2 ON c2.source_id = s2.id
			WHERE c2.id >= ? AND c2.id <= ?
			AND c2.command_text LIKE ? ESCAPE '\'
			AND s2.pid = ?
			AND s2.active = 1
			GROUP BY c2.command_text
		)
		ORDER BY c.id ASC`

	rows, err := db.conn.Query(query, first, last, pattern, sessionPid)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands by range with pattern (internal): %w", err)
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		cmd, err := scanCommand(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		commands = append(commands, *cmd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating commands: %w", err)
	}

	return commands, nil
}

// CloseSession marks all active sources from a session as inactive
// Returns the number of source records updated
func (db *DB) CloseSession(sessionPid int64) (int64, error) {
	result, err := db.conn.Exec(`
		UPDATE sources
		SET active = 0
		WHERE pid = ? AND active = 1`,
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

	// Look up source_id for this session (if provided)
	var sourceID sql.NullInt64
	if opts.SourceApp != "" && opts.SourcePid > 0 {
		err := db.conn.QueryRow(
			"SELECT id FROM sources WHERE app = ? AND pid = ? AND active = 1 ORDER BY id DESC LIMIT 1",
			opts.SourceApp, opts.SourcePid,
		).Scan(&sourceID)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to query source ID: %w", err)
		}
	}

	// Look up working_dir_id (if provided)
	var workingDirID sql.NullInt64
	if opts.WorkingDir != "" {
		err := db.conn.QueryRow(
			"SELECT id FROM working_dirs WHERE path = ?",
			opts.WorkingDir,
		).Scan(&workingDirID)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to query working_dir ID: %w", err)
		}
	}

	// Build base WHERE clause for common filters (prefix, IncludeShy)
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
	executeQuery := func(query string, args []any, ch chan<- queryResult) {
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

	// 1. Session query (if source_id found)
	// Includes working dir filter if also provided
	if sourceID.Valid {
		go func() {
			sessionWhere := baseWhere + " AND source_id = ?"
			sessionArgs := append(append([]any{}, baseArgs...), sourceID.Int64)

			query := `SELECT command_text FROM commands WHERE ` + sessionWhere + ` ORDER BY timestamp DESC LIMIT 1`
			executeQuery(query, sessionArgs, sessionChan)
		}()
	} else {
		sessionChan <- queryResult{nil, nil}
	}

	// 2. Working directory query (if working_dir_id found and no session filter)
	// Only runs if session query didn't run
	if workingDirID.Valid && !sourceID.Valid {
		go func() {
			workingDirWhere := baseWhere + " AND working_dir_id = ?"
			workingDirArgs := append(append([]any{}, baseArgs...), workingDirID.Int64)
			query := `SELECT command_text FROM commands WHERE ` + workingDirWhere + ` ORDER BY timestamp DESC LIMIT 1`
			executeQuery(query, workingDirArgs, workingDirChan)
		}()
	} else {
		workingDirChan <- queryResult{nil, nil}
	}

	// 3. Whole history query (always run)
	go func() {
		query := `SELECT command_text FROM commands WHERE ` + baseWhere + ` ORDER BY timestamp DESC LIMIT 1`
		executeQuery(query, baseArgs, historyChan)
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
		beforeQuery := `SELECT ` + commandSelectColumns + commandFromJoins + `
			WHERE c.id < ? AND s.pid = ?
			ORDER BY c.id DESC
			LIMIT ?`

		rows, err := db.conn.Query(beforeQuery, id, sessionPid, contextSize)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to query before commands: %w", err)
		}

		beforeCommands, err = db.scanCommandRows(rows)
		rows.Close()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to scan before commands: %w", err)
		}

		// Reverse beforeCommands to get chronological order (oldest first)
		for i, j := 0, len(beforeCommands)-1; i < j; i, j = i+1, j-1 {
			beforeCommands[i], beforeCommands[j] = beforeCommands[j], beforeCommands[i]
		}

		// Get commands after (same session, ID > target, ordered by ID ASC, limit contextSize)
		afterQuery := `SELECT ` + commandSelectColumns + commandFromJoins + `
			WHERE c.id > ? AND s.pid = ?
			ORDER BY c.id ASC
			LIMIT ?`

		rows, err = db.conn.Query(afterQuery, id, sessionPid, contextSize)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to query after commands: %w", err)
		}

		afterCommands, err = db.scanCommandRows(rows)
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

	query := `SELECT ` + commandSelectColumns + commandFromJoins + `
		WHERE c.id >= ? AND c.id <= ?
		ORDER BY c.id ASC`

	rows, err := db.conn.Query(query, startID, endID)
	if err != nil {
		return nil, fmt.Errorf("failed to query commands by ID range: %w", err)
	}
	defer rows.Close()

	return db.scanCommandRows(rows)
}

// scanCommandRows is a helper to scan multiple command rows using commandSelectColumns format
func (db *DB) scanCommandRows(rows *sql.Rows) ([]models.Command, error) {
	var commands []models.Command

	for rows.Next() {
		cmd, err := scanCommand(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}
		commands = append(commands, *cmd)
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
			w.path,
			COALESCE(g.branch, ''),
			COUNT(*) as command_count,
			MIN(c.timestamp) as first_time,
			MAX(c.timestamp) as last_time
		FROM commands c
		JOIN working_dirs w ON c.working_dir_id = w.id
		LEFT JOIN git_contexts g ON c.git_context_id = g.id
		WHERE c.timestamp >= ? AND c.timestamp < ?
		GROUP BY w.path, COALESCE(g.branch, '')
		ORDER BY (MAX(c.timestamp) - MIN(c.timestamp)) DESC, COUNT(*) DESC, w.path ASC
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
