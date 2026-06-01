-- Migration: 000002_settings_and_exports
-- Description: Add settings and exports tables for settings handler
-- Database: SQLite via modernc.org/sqlite (libsql compatible)

-- Settings table for storing JSON configuration
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- Export jobs tracking table
CREATE TABLE IF NOT EXISTS exports (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'preparing'
        CHECK (status IN ('preparing', 'ready', 'failed')),
    format TEXT NOT NULL DEFAULT 'json',
    file_path TEXT,
    size_bytes INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    expires_at DATETIME NOT NULL
);

CREATE INDEX idx_exports_status ON exports(status);
CREATE INDEX idx_exports_expires_at ON exports(expires_at);