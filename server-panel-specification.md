# Juvia Panel Project Specification
## Single-Server Self-Hosted PaaS for Developers

**Version:** 1.0  
**Date:** 2026-05-31  
**Status:** Draft  
**Target:** Single VPS / Dedicated Server Deployment

---

## 1. Executive Summary

### 1.1 Project Vision
Build a modern, single-server control panel that transforms a standard Linux VPS or dedicated server into a personal Platform-as-a-Service (PaaS). The panel enables developers and server owners to deploy applications, manage databases, configure domains, and handle SSL certificates through an intuitive web interface — without requiring DevOps expertise.

### 1.2 Core Philosophy
- **App-centric, not account-centric:** Users think in projects and applications, not hosting accounts.
- **Git-native deployment:** Push code, go live. The panel handles builds, containers, and routing.
- **Zero lock-in:** All applications run as standard Docker containers. On uninstall, the panel exports every app and service to a standalone `docker-compose.yml` with its `.env` file, then offers a full purge that removes all containers, volumes, networks, and configuration. No ghost containers left behind.
- **Batteries included, but swappable:** Sensible defaults for common stacks, with escape hatches for custom configurations.

### 1.3 Target Audience
- Indie developers shipping side projects and SaaS products
- Small agencies managing 5–30 client applications on one powerful server
- Technical founders who want Heroku-like convenience without Heroku pricing
- Self-hosting enthusiasts deploying open-source tools (Plausible, N8N, Ghost, etc.)

### 1.4 Success Criteria
- One-command installation on a fresh Ubuntu/Debian server (< 5 minutes)
- First application deployed from GitHub within 10 minutes of installation
- Zero-downtime deployments with automatic rollback capability
- Built-in database and cache provisioning without manual Docker commands
- Automatic SSL certificate provisioning and renewal

---

## 2. Functional Requirements

### 2.1 Application Management

#### 2.1.1 App Creation & Configuration
| ID | Requirement | Priority |
|----|-------------|----------|
| AM-01 | Create an app from a Git repository (GitHub, GitLab, Bitbucket, Gitea) | P0 |
| AM-02 | Create an app from local file upload (ZIP/tar archive) | P1 |
| AM-03 | Create an app from a Docker Compose file or Dockerfile | P1 |
| AM-04 | Support multiple build strategies: Buildpacks (auto-detect), Dockerfile, Nixpacks, static sites | P0 |
| AM-05 | Configure app name, primary domain, and additional aliases | P0 |
| AM-06 | Environment variable management: key-value pairs, secrets masking, bulk import (.env file) | P0 |
| AM-07 | Persistent storage volumes: create, mount to specific container paths, backup | P1 |
| AM-08 | Health check configuration: HTTP endpoint, interval, timeout, failure threshold | P1 |
| AM-09 | Resource limits per app: CPU cores, RAM (soft/hard), disk I/O | P2 |
| AM-10 | Container restart policy: always, on-failure, unless-stopped | P1 |

#### 2.1.2 Deployment & Lifecycle
| ID | Requirement | Priority |
|----|-------------|----------|
| DL-01 | Git push-to-deploy via webhooks (auto-deploy on push to configured branch) | P0 |
| DL-02 | Manual deploy button with branch/tag/commit selection | P0 |
| DL-03 | Zero-downtime deployment using blue/green or rolling strategy | P0 |
| DL-04 | One-click rollback to any previous deployment | P0 |
| DL-05 | Pre-deploy and post-deploy hooks (custom scripts) | P1 |
| DL-06 | Deployment history with commit SHA, timestamp, status, and deployer | P1 |
| DL-07 | Build logs streaming in real-time during deployment | P0 |
| DL-08 | Cancel in-progress deployment | P1 |

#### 2.1.3 Domain & Networking
| ID | Requirement | Priority |
|----|-------------|----------|
| DN-01 | Attach custom domains to apps with automatic DNS validation guidance | P0 |
| DN-02 | Automatic Let's Encrypt SSL certificate provisioning and renewal | P0 |
| DN-03 | Wildcard certificate support (*.example.com) | P1 |
| DN-04 | Force HTTPS redirect configuration | P0 |
| DN-05 | Path-based routing (/api → app1, / → app2) | P1 |
| DN-06 | Basic authentication and IP whitelist per app | P1 |
| DN-07 | Custom Caddy configuration snippets (advanced users) | P2 |
| DN-08 | WebSocket support for real-time logs and events through reverse proxy | P0 |

### 2.2 Service & Database Management

#### 2.2.1 Managed Services
| ID | Requirement | Priority |
|----|-------------|----------|
| DB-01 | One-click provision: PostgreSQL, MySQL, MariaDB, MongoDB, Redis, Memcached | P0 |
| DB-02 | Connection string generation with copy-to-clipboard | P0 |
| DB-03 | Web-based admin interfaces (phpMyAdmin alternative, pgAdmin-lite, Redis Commander) | P1 |
| DB-04 | Automated backups: scheduled (daily/weekly), manual trigger, retention policy | P0 |
| DB-05 | Backup destinations: local storage, S3-compatible (AWS, MinIO, Wasabi, Backblaze), SFTP | P0 |
| DB-06 | One-click restore from backup (overwrite or create new instance) | P0 |
| DB-07 | Database user management and access controls | P1 |
| DB-08 | Resource limits per service (CPU, RAM) | P2 |
| DB-09 | Service logs and metrics (queries/sec, connections, memory) | P1 |

#### 2.2.2 One-Click App Templates
| ID | Requirement | Priority |
|----|-------------|----------|
| TP-01 | Pre-configured templates: WordPress, Ghost, Plausible, N8N, Cal.com, Nextcloud, Vaultwarden | P1 |
| TP-02 | Template marketplace / registry with version pinning | P2 |
| TP-03 | Custom template creation and import (Docker Compose-based) | P2 |

### 2.3 Server Management

#### 2.3.1 System Overview
| ID | Requirement | Priority |
|----|-------------|----------|
| SM-01 | Real-time server metrics: CPU usage (per core), RAM usage, disk I/O, network I/O | P0 |
| SM-02 | Per-app resource consumption breakdown | P0 |
| SM-03 | Disk usage by app, service, and system | P1 |
| SM-04 | Uptime monitoring and alerts | P1 |
| SM-05 | Historical metrics retention (7 days minimum, 30 days preferred) | P1 |

#### 2.3.2 Log Management
| ID | Requirement | Priority |
|----|-------------|----------|
| LG-01 | Real-time log streaming for apps and services (WebSocket-based) | P0 |
| LG-02 | Search and filter logs by time range, keyword, severity | P1 |
| LG-03 | Log retention policy and archival | P2 |
| LG-04 | Caddy access logs with request analytics | P1 |
| LG-05 | Error log aggregation and alerting | P2 |

#### 2.3.3 Maintenance Tools
| ID | Requirement | Priority |
|----|-------------|----------|
| MT-01 | Web-based terminal (ttyd backend + xterm.js frontend) for host and containers | P0 |
| MT-02 | File manager: upload, download, edit, archive extraction, permissions | P1 |
| MT-03 | Cron job scheduler: UI-based creation, logs, failure notifications | P1 |
| MT-04 | Firewall management (UFW): port open/close, IP allow/block, status | P1 |
| MT-05 | OS package update notifications and one-click security updates | P1 |
| MT-06 | Docker image and volume cleanup (prune unused resources) | P1 |

### 2.4 User & Team Management

| ID | Requirement | Priority |
|----|-------------|----------|
| UM-01 | Single admin user (owner) with full control | P0 |
| UM-02 | Team member invitation via email with role assignment | P1 |
| UM-03 | Role-based access control: Owner, Admin, Developer, Viewer | P1 |
| UM-04 | Activity audit log: who did what, when | P1 |
| UM-05 | API key generation with scoped permissions | P1 |
| UM-06 | Two-factor authentication (TOTP) | P1 |
| UM-07 | Session management and remote logout | P1 |

---

## 3. Non-Functional Requirements

### 3.1 Performance
- **Panel UI load time:** < 2 seconds for dashboard on a 100ms latency connection
- **Deployment trigger latency:** < 5 seconds from git push to build start
- **Log streaming latency:** < 500ms from container stdout to browser
- **Concurrent app support:** Minimum 50 apps/services on a 4 CPU / 8GB RAM server
- **Agent memory footprint:** < 100MB RAM on idle

### 3.2 Reliability
- **Panel availability:** Panel daemon must restart automatically on crash (systemd service)
- **Deployment atomicity:** Failed deployments must not leave the app in a broken state
- **SSL renewal:** Must retry failed renewals and alert user 7 days before expiry
- **Backup integrity:** Verify backups via test restore on schedule (monthly)

### 3.3 Security
- **Secrets encryption:** All environment variables and database credentials encrypted at rest (AES-256 or equivalent)
- **Communication:** All panel communication over HTTPS/WSS. No HTTP fallback in production.
- **Container isolation:** Apps run in unprivileged containers with no host root access by default
- **Sandboxing:** File manager and web terminal run in sandboxed sessions with timeout
- **Input validation:** Strict validation on all API endpoints, file uploads, and domain inputs
- **Dependency scanning:** Automated CVE scanning for Docker base images (weekly)

### 3.4 Scalability (Within Single Server)
- **Horizontal app scaling:** Support multiple container instances per app with round-robin load balancing (P2)
- **Vertical scaling:** Hot-adjust resource limits without full container restart where possible
- **Storage scaling:** Support external storage mounts (NFS, S3 FUSE) for app data (P2)

### 3.5 Compatibility
- **Supported OS:** Ubuntu 22.04/24.04 LTS, Debian 12, AlmaLinux 9 (P0). Ubuntu/Debian primary.
- **Architecture:** AMD64 (P0), ARM64 (P1)
- **Browser support:** Chrome, Firefox, Safari, Edge (last 2 versions)
- **Docker version:** Compatible with Docker Engine 24.0+ and Docker Compose v2

---

## 4. Architecture & Technical Design

### 4.1 System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      User Browser                            │
│         (HTTPS / WebSocket for logs & events)               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Reverse Proxy Layer                       │
│              (Caddy or Nginx - Host Port 80/443)            │
│         • Routes panel.domain.com → Panel UI               │
│         • Routes app1.com, app2.com → App Containers       │
│         • Automatic SSL termination                        │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌──────────────┐    ┌─────────────────┐    ┌──────────────┐
│  Panel UI    │    │   Panel API     │    │ Panel Agent  │
│  (Next.js)   │◄──►│   (REST + WS)   │◄──►│  (Go/Node)   │
│  Static SPA  │    │   (Go/Node)     │    │  Daemon      │
└──────────────┘    └─────────────────┘    └──────────────┘
                                                  │
                    ┌─────────────────────────────┼─────────────┐
                    ▼                             ▼             ▼
           ┌──────────────┐           ┌─────────────────┐  ┌──────────┐
           │  Panel DB    │           │ Docker Engine   │  │  System  │
           │ (SQLite/PG)  │           │ + Compose       │  │  (UFW,   │
           │  • Apps      │           │  • App Containers│  │  Cron,   │
           │  • Services  │           │  • DB Containers│  │  Logs)   │
           │  • Users     │           │  • Networks     │  │          │
           │  • Deploys   │           │  • Volumes      │  │          │
           └──────────────┘           └─────────────────┘  └──────────┘
```

### 4.2 Component Breakdown

#### 4.2.1 Panel Agent (Host Daemon)
- **Language:** Go 1.22+
- **Framework:** Standard library + `gorilla/websocket` for event streaming
- **Responsibilities:**
  - Execute Docker commands via Docker Engine API
  - Manage filesystem operations (volumes, backups, file manager)
  - Monitor system resources (cgroups v2, disk, network)
  - Run scheduled tasks (cron, SSL renewal, backup jobs)
  - Stream logs and events to Panel API via Unix domain socket
- **Communication:** Unix domain socket at `/var/run/panel/agent.sock` (never exposed externally)
- **Security:** Runs as `panel` user with `docker` group membership; passwordless sudo for specific firewall/package operations via whitelist

#### 4.2.2 Panel API
- **Language:** Go 1.22+
- **Framework:** Gin v1.9+ (HTTP router + middleware)
- **Database:** SQLite via `modernc.org/sqlite` (libsql compatible). Single file at `/var/panel/panel.db`. No PostgreSQL required.
- **Responsibilities:**
  - RESTful API for all CRUD operations
  - WebSocket hub for real-time events (deploy status, logs, metrics) via `gorilla/websocket`
  - Git webhook receiver (GitHub/GitLab/Bitbucket/Gitea push events)
  - Business logic and validation
  - Authentication & authorization (JWT-based via `golang-jwt/jwt/v5`)
- **Communication:** Binds to `127.0.0.1:2053`. Caddy reverse-proxies to this port. Never exposed directly.

#### 4.2.3 Panel UI (Dashboard)
- **Framework:** Next.js 14 (App Router)
- **Language:** TypeScript 5.3+
- **Styling:** Tailwind CSS 3.4+
- **Component Library:** shadcn/ui (built on Radix UI primitives)
- **State Management:** Zustand (global state) + React Query (server state)
- **Key UI Patterns:**
  - Dark mode default (developer preference)
  - Command palette (Cmd+K) for quick navigation
  - Real-time indicators for deployment status, health checks
  - Terminal emulator (xterm.js) embedded in browser
  - Log viewer with virtual scrolling for large files
- **Build Output:** Static export (`output: 'export'`) served as static files by Caddy. No Node.js runtime required for UI in production.

#### 4.2.4 Reverse Proxy (Edge Router)
- **Server:** Caddy 2.7+
- **Rationale:** Automatic HTTPS (no certbot), dynamic config reloading via JSON API, native WebSocket support, single static binary
- **Configuration:** Caddy reads from `/etc/panel/Caddyfile` which the Panel API regenerates on app/domain changes
- **Responsibilities:**
  - Route panel UI traffic (`panel.example.com` → static files)
  - Route Panel API traffic (`panel.example.com/api` → `127.0.0.1:2053`)
  - Route application traffic by domain (`app.example.com` → `localhost:APP_PORT`)
  - Handle SSL termination and automatic Let's Encrypt renewal
  - Enforce basic auth / IP whitelist per app (via Caddy directives)

### 4.3 Data Model (Core Entities)

```
Server
├── Apps[]
│   ├── Deployments[]
│   ├── EnvironmentVariables[]
│   ├── Domains[]
│   ├── Volumes[]
│   └── HealthChecks[]
├── Services[] (Databases, Caches, etc.)
│   ├── Backups[]
│   └── Connections[]
├── Users[]
│   ├── APIKeys[]
│   └── Sessions[]
├── Certificates[] (SSL)
├── CronJobs[]
└── FirewallRules[]
```

### 4.4 API Design (REST + WebSocket)

#### REST Endpoints (v1)
```
GET    /api/v1/apps                    # List all apps
POST   /api/v1/apps                    # Create new app
GET    /api/v1/apps/:id                # Get app details
PUT    /api/v1/apps/:id                # Update app config
DELETE /api/v1/apps/:id                # Delete app and resources
POST   /api/v1/apps/:id/deploy         # Trigger manual deploy
GET    /api/v1/apps/:id/deployments    # List deployment history
POST   /api/v1/apps/:id/rollback       # Rollback to previous deploy
GET    /api/v1/apps/:id/logs           # Stream logs (SSE or upgrade to WS)

GET    /api/v1/services                # List managed services
POST   /api/v1/services                # Create service (DB, cache)
GET    /api/v1/services/:id/backups   # List backups
POST   /api/v1/services/:id/backups   # Trigger backup
POST   /api/v1/services/:id/restore   # Restore from backup

GET    /api/v1/server/metrics         # Current server metrics
GET    /api/v1/server/logs            # System logs
GET    /api/v1/server/updates         # Available OS updates
POST   /api/v1/server/update          # Apply OS updates

WS     /api/v1/stream                 # Real-time events hub
```

#### WebSocket Events
```
app.deploy.started    { appId, deploymentId, commitSha }
app.deploy.progress   { appId, deploymentId, step, message }
app.deploy.success    { appId, deploymentId, timestamp }
app.deploy.failed     { appId, deploymentId, error, logs }
app.logs              { appId, timestamp, stream, message }
service.metrics       { serviceId, cpu, memory, connections }
server.metrics        { cpu, memory, disk, network }
```

---

## 5. User Experience & Interface

### 5.1 Primary User Flows

#### Flow 1: First-Time Setup (5-minute goal)
1. User runs install command on fresh VPS
2. Script installs Docker, Panel Agent, Panel API, Caddy, and Panel UI
3. Browser opens to `https://panel.<server-ip>.sslip.io` (or user-configured domain)
4. User creates admin account (username, password, 2FA setup)
5. Dashboard shows "Create Your First App" CTA

#### Flow 2: Deploy from GitHub (10-minute goal)
1. User clicks "New App" → selects "Git Repository"
2. Authenticates with GitHub (OAuth) or provides repository URL
3. Selects repository, branch, and build strategy (auto-detected)
4. Configures environment variables (optional)
5. Attaches domain (optional, can use generated subdomain)
6. Clicks "Deploy"
7. Real-time build logs stream in UI
8. App goes live with SSL certificate

#### Flow 3: Add a Database
1. User clicks "New Service" → selects PostgreSQL
2. Sets name, version, and root password
3. Service provisions in < 30 seconds
4. Connection string displayed with copy button
5. User pastes connection string into app environment variables

### 5.2 Dashboard Layout

```
┌────────────────────────────────────────────────────────────┐
│  Logo    Dashboard   Apps   Services   Server   Settings     │  ← Top Nav
├────────────────────────────────────────────────────────────┤
│                                                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Server CPU   │  │ Server RAM   │  │ Disk Usage   │     │  ← Overview Cards
│  │   34%        │  │   62%        │  │   45%        │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
│                                                            │
│  ┌────────────────────────────────────────────────────┐   │
│  │ Recent Activity                                     │   │
│  │ • App "api-prod" deployed successfully (2 min ago) │   │
│  │ • Database "main-pg" backup completed (1 hr ago)   │   │
│  │ • SSL certificate renewed for example.com          │   │
│  └────────────────────────────────────────────────────┘   │
│                                                            │
│  ┌──────────────────────┐  ┌──────────────────────────┐  │
│  │ Running Apps (6)      │  │ Active Services (4)      │  │
│  │ • api-prod    ● live  │  │ • main-pg      ● healthy │  │
│  │ • web-client  ● live  │  │ • redis-cache  ● healthy │  │
│  │ • worker      ● live  │  │ • minio        ● healthy │  │
│  │ • ...                 │  │ • ...                    │  │
│  └──────────────────────┘  └──────────────────────────┘  │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 5.3 Design Principles
- **Information density:** Developers want data, not whitespace. Compact tables, monospace fonts for logs/code.
- **Progressive disclosure:** Simple defaults upfront, advanced settings behind "Advanced" toggles.
- **Status at a glance:** Color-coded health indicators (green = healthy, yellow = deploying, red = failed).
- **Keyboard shortcuts:** `Cmd+K` command palette, `Esc` to close modals, `Shift+D` deploy current app.

---

## 6. Security Specification

### 6.1 Authentication & Authorization
- **Primary auth:** Username/password with bcrypt hashing (cost factor 12+)
- **2FA:** TOTP (RFC 6238) via QR code setup, enforced optionally
- **Session management:** JWT access tokens (15-min expiry) + HTTP-only refresh tokens (7-day expiry)
- **API keys:** HMAC-SHA256 signed keys with granular scopes (read, deploy, manage)
- **Rate limiting:** 100 requests/minute per IP, 20 failed login attempts/hour before lockout

### 6.2 Network Security
- **Panel exposure:** Panel UI/API accessible only via HTTPS. Bind to localhost + proxy through Caddy.
- **Firewall defaults:** UFW enabled, ports 22 (SSH), 80 (HTTP), 443 (HTTPS) open. All other ports closed.
- **App isolation:** Each app in its own Docker network. Inter-app communication only via explicit service links.
- **Container hardening:** Run as non-root user, read-only root filesystem where possible, drop unnecessary capabilities.

### 6.3 Data Security
- **Secrets storage:** Environment variables encrypted at rest using AES-256-GCM with a key derived from a server-specific master key (stored in `/etc/panel/keys` with 600 permissions).
- **Backup encryption:** Optional GPG encryption for remote backups before upload.
- **File manager sandbox:** Chrooted to `/var/panel/apps/{appId}/` or container filesystem. No host root access.
- **Terminal sandbox:** Web terminal connects to specific containers or a restricted host user, never root directly.

### 6.4 Audit & Compliance
- **Audit log:** Immutable log of all administrative actions (user, action, resource, timestamp, IP, user-agent).
- **Log retention:** 30 days for application logs, 90 days for audit logs, 7 days for system metrics.
- **Vulnerability scanning:** Weekly scan of panel dependencies and Docker base images. Alert on critical CVEs.

---

## 7. Installation & Deployment

### 7.1 Installation Script
```bash
# One-line installer (target experience)
curl -fsSL https://get.panel.dev | bash

# Or with options
curl -fsSL https://get.panel.dev | bash -s --   --domain panel.example.com   --email admin@example.com   --database sqlite
```

### 7.2 Installation Process
1. **System check:** Verify OS, architecture, kernel version, available disk space (min 20GB), RAM (min 2GB).
2. **Dependency installation:** Docker Engine, Docker Compose plugin, Caddy, Git, jq.
3. **User creation:** Create `panel` system user, add to `docker` group.
4. **Agent installation:** Download latest agent binary to `/usr/local/bin/panel-agent`, create systemd service.
5. **API installation:** Download API binary to `/usr/local/bin/panel-api`, create systemd service.
6. **UI deployment:** Pull latest UI Docker image or static build, serve via Caddy.
7. **Database initialization:** Create SQLite database at `/var/panel/panel.db` or initialize PostgreSQL.
8. **SSL setup:** Generate initial self-signed cert, replace with Let's Encrypt once domain is configured.
9. **First-boot wizard:** Open browser to complete admin account creation.

### 7.3 Update Mechanism
- **Automatic updates:** Optional nightly check for panel updates. Agent and API update via binary replacement with systemd restart. UI update via Docker image pull.
- **Rollback:** Keep previous binary version for 7 days. One-command rollback if update fails.
- **Breaking changes:** Major version updates require manual migration with guided CLI wizard.

### 7.4 Uninstallation

The panel maintains a complete manifest of every resource it created. Uninstallation is deterministic and reversible.

#### Export-First Flow (Default)
```bash
panel uninstall
```

The panel will:
1. **Export** every app and managed service to `/var/panel/export/{appId}/` as:
   - `docker-compose.yml` — fully runnable, standard Docker Compose file
   - `.env` — all environment variables and connection strings
   - `volumes/` — symlink or copy of persistent data paths (optional, `--with-data`)
2. **Print** manual restart instructions:
   ```bash
   cd /var/panel/export/api-prod && docker compose up -d
   ```
3. **Prompt** the user:
   - `Keep exports and stop panel` — panel removed, exported apps remain
   - `Purge everything` — remove panel, exports, containers, volumes, networks, and all data

#### Direct Purge (Non-Interactive)
```bash
panel uninstall --purge              # Remove panel + all apps + all data
panel uninstall --purge --keep-data    # Remove panel + all apps, but preserve host volume directories
```

#### What Gets Removed in a Full Purge
- All Docker containers, images (panel-managed), volumes, and networks
- All Caddy vhost configurations
- All SSL certificates obtained via the panel
- All systemd services created by the panel (agent, API, cron jobs)
- All firewall rules added by the panel
- Panel binaries, database, and configuration files
- The `panel` system user and group

**Safety guard:** The purge reads the manifest and only removes resources the panel created. User-installed system packages or custom files outside panel-managed paths are never touched.

---

## 8. Development Roadmap

### Phase 1: MVP — Core Platform (Months 1–3)
**Goal:** Deploy a Node.js app from GitHub with auto-SSL.
- [ ] Agent daemon with Docker integration
- [ ] REST API for apps, services, domains
- [ ] React/Next.js dashboard with app list, create app flow
- [ ] Git webhook receiver and push-to-deploy
- [ ] Caddy reverse proxy with dynamic config
- [ ] SQLite database schema
- [ ] One-command installer for Ubuntu 24.04
- [ ] Basic auth and admin user

### Phase 2: Sellable — Developer Essentials (Months 4–6)
**Goal:** Production-ready for solo developers.
- [ ] Environment variables with secrets encryption
- [ ] Database services: PostgreSQL, MySQL, Redis
- [ ] Backup system (local + S3)
- [ ] Log streaming and file manager
- [ ] Web terminal
- [ ] Deployment history and rollback
- [ ] Cron job scheduler
- [ ] Firewall GUI
- [ ] Server metrics dashboard
- [ ] Documentation and quick-start guides

### Phase 3: Competitive — Team & Advanced (Months 7–9)
**Goal:** Usable by small teams and agencies.
- [ ] Multi-user support with RBAC
- [ ] Team invitations and activity audit log
- [ ] API keys with scoped permissions
- [ ] One-click app templates (WordPress, Ghost, etc.)
- [ ] Pre/post deploy hooks
- [ ] Health checks and auto-restart
- [ ] Resource limits per app
- [ ] 2FA support
- [ ] Custom Caddy snippets
- [ ] Path-based routing

### Phase 4: Premium — Scale & Intelligence (Months 10–12)
**Goal:** Enterprise-grade single-server platform.
- [ ] Multi-app instance scaling (load balancing)
- [ ] Advanced monitoring (request latency, error rates)
- [ ] Alerting (email, Slack, Discord webhooks)
- [ ] Staging environments (clone production)
- [ ] CI/CD pipeline integration (GitHub Actions, GitLab CI)
- [ ] Docker Compose import/export
- [ ] Template marketplace
- [ ] ARM64 support
- [ ] White-labeling options
- [ ] Plugin/extension system

---

## 9. Monetization Strategy

### 9.1 Pricing Tiers

| Tier | Price | Target | Features |
|------|-------|--------|----------|
| **Community** | Free | Hobbyists, OSS | Single server, unlimited apps, core features, community support |
| **Pro** | $15/server/mo | Indie developers, freelancers | + Team members (up to 5), advanced monitoring, priority backups, email support |
| **Business** | $49/server/mo | Small agencies, SaaS founders | + Unlimited team members, audit logs, SSO/SAML, SLA, dedicated support channel |
| **Enterprise** | Custom | Large teams | + Multi-server (future), on-premise licensing, custom integrations, phone support |

### 9.2 License Enforcement (Non-Intrusive)
- **Community:** No license key required. Honest honor system with subtle "Powered by Panel" footer.
- **Pro/Business:** License key validated against license server (weekly check, 7-day grace period offline). No phone-home for usage data.
- **Enterprise:** Self-hosted license server option for air-gapped environments.

### 9.3 Revenue Expansion
- **Cloud marketplace:** One-click deploy on DigitalOcean, Hetzner, AWS Marketplace (referral revenue)
- **Managed backups:** Optional managed backup storage (S3-compatible) with per-GB pricing
- **Premium templates:** Paid one-click templates for complex stacks (e.g., ERPNext, Mastodon)
- **Support contracts:** Annual support packages for Business/Enterprise tiers

---

## 10. Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Docker socket exposure leading to container escape | Critical | Medium | Agent runs as non-root, uses Docker API with scoped permissions, never exposes socket externally |
| SSL certificate expiry causing downtime | High | Low | Automated renewal with 30/7/1-day alerts, auto-retry logic, fallback to self-signed |
| Git provider webhook delivery failure | Medium | Medium | Support manual deploy button, polling fallback, webhook retry logic |
| Database corruption during backup/restore | High | Low | Backup verification checksums, test restore automation, point-in-time recovery where supported |
| Panel update bricks running apps | High | Low | Blue/green update for panel components, automatic rollback on health check failure, keep previous version |
| Resource exhaustion on small VPS | Medium | High | Built-in resource warnings, default conservative limits, clear minimum requirements |
| Competition from Coolify/CapRover | Medium | High | Differentiate on UX polish, security defaults, and support quality; open core model builds trust |

---

## 11. Glossary

| Term | Definition |
|------|------------|
| **App** | A deployed application, typically running in one or more Docker containers |
| **Service** | A managed dependency (database, cache, queue) running as a Docker container |
| **Deployment** | A specific version of an app built and released from a Git commit |
| **Buildpack** | An automated build strategy that detects the app language and builds accordingly |
| **Agent** | The host daemon that executes Docker and system commands on behalf of the panel |
| **Caddy** | A modern reverse proxy and web server with automatic HTTPS |
| **PaaS** | Platform-as-a-Service; abstracts infrastructure so developers focus on code |
| **Blue/Green Deploy** | A zero-downtime strategy where the new version runs alongside the old before switching traffic |

---

## 12. Appendix

### 12.1 Technology Stack — Definitive Choices

| Layer | Technology | Version | Purpose |
|-------|-----------|---------|---------|
| **Agent Daemon** | Go | 1.22+ | Host-level operations: Docker, filesystem, monitoring, scheduled tasks. Compiled to single static binary. |
| **API Server** | Go + Gin | 1.22+ / v1.9+ | REST API, WebSocket hub, webhook receiver, business logic. Compiled to single static binary. |
| **Panel Database** | SQLite (libsql) | 3.45+ | Panel metadata storage. Single file at `/var/panel/panel.db`. Zero configuration. |
| **Database Library** | sqlx | v1.3+ | Structured SQL queries with type-safe scanning. No ORM magic. |
| **Reverse Proxy** | Caddy | 2.7+ | Edge routing, automatic HTTPS, dynamic config reload. Single static binary. |
| **UI Framework** | Next.js | 14 (App Router) | React framework with static export for production. |
| **UI Language** | TypeScript | 5.3+ | Type safety across UI components and API client. |
| **UI Styling** | Tailwind CSS | 3.4+ | Utility-first CSS with custom design tokens. |
| **UI Components** | shadcn/ui | Latest | Accessible, composable components built on Radix UI primitives. |
| **UI State** | Zustand | 4.5+ | Lightweight global state management. |
| **Server State** | React Query (TanStack Query) | 5.0+ | Data fetching, caching, synchronization, and background updates. |
| **Real-time** | WebSocket | Native | Bidirectional streaming for logs, events, and terminal. Backend: `gorilla/websocket`. Frontend: native `WebSocket`. |
| **Terminal** | ttyd | 1.7+ | Web-based terminal sharing via WebSocket. Caddy reverse-proxies to ttyd. |
| **Code Editor** | Monaco Editor | 0.45+ | VS Code core editor for file manager. Package: `@monaco-editor/react`. |
| **Icons** | Lucide React | Latest | Consistent, lightweight SVG icons. |
| **Charts** | Recharts | 2.10+ | Composable React charts for server metrics. |
| **Container Runtime** | Docker Engine + Compose | 24.0+ / v2 | Universal container runtime. Docker Compose v2 plugin for stack orchestration. |
| **Auth** | JWT | golang-jwt/jwt/v5 | Stateless authentication with access + refresh tokens. |
| **Secrets** | AES-256-GCM | Standard library | Environment variable and credential encryption at rest. |
| **Process Manager** | systemd | — | Agent and API run as systemd services with auto-restart. |

### 12.2 Minimum Server Requirements

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 2 cores | 4+ cores |
| RAM | 2 GB | 4+ GB |
| Disk | 20 GB SSD | 50+ GB SSD |
| Network | 100 Mbps | 1 Gbps |
| OS | Ubuntu 24.04 LTS | Ubuntu 24.04 LTS |

### 12.3 Supported Application Runtimes

| Runtime | Buildpack Detection | Dockerfile Support | Notes |
|---------|---------------------|--------------------|-------|
| Node.js | `package.json` | Yes | npm, yarn, pnpm |
| Python | `requirements.txt`, `pyproject.toml` | Yes | pip, poetry, conda |
| Go | `go.mod` | Yes | Static binary or server |
| Ruby | `Gemfile` | Yes | Bundler |
| PHP | `composer.json` | Yes | Laravel, Symfony |
| Java | `pom.xml`, `build.gradle` | Yes | Spring Boot |
| Rust | `Cargo.toml` | Yes | Cargo build |
| Static Sites | `index.html` | Yes | Caddy serve |
| Docker Compose | `docker-compose.yml` | N/A | Direct deployment |

### 12.4 File Structure (Host)

```
/etc/panel/
├── config.yml          # Panel configuration
├── keys/               # Encryption keys (600 permissions)
└── ssl/                # SSL certificates and private keys

/var/panel/
├── panel.db            # SQLite database
├── apps/               # App persistent storage
│   ├── {appId}/
│   │   ├── data/       # User uploads and persistent files
│   │   └── env/        # Environment variable files (encrypted)
├── services/           # Service persistent storage
│   ├── {serviceId}/
│   │   └── data/       # Database files
├── backups/            # Local backup storage
├── logs/               # Panel and system logs
└── tmp/                # Temporary build artifacts

/usr/local/bin/
├── panel-agent         # Agent daemon binary
├── panel-api           # API server binary
└── panel-cli           # CLI management tool

/opt/panel/
└── ui/                 # Static UI build files
```

---

*End of Specification*
