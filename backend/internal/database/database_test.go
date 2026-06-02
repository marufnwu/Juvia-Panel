package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"panel-api/internal/config"
)

func TestNew(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &config.Config{
		DBPath: dbPath,
	}

	// Test database creation
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Verify connection works
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
}

func findMigrationsDir() string {
	// Try to find migrations directory by checking various paths
	candidates := []string{
		"migrations",
		"../migrations",
		"../../migrations",
		"../../../migrations",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			absPath, err := filepath.Abs(candidate)
			if err == nil {
				return absPath
			}
			return candidate
		}
	}
	return ""
}

func TestRunMigrations(t *testing.T) {
	// Find migrations directory
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		t.Skip("Migrations directory not found, skipping test")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &config.Config{
		DBPath: dbPath,
	}

	// Create database
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	t.Logf("Using migrations directory: %s", migrationsDir)

	os.Setenv("PANEL_MIGRATIONS_DIR", migrationsDir)

	// Run migrations
	if err := RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify tables exist
	tables := []string{
		"users",
		"sessions",
		"api_keys",
		"apps",
		"app_domains",
		"app_env_vars",
		"app_volumes",
		"deployments",
		"services",
		"service_app_links",
		"backups",
		"server_info",
		"firewall_rules",
		"cron_jobs",
		"cron_executions",
		"user_invites",
		"activity_log",
		"notifications",
		"schema_migrations",
	}

	ctx := context.Background()
	for _, table := range tables {
		var count int
		err := db.GetContext(ctx, &count, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table)
		if err != nil {
			t.Errorf("Failed to check table %s: %v", table, err)
			continue
		}
		t.Logf("Table %s: count=%d", table, count)
		if count != 1 {
			t.Errorf("Table %s does not exist (count=%d)", table, count)
		}
	}
}

func TestGetAppliedMigrations(t *testing.T) {
	// Find migrations directory
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		t.Skip("Migrations directory not found, skipping test")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &config.Config{
		DBPath: dbPath,
	}

	// Create database
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	os.Setenv("PANEL_MIGRATIONS_DIR", migrationsDir)

	// Run migrations first
	if err := RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Get applied migrations
	versions, err := GetAppliedMigrations(db)
	if err != nil {
		t.Fatalf("Failed to get applied migrations: %v", err)
	}

	if len(versions) == 0 {
		t.Error("Expected at least one migration to be applied")
	}
}