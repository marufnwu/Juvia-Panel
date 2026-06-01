# Server Panel - GitHub Actions CI/CD

This repository uses GitHub Actions for automated builds and releases.

## Workflows

### Release Workflow (`.github/workflows/release.yml`)

Triggered when a version tag is pushed (`v*`).

**Builds:**
- `panel-api` - Go API server binary for Linux amd64
- `panel-agent` - Go agent daemon binary for Linux amd64
- `panel-ui.tar.gz` - Static Next.js frontend build
- `panel-migrations.tar.gz` - Database migration SQL files

**Release Artifacts:**
- `panel-api-linux-amd64` - API server binary
- `panel-agent-linux-amd64` - Agent daemon binary  
- `panel-ui.tar.gz` - Frontend static files
- `panel-migrations.tar.gz` - SQL migrations
- `checksums.txt` - SHA-256 checksums for all files

## How to Release

1. Update version in code (if needed)
2. Create and push a version tag:

```bash
# Example: Creating release v1.0.0
git tag v1.0.0
git push origin v1.0.0
```

The workflow will:
1. Build all binaries with Go 1.22
2. Build frontend with Node 20
3. Package migrations
4. Create a GitHub Release with all artifacts

## Manual Testing

To trigger a build without release, push to main branch:

```bash
git push origin main
```

This will run tests but not create a release.

## Local Development Build

```bash
# Backend
cd backend
go build -o panel-api ./cmd/api
go build -o panel-agent ./cmd/agent

# Frontend
cd frontend
npm install
npm run build