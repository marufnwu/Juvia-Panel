ALTER TABLE apps ADD COLUMN compose_config TEXT;
ALTER TABLE apps ADD COLUMN compose_project TEXT;

CREATE TABLE compose_services (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL,
    service_name TEXT NOT NULL,
    container_id TEXT,
    image TEXT,
    internal_port INTEGER,
    external_port INTEGER,
    status TEXT DEFAULT 'stopped'
        CHECK (status IN ('running', 'stopped', 'failed', 'restarting')),
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE INDEX idx_compose_services_app_id ON compose_services(app_id);