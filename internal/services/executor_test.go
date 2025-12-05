package services_test

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

func setupTestDB(t *testing.T) (*database.DB, *sql.DB, *config.Config) {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	db := &database.DB{DB: sqlDB}

	// Create tables
	_, err = sqlDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			is_admin BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE apps (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			working_dir TEXT,
			token TEXT NOT NULL UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE commands (
			id TEXT PRIMARY KEY,
			app_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			command TEXT NOT NULL,
			timeout_seconds INTEGER DEFAULT 300,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (app_id) REFERENCES apps(id)
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
			FOREIGN KEY (command_id) REFERENCES commands(id)
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	cfg := &config.Config{
		Execution: config.ExecutionConfig{
			DefaultTimeout: 300,
			MaxTimeout:     3600,
		},
	}

	return db, sqlDB, cfg
}

func TestExecutorService_CreateExecution(t *testing.T) {
	db, sqlDB, cfg := setupTestDB(t)
	defer sqlDB.Close()

	appSvc := services.NewAppService(db)
	execSvc := services.NewExecutorService(db, cfg, appSvc)

	// Create test app and command
	app, err := appSvc.CreateApp(&models.CreateAppRequest{
		Name:        "Test App",
		Description: "Test Description",
		WorkingDir:  "/tmp",
	})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	cmd, err := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:           "test",
		Description:    "test command",
		Command:        "echo test",
		TimeoutSeconds: 30,
	})
	if err != nil {
		t.Fatalf("failed to create command: %v", err)
	}

	// Create execution
	exec, err := execSvc.CreateExecution(cmd.ID, 0)
	if err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	if exec.ID == "" {
		t.Error("expected execution ID to be set")
	}

	if exec.CommandID != cmd.ID {
		t.Errorf("expected command ID %q, got %q", cmd.ID, exec.CommandID)
	}

	if exec.UserID != 0 {
		t.Errorf("expected user ID 0, got %d", exec.UserID)
	}

	if exec.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", exec.Status)
	}
}

func TestExecutorService_GetExecutionByID(t *testing.T) {
	db, sqlDB, cfg := setupTestDB(t)
	defer sqlDB.Close()

	appSvc := services.NewAppService(db)
	execSvc := services.NewExecutorService(db, cfg, appSvc)

	// Create test app and command
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:        "Test App",
		Description: "Test Description",
		WorkingDir:  "/tmp",
	})
	cmd, _ := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:           "test",
		Description:    "test command",
		Command:        "echo test",
		TimeoutSeconds: 30,
	})

	// Create execution
	exec, err := execSvc.CreateExecution(cmd.ID, 0)
	if err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	// Get execution by ID
	retrieved, err := execSvc.GetExecutionByID(exec.ID)
	if err != nil {
		t.Fatalf("failed to get execution: %v", err)
	}

	if retrieved.ID != exec.ID {
		t.Errorf("expected ID %q, got %q", exec.ID, retrieved.ID)
	}

	// Test non-existent execution
	_, err = execSvc.GetExecutionByID("nonexistent")
	if err != services.ErrExecutionNotFound {
		t.Errorf("expected ErrExecutionNotFound, got %v", err)
	}
}

func TestExecutorService_GetExecutions(t *testing.T) {
	db, sqlDB, cfg := setupTestDB(t)
	defer sqlDB.Close()

	appSvc := services.NewAppService(db)
	execSvc := services.NewExecutorService(db, cfg, appSvc)

	// Create test app and command
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:        "Test App",
		Description: "Test Description",
		WorkingDir:  "/tmp",
	})
	cmd, _ := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:           "test",
		Description:    "test command",
		Command:        "echo test",
		TimeoutSeconds: 30,
	})

	// Create executions with different user IDs
	exec1, _ := execSvc.CreateExecution(cmd.ID, 0) // API execution
	exec2, _ := execSvc.CreateExecution(cmd.ID, 1) // User execution (user doesn't exist)

	// Get all executions
	executions, err := execSvc.GetExecutions(50, 0)
	if err != nil {
		t.Fatalf("failed to get executions: %v", err)
	}

	if len(executions) != 2 {
		t.Errorf("expected 2 executions, got %d", len(executions))
	}

	// Verify API execution shows "API" as username
	foundAPI := false
	foundUser := false
	for _, ex := range executions {
		if ex.ID == exec1.ID {
			foundAPI = true
			if ex.Username != "API" {
				t.Errorf("expected username 'API' for exec1, got %q", ex.Username)
			}
		}
		if ex.ID == exec2.ID {
			foundUser = true
			// User with ID 1 doesn't exist, so LEFT JOIN should still work
			if ex.Username == "" {
				t.Error("expected username to be set for exec2")
			}
		}
	}

	if !foundAPI {
		t.Error("API execution not found in results")
	}
	if !foundUser {
		t.Error("user execution not found in results")
	}
}

func TestExecutorService_GetExecutions_Pagination(t *testing.T) {
	db, sqlDB, cfg := setupTestDB(t)
	defer sqlDB.Close()

	appSvc := services.NewAppService(db)
	execSvc := services.NewExecutorService(db, cfg, appSvc)

	// Create test app and command
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:        "Test App",
		Description: "Test Description",
		WorkingDir:  "/tmp",
	})
	cmd, _ := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:           "test",
		Description:    "test command",
		Command:        "echo test",
		TimeoutSeconds: 30,
	})

	// Create 10 executions
	for i := 0; i < 10; i++ {
		execSvc.CreateExecution(cmd.ID, 0)
	}

	// Test pagination
	page1, err := execSvc.GetExecutions(5, 0)
	if err != nil {
		t.Fatalf("failed to get page 1: %v", err)
	}
	if len(page1) != 5 {
		t.Errorf("expected 5 executions on page 1, got %d", len(page1))
	}

	page2, err := execSvc.GetExecutions(5, 5)
	if err != nil {
		t.Fatalf("failed to get page 2: %v", err)
	}
	if len(page2) != 5 {
		t.Errorf("expected 5 executions on page 2, got %d", len(page2))
	}

	// Verify no duplicates
	for _, e1 := range page1 {
		for _, e2 := range page2 {
			if e1.ID == e2.ID {
				t.Error("found duplicate execution across pages")
			}
		}
	}
}
