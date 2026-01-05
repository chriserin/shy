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
	defaultDBPath = "~/.local/share/shy/history.db"
	createTableSQL = `
		CREATE TABLE IF NOT EXISTS commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			exit_status INTEGER NOT NULL,
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

	return &DB{
		conn: conn,
		path: dbPath,
	}, nil
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
	result, err := db.conn.Exec(`
		INSERT INTO commands (timestamp, exit_status, command_text, working_dir, git_repo, git_branch)
		VALUES (?, ?, ?, ?, ?, ?)`,
		cmd.Timestamp,
		cmd.ExitStatus,
		cmd.CommandText,
		cmd.WorkingDir,
		cmd.GitRepo,
		cmd.GitBranch,
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
		SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch
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
			SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch
			FROM (
				SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch
				FROM commands
				%s
				ORDER BY timestamp DESC
				LIMIT %d
			)
			ORDER BY timestamp ASC`, whereClause, limit)
	} else {
		query = fmt.Sprintf(`
			SELECT id, timestamp, exit_status, command_text, working_dir, git_repo, git_branch
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
