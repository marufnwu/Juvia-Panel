# Server Panel — API Specification
## REST API + WebSocket Events

**Version:** 1.0  
**Date:** 2026-06-01  
**Base URL:** `https://panel.example.com/api/v1`  
**Protocol:** HTTPS only. HTTP is permanently redirected to HTTPS.  
**Content-Type:** `application/json` for all requests and responses.  
**Authentication:** JWT Bearer token in `Authorization` header. Refresh token in HTTP-only cookie.

---

## Table of Contents

1. [Authentication](#1-authentication)
2. [Apps](#2-apps)
3. [Deployments](#3-deployments)
4. [Services](#4-services)
5. [Backups](#5-backups)
6. [Server](#6-server)
7. [Domains & SSL](#7-domains--ssl)
8. [Cron Jobs](#8-cron-jobs)
9. [Firewall](#9-firewall)
10. [Users & Team](#10-users--team)
11. [Settings](#11-settings)
12. [WebSocket Events](#12-websocket-events)
13. [Error Handling](#13-error-handling)
14. [Rate Limiting](#14-rate-limiting)

---

## 1. Authentication

### 1.1 Login
**POST** `/auth/login`

Request body:
```json
{
  "username": "admin",
  "password": "secure_password_123",
  "totp_code": "123456"
}
```

Response `200 OK`:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "owner",
    "two_factor_enabled": true
  }
}
```

Response `401 Unauthorized`:
```json
{
  "error": "invalid_credentials",
  "message": "Username or password is incorrect."
}
```

Response `403 Forbidden` (2FA required):
```json
{
  "error": "totp_required",
  "message": "Two-factor authentication code is required."
}
```

**Notes:**
- `totp_code` is optional unless 2FA is enabled for the user.
- Refresh token is set as HTTP-only cookie `refresh_token` on successful login.
- Access token expires in 15 minutes (900 seconds).

### 1.2 Refresh Token
**POST** `/auth/refresh`

Reads `refresh_token` from HTTP-only cookie. No request body required.

Response `200 OK`:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

Response `401 Unauthorized`:
```json
{
  "error": "invalid_refresh_token",
  "message": "Refresh token is invalid or expired. Please log in again."
}
```

### 1.3 Logout
**POST** `/auth/logout`

Invalidates the refresh token cookie and blacklists the current access token until expiry.

Response `200 OK`:
```json
{
  "message": "Logged out successfully."
}
```

### 1.4 Setup 2FA
**POST** `/auth/2fa/setup`

Response `200 OK`:
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code_url": "otpauth://totp/Panel:admin?secret=JBSWY3DPEHPK3PXP&issuer=Panel",
  "backup_codes": ["12345678", "87654321", "11223344", "55667788", "99887766"]
}
```

**POST** `/auth/2fa/verify` — Verify and enable 2FA:
```json
{
  "totp_code": "123456",
  "secret": "JBSWY3DPEHPK3PXP"
}
```

Response `200 OK`:
```json
{
  "message": "Two-factor authentication enabled successfully."
}
```

### 1.5 Disable 2FA
**POST** `/auth/2fa/disable`

Requires current password and TOTP code.

Request:
```json
{
  "password": "secure_password_123",
  "totp_code": "123456"
}
```

---

## 2. Apps

### 2.1 List Apps
**GET** `/apps`

Query parameters:
- `status` (optional): `running`, `stopped`, `failed`, `deploying`, `all` (default: `all`)
- `runtime` (optional): `nodejs`, `python`, `go`, `php`, `ruby`, `static`, `docker`
- `search` (optional): string — filters by name or domain
- `sort` (optional): `name`, `updated_at`, `status` (default: `updated_at`)
- `order` (optional): `asc`, `desc` (default: `desc`)
- `page` (optional): integer (default: 1)
- `per_page` (optional): integer (default: 20, max: 100)

Response `200 OK`:
```json
{
  "data": [
    {
      "id": "app_123456",
      "name": "api-prod",
      "status": "running",
      "health_status": "healthy",
      "runtime": "nodejs",
      "runtime_version": "20.11.0",
      "primary_domain": "api.example.com",
      "domains": ["api.example.com", "www.api.example.com"],
      "source": {
        "type": "git",
        "provider": "github",
        "repo_url": "https://github.com/user/api",
        "branch": "main",
        "last_commit": "abc1234",
        "last_commit_message": "Fix auth middleware"
      },
      "build_strategy": "nixpacks",
      "container_id": "a1b2c3d4e5f6",
      "ports": {
        "internal": 3000,
        "external": null
      },
      "env_count": 8,
      "volume_count": 2,
      "resource_usage": {
        "cpu_percent": 12.5,
        "memory_mb": 256,
        "memory_limit_mb": 512
      },
      "last_deployed_at": "2024-06-01T12:34:56Z",
      "created_at": "2024-05-15T08:00:00Z",
      "updated_at": "2024-06-01T12:34:56Z"
    }
  ],
  "meta": {
    "total": 6,
    "page": 1,
    "per_page": 20,
    "total_pages": 1
  }
}
```

### 2.2 Get App
**GET** `/apps/{id}`

Response `200 OK`:
```json
{
  "id": "app_123456",
  "name": "api-prod",
  "status": "running",
  "health_status": "healthy",
  "runtime": "nodejs",
  "runtime_version": "20.11.0",
  "primary_domain": "api.example.com",
  "domains": [
    {
      "domain": "api.example.com",
      "ssl_status": "valid",
      "ssl_expires_at": "2026-09-01T00:00:00Z",
      "force_https": true
    }
  ],
  "source": {
    "type": "git",
    "provider": "github",
    "repo_url": "https://github.com/user/api",
    "branch": "main",
    "auto_deploy": true,
    "last_commit": "abc1234",
    "last_commit_message": "Fix auth middleware",
    "last_commit_author": "john",
    "last_commit_timestamp": "2024-06-01T12:30:00Z"
  },
  "build": {
    "strategy": "nixpacks",
    "build_command": "npm run build",
    "start_command": "npm start",
    "pre_deploy_hook": "npm run migrate",
    "post_deploy_hook": null,
    "dockerfile_path": null,
    "health_check": {
      "path": "/health",
      "interval": 30,
      "timeout": 5,
      "retries": 3
    }
  },
  "resources": {
    "cpu_limit": 2,
    "memory_limit_mb": 512,
    "memory_swap_mb": 512
  },
  "container": {
    "id": "a1b2c3d4e5f6",
    "image": "panel-app-api-prod:latest",
    "status": "running",
    "restart_policy": "unless-stopped",
    "ports": [3000],
    "network": "panel_apps"
  },
  "volumes": [
    {
      "host_path": "/var/panel/apps/app_123456/volumes/data",
      "container_path": "/app/data",
      "size_mb": 2345
    }
  ],
  "connected_services": [
    { "id": "svc_789", "name": "main-pg", "type": "postgresql" },
    { "id": "svc_012", "name": "redis-cache", "type": "redis" }
  ],
  "created_at": "2024-05-15T08:00:00Z",
  "updated_at": "2024-06-01T12:34:56Z"
}
```

### 2.3 Create App
**POST** `/apps`

Request body:
```json
{
  "name": "api-prod",
  "source": {
    "type": "git",
    "repo_url": "https://github.com/user/api",
    "branch": "main",
    "auto_deploy": true
  },
  "build": {
    "strategy": "nixpacks",
    "build_command": "npm run build",
    "start_command": "npm start",
    "health_check": {
      "path": "/health",
      "interval": 30,
      "timeout": 5,
      "retries": 3
    }
  },
  "domain": {
    "primary": "api.example.com",
    "force_https": true
  },
  "environment": {
    "NODE_ENV": "production",
    "PORT": "3000"
  },
  "resources": {
    "cpu_limit": 2,
    "memory_limit_mb": 512
  }
}
```

Response `201 Created`:
```json
{
  "id": "app_123456",
  "name": "api-prod",
  "status": "deploying",
  "message": "App created. Deployment started.",
  "deployment_id": "dep_789012"
}
```

Response `409 Conflict` (name exists):
```json
{
  "error": "app_name_exists",
  "message": "An app with the name 'api-prod' already exists."
}
```

Response `422 Unprocessable` (invalid Git URL):
```json
{
  "error": "invalid_git_url",
  "message": "Repository URL is not accessible. Ensure it is public or add an SSH key."
}
```

### 2.4 Update App
**PUT** `/apps/{id}`

Partial update. Only provided fields are modified.

Request body (example — update domain and build command):
```json
{
  "domain": {
    "primary": "api.example.com",
    "aliases": ["www.api.example.com"],
    "force_https": true
  },
  "build": {
    "build_command": "npm run build:prod",
    "start_command": "npm run start:prod"
  }
}
```

Response `200 OK`:
```json
{
  "id": "app_123456",
  "name": "api-prod",
  "message": "App updated. Restart required for some changes to take effect.",
  "requires_restart": true
}
```

### 2.5 Delete App
**DELETE** `/apps/{id}`

Query parameters:
- `force` (optional): boolean — skip confirmation checks (default: false)
- `delete_volumes` (optional): boolean — delete persistent data (default: false)

Response `200 OK`:
```json
{
  "message": "App 'api-prod' and all associated resources deleted.",
  "deleted_resources": {
    "container": true,
    "volumes": false,
    "domains": true,
    "ssl_certificates": true,
    "nginx_config": true
  }
}
```

Response `400 Bad Request` (volumes exist, delete_volumes not set):
```json
{
  "error": "volumes_exist",
  "message": "App has persistent volumes. Set delete_volumes=true to remove them, or export data first."
}
```

### 2.6 Restart App
**POST** `/apps/{id}/restart`

Response `200 OK`:
```json
{
  "message": "App 'api-prod' is restarting.",
  "status": "restarting"
}
```

### 2.7 Stop App
**POST** `/apps/{id}/stop`

Response `200 OK`:
```json
{
  "message": "App 'api-prod' stopped.",
  "status": "stopped"
}
```

### 2.8 Start App
**POST** `/apps/{id}/start`

Response `200 OK`:
```json
{
  "message": "App 'api-prod' started.",
  "status": "running"
}
```

### 2.9 Get App Logs
**GET** `/apps/{id}/logs`

Query parameters:
- `stream` (optional): `stdout`, `stderr`, `both` (default: `both`)
- `tail` (optional): integer — last N lines (default: 100, max: 10000)
- `since` (optional): ISO 8601 timestamp — logs after this time
- `search` (optional): string — filter by keyword
- `follow` (optional): boolean — if true, upgrades to WebSocket (default: false)

Response `200 OK`:
```json
{
  "app_id": "app_123456",
  "stream": "both",
  "lines": [
    {
      "timestamp": "2024-06-01T12:34:56.123Z",
      "stream": "stdout",
      "message": "GET /api/users 200 45ms"
    },
    {
      "timestamp": "2024-06-01T12:34:57.456Z",
      "stream": "stderr",
      "message": "ERROR: Connection timeout to redis"
    }
  ],
  "total_lines": 2456
}
```

**WebSocket upgrade:** When `follow=true`, the connection upgrades to WebSocket. The client receives newline-delimited JSON objects:
```json
{"timestamp":"2024-06-01T12:35:01Z","stream":"stdout","message":"Worker job completed"}
```

### 2.10 Get App Environment Variables
**GET** `/apps/{id}/env`

Response `200 OK`:
```json
{
  "app_id": "app_123456",
  "variables": [
    {
      "key": "NODE_ENV",
      "value": "production",
      "is_secret": false,
      "created_at": "2024-05-15T08:00:00Z",
      "updated_at": "2024-05-15T08:00:00Z"
    },
    {
      "key": "API_SECRET_KEY",
      "value": "sk_live_abc123...",
      "is_secret": true,
      "created_at": "2024-05-15T08:00:00Z",
      "updated_at": "2024-05-15T08:00:00Z"
    }
  ]
}
```

### 2.11 Update Environment Variables
**PUT** `/apps/{id}/env`

Request body:
```json
{
  "variables": [
    { "key": "NODE_ENV", "value": "production", "is_secret": false },
    { "key": "NEW_VAR", "value": "hello", "is_secret": false }
  ],
  "delete_keys": ["OLD_VAR"]
}
```

Response `200 OK`:
```json
{
  "message": "Environment variables updated. Restart app to apply changes.",
  "updated_count": 2,
  "deleted_count": 1
}
```

### 2.12 Import Environment from .env
**POST** `/apps/{id}/env/import`

Request body (multipart/form-data or JSON):
```json
{
  "content": "NODE_ENV=production
PORT=3000
API_KEY=secret123"
}
```

Response `200 OK`:
```json
{
  "message": "Imported 3 variables.",
  "imported": 3,
  "skipped": 0
}
```

### 2.13 Get App Volumes
**GET** `/apps/{id}/volumes`

Response `200 OK`:
```json
{
  "app_id": "app_123456",
  "volumes": [
    {
      "id": "vol_001",
      "host_path": "/var/panel/apps/app_123456/volumes/data",
      "container_path": "/app/data",
      "size_mb": 2345,
      "created_at": "2024-05-15T08:00:00Z"
    }
  ]
}
```

### 2.14 Add Volume
**POST** `/apps/{id}/volumes`

Request:
```json
{
  "container_path": "/app/uploads",
  "name": "uploads"
}
```

Response `201 Created`:
```json
{
  "id": "vol_002",
  "host_path": "/var/panel/apps/app_123456/volumes/uploads",
  "container_path": "/app/uploads",
  "size_mb": 0,
  "created_at": "2024-06-01T12:00:00Z"
}
```

### 2.15 Delete Volume
**DELETE** `/apps/{id}/volumes/{volume_id}`

Query parameter: `delete_data` (default: false)

Response `200 OK`:
```json
{
  "message": "Volume removed from app configuration.",
  "data_deleted": false
}
```

---

## 3. Deployments

### 3.1 List Deployments
**GET** `/apps/{id}/deployments`

Query parameters:
- `status` (optional): `success`, `failed`, `in_progress`, `all`
- `page`, `per_page`

Response `200 OK`:
```json
{
  "data": [
    {
      "id": "dep_789012",
      "app_id": "app_123456",
      "status": "success",
      "commit": "abc1234",
      "commit_message": "Fix auth middleware",
      "commit_author": "john",
      "branch": "main",
      "build_duration_seconds": 45,
      "deploy_duration_seconds": 3,
      "started_at": "2024-06-01T12:34:00Z",
      "completed_at": "2024-06-01T12:34:56Z",
      "triggered_by": "git_push",
      "triggered_by_user": "john"
    }
  ],
  "meta": {
    "total": 24,
    "page": 1,
    "per_page": 20
  }
}
```

### 3.2 Trigger Deployment
**POST** `/apps/{id}/deploy`

Request body:
```json
{
  "branch": "main",
  "commit": "abc1234",
  "force": false
}
```

- `branch` (optional): defaults to configured branch
- `commit` (optional): deploy specific commit, defaults to latest
- `force` (optional): redeploy even if commit hasn't changed

Response `202 Accepted`:
```json
{
  "deployment_id": "dep_789013",
  "status": "queued",
  "message": "Deployment queued."
}
```

### 3.3 Get Deployment
**GET** `/deployments/{deployment_id}`

Response `200 OK`:
```json
{
  "id": "dep_789012",
  "app_id": "app_123456",
  "status": "success",
  "commit": "abc1234",
  "commit_message": "Fix auth middleware",
  "commit_author": "john",
  "branch": "main",
  "build_logs_url": "/api/v1/deployments/dep_789012/logs",
  "build_duration_seconds": 45,
  "deploy_duration_seconds": 3,
  "started_at": "2024-06-01T12:34:00Z",
  "completed_at": "2024-06-01T12:34:56Z",
  "triggered_by": "git_push",
  "triggered_by_user": "john"
}
```

### 3.4 Get Deployment Logs
**GET** `/deployments/{deployment_id}/logs`

Response `200 OK`:
```json
{
  "deployment_id": "dep_789012",
  "lines": [
    { "timestamp": "2024-06-01T12:34:01Z", "level": "info", "message": "Cloning repository..." },
    { "timestamp": "2024-06-01T12:34:05Z", "level": "info", "message": "Installing dependencies..." },
    { "timestamp": "2024-06-01T12:34:30Z", "level": "info", "message": "Building application..." },
    { "timestamp": "2024-06-01T12:34:55Z", "level": "info", "message": "Build successful." },
    { "timestamp": "2024-06-01T12:34:56Z", "level": "info", "message": "Deployment complete." }
  ]
}
```

**WebSocket:** Connect to `/api/v1/stream` and subscribe to `deployment.{deployment_id}.logs` for real-time build logs.

### 3.5 Rollback Deployment
**POST** `/apps/{id}/rollback`

Request body:
```json
{
  "deployment_id": "dep_789012"
}
```

Response `202 Accepted`:
```json
{
  "message": "Rollback initiated.",
  "new_deployment_id": "dep_789014",
  "target_deployment_id": "dep_789012"
}
```

### 3.6 Cancel Deployment
**POST** `/deployments/{deployment_id}/cancel`

Only works if deployment is `queued` or `in_progress`.

Response `200 OK`:
```json
{
  "message": "Deployment cancelled.",
  "status": "cancelled"
}
```

---

## 4. Services

### 4.1 List Services
**GET** `/services`

Query parameters: `type`, `status`, `search`, `page`, `per_page`

Response `200 OK`:
```json
{
  "data": [
    {
      "id": "svc_789",
      "name": "main-pg",
      "type": "postgresql",
      "version": "15.4",
      "status": "running",
      "port": 5432,
      "data_size_mb": 2345,
      "connected_apps": 3,
      "resource_usage": {
        "cpu_percent": 5.2,
        "memory_mb": 256
      },
      "last_backup_at": "2024-06-01T02:00:00Z",
      "created_at": "2024-05-10T10:00:00Z",
      "updated_at": "2024-05-10T10:00:00Z"
    }
  ],
  "meta": {
    "total": 4,
    "page": 1,
    "per_page": 20
  }
}
```

### 4.2 Get Service
**GET** `/services/{id}`

Response `200 OK`:
```json
{
  "id": "svc_789",
  "name": "main-pg",
  "type": "postgresql",
  "version": "15.4",
  "status": "running",
  "port": 5432,
  "internal_host": "main-pg",
  "container_id": "g7h8i9j0",
  "data_size_mb": 2345,
  "resource_usage": {
    "cpu_percent": 5.2,
    "memory_mb": 256,
    "memory_limit_mb": 512,
    "connections_active": 12,
    "connections_max": 100
  },
  "credentials": {
    "host": "localhost",
    "port": 5432,
    "database": "main-pg",
    "username": "main-pg-user",
    "password": "auto_generated_password_123",
    "connection_string": "postgres://main-pg-user:auto_generated_password_123@localhost:5432/main-pg"
  },
  "backup_schedule": {
    "enabled": true,
    "frequency": "daily",
    "time": "02:00",
    "timezone": "UTC",
    "retention_days": 7,
    "destination": "s3"
  },
  "connected_apps": [
    { "id": "app_123456", "name": "api-prod" },
    { "id": "app_123457", "name": "web-client" }
  ],
  "created_at": "2024-05-10T10:00:00Z",
  "updated_at": "2024-05-10T10:00:00Z"
}
```

### 4.3 Create Service
**POST** `/services`

Request:
```json
{
  "name": "analytics-pg",
  "type": "postgresql",
  "version": "15",
  "port": 5433,
  "root_password": "custom_password_123",
  "resources": {
    "memory_limit_mb": 256
  },
  "backup_schedule": {
    "enabled": true,
    "frequency": "daily",
    "time": "03:00",
    "retention_days": 7,
    "destination": "s3"
  }
}
```

- `port` (optional): auto-assigned if not provided
- `root_password` (optional): auto-generated if not provided
- `backup_schedule` (optional): uses defaults if not provided

Response `201 Created`:
```json
{
  "id": "svc_999",
  "name": "analytics-pg",
  "type": "postgresql",
  "status": "creating",
  "port": 5433,
  "credentials": {
    "host": "localhost",
    "port": 5433,
    "database": "analytics-pg",
    "username": "analytics-pg-user",
    "password": "auto_generated_password_456",
    "connection_string": "postgres://analytics-pg-user:auto_generated_password_456@localhost:5433/analytics-pg"
  },
  "message": "Service is being provisioned. This may take 30-60 seconds."
}
```

### 4.4 Update Service
**PUT** `/services/{id}`

Partial update. Only `backup_schedule`, `resources`, `name` can be updated. Version changes require recreate.

### 4.5 Delete Service
**DELETE** `/services/{id}`

Query parameter: `force` (default: false) — allows deletion even if apps are connected.

Response `200 OK`:
```json
{
  "message": "Service 'main-pg' deleted.",
  "backup_created": true,
  "backup_id": "bak_123"
}
```

### 4.6 Restart Service
**POST** `/services/{id}/restart`

### 4.7 Get Service Logs
**GET** `/services/{id}/logs`

Same query parameters as app logs.

### 4.8 Test Connection
**POST** `/services/{id}/test-connection`

Tests connectivity from the panel to the service using stored credentials.

Response `200 OK`:
```json
{
  "success": true,
  "latency_ms": 2,
  "message": "Connected to PostgreSQL 15.4."
}
```

Response `503 Service Unavailable`:
```json
{
  "success": false,
  "latency_ms": null,
  "message": "Connection refused. Is the service running?"
}
```

---

## 5. Backups

### 5.1 List Backups
**GET** `/backups`

Query parameters: `app_id`, `service_id`, `status`, `destination`, `page`, `per_page`

Response `200 OK`:
```json
{
  "data": [
    {
      "id": "bak_123",
      "name": "main-pg-20240601-020000",
      "target_type": "service",
      "target_id": "svc_789",
      "target_name": "main-pg",
      "status": "success",
      "size_mb": 234,
      "destination": "s3",
      "destination_path": "s3://my-bucket/backups/main-pg-20240601-020000.sql.gz",
      "started_at": "2024-06-01T02:00:00Z",
      "completed_at": "2024-06-01T02:01:30Z",
      "triggered_by": "schedule",
      "checksum": "sha256:abc123..."
    }
  ],
  "meta": {
    "total": 42,
    "page": 1,
    "per_page": 20
  }
}
```

### 5.2 Create Backup
**POST** `/backups`

Request:
```json
{
  "target_type": "service",
  "target_id": "svc_789",
  "destination": "s3"
}
```

Response `202 Accepted`:
```json
{
  "backup_id": "bak_124",
  "status": "in_progress",
  "message": "Backup started."
}
```

### 5.3 Restore Backup
**POST** `/backups/{id}/restore`

Request:
```json
{
  "target_id": "svc_789",
  "create_new": false
}
```

- `target_id` (optional): restore to different service/app
- `create_new` (optional): if true, creates a new service/app from backup instead of overwriting

Response `202 Accepted`:
```json
{
  "restore_id": "rst_456",
  "status": "in_progress",
  "message": "Restoring backup to 'main-pg'. A snapshot of current data was created first."
}
```

### 5.4 Delete Backup
**DELETE** `/backups/{id}`

Response `200 OK`:
```json
{
  "message": "Backup deleted from S3 and local index."
}
```

### 5.5 Get Backup Settings
**GET** `/backup-settings`

Response `200 OK`:
```json
{
  "default_schedule": {
    "frequency": "daily",
    "time": "02:00",
    "timezone": "UTC",
    "retention_days": 7
  },
  "default_destination": "s3",
  "s3_config": {
    "endpoint": "https://s3.amazonaws.com",
    "bucket": "my-backup-bucket",
    "region": "us-east-1",
    "path_prefix": "backups/"
  }
}
```

### 5.6 Update Backup Settings
**PUT** `/backup-settings`

---

## 6. Server

### 6.1 Get Server Info
**GET** `/server`

Response `200 OK`:
```json
{
  "hostname": "my-vps-01",
  "os": "Ubuntu 24.04 LTS",
  "kernel": "6.8.0-31-generic",
  "architecture": "amd64",
  "panel_version": "1.2.3",
  "resources": {
    "cpu_cores": 4,
    "cpu_model": "AMD EPYC 7B13",
    "memory_total_mb": 8192,
    "disk_total_gb": 100,
    "disk_used_gb": 45
  },
  "uptime_seconds": 1234567,
  "timezone": "UTC",
  "created_at": "2024-05-01T00:00:00Z"
}
```

### 6.2 Get Server Metrics
**GET** `/server/metrics`

Query parameters:
- `metric` (optional): `cpu`, `memory`, `disk`, `network`, `load` (default: all)
- `range` (optional): `1h`, `6h`, `24h`, `7d` (default: `1h`)

Response `200 OK`:
```json
{
  "cpu": {
    "current_percent": 34.2,
    "per_core": [12.5, 45.2, 28.1, 51.0],
    "history": [
      { "timestamp": "2024-06-01T11:00:00Z", "value": 30.1 },
      { "timestamp": "2024-06-01T11:05:00Z", "value": 32.4 },
      { "timestamp": "2024-06-01T11:10:00Z", "value": 34.2 }
    ]
  },
  "memory": {
    "current_mb": 4960,
    "total_mb": 8192,
    "percent": 60.5,
    "history": [
      { "timestamp": "2024-06-01T11:00:00Z", "value": 4800 },
      { "timestamp": "2024-06-01T11:05:00Z", "value": 4900 },
      { "timestamp": "2024-06-01T11:10:00Z", "value": 4960 }
    ]
  },
  "disk": {
    "used_gb": 45,
    "total_gb": 100,
    "percent": 45.0,
    "io_read_mbps": 12.5,
    "io_write_mbps": 3.2
  },
  "network": {
    "inbound_mbps": 45.2,
    "outbound_mbps": 12.8,
    "connections_active": 234
  },
  "load": {
    "1min": 0.45,
    "5min": 0.52,
    "15min": 0.38
  }
}
```

### 6.3 Get Processes
**GET** `/server/processes`

Query parameters: `sort` (cpu, memory, pid), `search`

Response `200 OK`:
```json
{
  "processes": [
    {
      "pid": 1234,
      "name": "node",
      "user": "app_123456",
      "cpu_percent": 12.5,
      "memory_mb": 256,
      "memory_percent": 3.1,
      "time": "2d 04:12",
      "command": "node server.js"
    }
  ],
  "total_count": 87
}
```

### 6.4 Kill Process
**POST** `/server/processes/{pid}/kill`

Request:
```json
{
  "signal": "SIGTERM"
}
```

Response `200 OK`:
```json
{
  "message": "Signal SIGTERM sent to process 1234."
}
```

### 6.5 Get Disk Usage
**GET** `/server/disks`

Response `200 OK`:
```json
{
  "disks": [
    {
      "mount": "/",
      "filesystem": "ext4",
      "total_gb": 100,
      "used_gb": 45,
      "free_gb": 55,
      "percent": 45
    }
  ],
  "largest_directories": [
    { "path": "/var/panel/apps", "size_gb": 12.3 },
    { "path": "/var/lib/docker", "size_gb": 8.5 }
  ]
}
```

### 6.6 Get Network Info
**GET** `/server/network`

Response `200 OK`:
```json
{
  "interfaces": [
    {
      "name": "eth0",
      "ip_address": "192.168.1.100",
      "mac_address": "00:11:22:33:44:55",
      "status": "up"
    }
  ],
  "open_ports": [
    { "port": 22, "protocol": "tcp", "service": "ssh" },
    { "port": 80, "protocol": "tcp", "service": "http" },
    { "port": 443, "protocol": "tcp", "service": "https" },
    { "port": 5432, "protocol": "tcp", "service": "main-pg" }
  ],
  "bandwidth_24h": {
    "inbound_gb": 45.2,
    "outbound_gb": 12.8
  }
}
```

### 6.7 Get Available Updates
**GET** `/server/updates`

Response `200 OK`:
```json
{
  "security_updates": 3,
  "total_updates": 12,
  "packages": [
    {
      "name": "linux-image-generic",
      "current_version": "6.8.0-31",
      "new_version": "6.8.0-35",
      "severity": "security"
    }
  ],
  "panel_update_available": true,
  "panel_current_version": "1.2.3",
  "panel_latest_version": "1.2.4",
  "panel_changelog": "Fixed memory leak in agent daemon."
}
```

### 6.8 Install Updates
**POST** `/server/updates`

Request:
```json
{
  "type": "security"
}
```

Response `202 Accepted`:
```json
{
  "message": "Installing 3 security updates. Server will not reboot."
}
```

### 6.9 Reboot Server
**POST** `/server/reboot`

Response `202 Accepted`:
```json
{
  "message": "Server reboot initiated. Panel will be unavailable for 30-60 seconds."
}
```

---

## 7. Domains & SSL

### 7.1 List Domains
**GET** `/domains`

Response `200 OK`:
```json
{
  "data": [
    {
      "domain": "api.example.com",
      "app_id": "app_123456",
      "app_name": "api-prod",
      "ssl_status": "valid",
      "ssl_provider": "letsencrypt",
      "ssl_issued_at": "2024-03-01T00:00:00Z",
      "ssl_expires_at": "2024-06-01T00:00:00Z",
      "ssl_auto_renew": true,
      "force_https": true,
      "dns_valid": true
    }
  ]
}
```

### 7.2 Add Domain to App
**POST** `/apps/{id}/domains`

Request:
```json
{
  "domain": "www.api.example.com",
  "force_https": true
}
```

Response `201 Created`:
```json
{
  "domain": "www.api.example.com",
  "ssl_status": "pending",
  "message": "Domain added. SSL certificate will be provisioned automatically."
}
```

### 7.3 Remove Domain
**DELETE** `/apps/{id}/domains/{domain}`

Response `200 OK`:
```json
{
  "message": "Domain 'www.api.example.com' removed from app 'api-prod'."
}
```

### 7.4 Renew SSL
**POST** `/domains/{domain}/renew`

Response `200 OK`:
```json
{
  "domain": "api.example.com",
  "ssl_status": "valid",
  "ssl_expires_at": "2024-09-01T00:00:00Z",
  "message": "SSL certificate renewed successfully."
}
```

Response `422 Unprocessable`:
```json
{
  "error": "dns_invalid",
  "message": "Domain DNS does not point to this server. Cannot issue SSL certificate."
}
```

### 7.5 Validate DNS
**GET** `/domains/{domain}/validate-dns`

Response `200 OK`:
```json
{
  "domain": "api.example.com",
  "dns_valid": true,
  "a_record": "192.168.1.100",
  "server_ip": "192.168.1.100",
  "message": "DNS is correctly configured."
}
```

---

## 8. Cron Jobs

### 8.1 List Cron Jobs
**GET** `/cron-jobs`

Response `200 OK`:
```json
{
  "data": [
    {
      "id": "cron_001",
      "name": "db-cleanup",
      "status": "active",
      "schedule": "0 2 * * *",
      "schedule_human": "Daily at 2:00 AM",
      "command": "npm run cleanup",
      "target": {
        "type": "app",
        "id": "app_123456",
        "name": "api-prod"
      },
      "last_run": {
        "started_at": "2024-06-01T02:00:00Z",
        "status": "success",
        "duration_seconds": 45
      },
      "next_run": "2024-06-02T02:00:00Z",
      "notify_on_failure": true,
      "log_retention": 10,
      "created_at": "2024-05-15T08:00:00Z"
    }
  ]
}
```

### 8.2 Create Cron Job
**POST** `/cron-jobs`

Request:
```json
{
  "name": "sitemap-generator",
  "schedule": "0 */6 * * *",
  "command": "php artisan sitemap:generate",
  "target": {
    "type": "app",
    "id": "app_123456"
  },
  "notify_on_failure": true,
  "log_retention": 10
}
```

Response `201 Created`:
```json
{
  "id": "cron_002",
  "name": "sitemap-generator",
  "status": "active",
  "next_run": "2024-06-01T18:00:00Z",
  "message": "Cron job created."
}
```

### 8.3 Get Cron Job
**GET** `/cron-jobs/{id}`

### 8.4 Update Cron Job
**PUT** `/cron-jobs/{id}`

### 8.5 Delete Cron Job
**DELETE** `/cron-jobs/{id}`

### 8.6 Get Execution History
**GET** `/cron-jobs/{id}/history`

Response `200 OK`:
```json
{
  "data": [
    {
      "id": "exec_123",
      "cron_job_id": "cron_001",
      "started_at": "2024-06-01T02:00:00Z",
      "completed_at": "2024-06-01T02:00:45Z",
      "status": "success",
      "exit_code": 0,
      "output": "Cleanup completed. 45 rows deleted.
",
      "error_output": ""
    }
  ]
}
```

---

## 9. Firewall

### 9.1 Get Firewall Status
**GET** `/firewall`

Response `200 OK`:
```json
{
  "enabled": true,
  "backend": "ufw",
  "default_policy": {
    "incoming": "deny",
    "outgoing": "allow"
  },
  "rules": [
    {
      "id": "fw_001",
      "port": 22,
      "protocol": "tcp",
      "source": "any",
      "action": "allow",
      "description": "SSH",
      "app_name": null
    },
    {
      "id": "fw_002",
      "port": 443,
      "protocol": "tcp",
      "source": "any",
      "action": "allow",
      "description": "HTTPS",
      "app_name": null
    },
    {
      "id": "fw_003",
      "port": 3000,
      "protocol": "tcp",
      "source": "10.0.0.0/8",
      "action": "allow",
      "description": "API internal",
      "app_name": "api-prod"
    }
  ],
  "recent_blocks": [
    {
      "timestamp": "2024-06-01T12:34:56Z",
      "source_ip": "192.168.1.45",
      "port": 22,
      "protocol": "tcp",
      "reason": "brute_force"
    }
  ]
}
```

### 9.2 Add Rule
**POST** `/firewall/rules`

Request:
```json
{
  "port": 8080,
  "protocol": "tcp",
  "source": "any",
  "action": "allow",
  "description": "Custom API port"
}
```

Response `201 Created`:
```json
{
  "id": "fw_004",
  "port": 8080,
  "protocol": "tcp",
  "source": "any",
  "action": "allow",
  "description": "Custom API port",
  "message": "Firewall rule added."
}
```

### 9.3 Delete Rule
**DELETE** `/firewall/rules/{id}`

### 9.4 Enable/Disable Firewall
**POST** `/firewall/toggle`

Request:
```json
{
  "enabled": true
}
```

---

## 10. Users & Team

### 10.1 Get Current User
**GET** `/users/me`

Response `200 OK`:
```json
{
  "id": 1,
  "username": "admin",
  "email": "admin@example.com",
  "role": "owner",
  "two_factor_enabled": true,
  "created_at": "2024-05-01T00:00:00Z",
  "last_login_at": "2024-06-01T12:00:00Z",
  "last_login_ip": "192.168.1.50"
}
```

### 10.2 List Team Members
**GET** `/users`

Response `200 OK`:
```json
{
  "data": [
    {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "role": "owner",
      "status": "active",
      "last_active_at": "2024-06-01T12:00:00Z"
    },
    {
      "id": 2,
      "username": "john",
      "email": "john@example.com",
      "role": "developer",
      "status": "active",
      "last_active_at": "2024-06-01T10:00:00Z"
    }
  ]
}
```

### 10.3 Invite User
**POST** `/users/invite`

Request:
```json
{
  "email": "sarah@example.com",
  "role": "developer"
}
```

Response `201 Created`:
```json
{
  "invite_id": "inv_123",
  "email": "sarah@example.com",
  "role": "developer",
  "status": "pending",
  "invite_url": "https://panel.example.com/invite/inv_123",
  "expires_at": "2024-06-08T12:00:00Z"
}
```

### 10.4 Update User Role
**PUT** `/users/{id}/role`

Request:
```json
{
  "role": "admin"
}
```

### 10.5 Delete User
**DELETE** `/users/{id}`

### 10.6 List API Keys
**GET** `/users/me/api-keys`

Response `200 OK`:
```json
{
  "data": [
    {
      "id": "key_001",
      "name": "CI/CD Deploy",
      "scopes": ["deploy", "read"],
      "last_used_at": "2024-06-01T10:00:00Z",
      "created_at": "2024-05-15T08:00:00Z"
    }
  ]
}
```

### 10.7 Create API Key
**POST** `/users/me/api-keys`

Request:
```json
{
  "name": "GitHub Actions",
  "scopes": ["deploy"]
}
```

Response `201 Created`:
```json
{
  "id": "key_002",
  "name": "GitHub Actions",
  "token": "pk_live_abc123...",
  "scopes": ["deploy"],
  "created_at": "2024-06-01T12:00:00Z",
  "warning": "This token is shown only once. Copy it now."
}
```

### 10.8 Revoke API Key
**DELETE** `/users/me/api-keys/{id}`

---

## 11. Settings

### 11.1 Get Panel Settings
**GET** `/settings/panel`

Response `200 OK`:
```json
{
  "panel_name": "My Panel",
  "panel_domain": "panel.example.com",
  "default_app_subdomain": "{app}.panel.example.com",
  "default_build_strategy": "nixpacks",
  "auto_check_updates": true,
  "maintenance_window": "02:00-04:00 UTC",
  "backup_defaults": {
    "frequency": "daily",
    "time": "02:00",
    "retention_days": 7,
    "destination": "s3"
  }
}
```

### 11.2 Update Panel Settings
**PUT** `/settings/panel`

### 11.3 Get Server Settings
**GET** `/settings/server`

Response `200 OK`:
```json
{
  "hostname": "my-vps-01",
  "timezone": "UTC",
  "swap_enabled": true,
  "swap_size_mb": 2048,
  "auto_security_updates": true,
  "update_schedule": "daily"
}
```

### 11.4 Update Server Settings
**PUT** `/settings/server`

### 11.5 Get Notification Settings
**GET** `/settings/notifications`

Response `200 OK`:
```json
{
  "email": {
    "enabled": true,
    "smtp_host": "smtp.example.com",
    "smtp_port": 587,
    "smtp_username": "alerts@example.com",
    "from_address": "panel@example.com"
  },
  "webhooks": {
    "slack": "https://hooks.slack.com/...",
    "discord": "https://discord.com/api/webhooks/..."
  },
  "events": {
    "deployment_failed": { "email": true, "webhook": true },
    "deployment_succeeded": { "email": false, "webhook": false },
    "ssl_expiring": { "email": true, "webhook": true },
    "backup_failed": { "email": true, "webhook": true },
    "server_resource_alert": { "email": true, "webhook": true }
  }
}
```

### 11.6 Update Notification Settings
**PUT** `/settings/notifications`

### 11.7 Export Panel Data
**POST** `/settings/export`

Response `200 OK` (triggers file download):
```json
{
  "download_url": "/api/v1/settings/export/download?token=abc123",
  "expires_at": "2024-06-01T13:00:00Z"
}
```

---

## 12. WebSocket Events

### 12.1 Connection
Connect to `wss://panel.example.com/api/v1/stream` with the JWT access token in the `Authorization` header.

### 12.2 Authentication
Send auth message immediately after connection:
```json
{
  "type": "auth",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

Response:
```json
{
  "type": "auth_success",
  "message": "Authenticated as admin."
}
```

### 12.3 Subscribing to Events
```json
{
  "type": "subscribe",
  "channels": ["app.app_123456", "server.metrics", "deployments"]
}
```

Response:
```json
{
  "type": "subscribed",
  "channels": ["app.app_123456", "server.metrics", "deployments"]
}
```

### 12.4 Event Types

#### App Events
```json
{
  "type": "app.deploy.started",
  "timestamp": "2024-06-01T12:34:00Z",
  "app_id": "app_123456",
  "deployment_id": "dep_789013",
  "commit": "abc1234",
  "branch": "main"
}
```

```json
{
  "type": "app.deploy.progress",
  "timestamp": "2024-06-01T12:34:30Z",
  "app_id": "app_123456",
  "deployment_id": "dep_789013",
  "step": "building",
  "message": "Installing dependencies...",
  "percent": 45
}
```

```json
{
  "type": "app.deploy.success",
  "timestamp": "2024-06-01T12:34:56Z",
  "app_id": "app_123456",
  "deployment_id": "dep_789013",
  "duration_seconds": 56
}
```

```json
{
  "type": "app.deploy.failed",
  "timestamp": "2024-06-01T12:34:20Z",
  "app_id": "app_123456",
  "deployment_id": "dep_789013",
  "error": "Build failed: npm ERR! missing script: build",
  "step": "building"
}
```

```json
{
  "type": "app.logs",
  "timestamp": "2024-06-01T12:35:01Z",
  "app_id": "app_123456",
  "stream": "stdout",
  "message": "GET /api/users 200 45ms"
}
```

```json
{
  "type": "app.status_changed",
  "timestamp": "2024-06-01T12:35:00Z",
  "app_id": "app_123456",
  "old_status": "deploying",
  "new_status": "running",
  "health_status": "healthy"
}
```

#### Service Events
```json
{
  "type": "service.metrics",
  "timestamp": "2024-06-01T12:35:00Z",
  "service_id": "svc_789",
  "cpu_percent": 5.2,
  "memory_mb": 256,
  "connections": 12
}
```

```json
{
  "type": "service.backup.completed",
  "timestamp": "2024-06-01T02:01:30Z",
  "service_id": "svc_789",
  "backup_id": "bak_123",
  "size_mb": 234
}
```

#### Server Events
```json
{
  "type": "server.metrics",
  "timestamp": "2024-06-01T12:35:00Z",
  "cpu_percent": 34.2,
  "memory_percent": 60.5,
  "disk_percent": 45.0,
  "load_1min": 0.45
}
```

```json
{
  "type": "server.alert",
  "timestamp": "2024-06-01T12:35:00Z",
  "severity": "warning",
  "metric": "cpu",
  "value": 85.2,
  "threshold": 80,
  "message": "CPU usage is above 80% for 5 minutes."
}
```

#### Notification Events
```json
{
  "type": "notification",
  "timestamp": "2024-06-01T12:34:56Z",
  "id": "notif_001",
  "title": "Deployment failed",
  "message": "App 'api-prod' deployment failed: Build error.",
  "severity": "error",
  "link": "/apps/app_123456/deployments",
  "read": false
}
```

### 12.5 Ping/Pong
Client sends ping every 30 seconds:
```json
{ "type": "ping" }
```

Server responds:
```json
{ "type": "pong", "timestamp": "2024-06-01T12:35:30Z" }
```

If server does not respond within 10 seconds, client should reconnect.

---

## 13. Error Handling

### 13.1 Error Response Format
All errors follow this structure:
```json
{
  "error": "error_code_snake_case",
  "message": "Human-readable description of what went wrong.",
  "details": {
    "field": "specific_field",
    "reason": "why_it_failed"
  },
  "request_id": "req_abc123",
  "timestamp": "2024-06-01T12:34:56Z"
}
```

### 13.2 HTTP Status Codes
| Code | Meaning | When Used |
|------|---------|-----------|
| `200` | OK | Successful GET, PUT, DELETE |
| `201` | Created | Successful POST creating a resource |
| `202` | Accepted | Long-running operation started (deploy, backup, restore) |
| `400` | Bad Request | Invalid request body, missing required fields |
| `401` | Unauthorized | Missing or invalid JWT token |
| `403` | Forbidden | Valid token but insufficient permissions (role-based) |
| `404` | Not Found | Resource does not exist |
| `409` | Conflict | Resource already exists (duplicate name), or resource is in a state that prevents the action |
| `422` | Unprocessable | Validation failed (DNS not pointing, Git repo inaccessible, invalid cron expression) |
| `429` | Too Many Requests | Rate limit exceeded |
| `500` | Internal Server Error | Unexpected server error. `request_id` is provided for support. |
| `503` | Service Unavailable | Service is down or temporarily unavailable (database connection failed) |

### 13.3 Common Error Codes
| Code | HTTP Status | Description |
|------|-------------|-------------|
| `invalid_credentials` | 401 | Wrong username or password |
| `totp_required` | 403 | 2FA code required but not provided |
| `invalid_totp` | 403 | 2FA code is incorrect |
| `token_expired` | 401 | JWT access token has expired |
| `invalid_refresh_token` | 401 | Refresh token is invalid or revoked |
| `insufficient_permissions` | 403 | User role does not allow this action |
| `app_not_found` | 404 | App ID does not exist |
| `service_not_found` | 404 | Service ID does not exist |
| `app_name_exists` | 409 | App name already taken |
| `service_name_exists` | 409 | Service name already taken |
| `deployment_in_progress` | 409 | Cannot start new deployment while one is running |
| `volumes_exist` | 400 | App has persistent volumes; explicit flag required to delete |
| `invalid_git_url` | 422 | Repository URL is not accessible |
| `invalid_cron_expression` | 422 | Cron expression syntax is invalid |
| `dns_invalid` | 422 | Domain DNS does not point to this server |
| `port_conflict` | 409 | Port is already in use by another app or service |
| `resource_limit_exceeded` | 422 | Requested resources exceed server capacity |
| `backup_in_progress` | 409 | Cannot start backup; another backup is running for this target |
| `rate_limit_exceeded` | 429 | Too many requests |
| `internal_error` | 500 | Unexpected server error |

---

## 14. Rate Limiting

### 14.1 Limits
| Endpoint Group | Limit | Window |
|----------------|-------|--------|
| Authentication (login) | 5 requests | 1 minute |
| Authentication (refresh) | 10 requests | 1 minute |
| All API endpoints | 100 requests | 1 minute |
| WebSocket connections | 5 connections | Per IP |
| Deployment triggers | 10 requests | 1 minute |
| Backup creation | 5 requests | 1 hour |

### 14.2 Headers
All responses include rate limit headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1717257600
```

### 14.3 Exceeded Response
```json
{
  "error": "rate_limit_exceeded",
  "message": "Rate limit exceeded. Try again in 45 seconds.",
  "retry_after": 45
}
```

---

*End of API Specification*
