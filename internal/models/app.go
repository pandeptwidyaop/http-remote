// Package models defines data models for applications, commands, and executions.
package models

import "time"

type App struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	WorkingDir  string    `json:"working_dir"`
	Token       string    `json:"token,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateAppRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	WorkingDir  string `json:"working_dir" binding:"required"`
}

type UpdateAppRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	WorkingDir  string `json:"working_dir"`
}
