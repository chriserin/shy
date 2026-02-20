package migrations

import (
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed 001_initial_schema.sql
var initialSchemaSQL string

//go:embed 002_starred_commands.sql
var starredCommandsSQL string

// All contains all migrations in order. Each migration's index+1 is its version number.
var All = []string{
	initialSchemaSQL,   // version 1
	starredCommandsSQL, // version 2
}

// Migrate runs all pending migrations on the database.
// It reads the current version from PRAGMA user_version and runs any
// migrations with index >= current version. Each migration runs in its
// own transaction. If a migration fails, it rolls back and stops.
func Migrate(db *sql.DB) error {
	// 1. Read current version from PRAGMA user_version
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("failed to read schema version: %w", err)
	}

	// 2. For each migration with index >= current version:
	//    a. Begin transaction
	//    b. Execute SQL
	//    c. Set PRAGMA user_version = index + 1
	//    d. Commit
	for i := version; i < len(All); i++ {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", i+1, err)
		}

		if _, err := tx.Exec(All[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}

		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to set schema version to %d: %w", i+1, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", i+1, err)
		}
	}

	return nil
}
