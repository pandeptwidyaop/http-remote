// Package models defines data models for applications, commands, and executions.
package models

import "time"

// App represents an application with its configuration.
type App struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	WorkingDir  string    `json:"working_dir"`
	Token       string    `json:"token,omitempty"`
}

// CreateAppRequest contains the data for creating a new application.
type CreateAppRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	WorkingDir  string `json:"working_dir" binding:"required"`
}

// UpdateAppRequest contains the data for updating an existing application.
type UpdateAppRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	WorkingDir  string `json:"working_dir"`
}
