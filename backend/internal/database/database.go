package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"panel-api/internal/config"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// DB wraps the sqlx database connection for use in handlers
type DB = sqlx.DB

// New creates a new database connection and ensures the database file and directory exist.
func New(cfg *config.Config) (*sqlx.DB, error) {
	// Ensure the data directory exists
	dataDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}

	// Connect to SQLite database
	db, err := sqlx.Connect("sqlite", cfg.DBPath+"?_journal_mode=WAL&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database at %s: %w", cfg.DBPath, err)
	}

	// Set busy_timeout via PRAGMA since connection string may not work with all drivers
	// This controls how long SQLite waits for a lock (5 seconds)
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		return nil, fmt.Errorf("failed to set busy_timeout: %w", err)
	}

	// Configure connection pool for SQLite
	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func Close(db *sqlx.DB) error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// DefaultQueryTimeout is the default timeout for database queries
const DefaultQueryTimeout = 10 * time.Second

// WithQueryTimeout wraps a context with a default query timeout.
// Use this for all database operations to prevent hanging queries.
func WithQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, DefaultQueryTimeout)
}
