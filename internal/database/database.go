// Package database provides SQLite database access and migration management.
package database

import (
	"database/sql"
	"os"
	"path/filepath"

	// SQLite driver for database/sql
	_ "github.com/mattn/go-sqlite3"
)

// DB wraps a sql.DB connection with additional functionality.
type DB struct {
	*sql.DB
}

// New creates a new database connection and ensures the parent directory exists.
func New(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

// Migrate runs all database migrations.
func (db *DB) Migrate() error {
	return runMigrations(db.DB)
}
