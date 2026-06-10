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
    log_info "To clean up partial installation, run: sudo juvia uninstall --purge"
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

# Always build from source to ensure latest code is used
BUILD_FROM_SOURCE=true
RELEASE_TAG=""

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

    # Install nixpacks for build support (auto/nixpacks build strategy)
    if ! command -v nixpacks &> /dev/null; then
        log_info "Installing nixpacks (build tool)..."
        if ! command -v cargo &> /dev/null; then
            log_info "Installing Rust toolchain for nixpacks..."
            curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
        fi
        NIXPACKS_DIR="$HOME/.cargo/bin"
        if [[ -f "$NIXPACKS_DIR/nixpacks" ]]; then
            cp "$NIXPACKS_DIR/nixpacks" /usr/local/bin/nixpacks
            chmod +x /usr/local/bin/nixpacks
            chown root:root /usr/local/bin/nixpacks
            log_info "nixpacks installed"
        else
            log_warn "nixpacks not found after cargo install, skipping"
        fi
    fi

    # Build frontend
    if ! command -v npm &> /dev/null; then
        log_info "Installing Node.js..."
        curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
        apt-get install -y nodejs
    fi

    cd "$TEMP_CLONE_DIR/frontend"
    npm install --legacy-peer-deps
    npm run build

    # Install UI (server mode)
    if [[ -d "$TEMP_CLONE_DIR/frontend/.next" ]]; then
        rm -rf "$INSTALL_DIR/ui"
        mkdir -p "$INSTALL_DIR/ui"
        cp -r "$TEMP_CLONE_DIR/frontend/.next" "$INSTALL_DIR/ui/.next"
        cp "$TEMP_CLONE_DIR/frontend/package.json" "$INSTALL_DIR/ui/"
        cp "$TEMP_CLONE_DIR/frontend/next.config.js" "$INSTALL_DIR/ui/"
        cp "$TEMP_CLONE_DIR/frontend/.env.production" "$INSTALL_DIR/ui/" 2>/dev/null || true
        if [[ -d "$TEMP_CLONE_DIR/frontend/node_modules" ]]; then
            cp -r "$TEMP_CLONE_DIR/frontend/node_modules" "$INSTALL_DIR/ui/"
        fi
        chown -R juvia:juvia "$INSTALL_DIR/ui"
        log_info "UI installed to $INSTALL_DIR/ui (server mode)"
    fi

    # Copy migrations
    mkdir -p "$CONFIG_DIR/migrations"
    if [[ -d "$TEMP_CLONE_DIR/backend/migrations" ]]; then
        cp "$TEMP_CLONE_DIR/backend/migrations/"*.sql "$CONFIG_DIR/migrations/" 2>/dev/null || true
        chown -R juvia:juvia "$CONFIG_DIR/migrations"
    fi

    # Copy Caddyfile
    if [[ -f "$TEMP_CLONE_DIR/backend/config/Caddyfile" ]]; then
        mkdir -p "$CONFIG_DIR/caddy"
        cp "$TEMP_CLONE_DIR/backend/config/Caddyfile" "$CONFIG_DIR/caddy/Caddyfile"
        chown juvia:juvia "$CONFIG_DIR/caddy/Caddyfile"
    fi

    rm -rf "$TEMP_CLONE_DIR"
fi

if [[ -z "$REPO_VERSION" || "$REPO_VERSION" == "latest" ]]; then
    log_warn "Could not determine exact version, using 'unknown'"
    REPO_VERSION="unknown"
fi

chmod +x /usr/local/bin/juvia-{api,agent,cli}
log_info "Binaries installed (version: $REPO_VERSION)"

# Install CLI and scripts from release branch
mkdir -p "$CONFIG_DIR/scripts"
SCRIPTS_BASE="https://raw.githubusercontent.com/marufnwu/Juvia-Panel/${REPO_BRANCH}/scripts"
for script in juvia install.sh update.sh uninstall.sh reset.sh; do
    curl -sSL "$SCRIPTS_BASE/$script" -o "/tmp/$script" 2>/dev/null || true
done
if [[ -f "/tmp/juvia" ]]; then
    chmod +x /tmp/juvia
    mv /tmp/juvia "/usr/local/bin/juvia"
    for script in install.sh update.sh uninstall.sh reset.sh; do
        if [[ -f "/tmp/$script" ]]; then
            chmod +x "/tmp/$script"
            mv "/tmp/$script" "$CONFIG_DIR/scripts/$script"
        fi
    done
    log_info "Installed juvia CLI and scripts"
fi

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

# Verify all required key files exist
for keyfile in "$CONFIG_DIR/keys/master" "$CONFIG_DIR/jwt-secret" "$CONFIG_DIR/encryption-key"; do
    if [[ ! -f "$keyfile" ]]; then
        log_error "Required key file missing: $keyfile"
        exit 1
    fi
done
log_info "All key files verified"

step "Initializing database"
DB_PATH="$DATA_DIR/panel.db"

# Create database file if needed
if [[ ! -f "$DB_PATH" ]]; then
    touch "$DB_PATH"
    chown juvia:juvia "$DB_PATH" 2>/dev/null || true
fi

# Set WAL mode and other pragmas. The Go API will run migrations on first start
# using embedded SQL migrations - this avoids permission issues and double-application.
sqlite3 "$DB_PATH" "PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;" 2>/dev/null || true
log_info "Database initialized. Migrations will be applied by API on first start."

step "Setting up systemd services"

JWT_SECRET=$(cat "$CONFIG_DIR/jwt-secret")
MASTER_KEY=$(cat "$CONFIG_DIR/keys/master")
ENCRYPTION_KEY=$(cat "$CONFIG_DIR/encryption-key")

# Create environment file for secrets (readable only by root:juvia)
cat > "$CONFIG_DIR/env" <<EOF
PANEL_JWT_SECRET=$JWT_SECRET
PANEL_MASTER_KEY=$MASTER_KEY
PANEL_ENCRYPTION_KEY=$ENCRYPTION_KEY
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
WorkingDirectory=$DATA_DIR
ExecStart=/usr/local/bin/juvia-agent --config $CONFIG_DIR/config.yml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
Environment=PANEL_AGENT_SOCKET=/var/run/panel/agent.sock
Environment=PANEL_DATA_DIR=$DATA_DIR
Environment=PANEL_CONFIG_DIR=$CONFIG_DIR
Environment=PANEL_MIGRATIONS_DIR=$CONFIG_DIR/migrations
EnvironmentFile=$CONFIG_DIR/env

# Security hardening
NoNewPrivileges=false
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/run/panel $DATA_DIR/tmp $DATA_DIR/logs

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=juvia-agent

# Hardening
LimitNOFILE=65536
TimeoutStartSec=30
TimeoutStopSec=30

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/juvia-api.service <<EOF
[Unit]
Description=Juvia Panel API Server
After=network.target docker.service juvia-agent.service
Requires=docker.service

[Service]
Type=simple
User=juvia
Group=juvia
WorkingDirectory=$DATA_DIR
ExecStart=/usr/local/bin/juvia-api --config $CONFIG_DIR/config.yml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
Environment=PANEL_DATA_DIR=$DATA_DIR
Environment=PANEL_CONFIG_DIR=$CONFIG_DIR
Environment=PANEL_MIGRATIONS_DIR=$CONFIG_DIR/migrations
Environment=PANEL_CADDY_CONFIG=$CONFIG_DIR/caddy/Caddyfile
Environment=PANEL_LOG_DIR=$DATA_DIR/logs
EnvironmentFile=$CONFIG_DIR/env

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=juvia-api

# Hardening
LimitNOFILE=65536
TimeoutStartSec=30
TimeoutStopSec=30

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/juvia-ui.service <<EOF
[Unit]
Description=Juvia Panel UI (Next.js Server)
After=network.target juvia-api.service
Requires=juvia-api.service

[Service]
Type=simple
User=juvia
Group=juvia
WorkingDirectory=$INSTALL_DIR/ui
ExecStart=/usr/bin/node $INSTALL_DIR/ui/node_modules/.bin/next start --port 3000
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=juvia-ui
Environment=NODE_ENV=production
Environment=PORT=3000
EnvironmentFile=$CONFIG_DIR/env

LimitNOFILE=65536
TimeoutStartSec=30
TimeoutStopSec=10

NoNewPrivileges=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths=$INSTALL_DIR/ui/.next

[Install]
WantedBy=multi-user.target
EOF

if [[ "$SKIP_CADDY" == "false" ]]; then
    cat > /etc/systemd/system/juvia-caddy.service <<EOF
[Unit]
Description=Juvia Panel Caddy Reverse Proxy
After=network.target juvia-api.service juvia-ui.service
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
EnvironmentFile=$CONFIG_DIR/env

[Install]
WantedBy=multi-user.target
EOF
fi

systemctl daemon-reload

systemctl enable juvia-agent 2>/dev/null || true
systemctl enable juvia-api
systemctl enable juvia-ui
if [[ "$SKIP_CADDY" == "false" ]]; then
    systemctl enable juvia-caddy 2>/dev/null || true
fi

log_info "Systemd services configured"

step "Configuring firewall"
if [[ "$SKIP_FIREWALL" == "false" ]] && command -v ufw &> /dev/null; then
    if ! ufw status | grep -q "Status: active"; then
        ufw --force enable
    fi

    ufw allow 22/tcp comment 'SSH'
    ufw allow 80/tcp comment 'HTTP (for HTTPS redirect)'
    ufw allow 443/tcp comment 'HTTPS'
    ufw allow 2053/tcp comment 'Panel UI'

    ufw delete allow 8080/tcp 2>/dev/null || true

    log_info "Firewall configured with SSH (22), HTTP (80), HTTPS (443), Panel UI (2053) allowed"
    FIREWALL_RULES_ADDED='[{"port":22,"protocol":"tcp","action":"allow"},{"port":80,"protocol":"tcp","action":"allow"},{"port":443,"protocol":"tcp","action":"allow"},{"port":2053,"protocol":"tcp","action":"allow"}]'
else
    log_info "Firewall configuration skipped"
    FIREWALL_RULES_ADDED='[]'
fi

step "Starting services"
systemctl start juvia-agent 2>/dev/null || true
sleep 2
systemctl start juvia-api
systemctl start juvia-ui

if [[ "$SKIP_CADDY" == "false" ]]; then
    systemctl start juvia-caddy 2>/dev/null || true
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
    log_info "  https://$SERVER_IP (or http://$SERVER_IP for initial access)"
fi
log_info ""
log_info "Next steps:"
log_info "  1. Create your admin account at the URL above"
log_info "  2. Configure your server domain and SSL"
log_info "  3. Deploy your first app"

step "Creating installation manifest"

# Build manifest using jq to ensure valid JSON
PACKAGES_JSON=$(printf '%s\n' "${PACKAGES_INSTALLED[@]}" | jq -R . | jq -s .)
API_SIZE=$(stat -c%s /usr/local/bin/juvia-api 2>/dev/null || echo 0)
AGENT_SIZE=$(stat -c%s /usr/local/bin/juvia-agent 2>/dev/null || echo 0)
CLI_SIZE=$(stat -c%s /usr/local/bin/juvia-cli 2>/dev/null || echo 0)
JUVIA_UID=$(id -u juvia 2>/dev/null || echo null)

jq -n \
  --arg version "${REPO_VERSION:-unknown}" \
  --arg installed_at "$(date -Iseconds)" \
  --arg repo_url "$REPO_URL" \
  --arg repo_branch "$REPO_BRANCH" \
  --arg os "$OS_ID $OS_VERSION" \
  --arg kernel "$KERNEL_VERSION" \
  --arg arch "$ARCH" \
  --arg data_dir "$DATA_DIR" \
  --arg config_dir "$CONFIG_DIR" \
  --arg install_dir "$INSTALL_DIR" \
  --argjson juvia_uid "$JUVIA_UID" \
  --argjson api_size "$API_SIZE" \
  --argjson agent_size "$AGENT_SIZE" \
  --argjson cli_size "$CLI_SIZE" \
  --argjson packages "$PACKAGES_JSON" \
  --argjson firewall "$FIREWALL_RULES_ADDED" \
  '{
    version: $version,
    installed_at: $installed_at,
    installer_version: "2.0.0",
    source: "git",
    repo_url: $repo_url,
    repo_branch: $repo_branch,
    system: {
      os: $os,
      kernel: $kernel,
      architecture: $arch
    },
    users_created: [
      {
        username: "juvia",
        uid: $juvia_uid,
        home: $data_dir,
        groups: ["juvia", "docker"]
      }
    ],
    directories_created: [
      $data_dir,
      $config_dir,
      $install_dir,
      "/var/run/panel"
    ],
    files_installed: [
      {path: "/usr/local/bin/juvia-api", size: $api_size},
      {path: "/usr/local/bin/juvia-agent", size: $agent_size},
      {path: "/usr/local/bin/juvia-cli", size: $cli_size}
    ],
    services_created: [
      "juvia-agent.service",
      "juvia-api.service",
      "juvia-ui.service",
      "juvia-caddy.service"
    ],
    firewall_rules_added: $firewall,
    packages_installed: $packages
  }' > "$DATA_DIR/.juvia-manifest.json"

chown juvia:juvia "$DATA_DIR/.juvia-manifest.json" 2>/dev/null || true

log_info "Manifest created at $DATA_DIR/.juvia-manifest.json"
log_info ""
log_info "========================================="
log_info "Installation successful!"
log_info "========================================="