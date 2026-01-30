#!/bin/bash
# generate-testdata.sh - Generate test databases for performance testing using pure SQL
#
# Creates normalized test databases with working_dirs, git_contexts, sources, and commands tables.
# Lookup table data is pre-generated, then commands randomly reference those IDs.

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0m'

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TESTDATA_DIR="$PROJECT_ROOT/testdata/perf"

mkdir -p "$TESTDATA_DIR"

echo -e "${BLUE}Generating test databases for performance testing...${NC}"
echo ""

# Function to generate a database
generate_database() {
    local name=$1
    local size=$2
    local db_path="$TESTDATA_DIR/test-sqlite-history-${name}.db"

    rm -f "$db_path"

    echo -e "${BLUE}Generating $name database ($size commands)...${NC}"
    local start_time=$(date +%s)

    sqlite3 "$db_path" <<'SCHEMA'
-- Create lookup tables
CREATE TABLE working_dirs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE
);

CREATE TABLE git_contexts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo TEXT,
    branch TEXT,
    UNIQUE(repo, branch)
);

CREATE TABLE sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    app TEXT NOT NULL,
    pid INTEGER NOT NULL,
    active INTEGER DEFAULT 1
);

CREATE TABLE commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    exit_status INTEGER NOT NULL,
    duration INTEGER NOT NULL,
    command_text TEXT NOT NULL,
    working_dir_id INTEGER NOT NULL REFERENCES working_dirs(id),
    git_context_id INTEGER REFERENCES git_contexts(id),
    source_id INTEGER REFERENCES sources(id),
    is_duplicate INTEGER DEFAULT 0
);

-- Create indexes (matching CreateIndexesSQL in db.go)
CREATE INDEX idx_source_timestamp ON commands (source_id, timestamp DESC);
CREATE INDEX idx_working_dir_timestamp ON commands (working_dir_id, timestamp DESC);
CREATE INDEX idx_timestamp_desc ON commands (timestamp DESC);
CREATE INDEX idx_sources_app_pid_active ON sources (app, pid, active);
CREATE INDEX idx_working_dirs_path ON working_dirs (path);
CREATE INDEX idx_command_text_id ON commands (command_text, id DESC);
CREATE INDEX idx_not_duplicate ON commands (id DESC) WHERE is_duplicate = 0;

-- Enable WAL mode
PRAGMA journal_mode=WAL;

-- Populate working_dirs lookup table (7 entries, IDs 1-7)
INSERT INTO working_dirs (path) VALUES
    ('/home/user/projects/shy'),
    ('/home/user/projects/webapp'),
    ('/home/user/projects/api'),
    ('/home/user/projects/frontend'),
    ('/home/user/documents'),
    ('/home/user'),
    ('/tmp');

-- Populate git_contexts lookup table (25 combinations, IDs 1-25)
-- 5 repos x 5 branches = 25 combinations (including NULL combos)
INSERT INTO git_contexts (repo, branch) VALUES
    ('github.com/user/shy', 'main'),
    ('github.com/user/shy', 'develop'),
    ('github.com/user/shy', 'feature/new-feature'),
    ('github.com/user/shy', 'feature/update'),
    ('github.com/user/shy', 'bugfix/issue-123'),
    ('github.com/user/webapp', 'main'),
    ('github.com/user/webapp', 'develop'),
    ('github.com/user/webapp', 'feature/new-feature'),
    ('github.com/user/webapp', 'feature/update'),
    ('github.com/user/webapp', 'bugfix/issue-123'),
    ('github.com/user/api', 'main'),
    ('github.com/user/api', 'develop'),
    ('github.com/user/api', 'feature/new-feature'),
    ('github.com/user/api', 'feature/update'),
    ('github.com/user/api', 'bugfix/issue-123'),
    ('github.com/company/project', 'main'),
    ('github.com/company/project', 'develop'),
    ('github.com/company/project', 'feature/new-feature'),
    ('github.com/company/project', 'feature/update'),
    ('github.com/company/project', 'bugfix/issue-123');

-- Populate sources lookup table (10 combinations, IDs 1-10)
-- 2 apps x 5 pids = 10 combinations
INSERT INTO sources (app, pid, active) VALUES
    ('zsh', 12345, 1),
    ('zsh', 67890, 1),
    ('zsh', 11111, 1),
    ('zsh', 22222, 1),
    ('zsh', 33333, 1),
    ('bash', 12345, 1),
    ('bash', 67890, 1),
    ('bash', 11111, 1),
    ('bash', 22222, 1),
    ('bash', 33333, 1);
SCHEMA

    # Generate commands using a recursive CTE for efficiency
    # This generates $size rows with random foreign key references

    # Check for local history database
    local history_db="${XDG_DATA_HOME:-$HOME/.local/share}/shy/history.db"
    local history_cmds=""

    if [[ -f "$history_db" ]]; then
        echo -e "${YELLOW}  Appending unique commands from local history...${NC}"
        # Extract unique commands, escape single quotes for SQL
        history_cmds=$(
            sqlite3 "$history_db" "SELECT DISTINCT command_text FROM commands" 2>/dev/null |
                sed "s/'/''/g" |
                awk '{printf "    ('\''%s'\''),\n", $0}' |
                head -n -1
            sqlite3 "$history_db" "SELECT DISTINCT command_text FROM commands" 2>/dev/null |
                sed "s/'/''/g" |
                tail -1 |
                awk '{printf "    ('\''%s'\'')", $0}'
        )
    fi

    sqlite3 "$db_path" <<EOF
-- Sample command texts (48 base commands + local history)
CREATE TEMP TABLE sample_commands (idx INTEGER PRIMARY KEY, cmd TEXT UNIQUE);
INSERT OR IGNORE INTO sample_commands (cmd) VALUES
    ('ls -la'),
    ('cd /home/user/projects'),
    ('git status'),
    ('git add .'),
    ('git commit -m ''update'''),
    ('git push'),
    ('git pull'),
    ('git log'),
    ('git diff'),
    ('git branch'),
    ('npm install'),
    ('npm test'),
    ('npm run build'),
    ('npm start'),
    ('go build'),
    ('go test ./...'),
    ('go mod tidy'),
    ('docker ps'),
    ('docker-compose up -d'),
    ('docker-compose down'),
    ('kubectl get pods'),
    ('kubectl logs'),
    ('ssh user@server'),
    ('vim README.md'),
    ('cat file.txt'),
    ('grep -r ''pattern'' .'),
    ('find . -name ''*.go'''),
    ('make build'),
    ('cargo build'),
    ('cargo test'),
    ('python script.py'),
    ('pytest'),
    ('curl https://api.example.com'),
    ('wget https://example.com/file'),
    ('tar -xzf archive.tar.gz'),
    ('rsync -av src/ dest/'),
    ('systemctl status nginx'),
    ('journalctl -u service'),
    ('top'),
    ('htop'),
    ('df -h'),
    ('du -sh *'),
    ('ps aux'),
    ('netstat -tuln'),
    ('echo ''hello world'''),
    ('mkdir -p dir'),
    ('rm -rf temp'),
    ('cp file1 file2');
${history_cmds:+
-- Commands from local history
INSERT OR IGNORE INTO sample_commands (cmd) VALUES
$history_cmds;
}

-- Reindex to remove gaps from INSERT OR IGNORE
CREATE TEMP TABLE sample_commands_reindexed (idx INTEGER PRIMARY KEY, cmd TEXT);
INSERT INTO sample_commands_reindexed (cmd) SELECT cmd FROM sample_commands;
DROP TABLE sample_commands;
ALTER TABLE sample_commands_reindexed RENAME TO sample_commands;

-- Count of lookup tables
-- working_dirs: 7 entries
-- git_contexts: 20 entries
-- sources: 10 entries
-- sample_commands: 48 base + local history (dynamic)

-- Base timestamp (Jan 1, 2024)
-- 1704067200

-- Generate $size commands using recursive CTE
WITH RECURSIVE
    counter(n) AS (
        SELECT 1
        UNION ALL
        SELECT n + 1 FROM counter WHERE n < $size
    ),
    cmd_count AS (SELECT MAX(idx) as max_idx FROM sample_commands),
    random_data AS (
        SELECT
            n,
            -- Timestamp: base + n * random(1-60)
            -- Note: (random()+n*0) forces per-row evaluation
            1704067200 + n * (abs((random()+n*0) % 60) + 1) as ts,
            -- Exit status: 0 most of the time, 1 about 10%
            CASE WHEN abs((random()+n*0) % 10) = 0 THEN 1 ELSE 0 END as exit_status,
            -- Duration: 100-10000 ms
            abs((random()+n*0) % 9900) + 100 as duration,
            -- Random command (1 to count of sample_commands)
            abs((random()+n*0) % (SELECT max_idx FROM cmd_count)) + 1 as cmd_idx,
            -- Random working_dir_id (1-7)
            abs((random()+n*0) % 7) + 1 as wd_id,
            -- Random git_context_id (1-25, where 21-25 are NULL repos)
            abs((random()+n*0) % 25) + 1 as gc_id,
            -- Random source_id (1-10)
            abs((random()+n*0) % 10) + 1 as src_id
        FROM counter
    )
INSERT INTO commands (timestamp, exit_status, duration, command_text, working_dir_id, git_context_id, source_id)
SELECT
    rd.ts,
    rd.exit_status,
    rd.duration,
    sc.cmd,
    rd.wd_id,
    rd.gc_id,
    rd.src_id
FROM random_data rd
JOIN sample_commands sc ON sc.idx = rd.cmd_idx;

-- Drop temp table
DROP TABLE sample_commands;

-- Mark duplicates: all but the most recent occurrence of each command_text
UPDATE commands SET is_duplicate = 1
WHERE id NOT IN (
    SELECT MAX(id) FROM commands GROUP BY command_text
);

-- Checkpoint to flush WAL
PRAGMA wal_checkpoint(TRUNCATE);
EOF

    local end_time=$(date +%s)
    local elapsed=$((end_time - start_time))
    local size_bytes=$(stat -c%s "$db_path" 2>/dev/null || stat -f%z "$db_path")
    local size_mb=$(echo "scale=2; $size_bytes / 1024 / 1024" | bc)

    echo -e "${GREEN}Created $db_path (${size_mb} MB) in ${elapsed}s${NC}"
    echo ""
}

# Generate databases
generate_database "medium" 10000
generate_database "large" 100000
generate_database "xlarge" 1000000
generate_database "xl-2" 2000000
generate_database "xl-3" 3000000
generate_database "xl-4" 4000000
generate_database "xl-5" 5000000

echo -e "${GREEN}All test databases created in $TESTDATA_DIR${NC}"
