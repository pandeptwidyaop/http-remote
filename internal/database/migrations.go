package database

import (
	"database/sql"
)

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		is_admin BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,

	`CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	)`,

	`CREATE TABLE IF NOT EXISTS apps (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		description TEXT,
		working_dir TEXT NOT NULL,
		token TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,

	`CREATE TABLE IF NOT EXISTS commands (
		id TEXT PRIMARY KEY,
		app_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		command TEXT NOT NULL,
		timeout_seconds INTEGER DEFAULT 300,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
	)`,

	`CREATE TABLE IF NOT EXISTS executions (
		id TEXT PRIMARY KEY,
		command_id TEXT NOT NULL,
		user_id INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		output TEXT,
		exit_code INTEGER,
		started_at DATETIME,
		finished_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (command_id) REFERENCES commands(id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	)`,

	`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)`,
	`CREATE INDEX IF NOT EXISTS idx_commands_app_id ON commands(app_id)`,
	`CREATE INDEX IF NOT EXISTS idx_executions_command_id ON executions(command_id)`,
	`CREATE INDEX IF NOT EXISTS idx_executions_user_id ON executions(user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status)`,
	`CREATE INDEX IF NOT EXISTS idx_apps_token ON apps(token)`,
}

func runMigrations(db *sql.DB) error {
	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return err
		}
	}
	return nil
}
