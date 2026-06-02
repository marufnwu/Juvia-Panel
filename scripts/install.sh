#!/bin/bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

REPO_URL="${PANEL_REPO_URL:-https://github.com/marufnwu/Juvia-Panel.git}"
REPO_BRANCH="${PANEL_REPO_BRANCH:-master}"
DATA_DIR="${PANEL_DATA_DIR:-/var/panel}"
CONFIG_DIR="${PANEL_CONFIG_DIR:-/etc/panel}"
INSTALL_DIR="${PANEL_INSTALL_DIR:-/opt/panel}"
DOMAIN="${PANEL_DOMAIN:-}"
EMAIL="${PANEL_EMAIL:-}"
SKIP_DOCKER=false
SKIP_CADDY=false
SKIP_FIREWALL=false
DEBUG=false

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

CURRENT_STEP=0
TOTAL_STEPS=11

step() {
    CURRENT_STEP=$((CURRENT_STEP + 1))
    log_step "Step $CURRENT_STEP/$TOTAL_STEPS: $1"
}

cleanup_on_error() {
    local exit_code=$?
    log_error "Installation failed at step $CURRENT_STEP with exit code $exit_code"
    log_info "Check /var/log/panel-install.log for details"
    log_info "To clean up partial installation, run: sudo juvia-uninstall --purge"
    exit $exit_code
}
trap cleanup_on_error ERR

while [[ $# -gt 0 ]]; do
    case $1 in
        --repo-url) REPO_URL="$2"; shift 2 ;;
        --repo-branch) REPO_BRANCH="$2"; shift 2 ;;
        --data-dir) DATA_DIR="$2"; shift 2 ;;
        --config-dir) CONFIG_DIR="$2"; shift 2 ;;
        --install-dir) INSTALL_DIR="$2"; shift 2 ;;
        --domain) DOMAIN="$2"; shift 2 ;;
        --email) EMAIL="$2"; shift 2 ;;
        --skip-docker) SKIP_DOCKER=true; shift ;;
        --skip-caddy) SKIP_CADDY=true; shift ;;
        --skip-firewall) SKIP_FIREWALL=true; shift ;;
        --debug) DEBUG=true; shift ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

[[ "$DEBUG" == "true" ]] && set -x

step "Checking system requirements"
if ! command -v apt-get &> /dev/null; then
    log_error "This script requires Debian/Ubuntu-based Linux"
    exit 1
fi

if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

OS_ID=$(lsb_release -is 2>/dev/null || echo "unknown")
OS_ID_LIKE=$(cat /etc/os-release 2>/dev/null | grep -i "^ID_LIKE=" | cut -d= -f2 | tr -d '"' || echo "")
OS_VERSION=$(lsb_release -rs 2>/dev/null || echo "unknown")
KERNEL_VERSION=$(uname -r)
ARCH=$(uname -m)

log_info "System: $OS_ID $OS_VERSION (kernel $KERNEL_VERSION, arch $ARCH)"

if [[ "$OS_ID" != "Ubuntu" && "$OS_ID" != "Debian" && ! "$OS_ID_LIKE" =~ ^(ubuntu|debian) ]]; then
    log_error "Unsupported OS: $OS_ID. Supported: Ubuntu 24.04+, Debian 12+"
    exit 1
fi

if [[ "$ARCH" != "x86_64" && "$ARCH" != "amd64" ]]; then
    log_warn "Architecture $ARCH is not officially supported (amd64 only)"
fi

KERNEL_MAJOR=$(echo "$KERNEL_VERSION" | cut -d. -f1)
KERNEL_MINOR=$(echo "$KERNEL_VERSION" | cut -d. -f2)
if [[ "$KERNEL_MAJOR" -lt 6 ]] || [[ "$KERNEL_MAJOR" -eq 6 && "$KERNEL_MINOR" -lt 8 ]]; then
    log_warn "Kernel $KERNEL_VERSION is older than recommended (6.8+)"
fi

TOTAL_MEM_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
TOTAL_MEM_GB=$((TOTAL_MEM_KB / 1024 / 1024))
if [[ "$TOTAL_MEM_GB" -lt 2 ]]; then
    log_error "Insufficient RAM: ${TOTAL_MEM_GB}GB (minimum 2GB required)"
    exit 1
fi
log_info "RAM: ${TOTAL_MEM_GB}GB"

AVAILABLE_DISK_KB=$(df -k / | tail -1 | awk '{print $4}')
AVAILABLE_DISK_GB=$((AVAILABLE_DISK_KB / 1024 / 1024))
if [[ "$AVAILABLE_DISK_GB" -lt 20 ]]; then
    log_error "Insufficient disk space: ${AVAILABLE_DISK_GB}GB available (minimum 20GB required)"
    exit 1
fi
log_info "Disk space: ${AVAILABLE_DISK_GB}GB available"

if ! command -v curl &> /dev/null; then
    apt-get update -qq
    apt-get install -y curl
fi

if ! command -v jq &> /dev/null; then
    apt-get update -qq
    apt-get install -y jq
fi

log_info "System check passed"

step "Installing dependencies"
PACKAGES_INSTALLED=()

if [[ "$SKIP_DOCKER" == "false" ]]; then
    log_info "Installing Docker CE and Docker Compose plugin..."

    if ! command -v docker &> /dev/null || ! docker version &>/dev/null; then
        apt-get update -qq
        apt-get install -y ca-certificates curl gnupg

        install -m 0755 -d /etc/apt/keyrings
        DOCKER_DISTRO=$(lsb_release -is | tr '[:upper:]' '[:lower:]')
        curl -fsSL https://download.docker.com/linux/$DOCKER_DISTRO/gpg -o /etc/apt/keyrings/docker.gpg
        chmod a+r /etc/apt/keyrings/docker.gpg

        DOCKER_CODENAME=$(lsb_release -cs 2>/dev/null || echo "stable")
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$DOCKER_DISTRO $DOCKER_CODENAME stable" > /etc/apt/sources.list.d/docker.list

        apt-get update -qq
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

        systemctl enable docker
        systemctl start docker
        PACKAGES_INSTALLED+=("docker-ce" "docker-ce-cli" "containerd.io" "docker-buildx-plugin" "docker-compose-plugin")
    else
        log_warn "Docker already installed, skipping"
    fi

    DOCKER_VERSION=$(docker --version 2>/dev/null | grep -oP '\d+\.\d+\.\d+' | head -1 || echo "unknown")
    log_info "Docker version: $DOCKER_VERSION"
fi

if [[ "$SKIP_CADDY" == "false" ]]; then
    log_info "Installing Caddy..."

    if ! command -v caddy &> /dev/null; then
        apt-get update -qq
        apt-get install -y debian-keyring debian-archive-keyring apt-transport-https

        curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg 2>/dev/null || true
        curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null

        apt-get update -qq
        apt-get install -y caddy
        PACKAGES_INSTALLED+=("caddy")
    else
        log_warn "Caddy already installed, skipping"
    fi

    CADDY_VERSION=$(caddy version 2>/dev/null | head -1 || echo "unknown")
    log_info "Caddy version: $CADDY_VERSION"
fi

if ! command -v git &> /dev/null; then
    apt-get update -qq
    apt-get install -y git
    PACKAGES_INSTALLED+=("git")
fi

if ! command -v sqlite3 &> /dev/null; then
    apt-get update -qq
    apt-get install -y sqlite3
    PACKAGES_INSTALLED+=("sqlite3")
fi

log_info "Dependencies installed"

step "Creating Juvia Panel user and directories"
if ! id juvia &>/dev/null; then
    useradd -r -s /bin/false -d "$DATA_DIR" juvia
    log_info "Created juvia user"
else
    log_info "juvia user already exists"
fi

usermod -aG docker juvia

mkdir -p "$DATA_DIR"/{apps,backups,logs,volumes,tmp/builds}
mkdir -p "$CONFIG_DIR"/{caddy,migrations,keys}
mkdir -p "$INSTALL_DIR"/{bin,ui}
mkdir -p /var/run/panel
mkdir -p /usr/local/bin

chown -R juvia:juvia "$DATA_DIR" 2>/dev/null || true
chown -R juvia:juvia "$CONFIG_DIR" 2>/dev/null || true
chown -R juvia:juvia "$INSTALL_DIR" 2>/dev/null || true
chown -R juvia:juvia /var/run/panel 2>/dev/null || true

chmod 750 "$DATA_DIR"
chmod 750 "$CONFIG_DIR"
chmod 755 "$INSTALL_DIR"
chmod 755 /var/run/panel

log_info "Directories created with ownership set to juvia:juvia"

step "Downloading Juvia Panel binaries"
log_info "Fetching latest release from GitHub..."

ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH_SUFFIX="amd64" ;;
    aarch64|arm64) ARCH_SUFFIX="arm64" ;;
    *) log_error "Unsupported architecture: $ARCH"; exit 1 ;;
esac

DOWNLOAD_DIR="/tmp/juvia-panel-download"
rm -rf "$DOWNLOAD_DIR"
mkdir -p "$DOWNLOAD_DIR"

# Get latest release tag from GitHub API
RELEASE_TAG=$(curl -sf "https://api.github.com/repos/marufnwu/Juvia-Panel/releases/latest" | jq -r '.tag_name // empty' 2>/dev/null || echo "")

BUILD_FROM_SOURCE=false

if [[ -z "$RELEASE_TAG" ]]; then
    log_warn "Could not fetch latest release, will build from source"
    BUILD_FROM_SOURCE=true
else
    log_info "Latest release: $RELEASE_TAG"
    BASE_URL="https://github.com/marufnwu/Juvia-Panel/releases/download/${RELEASE_TAG}"

    BUNDLE_URL="${BASE_URL}/juvia-release-${ARCH_SUFFIX}.tar.gz"
    log_info "Downloading bundle from $BUNDLE_URL..."

    if ! curl -sfL "$BUNDLE_URL" -o "$DOWNLOAD_DIR/juvia-release.tar.gz"; then
        log_warn "Failed to download release bundle, will build from source"
        BUILD_FROM_SOURCE=true
    fi
fi

if [[ "$BUILD_FROM_SOURCE" == "false" ]]; then
    log_info "Extracting release bundle..."
    mkdir -p "$DOWNLOAD_DIR/extracted"
    tar xzf "$DOWNLOAD_DIR/juvia-release.tar.gz" -C "$DOWNLOAD_DIR/extracted/"

    # Install binaries
    for binary in juvia-api juvia-agent juvia-cli; do
        if [[ -f "$DOWNLOAD_DIR/extracted/$binary" ]]; then
            chmod +x "$DOWNLOAD_DIR/extracted/$binary"
            mv "$DOWNLOAD_DIR/extracted/$binary" "/usr/local/bin/$binary"
            log_info "Installed $binary"
        else
            log_warn "Binary $binary not found in bundle"
            BUILD_FROM_SOURCE=true
        fi
    done

    # Install UI
    if [[ -f "$DOWNLOAD_DIR/extracted/juvia-ui.tar.gz" ]]; then
        mkdir -p "$INSTALL_DIR/ui"
        tar xzf "$DOWNLOAD_DIR/extracted/juvia-ui.tar.gz" -C "$INSTALL_DIR/ui/"
        log_info "UI installed to $INSTALL_DIR/ui"
    fi

    # Copy migrations
    if [[ -d "$DOWNLOAD_DIR/extracted/migrations" ]]; then
        mkdir -p "$CONFIG_DIR/migrations"
        cp "$DOWNLOAD_DIR/extracted/migrations/"*.sql "$CONFIG_DIR/migrations/" 2>/dev/null || true
        log_info "Migrations copied to $CONFIG_DIR/migrations"
    fi

    # Copy Caddyfile
    if [[ -f "$DOWNLOAD_DIR/extracted/config/Caddyfile" ]]; then
        mkdir -p "$CONFIG_DIR/caddy"
        cp "$DOWNLOAD_DIR/extracted/config/Caddyfile" "$CONFIG_DIR/caddy/Caddyfile"
        log_info "Caddyfile copied to $CONFIG_DIR/caddy"
    fi

    REPO_VERSION="$RELEASE_TAG"
fi

if [[ "$BUILD_FROM_SOURCE" == "true" ]]; then
    log_info "Building from source..."
    TEMP_CLONE_DIR="/tmp/juvia-panel-clone"
    rm -rf "$TEMP_CLONE_DIR"

    log_info "Cloning $REPO_URL (branch: $REPO_BRANCH)..."
    git clone --depth 1 --branch "$REPO_BRANCH" "$REPO_URL" "$TEMP_CLONE_DIR"

    if [[ ! -d "$TEMP_CLONE_DIR" ]]; then
        log_error "Failed to clone repository"
        exit 1
    fi

    REPO_VERSION=$(git -C "$TEMP_CLONE_DIR" describe --tags 2>/dev/null || git -C "$TEMP_CLONE_DIR" rev-parse --short HEAD 2>/dev/null || echo "latest")
    log_info "Cloned repository at version: $REPO_VERSION"

    # Build Go binaries
    cd "$TEMP_CLONE_DIR/backend"

    if ! command -v go &> /dev/null; then
        log_info "Installing Go..."
        apt-get update -qq
        apt-get install -y golang-go
    fi

    log_info "Building Go binaries..."
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /usr/local/bin/juvia-api ./cmd/api/
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /usr/local/bin/juvia-agent ./cmd/agent/
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /usr/local/bin/juvia-cli ./cmd/debug/

    chmod +x /usr/local/bin/juvia-{api,agent,cli}

    # Build frontend
    if ! command -v npm &> /dev/null; then
        log_info "Installing Node.js..."
        curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
        apt-get install -y nodejs
    fi

    cd "$TEMP_CLONE_DIR/frontend"
    npm install --legacy-peer-deps
    npm run build

    # Install UI (standalone mode)
    if [[ -f "$TEMP_CLONE_DIR/frontend/.next/standalone/server.js" ]]; then
        rm -rf "$INSTALL_DIR/ui"
        mkdir -p "$INSTALL_DIR/ui"
        cp -r "$TEMP_CLONE_DIR/frontend/.next/standalone/"* "$INSTALL_DIR/ui/"
        cp -r "$TEMP_CLONE_DIR/frontend/.next" "$INSTALL_DIR/ui/"
        cp "$TEMP_CLONE_DIR/frontend/package.json" "$INSTALL_DIR/ui/"
        chown -R juvia:juvia "$INSTALL_DIR/ui"
        log_info "UI (standalone) installed to $INSTALL_DIR/ui"
    elif [[ -d "$TEMP_CLONE_DIR/frontend/out" ]]; then
        rm -rf "$INSTALL_DIR/ui/out"
        cp -r "$TEMP_CLONE_DIR/frontend/out" "$INSTALL_DIR/ui/"
        log_info "UI (static) installed to $INSTALL_DIR/ui"
    fi

    # Copy migrations
    mkdir -p "$CONFIG_DIR/migrations"
    if [[ -d "$TEMP_CLONE_DIR/backend/migrations" ]]; then
        cp "$TEMP_CLONE_DIR/backend/migrations/"*.sql "$CONFIG_DIR/migrations/" 2>/dev/null || true
    fi
    if [[ -d "$TEMP_CLONE_DIR/scripts/migrations" ]]; then
        cp "$TEMP_CLONE_DIR/scripts/migrations/"*.sql "$CONFIG_DIR/migrations/" 2>/dev/null || true
    fi

    # Copy Caddyfile
    if [[ -f "$TEMP_CLONE_DIR/backend/config/Caddyfile" ]]; then
        mkdir -p "$CONFIG_DIR/caddy"
        cp "$TEMP_CLONE_DIR/backend/config/Caddyfile" "$CONFIG_DIR/caddy/Caddyfile"
    fi

    rm -rf "$TEMP_CLONE_DIR"
fi

chmod +x /usr/local/bin/juvia-{api,agent,cli}
log_info "Binaries installed (version: $REPO_VERSION)"

step "Generating configuration"
openssl rand -hex 32 > "$CONFIG_DIR/keys/master"
chmod 600 "$CONFIG_DIR/keys/master"

openssl rand -hex 64 > "$CONFIG_DIR/jwt-secret"
chmod 600 "$CONFIG_DIR/jwt-secret"

openssl rand -hex 32 > "$CONFIG_DIR/encryption-key"
chmod 600 "$CONFIG_DIR/encryption-key"

cat > "$CONFIG_DIR/config.yml" <<EOF
app:
  name: "Juvia Panel"
  host: 127.0.0.1
  port: 9090
  env: production
  data_dir: $DATA_DIR
  log_dir: $DATA_DIR/logs
  install_dir: $INSTALL_DIR

database:
  url: sqlite://$DATA_DIR/panel.db

security:
  master_key_file: $CONFIG_DIR/keys/master
  jwt_secret_file: $CONFIG_DIR/jwt-secret
  encryption_key_file: $CONFIG_DIR/encryption-key

agent:
  socket: /var/run/panel/agent.sock
  tcp_port: 9091

caddy:
  config: $CONFIG_DIR/caddy/Caddyfile
  data_dir: $DATA_DIR/caddy

server:
  domain: "$DOMAIN"
  panel_domain: "$DOMAIN"
  email: "$EMAIL"
EOF

chmod 644 "$CONFIG_DIR/config.yml"
log_info "Configuration generated at $CONFIG_DIR/config.yml"

step "Initializing database"
DB_PATH="$DATA_DIR/panel.db"

if [[ -f "$DB_PATH" ]]; then
    log_warn "Database already exists at $DB_PATH"
    # Check if tables already exist
    TABLE_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';" 2>/dev/null || echo "0")
    if [[ "$TABLE_COUNT" -gt 0 ]]; then
        log_info "Database already has $TABLE_COUNT tables, skipping migrations"
    else
        log_info "Applying database migrations..."
        for migration in "$CONFIG_DIR/migrations"/000001_init.up.sql "$CONFIG_DIR/migrations"/000002_settings_and_exports.up.sql; do
            if [[ -f "$migration" ]]; then
                log_info "Applying $(basename "$migration")..."
                sqlite3 "$DB_PATH" < "$migration" 2>/dev/null || true
            else
                log_warn "Migration not found: $migration"
            fi
        done
    fi
else
    touch "$DB_PATH"
    chown juvia:juvia "$DB_PATH" 2>/dev/null || true
    log_info "Applying database migrations..."
    for migration in "$CONFIG_DIR/migrations"/000001_init.up.sql "$CONFIG_DIR/migrations"/000002_settings_and_exports.up.sql; do
        if [[ -f "$migration" ]]; then
            log_info "Applying $(basename "$migration")..."
            sqlite3 "$DB_PATH" < "$migration" 2>/dev/null || true
        else
            log_warn "Migration not found: $migration"
        fi
    done
fi

sqlite3 "$DB_PATH" "PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;"
log_info "Database initialized with WAL mode and foreign keys enabled"

step "Setting up systemd services"

JWT_SECRET=$(cat "$CONFIG_DIR/jwt-secret")
MASTER_KEY=$(cat "$CONFIG_DIR/keys/master")

# Create environment file for secrets (readable only by root:juvia)
cat > "$CONFIG_DIR/env" <<EOF
PANEL_JWT_SECRET=$JWT_SECRET
PANEL_MASTER_KEY=$MASTER_KEY
EOF
chmod 640 "$CONFIG_DIR/env"
chown root:juvia "$CONFIG_DIR/env"

cat > /etc/systemd/system/juvia-agent.service <<EOF
[Unit]
Description=Juvia Panel Agent
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=juvia
Group=juvia
ExecStart=/usr/local/bin/juvia-agent start --config $CONFIG_DIR/config.yml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
EnvironmentFile=$CONFIG_DIR/env

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/juvia-api.service <<EOF
[Unit]
Description=Juvia Panel API Server
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=juvia
Group=juvia
ExecStart=/usr/local/bin/juvia-api serve --config $CONFIG_DIR/config.yml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
EnvironmentFile=$CONFIG_DIR/env

[Install]
WantedBy=multi-user.target
EOF

if [[ "$SKIP_CADDY" == "false" ]]; then
    cat > /etc/systemd/system/juvia-caddy.service <<EOF
[Unit]
Description=Juvia Panel Caddy Reverse Proxy
After=network.target juvia-api.service
Requires=juvia-api.service

[Service]
Type=simple
User=juvia
Group=juvia
ExecStart=/usr/bin/caddy run --config $CONFIG_DIR/caddy/Caddyfile --adapter caddyfile
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
fi

# Next.js UI service (standalone mode)
if [[ -f "$INSTALL_DIR/ui/server.js" ]]; then
    cat > /etc/systemd/system/juvia-ui.service <<EOF
[Unit]
Description=Juvia Panel UI (Next.js)
After=network.target juvia-api.service

[Service]
Type=simple
User=juvia
Group=juvia
WorkingDirectory=$INSTALL_DIR/ui
ExecStart=/usr/bin/node $INSTALL_DIR/ui/server.js
Restart=always
RestartSec=5
Environment=PORT=3000
Environment=NODE_ENV=production
EnvironmentFile=$CONFIG_DIR/env

[Install]
WantedBy=multi-user.target
EOF
fi

systemctl daemon-reload

systemctl enable juvia-agent 2>/dev/null || true
systemctl enable juvia-api
if [[ "$SKIP_CADDY" == "false" ]]; then
    systemctl enable juvia-caddy 2>/dev/null || true
fi
if [[ -f "$INSTALL_DIR/ui/server.js" ]]; then
    systemctl enable juvia-ui 2>/dev/null || true
fi

log_info "Systemd services configured"

step "Configuring firewall"
if [[ "$SKIP_FIREWALL" == "false" ]] && command -v ufw &> /dev/null; then
    if ! ufw status | grep -q "Status: active"; then
        ufw --force enable
    fi

    ufw allow 22/tcp comment 'SSH'
    ufw allow 80/tcp comment 'HTTP (for HTTPS redirect)'
    ufw allow 2053/tcp comment 'Juvia Panel'
    ufw allow 443/tcp comment 'HTTPS'

    ufw delete allow 8080/tcp 2>/dev/null || true

    log_info "Firewall configured with SSH, HTTP, Juvia Panel (2053), HTTPS allowed"
    FIREWALL_RULES_ADDED='[{"port":22,"protocol":"tcp","action":"allow"},{"port":80,"protocol":"tcp","action":"allow"},{"port":2053,"protocol":"tcp","action":"allow"},{"port":443,"protocol":"tcp","action":"allow"}]'
else
    log_info "Firewall configuration skipped"
    FIREWALL_RULES_ADDED='[]'
fi

step "Starting services"
systemctl start juvia-agent 2>/dev/null || true
sleep 2
systemctl start juvia-api

if [[ "$SKIP_CADDY" == "false" ]]; then
    systemctl start juvia-caddy 2>/dev/null || true
fi

if [[ -f "$INSTALL_DIR/ui/server.js" ]]; then
    systemctl start juvia-ui 2>/dev/null || true
fi

log_info "Waiting for services to become healthy..."

MAX_WAIT=60
WAITED=0
while [[ $WAITED -lt $MAX_WAIT ]]; do
    if curl -sf --max-time 5 http://localhost:9090/health > /dev/null 2>&1; then
        log_info "Juvia Panel API is healthy"
        break
    fi
    sleep 2
    WAITED=$((WAITED + 2))
done

if [[ $WAITED -ge $MAX_WAIT ]]; then
    log_warn "Health check timeout reached, but installation continued"
fi

step "Completing setup"
SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || echo "localhost")
log_info "Installation complete!"
log_info ""
log_info "Juvia Panel is running at:"
if [[ -n "$DOMAIN" ]]; then
    log_info "  https://panel.$DOMAIN"
else
    log_info "  http://$SERVER_IP:2053"
fi
log_info ""
log_info "Next steps:"
log_info "  1. Create your admin account at the URL above"
log_info "  2. Configure your server domain and SSL"
log_info "  3. Deploy your first app"

step "Creating installation manifest"

cat > "$DATA_DIR/.juvia-manifest.json" <<EOF
{
  "version": "$REPO_VERSION",
  "installed_at": "$(date -Iseconds)",
  "installer_version": "2.0.0",
  "source": "git",
  "repo_url": "$REPO_URL",
  "repo_branch": "$REPO_BRANCH",
  "system": {
    "os": "$OS_ID $OS_VERSION",
    "kernel": "$KERNEL_VERSION",
    "architecture": "$ARCH"
  },
  "users_created": [
    {
      "username": "juvia",
      "uid": $(id -u juvia 2>/dev/null || echo "null"),
      "home": "$DATA_DIR",
      "groups": ["juvia", "docker"]
    }
  ],
  "directories_created": [
    "$DATA_DIR",
    "$CONFIG_DIR",
    "$INSTALL_DIR",
    "/var/run/panel"
  ],
  "files_installed": [
    {"path": "/usr/local/bin/juvia-api", "size": $(stat -c%s /usr/local/bin/juvia-api 2>/dev/null || echo "0")},
    {"path": "/usr/local/bin/juvia-agent", "size": $(stat -c%s /usr/local/bin/juvia-agent 2>/dev/null || echo "0")},
    {"path": "/usr/local/bin/juvia-cli", "size": $(stat -c%s /usr/local/bin/juvia-cli 2>/dev/null || echo "0")}
  ],
  "services_created": [
    "juvia-agent.service",
    "juvia-api.service",
    "juvia-caddy.service",
    "juvia-ui.service"
  ],
  "firewall_rules_added": $FIREWALL_RULES_ADDED,
  "packages_installed": $(printf '%s\n' "${PACKAGES_INSTALLED[@]}" | jq -R . | jq -s .)
}
EOF

chown juvia:juvia "$DATA_DIR/.juvia-manifest.json" 2>/dev/null || true

log_info "Manifest created at $DATA_DIR/.juvia-manifest.json"
log_info ""
log_info "========================================="
log_info "Installation successful!"
log_info "========================================="