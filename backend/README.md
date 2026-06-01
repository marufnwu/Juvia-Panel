# Panel API

Server Panel backend API built with Go.

## Requirements

- Go 1.22+
- SQLite via `modernc.org/sqlite`
- Docker (for app deployment)

## Project Structure

```
backend/
├── cmd/api/          # Application entry point
├── internal/
│   ├── config/       # Configuration loading
│   ├── database/     # Database connection and migrations
│   ├── handlers/     # HTTP request handlers
│   ├── middleware/   # Gin middleware
│   └── models/       # Data models
├── migrations/      # SQL migrations
└── pkg/             # Shared utilities
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PANEL_ENV` | Environment mode | `development` |
| `PANEL_DATA_DIR` | Data directory | `/var/panel` |
| `PANEL_CONFIG_DIR` | Config directory | `/etc/panel` |
| `PANEL_API_PORT` | API server port | `2053` |
| `PANEL_DB_PATH` | SQLite database path | `/var/panel/panel.db` |
| `PANEL_JWT_SECRET` | JWT signing secret (required, min 32 chars) | - |
| `PANEL_JWT_EXPIRY` | Access token expiry | `15m` |
| `PANEL_REFRESH_EXPIRY` | Refresh token expiry | `168h` (7 days) |
| `PANEL_MASTER_KEY` | AES-256 encryption key | - |
| `PANEL_DOMAIN` | Panel domain | - |
| `PANEL_LOG_LEVEL` | Log level | `info` |

## Getting Started

1. Create required directories:
   ```bash
   sudo mkdir -p /var/panel /etc/panel
   ```

2. Set environment variables:
   ```bash
   export PANEL_JWT_SECRET="your-secret-at-least-32-characters-long"
   ```

3. Build and run:
   ```bash
   go build -o panel-api ./cmd/api
   ./panel-api
   ```

4. Test health endpoint:
   ```bash
   curl http://localhost:2053/health
   ```

## Development

```bash
# Download dependencies
go mod tidy

# Run tests
go test ./...

# Run with live reload (requires air)
air
```

## API Endpoints

### Health Check
- `GET /health` - Server health status

### Authentication (TODO)
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - Logout

### Users (TODO)
- `GET /api/v1/users/me` - Get current user
- `POST /api/v1/users/invite` - Invite team member

## Database

SQLite database stored at `/var/panel/panel.db`. Migrations are stored in `migrations/` directory.

Run migrations manually:
```bash
sqlite3 /var/panel/panel.db < migrations/000001_init.up.sql
```
