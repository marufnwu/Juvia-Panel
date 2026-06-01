# Juvia Panel

**Self-host your apps, databases, and services** — no vendor lock-in, no complex infrastructure to manage.

Juvia Panel is a complete platform for deploying and running web applications on your own server. Connect your Git repository, choose a runtime, and your app is live with automatic HTTPS. Add a PostgreSQL database or Redis cache with one click. Monitor health, schedule backups, and manage everything from a clean web dashboard.

---

## What You Get

### One-Click App Deployment
Deploy from any Git repository. Juvia Panel detects your runtime (Node.js, Python, PHP, Go, Ruby, or static files) using Nixpacks and builds a Docker image automatically. Custom Dockerfiles are fully supported.

### Managed Databases & Caches
Spin up PostgreSQL, MySQL, Redis, MongoDB, or MinIO in seconds. No manual Docker Compose, no terminal commands.

### Automatic HTTPS
Caddy 2 handles TLS certificate issuance and renewal — every app you deploy gets a valid certificate automatically.

### Health Monitoring
Every container runs with automatic health checks. If an app crashes, the agent restarts it and alerts you.

### Backups
Manual and scheduled backups for all services. One-click restore from any backup point.

### App Templates
One-click deploy for popular stacks — WordPress, Ghost, Plausible Analytics, Django, Node.js Express, plus database and cache templates.

### Team Management
Invite developers, operators, and viewers. Role-based access: owner, admin, developer, viewer.

### Real-Time Dashboard
Live CPU, RAM, and disk metrics via WebSocket. Watch deployment progress in real time. Full activity log of every change.

### API Access
Generate scoped API keys to automate anything from CI/CD pipelines to custom integrations.

### Browser Terminal
Access container shells directly from the browser with a full xterm.js terminal — no SSH client needed.

---

## Requirements

| | |
|---|---|
| **OS** | Ubuntu 24.04+ or Debian 12+ (amd64) |
| **Kernel** | 6.8 or later |
| **RAM** | 2 GB minimum (4 GB recommended) |
| **Disk** | 20 GB minimum |
| **Network** | Port 80 and 443 open |

---

## Install

Clone the repository and run the install script:

```bash
git clone https://github.com/marufnwu/Juvia-Panel.git
cd Juvia-Panel
sudo bash scripts/install.sh
```

Or install directly from git:

```bash
curl -sSL https://raw.githubusercontent.com/marufnwu/Juvia-Panel/main/scripts/install.sh | sudo bash -s -- --repo-branch main
```

You'll be prompted for:
- **Domain** — the public domain for the panel (e.g. `panel.example.com`)
- **Email** — for TLS certificate notifications

The script will:
1. Check your system (OS version, RAM, disk, kernel)
2. Install Docker CE and Caddy 2
3. Create the `juvia` user and directory structure
4. Clone the repository and build the binaries from source
5. Generate security keys and configuration
6. Initialize the SQLite database and run migrations
7. Install and enable systemd services (agent, API, reverse proxy)
8. Configure the firewall (SSH, HTTP, HTTPS allowed)
9. Start everything and verify with a health check

At the end you'll see the URL where Juvia Panel is running:
```
Juvia Panel is running at:
  https://panel.example.com
```

Visit that URL and create your first admin account.

---

## Updating

```bash
# Check for a new version
sudo panel-update check

# Update to the latest version
sudo panel-update

# Update to a specific version
sudo panel-update --version v1.2.0

# Update with no automatic rollback on failure
sudo panel-update --no-rollback
```

Updates download the new binary, run any database migrations, restart the service, and run smoke tests. If anything fails, the previous version is restored automatically.

---

## Uninstall

```bash
# Standard uninstall (keeps app data and volumes)
sudo panel-uninstall

# Full purge including all app data and volumes
sudo panel-uninstall --purge

# Export docker-compose.yml and .env files before removing
sudo panel-uninstall --export
```

Uninstall reads the installation manifest to cleanly remove everything Panel created — services, users, directories, and firewall rules.

---

## Managing Apps

### Deploy from Git

1. Go to **Apps → New App**
2. Enter your Git repository URL (GitHub, GitLab, or self-hosted)
3. Select the branch to deploy
4. Choose a build strategy:
   - **Auto** — Nixpacks detects your runtime automatically
   - **Nixpacks** — explicit runtime selection
   - **Dockerfile** — your own `Dockerfile` in the repository
   - **Static** — serve static files from `/public`
5. Add environment variables (secrets are encrypted)
6. Click **Deploy**

### App Statuses

| Status | Meaning |
|--------|---------|
| `running` | Container is up and healthy |
| `stopped` | Container was manually stopped |
| `deploying` | Build or deployment in progress |
| `failed` | Last deployment failed |

### Deployment Actions

- **Deploy** — pull latest code and rebuild
- **Restart** — stop and start the container (no rebuild)
- **Stop** — halt the container
- **Logs** — stream live container output
- **Rollback** — revert to a previous successful deployment

---

## Managing Services

### Create a Service

1. Go to **Services → New Service**
2. Choose the type: PostgreSQL, MySQL, Redis, MongoDB, MinIO, or Custom
3. Set the version and resource limits
4. Click **Create**

### Backup & Restore

1. Go to the service detail page
2. Click **Backup** to create a manual snapshot
3. Under **Backups**, click **Restore** on any previous backup

---

## App Templates

Visit **Templates** to browse pre-configured Docker Compose setups:

| Template | Category | Runtimes |
|----------|----------|----------|
| WordPress | CMS | PHP, MySQL |
| Ghost | CMS | Node.js, MySQL |
| Plausible Analytics | Analytics | Elixir, PostgreSQL |
| Node.js Express | Framework | Node.js |
| Python Django | Framework | Python |
| PostgreSQL Database | Database | PostgreSQL |
| Redis Cache | Cache | Redis |

Click **View Compose** to see the Docker Compose file, or copy the URL to use in your own deployment pipeline.

---

## Environment Variables

Environment variables are stored encrypted in the database and injected into your container at runtime.

- **Normal** — visible in the UI and passed to the container
- **Secret** — masked in the UI, encrypted at rest, only passed to the container

To update an app's environment variables, go to **Apps → [app name] → Environment**.

---

## Team Members & Roles

| Role | Deploy | Manage Services | Manage Team | Server Settings |
|------|--------|-----------------|-------------|-----------------|
| Owner | ✓ | ✓ | ✓ | ✓ |
| Admin | ✓ | ✓ | ✓ | ✓ |
| Developer | ✓ | — | — | — |
| Viewer | — | — | — | — |

To invite a team member, go to **Team → Invite** and enter their email address. They'll receive a link to create an account.

---

## API Keys

Go to **Settings → API Keys** to create programmatic access keys.

- Set an expiration date
- Add a description to track usage
- Copy the token immediately — it won't be shown again

Example API call:
```bash
curl -H "Authorization: Bearer sk_live_xxxx" \
  https://panel.example.com/api/v1/apps
```

---

## Monitoring

The **Dashboard** shows:
- Server CPU, RAM, and disk usage (live via WebSocket)
- List of running apps and services with status
- Recent activity across all resources

The **Server** page provides:
- Full disk usage breakdown
- Running processes
- Network I/O statistics
- Container resource usage

---

## Troubleshooting

### App won't deploy
Check the **Logs** tab on the app detail page. Common causes:
- Build command failed (check your `package.json` or `Dockerfile`)
- Environment variable mismatch
- Insufficient memory limit

### Service connection failed
- Verify the service status is `running`
- Check the connection string in **Services → [service] → Connection Info**
- Ensure your app's environment variables point to the correct host/port

### Domain not resolving
- Check your DNS A record points to your server's public IP
- Allow up to 5 minutes for TLS certificate issuance
- Try HTTP first to confirm Caddy is routing requests

### Need help?
1. Check the activity log at **Activity** for error details
2. View logs: `journalctl -u panel-api -u panel-agent`
3. Re-run the install script with `--debug` for verbose output

---

## Frequently Asked Questions

**Can I use my own domain?**
Yes. Any domain you own and control. Just point a DNS A record at your server's public IP.

**Does it support custom ports?**
Yes. You can map any internal port to an external one. By default, Caddy routes `:443` traffic based on the `Host` header to the correct app container.

**Can I run multiple apps on the same server?**
Yes. That's the primary use case. Each app gets its own Docker container on a shared `panel_apps` Docker network, with Caddy routing based on domain.

**What happens if the server restarts?**
Systemd services are configured with `Restart=always`, so the API server, agent, and Caddy all restart automatically after a reboot.

**Is my data backed up automatically?**
You can configure automated backup schedules per service. Manual backups are always available. Backups are stored locally at `/var/panel/backups`.

**Can I migrate my data off?**
Yes. Use `panel-uninstall --export` to generate `docker-compose.yml` and `.env` files for each app, plus copy volume data, before uninstalling.

---

## License

MIT