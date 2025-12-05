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
		user_id INTEGER,
		status TEXT NOT NULL DEFAULT 'pending',
		output TEXT,
		exit_code INTEGER,
		started_at DATETIME,
		finished_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (command_id) REFERENCES commands(id)
	)`,

	`CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		username TEXT,
		action TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_id TEXT,
		ip_address TEXT,
		user_agent TEXT,
		details TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
	)`,

	`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)`,
	`CREATE INDEX IF NOT EXISTS idx_commands_app_id ON commands(app_id)`,
	`CREATE INDEX IF NOT EXISTS idx_executions_command_id ON executions(command_id)`,
	`CREATE INDEX IF NOT EXISTS idx_executions_user_id ON executions(user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status)`,
	`CREATE INDEX IF NOT EXISTS idx_apps_token ON apps(token)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at)`,
}

func runMigrations(db *sql.DB) error {
	// Run initial migrations
	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return err
		}
	}

	// Run post-migrations for backward compatibility
	if err := migrateExecutionsTable(db); err != nil {
		return err
	}

	return nil
}

// migrateExecutionsTable handles backward compatibility for executions table
// Removes foreign key constraint on user_id for API deployments
func migrateExecutionsTable(db *sql.DB) error {
	// Check if we need to migrate by checking if foreign key exists
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name='executions'
		AND sql LIKE '%FOREIGN KEY (user_id)%'
	`).Scan(&count)

	if err != nil || count == 0 {
		// Already migrated or table doesn't exist yet
		return nil
	}

	// Begin transaction for migration
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create new executions table without user_id foreign key
	_, err = tx.Exec(`
		CREATE TABLE executions_new (
			id TEXT PRIMARY KEY,
			command_id TEXT NOT NULL,
			user_id INTEGER,
			status TEXT NOT NULL DEFAULT 'pending',
			output TEXT,
			exit_code INTEGER,
			started_at DATETIME,
			finished_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (command_id) REFERENCES commands(id)
		)
	`)
	if err != nil {
		return err
	}

	// Copy data from old table
	_, err = tx.Exec(`
		INSERT INTO executions_new
		SELECT id, command_id, user_id, status, output, exit_code, started_at, finished_at, created_at
		FROM executions
	`)
	if err != nil {
		return err
	}

	// Drop old table
	_, err = tx.Exec(`DROP TABLE executions`)
	if err != nil {
		return err
	}

	// Rename new table
	_, err = tx.Exec(`ALTER TABLE executions_new RENAME TO executions`)
	if err != nil {
		return err
	}

	// Recreate indexes
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_executions_command_id ON executions(command_id)`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_executions_user_id ON executions(user_id)`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status)`)
	if err != nil {
		return err
	}

	return tx.Commit()
}
