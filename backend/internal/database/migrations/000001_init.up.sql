-- Migration: 000001_init
-- Description: Initial database schema for Server Panel
-- Database: SQLite via modernc.org/sqlite (libsql compatible)
-- File: /var/panel/panel.db

-- Enable foreign keys (required for SQLite)
PRAGMA foreign_keys = ON;

-- =====================================================
-- CORE TABLES
-- =====================================================

-- Schema migrations (auto-managed by golang-migrate)
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    dirty BOOLEAN NOT NULL DEFAULT 0
);

-- Users table (Panel users and team members)
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'viewer'
        CHECK (role IN ('owner', 'admin', 'developer', 'viewer')),
    two_factor_secret TEXT,
    two_factor_enabled BOOLEAN NOT NULL DEFAULT 0,
    two_factor_backup_codes TEXT,
    avatar_url TEXT,
    last_login_at DATETIME,
    last_login_ip TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);

-- API keys for programmatic access
CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    scopes TEXT NOT NULL,
    last_used_at DATETIME,
    expires_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    revoked_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_token_hash ON api_keys(token_hash);

-- Active user sessions for logout-everywhere and session management
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    refresh_token_hash TEXT NOT NULL UNIQUE,
    ip_address TEXT,
    user_agent TEXT,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- =====================================================
-- APP TABLES
-- =====================================================

-- Deployed applications
CREATE TABLE apps (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'stopped'
        CHECK (status IN ('running', 'stopped', 'failed', 'deploying', 'restarting')),
    health_status TEXT DEFAULT 'unknown'
        CHECK (health_status IN ('healthy', 'unhealthy', 'unknown')),
    runtime TEXT NOT NULL
        CHECK (runtime IN ('nodejs', 'python', 'go', 'php', 'ruby', 'static', 'docker')),
    runtime_version TEXT,

    source_type TEXT NOT NULL DEFAULT 'git'
        CHECK (source_type IN ('git', 'upload', 'docker_compose')),
    source_config TEXT NOT NULL,

    build_strategy TEXT NOT NULL DEFAULT 'nixpacks'
        CHECK (build_strategy IN ('nixpacks', 'dockerfile', 'static', 'custom')),
    build_config TEXT,

    container_id TEXT,
    container_image TEXT,
    internal_port INTEGER DEFAULT 3000,
    restart_policy TEXT DEFAULT 'unless-stopped'
        CHECK (restart_policy IN ('always', 'on-failure', 'unless-stopped')),

    health_check_path TEXT DEFAULT '/health',
    health_check_interval INTEGER DEFAULT 30,
    health_check_timeout INTEGER DEFAULT 5,
    health_check_retries INTEGER DEFAULT 3,

    cpu_limit REAL,
    memory_limit_mb INTEGER,
    memory_swap_mb INTEGER,

    created_by INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_apps_status ON apps(status);
CREATE INDEX idx_apps_runtime ON apps(runtime);
CREATE INDEX idx_apps_created_by ON apps(created_by);
CREATE INDEX idx_apps_updated_at ON apps(updated_at);
CREATE INDEX idx_apps_status_created ON apps(status, created_at);

-- Domains attached to apps
CREATE TABLE app_domains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    app_id TEXT NOT NULL,
    domain TEXT NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT 0,
    force_https BOOLEAN NOT NULL DEFAULT 1,
    ssl_status TEXT DEFAULT 'pending'
        CHECK (ssl_status IN ('pending', 'valid', 'expiring', 'failed')),
    ssl_provider TEXT,
    ssl_cert_path TEXT,
    ssl_key_path TEXT,
    ssl_issued_at DATETIME,
    ssl_expires_at DATETIME,
    ssl_auto_renew BOOLEAN NOT NULL DEFAULT 1,
    dns_valid BOOLEAN DEFAULT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE,
    UNIQUE(app_id, domain)
);

CREATE INDEX idx_app_domains_app_id ON app_domains(app_id);
CREATE INDEX idx_app_domains_domain ON app_domains(domain);
CREATE INDEX idx_app_domains_ssl_expires ON app_domains(ssl_expires_at)
    WHERE ssl_status = 'valid' AND ssl_auto_renew = 1;

-- Environment variables for apps
CREATE TABLE app_env_vars (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    app_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    is_secret BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE,
    UNIQUE(app_id, key)
);

CREATE INDEX idx_app_env_vars_app_id ON app_env_vars(app_id);

-- Persistent storage volumes mounted into app containers
CREATE TABLE app_volumes (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL,
    name TEXT NOT NULL,
    host_path TEXT NOT NULL,
    container_path TEXT NOT NULL,
    size_mb INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE INDEX idx_app_volumes_app_id ON app_volumes(app_id);

-- Deployment history for apps
CREATE TABLE deployments (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'in_progress', 'success', 'failed', 'cancelled')),

    commit_sha TEXT,
    commit_message TEXT,
    commit_author TEXT,
    branch TEXT,

    build_strategy TEXT,
    build_logs TEXT,
    build_duration_seconds INTEGER,
    deploy_duration_seconds INTEGER,

    triggered_by TEXT NOT NULL DEFAULT 'manual'
        CHECK (triggered_by IN ('manual', 'git_push', 'webhook', 'rollback', 'api')),
    triggered_by_user_id INTEGER,

    rollback_of_id TEXT,

    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE,
    FOREIGN KEY (triggered_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (rollback_of_id) REFERENCES deployments(id) ON DELETE SET NULL
);

CREATE INDEX idx_deployments_app_id ON deployments(app_id);
CREATE INDEX idx_deployments_status ON deployments(status);
CREATE INDEX idx_deployments_created_at ON deployments(created_at);
CREATE INDEX idx_deployments_app_status ON deployments(app_id, status);
CREATE INDEX idx_deployments_app_created ON deployments(app_id, created_at DESC);

-- =====================================================
-- SERVICE TABLES
-- =====================================================

-- Managed databases, caches, and other backing services
CREATE TABLE services (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL
        CHECK (type IN ('postgresql', 'mysql', 'mariadb', 'mongodb', 'redis', 'memcached', 'minio', 'custom')),
    version TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'stopped'
        CHECK (status IN ('running', 'stopped', 'failed', 'creating', 'restarting')),

    internal_port INTEGER NOT NULL,
    internal_host TEXT NOT NULL,
    container_id TEXT,
    container_image TEXT,

    credentials TEXT NOT NULL,

    memory_limit_mb INTEGER,
    cpu_limit REAL,

    data_path TEXT NOT NULL,
    data_size_mb INTEGER DEFAULT 0,

    backup_enabled BOOLEAN NOT NULL DEFAULT 1,
    backup_frequency TEXT DEFAULT 'daily'
        CHECK (backup_frequency IN ('hourly', 'daily', 'weekly')),
    backup_time TEXT DEFAULT '02:00',
    backup_retention_days INTEGER DEFAULT 7,
    backup_destination TEXT DEFAULT 'local'
        CHECK (backup_destination IN ('local', 's3')),

    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_services_type ON services(type);
CREATE INDEX idx_services_status ON services(status);
CREATE INDEX idx_services_type_status ON services(type, status);

-- Many-to-many relationship between services and apps
CREATE TABLE service_app_links (
    service_id TEXT NOT NULL,
    app_id TEXT NOT NULL,
    connection_env_key TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),

    PRIMARY KEY (service_id, app_id),
    FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE,
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE INDEX idx_service_app_links_app_id ON service_app_links(app_id);

-- Backup records for apps and services
CREATE TABLE backups (
    id TEXT PRIMARY KEY,
    target_type TEXT NOT NULL
        CHECK (target_type IN ('app', 'service')),
    target_id TEXT NOT NULL,
    target_name TEXT NOT NULL,

    status TEXT NOT NULL DEFAULT 'in_progress'
        CHECK (status IN ('in_progress', 'success', 'failed')),
    size_mb INTEGER,

    destination TEXT NOT NULL
        CHECK (destination IN ('local', 's3')),
    destination_path TEXT NOT NULL,

    checksum TEXT,
    checksum_algorithm TEXT DEFAULT 'sha256',

    triggered_by TEXT DEFAULT 'schedule'
        CHECK (triggered_by IN ('schedule', 'manual', 'api')),
    triggered_by_user_id INTEGER,

    started_at DATETIME NOT NULL DEFAULT (datetime('now')),
    completed_at DATETIME,

    FOREIGN KEY (triggered_by_user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_backups_target ON backups(target_type, target_id);
CREATE INDEX idx_backups_status ON backups(status);
CREATE INDEX idx_backups_started_at ON backups(started_at);
CREATE INDEX idx_backups_target_started ON backups(target_type, target_id, started_at DESC);

-- =====================================================
-- SERVER & SYSTEM TABLES
-- =====================================================

-- Single-row table with server configuration
CREATE TABLE server_info (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    hostname TEXT NOT NULL,
    os TEXT NOT NULL,
    kernel TEXT,
    architecture TEXT NOT NULL DEFAULT 'amd64',
    timezone TEXT NOT NULL DEFAULT 'UTC',

    cpu_cores INTEGER,
    cpu_model TEXT,
    memory_total_mb INTEGER,
    disk_total_gb INTEGER,

    panel_version TEXT NOT NULL DEFAULT '1.0.0',
    panel_domain TEXT,
    default_app_subdomain TEXT DEFAULT '{app}.panel.local',

    auto_security_updates BOOLEAN NOT NULL DEFAULT 1,
    firewall_enabled BOOLEAN NOT NULL DEFAULT 1,

    default_backup_frequency TEXT DEFAULT 'daily',
    default_backup_time TEXT DEFAULT '02:00',
    default_backup_retention_days INTEGER DEFAULT 7,
    default_backup_destination TEXT DEFAULT 'local',

    s3_config TEXT,

    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- Insert default row
INSERT OR IGNORE INTO server_info (id, hostname, os, architecture)
VALUES (1, 'localhost', 'Unknown', 'amd64');

-- Firewall rules managed by the panel
CREATE TABLE firewall_rules (
    id TEXT PRIMARY KEY,
    port INTEGER NOT NULL,
    protocol TEXT NOT NULL DEFAULT 'tcp'
        CHECK (protocol IN ('tcp', 'udp', 'tcp/udp')),
    source TEXT NOT NULL DEFAULT 'any',
    action TEXT NOT NULL DEFAULT 'allow'
        CHECK (action IN ('allow', 'deny')),
    description TEXT,
    app_id TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE INDEX idx_firewall_rules_app_id ON firewall_rules(app_id);
CREATE INDEX idx_firewall_rules_port ON firewall_rules(port);

-- Scheduled tasks
CREATE TABLE cron_jobs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'paused', 'error')),
    schedule TEXT NOT NULL,
    command TEXT NOT NULL,

    target_type TEXT NOT NULL
        CHECK (target_type IN ('app', 'service', 'server')),
    target_id TEXT,

    notify_on_failure BOOLEAN NOT NULL DEFAULT 1,
    log_retention INTEGER DEFAULT 10,

    last_run_at DATETIME,
    last_run_status TEXT,
    next_run_at DATETIME,
    created_by INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_cron_jobs_status ON cron_jobs(status);
CREATE INDEX idx_cron_jobs_target ON cron_jobs(target_type, target_id);
CREATE INDEX idx_cron_jobs_next_run ON cron_jobs(next_run_at) WHERE status = 'active';

-- Execution history for cron jobs
CREATE TABLE cron_executions (
    id TEXT PRIMARY KEY,
    cron_job_id TEXT NOT NULL,
    status TEXT NOT NULL
        CHECK (status IN ('success', 'failed', 'timeout')),
    exit_code INTEGER,
    output TEXT,
    error_output TEXT,
    duration_seconds INTEGER,
    started_at DATETIME NOT NULL DEFAULT (datetime('now')),
    completed_at DATETIME,

    FOREIGN KEY (cron_job_id) REFERENCES cron_jobs(id) ON DELETE CASCADE
);

CREATE INDEX idx_cron_executions_cron_job_id ON cron_executions(cron_job_id);
CREATE INDEX idx_cron_executions_started_at ON cron_executions(started_at);

-- =====================================================
-- USER & TEAM TABLES
-- =====================================================

-- Pending team invitations
CREATE TABLE user_invites (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'developer'
        CHECK (role IN ('admin', 'developer', 'viewer')),
    token_hash TEXT NOT NULL UNIQUE,
    invited_by INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    accepted_at DATETIME,
    accepted_by_user_id INTEGER,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (invited_by) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (accepted_by_user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_user_invites_email ON user_invites(email);
CREATE INDEX idx_user_invites_expires_at ON user_invites(expires_at) WHERE accepted_at IS NULL;

-- =====================================================
-- AUDIT & LOG TABLES
-- =====================================================

-- Immutable audit trail of all actions
CREATE TABLE activity_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    user_username TEXT,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    details TEXT,
    ip_address TEXT,
    user_agent TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_activity_log_user_id ON activity_log(user_id);
CREATE INDEX idx_activity_log_action ON activity_log(action);
CREATE INDEX idx_activity_log_resource ON activity_log(resource_type, resource_id);
CREATE INDEX idx_activity_log_created_at ON activity_log(created_at);
CREATE INDEX idx_activity_log_user_created ON activity_log(user_id, created_at DESC);

-- User notifications
CREATE TABLE notifications (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'info'
        CHECK (severity IN ('info', 'warning', 'error', 'success')),
    link TEXT,
    read_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_read ON notifications(user_id, read_at) WHERE read_at IS NULL;
CREATE INDEX idx_notifications_unread ON notifications(user_id) WHERE read_at IS NULL;

-- =====================================================
-- TRIGGERS
-- =====================================================

-- Auto-update updated_at triggers
CREATE TRIGGER apps_updated_at AFTER UPDATE ON apps BEGIN
    UPDATE apps SET updated_at = datetime('now') WHERE id = new.id;
END;

CREATE TRIGGER services_updated_at AFTER UPDATE ON services BEGIN
    UPDATE services SET updated_at = datetime('now') WHERE id = new.id;
END;

CREATE TRIGGER cron_jobs_updated_at AFTER UPDATE ON cron_jobs BEGIN
    UPDATE cron_jobs SET updated_at = datetime('now') WHERE id = new.id;
END;

CREATE TRIGGER server_info_updated_at AFTER UPDATE ON server_info BEGIN
    UPDATE server_info SET updated_at = datetime('now') WHERE id = new.id;
END;

-- Deployment status change logging
CREATE TRIGGER deployment_status_change AFTER UPDATE OF status ON deployments
WHEN old.status != new.status BEGIN
    INSERT INTO activity_log (user_id, user_username, action, resource_type, resource_id, details, created_at)
    VALUES (
        NULL,
        'system',
        'deployment.status_changed',
        'deployment',
        new.id,
        json_object('old_status', old.status, 'new_status', new.status, 'app_id', new.app_id),
        datetime('now')
    );
END;

-- =====================================================
-- FULL-TEXT SEARCH (Optional - disabled due to FTS5 compatibility issues)
-- FTS5 is not available on all SQLite builds. Re-enable when needed.
-- =====================================================

-- CREATE VIRTUAL TABLE apps_fts USING fts5(
--     name,
--     content='apps',
--     content_rowid='rowid'
-- );

-- Triggers to keep FTS index in sync
-- CREATE TRIGGER apps_fts_insert AFTER INSERT ON apps BEGIN
--     INSERT INTO apps_fts(rowid, name) VALUES (new.rowid, new.name);
-- END;

-- CREATE TRIGGER apps_fts_delete AFTER DELETE ON apps BEGIN
--     INSERT INTO apps_fts(apps_fts, rowid, name) VALUES ('delete', old.rowid, old.name);
-- END;

-- CREATE TRIGGER apps_fts_update AFTER UPDATE ON apps BEGIN
--     INSERT INTO apps_fts(apps_fts, rowid, name) VALUES ('delete', old.rowid, old.name);
--     INSERT INTO apps_fts(rowid, name) VALUES (new.rowid, new.name);
-- END;

-- =====================================================
-- VERIFICATION
-- =====================================================

-- Verify all tables created
-- SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;
