#!/bin/bash
# generate-testdata-sql.sh - Generate test databases using raw SQL (fastest method)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_DATA_DIR="$PROJECT_ROOT/testdata/perf"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Generating test databases using SQL...${NC}"
echo ""

# Create testdata/perf directory
mkdir -p "$TEST_DATA_DIR"

# Function to generate SQL for database
generate_sql() {
    local size=$1
    local base_ts=1704067200  # Jan 1, 2024

    # SQL header (schema will be created automatically by shy, but we can ensure it exists)
    cat << 'EOF'
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA cache_size=10000;
PRAGMA temp_store=MEMORY;

-- Ensure schema exists
CREATE TABLE IF NOT EXISTS commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    exit_status INTEGER NOT NULL,
    command_text TEXT NOT NULL,
    working_dir TEXT NOT NULL,
    git_repo TEXT,
    git_branch TEXT,
    duration INTEGER,
    source_app TEXT,
    source_pid INTEGER,
    source_active INTEGER
);

BEGIN TRANSACTION;
EOF

    # Commands array
    local commands=(
        "ls -la"
        "cd /home/user/projects"
        "git status"
        "git add ."
        "git commit -m 'update'"
        "git push"
        "git pull"
        "git log"
        "git diff"
        "npm install"
        "npm test"
        "npm run build"
        "go build"
        "go test ./..."
        "docker ps"
        "docker-compose up -d"
        "kubectl get pods"
        "ssh user@server"
        "vim README.md"
        "cat file.txt"
        "grep -r 'pattern' ."
        "find . -name '*.go'"
        "make build"
        "cargo build"
        "python script.py"
        "pytest"
        "curl https://api.example.com"
        "echo 'hello world'"
        "mkdir -p dir"
        "rm -rf temp"
    )

    local dirs=(
        "/home/user/projects/shy"
        "/home/user/projects/webapp"
        "/home/user/projects/api"
        "/home/user/documents"
        "/home/user"
        "/tmp"
    )

    local git_repos=(
        "github.com/user/shy"
        "github.com/user/webapp"
        "github.com/user/api"
        "NULL"
    )

    local git_branches=(
        "main"
        "develop"
        "feature/new-feature"
        "bugfix/issue-123"
        "NULL"
    )

    local source_apps=("zsh" "bash")
    local source_pids=(12345 67890 11111 22222 33333)

    # Generate INSERT statements
    for ((i=1; i<=$size; i++)); do
        # Pick random values
        local cmd="${commands[$((RANDOM % ${#commands[@]}))]}"
        local dir="${dirs[$((RANDOM % ${#dirs[@]}))]}"
        local repo="${git_repos[$((RANDOM % ${#git_repos[@]}))]}"
        local branch="${git_branches[$((RANDOM % ${#git_branches[@]}))]}"
        local app="${source_apps[$((RANDOM % ${#source_apps[@]}))]}"
        local pid="${source_pids[$((RANDOM % ${#source_pids[@]}))]}"
        local status=$((RANDOM % 10 == 0 ? 1 : 0))  # 10% failure rate

        # Increment timestamp (1-60 seconds between commands)
        local ts=$((base_ts + i * (RANDOM % 60 + 1)))

        # Random duration (100ms to 10s)
        local duration=$((RANDOM % 9900 + 100))

        # Escape single quotes in command
        cmd="${cmd//\'/\'\'}"

        # Build INSERT statement
        echo "INSERT INTO commands (timestamp, exit_status, command_text, working_dir, git_repo, git_branch, duration, source_app, source_pid, source_active) VALUES ($ts, $status, '$cmd', '$dir', $repo, $branch, $duration, '$app', $pid, 1);"

        # Progress indicator (every 10000 commands)
        if [ $((i % 10000)) -eq 0 ]; then
            echo "-- Inserted $i / $size commands" >&2
        fi
    done

    # SQL footer
    cat << 'EOF'
COMMIT;

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_timestamp ON commands(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_command_text ON commands(command_text);
CREATE INDEX IF NOT EXISTS idx_working_dir ON commands(working_dir);
CREATE INDEX IF NOT EXISTS idx_source_session ON commands(source_app, source_pid, source_active);

-- Analyze for query optimizer
ANALYZE;
EOF
}

# Function to create database
create_database() {
    local name=$1
    local size=$2
    local db_path="$TEST_DATA_DIR/history-${name}.db"

    echo -e "${BLUE}Generating ${name} database with ${size} commands...${NC}"

    # Remove existing database
    rm -f "$db_path"

    # Generate SQL and pipe to sqlite3
    local start_time=$(date +%s)
    generate_sql "$size" | sqlite3 "$db_path" 2>&1 | grep "Inserted" || true
    local end_time=$(date +%s)
    local elapsed=$((end_time - start_time))

    # Get database size
    local db_size=$(du -h "$db_path" | cut -f1)

    echo -e "${GREEN}✓ Created $db_path ($db_size) in ${elapsed}s${NC}"
    echo ""
}

# Generate databases
create_database "medium" 10000
create_database "large" 100000

# Uncomment to generate xlarge (takes 2-3 minutes with SQL)
# create_database "xlarge" 1000000

echo -e "${GREEN}✓ All test databases created in $TEST_DATA_DIR${NC}"
echo ""
echo "Test databases:"
ls -lh "$TEST_DATA_DIR"/*.db
