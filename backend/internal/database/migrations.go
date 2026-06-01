package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

// RunMigrations runs all pending database migrations.
func RunMigrations(db *sqlx.DB) error {
	migrationsDir := os.Getenv("PANEL_MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrationFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Must be at least 10 chars (e.g., 000001.up.sql) and end with .up.sql
		if len(name) >= 10 && len(name) >= 7 && name[len(name)-7:] == ".up.sql" {
			migrationFiles = append(migrationFiles, name)
		}
	}

	sort.Slice(migrationFiles, func(i, j int) bool {
		return strings.Compare(migrationFiles[i][:6], migrationFiles[j][:6]) < 0
	})

	ctx := context.Background()

	// Check if this is a fresh database (no schema_migrations table)
	currentVersion := 0
	var tableCount int
	err = db.GetContext(ctx, &tableCount, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'")
	if err == nil && tableCount > 0 {
		_ = db.GetContext(ctx, &currentVersion, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations WHERE dirty = 0")
	}

	for _, filename := range migrationFiles {
		versionStr := filename[:6]
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return fmt.Errorf("invalid migration version in filename %s: %w", filename, err)
		}

		if version <= currentVersion {
			continue
		}

		migrationPath := filepath.Join(migrationsDir, filename)
		migrationSQL, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migrationPath, err)
		}

		if _, err := db.ExecContext(ctx, string(migrationSQL)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		// Only record if schema_migrations table exists
		var count int
		_ = db.GetContext(ctx, &count, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'")
		if count > 0 {
			_, _ = db.ExecContext(ctx, "INSERT INTO schema_migrations (version, dirty) VALUES (?, 0)", version)
		}
	}

	return nil
}

// GetAppliedMigrations returns all applied migration versions.
func GetAppliedMigrations(db *sqlx.DB) ([]int, error) {
	ctx := context.Background()

	var tableExists int
	err := db.GetContext(ctx, &tableExists, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'")
	if err != nil || tableExists == 0 {
		return []int{}, nil
	}

	var versions []int
	err = db.SelectContext(ctx, &versions, "SELECT version FROM schema_migrations WHERE dirty = 0 ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	return versions, nil
}
