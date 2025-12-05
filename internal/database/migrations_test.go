package database

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	return db
}

func TestCreateMigrationsTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := createMigrationsTable(db)
	if err != nil {
		t.Fatalf("failed to create migrations table: %v", err)
	}

	// Verify table exists
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master 
		WHERE type='table' AND name='migrations'
	`).Scan(&count)
	
	if err != nil {
		t.Fatalf("failed to query migrations table: %v", err)
	}
	
	if count != 1 {
		t.Errorf("expected 1 migrations table, got %d", count)
	}
}

func TestRecordMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := createMigrationsTable(db)
	if err != nil {
		t.Fatalf("failed to create migrations table: %v", err)
	}

	err = recordMigration(db, "test_migration", 1)
	if err != nil {
		t.Fatalf("failed to record migration: %v", err)
	}

	// Verify migration was recorded
	var migrationName string
	var batch int
	err = db.QueryRow(`
		SELECT migration, batch FROM migrations WHERE migration = ?
	`, "test_migration").Scan(&migrationName, &batch)
	
	if err != nil {
		t.Fatalf("failed to query migration: %v", err)
	}
	
	if migrationName != "test_migration" {
		t.Errorf("expected migration name 'test_migration', got %q", migrationName)
	}
	
	if batch != 1 {
		t.Errorf("expected batch 1, got %d", batch)
	}
}

func TestHasMigrationRun(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := createMigrationsTable(db)
	if err != nil {
		t.Fatalf("failed to create migrations table: %v", err)
	}

	// Should return false for non-existent migration
	hasRun, err := hasMigrationRun(db, "nonexistent")
	if err != nil {
		t.Fatalf("failed to check migration: %v", err)
	}
	if hasRun {
		t.Error("expected hasRun to be false for nonexistent migration")
	}

	// Record a migration
	err = recordMigration(db, "test_migration", 1)
	if err != nil {
		t.Fatalf("failed to record migration: %v", err)
	}

	// Should return true for existing migration
	hasRun, err = hasMigrationRun(db, "test_migration")
	if err != nil {
		t.Fatalf("failed to check migration: %v", err)
	}
	if !hasRun {
		t.Error("expected hasRun to be true for existing migration")
	}
}

func TestRunMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := runMigrations(db)
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Verify all expected tables exist
	expectedTables := []string{"users", "sessions", "apps", "commands", "executions", "audit_logs", "migrations"}
	
	for _, table := range expectedTables {
		var count int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master 
			WHERE type='table' AND name=?
		`, table).Scan(&count)
		
		if err != nil {
			t.Fatalf("failed to query table %s: %v", table, err)
		}
		
		if count != 1 {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestMigrateExecutionsTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create old-style executions table with user_id FK
	_, err := db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY)
	`)
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (id) VALUES (1)
	`)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE commands (id TEXT PRIMARY KEY)
	`)
	if err != nil {
		t.Fatalf("failed to create commands table: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO commands (id) VALUES ('cmd-1')
	`)
	if err != nil {
		t.Fatalf("failed to insert test command: %v", err)
	}

	_, err = db.Exec(`
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
		)
	`)
	if err != nil {
		t.Fatalf("failed to create old executions table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO executions (id, command_id, user_id, status) 
		VALUES ('exec-1', 'cmd-1', 1, 'success')
	`)
	if err != nil {
		t.Fatalf("failed to insert test execution: %v", err)
	}

	// Run migration
	err = migrateExecutionsTable(db)
	if err != nil {
		t.Fatalf("failed to migrate executions table: %v", err)
	}

	// Verify data was preserved
	var status string
	err = db.QueryRow(`
		SELECT status FROM executions WHERE id = 'exec-1'
	`).Scan(&status)
	
	if err != nil {
		t.Fatalf("failed to query migrated data: %v", err)
	}
	
	if status != "success" {
		t.Errorf("expected status 'success', got %q", status)
	}

	// Verify FK on user_id was removed (can insert with non-existent user_id)
	_, err = db.Exec(`
		INSERT INTO executions (id, command_id, user_id, status)
		VALUES ('exec-2', 'cmd-1', 0, 'pending')
	`)
	if err != nil {
		t.Errorf("should be able to insert execution with user_id=0 after migration: %v", err)
	}
}

func TestAddAppsUpdatedAtColumn(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create old apps table without updated_at column
	_, err := db.Exec(`
		CREATE TABLE apps (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			working_dir TEXT NOT NULL,
			token TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create apps table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO apps (id, name, working_dir, token)
		VALUES ('app-1', 'Test App', '/tmp', 'test-token-123')
	`)
	if err != nil {
		t.Fatalf("failed to insert test app: %v", err)
	}

	// Verify updated_at column doesn't exist
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('apps')
		WHERE name = 'updated_at'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to check for updated_at column: %v", err)
	}
	if count != 0 {
		t.Fatal("updated_at column should not exist before migration")
	}

	// Run migration
	err = addAppsUpdatedAtColumn(db)
	if err != nil {
		t.Fatalf("failed to add updated_at column: %v", err)
	}

	// Verify updated_at column exists
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('apps')
		WHERE name = 'updated_at'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to check for updated_at column: %v", err)
	}
	if count != 1 {
		t.Error("updated_at column should exist after migration")
	}

	// Verify old data still exists and has updated_at set
	var name string
	var updatedAt string
	err = db.QueryRow(`
		SELECT name, updated_at FROM apps WHERE id = 'app-1'
	`).Scan(&name, &updatedAt)
	if err != nil {
		t.Fatalf("failed to query migrated data: %v", err)
	}
	if name != "Test App" {
		t.Errorf("expected name 'Test App', got %q", name)
	}
	if updatedAt == "" {
		t.Error("expected updated_at to be set with default value")
	}

	// Verify new inserts work with updated_at
	_, err = db.Exec(`
		INSERT INTO apps (id, name, working_dir, token)
		VALUES ('app-2', 'Test App 2', '/tmp', 'test-token-456')
	`)
	if err != nil {
		t.Errorf("should be able to insert apps after migration: %v", err)
	}

	// Running migration again should be idempotent
	err = addAppsUpdatedAtColumn(db)
	if err != nil {
		t.Errorf("migration should be idempotent: %v", err)
	}
}
