# Juvia Panel — Install / Uninstall / Update Scripts Specification
## Lifecycle Management for Single-Server PaaS

**Version:** 1.0  
**Date:** 2026-06-01  
**Target OS:** Ubuntu 24.04 LTS (primary), Debian 12 (secondary)  
**Architecture:** amd64 (primary), arm64 (future)

---

## Table of Contents

1. [Design Principles](#1-design-principles)
2. [Installation Script](#2-installation-script)
3. [Uninstallation Script](#3-uninstallation-script)
4. [Update Script](#4-update-script)
5. [Version Management](#5-version-management)
6. [Rollback Strategy](#6-rollback-strategy)
7. [Error Handling & Recovery](#7-error-handling--recovery)
8. [Security Considerations](#8-security-considerations)
9. [Testing Matrix](#9-testing-matrix)

---

## 1. Design Principles

| Principle | Implementation |
|-----------|---------------|
| **One-command install** | `curl -fsSL https://get.panel.dev | bash` |
| **Zero config to start** | Sensible defaults, wizard for customization |
| **Atomic operations** | Each step is reversible until commit point |
| **Manifest tracking** | Every file, user, service created is recorded |
| **Clean uninstall** | Reads manifest, removes everything it created |
| **Safe update** | Backup current version, replace, verify, rollback on failure |
| **No lock-in** | Export apps before uninstall, standard Docker containers |

---

## 2. Installation Script

### 2.1 User Experience

```bash
# One-line install (production)
curl -fsSL https://get.panel.dev | bash

# With options (advanced)
curl -fsSL https://get.panel.dev | bash -s --   --version 1.2.3   --data-dir /opt/panel   --domain panel.example.com   --email admin@example.com   --skip-docker   --skip-caddy
```

### 2.2 Installation Flow

```
STEP 1: System Check
  - OS version (Ubuntu 24.04 / Debian 12)
  - Architecture (amd64)
  - Kernel version (6.8+)
  - Root or sudo access
  - Available disk space (min 20GB)
  - Available RAM (min 2GB)
  - Internet connectivity

STEP 2: Dependency Installation
  - Docker Engine 24.0+
  - Docker Compose v2 plugin
  - Caddy 2.7+
  - Git
  - jq
  - curl
  - sqlite3 (for CLI inspection)

STEP 3: User & Directory Setup
  - Create system user: `panel` (UID auto-assigned)
  - Add `panel` to `docker` group
  - Create directories:
    - /var/panel (data, 750)
    - /etc/panel (config, 750)
    - /opt/panel (binaries, 755)
    - /var/run/panel (runtime, 755)
  - Set ownership: panel:panel

STEP 4: Binary Installation
  - Download panel-agent (Go binary, ~20MB)
  - Download panel-api (Go binary, ~25MB)
  - Download panel-cli (Go binary, ~10MB)
  - Download panel-ui (static files, ~5MB)
  - Verify checksums (SHA-256) against manifest
  - Install to /usr/local/bin/ (agent, api, cli)
  - Install UI to /opt/panel/ui/

STEP 5: Configuration Generation
  - Generate /etc/panel/config.yml
  - Generate master encryption key (/etc/panel/keys/master)
  - Generate JWT secret
  - Set permissions: 600 (keys), 644 (config)
  - Create initial Caddyfile

STEP 6: Database Initialization
  - Create /var/panel/panel.db
  - Run all migrations (000001_init.up.sql)
  - Insert default server_info row
  - Enable WAL mode, foreign keys

STEP 7: systemd Service Setup
  - Create /etc/systemd/system/panel-agent.service
  - Create /etc/systemd/system/panel-api.service
  - Create /etc/systemd/system/panel-caddy.service
  - Reload systemd daemon
  - Enable services (start on boot)

STEP 8: Firewall Configuration
  - UFW: allow 22 (SSH)
  - UFW: allow 80 (HTTP)
  - UFW: allow 443 (HTTPS)
  - UFW: deny all other incoming
  - UFW: enable (if not already)

STEP 9: Service Start
  - Start panel-agent
  - Start panel-api
  - Start Caddy (with panel config)
  - Wait for health checks (max 60s)
  - Verify: curl http://localhost:2053/health

STEP 10: First-Run Wizard
  - Print panel URL (https://server-ip or custom domain)
  - Open browser (if desktop environment detected)
  - User completes: admin account, server name, timezone
  - Optional: domain setup, SSL, S3 backup config

STEP 11: Manifest Creation
  - Write /var/panel/.panel-manifest.json
  - Records every file, user, service, rule created
  - Used by uninstall script for clean removal
```

### 2.3 Installation Manifest

`/var/panel/.panel-manifest.json`:
```json
{
  "version": "1.0.0",
  "installed_at": "2024-06-01T12:34:56Z",
  "installer_version": "1.0.0",
  "system": {
    "os": "Ubuntu 24.04 LTS",
    "kernel": "6.8.0-31-generic",
    "architecture": "amd64"
  },
  "users_created": [
    {
      "username": "panel",
      "uid": 998,
      "home": "/var/panel",
      "groups": ["panel", "docker"]
    }
  ],
  "directories_created": [
    "/var/panel",
    "/etc/panel",
    "/opt/panel",
    "/var/run/panel"
  ],
  "files_installed": [
    {
      "path": "/usr/local/bin/panel-agent",
      "checksum": "sha256:abc123...",
      "size": 20485760
    },
    {
      "path": "/usr/local/bin/panel-api",
      "checksum": "sha256:def456...",
      "size": 25600000
    },
    {
      "path": "/opt/panel/ui/index.html",
      "checksum": "sha256:ghi789...",
      "size": 5120
    }
  ],
  "services_created": [
    "panel-agent.service",
    "panel-api.service",
    "panel-caddy.service"
  ],
  "firewall_rules_added": [
    {"port": 80, "protocol": "tcp", "action": "allow"},
    {"port": 443, "protocol": "tcp", "action": "allow"}
  ],
  "packages_installed": [
    "docker-ce",
    "docker-compose-plugin",
    "caddy",
    "git",
    "jq"
  ]
}
```

### 2.4 Installation Script Code Structure

```bash
#!/bin/bash
set -euo pipefail

# Colors for output
RED='[0;31m'
GREEN='[0;32m'
YELLOW='[1;33m'
NC='[0m'

# Configuration (override with flags)
VERSION="${PANEL_VERSION:-latest}"
DATA_DIR="${PANEL_DATA_DIR:-/var/panel}"
CONFIG_DIR="${PANEL_CONFIG_DIR:-/etc/panel}"
INSTALL_DIR="${PANEL_INSTALL_DIR:-/opt/panel}"
DOMAIN="${PANEL_DOMAIN:-}"
EMAIL="${PANEL_EMAIL:-}"
SKIP_DOCKER=false
SKIP_CADDY=false

# Logging
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Step tracking
CURRENT_STEP=0
TOTAL_STEPS=11

step() {
  CURRENT_STEP=$((CURRENT_STEP + 1))
  log_info "Step $CURRENT_STEP/$TOTAL_STEPS: $1"
}

# Error handler
cleanup_on_error() {
  local exit_code=$?
  log_error "Installation failed at step $CURRENT_STEP with exit code $exit_code"
  log_info "Check /var/log/panel-install.log for details"
  log_info "To clean up partial installation, run: panel uninstall --purge"
  exit $exit_code
}
trap cleanup_on_error ERR

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --version) VERSION="$2"; shift 2 ;;
    --data-dir) DATA_DIR="$2"; shift 2 ;;
    --domain) DOMAIN="$2"; shift 2 ;;
    --email) EMAIL="$2"; shift 2 ;;
    --skip-docker) SKIP_DOCKER=true; shift ;;
    --skip-caddy) SKIP_CADDY=true; shift ;;
    *) log_error "Unknown option: $1"; exit 1 ;;
  esac
done

# Step 1: System Check
step "Checking system requirements"
# Verify OS, arch, kernel, disk, RAM, root access

# Step 2: Dependencies
step "Installing dependencies"
# Install Docker, Caddy, Git, jq

# Step 3: User & Directories
step "Creating panel user and directories"
# useradd panel, mkdir dirs, chown

# Step 4: Binaries
step "Downloading panel binaries"
# curl downloads, checksum verify

# Step 5: Config
step "Generating configuration"
# config.yml, keys, JWT secret, Caddyfile

# Step 6: Database
step "Initializing database"
# sqlite3, migrations, WAL mode

# Step 7: systemd
step "Setting up systemd services"
# service files, daemon-reload, enable

# Step 8: Firewall
step "Configuring firewall"
# ufw allow 22,80,443

# Step 9: Start Services
step "Starting panel services"
# systemctl start, health check

# Step 10: First-Run
step "Completing setup"
# Print URL, open browser

# Step 11: Manifest
step "Creating installation manifest"
# Write .panel-manifest.json

log_info "Installation complete!"
log_info "Panel URL: https://${DOMAIN:-$(curl -s ifconfig.me)}"
log_info "Complete setup at the URL above"
```

---

## 3. Uninstallation Script

### 3.1 User Experience

```bash
# Interactive uninstall (default)
panel uninstall

# Export apps and stop panel
panel uninstall --export-only

# Full purge (delete everything)
panel uninstall --purge

# Purge but keep volume data
panel uninstall --purge --keep-data
```

### 3.2 Uninstallation Flow

```
STEP 1: Read Manifest
  - Load /var/panel/.panel-manifest.json
  - If missing, scan for panel-managed resources

STEP 2: Export Apps (unless --purge without --keep-data)
  - For each app:
    - Generate docker-compose.yml
    - Export .env file
    - Copy volume data (optional: --with-data)
  - Save to /var/panel/export/{timestamp}/
  - Print restart instructions

STEP 3: Stop Services
  - systemctl stop panel-agent
  - systemctl stop panel-api
  - systemctl stop panel-caddy
  - Wait 10 seconds for graceful shutdown

STEP 4: Remove Containers
  - docker stop $(panel-managed containers)
  - docker rm $(panel-managed containers)
  - docker rmi $(panel-managed images)
  - docker network rm panel-networks
  - docker volume rm panel-volumes (unless --keep-data)

STEP 5: Remove systemd Services
  - systemctl disable panel-agent
  - systemctl disable panel-api
  - systemctl disable panel-caddy
  - rm /etc/systemd/system/panel-*.service
  - systemctl daemon-reload

STEP 6: Remove Files
  - rm /usr/local/bin/panel-agent
  - rm /usr/local/bin/panel-api
  - rm /usr/local/bin/panel-cli
  - rm -rf /opt/panel/ui
  - rm -rf /etc/panel (config, keys, Caddyfile)
  - rm -rf /var/run/panel
  - Keep /var/panel if --export-only
  - Remove /var/panel if --purge

STEP 7: Remove User (optional)
  - userdel panel (if no other services use it)
  - Keep user if --keep-user flag

STEP 8: Cleanup Firewall
  - Remove panel-added UFW rules
  - Do NOT disable UFW entirely (other apps may need it)

STEP 9: Verify Cleanup
  - Check no panel processes running: ps aux | grep panel
  - Check no panel ports listening: ss -tlnp | grep panel
  - Print summary of what was removed
  - Print export directory if apps were exported
```

### 3.3 Export Format

Each exported app gets a directory:

```
/var/panel/export/20240601-123456/
├── api-prod/
│   ├── docker-compose.yml
│   ├── .env
│   └── volumes/
│       ├── data/
│       └── uploads/
├── web-client/
│   ├── docker-compose.yml
│   ├── .env
│   └── volumes/
│       └── dist/
└── main-pg/
    ├── docker-compose.yml
    ├── .env
    └── volumes/
        └── data/
```

**docker-compose.yml example:**
```yaml
version: "3.8"
services:
  api-prod:
    image: panel-app-api-prod:latest
    container_name: api-prod
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      NODE_ENV: production
      PORT: "3000"
      DATABASE_URL: postgres://user:pass@main-pg:5432/db
    volumes:
      - ./volumes/data:/app/data
      - ./volumes/uploads:/app/uploads
    networks:
      - panel-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/health"]
      interval: 30s
      timeout: 5s
      retries: 3

networks:
  panel-network:
    external: true
```

### 3.4 Uninstall Script Code Structure

```bash
#!/bin/bash
set -euo pipefail

# Parse arguments
EXPORT_ONLY=false
PURGE=false
KEEP_DATA=false
KEEP_USER=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --export-only) EXPORT_ONLY=true; shift ;;
    --purge) PURGE=true; shift ;;
    --keep-data) KEEP_DATA=true; shift ;;
    --keep-user) KEEP_USER=true; shift ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# Load manifest
MANIFEST="/var/panel/.panel-manifest.json"
if [[ -f "$MANIFEST" ]]; then
  MANIFEST_DATA=$(cat "$MANIFEST")
else
  echo "Warning: Manifest not found. Scanning for panel resources..."
fi

# Step 1: Export apps
if [[ "$EXPORT_ONLY" == true ]] || [[ "$PURGE" == false ]]; then
  echo "Exporting apps..."
  EXPORT_DIR="/var/panel/export/$(date +%Y%m%d-%H%M%S)"
  mkdir -p "$EXPORT_DIR"

  # For each app in database
  sqlite3 /var/panel/panel.db "SELECT id, name FROM apps;" |   while IFS='|' read -r id name; do
    APP_DIR="$EXPORT_DIR/$name"
    mkdir -p "$APP_DIR/volumes"

    # Generate docker-compose.yml
    # Generate .env
    # Copy volumes (if not --keep-data)

    echo "Exported: $name"
  done

  echo "Apps exported to: $EXPORT_DIR"
  echo "To restart: cd $EXPORT_DIR/<app> && docker compose up -d"
fi

# If export-only, stop here
if [[ "$EXPORT_ONLY" == true ]]; then
  echo "Panel stopped. Exported apps remain runnable."
  exit 0
fi

# Step 2-9: Full removal
# Stop services, remove containers, remove files, cleanup

echo "Panel uninstalled successfully."
if [[ "$PURGE" == true ]]; then
  echo "All data removed."
else
  echo "App data preserved in /var/panel/"
fi
```

---

## 4. Update Script

### 4.1 User Experience

```bash
# Check for updates
panel update check

# Update to latest
panel update

# Update to specific version
panel update --version 1.2.3

# Update with automatic rollback on failure
panel update --auto-rollback

# View update history
panel update history
```

### 4.2 Update Flow

```
STEP 1: Check Current Version
  - Read /var/panel/.panel-manifest.json
  - Query update server: https://api.panel.dev/versions
  - Compare current vs latest
  - Show changelog

STEP 2: Pre-Update Backup
  - Backup database: cp panel.db panel.db.backup
  - Backup config: tar czf config-backup.tar.gz /etc/panel
  - Export apps (optional but recommended)
  - Store backups in /var/panel/backups/update/

STEP 3: Download New Version
  - Download new binaries to /opt/panel/.update/
  - Verify checksums
  - Download new UI static files
  - Download new migrations

STEP 4: Database Migration
  - Run pending migrations (golang-migrate)
  - If migration fails -> rollback to backup, exit

STEP 5: Replace Binaries (Atomic)
  - Stop services gracefully
  - mv current -> current.old
  - mv new -> current
  - If any step fails -> restore .old, restart old version

STEP 6: Start New Version
  - Start services
  - Wait for health checks (max 60s)
  - Verify API responds correctly
  - Verify UI loads

STEP 7: Post-Update Verification
  - Run smoke tests (built-in test suite)
  - Verify critical paths: login, app list, deploy
  - If tests fail -> rollback

STEP 8: Cleanup
  - Remove .old binaries (keep for 7 days)
  - Update manifest version
  - Log update event
  - Notify admin (if configured)
```

### 4.3 Rollback on Failure

If any step fails after database migration:

```bash
rollback_update() {
  log_error "Update failed. Rolling back..."

  # Stop new services
  systemctl stop panel-agent panel-api

  # Restore old binaries
  mv /usr/local/bin/panel-agent.old /usr/local/bin/panel-agent
  mv /usr/local/bin/panel-api.old /usr/local/bin/panel-api
  mv /opt/panel/ui.old /opt/panel/ui

  # Restore database (if migration was applied)
  cp /var/panel/panel.db.backup /var/panel/panel.db

  # Restore config
  tar xzf /var/panel/backups/update/config-backup.tar.gz -C /

  # Start old services
  systemctl start panel-agent panel-api

  # Verify old version works
  sleep 5
  curl -f http://localhost:8080/health || {
    log_error "Rollback failed! Manual intervention required."
    exit 1
  }

  log_info "Rollback successful. Running version: $(panel --version)"
}
```

### 4.4 Update Script Code Structure

```bash
#!/bin/bash
set -euo pipefail

VERSION="latest"
AUTO_ROLLBACK=true

while [[ $# -gt 0 ]]; do
  case $1 in
    check) MODE="check"; shift ;;
    --version) VERSION="$2"; shift 2 ;;
    --no-rollback) AUTO_ROLLBACK=false; shift ;;
    history) MODE="history"; shift ;;
    *) echo "Usage: panel update [check|--version X|--no-rollback]"; exit 1 ;;
  esac
done

# Check mode
if [[ "${MODE:-}" == "check" ]]; then
  CURRENT=$(panel --version)
  LATEST=$(curl -s https://api.panel.dev/versions/latest | jq -r '.version')

  if [[ "$CURRENT" == "$LATEST" ]]; then
    echo "Panel is up to date ($CURRENT)"
  else
    echo "Update available: $CURRENT -> $LATEST"
    curl -s https://api.panel.dev/versions/$LATEST/changelog
  fi
  exit 0
fi

# History mode
if [[ "${MODE:-}" == "history" ]]; then
  sqlite3 /var/panel/panel.db     "SELECT * FROM update_history ORDER BY created_at DESC;"
  exit 0
fi

# Update mode
CURRENT=$(panel --version)
echo "Updating Panel: $CURRENT -> $VERSION"

# Step 1: Backup
# Step 2: Download
# Step 3: Migrate
# Step 4: Replace
# Step 5: Start
# Step 6: Verify
# Step 7: Cleanup

echo "Update complete!"
```

---

## 5. Version Management

### 5.1 Version Format
Semantic versioning: MAJOR.MINOR.PATCH

| Component | When Incremented |
|-----------|-----------------|
| MAJOR | Breaking changes (database schema, API changes) |
| MINOR | New features, backwards compatible |
| PATCH | Bug fixes, security patches |

### 5.2 Version Storage
- Binary: compiled into binary at build time (`-ldflags "-X main.version=1.2.3"`)
- Database: `server_info.panel_version`
- Manifest: `.panel-manifest.json.version`
- API: `GET /server` returns version

### 5.3 Version Compatibility
| Panel Version | Database Schema | Supported OS |
|--------------|-----------------|--------------|
| 1.x | 1.x | Ubuntu 24.04, Debian 12 |
| 2.x (future) | 2.x | Ubuntu 24.04+, Debian 12+ |

---

## 6. Rollback Strategy

### 6.1 Automatic Rollback Triggers
- Health check fails after update
- Database migration fails
- Binary checksum mismatch
- Smoke tests fail

### 6.2 Rollback Window
- Old binaries kept for 7 days (`/usr/local/bin/panel-agent.old`)
- Database backup kept for 7 days (`panel.db.backup`)
- Config backup kept for 7 days
- After 7 days, old versions auto-deleted by cleanup job

### 6.3 Manual Rollback
```bash
panel update rollback --to 1.2.1
```

---

## 7. Error Handling & Recovery

### 7.1 Common Failure Points
| Step | Failure | Recovery |
|------|---------|----------|
| System check | Wrong OS | Exit with clear error, link to supported OS |
| Docker install | Network timeout | Retry 3x, then exit with manual install instructions |
| Binary download | Checksum mismatch | Delete, retry from mirror, then exit |
| Service start | Port conflict | Detect conflict, suggest free port, retry |
| Database init | Permission denied | Fix permissions, retry |
| Migration | Schema conflict | Restore backup, exit with error |

### 7.2 Recovery Commands
```bash
# Check install status
panel doctor

# Fix permissions
panel repair permissions

# Reinstall without data loss
panel reinstall --keep-data

# View install logs
cat /var/log/panel-install.log

# Debug mode (verbose output)
panel install --debug
```

---

## 8. Security Considerations

### 8.1 Installation
- Download binaries over HTTPS only
- Verify SHA-256 checksums against signed manifest
- GPG signature verification (future)
- Keys generated with 600 permissions
- No secrets in environment variables (use files)

### 8.2 Uninstallation
- Manifest prevents accidental deletion of user files
- Export encrypts sensitive env vars
- Database backup encrypted before removal

### 8.3 Update
- Binaries downloaded to temp location first
- Atomic replacement (mv, not cp)
- Rollback on any failure
- No downgrades without explicit flag (security)

---

## 9. Testing Matrix

| Scenario | Ubuntu 24.04 | Debian 12 | Fresh VM | Existing Docker | Existing Caddy |
|----------|-------------|-----------|----------|-----------------|----------------|
| Fresh install | Yes | Yes | Yes | Yes | Yes |
| Reinstall (keep data) | Yes | - | - | Yes | Yes |
| Uninstall (export) | Yes | - | Yes | - | - |
| Uninstall (purge) | Yes | - | Yes | - | - |
| Update (patch) | Yes | - | Yes | Yes | Yes |
| Update (minor) | Yes | - | Yes | Yes | Yes |
| Update (major) | Yes | - | - | - | - |
| Rollback | Yes | - | - | - | - |

---

*End of Install/Uninstall/Update Specification*
