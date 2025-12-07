package models

import "time"

// Command represents a command that can be executed for an application.
type Command struct {
	CreatedAt      time.Time `json:"created_at"`
	ID             string    `json:"id"`
	AppID          string    `json:"app_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Command        string    `json:"command"`
	TimeoutSeconds int       `json:"timeout_seconds"`
	SortOrder      int       `json:"sort_order"`
}

// CreateCommandRequest contains the data for creating a new command.
type CreateCommandRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	Command        string `json:"command" binding:"required"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// UpdateCommandRequest contains the data for updating an existing command.
type UpdateCommandRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Command        string `json:"command"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// ReorderCommandsRequest contains the data for reordering commands.
type ReorderCommandsRequest struct {
	CommandIDs []string `json:"command_ids" binding:"required"`
}
