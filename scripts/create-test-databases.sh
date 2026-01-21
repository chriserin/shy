#!/bin/bash
# create-test-databases.sh - Generate test databases with various sizes for performance testing
# This script uses the 'shy generate-testdata' command for efficient database generation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Creating test databases for performance testing...${NC}"
echo ""

# Build the shy binary
echo -e "${BLUE}Building shy binary...${NC}"
cd "$PROJECT_ROOT"
go build -o shy .
echo ""

# Run the generate-testdata command
./shy generate-testdata

exit 0

# OLD IMPLEMENTATION BELOW (kept for reference, but not executed)
# The CLI-based insertion method is very slow for large datasets
# The new Go command uses batch inserts and is much faster
# ============================================================

# Function to generate test database
generate_db() {
    local name=$1
    local size=$2
    local db_path="$TEST_DATA_DIR/history-${name}.db"

    echo -e "${BLUE}Generating ${name} database with ${size} commands...${NC}"

    # Remove existing database
    rm -f "$db_path"

    # Sample commands to use (realistic variety)
    local commands=(
        "ls -la"
        "cd /home/user/projects"
        "git status"
        "git add ."
        "git commit -m 'update'"
        "git push"
        "git pull"
        "npm install"
        "npm test"
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
        "wget https://example.com/file"
        "tar -xzf archive.tar.gz"
        "rsync -av src/ dest/"
        "systemctl status nginx"
        "journalctl -u service"
        "top"
        "htop"
    )

    local dirs=(
        "/home/user/projects/shy"
        "/home/user/projects/webapp"
        "/home/user/projects/api"
        "/home/user/documents"
        "/home/user"
        "/tmp"
        "/var/log"
    )

    local git_repos=(
        "github.com/user/shy"
        "github.com/user/webapp"
        "github.com/user/api"
        ""
    )

    local git_branches=(
        "main"
        "develop"
        "feature/new-feature"
        "bugfix/issue-123"
        ""
    )

    local source_apps=("zsh" "bash")
    local source_pids=(12345 67890 11111 22222 33333)

    # Base timestamp (Jan 1, 2024)
    local base_ts=1704067200

    # Insert commands
    for ((i = 1; i <= $size; i++)); do
        # Pick random values
        local cmd="${commands[$((RANDOM % ${#commands[@]}))]}"
        local dir="${dirs[$((RANDOM % ${#dirs[@]}))]}"
        local repo="${git_repos[$((RANDOM % ${#git_repos[@]}))]}"
        local branch="${git_branches[$((RANDOM % ${#git_branches[@]}))]}"
        local app="${source_apps[$((RANDOM % ${#source_apps[@]}))]}"
        local pid="${source_pids[$((RANDOM % ${#source_pids[@]}))]}"
        local status=$((RANDOM % 10 == 0 ? 1 : 0)) # 10% failure rate

        # Increment timestamp (1-60 seconds between commands)
        local ts=$((base_ts + i * (RANDOM % 60 + 1)))

        # Random duration (100ms to 10s)
        local duration=$((RANDOM % 9900 + 100))

        # Build insert command
        local insert_cmd="$SHY insert --db \"$db_path\" --command \"$cmd\" --dir \"$dir\" --status $status --timestamp $ts --source-app \"$app\" --source-pid $pid --duration $duration"

        # Add git info if present
        if [ -n "$repo" ]; then
            insert_cmd="$insert_cmd --git-repo \"$repo\""
        fi
        if [ -n "$branch" ]; then
            insert_cmd="$insert_cmd --git-branch \"$branch\""
        fi

        # Execute insert (suppress stdout output, stderr will show errors)
        eval "$insert_cmd" >/dev/null

        # Progress indicator
        if [ $((i % 1000)) -eq 0 ]; then
            echo -e "${GREEN}  Inserted $i / $size commands...${NC}"
        fi
    done

    echo -e "${GREEN}✓ Created $db_path with $size commands${NC}"

    # Show database size
    local db_size=$(du -h "$db_path" | cut -f1)
    echo -e "  Database size: $db_size"
}

# Generate databases of different sizes
generate_db "medium" 10000
generate_db "large" 100000
# Uncomment to generate xlarge database (takes ~10-15 minutes)
# generate_db "xlarge" 1000000

echo -e "${GREEN}✓ All test databases created in $TEST_DATA_DIR${NC}"
echo ""
echo "Test databases:"
ls -lh "$TEST_DATA_DIR"/*.db
