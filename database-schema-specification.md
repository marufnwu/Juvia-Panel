# Server Panel — Database Schema Specification
## SQLite (libsql) — Single-Server PaaS

**Version:** 1.0  
**Date:** 2026-06-01  
**Database:** SQLite via `modernc.org/sqlite` (libsql compatible)  
**File:** `/var/panel/panel.db`  
**Migration Tool:** golang-migrate (Go) or manual SQL files in `/var/panel/migrations/`  
**Backup:** Live copy via `VACUUM INTO` or SQLite `.backup` command

---

## Table of Contents

1. [Migration Strategy](#1-migration-strategy)
2. [Core Tables](#2-core-tables)
3. [App Tables](#3-app-tables)
4. [Service Tables](#4-service-tables)
5. [Server & System Tables](#5-server--system-tables)
6. [User & Team Tables](#6-user--team-tables)
7. [Audit & Log Tables](#7-audit--log-tables)
8. [Indexes](#8-indexes)
9. [Triggers](#9-triggers)
10. [Data Retention](#10-data-retention)
11. [Entity Relationship Diagram](#11-entity-relationship-diagram)

---

## 1. Migration Strategy

### 1.1 Migration Files
Stored in `/var/panel/migrations/`:
```
migrations/
├── 000001_init.up.sql
├── 000001_init.down.sql
├── 000002_add_backup_checksum.up.sql
├── 000002_add_backup_checksum.down.sql
└── ...
```

### 1.2 Migration Table
SQLite table `schema_migrations` is auto-managed by golang-migrate:
```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    dirty BOOLEAN NOT NULL DEFAULT 0
);
```

### 1.3 Migration Rules
- Every migration is idempotent (use `IF NOT EXISTS`, `DROP IF EXISTS`)
- Never modify existing column types in `.up.sql` — add new columns, migrate data, drop old in separate migrations
- `.down.sql` must reverse `.up.sql` exactly
- Test migrations on a copy of production data before deployment

---

## 2. Core Tables

### 2.1 `users`
Panel users and team members.

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,           -- bcrypt hash, cost 12+
    role TEXT NOT NULL DEFAULT 'viewer'    -- owner, admin, developer, viewer
        CHECK (role IN ('owner', 'admin', 'developer', 'viewer')),
    two_factor_secret TEXT,                -- encrypted TOTP secret, NULL if 2FA disabled
    two_factor_enabled BOOLEAN NOT NULL DEFAULT 0,
    two_factor_backup_codes TEXT,          -- JSON array of hashed backup codes
    avatar_url TEXT,
    last_login_at DATETIME,
    last_login_ip TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
```

### 2.2 `api_keys`
API keys for programmatic access.

```sql
CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,                   -- key_ + nanoid (12 chars)
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,       -- SHA-256 hash of the token (token itself is shown once)
    scopes TEXT NOT NULL,                  -- JSON array: ["read", "deploy", "manage"]
    last_used_at DATETIME,
    expires_at DATETIME,                   -- NULL = never expires
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    revoked_at DATETIME,                   -- NULL = active
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_token_hash ON api_keys(token_hash);
```

### 2.3 `sessions`
Active user sessions for logout-everywhere and session management.

```sql
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,                   -- sess_ + nanoid
    user_id INTEGER NOT NULL,
    refresh_token_hash TEXT NOT NULL UNIQUE, -- SHA-256 of refresh token
    ip_address TEXT,
    user_agent TEXT,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

---

## 3. App Tables

### 3.1 `apps`
Deployed applications.

```sql
CREATE TABLE apps (
    id TEXT PRIMARY KEY,                   -- app_ + nanoid (12 chars)
    name TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'stopped'
        CHECK (status IN ('running', 'stopped', 'failed', 'deploying', 'restarting')),
    health_status TEXT DEFAULT 'unknown'
        CHECK (health_status IN ('healthy', 'unhealthy', 'unknown')),
    runtime TEXT NOT NULL                  -- nodejs, python, go, php, ruby, static, docker
        CHECK (runtime IN ('nodejs', 'python', 'go', 'php', 'ruby', 'static', 'docker')),
    runtime_version TEXT,                  -- e.g., "20.11.0"

    -- Source configuration (stored as JSON for flexibility)
    source_type TEXT NOT NULL DEFAULT 'git'
        CHECK (source_type IN ('git', 'upload', 'docker_compose')),
    source_config TEXT NOT NULL,           -- JSON: {repo_url, branch, auto_deploy, provider}

    -- Build configuration
    build_strategy TEXT NOT NULL DEFAULT 'nixpacks'
        CHECK (build_strategy IN ('nixpacks', 'dockerfile', 'static', 'custom')),
    build_config TEXT,                     -- JSON: {build_command, start_command, dockerfile_path}

    -- Container info
    container_id TEXT,                     -- Docker container ID
    container_image TEXT,                  -- Docker image name
    internal_port INTEGER DEFAULT 3000,
    restart_policy TEXT DEFAULT 'unless-stopped'
        CHECK (restart_policy IN ('always', 'on-failure', 'unless-stopped')),

    -- Health check
    health_check_path TEXT DEFAULT '/health',
    health_check_interval INTEGER DEFAULT 30,  -- seconds
    health_check_timeout INTEGER DEFAULT 5,    -- seconds
    health_check_retries INTEGER DEFAULT 3,

    -- Resource limits
    cpu_limit REAL,                        -- number of cores, NULL = unlimited
    memory_limit_mb INTEGER,               -- NULL = unlimited
    memory_swap_mb INTEGER,                -- NULL = default

    -- Metadata
    created_by INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_apps_status ON apps(status);
CREATE INDEX idx_apps_runtime ON apps(runtime);
CREATE INDEX idx_apps_created_by ON apps(created_by);
CREATE INDEX idx_apps_updated_at ON apps(updated_at);
```

### 3.2 `app_domains`
Domains attached to apps. One app can have multiple domains.

```sql
CREATE TABLE app_domains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    app_id TEXT NOT NULL,
    domain TEXT NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT 0,
    force_https BOOLEAN NOT NULL DEFAULT 1,
    ssl_status TEXT DEFAULT 'pending'
        CHECK (ssl_status IN ('pending', 'valid', 'expiring', 'failed')),
    ssl_provider TEXT,                     -- letsencrypt, custom
    ssl_cert_path TEXT,
    ssl_key_path TEXT,
    ssl_issued_at DATETIME,
    ssl_expires_at DATETIME,
    ssl_auto_renew BOOLEAN NOT NULL DEFAULT 1,
    dns_valid BOOLEAN DEFAULT NULL,        -- NULL = not checked, 0/1 = result
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE,
    UNIQUE(app_id, domain)
);

CREATE INDEX idx_app_domains_app_id ON app_domains(app_id);
CREATE INDEX idx_app_domains_domain ON app_domains(domain);
CREATE INDEX idx_app_domains_ssl_expires ON app_domains(ssl_expires_at) 
    WHERE ssl_status = 'valid' AND ssl_auto_renew = 1;
```

### 3.3 `app_env_vars`
Environment variables for apps. Secrets are encrypted at application level before storage.

```sql
CREATE TABLE app_env_vars (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    app_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,                   -- encrypted if is_secret = 1
    is_secret BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE,
    UNIQUE(app_id, key)
);

CREATE INDEX idx_app_env_vars_app_id ON app_env_vars(app_id);
```

### 3.4 `app_volumes`
Persistent storage volumes mounted into app containers.

```sql
CREATE TABLE app_volumes (
    id TEXT PRIMARY KEY,                   -- vol_ + nanoid
    app_id TEXT NOT NULL,
    name TEXT NOT NULL,
    host_path TEXT NOT NULL,               -- absolute path on host
    container_path TEXT NOT NULL,          -- mount point inside container
    size_mb INTEGER DEFAULT 0,             -- calculated periodically
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE INDEX idx_app_volumes_app_id ON app_volumes(app_id);
```

### 3.5 `deployments`
Deployment history for apps.

```sql
CREATE TABLE deployments (
    id TEXT PRIMARY KEY,                   -- dep_ + nanoid
    app_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'in_progress', 'success', 'failed', 'cancelled')),

    -- Git info
    commit_sha TEXT,
    commit_message TEXT,
    commit_author TEXT,
    branch TEXT,

    -- Build info
    build_strategy TEXT,
    build_logs TEXT,                       -- stored as text, truncated if > 1MB
    build_duration_seconds INTEGER,
    deploy_duration_seconds INTEGER,

    -- Trigger info
    triggered_by TEXT NOT NULL DEFAULT 'manual'  -- manual, git_push, webhook, rollback, api
        CHECK (triggered_by IN ('manual', 'git_push', 'webhook', 'rollback', 'api')),
    triggered_by_user_id INTEGER,

    -- Rollback info
    rollback_of_id TEXT,                   -- reference to original deployment if this is a rollback

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
```

---

## 4. Service Tables

### 4.1 `services`
Managed databases, caches, and other backing services.

```sql
CREATE TABLE services (
    id TEXT PRIMARY KEY,                   -- svc_ + nanoid
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL
        CHECK (type IN ('postgresql', 'mysql', 'mariadb', 'mongodb', 'redis', 'memcached', 'minio', 'custom')),
    version TEXT NOT NULL,                 -- e.g., "15.4"
    status TEXT NOT NULL DEFAULT 'stopped'
        CHECK (status IN ('running', 'stopped', 'failed', 'creating', 'restarting')),

    -- Network
    internal_port INTEGER NOT NULL,
    internal_host TEXT NOT NULL,           -- service name for Docker DNS resolution
    container_id TEXT,
    container_image TEXT,

    -- Credentials (encrypted values)
    credentials TEXT NOT NULL,             -- JSON: {username, password, database, connection_string}

    -- Resources
    memory_limit_mb INTEGER,
    cpu_limit REAL,

    -- Data
    data_path TEXT NOT NULL,               -- host path to data directory
    data_size_mb INTEGER DEFAULT 0,

    -- Backup schedule
    backup_enabled BOOLEAN NOT NULL DEFAULT 1,
    backup_frequency TEXT DEFAULT 'daily'  -- hourly, daily, weekly
        CHECK (backup_frequency IN ('hourly', 'daily', 'weekly')),
    backup_time TEXT DEFAULT '02:00',      -- HH:MM in UTC
    backup_retention_days INTEGER DEFAULT 7,
    backup_destination TEXT DEFAULT 'local' -- local, s3
        CHECK (backup_destination IN ('local', 's3')),

    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_services_type ON services(type);
CREATE INDEX idx_services_status ON services(status);
```

### 4.2 `service_app_links`
Many-to-many relationship between services and apps.

```sql
CREATE TABLE service_app_links (
    service_id TEXT NOT NULL,
    app_id TEXT NOT NULL,
    connection_env_key TEXT,               -- e.g., "DATABASE_URL" — auto-injected into app env
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),

    PRIMARY KEY (service_id, app_id),
    FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE,
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE INDEX idx_service_app_links_app_id ON service_app_links(app_id);
```

### 4.3 `backups`
Backup records for apps and services.

```sql
CREATE TABLE backups (
    id TEXT PRIMARY KEY,                   -- bak_ + nanoid
    target_type TEXT NOT NULL              -- app, service
        CHECK (target_type IN ('app', 'service')),
    target_id TEXT NOT NULL,
    target_name TEXT NOT NULL,

    status TEXT NOT NULL DEFAULT 'in_progress'
        CHECK (status IN ('in_progress', 'success', 'failed')),
    size_mb INTEGER,

    destination TEXT NOT NULL              -- local, s3
        CHECK (destination IN ('local', 's3')),
    destination_path TEXT NOT NULL,        -- absolute path or S3 URI

    checksum TEXT,                         -- SHA-256 of backup file
    checksum_algorithm TEXT DEFAULT 'sha256',

    triggered_by TEXT DEFAULT 'schedule'   -- schedule, manual, api
        CHECK (triggered_by IN ('schedule', 'manual', 'api')),
    triggered_by_user_id INTEGER,

    started_at DATETIME NOT NULL DEFAULT (datetime('now')),
    completed_at DATETIME,

    FOREIGN KEY (triggered_by_user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_backups_target ON backups(target_type, target_id);
CREATE INDEX idx_backups_status ON backups(status);
CREATE INDEX idx_backups_created_at ON backups(created_at);
```

---

## 5. Server & System Tables

### 5.1 `server_info`
Single-row table with server configuration. Only one row ever exists (id = 1).

```sql
CREATE TABLE server_info (
    id INTEGER PRIMARY KEY CHECK (id = 1), -- enforce single row
    hostname TEXT NOT NULL,
    os TEXT NOT NULL,                      -- e.g., "Ubuntu 24.04 LTS"
    kernel TEXT,
    architecture TEXT NOT NULL DEFAULT 'amd64',
    timezone TEXT NOT NULL DEFAULT 'UTC',

    -- Resources
    cpu_cores INTEGER,
    cpu_model TEXT,
    memory_total_mb INTEGER,
    disk_total_gb INTEGER,

    -- Panel config
    panel_version TEXT NOT NULL DEFAULT '1.0.0',
    panel_domain TEXT,
    default_app_subdomain TEXT DEFAULT '{app}.panel.local',

    -- Security
    auto_security_updates BOOLEAN NOT NULL DEFAULT 1,
    firewall_enabled BOOLEAN NOT NULL DEFAULT 1,

    -- Backup defaults
    default_backup_frequency TEXT DEFAULT 'daily',
    default_backup_time TEXT DEFAULT '02:00',
    default_backup_retention_days INTEGER DEFAULT 7,
    default_backup_destination TEXT DEFAULT 'local',

    -- S3 config (encrypted)
    s3_config TEXT,                        -- JSON: {endpoint, bucket, region, access_key_id}

    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- Insert default row
INSERT INTO server_info (id, hostname, os, architecture) 
VALUES (1, 'localhost', 'Unknown', 'amd64');
```

### 5.2 `firewall_rules`
Firewall rules managed by the panel.

```sql
CREATE TABLE firewall_rules (
    id TEXT PRIMARY KEY,                   -- fw_ + nanoid
    port INTEGER NOT NULL,
    protocol TEXT NOT NULL DEFAULT 'tcp'
        CHECK (protocol IN ('tcp', 'udp', 'tcp/udp')),
    source TEXT NOT NULL DEFAULT 'any',    -- IP, CIDR, or "any"
    action TEXT NOT NULL DEFAULT 'allow'
        CHECK (action IN ('allow', 'deny')),
    description TEXT,
    app_id TEXT,                           -- NULL = system rule
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE INDEX idx_firewall_rules_app_id ON firewall_rules(app_id);
CREATE INDEX idx_firewall_rules_port ON firewall_rules(port);
```

### 5.3 `cron_jobs`
Scheduled tasks.

```sql
CREATE TABLE cron_jobs (
    id TEXT PRIMARY KEY,                   -- cron_ + nanoid
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'paused', 'error')),
    schedule TEXT NOT NULL,                -- cron expression: "0 2 * * *"
    command TEXT NOT NULL,

    -- Target
    target_type TEXT NOT NULL              -- app, service, server
        CHECK (target_type IN ('app', 'service', 'server')),
    target_id TEXT,                        -- NULL if target_type = 'server'

    -- Notifications
    notify_on_failure BOOLEAN NOT NULL DEFAULT 1,
    log_retention INTEGER DEFAULT 10,      -- number of recent runs to keep

    -- Metadata
    last_run_at DATETIME,
    last_run_status TEXT,                  -- success, failed
    next_run_at DATETIME,
    created_by INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_cron_jobs_status ON cron_jobs(status);
CREATE INDEX idx_cron_jobs_target ON cron_jobs(target_type, target_id);
CREATE INDEX idx_cron_jobs_next_run ON cron_jobs(next_run_at) WHERE status = 'active';
```

### 5.4 `cron_executions`
Execution history for cron jobs.

```sql
CREATE TABLE cron_executions (
    id TEXT PRIMARY KEY,                   -- exec_ + nanoid
    cron_job_id TEXT NOT NULL,
    status TEXT NOT NULL
        CHECK (status IN ('success', 'failed', 'timeout')),
    exit_code INTEGER,
    output TEXT,                           -- stdout
    error_output TEXT,                     -- stderr
    duration_seconds INTEGER,
    started_at DATETIME NOT NULL DEFAULT (datetime('now')),
    completed_at DATETIME,

    FOREIGN KEY (cron_job_id) REFERENCES cron_jobs(id) ON DELETE CASCADE
);

CREATE INDEX idx_cron_executions_cron_job_id ON cron_executions(cron_job_id);
CREATE INDEX idx_cron_executions_started_at ON cron_executions(started_at);
```

---

## 6. User & Team Tables

### 6.1 `user_invites`
Pending team invitations.

```sql
CREATE TABLE user_invites (
    id TEXT PRIMARY KEY,                   -- inv_ + nanoid
    email TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'developer'
        CHECK (role IN ('admin', 'developer', 'viewer')),
    token_hash TEXT NOT NULL UNIQUE,       -- SHA-256 of invite token
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
```

---

## 7. Audit & Log Tables

### 7.1 `activity_log`
Immutable audit trail of all actions.

```sql
CREATE TABLE activity_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,                       -- NULL = system action
    user_username TEXT,                    -- denormalized for history preservation
    action TEXT NOT NULL,                  -- e.g., "app.create", "deployment.trigger", "user.login"
    resource_type TEXT NOT NULL,           -- app, service, user, server, backup, etc.
    resource_id TEXT,                      -- ID of affected resource
    details TEXT,                          -- JSON with action-specific data
    ip_address TEXT,
    user_agent TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_activity_log_user_id ON activity_log(user_id);
CREATE INDEX idx_activity_log_action ON activity_log(action);
CREATE INDEX idx_activity_log_resource ON activity_log(resource_type, resource_id);
CREATE INDEX idx_activity_log_created_at ON activity_log(created_at);
```

### 7.2 `notifications`
User notifications.

```sql
CREATE TABLE notifications (
    id TEXT PRIMARY KEY,                   -- notif_ + nanoid
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'info'
        CHECK (severity IN ('info', 'warning', 'error', 'success')),
    link TEXT,                             -- relative URL to relevant page
    read_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_read ON notifications(user_id, read_at) WHERE read_at IS NULL;
```

---

## 8. Indexes

### 8.1 Performance-Critical Indexes
```sql
-- App lookups by status (dashboard filtering)
CREATE INDEX idx_apps_status_created ON apps(status, created_at);

-- Deployment lookups for app detail page
CREATE INDEX idx_deployments_app_created ON deployments(app_id, created_at DESC);

-- Service lookups by type (service list page)
CREATE INDEX idx_services_type_status ON services(type, status);

-- Backup lookups for restore
CREATE INDEX idx_backups_target_created ON backups(target_type, target_id, created_at DESC);

-- Activity log for audit queries
CREATE INDEX idx_activity_log_user_created ON activity_log(user_id, created_at DESC);

-- Notifications unread count
CREATE INDEX idx_notifications_unread ON notifications(user_id) WHERE read_at IS NULL;
```

### 8.2 Full-Text Search (Optional)
For searching app names, domains, and service names:
```sql
-- SQLite FTS5 extension for fast text search
CREATE VIRTUAL TABLE apps_fts USING fts5(
    name, 
    content='apps',
    content_rowid='rowid'
);

-- Trigger to keep FTS index in sync
CREATE TRIGGER apps_fts_insert AFTER INSERT ON apps BEGIN
    INSERT INTO apps_fts(rowid, name) VALUES (new.rowid, new.name);
END;

CREATE TRIGGER apps_fts_delete AFTER DELETE ON apps BEGIN
    INSERT INTO apps_fts(apps_fts, rowid, name) VALUES ('delete', old.rowid, old.name);
END;

CREATE TRIGGER apps_fts_update AFTER UPDATE ON apps BEGIN
    INSERT INTO apps_fts(apps_fts, rowid, name) VALUES ('delete', old.rowid, old.name);
    INSERT INTO apps_fts(rowid, name) VALUES (new.rowid, new.name);
END;
```

---

## 9. Triggers

### 9.1 Auto-Update `updated_at`
```sql
CREATE TRIGGER apps_updated_at AFTER UPDATE ON apps BEGIN
    UPDATE apps SET updated_at = datetime('now') WHERE id = new.id;
END;

CREATE TRIGGER services_updated_at AFTER UPDATE ON services BEGIN
    UPDATE services SET updated_at = datetime('now') WHERE id = new.id;
END;

CREATE TRIGGER cron_jobs_updated_at AFTER UPDATE ON cron_jobs BEGIN
    UPDATE cron_jobs SET updated_at = datetime('now') WHERE id = new.id;
END;
```

### 9.2 Deployment Status Change Logging
```sql
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
```

---

## 10. Data Retention

### 10.1 Retention Policies
| Table | Retention | Cleanup Method |
|-------|-----------|----------------|
| `deployments` | 90 days | Daily cron job deletes old records, archives build logs |
| `cron_executions` | 30 days | Daily cron job deletes old records |
| `activity_log` | 90 days | Daily cron job deletes old records, exports to CSV first |
| `notifications` | 30 days after read | Daily cron job deletes read notifications |
| `sessions` | On expiry + 7 days grace | Daily cron job deletes expired sessions |
| `backups` | Per backup schedule retention | Backup manager deletes old backups from storage and table |

### 10.2 Cleanup SQL
```sql
-- Run daily via agent cron
DELETE FROM deployments WHERE created_at < datetime('now', '-90 days');
DELETE FROM cron_executions WHERE started_at < datetime('now', '-30 days');
DELETE FROM activity_log WHERE created_at < datetime('now', '-90 days');
DELETE FROM notifications WHERE read_at IS NOT NULL AND read_at < datetime('now', '-30 days');
DELETE FROM sessions WHERE expires_at < datetime('now', '-7 days');
```

---

## 11. Entity Relationship Diagram

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│     users       │     │   api_keys      │     │   sessions      │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ id (PK)         │◄────┤ user_id (FK)    │     │ user_id (FK)    │
│ username        │     │ token_hash      │     │ refresh_token   │
│ email           │     │ scopes          │     │ expires_at      │
│ password_hash   │     │ last_used_at    │     └─────────────────┘
│ role            │     └─────────────────┘
│ 2fa_enabled     │
└────────┬────────┘
         │
         │ created_by
         ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│      apps       │◄────┤ app_domains     │     │  app_env_vars   │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ id (PK)         │     │ app_id (FK)     │     │ app_id (FK)     │
│ name            │     │ domain          │     │ key             │
│ status          │     │ is_primary      │     │ value           │
│ runtime         │     │ ssl_status      │     │ is_secret       │
│ source_config   │     │ ssl_expires_at  │     └─────────────────┘
│ build_config    │     └─────────────────┘
│ container_id    │
│ cpu_limit       │     ┌─────────────────┐     ┌─────────────────┐
│ memory_limit_mb │◄────┤  app_volumes    │     │  deployments    │
└────────┬────────┘     ├─────────────────┤     ├─────────────────┤
         │              │ app_id (FK)     │     │ app_id (FK)     │
         │              │ host_path       │     │ status          │
         │              │ container_path  │     │ commit_sha      │
         │              └─────────────────┘     │ triggered_by    │
         │                                      └─────────────────┘
         │
         │ (via service_app_links)
         │
         ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    services     │◄────┤service_app_links│     │    backups      │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ id (PK)         │     │ service_id (FK) │     │ target_type     │
│ name            │     │ app_id (FK)     │     │ target_id       │
│ type            │     └─────────────────┘     │ status          │
│ version         │                             │ destination     │
│ credentials     │                             │ checksum        │
│ data_path       │                             └─────────────────┘
│ backup_schedule │
└─────────────────┘

┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   server_info   │     │ firewall_rules  │     │   cron_jobs     │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ id = 1 (PK)     │     │ port            │     │ schedule        │
│ hostname        │     │ protocol        │     │ command         │
│ panel_version   │     │ source          │     │ target_type     │
│ s3_config       │     │ action          │     │ target_id       │
└─────────────────┘     │ app_id (FK)     │     │ notify_on_fail  │
                        └─────────────────┘     └────────┬────────┘
                                                         │
                                                         ▼
                                                ┌─────────────────┐
                                                │cron_executions  │
                                                ├─────────────────┤
                                                │ cron_job_id (FK)│
                                                │ status          │
                                                │ exit_code       │
                                                │ output          │
                                                └─────────────────┘

┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  user_invites   │     │  activity_log   │     │  notifications  │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ email           │     │ user_id         │     │ user_id (FK)    │
│ role            │     │ action          │     │ title           │
│ token_hash      │     │ resource_type   │     │ severity        │
│ invited_by (FK) │     │ resource_id     │     │ read_at         │
│ expires_at      │     │ details (JSON)  │     └─────────────────┘
└─────────────────┘     │ ip_address      │
                        └─────────────────┘
```

---

## Appendix A: Complete Initialization SQL

Run this to create a fresh database:

```sql
-- Enable foreign keys (required for SQLite)
PRAGMA foreign_keys = ON;

-- Run all CREATE TABLE statements above in order:
-- 1. schema_migrations (auto)
-- 2. users
-- 3. api_keys
-- 4. sessions
-- 5. apps
-- 6. app_domains
-- 7. app_env_vars
-- 8. app_volumes
-- 9. deployments
-- 10. services
-- 11. service_app_links
-- 12. backups
-- 13. server_info (with default row)
-- 14. firewall_rules
-- 15. cron_jobs
-- 16. cron_executions
-- 17. user_invites
-- 18. activity_log
-- 19. notifications

-- Then create all indexes
-- Then create all triggers
-- Then create FTS tables (optional)

-- Verify
SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;
```

---

## Appendix B: Go Structs (sqlx compatible)

```go
package models

import "time"

type User struct {
    ID                 int       `db:"id" json:"id"`
    Username           string    `db:"username" json:"username"`
    Email              string    `db:"email" json:"email"`
    PasswordHash       string    `db:"password_hash" json:"-"`
    Role               string    `db:"role" json:"role"`
    TwoFactorSecret    *string   `db:"two_factor_secret" json:"-"`
    TwoFactorEnabled   bool      `db:"two_factor_enabled" json:"two_factor_enabled"`
    TwoFactorBackupCodes *string `db:"two_factor_backup_codes" json:"-"`
    AvatarURL          *string   `db:"avatar_url" json:"avatar_url"`
    LastLoginAt        *time.Time `db:"last_login_at" json:"last_login_at"`
    LastLoginIP        *string   `db:"last_login_ip" json:"last_login_ip"`
    CreatedAt          time.Time `db:"created_at" json:"created_at"`
    UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`
}

type App struct {
    ID                string    `db:"id" json:"id"`
    Name              string    `db:"name" json:"name"`
    Status            string    `db:"status" json:"status"`
    HealthStatus      string    `db:"health_status" json:"health_status"`
    Runtime           string    `db:"runtime" json:"runtime"`
    RuntimeVersion    *string   `db:"runtime_version" json:"runtime_version"`
    SourceType        string    `db:"source_type" json:"source_type"`
    SourceConfig      string    `db:"source_config" json:"source_config"`
    BuildStrategy     string    `db:"build_strategy" json:"build_strategy"`
    BuildConfig       *string   `db:"build_config" json:"build_config"`
    ContainerID       *string   `db:"container_id" json:"container_id"`
    ContainerImage    *string   `db:"container_image" json:"container_image"`
    InternalPort      int       `db:"internal_port" json:"internal_port"`
    RestartPolicy     string    `db:"restart_policy" json:"restart_policy"`
    HealthCheckPath   string    `db:"health_check_path" json:"health_check_path"`
    HealthCheckInterval int     `db:"health_check_interval" json:"health_check_interval"`
    HealthCheckTimeout  int     `db:"health_check_timeout" json:"health_check_timeout"`
    HealthCheckRetries  int     `db:"health_check_retries" json:"health_check_retries"`
    CPULimit          *float64  `db:"cpu_limit" json:"cpu_limit"`
    MemoryLimitMB     *int      `db:"memory_limit_mb" json:"memory_limit_mb"`
    MemorySwapMB      *int      `db:"memory_swap_mb" json:"memory_swap_mb"`
    CreatedBy         int       `db:"created_by" json:"created_by"`
    CreatedAt         time.Time `db:"created_at" json:"created_at"`
    UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

type Service struct {
    ID                string    `db:"id" json:"id"`
    Name              string    `db:"name" json:"name"`
    Type              string    `db:"type" json:"type"`
    Version           string    `db:"version" json:"version"`
    Status            string    `db:"status" json:"status"`
    InternalPort      int       `db:"internal_port" json:"internal_port"`
    InternalHost      string    `db:"internal_host" json:"internal_host"`
    ContainerID       *string   `db:"container_id" json:"container_id"`
    ContainerImage    *string   `db:"container_image" json:"container_image"`
    Credentials       string    `db:"credentials" json:"credentials"`
    MemoryLimitMB     *int      `db:"memory_limit_mb" json:"memory_limit_mb"`
    CPULimit          *float64  `db:"cpu_limit" json:"cpu_limit"`
    DataPath          string    `db:"data_path" json:"data_path"`
    DataSizeMB        int       `db:"data_size_mb" json:"data_size_mb"`
    BackupEnabled     bool      `db:"backup_enabled" json:"backup_enabled"`
    BackupFrequency   string    `db:"backup_frequency" json:"backup_frequency"`
    BackupTime        string    `db:"backup_time" json:"backup_time"`
    BackupRetentionDays int     `db:"backup_retention_days" json:"backup_retention_days"`
    BackupDestination string    `db:"backup_destination" json:"backup_destination"`
    CreatedAt         time.Time `db:"created_at" json:"created_at"`
    UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}
```

---

*End of Database Schema Specification*
