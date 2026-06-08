# Juvia Panel - Comprehensive Backend Testing Plan

**Version:** 1.0  
**Date:** 2026-06-08  
**Scope:** Full Backend System Testing

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Test Environment Setup](#test-environment-setup)
3. [Phase 1: Infrastructure Tests](#phase-1-infrastructure-tests)
4. [Phase 2: Authentication & Authorization](#phase-2-authentication--authorization)
5. [Phase 3: Application Management](#phase-3-application-management)
6. [Phase 4: Service Management](#phase-4-service-management)
7. [Phase 5: Server Management](#phase-5-server-management)
8. [Phase 6: Real-time & WebSocket](#phase-6-real-time--websocket)
9. [Phase 7: Security Testing](#phase-7-security-testing)
10. [Phase 8: Performance Testing](#phase-8-performance-testing)
11. [Phase 9: Integration Testing](#phase-9-integration-testing)
12. [Phase 10: Disaster Recovery](#phase-10-disaster-recovery)
13. [Test Execution Checklist](#test-execution-checklist)

---

## Prerequisites

### Test Accounts
| Role | Username | Password | Permissions |
|------|----------|----------|-------------|
| Owner | admin | (set during setup) | Full access |
| Admin | admin2 | (invite required) | Admin access |
| Developer | dev1 | (invite required) | Deploy, manage apps |
| Viewer | viewer1 | (invite required) | Read-only access |

### Test Tools
```bash
# Install testing tools
apt-get install -y curl jq sqlite3 apache2-utils

# For load testing
npm install -g autocannon

# For WebSocket testing
npm install -g wscat
```

### Environment Variables
```bash
export API_URL="http://localhost:9090"
export PANEL_DOMAIN="103.143.0.169:2053"
export TEST_TOKEN=""
```

---

## Test Environment Setup

### 1.1 Reset Test Database
```bash
# Backup current database
sudo cp /var/panel/panel.db /var/panel/panel.db.backup

# Reset to fresh state
sudo juvia reset

# Verify clean state
sqlite3 /var/panel/panel.db "SELECT * FROM users;"
# Should show only admin user or empty state
```

### 1.2 Create Test Users
```bash
# Create via UI or API
# Owner: admin (created during setup)
# Then invite other users
```

### 1.3 Verify Base Services
```bash
# All services running
systemctl status juvia-api juvia-agent juvia-caddy docker

# All ports listening
ss -tlnp | grep -E "9090|9091|2053|2375"
```

---

## Phase 1: Infrastructure Tests

### 1.1 Database Tests

#### 1.1.1 Database Connectivity
```bash
# Test SQLite file exists and is readable
ls -la /var/panel/panel.db
file /var/panel/panel.db

# Test direct SQLite connection
sqlite3 /var/panel/panel.db "SELECT datetime('now');"

# Test WAL mode enabled
sqlite3 /var/panel/panel.db "PRAGMA journal_mode;"
# Expected: wal
```

#### 1.1.2 Schema Integrity
```bash
# List all tables
sqlite3 /var/panel/panel.db ".tables"
# Expected: activity_log, api_keys, app_domains, app_env_vars, app_volumes, 
#          apps, backups, cron_executions, cron_jobs, firewall_rules,
#          notifications, schema_migrations, server_info, service_app_links,
#          services, sessions, user_invites, users

# Verify all required columns
sqlite3 /var/panel/panel.db "PRAGMA table_info(users);"
sqlite3 /var/panel/panel.db "PRAGMA table_info(apps);"
sqlite3 /var/panel/panel.db "PRAGMA table_info(services);"

# Check foreign keys enabled
sqlite3 /var/panel/panel.db "PRAGMA foreign_keys;"
# Expected: foreign_keys=on
```

#### 1.1.3 Migration Tests
```bash
# Check migration table
sqlite3 /var/panel/panel.db "SELECT * FROM schema_migrations ORDER BY version;"

# Verify all migrations applied
# Expected: shows applied migrations with dirty=0
```

#### 1.1.4 Database Performance
```bash
# Check database size
ls -lh /var/panel/panel.db

# Check for corruption
sqlite3 /var/panel/panel.db "PRAGMA integrity_check;"
# Expected: ok

# Check WAL file size (if in WAL mode)
ls -lh /var/panel/panel.db-wal /var/panel/panel.db-shm
```

### 1.2 API Server Tests

#### 1.2.1 Server Startup
```bash
# Check service is running
systemctl status juvia-api

# Check process
ps aux | grep juvia-api

# Check port binding
ss -tlnp | grep 9090
# Expected: LISTEN on 127.0.0.1:9090
```

#### 1.2.2 Health Endpoint
```bash
# Basic health check
curl -s http://localhost:9090/health
# Expected: {"status":"ok","request_id":"..."}

# Health with detailed info
curl -s http://localhost:9090/health/detailed
```

#### 1.2.3 API Version
```bash
# Get API version
curl -s http://localhost:9090/api/v1/server | jq '.panel_version'

# Verify matches installed version
juvia-api --version 2>/dev/null || echo "Version flag not available"
```

#### 1.2.4 CORS Headers
```bash
# Test CORS preflight
curl -s -X OPTIONS http://localhost:9090/api/v1/server \
  -H "Origin: http://example.com" \
  -H "Access-Control-Request-Method: GET"
# Expected: CORS headers present
```

### 1.3 Agent Daemon Tests

#### 1.3.1 Agent Startup
```bash
# Check service status
systemctl status juvia-agent

# Check process
ps aux | grep juvia-agent

# Check socket exists
ls -la /var/run/panel/agent.sock
```

#### 1.3.2 Agent Communication
```bash
# Test agent TCP port (if exposed)
curl -s http://localhost:9091/health 2>/dev/null || echo "Agent TCP not exposed"

# Check agent logs
journalctl -u juvia-agent -n 20 --no-pager
```

### 1.4 Docker Integration Tests

#### 1.4.1 Docker Daemon
```bash
# Check Docker is running
systemctl status docker
docker version

# Check Docker socket
ls -la /var/run/docker.sock
```

#### 1.4.2 Docker Networks
```bash
# List panel networks
docker network ls | grep panel

# Check network configuration
docker network inspect panel_apps 2>/dev/null || echo "Network may not exist yet"
```

#### 1.4.3 Docker Images
```bash
# List panel images
docker images | grep -E "juvia|panel"

# Check image pulling
docker pull alpine:latest 2>/dev/null && echo "Docker pull works"
```

---

## Phase 2: Authentication & Authorization

### 2.1 User Authentication Tests

#### 2.1.1 Login with Valid Credentials
```bash
curl -s -X POST http://localhost:9090/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "yourpassword"
  }' | jq .
# Expected: 200 OK with access_token, token_type, expires_in, user
```

#### 2.1.2 Login with Invalid Password
```bash
curl -s -X POST http://localhost:9090/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "wrongpassword"
  }' | jq .
# Expected: 401 Unauthorized with error "invalid_credentials"
```

#### 2.1.3 Login with Non-existent User
```bash
curl -s -X POST http://localhost:9090/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "nonexistent",
    "password": "anypassword"
  }' | jq .
# Expected: 401 Unauthorized
```

#### 2.1.4 Login with Empty Credentials
```bash
curl -s -X POST http://localhost:9090/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "", "password": ""}' | jq .
# Expected: 400 Bad Request
```

### 2.2 Token Management Tests

#### 2.2.1 Token Refresh
```bash
# Get refresh token from login response
REFRESH_TOKEN="<refresh_token_cookie>"

curl -s -X POST http://localhost:9090/auth/refresh \
  -H "Content-Type: application/json" \
  -b "refresh_token=$REFRESH_TOKEN" | jq .
# Expected: 200 OK with new access_token
```

#### 2.2.2 Token Expiry
```bash
# Use expired token
curl -s -X GET http://localhost:9090/apps \
  -H "Authorization: Bearer expired_token_here" | jq .
# Expected: 401 Unauthorized with error "token_expired"
```

#### 2.2.3 Logout
```bash
curl -s -X POST http://localhost:9090/auth/logout \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: 200 OK
```

### 2.3 Role-Based Access Control Tests

#### 2.3.1 Owner Permissions
```bash
# Owner can do everything
curl -s -X GET http://localhost:9090/users -H "Authorization: Bearer $OWNER_TOKEN" | jq .
# Expected: 200 OK, list of users
```

#### 2.3.2 Admin Permissions
```bash
# Admin can manage apps and services
curl -s -X POST http://localhost:9090/apps -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "test-app", "runtime": "nodejs", "source": {...}}' | jq .
# Expected: 201 Created
```

#### 2.3.3 Developer Permissions
```bash
# Developer can deploy
curl -s -X POST http://localhost:9090/apps/app_xxx/deploy \
  -H "Authorization: Bearer $DEV_TOKEN" | jq .
# Expected: 202 Accepted

# Developer cannot delete app
curl -s -X DELETE http://localhost:9090/apps/app_xxx \
  -H "Authorization: Bearer $DEV_TOKEN" | jq .
# Expected: 403 Forbidden
```

#### 2.3.4 Viewer Permissions
```bash
# Viewer can read
curl -s -X GET http://localhost:9090/apps -H "Authorization: Bearer $VIEWER_TOKEN" | jq .
# Expected: 200 OK

# Viewer cannot create
curl -s -X POST http://localhost:9090/apps -H "Authorization: Bearer $VIEWER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "test"}' | jq .
# Expected: 403 Forbidden
```

### 2.4 Two-Factor Authentication Tests

#### 2.4.1 2FA Setup
```bash
# Get 2FA secret
curl -s -X POST http://localhost:9090/auth/2fa/setup \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: secret, qr_code_url, backup_codes
```

#### 2.4.2 2FA Verification
```bash
# Verify and enable
curl -s -X POST http://localhost:9090/auth/2fa/verify \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"totp_code": "123456", "secret": "JBSWY3DPEHPK3PXP"}' | jq .
# Expected: 200 OK
```

#### 2.4.3 Login with 2FA
```bash
curl -s -X POST http://localhost:9090/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "yourpassword",
    "totp_code": "123456"
  }' | jq .
# Expected: 200 OK with token
```

---

## Phase 3: Application Management

### 3.1 App Creation Tests

#### 3.1.1 Create App from Git
```bash
curl -s -X POST http://localhost:9090/apps \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-first-app",
    "runtime": "nodejs",
    "source": {
      "type": "git",
      "repo_url": "https://github.com/user/repo",
      "branch": "main"
    },
    "build": {
      "strategy": "nixpacks"
    }
  }' | jq .
# Expected: 201 Created with app_id
```

#### 3.1.2 Create App with Dockerfile
```bash
curl -s -X POST http://localhost:9090/apps \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "docker-app",
    "runtime": "docker",
    "source": {
      "type": "git",
      "repo_url": "https://github.com/user/repo",
      "branch": "main"
    },
    "build": {
      "strategy": "dockerfile"
    }
  }' | jq .
```

#### 3.1.3 Create App with Static Site
```bash
curl -s -X POST http://localhost:9090/apps \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "static-site",
    "runtime": "static",
    "source": {
      "type": "git",
      "repo_url": "https://github.com/user/static-site",
      "branch": "main"
    }
  }' | jq .
```

#### 3.1.4 Create App with Duplicate Name
```bash
curl -s -X POST http://localhost:9090/apps \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-first-app",
    "runtime": "nodejs"
  }' | jq .
# Expected: 409 Conflict with error "app_name_exists"
```

#### 3.1.5 Create App with Invalid Git URL
```bash
curl -s -X POST http://localhost:9090/apps \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "invalid-repo",
    "runtime": "nodejs",
    "source": {
      "type": "git",
      "repo_url": "https://invalid-repo-url-that-does-not-exist.com/repo"
    }
  }' | jq .
# Expected: 422 Unprocessable with error "invalid_git_url"
```

### 3.2 App Listing Tests

#### 3.2.1 List All Apps
```bash
curl -s -X GET http://localhost:9090/apps \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: 200 OK with data array
```

#### 3.2.2 Filter by Status
```bash
curl -s -X GET "http://localhost:9090/apps?status=running" \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: Only running apps

curl -s -X GET "http://localhost:9090/apps?status=failed" \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: Only failed apps
```

#### 3.2.3 Filter by Runtime
```bash
curl -s -X GET "http://localhost:9090/apps?runtime=nodejs" \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: Only Node.js apps
```

#### 3.2.4 Search Apps
```bash
curl -s -X GET "http://localhost:9090/apps?search=api" \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: Apps matching "api" in name or domain
```

#### 3.2.5 Pagination
```bash
curl -s -X GET "http://localhost:9090/apps?page=1&per_page=5" \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: Paginated response with meta info
```

### 3.3 App Update Tests

#### 3.3.1 Update App Name
```bash
curl -s -X PUT http://localhost:9090/apps/app_xxx \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "new-app-name"}' | jq .
# Expected: 200 OK
```

#### 3.3.2 Update Build Strategy
```bash
curl -s -X PUT http://localhost:9090/apps/app_xxx \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "build": {
      "strategy": "dockerfile",
      "dockerfile_path": "Dockerfile.prod"
    }
  }' | jq .
```

#### 3.3.3 Update Environment Variables
```bash
curl -s -X PUT http://localhost:9090/apps/app_xxx \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "environment": {
      "NODE_ENV": "production",
      "API_URL": "https://api.example.com"
    }
  }' | jq .
```

### 3.4 App Deletion Tests

#### 3.4.1 Delete App
```bash
curl -s -X DELETE http://localhost:9090/apps/app_xxx \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: 200 OK
```

#### 3.4.2 Delete App with Volumes
```bash
curl -s -X DELETE "http://localhost:9090/apps/app_xxx?delete_volumes=true" \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: 200 OK with deleted_resources
```

#### 3.4.3 Delete Non-existent App
```bash
curl -s -X DELETE http://localhost:9090/apps/app_nonexistent \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: 404 Not Found
```

### 3.5 App Deployment Tests

#### 3.5.1 Trigger Deployment
```bash
curl -s -X POST http://localhost:9090/apps/app_xxx/deploy \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: 202 Accepted with deployment_id
```

#### 3.5.2 Deploy Specific Branch
```bash
curl -s -X POST http://localhost:9090/apps/app_xxx/deploy \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"branch": "develop"}' | jq .
```

#### 3.5.3 Deploy Specific Commit
```bash
curl -s -X POST http://localhost:9090/apps/app_xxx/deploy \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"commit": "abc123def"}' | jq .
```

#### 3.5.4 List Deployments
```bash
curl -s -X GET http://localhost:9090/apps/app_xxx/deployments \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: List of deployments
```

#### 3.5.5 Get Deployment Details
```bash
curl -s -X GET http://localhost:9090/deployments/dep_xxx \
  -H "Authorization: Bearer $TOKEN" | jq .
```

#### 3.5.6 Get Deployment Logs
```bash
curl -s -X GET http://localhost:9090/deployments/dep_xxx/logs \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: Build log lines
```

#### 3.5.7 Rollback Deployment
```bash
curl -s -X POST http://localhost:9090/apps/app_xxx/rollback \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"deployment_id": "dep_yyy"}' | jq .
# Expected: 202 Accepted
```

### 3.6 App Logs Tests

#### 3.6.1 Get App Logs
```bash
curl -s -X GET http://localhost:9090/apps/app_xxx/logs \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: Log lines array
```

#### 3.6.2 Filter Logs by Stream
```bash
curl -s -X GET "http://localhost:9090/apps/app_xxx/logs?stream=stdout" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

#### 3.6.3 Get Last N Lines
```bash
curl -s -X GET "http://localhost:9090/apps/app_xxx/logs?tail=50" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

#### 3.6.4 Search Logs
```bash
curl -s -X GET "http://localhost:9090/apps/app_xxx/logs?search=error" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 3.7 App Environment Variables Tests

#### 3.7.1 Get Environment Variables
```bash
curl -s -X GET http://localhost:9090/apps/app_xxx/env \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: Variables with masked secrets
```

#### 3.7.2 Add Environment Variable
```bash
curl -s -X PUT http://localhost:9090/apps/app_xxx/env \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "variables": [
      {"key": "NEW_VAR", "value": "value", "is_secret": false}
    ]
  }' | jq .
```

#### 3.7.3 Add Secret Variable
```bash
curl -s -X PUT http://localhost:9090/apps/app_xxx/env \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "variables": [
      {"key": "API_KEY", "value": "secret123", "is_secret": true}
    ]
  }' | jq .
# Expected: Value masked in response
```

#### 3.7.4 Import from .env File
```bash
curl -s -X POST http://localhost:9090/apps/app_xxx/env/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "NODE_ENV=production\nPORT=3000\nAPI_KEY=secret"
  }' | jq .
```

### 3.8 App Volumes Tests

#### 3.8.1 List Volumes
```bash
curl -s -X GET http://localhost:9090/apps/app_xxx/volumes \
  -H "Authorization: Bearer $TOKEN" | jq .
```

#### 3.8.2 Add Volume
```bash
curl -s -X POST http://localhost:9090/apps/app_xxx/volumes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "uploads",
    "container_path": "/app/uploads"
  }' | jq .
```

#### 3.8.3 Delete Volume
```bash
curl -s -X DELETE http://localhost:9090/apps/app_xxx/volumes/vol_xxx \
  -H "Authorization: Bearer $TOKEN" | jq .
```

---

## Phase 4: Service Management

### 4.1 Service Creation Tests

#### 4.1.1 Create PostgreSQL Service
```bash
curl -s -X POST http://localhost:9090/services \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "main-db",
    "type": "postgresql",
    "version": "15"
  }' | jq .
# Expected: 201 Created
```

#### 4.1.2 Create MySQL Service
```bash
curl -s -X POST http://localhost:9090/services \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "mysql-db",
    "type": "mysql",
    "version": "8.0"
  }' | jq .
```

#### 4.1.3 Create Redis Service
```bash
curl -s -X POST http://localhost:9090/services \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "redis-cache",
    "type": "redis",
    "version": "7"
  }' | jq .
```

#### 4.1.4 Create with Custom Port
```bash
curl -s -X POST http://localhost:9090/services \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "custom-db",
    "type": "postgresql",
    "version": "15",
    "port": 5433
  }' | jq .
```

#### 4.1.5 Create with Custom Password
```bash
curl -s -X POST http://localhost:9090/services \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "secure-db",
    "type": "postgresql",
    "version": "15",
    "root_password": "mysecretpassword"
  }' | jq .
```

### 4.2 Service Listing Tests

#### 4.2.1 List All Services
```bash
curl -s -X GET http://localhost:9090/services \
  -H "Authorization: Bearer $TOKEN" | jq .
```

#### 4.2.2 Filter by Type
```bash
curl -s -X GET "http://localhost:9090/services?type=postgresql" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

#### 4.2.3 Filter by Status
```bash
curl -s -X GET "http://localhost:9090/services?status=running" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 4.3 Service Operations Tests

#### 4.3.1 Restart Service
```bash
curl -s -X POST http://localhost:9090/services/svc_xxx/restart \
  -H "Authorization: Bearer $TOKEN" | jq .
```

#### 4.3.2 Get Service Logs
```bash
curl -s -X GET http://localhost:9090/services/svc_xxx/logs \
  -H "Authorization: Bearer $TOKEN" | jq .
```

#### 4.3.3 Test Connection
```bash
curl -s -X POST http://localhost:9090/services/svc_xxx/test-connection \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: success: true/false
```

### 4.4 Service Deletion Tests

#### 4.4.1 Delete Service
```bash
curl -s -X DELETE http://localhost:9090/services/svc_xxx \
  -H "Authorization: Bearer $TOKEN" | jq .
```

#### 4.4.2 Force Delete Connected Service
```bash
curl -s -X DELETE "http://localhost:9090/services/svc_xxx?force=true" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

---

## Phase 5: Server Management

### 5.1 Server Info Tests
```bash
curl -s -X GET http://localhost:9090/server \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: hostname, os, kernel, architecture, panel_version, resources
```

### 5.2 Server Metrics Tests
```bash
# All metrics
curl -s -X GET http://localhost:9090/server/metrics \
  -H "Authorization: Bearer $TOKEN" | jq .

# Specific metric
curl -s -X GET "http://localhost:9090/server/metrics?metric=cpu" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Time range
curl -s -X GET "http://localhost:9090/server/metrics?range=24h" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 5.3 Process Management Tests
```bash
# List processes
curl -s -X GET http://localhost:9090/server/processes \
  -H "Authorization: Bearer $TOKEN" | jq .

# Sort by CPU
curl -s -X GET "http://localhost:9090/server/processes?sort=cpu" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Kill process
curl -s -X POST http://localhost:9090/server/processes/1234/kill \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"signal": "SIGTERM"}' | jq .
```

### 5.4 Disk Usage Tests
```bash
curl -s -X GET http://localhost:9090/server/disks \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: disk usage info
```

### 5.5 Network Info Tests
```bash
curl -s -X GET http://localhost:9090/server/network \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: interfaces, open ports, bandwidth
```

### 5.6 Firewall Tests

#### 5.6.1 Get Firewall Status
```bash
curl -s -X GET http://localhost:9090/firewall \
  -H "Authorization: Bearer $TOKEN" | jq .
```

#### 5.6.2 Add Firewall Rule
```bash
curl -s -X POST http://localhost:9090/firewall/rules \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "port": 8080,
    "protocol": "tcp",
    "source": "any",
    "action": "allow",
    "description": "Custom app port"
  }' | jq .
```

#### 5.6.3 Delete Firewall Rule
```bash
curl -s -X DELETE http://localhost:9090/firewall/rules/fw_xxx \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 5.7 Updates Tests
```bash
# Check for updates
curl -s -X GET http://localhost:9090/server/updates \
  -H "Authorization: Bearer $TOKEN" | jq .

# Apply updates
curl -s -X POST http://localhost:9090/server/updates \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type": "security"}' | jq .
```

---

## Phase 6: Real-time & WebSocket

### 6.1 WebSocket Connection Tests
```javascript
// Connect
const ws = new WebSocket('ws://localhost:9090/api/v1/stream');

// Authenticate
ws.send(JSON.stringify({
  type: 'auth',
  token: '<access_token>'
}));

// Subscribe
ws.send(JSON.stringify({
  type: 'subscribe',
  channels: ['server.metrics', 'deployments']
}));

// Keep alive
setInterval(() => ws.send(JSON.stringify({type: 'ping'})), 30000);
```

### 6.2 Event Types Tests
| Event | Trigger | Verify |
|-------|---------|--------|
| app.deploy.started | Deploy app | Event received |
| app.deploy.progress | Build progress | Progress updates |
| app.deploy.success | Build complete | Success event |
| app.deploy.failed | Build fails | Error event |
| app.logs | App outputs logs | Log stream |
| server.metrics | Periodic | Metrics updates |

---

## Phase 7: Security Testing

### 7.1 SQL Injection Tests
```bash
# Test login with SQL injection
curl -s -X POST http://localhost:9090/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin\" OR \"1\"=\"1", "password": "anything"}' | jq .
# Expected: 401 Unauthorized (not exposed)

# Test app name with SQL injection
curl -s -X POST http://localhost:9090/apps \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "app\"; DROP TABLE users; --", "runtime": "nodejs"}' | jq .
# Expected: 400 or validation error
```

### 7.2 XSS Tests
```bash
# Test app name with XSS
curl -s -X POST http://localhost:9090/apps \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "<script>alert(1)</script>", "runtime": "nodejs"}' | jq .
# Expected: Sanitized or rejected
```

### 7.3 Rate Limiting Tests
```bash
# Make many rapid requests
for i in {1..150}; do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:9090/health
done | sort | uniq -c
# Expected: Most return 200, some return 429
```

### 7.4 Token Security Tests
```bash
# Test token in URL (should be rejected or deemphasized)
curl -s "http://localhost:9090/apps?token=abc123" | jq .
# Expected: 401 (token should not be in query params)

# Test replay attack
curl -s -X POST http://localhost:9090/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "yourpassword"}' | jq .
# Use same refresh token twice - second should fail
```

### 7.5 Privilege Escalation Tests
```bash
# Viewer trying admin actions
curl -s -X DELETE http://localhost:9090/users/2 \
  -H "Authorization: Bearer $VIEWER_TOKEN" | jq .
# Expected: 403 Forbidden

# Developer trying to create users
curl -s -X POST http://localhost:9090/users/invite \
  -H "Authorization: Bearer $DEV_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "new@example.com", "role": "admin"}' | jq .
# Expected: 403 Forbidden
```

---

## Phase 8: Performance Testing

### 8.1 Load Testing
```bash
# Install autocannon
npm install -g autocannon

# Test API endpoint
autocannon -c 100 -d 10 http://localhost:9090/health

# Test app listing
autocannon -c 50 -d 10 -H "Authorization: Bearer $TOKEN" \
  http://localhost:9090/apps
```

### 8.2 Concurrent Requests
```bash
# Test concurrent logins
for i in {1..50}; do
  curl -s -X POST http://localhost:9090/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username": "admin", "password": "yourpassword"}'&
done
wait
# Expected: All succeed or proper rate limiting
```

### 8.3 Database Performance
```bash
# Time a query
time sqlite3 /var/panel/panel.db "SELECT * FROM apps;"

# Check for slow queries
sqlite3 /var/panel/panel.db "EXPLAIN QUERY PLAN SELECT * FROM apps WHERE status = 'running';"
```

---

## Phase 9: Integration Testing

### 9.1 Git Repository Integration
```bash
# Test GitHub webhook (simulate)
curl -s -X POST http://localhost:9090/webhooks/github \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -d '{
    "ref": "refs/heads/main",
    "repository": {"clone_url": "https://github.com/user/repo"},
    "commits": [{"id": "abc123", "message": "Test commit"}]
  }' | jq .
```

### 9.2 SSL Certificate Tests
```bash
# Check SSL certificates
ls -la /etc/panel/ssl/

# Check certificate expiry
openssl x509 -in /etc/panel/ssl/cert.pem -noout -dates
```

### 9.3 Backup Integration Tests
```bash
# Create manual backup
curl -s -X POST http://localhost:9090/backups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "target_type": "service",
    "target_id": "svc_xxx",
    "destination": "local"
  }' | jq .

# List backups
curl -s -X GET http://localhost:9090/backups \
  -H "Authorization: Bearer $TOKEN" | jq .
```

---

## Phase 10: Disaster Recovery

### 10.1 Database Backup Tests
```bash
# Create backup
cp /var/panel/panel.db /var/panel/panel.db.test-backup

# Verify backup
sqlite3 /var/panel/panel.db.test-backup "SELECT count(*) FROM apps;"
```

### 10.2 Database Restore Tests
```bash
# Simulate corruption
echo "DELETE FROM apps;" | sqlite3 /var/panel/panel.db

# Restore from backup
cp /var/panel/panel.db.test-backup /var/panel/panel.db

# Verify restored
sqlite3 /var/panel/panel.db "SELECT count(*) FROM apps;"
```

### 10.3 Service Restart Tests
```bash
# Restart API
sudo systemctl restart juvia-api

# Verify comes back up
sleep 5
curl -s http://localhost:9090/health
# Expected: {"status":"ok"}
```

### 10.4 Container Recovery Tests
```bash
# Stop a container manually
docker stop app_container

# Check panel detects failure
curl -s -X GET http://localhost:9090/apps/app_xxx \
  -H "Authorization: Bearer $TOKEN" | jq '.status'
# Expected: "failed" or "stopped"

# Restart via panel
curl -s -X POST http://localhost:9090/apps/app_xxx/start \
  -H "Authorization: Bearer $TOKEN" | jq .

# Verify container running
docker ps | grep app_xxx
```

---

## Test Execution Checklist

### Pre-Test Setup
- [ ] Reset test database
- [ ] Create test users (owner, admin, developer, viewer)
- [ ] Verify all services running
- [ ] Obtain valid tokens for each role

### Phase Execution
- [ ] Phase 1: Infrastructure - All tests pass
- [ ] Phase 2: Authentication - All tests pass
- [ ] Phase 3: Applications - All tests pass
- [ ] Phase 4: Services - All tests pass
- [ ] Phase 5: Server - All tests pass
- [ ] Phase 6: WebSocket - All tests pass
- [ ] Phase 7: Security - All tests pass
- [ ] Phase 8: Performance - Meets benchmarks
- [ ] Phase 9: Integration - All integrations work
- [ ] Phase 10: Recovery - Backup/restore works

### Post-Test
- [ ] Document any failures
- [ ] Create issue tickets for bugs
- [ ] Clean up test data

---

## Success Criteria

| Category | Pass Rate | Critical Tests |
|----------|----------|---------------|
| Infrastructure | 100% | Database, API, Agent |
| Authentication | 100% | Login, Token refresh, RBAC |
| Applications | 100% | CRUD, Deploy, Logs |
| Services | 100% | CRUD, Connection |
| Server | 100% | Metrics, Firewall |
| WebSocket | 100% | Connection, Events |
| Security | 100% | No injection, proper auth |
| Performance | >80% | Load tests complete |
| Integration | >80% | Git, SSL, Backup |
| Recovery | 100% | Backup/Restore works |

---

*End of Comprehensive Testing Plan*