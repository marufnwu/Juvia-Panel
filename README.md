# Juvia Panel

**Self-host your apps, databases, and services** — no vendor lock-in, no complex infrastructure to manage.

Juvia Panel is a complete platform for deploying and running web applications on your own server. Connect your Git repository, choose a runtime, and your app is live with automatic HTTPS. Add a PostgreSQL database or Redis cache with one click. Monitor health, schedule backups, and manage everything from a clean web dashboard.

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

## Quick Install

Install Juvia Panel with a single command:

```bash
curl -sSL https://raw.githubusercontent.com/marufnwu/Juvia-Panel/master/scripts/install.sh | sudo bash
```

That's it! The installer will handle everything.

After installation completes, you'll see:
```
Juvia Panel is running at:
  http://YOUR_SERVER_IP:2053
```

Visit that URL and create your first admin account.

---

## The `juvia` CLI

After installation, the `juvia` command is available for managing your panel:

| Command | Description |
|---------|-------------|
| `juvia update` | Update panel to latest version |
| `juvia uninstall` | Remove panel from the system |
| `juvia reset` | Reset panel database (fresh install) |

### Update

```bash
# Check for updates
sudo juvia update check

# Update to latest version
sudo juvia update

# Update to specific version
sudo juvia update --version v1.2.0

# Rollback if something goes wrong
sudo juvia update rollback
```

### Uninstall

```bash
# Standard uninstall (keeps app data)
sudo juvia uninstall

# Full purge (removes everything including data)
sudo juvia uninstall --purge
```

### Reset

If you lost your admin password or want to start fresh:

```bash
# Reset panel (backups database first)
sudo juvia reset

# Reset without backup
sudo juvia reset --yes
```

After reset, visit your panel URL to create a new admin account.

---

## Install Options

```bash
curl -sSL https://raw.githubusercontent.com/marufnwu/Juvia-Panel/master/scripts/install.sh | sudo bash -s -- \
  --domain panel.example.com \
  --email admin@example.com \
  --data-dir /var/panel \
  --config-dir /etc/panel \
  --install-dir /opt/panel \
  --skip-docker \
  --skip-caddy \
  --skip-firewall
```

| Flag | Default | Description |
|------|---------|-------------|
| `--domain` | _(empty)_ | Public domain for the panel |
| `--email` | _(empty)_ | Email for TLS certificate notifications |
| `--data-dir` | `/var/panel` | App data, databases, backups, logs |
| `--config-dir` | `/etc/panel` | Configuration files and secrets |
| `--install-dir` | `/opt/panel` | Binaries and UI static files |
| `--skip-docker` | `false` | Skip Docker CE installation |
| `--skip-caddy` | `false` | Skip Caddy 2 installation |
| `--skip-firewall` | `false` | Skip UFW firewall configuration |

---

## What the Installer Does

1. Checks your system (OS version, RAM, disk, kernel)
2. Installs Docker CE and Caddy 2 (unless skipped)
3. Creates the `juvia` user and directory structure
4. Downloads pre-built binaries from GitHub Releases (or builds from source if unavailable)
5. Generates security keys and configuration
6. Initializes the SQLite database and runs migrations
7. Installs and enables systemd services
8. Configures the firewall (unless skipped)
9. Starts everything and verifies with a health check

---

## Managing Services

Systemd services are used for reliability and auto-restart:

| Service | Description |
|---------|-------------|
| `juvia-agent` | Docker container management |
| `juvia-api` | REST API server |
| `juvia-caddy` | Reverse proxy with automatic TLS |

Manage them with:
```bash
sudo systemctl status juvia-api juvia-caddy
sudo systemctl restart juvia-api
sudo journalctl -u juvia-api -f
```

---

## Managing Apps

### Deploy from Git

1. Go to **Apps → New App**
2. Enter your Git repository URL
3. Select the branch to deploy
4. Choose a build strategy:
   - **Auto** — Nixpacks detects your runtime automatically
   - **Nixpacks** — explicit runtime selection
   - **Dockerfile** — use your own `Dockerfile`
   - **Static** — serve static files from `/public`
5. Add environment variables if needed
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

To invite a team member, go to **Team → Invite** and enter their email address.

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

### `juvia: command not found`

If you installed before the `juvia` CLI was available, download it manually:

```bash
sudo curl -sSL https://raw.githubusercontent.com/marufnwu/Juvia-Panel/master/scripts/juvia -o /usr/local/bin/juvia
sudo chmod +x /usr/local/bin/juvia
```

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
2. View logs: `journalctl -u juvia-api -u juvia-agent`
3. Run with `--debug` flag for verbose output

---

## License

MIT