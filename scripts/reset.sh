#!/bin/bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

DATA_DIR="${PANEL_DATA_DIR:-/var/panel}"
CONFIG_DIR="${PANEL_CONFIG_DIR:-/etc/panel}"
KEEP_BACKUP=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --yes) KEEP_BACKUP=false; shift ;;
        --debug) set -x; shift ;;
        *) echo "Usage: juvia reset [--yes] [--debug]"; exit 1 ;;
    esac
done

if [[ $EUID -ne 0 ]]; then
    log_error "This command must be run as root (use sudo)"
    exit 1
fi

log_step "Resetting Juvia Panel to fresh installation state..."

log_step "Step 1: Stopping API service"
systemctl stop juvia-api 2>/dev/null || true
log_info "API stopped"

log_step "Step 2: Backing up and removing database"
DB_PATH="$DATA_DIR/panel.db"
CONFIG_DB_PATH="$CONFIG_DIR/panel.db"

if [[ -f "$DB_PATH" ]]; then
    if [[ "$KEEP_BACKUP" == "true" ]]; then
        BACKUP_PATH="${DB_PATH}.backup.$(date +%Y%m%d-%H%M%S)"
        mv "$DB_PATH" "$BACKUP_PATH"
        log_info "Database backed up to: $BACKUP_PATH"
    else
        rm -f "$DB_PATH"
        log_info "Database removed"
    fi
elif [[ -f "$CONFIG_DB_PATH" ]]; then
    if [[ "$KEEP_BACKUP" == "true" ]]; then
        BACKUP_PATH="${CONFIG_DB_PATH}.backup.$(date +%Y%m%d-%H%M%S)"
        mv "$CONFIG_DB_PATH" "$BACKUP_PATH"
        log_info "Database backed up to: $BACKUP_PATH"
    else
        rm -f "$CONFIG_DB_PATH"
        log_info "Database removed"
    fi
else
    log_warn "No database found at $DB_PATH or $CONFIG_DB_PATH"
fi

log_step "Step 3: Starting API service"
systemctl start juvia-api 2>/dev/null || true
sleep 2

log_step "Step 4: Verifying reset"
if curl -sf --max-time 10 http://localhost:9090/health > /dev/null 2>&1; then
    log_info "API is healthy"
    log_info ""
    log_info "========================================="
    log_info "Reset complete! Database has been cleared."
    log_info "Visit your panel URL to create a new admin account."
    log_info "========================================="
else
    log_error "API failed to start. Check: journalctl -u juvia-api"
    exit 1
fi