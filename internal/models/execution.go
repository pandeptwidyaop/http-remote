package models

import "time"

type ExecutionStatus string

const (
	StatusPending ExecutionStatus = "pending"
	StatusRunning ExecutionStatus = "running"
	StatusSuccess ExecutionStatus = "success"
	StatusFailed  ExecutionStatus = "failed"
)

type Execution struct {
	ID         string          `json:"id"`
	CommandID  string          `json:"command_id"`
	UserID     int64           `json:"user_id"`
	Status     ExecutionStatus `json:"status"`
	Output     string          `json:"output"`
	ExitCode   *int            `json:"exit_code"`
	StartedAt  *time.Time      `json:"started_at"`
	FinishedAt *time.Time      `json:"finished_at"`
	CreatedAt  time.Time       `json:"created_at"`
}

type ExecutionWithDetails struct {
	Execution
	CommandName string `json:"command_name"`
	AppName     string `json:"app_name"`
	Username    string `json:"username"`
}
