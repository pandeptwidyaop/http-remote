// Package services provides business logic for application management.
package services

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/models"
)

var (
	// ErrAppNotFound indicates the requested app was not found.
	ErrAppNotFound = errors.New("app not found")
	// ErrAppExists indicates an app with the same name already exists.
	ErrAppExists = errors.New("app already exists")
	// ErrCommandNotFound indicates the requested command was not found.
	ErrCommandNotFound = errors.New("command not found")
	// ErrInvalidToken indicates the provided authentication token is invalid.
	ErrInvalidToken = errors.New("invalid token")
)

// AppService manages applications and their commands.
type AppService struct {
	db *database.DB
}

// NewAppService creates a new AppService instance.
func NewAppService(db *database.DB) *AppService {
	return &AppService{db: db}
}

// CreateApp creates a new application with a generated token.
func (s *AppService) CreateApp(req *models.CreateAppRequest) (*models.App, error) {
	id := uuid.New().String()
	token := uuid.New().String()

	_, err := s.db.Exec(
		"INSERT INTO apps (id, name, description, working_dir, token) VALUES (?, ?, ?, ?, ?)",
		id, req.Name, req.Description, req.WorkingDir, token,
	)
	if err != nil {
		return nil, ErrAppExists
	}

	return s.GetAppByID(id)
}

// GetAppByID retrieves an application by its ID.
func (s *AppService) GetAppByID(id string) (*models.App, error) {
	var app models.App
	err := s.db.QueryRow(
		"SELECT id, name, description, working_dir, token, created_at, updated_at FROM apps WHERE id = ?",
		id,
	).Scan(&app.ID, &app.Name, &app.Description, &app.WorkingDir, &app.Token, &app.CreatedAt, &app.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrAppNotFound
	}
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// GetAppByToken retrieves an application by its authentication token.
func (s *AppService) GetAppByToken(token string) (*models.App, error) {
	var app models.App
	err := s.db.QueryRow(
		"SELECT id, name, description, working_dir, token, created_at, updated_at FROM apps WHERE token = ?",
		token,
	).Scan(&app.ID, &app.Name, &app.Description, &app.WorkingDir, &app.Token, &app.CreatedAt, &app.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// GetAppByName retrieves an application by its name.
func (s *AppService) GetAppByName(name string) (*models.App, error) {
	var app models.App
	err := s.db.QueryRow(
		"SELECT id, name, description, working_dir, token, created_at, updated_at FROM apps WHERE name = ?",
		name,
	).Scan(&app.ID, &app.Name, &app.Description, &app.WorkingDir, &app.Token, &app.CreatedAt, &app.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrAppNotFound
	}
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// GetAllApps retrieves all applications ordered by name with command counts.
func (s *AppService) GetAllApps() ([]models.App, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.name, a.description, a.working_dir, a.token, a.created_at, a.updated_at,
		       (SELECT COUNT(*) FROM commands c WHERE c.app_id = a.id) as command_count
		FROM apps a
		ORDER BY a.name
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var apps []models.App
	for rows.Next() {
		var app models.App
		if err := rows.Scan(&app.ID, &app.Name, &app.Description, &app.WorkingDir, &app.Token, &app.CreatedAt, &app.UpdatedAt, &app.CommandCount); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, nil
}

// UpdateApp updates an existing application.
func (s *AppService) UpdateApp(id string, req *models.UpdateAppRequest) (*models.App, error) {
	app, err := s.GetAppByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		app.Name = req.Name
	}
	if req.Description != "" {
		app.Description = req.Description
	}
	if req.WorkingDir != "" {
		app.WorkingDir = req.WorkingDir
	}

	_, err = s.db.Exec(
		"UPDATE apps SET name = ?, description = ?, working_dir = ?, updated_at = ? WHERE id = ?",
		app.Name, app.Description, app.WorkingDir, time.Now(), id,
	)
	if err != nil {
		return nil, err
	}

	return s.GetAppByID(id)
}

// RegenerateToken generates a new authentication token for an application.
func (s *AppService) RegenerateToken(id string) (*models.App, error) {
	_, err := s.GetAppByID(id)
	if err != nil {
		return nil, err
	}

	newToken := uuid.New().String()
	_, err = s.db.Exec(
		"UPDATE apps SET token = ?, updated_at = ? WHERE id = ?",
		newToken, time.Now(), id,
	)
	if err != nil {
		return nil, err
	}

	return s.GetAppByID(id)
}

// DeleteApp deletes an application and all related commands and executions.
func (s *AppService) DeleteApp(id string) error {
	// First delete related executions
	_, err := s.db.Exec("DELETE FROM executions WHERE command_id IN (SELECT id FROM commands WHERE app_id = ?)", id)
	if err != nil {
		return err
	}

	// Then delete the app (commands will be deleted by CASCADE)
	result, err := s.db.Exec("DELETE FROM apps WHERE id = ?", id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrAppNotFound
	}
	return nil
}

// CreateCommand creates a new command for an application.
func (s *AppService) CreateCommand(appID string, req *models.CreateCommandRequest) (*models.Command, error) {
	if _, err := s.GetAppByID(appID); err != nil {
		return nil, err
	}

	id := uuid.New().String()
	timeout := req.TimeoutSeconds
	if timeout == 0 {
		timeout = 300
	}

	// Get the next sort_order for this app
	var maxOrder int
	err := s.db.QueryRow(
		"SELECT COALESCE(MAX(sort_order), -1) FROM commands WHERE app_id = ?",
		appID,
	).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}
	sortOrder := maxOrder + 1

	_, err = s.db.Exec(
		"INSERT INTO commands (id, app_id, name, description, command, timeout_seconds, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, appID, req.Name, req.Description, req.Command, timeout, sortOrder,
	)
	if err != nil {
		return nil, err
	}

	return s.GetCommandByID(id)
}

// GetCommandByID retrieves a command by its ID.
func (s *AppService) GetCommandByID(id string) (*models.Command, error) {
	var cmd models.Command
	err := s.db.QueryRow(
		"SELECT id, app_id, name, description, command, timeout_seconds, created_at, COALESCE(sort_order, 0) FROM commands WHERE id = ?",
		id,
	).Scan(&cmd.ID, &cmd.AppID, &cmd.Name, &cmd.Description, &cmd.Command, &cmd.TimeoutSeconds, &cmd.CreatedAt, &cmd.SortOrder)

	if err == sql.ErrNoRows {
		return nil, ErrCommandNotFound
	}
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

// GetCommandsByAppID retrieves all commands for a specific application ordered by sort_order.
func (s *AppService) GetCommandsByAppID(appID string) ([]models.Command, error) {
	rows, err := s.db.Query(
		"SELECT id, app_id, name, description, command, timeout_seconds, created_at, COALESCE(sort_order, 0) FROM commands WHERE app_id = ? ORDER BY sort_order, created_at",
		appID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var commands []models.Command
	for rows.Next() {
		var cmd models.Command
		if err := rows.Scan(&cmd.ID, &cmd.AppID, &cmd.Name, &cmd.Description, &cmd.Command, &cmd.TimeoutSeconds, &cmd.CreatedAt, &cmd.SortOrder); err != nil {
			return nil, err
		}
		commands = append(commands, cmd)
	}
	return commands, nil
}

// GetDefaultCommandByAppID retrieves the default (first by sort_order) command for an application.
func (s *AppService) GetDefaultCommandByAppID(appID string) (*models.Command, error) {
	var cmd models.Command
	err := s.db.QueryRow(
		"SELECT id, app_id, name, description, command, timeout_seconds, created_at, COALESCE(sort_order, 0) FROM commands WHERE app_id = ? ORDER BY sort_order, created_at LIMIT 1",
		appID,
	).Scan(&cmd.ID, &cmd.AppID, &cmd.Name, &cmd.Description, &cmd.Command, &cmd.TimeoutSeconds, &cmd.CreatedAt, &cmd.SortOrder)

	if err == sql.ErrNoRows {
		return nil, ErrCommandNotFound
	}
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

// UpdateCommand updates an existing command.
func (s *AppService) UpdateCommand(id string, req *models.UpdateCommandRequest) (*models.Command, error) {
	cmd, err := s.GetCommandByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		cmd.Name = req.Name
	}
	if req.Description != "" {
		cmd.Description = req.Description
	}
	if req.Command != "" {
		cmd.Command = req.Command
	}
	if req.TimeoutSeconds > 0 {
		cmd.TimeoutSeconds = req.TimeoutSeconds
	}

	_, err = s.db.Exec(
		"UPDATE commands SET name = ?, description = ?, command = ?, timeout_seconds = ? WHERE id = ?",
		cmd.Name, cmd.Description, cmd.Command, cmd.TimeoutSeconds, id,
	)
	if err != nil {
		return nil, err
	}

	return s.GetCommandByID(id)
}

// DeleteCommand deletes a command and all related executions.
func (s *AppService) DeleteCommand(id string) error {
	// First delete related executions
	_, err := s.db.Exec("DELETE FROM executions WHERE command_id = ?", id)
	if err != nil {
		return err
	}

	// Then delete the command
	result, err := s.db.Exec("DELETE FROM commands WHERE id = ?", id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrCommandNotFound
	}
	return nil
}

// ReorderCommands updates the sort_order of commands based on the provided order.
func (s *AppService) ReorderCommands(appID string, commandIDs []string) error {
	// Verify app exists
	if _, err := s.GetAppByID(appID); err != nil {
		return err
	}

	// Begin transaction
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Update sort_order for each command
	for i, cmdID := range commandIDs {
		result, err := tx.Exec(
			"UPDATE commands SET sort_order = ? WHERE id = ? AND app_id = ?",
			i, cmdID, appID,
		)
		if err != nil {
			return err
		}

		// Verify the command belongs to this app
		rows, _ := result.RowsAffected()
		if rows == 0 {
			return ErrCommandNotFound
		}
	}

	return tx.Commit()
}
