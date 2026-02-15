package store

// migrations holds all schema migrations in order.
// Each migration is a SQL string that creates or alters tables.
// Migrations are applied in order and tracked via schema_version table.
var migrations = []string{
	// Migration 1: Core tables
	`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		project TEXT NOT NULL,
		prompt TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		priority TEXT NOT NULL DEFAULT 'normal',
		session_id TEXT NOT NULL DEFAULT '',
		pid INTEGER NOT NULL DEFAULT 0,
		git_branch TEXT NOT NULL DEFAULT '',
		output TEXT NOT NULL DEFAULT '',
		progress TEXT NOT NULL DEFAULT '',
		error TEXT NOT NULL DEFAULT '',
		cost_usd REAL NOT NULL DEFAULT 0,
		turns INTEGER NOT NULL DEFAULT 0,
		timeout_minutes INTEGER NOT NULL DEFAULT 30,
		dry_run INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		started_at TEXT NOT NULL DEFAULT '',
		completed_at TEXT NOT NULL DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project);
	CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);

	CREATE TABLE IF NOT EXISTS task_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL REFERENCES tasks(id),
		event_type TEXT NOT NULL,
		message TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_task_events_task_id ON task_events(task_id);

	CREATE TABLE IF NOT EXISTS oauth_tokens (
		token_hash TEXT PRIMARY KEY,
		token_type TEXT NOT NULL,
		client_id TEXT NOT NULL,
		scope TEXT NOT NULL DEFAULT '',
		expires_at TEXT NOT NULL,
		revoked INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE INDEX IF NOT EXISTS idx_oauth_tokens_client_id ON oauth_tokens(client_id);

	CREATE TABLE IF NOT EXISTS oauth_codes (
		code_hash TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		redirect_uri TEXT NOT NULL DEFAULT '',
		code_challenge TEXT NOT NULL DEFAULT '',
		scope TEXT NOT NULL DEFAULT '',
		expires_at TEXT NOT NULL,
		used INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);`,

	// Migration 2: Add type column to tasks for herald_push (linked tasks)
	`ALTER TABLE tasks ADD COLUMN type TEXT NOT NULL DEFAULT 'dispatched';
	CREATE INDEX IF NOT EXISTS idx_tasks_session_id ON tasks(session_id);`,

	// Migration 3: Add context column for human-readable task intent
	`ALTER TABLE tasks ADD COLUMN context TEXT NOT NULL DEFAULT '';`,
}
