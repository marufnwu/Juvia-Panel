#!/bin/bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

VERSION="latest"
AUTO_ROLLBACK=true
MODE=""
DEBUG=false

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

DATA_DIR="${PANEL_DATA_DIR:-/var/panel}"
CONFIG_DIR="${PANEL_CONFIG_DIR:-/etc/panel}"
INSTALL_DIR="${PANEL_INSTALL_DIR:-/opt/panel}"
REPO_URL="${PANEL_REPO_URL:-https://github.com/marufnwu/Juvia-Panel.git}"
REPO_BRANCH="${PANEL_REPO_BRANCH:-master}"

CURRENT_VERSION_FILE="$DATA_DIR/.version"

while [[ $# -gt 0 ]]; do
    case $1 in
        check) MODE="check"; shift ;;
        --version) VERSION="$2"; shift 2 ;;
        --no-rollback) AUTO_ROLLBACK=false; shift ;;
        --auto-rollback) AUTO_ROLLBACK=true; shift ;;
        history) MODE="history"; shift ;;
        rollback) MODE="rollback"; shift ;;
        --debug) DEBUG=true; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

[[ "$DEBUG" == "true" ]] && set -x

if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

rollback_update() {
    log_error "Update failed. Rolling back..."

    if [[ -f "/usr/local/bin/juvia-api.old" ]]; then
        mv /usr/local/bin/juvia-api.old /usr/local/bin/juvia-api
        log_info "Restored previous juvia-api binary"
    fi

    if [[ -f "/usr/local/bin/juvia-agent.old" ]]; then
        mv /usr/local/bin/juvia-agent.old /usr/local/bin/juvia-agent
        log_info "Restored previous juvia-agent binary"
    fi

    if [[ -f "$DATA_DIR/panel.db.backup" ]]; then
        cp "$DATA_DIR/panel.db.backup" "$DATA_DIR/panel.db"
        log_info "Restored database backup"
    fi

    if [[ -f "$DATA_DIR/config-backup.tar.gz" ]]; then
        BACKUP_SIZE=$(stat -c%s "$DATA_DIR/config-backup.tar.gz" 2>/dev/null || echo 0)
        if [[ "$BACKUP_SIZE" -gt 1024 ]]; then
            tar xzf "$DATA_DIR/config-backup.tar.gz" -C / 2>/dev/null || true
            log_info "Restored configuration backup"
        else
            log_warn "Config backup looks too small (${BACKUP_SIZE} bytes), skipping restore"
        fi
    fi

    # Restore UI if backup exists
    if [[ -d "$INSTALL_DIR/ui.old" ]]; then
        rm -rf "$INSTALL_DIR/ui"
        mv "$INSTALL_DIR/ui.old" "$INSTALL_DIR/ui"
        log_info "Restored previous UI"
    fi

    systemctl restart juvia-agent 2>/dev/null || true
    systemctl restart juvia-api
    if systemctl is-enabled juvia-caddy &>/dev/null; then
        systemctl restart juvia-caddy 2>/dev/null || true
    fi

    sleep 5
    if curl -sf --max-time 10 http://localhost:9090/health > /dev/null 2>&1; then
        log_info "Rollback successful"
    else
        log_error "Rollback failed! Manual intervention required."
        log_error "Please check: systemctl status juvia-api juvia-agent"
        exit 1
    fi
}

get_current_version() {
    if [[ -f "$CURRENT_VERSION_FILE" ]]; then
        cat "$CURRENT_VERSION_FILE"
    elif [[ -f "/usr/local/bin/juvia-api" ]]; then
        /usr/local/bin/juvia-api --version 2>/dev/null || echo "unknown"
    else
        echo "not-installed"
    fi
}

if [[ "${MODE:-}" == "check" ]]; then
    CURRENT=$(get_current_version)
    log_info "Current version: $CURRENT"

    # Clone to temporary directory to check version
    TEMP_CLONE="/tmp/juvia-update-check"
    rm -rf "$TEMP_CLONE"
    
    log_info "Checking for updates from $REPO_URL..."
    if git clone --depth 1 --branch "$REPO_BRANCH" "$REPO_URL" "$TEMP_CLONE" 2>/dev/null; then
        LATEST=$(git -C "$TEMP_CLONE" describe --tags 2>/dev/null || git -C "$TEMP_CLONE" rev-parse --short HEAD 2>/dev/null || echo "unknown")
        rm -rf "$TEMP_CLONE"
        
        if [[ "$CURRENT" == "$LATEST" ]]; then
            log_info "Juvia Panel is up to date ($CURRENT)"
        else
            log_info "Update available: $CURRENT -> $LATEST"
        fi
    else
        log_warn "Could not check for updates (git clone failed)"
        log_info "Current version: $CURRENT"
    fi
    exit 0
fi

if [[ "${MODE:-}" == "history" ]]; then
    if [[ -f "$DATA_DIR/panel.db" ]]; then
        log_info "Update history from database:"
        sqlite3 "$DATA_DIR/panel.db" "SELECT id, from_version, to_version, status, created_at, details FROM update_history ORDER BY created_at DESC LIMIT 20;" 2>/dev/null || echo "No update history found"
    else
        log_error "Database not found at $DATA_DIR/panel.db"
    fi
    exit 0
fi

if [[ "${MODE:-}" == "rollback" ]]; then
    log_info "Initiating rollback..."
    rollback_update
    exit 0
fi

CURRENT=$(get_current_version)
TARGET_VERSION="${VERSION:-latest}"

log_step "Updating Juvia Panel: $CURRENT -> $TARGET_VERSION"

log_step "Step 1: Pre-update backup"
mkdir -p "$DATA_DIR/backups/update"

if [[ -f "$DATA_DIR/panel.db" ]]; then
    cp "$DATA_DIR/panel.db" "$DATA_DIR/panel.db.backup"
    log_info "Database backed up to $DATA_DIR/panel.db.backup"
fi

if [[ -d "$CONFIG_DIR" ]]; then
    tar czf "$DATA_DIR/config-backup.tar.gz" -C / etc/panel 2>/dev/null || true
    log_info "Configuration backed up to $DATA_DIR/config-backup.tar.gz"
fi

if [[ -f "/usr/local/bin/juvia-agent" ]]; then
    cp /usr/local/bin/juvia-agent "$DATA_DIR/backups/update/juvia-agent.old" 2>/dev/null || true
fi
if [[ -f "/usr/local/bin/juvia-api" ]]; then
    cp /usr/local/bin/juvia-api "$DATA_DIR/backups/update/juvia-api.old" 2>/dev/null || true
fi

log_info "Pre-update backup complete"

log_step "Step 2: Downloading update"
DOWNLOAD_DIR="/tmp/juvia-panel-update-download"
rm -rf "$DOWNLOAD_DIR"
mkdir -p "$DOWNLOAD_DIR"

ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH_SUFFIX="amd64" ;;
    aarch64|arm64) ARCH_SUFFIX="arm64" ;;
    *) log_error "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Always build from source to ensure latest code is used
BUILD_FROM_SOURCE=true
RELEASE_TAG=""

if [[ "$BUILD_FROM_SOURCE" == "true" ]]; then
    TEMP_CLONE="/tmp/juvia-panel-update"
    rm -rf "$TEMP_CLONE"

    log_info "Cloning $REPO_URL (branch: $REPO_BRANCH)..."
    if ! git clone --depth 1 --branch "$REPO_BRANCH" "$REPO_URL" "$TEMP_CLONE" 2>/dev/null; then
        log_error "Failed to clone repository"
        exit 1
    fi

    NEW_VERSION=$(git -C "$TEMP_CLONE" describe --tags 2>/dev/null || git -C "$TEMP_CLONE" rev-parse --short HEAD 2>/dev/null || echo "latest")

    log_info "Building Go binaries..."
    cd "$TEMP_CLONE/backend"
    mkdir -p "$DOWNLOAD_DIR/extracted"

    CGO_ENABLED=0 go build -ldflags="-s -w" -o "$DOWNLOAD_DIR/extracted/juvia-api" ./cmd/api/
    CGO_ENABLED=0 go build -ldflags="-s -w" -o "$DOWNLOAD_DIR/extracted/juvia-agent" ./cmd/agent/
    CGO_ENABLED=0 go build -ldflags="-s -w" -o "$DOWNLOAD_DIR/extracted/juvia-cli" ./cmd/debug/

    log_info "Building UI..."
    cd "$TEMP_CLONE/frontend"
    npm ci --legacy-peer-deps --silent 2>/dev/null || npm install --legacy-peer-deps 2>/dev/null || true
    npm run build 2>/dev/null || log_warn "UI build failed, preserving existing UI"

    if [[ -d "$TEMP_CLONE/frontend/out" ]]; then
        cd "$TEMP_CLONE/frontend"
        tar -czf "$DOWNLOAD_DIR/extracted/juvia-ui.tar.gz" out/
    else
        log_warn "UI build output not found, preserving existing UI"
    fi

    # Copy migrations
    mkdir -p "$DOWNLOAD_DIR/extracted/migrations"
    cp "$TEMP_CLONE/backend/migrations/"*.sql "$DOWNLOAD_DIR/extracted/migrations/" 2>/dev/null || true

    # Copy Caddyfile to download directory for deployment
    mkdir -p "$DOWNLOAD_DIR/extracted/config"
    cp "$TEMP_CLONE/backend/config/Caddyfile" "$DOWNLOAD_DIR/extracted/config/Caddyfile" 2>/dev/null || true

    rm -rf "$TEMP_CLONE"
    NEW_VERSION="${NEW_VERSION:-$RELEASE_TAG}"
else
    NEW_VERSION="$RELEASE_TAG"
fi

log_info "Download complete (version: $NEW_VERSION)"

log_step "Step 3: Stopping services"
systemctl stop juvia-agent 2>/dev/null || true
systemctl stop juvia-api 2>/dev/null || true
systemctl stop juvia-caddy 2>/dev/null || true
log_info "Services stopped"

log_step "Step 3b: Deploying migrations"
DB_PATH="$DATA_DIR/panel.db"

# Copy any new migrations from the download - the API process applies them on next start
if [[ -d "$DOWNLOAD_DIR/extracted/migrations" ]]; then
    mkdir -p "$CONFIG_DIR/migrations"
    cp "$DOWNLOAD_DIR/extracted/migrations/"*.sql "$CONFIG_DIR/migrations/" 2>/dev/null || true
    chown -R juvia:juvia "$CONFIG_DIR/migrations" 2>/dev/null || true
    log_info "Migrations deployed (will be applied by API on next start)"
fi

log_step "Step 4: Applying update (atomic replace)"

for binary in juvia-api juvia-agent juvia-cli; do
    if [[ -f "$DOWNLOAD_DIR/extracted/$binary" ]]; then
        mv "/usr/local/bin/$binary" "/usr/local/bin/${binary}.old" 2>/dev/null || true
        mv "$DOWNLOAD_DIR/extracted/$binary" "/usr/local/bin/$binary"
        chmod +x "/usr/local/bin/$binary"
        log_info "Updated $binary"
    fi
done

if [[ -f "$DOWNLOAD_DIR/extracted/juvia-ui.tar.gz" ]]; then
    # Verify the tarball contains the expected static export structure
    if ! tar tzf "$DOWNLOAD_DIR/extracted/juvia-ui.tar.gz" | grep -q "^out/index.html$"; then
        log_error "UI tarball appears to be invalid (no out/index.html found)"
        log_error "Refusing to update UI with corrupted package"
        exit 1
    fi

    rm -rf "$INSTALL_DIR/ui.old"
    if [[ -d "$INSTALL_DIR/ui" ]]; then
        mv "$INSTALL_DIR/ui" "$INSTALL_DIR/ui.old" 2>/dev/null || true
    fi
    mkdir -p "$INSTALL_DIR/ui"
    tar xzf "$DOWNLOAD_DIR/extracted/juvia-ui.tar.gz" -C "$INSTALL_DIR/ui/"
    chown -R juvia:juvia "$INSTALL_DIR/ui" 2>/dev/null || true
    log_info "UI updated"
else
    log_warn "UI tarball not found in update package"
fi

# Update Caddyfile if new one is available
if [[ -f "$DOWNLOAD_DIR/extracted/config/Caddyfile" ]]; then
    cp "$DOWNLOAD_DIR/extracted/config/Caddyfile" "$CONFIG_DIR/caddy/Caddyfile"
    log_info "Caddyfile updated"
    # Reload Caddy so the new config takes effect
    if systemctl is-active juvia-caddy &>/dev/null; then
        systemctl reload juvia-caddy 2>/dev/null || true
    fi
fi

# Update CLI in /usr/local/bin
if [[ -f "$DOWNLOAD_DIR/extracted/scripts/juvia" ]]; then
    install -m 755 "$DOWNLOAD_DIR/extracted/scripts/juvia" /usr/local/bin/juvia
    log_info "Juvia CLI updated"
fi

rm -rf "$DOWNLOAD_DIR"
log_info "Binaries and UI updated"

log_step "Step 5: Starting services"
systemctl start juvia-agent 2>/dev/null || true
sleep 2
systemctl start juvia-api
if systemctl is-enabled juvia-caddy &>/dev/null; then
    systemctl start juvia-caddy 2>/dev/null || true
fi
log_info "Services started"

log_step "Step 6: Verifying update"
sleep 5

wait_for_healthy() {
    local max_wait=60
    local waited=0
    while [[ $waited -lt $max_wait ]]; do
        if curl -sf --max-time 5 http://localhost:9090/health > /dev/null 2>&1; then
            return 0
        fi
        sleep 2
        waited=$((waited + 2))
    done
    return 1
}

if wait_for_healthy; then
    log_info "Juvia Panel API is healthy"

    log_step "Step 7: Cleanup and post-update"

    echo "$NEW_VERSION" > "$CURRENT_VERSION_FILE"

    if [[ -f "/usr/local/bin/juvia-api.old" ]] && [[ "$AUTO_ROLLBACK" == "true" ]]; then
        rm -f /usr/local/bin/juvia-api.old
        log_info "Cleaned up old juvia-api binary"
    fi
    if [[ -f "/usr/local/bin/juvia-agent.old" ]] && [[ "$AUTO_ROLLBACK" == "true" ]]; then
        rm -f /usr/local/bin/juvia-agent.old
        log_info "Cleaned up old juvia-agent binary"
    fi

    find "$DATA_DIR/backups/update" -type f -mtime +7 -delete 2>/dev/null || true
    log_info "Cleaned up backups older than 7 days"

    if [[ -f "$DATA_DIR/panel.db" ]]; then
        UPDATE_ID=$(date +%s)
        sqlite3 "$DATA_DIR/panel.db" "INSERT INTO settings (key, value, updated_at) VALUES ('last_update_id', '$UPDATE_ID', datetime('now')) ON CONFLICT(key) DO UPDATE SET value='$UPDATE_ID', updated_at=datetime('now');" 2>/dev/null || true
    fi

    log_info ""
    log_info "========================================="
    log_info "Update complete! Now running version: $NEW_VERSION"
    log_info "========================================="

else
    if [[ "$AUTO_ROLLBACK" == "true" ]]; then
        log_error "Health check failed. Rolling back..."
        rollback_update
    else
        log_error "Health check failed. Run 'juvia update rollback' to restore."
    fi
    exit 1
fi