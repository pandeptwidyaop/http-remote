package database

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestMigrationBackwardCompatibility tests that migrations work on existing databases
func TestMigrationBackwardCompatibility(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Simulate an old database - create tables without updated_at in apps
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			is_admin BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);

		CREATE TABLE apps (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			working_dir TEXT NOT NULL,
			token TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE commands (
			id TEXT PRIMARY KEY,
			app_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			command TEXT NOT NULL,
			timeout_seconds INTEGER DEFAULT 300,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
		);

		CREATE TABLE executions (
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
		);

		CREATE TABLE audit_logs (
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
		);
	`)
	if err != nil {
		t.Fatalf("failed to create old schema: %v", err)
	}

	// Insert some old data
	_, err = db.Exec(`
		INSERT INTO apps (id, name, working_dir, token, created_at)
		VALUES ('old-app-1', 'Old App', '/tmp', 'old-token-123', '2024-01-01 00:00:00')
	`)
	if err != nil {
		t.Fatalf("failed to insert old app: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (id, username, password_hash) VALUES (1, 'testuser', 'hash')
	`)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO commands (id, app_id, name, command) VALUES ('cmd-1', 'old-app-1', 'test', 'echo test')
	`)
	if err != nil {
		t.Fatalf("failed to insert command: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO executions (id, command_id, user_id, status)
		VALUES ('exec-1', 'cmd-1', 1, 'success')
	`)
	if err != nil {
		t.Fatalf("failed to insert execution: %v", err)
	}

	// Run migrations (this should add updated_at to apps and fix executions table)
	err = runMigrations(db)
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Verify apps table now has updated_at column
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('apps')
		WHERE name = 'updated_at'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to check for updated_at column: %v", err)
	}
	if count != 1 {
		t.Error("apps table should have updated_at column after migration")
	}

	// Verify old data still exists and has updated_at set
	var name, updatedAt string
	err = db.QueryRow(`
		SELECT name, updated_at FROM apps WHERE id = 'old-app-1'
	`).Scan(&name, &updatedAt)
	if err != nil {
		t.Fatalf("failed to query migrated app: %v", err)
	}
	if name != "Old App" {
		t.Errorf("expected name 'Old App', got %q", name)
	}
	if updatedAt == "" {
		t.Error("expected updated_at to be set after migration")
	}

	// Verify executions table can now accept user_id = 0 (API executions)
	_, err = db.Exec(`
		INSERT INTO executions (id, command_id, user_id, status)
		VALUES ('exec-api', 'cmd-1', 0, 'success')
	`)
	if err != nil {
		t.Errorf("should be able to insert API execution after migration: %v", err)
	}

	// Verify we can query executions with LEFT JOIN (testing executor service compatibility)
	var username string
	err = db.QueryRow(`
		SELECT COALESCE(u.username, 'API') as username
		FROM executions e
		LEFT JOIN users u ON e.user_id = u.id
		WHERE e.id = 'exec-api'
	`).Scan(&username)
	if err != nil {
		t.Fatalf("failed to query API execution: %v", err)
	}
	if username != "API" {
		t.Errorf("expected username 'API' for API execution, got %q", username)
	}

	// Verify new apps can be created
	_, err = db.Exec(`
		INSERT INTO apps (id, name, working_dir, token)
		VALUES ('new-app-1', 'New App', '/tmp', 'new-token-456')
	`)
	if err != nil {
		t.Errorf("should be able to insert new apps after migration: %v", err)
	}

	// Verify new app has updated_at set
	err = db.QueryRow(`
		SELECT updated_at FROM apps WHERE id = 'new-app-1'
	`).Scan(&updatedAt)
	if err != nil {
		t.Fatalf("failed to query new app: %v", err)
	}
	if updatedAt == "" {
		t.Error("expected updated_at to be set for new app")
	}

	// Verify commands table now has sort_order column
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('commands')
		WHERE name = 'sort_order'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to check for sort_order column: %v", err)
	}
	if count != 1 {
		t.Error("commands table should have sort_order column after migration")
	}

	// Verify existing command has sort_order set
	var sortOrder int
	err = db.QueryRow(`
		SELECT sort_order FROM commands WHERE id = 'cmd-1'
	`).Scan(&sortOrder)
	if err != nil {
		t.Fatalf("failed to query command sort_order: %v", err)
	}
	// Existing command should have sort_order = 0 (first command)
	if sortOrder != 0 {
		t.Errorf("expected sort_order 0 for first command, got %d", sortOrder)
	}

	// Verify index was created
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'index' AND name = 'idx_commands_sort_order'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to check for sort_order index: %v", err)
	}
	if count != 1 {
		t.Error("idx_commands_sort_order index should exist after migration")
	}
}
