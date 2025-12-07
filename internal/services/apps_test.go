package services_test

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

func setupAppTestDB(t *testing.T) (*database.DB, *sql.DB) {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	db := &database.DB{DB: sqlDB}

	// Create apps table
	_, err = sqlDB.Exec(`
		CREATE TABLE apps (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			working_dir TEXT NOT NULL,
			token TEXT UNIQUE NOT NULL,
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
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
		);

		CREATE INDEX idx_commands_sort_order ON commands(app_id, sort_order);

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
			FOREIGN KEY (command_id) REFERENCES commands(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	return db, sqlDB
}

func TestAppService_CreateApp(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	app, err := appSvc.CreateApp(&models.CreateAppRequest{
		Name:        "Test App",
		Description: "Test Description",
		WorkingDir:  "/tmp/test",
	})

	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	if app.ID == "" {
		t.Error("expected app ID to be set")
	}

	if app.Token == "" {
		t.Error("expected token to be generated")
	}

	if app.Name != "Test App" {
		t.Errorf("expected name 'Test App', got %q", app.Name)
	}

	if app.WorkingDir != "/tmp/test" {
		t.Errorf("expected working_dir '/tmp/test', got %q", app.WorkingDir)
	}
}

func TestAppService_CreateApp_DuplicateName(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create first app
	_, err := appSvc.CreateApp(&models.CreateAppRequest{
		Name:       "Duplicate App",
		WorkingDir: "/tmp/test",
	})
	if err != nil {
		t.Fatalf("failed to create first app: %v", err)
	}

	// Try to create app with same name
	_, err = appSvc.CreateApp(&models.CreateAppRequest{
		Name:       "Duplicate App",
		WorkingDir: "/tmp/test",
	})

	if err != services.ErrAppExists {
		t.Errorf("expected ErrAppExists, got %v", err)
	}
}

func TestAppService_GetAppByID(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create app
	created, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:       "Test App",
		WorkingDir: "/tmp/test",
	})

	// Get app by ID
	app, err := appSvc.GetAppByID(created.ID)
	if err != nil {
		t.Fatalf("failed to get app: %v", err)
	}

	if app.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, app.ID)
	}

	if app.Name != created.Name {
		t.Errorf("expected name %q, got %q", created.Name, app.Name)
	}

	// Test non-existent app
	_, err = appSvc.GetAppByID("nonexistent")
	if err != services.ErrAppNotFound {
		t.Errorf("expected ErrAppNotFound, got %v", err)
	}
}

func TestAppService_GetAllApps(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create multiple apps
	_, _ = appSvc.CreateApp(&models.CreateAppRequest{Name: "App 1", WorkingDir: "/tmp/1"})
	_, _ = appSvc.CreateApp(&models.CreateAppRequest{Name: "App 2", WorkingDir: "/tmp/2"})
	_, _ = appSvc.CreateApp(&models.CreateAppRequest{Name: "App 3", WorkingDir: "/tmp/3"})

	apps, err := appSvc.GetAllApps()
	if err != nil {
		t.Fatalf("failed to get all apps: %v", err)
	}

	if len(apps) != 3 {
		t.Errorf("expected 3 apps, got %d", len(apps))
	}
}

func TestAppService_UpdateApp(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create app
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:        "Original Name",
		Description: "Original Description",
		WorkingDir:  "/tmp/original",
	})

	// Update app
	updated, err := appSvc.UpdateApp(app.ID, &models.UpdateAppRequest{
		Name:        "Updated Name",
		Description: "Updated Description",
		WorkingDir:  "/tmp/updated",
	})
	if err != nil {
		t.Fatalf("failed to update app: %v", err)
	}

	// Verify update
	if updated.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %q", updated.Name)
	}
	if updated.Description != "Updated Description" {
		t.Errorf("expected description 'Updated Description', got %q", updated.Description)
	}
	if updated.WorkingDir != "/tmp/updated" {
		t.Errorf("expected working_dir '/tmp/updated', got %q", updated.WorkingDir)
	}
}

func TestAppService_DeleteApp(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create app
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:       "To Delete",
		WorkingDir: "/tmp/delete",
	})

	// Delete app
	err := appSvc.DeleteApp(app.ID)
	if err != nil {
		t.Fatalf("failed to delete app: %v", err)
	}

	// Verify deletion
	_, err = appSvc.GetAppByID(app.ID)
	if err != services.ErrAppNotFound {
		t.Errorf("expected ErrAppNotFound after deletion, got %v", err)
	}

	// Try to delete non-existent app
	err = appSvc.DeleteApp("nonexistent")
	if err != services.ErrAppNotFound {
		t.Errorf("expected ErrAppNotFound for non-existent app, got %v", err)
	}
}

func TestAppService_RegenerateToken(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create app
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:       "Test App",
		WorkingDir: "/tmp/test",
	})

	originalToken := app.Token

	// Regenerate token
	updatedApp, err := appSvc.RegenerateToken(app.ID)
	if err != nil {
		t.Fatalf("failed to regenerate token: %v", err)
	}

	if updatedApp.Token == originalToken {
		t.Error("expected new token to be different from original")
	}

	// Verify token in database
	retrieved, _ := appSvc.GetAppByID(app.ID)
	if retrieved.Token != updatedApp.Token {
		t.Errorf("expected token %q, got %q", updatedApp.Token, retrieved.Token)
	}
}

func TestAppService_CreateCommand(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create app
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:       "Test App",
		WorkingDir: "/tmp/test",
	})

	// Create command
	cmd, err := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:           "deploy",
		Description:    "Deploy command",
		Command:        "git pull && npm install",
		TimeoutSeconds: 600,
	})

	if err != nil {
		t.Fatalf("failed to create command: %v", err)
	}

	if cmd.ID == "" {
		t.Error("expected command ID to be set")
	}

	if cmd.AppID != app.ID {
		t.Errorf("expected app_id %q, got %q", app.ID, cmd.AppID)
	}

	if cmd.Name != "deploy" {
		t.Errorf("expected name 'deploy', got %q", cmd.Name)
	}

	if cmd.TimeoutSeconds != 600 {
		t.Errorf("expected timeout 600, got %d", cmd.TimeoutSeconds)
	}
}

func TestAppService_ListCommands(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create app
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:       "Test App",
		WorkingDir: "/tmp/test",
	})

	// Create multiple commands
	_, _ = appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:    "deploy",
		Command: "echo deploy",
	})
	_, _ = appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:    "build",
		Command: "echo build",
	})

	// List commands
	commands, err := appSvc.GetCommandsByAppID(app.ID)
	if err != nil {
		t.Fatalf("failed to list commands: %v", err)
	}

	if len(commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(commands))
	}
}

func TestAppService_DeleteCommand(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create app and command
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:       "Test App",
		WorkingDir: "/tmp/test",
	})
	cmd, _ := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:    "deploy",
		Command: "echo deploy",
	})

	// Delete command
	err := appSvc.DeleteCommand(cmd.ID)
	if err != nil {
		t.Fatalf("failed to delete command: %v", err)
	}

	// Verify deletion
	_, err = appSvc.GetCommandByID(cmd.ID)
	if err != services.ErrCommandNotFound {
		t.Errorf("expected ErrCommandNotFound after deletion, got %v", err)
	}
}

func TestAppService_ReorderCommands(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create app
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:       "Test App",
		WorkingDir: "/tmp/test",
	})

	// Create 3 commands
	cmd1, _ := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:    "first",
		Command: "echo first",
	})
	cmd2, _ := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:    "second",
		Command: "echo second",
	})
	cmd3, _ := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:    "third",
		Command: "echo third",
	})

	// Verify initial order
	commands, _ := appSvc.GetCommandsByAppID(app.ID)
	if len(commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(commands))
	}
	if commands[0].ID != cmd1.ID {
		t.Errorf("expected first command to be cmd1")
	}
	if commands[1].ID != cmd2.ID {
		t.Errorf("expected second command to be cmd2")
	}
	if commands[2].ID != cmd3.ID {
		t.Errorf("expected third command to be cmd3")
	}

	// Reorder: move third to first position
	newOrder := []string{cmd3.ID, cmd1.ID, cmd2.ID}
	err := appSvc.ReorderCommands(app.ID, newOrder)
	if err != nil {
		t.Fatalf("failed to reorder commands: %v", err)
	}

	// Verify new order
	commands, _ = appSvc.GetCommandsByAppID(app.ID)
	if commands[0].ID != cmd3.ID {
		t.Errorf("expected first command to be cmd3, got %s", commands[0].Name)
	}
	if commands[1].ID != cmd1.ID {
		t.Errorf("expected second command to be cmd1, got %s", commands[1].Name)
	}
	if commands[2].ID != cmd2.ID {
		t.Errorf("expected third command to be cmd2, got %s", commands[2].Name)
	}

	// Verify sort_order values
	if commands[0].SortOrder != 0 {
		t.Errorf("expected first command sort_order to be 0, got %d", commands[0].SortOrder)
	}
	if commands[1].SortOrder != 1 {
		t.Errorf("expected second command sort_order to be 1, got %d", commands[1].SortOrder)
	}
	if commands[2].SortOrder != 2 {
		t.Errorf("expected third command sort_order to be 2, got %d", commands[2].SortOrder)
	}
}

func TestAppService_ReorderCommands_InvalidApp(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	err := appSvc.ReorderCommands("nonexistent-app", []string{"cmd1", "cmd2"})
	if err != services.ErrAppNotFound {
		t.Errorf("expected ErrAppNotFound, got %v", err)
	}
}

func TestAppService_ReorderCommands_InvalidCommand(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create app
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:       "Test App",
		WorkingDir: "/tmp/test",
	})

	// Create one command
	cmd, _ := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:    "test",
		Command: "echo test",
	})

	// Try to reorder with invalid command ID
	err := appSvc.ReorderCommands(app.ID, []string{cmd.ID, "invalid-cmd-id"})
	if err != services.ErrCommandNotFound {
		t.Errorf("expected ErrCommandNotFound for invalid command, got %v", err)
	}
}

func TestAppService_CreateCommand_SortOrder(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create app
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:       "Test App",
		WorkingDir: "/tmp/test",
	})

	// Create commands and verify sort_order increments
	cmd1, err := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:    "first",
		Command: "echo first",
	})
	if err != nil {
		t.Fatalf("failed to create first command: %v", err)
	}
	if cmd1.SortOrder != 0 {
		t.Errorf("expected first command sort_order to be 0, got %d", cmd1.SortOrder)
	}

	cmd2, err := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:    "second",
		Command: "echo second",
	})
	if err != nil {
		t.Fatalf("failed to create second command: %v", err)
	}
	if cmd2.SortOrder != 1 {
		t.Errorf("expected second command sort_order to be 1, got %d", cmd2.SortOrder)
	}

	cmd3, err := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:    "third",
		Command: "echo third",
	})
	if err != nil {
		t.Fatalf("failed to create third command: %v", err)
	}
	if cmd3.SortOrder != 2 {
		t.Errorf("expected third command sort_order to be 2, got %d", cmd3.SortOrder)
	}
}

func TestAppService_GetDefaultCommand_UsesSortOrder(t *testing.T) {
	db, sqlDB := setupAppTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	appSvc := services.NewAppService(db)

	// Create app
	app, _ := appSvc.CreateApp(&models.CreateAppRequest{
		Name:       "Test App",
		WorkingDir: "/tmp/test",
	})

	// Create commands in order
	cmd1, _ := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:    "first",
		Command: "echo first",
	})
	cmd2, _ := appSvc.CreateCommand(app.ID, &models.CreateCommandRequest{
		Name:    "second",
		Command: "echo second",
	})

	// Default should be first command
	defaultCmd, err := appSvc.GetDefaultCommandByAppID(app.ID)
	if err != nil {
		t.Fatalf("failed to get default command: %v", err)
	}
	if defaultCmd.ID != cmd1.ID {
		t.Errorf("expected default to be cmd1")
	}

	// Reorder: make cmd2 first
	err = appSvc.ReorderCommands(app.ID, []string{cmd2.ID, cmd1.ID})
	if err != nil {
		t.Fatalf("failed to reorder: %v", err)
	}

	// Now default should be cmd2
	defaultCmd, err = appSvc.GetDefaultCommandByAppID(app.ID)
	if err != nil {
		t.Fatalf("failed to get default command: %v", err)
	}
	if defaultCmd.ID != cmd2.ID {
		t.Errorf("expected default to be cmd2 after reorder")
	}
}
