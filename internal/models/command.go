package models

import "time"

type Command struct {
	ID             string    `json:"id"`
	AppID          string    `json:"app_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Command        string    `json:"command"`
	TimeoutSeconds int       `json:"timeout_seconds"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateCommandRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	Command        string `json:"command" binding:"required"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type UpdateCommandRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Command        string `json:"command"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}
