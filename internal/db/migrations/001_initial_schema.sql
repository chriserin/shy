CREATE TABLE IF NOT EXISTS working_dirs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	path TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS git_contexts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	repo TEXT,
	branch TEXT,
	UNIQUE(repo, branch)
);

CREATE TABLE IF NOT EXISTS sources (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	app TEXT NOT NULL,
	pid INTEGER NOT NULL,
	active INTEGER DEFAULT 1,
	UNIQUE(app, pid, active)
);

CREATE TABLE IF NOT EXISTS commands (
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

CREATE INDEX IF NOT EXISTS idx_source_timestamp ON commands (source_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_working_dir_timestamp ON commands (working_dir_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_timestamp_desc ON commands (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_sources_app_pid_active ON sources (app, pid, active);
CREATE INDEX IF NOT EXISTS idx_working_dirs_path ON working_dirs (path);
CREATE INDEX IF NOT EXISTS idx_command_text_id ON commands (command_text, id DESC);
CREATE INDEX IF NOT EXISTS idx_not_duplicate ON commands (id DESC) WHERE is_duplicate = 0;
