package models

import "time"

// ExecutionStatus represents the status of a command execution.
type ExecutionStatus string

const (
	// StatusPending indicates the execution is waiting to start.
	StatusPending ExecutionStatus = "pending"
	// StatusRunning indicates the execution is currently running.
	StatusRunning ExecutionStatus = "running"
	// StatusSuccess indicates the execution completed successfully.
	StatusSuccess ExecutionStatus = "success"
	// StatusFailed indicates the execution failed.
	StatusFailed ExecutionStatus = "failed"
)

// Execution represents a command execution instance.
type Execution struct {
	CreatedAt  time.Time       `json:"created_at"`
	ExitCode   *int            `json:"exit_code"`
	StartedAt  *time.Time      `json:"started_at"`
	FinishedAt *time.Time      `json:"finished_at"`
	ID         string          `json:"id"`
	CommandID  string          `json:"command_id"`
	Status     ExecutionStatus `json:"status"`
	Output     string          `json:"output"`
	UserID     int64           `json:"user_id"`
}

// ExecutionWithDetails extends Execution with additional related information.
type ExecutionWithDetails struct {
	Execution
	CommandName string `json:"command_name"`
	AppName     string `json:"app_name"`
	Username    string `json:"username"`
}
