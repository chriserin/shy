#!/bin/bash
# migrate-sqlite-to-duckdb.sh - Migrate shy history from SQLite to DuckDB
#
# This script exports all commands from the SQLite database and imports them
# into a new DuckDB database, then replaces the old database.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Default paths
XDG_DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}"
DEFAULT_DB_PATH="$XDG_DATA_HOME/shy/history.db"

# Parse arguments
DB_PATH="${1:-$DEFAULT_DB_PATH}"

echo -e "${BLUE}Shy SQLite to DuckDB Migration${NC}"
echo ""

# Resolve ~ in path
DB_PATH="${DB_PATH/#\~/$HOME}"

# Check if source database exists
if [ ! -f "$DB_PATH" ]; then
    echo -e "${RED}Error: SQLite database not found at $DB_PATH${NC}"
    exit 1
fi

# Check if it's actually a SQLite database
if ! file "$DB_PATH" | grep -q "SQLite"; then
    echo -e "${RED}Error: $DB_PATH does not appear to be a SQLite database${NC}"
    exit 1
fi

echo -e "${BLUE}Source database:${NC} $DB_PATH"

# Create paths for new and backup databases
DB_DIR=$(dirname "$DB_PATH")
DB_NAME=$(basename "$DB_PATH" .db)
NEW_DB_PATH="$DB_DIR/${DB_NAME}.duckdb.tmp"
BACKUP_PATH="$DB_DIR/${DB_NAME}.sqlite.backup"

# Remove any existing temp file
rm -f "$NEW_DB_PATH"

echo -e "${BLUE}Creating new DuckDB database...${NC}"

# Get the count of commands to migrate
COMMAND_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM commands;")
echo -e "${BLUE}Commands to migrate:${NC} $COMMAND_COUNT"

# Get the max ID for sequence initialization
MAX_ID=$(sqlite3 "$DB_PATH" "SELECT COALESCE(MAX(id), 0) FROM commands;")

# Create DuckDB database and import data using DuckDB's sqlite extension
duckdb "$NEW_DB_PATH" <<EOF
-- Install and load sqlite extension
INSTALL sqlite;
LOAD sqlite;

-- Create sequence starting after the max existing ID
CREATE SEQUENCE id_sequence START $((MAX_ID + 1));

-- Create the commands table with DuckDB schema
CREATE TABLE commands (
    id INTEGER PRIMARY KEY DEFAULT nextval('id_sequence'),
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

-- Attach the SQLite database
ATTACH '$DB_PATH' AS sqlite_db (TYPE sqlite);

-- Insert all data from SQLite, preserving IDs
INSERT INTO commands (id, timestamp, exit_status, duration, command_text, working_dir, git_repo, git_branch, source_app, source_pid, source_active)
SELECT id, timestamp, exit_status, duration, command_text, working_dir, git_repo, git_branch, source_app, source_pid, source_active
FROM sqlite_db.commands;

-- Verify the migration
SELECT 'Migrated ' || COUNT(*) || ' commands' FROM commands;
EOF

# Verify the new database has the same count
NEW_COUNT=$(duckdb -noheader -column "$NEW_DB_PATH" "SELECT COUNT(*) FROM commands;" 2>/dev/null | tail -1)

NEW_COUNT=$(echo $NEW_COUNT | tr -d ' ')

if [ "$NEW_COUNT" != "$COMMAND_COUNT" ]; then
    echo -e "${RED}Error: Migration verification failed!${NC}"
    echo -e "  Expected: $COMMAND_COUNT commands"
    echo -e "  Got: $NEW_COUNT commands"
    rm -f "$NEW_DB_PATH"
    exit 1
fi

echo -e "${GREEN}Migration verified: $NEW_COUNT commands${NC}"

# Backup old database
echo -e "${BLUE}Backing up SQLite database to:${NC} $BACKUP_PATH"
cp "$DB_PATH" "$BACKUP_PATH"

# Replace old database with new one
echo -e "${BLUE}Replacing database...${NC}"
mv "$NEW_DB_PATH" "$DB_PATH"

# Clean up any SQLite WAL/SHM files
rm -f "${DB_PATH}-wal" "${DB_PATH}-shm"

echo ""
echo -e "${GREEN}Migration complete!${NC}"
echo -e "  New DuckDB database: $DB_PATH"
echo -e "  SQLite backup: $BACKUP_PATH"
echo ""
echo -e "${YELLOW}Note: Set SHY_DB_TYPE=duckdb in your shell to use the new database.${NC}"
echo -e "${YELLOW}Add this to your .zshrc or .bashrc:${NC}"
echo -e "  export SHY_DB_TYPE=duckdb"
