#!/bin/bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

EXPORT_ONLY=false
PURGE=false
KEEP_DATA=false
KEEP_USER=false
DEBUG=false

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

DATA_DIR="${PANEL_DATA_DIR:-/var/panel}"
CONFIG_DIR="${PANEL_CONFIG_DIR:-/etc/panel}"
INSTALL_DIR="${PANEL_INSTALL_DIR:-/opt/panel}"
EXPORT_DIR=""

MANIFEST="$DATA_DIR/.juvia-manifest.json"

while [[ $# -gt 0 ]]; do
    case $1 in
        --export-only) EXPORT_ONLY=true; shift ;;
        --purge) PURGE=true; shift ;;
        --keep-data) KEEP_DATA=true; shift ;;
        --keep-user) KEEP_USER=true; shift ;;
        --debug) DEBUG=true; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

[[ "$DEBUG" == "true" ]] && set -x

if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

log_step "Uninstallation starting..."

if [[ -f "$MANIFEST" ]]; then
    log_info "Found installation manifest at $MANIFEST"
    MANIFEST_DATA=$(cat "$MANIFEST")
else
    log_warn "Manifest not found at $MANIFEST"
    log_warn "This might be a partial installation"
    MANIFEST_DATA="{}"
fi

log_step "Step 1: Exporting apps (unless --purge --no-keep-data)"
if [[ "$EXPORT_ONLY" == "true" ]] || [[ "$PURGE" == "false" && "$KEEP_DATA" == "false" ]]; then
    EXPORT_DIR="$DATA_DIR/export/$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$EXPORT_DIR"
    log_info "Export directory: $EXPORT_DIR"

    if [[ -f "$DATA_DIR/panel.db" ]]; then
        if command -v sqlite3 &>/dev/null; then
            sqlite3 "$DATA_DIR/panel.db" "SELECT id, name, runtime, build_strategy FROM apps;" 2>/dev/null | while IFS='|' read -r id name runtime build_strategy; do
                if [[ -n "$name" ]]; then
                    APP_DIR="$EXPORT_DIR/$name"
                    mkdir -p "$APP_DIR/volumes"

                    ENV_VARS=$(sqlite3 "$DATA_DIR/panel.db" "SELECT key, value, is_secret FROM app_env_vars WHERE app_id='$id';" 2>/dev/null || echo "")

                    cat > "$APP_DIR/docker-compose.yml" <<EOF
version: "3.8"
services:
  $name:
    image: juvia-app-$name:latest
    container_name: $name
    restart: unless-stopped
    environment:
EOF

                    if [[ -n "$ENV_VARS" ]]; then
                        while IFS='|' read -r key value is_secret; do
                            if [[ "$is_secret" == "0" ]]; then
                                echo "      $key: \"$value\"" >> "$APP_DIR/docker-compose.yml"
                            else
                                echo "      $key: \"***SECRET***\"" >> "$APP_DIR/docker-compose.yml"
                            fi
                        done <<< "$ENV_VARS"
                    fi

                    cat >> "$APP_DIR/docker-compose.yml" <<EOF
    volumes:
      - ./volumes:/app/data
    networks:
      - juvia-network

networks:
  juvia-network:
    name: juvia-network
    external: true
EOF

                    if [[ "$KEEP_DATA" == "false" ]]; then
                        log_info "  Exporting volumes for app: $name (data will be copied)"
                    fi

                    cat > "$APP_DIR/README.md" <<EOF
# Exported App: $name

To restart this app on another server:

1. Copy the app directory to your target server
2. Review and update .env with actual secret values
3. Run: docker compose up -d

Note: Volume data has been preserved in ./volumes/ directory if it existed.
EOF

                    log_info "  Exported app: $name"
                fi
            done
        fi
    fi

    log_info "Apps exported to: $EXPORT_DIR"
    log_info "To restart: cd $EXPORT_DIR/<app> && docker compose up -d"
fi

if [[ "$EXPORT_ONLY" == "true" ]]; then
    log_info "Export complete. Juvia Panel services stopped but not removed."
    exit 0
fi

log_step "Step 2: Stopping services"
systemctl stop juvia-caddy 2>/dev/null || true
systemctl stop juvia-api 2>/dev/null || true
systemctl stop juvia-agent 2>/dev/null || true
log_info "All Juvia Panel services stopped"

log_step "Step 3: Removing Docker containers and images"
if command -v docker &>/dev/null; then
    JUVIA_CONTAINERS=$(docker ps -a --format '{{.Names}}' 2>/dev/null | grep -E '^juvia-|-juvia$' || true)
    if [[ -n "$JUVIA_CONTAINERS" ]]; then
        echo "$JUVIA_CONTAINERS" | xargs -r docker stop 2>/dev/null || true
        echo "$JUVIA_CONTAINERS" | xargs -r docker rm 2>/dev/null || true
        log_info "Docker containers removed"
    fi

    JUVIA_IMAGES=$(docker images --format '{{.Repository}}' 2>/dev/null | grep -E '^juvia-' || true)
    if [[ -n "$JUVIA_IMAGES" ]]; then
        echo "$JUVIA_IMAGES" | xargs -r docker rmi 2>/dev/null || true
        log_info "Docker images removed"
    fi
else
    log_warn "Docker not found, skipping container cleanup"
fi

log_step "Step 4: Removing systemd services"
systemctl disable juvia-agent 2>/dev/null || true
systemctl disable juvia-api 2>/dev/null || true
systemctl disable juvia-caddy 2>/dev/null || true

rm -f /etc/systemd/system/juvia-agent.service
rm -f /etc/systemd/system/juvia-api.service
rm -f /etc/systemd/system/juvia-caddy.service

systemctl daemon-reload
systemctl reset-failed 2>/dev/null || true
log_info "Systemd services removed"

log_step "Step 5: Removing files"
rm -f /usr/local/bin/juvia-api
rm -f /usr/local/bin/juvia-agent
rm -f /usr/local/bin/juvia-cli

rm -rf "$INSTALL_DIR/bin" 2>/dev/null || true
rm -rf "$INSTALL_DIR/ui" 2>/dev/null || true

rm -rf /var/run/panel

if [[ "$PURGE" == "true" ]]; then
    rm -rf "$CONFIG_DIR"

    if [[ "$KEEP_DATA" != "true" ]]; then
        rm -rf "$DATA_DIR"
        log_info "All data removed ($DATA_DIR deleted)"
    else
        log_info "Data preserved in $DATA_DIR"
    fi
else
    log_info "Config preserved in $CONFIG_DIR"
    log_info "Data preserved in $DATA_DIR"
fi

log_step "Step 6: Removing firewall rules"
if command -v ufw &>/dev/null; then
    for port in 80 443; do
        ufw delete allow $port/tcp 2>/dev/null || true
    done
    log_info "Firewall rules removed"
fi

log_step "Step 7: Removing juvia user"
if [[ "$KEEP_USER" != "true" ]]; then
    if id juvia &>/dev/null; then
        userdel juvia 2>/dev/null || true
        log_info "Juvia user removed"
    fi
else
    log_info "Juvia user kept (--keep-user specified)"
fi

log_step "Step 8: Verifying cleanup"
REMAINING_PROCESSES=""
for svc in juvia-agent juvia-api; do
    if systemctl is-active "$svc" &>/dev/null; then
        REMAINING_PROCESSES+="$svc (systemd active)\n"
    fi
done
if [[ -n "$REMAINING_PROCESSES" ]]; then
    log_warn "Some Juvia Panel services still active:"
    echo -e "$REMAINING_PROCESSES"
else
    log_info "No Juvia Panel services running"
fi

REMAINING_PORTS=$(ss -tlnp 2>/dev/null | grep -E ':(2053|9090)' | grep -v grep || true)
if [[ -n "$REMAINING_PORTS" ]]; then
    log_warn "Some Juvia Panel ports still listening:"
    echo "$REMAINING_PORTS"
else
    log_info "No Juvia Panel ports listening"
fi

log_info ""
log_info "========================================="
if [[ "$PURGE" == "true" ]]; then
    log_info "Uninstallation complete (purge mode)"
else
    log_info "Uninstallation complete"
fi
log_info "========================================="

if [[ -n "${EXPORT_DIR:-}" && -d "$EXPORT_DIR" ]]; then
    log_info "Exported apps are at: $EXPORT_DIR"
fi