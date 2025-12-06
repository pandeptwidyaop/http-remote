package models

// AppBackup represents an app with its commands for backup/export
type AppBackup struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	WorkingDir  string          `json:"working_dir"`
	Commands    []CommandBackup `json:"commands"`
}

// CommandBackup represents a command for backup/export (without IDs)
type CommandBackup struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Command        string `json:"command"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// BackupData represents the full backup structure
type BackupData struct {
	Version    string      `json:"version"`
	ExportedAt string      `json:"exported_at"`
	Apps       []AppBackup `json:"apps"`
}
