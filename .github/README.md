# Juvia Panel - CI/CD

This repository uses GitHub Actions for automated builds and releases.

## Workflows

### Release Workflow (`.github/workflows/release.yml`)

Triggered when a version tag is pushed (`v*`).

**Matrix builds:** `amd64` and `arm64`

**Per-architecture artifacts:**

| Artifact | Description |
|----------|-------------|
| `juvia-api` | Go API server binary |
| `juvia-agent` | Go agent daemon binary |
| `juvia-cli` | Go CLI debug binary |
| `juvia-ui.tar.gz` | Next.js standalone frontend |
| `juvia-release-{arch}.tar.gz` | Bundle containing migrations, scripts, and Caddyfile |
| `checksums-{arch}.txt` | SHA-256 checksums |

### Test Workflow

Runs on every push:
- `go test ./...` for backend
- `npm run lint` for frontend

---

## How to Release

1. Commit and push all changes to `master`
2. Create and push a version tag:

```bash
git tag v1.1.0
git push origin v1.1.0
```

The workflow will:
1. Build Go binaries for amd64 and arm64 (CGO_ENABLED=0)
2. Build Next.js frontend with Node 20
3. Package migrations, scripts, and Caddyfile into a release bundle
4. Generate SHA-256 checksums
5. Create a GitHub Release with all artifacts

---

## Install / Update from Release

Users install or update by running:

```bash
# Install
curl -sSL https://raw.githubusercontent.com/marufnwu/Juvia-Panel/master/scripts/install.sh | sudo bash

# Update
curl -sSL https://raw.githubusercontent.com/marufnwu/Juvia-Panel/master/scripts/update.sh | sudo bash
```

The scripts download `juvia-release-{arch}.tar.gz` from the latest GitHub Release, extract binaries and UI, and deploy. If no release bundle is found, they fall back to building from source.

---

## Local Development Build

```bash
# Backend
cd backend
CGO_ENABLED=0 go build -ldflags="-s -w" -o juvia-api ./cmd/api/
CGO_ENABLED=0 go build -ldflags="-s -w" -o juvia-agent ./cmd/agent/
CGO_ENABLED=0 go build -ldflags="-s -w" -o juvia-cli ./cmd/debug/

# Frontend
cd frontend
npm install
npm run build
```

The frontend standalone build is at `frontend/.next/standalone/`.

---

## Directory Structure

```
/etc/panel/              # Configuration
├── config.yml           # Main config
├── env                  # Environment secrets
├── jwt-secret           # JWT signing key
├── encryption-key       # AES-256 encryption key
├── keys/master          # Master encryption key
├── caddy/Caddyfile      # Caddy reverse proxy config
└── migrations/          # SQL migration files

/var/panel/              # Data
├── panel.db             # SQLite database
├── apps/                # App source and builds
├── backups/             # Service backups
├── logs/                # Application logs
└── volumes/             # Docker volumes

/opt/panel/              # Installation
├── bin/                 # (reserved)
└── ui/                  # Next.js standalone build

/usr/local/bin/          # Binaries
├── juvia-api
├── juvia-agent
└── juvia-cli
```
