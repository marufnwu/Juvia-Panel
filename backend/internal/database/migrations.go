package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

// RunMigrations runs all pending database migrations.
// Tries multiple sources in order:
//   1. PANEL_MIGRATIONS_DIR env var (filesystem)
//   2. Embedded migrations compiled into the binary
//   3. Filesystem fallback to ./migrations (relative to working dir)
func RunMigrations(db *sqlx.DB) error {
	files, err := loadMigrations()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		// No migrations found anywhere - this is OK for a fresh install
		// where migrations may be applied separately by the install script
		return nil
	}

	sort.Slice(files, func(i, j int) bool {
		return strings.Compare(files[i].name[:6], files[j].name[:6]) < 0
	})

	ctx := context.Background()

	// Check if this is a fresh database (no schema_migrations table)
	currentVersion := 0
	var tableCount int
	err = db.GetContext(ctx, &tableCount, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'")
	if err == nil && tableCount > 0 {
		_ = db.GetContext(ctx, &currentVersion, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations WHERE dirty = 0")
	}

	for _, mf := range files {
		versionStr := mf.name[:6]
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return fmt.Errorf("invalid migration version in filename %s: %w", mf.name, err)
		}

		if version <= currentVersion {
			continue
		}

		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", mf.name, err)
		}

		if _, err := tx.ExecContext(ctx, string(mf.data)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", mf.name, err)
		}

		var count int
		err = tx.GetContext(ctx, &count, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'")
		if err == nil && count > 0 {
			_, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations (version, dirty) VALUES (?, 0)", version)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to record migration %s: %w", mf.name, err)
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", mf.name, err)
		}
	}

	return nil
}

type migrationFile struct {
	name string
	data []byte
}

// loadMigrations loads migrations from env-dir, then embedded, then filesystem fallback.
func loadMigrations() ([]migrationFile, error) {
	// 1. Try PANEL_MIGRATIONS_DIR from env (set in systemd unit)
	if dir := os.Getenv("PANEL_MIGRATIONS_DIR"); dir != "" {
		if files, err := loadFromDir(dir); err == nil && len(files) > 0 {
			return files, nil
		}
	}

	// 2. Try embedded migrations
	if files, err := loadFromEmbedded(); err == nil && len(files) > 0 {
		return files, nil
	}

	// 3. Filesystem fallback (./migrations)
	if files, err := loadFromDir("migrations"); err == nil {
		return files, nil
	}

	return nil, nil
}

func loadFromDir(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []migrationFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) >= 10 && name[len(name)-7:] == ".up.sql" {
			data, err := os.ReadFile(dir + "/" + name)
			if err != nil {
				return nil, fmt.Errorf("failed to read %s: %w", name, err)
			}
			files = append(files, migrationFile{name: name, data: data})
		}
	}
	return files, nil
}

func loadFromEmbedded() ([]migrationFile, error) {
	entries, err := fs.ReadDir(embeddedMigrations, "migrations")
	if err != nil {
		return nil, err
	}
	var files []migrationFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) >= 10 && name[len(name)-7:] == ".up.sql" {
			data, err := embeddedMigrations.ReadFile("migrations/" + name)
			if err != nil {
				return nil, err
			}
			files = append(files, migrationFile{name: name, data: data})
		}
	}
	return files, nil
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
