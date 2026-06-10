# Plan: Docker Compose Backend Support

## Context
- DB schema: `apps.source_type` CHECK constraint already includes `docker_compose`
- Frontend create page: shows a textarea for compose YAML but it's NOT wired to React state
- API `CreateApp`: accepts `docker_compose` as `source.type` but `executeDeployment` has zero compose-specific logic
- Agent: no `docker compose` commands at all

## Goal
End-to-end Docker Compose support: create apps from compose YAML, run `docker compose up`, get logs/stats from any service, stop/remove the compose project.

---

## A. Database Schema

### A-1. New column: `apps.compose_config` (JSON)
- Stores the full compose YAML content
- Stores parsed service list (names, ports, images)
- Migration `000004_compose.up.sql`

### A-2. New column: `apps.compose_project`
- Stores the docker compose project name (e.g., `panel-app_xxx`)
- Used by agent to identify the compose project for `docker compose -p <name>` commands

### A-3. New table: `compose_services`
```sql
CREATE TABLE compose_services (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL,
    service_name TEXT NOT NULL,
    container_id TEXT,
    image TEXT,
    internal_port INTEGER,
    external_port INTEGER,
    status TEXT DEFAULT 'stopped',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);
CREATE INDEX idx_compose_services_app_id ON compose_services(app_id);
```

---

## B. Backend Agent Changes

### B-1. New: `ComposeManager` (`agent/compose.go`)
- `ComposeUp(ctx, params)` → `docker compose -p <project> -f <file> up -d`
- `ComposeDown(ctx, projectName, removeVolumes bool)` → `docker compose -p <project> down`
- `ComposeStop(ctx, projectName)` → `docker compose -p <project> stop`
- `ComposeStart(ctx, projectName)` → `docker compose -p <project> start`
- `ComposeRestart(ctx, projectName)` → `docker compose -p <project> restart`
- `ComposeLogs(ctx, projectName, service, stream, tail)` → `docker compose -p <project> logs <service> --tail=N`
- `ComposePs(ctx, projectName)` → `docker compose -p <project> ps --format json` → parse service list
- `ComposeServices(ctx, projectName)` → list service names
- `ComposeExec(ctx, projectName, service, command)` → exec into a service
- `ComposeStats(ctx, projectName, service)` → per-service stats

### B-2. New command constants
```go
CmdComposeUp    = "compose_up"
CmdComposeDown  = "compose_down"
CmdComposeStop  = "compose_stop"
CmdComposeStart = "compose_start"
CmdComposeLogs  = "compose_logs"
CmdComposePs    = "compose_ps"
CmdComposeExec  = "compose_exec"
```

### B-3. Agent command handler
Add switch cases in `agent.go` for each compose command.

### B-4. Client methods
```go
func (c *Client) ComposeUp(ctx, params) error
func (c *Client) ComposeDown(ctx, projectName string, removeVolumes bool) error
func (c *Client) ComposePs(ctx, projectName string) ([]ComposeServiceInfo, error)
func (c *Client) ComposeLogs(ctx, projectName, service, stream string, tail int) ([]LogLine, error)
```

---

## C. Backend API Changes

### C-1. `models.go` additions
```go
// ComposeConfig stored in apps.compose_config (JSON)
type ComposeConfig struct {
    Version string                       `json:"version,omitempty"`
    Services map[string]ComposeService   `json:"services"`
    Networks map[string]interface{}     `json:"networks,omitempty"`
    Volumes map[string]interface{}      `json:"volumes,omitempty"`
}

type ComposeService struct {
    Image       string `json:"image,omitempty"`
    Build       string `json:"build,omitempty"`
    Ports       []string `json:"ports,omitempty"`
    Environment map[string]string `json:"environment,omitempty"`
    Volumes     []string `json:"volumes,omitempty"`
    DependsOn   []string `json:"depends_on,omitempty"`
    Command     string `json:"command,omitempty"`
    Restart     string `json:"restart,omitempty"`
}
```

### C-2. `CreateApp` flow changes
When `req.Source.Type == "docker_compose"`:
1. Parse the compose YAML from `req.Source.ComposeConfig` (new field)
2. Validate with a simple YAML parser (check for `services`, warn if missing `version`)
3. Store the raw YAML in `app.SourceConfig` AND `app.ComposeConfig` (JSON)
4. Set `app.BuildStrategy = "compose"` (new strategy)
5. Set `app.ComposeProject = "panel-app_" + appID`

### C-3. `executeDeployment` changes
When `app.BuildStrategy == "compose"`:
1. Write the compose YAML to `<dataDir>/apps/<appID>/docker-compose.yml`
2. Write `.env` file with app env vars to `<dataDir>/apps/<appID>/env`
3. Call `h.agent.ComposeUp(ctx, params)` with:
   - Project name: `app.ComposeProject`
   - Compose file path: `<dataDir>/apps/<appID>/docker-compose.yml`
   - Env file: `<dataDir>/apps/<appID>/env`
4. After `ComposeUp`, call `ComposePs` to get container IDs
5. Store each service container ID in `compose_services` table
6. Store the main service container ID in `app.ContainerID` (first service alphabetically)

### C-4. `executeRollback` changes
Same compose logic: write YAML, `ComposeUp`, `ComposePs`, store containers.

### C-5. `StartApp` changes
For compose apps: `ComposeStart` instead of `agent.Start`.

### C-6. `StopApp` changes
For compose apps: `ComposeStop` instead of `agent.Stop`.

### C-7. `RestartApp` changes
For compose apps: `ComposeRestart` instead of `agent.Restart`.

### C-8. `DeleteApp` changes
For compose apps: `ComposeDown` with `removeVolumes=true` before deleting DB records.

### C-9. `GetAppLogs` changes
For compose apps: `ComposeLogs` per service. Add `service` query param.

### C-10. `GetAppMetrics` changes
For compose apps: `ComposeStats` per service. Add `service` query param.

### C-11. `HealthChecker` changes
For compose apps: monitor the main service container (stored in `app.ContainerID`).

### C-12. New API endpoint: `GET /apps/:id/services`
List all services in a compose app from `compose_services` table.

### C-13. New API endpoint: `GET /apps/:id/services/:service/logs`
Get logs for a specific compose service.

### C-14. New API endpoint: `GET /apps/:id/services/:service/stats`
Get metrics for a specific compose service.

### C-15. New API endpoint: `POST /apps/:id/services/:service/exec`
Execute a command in a specific compose service.

### C-16. Repository additions
- `GetComposeServices(ctx, appID)` → `[]ComposeService`
- `CreateComposeService(ctx, svc)` → insert
- `UpdateComposeService(ctx, svc)` → update status/container_id
- `DeleteComposeServices(ctx, appID)` → delete all for app

---

## D. Frontend Changes

### D-1. Create App page (`apps/new/page.tsx`)
- The `docker-compose.yml` textarea is already there but **not wired to state**
- Add `composeConfig: string` to `AppConfig`
- Add `composeConfig` to `initialConfig`
- Wire the textarea: `value={config.composeConfig}` / `onChange={...}`
- In `handleCreate`, when `sourceType === 'docker'`, include `compose_config: config.composeConfig`

### D-2. `api.ts` changes
- Add `compose_config?: string` to `CreateAppInput.source`

### D-3. `AppDetailClient.tsx` changes
- Overview tab: for compose apps, show list of services
- Logs tab: for compose apps, add a service dropdown selector
- Metrics tab: for compose apps, add a service dropdown selector
- Settings tab: for compose apps, add a compose YAML editor

### D-4. Compose file editor
- Add a textarea in the settings tab for compose apps
- `PUT /apps/:id/compose` to update the compose YAML
- Add a `ComposeUpdateRequest` API call

---

## E. Health Checks for Compose

### E-1. Compose health check strategy
- Default: check the first service in the compose file
- Or: check all services and report overall health
- Use `healthcheck` stanza from compose YAML if present
- Store per-service health in `compose_services.health_status`

---

## F. Caddy / Proxy Integration

### F-1. Compose port mapping
- The compose YAML defines `ports` for each service
- Parse `ports` and create Caddy routes for the first service with a port
- Store the mapped port in `compose_services.external_port`
- `AddDomain` for compose apps: use the first service's port

---

## G. Validation & Error Handling

### G-1. Compose YAML validation
- Parse YAML to verify `services` exists
- Check for duplicate service names
- Warn if `version` is missing (not fatal, but deprecated)
- Check for unsupported Docker Compose features

### G-2. Compose up/down error handling
- `ComposeUp` failure: capture stderr, show in build logs
- `ComposeDown` failure: retry with `docker compose -p <name> down -v` then force

---

## H. Execution Order

| # | Task | Complexity | Depends On |
|---|------|-----------|------------|
| 1 | DB migration (`compose_config`, `compose_project`, `compose_services` table) | S | — |
| 2 | Agent: `ComposeManager` + commands + client methods | L | — |
| 3 | API: `ComposeConfig` types, `CreateApp` compose parsing | M | #1 |
| 4 | API: `executeDeployment` compose path | M | #2, #3 |
| 5 | API: Start/Stop/Restart/Delete for compose apps | M | #2 |
| 6 | API: `GetAppLogs` / `GetAppMetrics` / `HealthChecker` for compose | M | #2 |
| 7 | API: `GET /apps/:id/services` endpoint | S | #1 |
| 8 | API: Service-specific logs/stats/exec endpoints | M | #2 |
| 9 | Frontend: Wire compose textarea to state | S | — |
| 10 | Frontend: App detail compose UI (services list, logs, stats) | M | #7 |
| 11 | Frontend: Compose YAML editor in settings | M | #8 |
| 12 | Caddy: compose port mapping | M | #2 |
| 13 | Agent: compose health check | M | #2 |

---

## Notes

- **Compose project name**: `panel-app_<appID>` → `docker compose -p panel-app_app_xxx`
- **Compose file path**: `<dataDir>/apps/<appID>/docker-compose.yml`
- **Env file**: `<dataDir>/apps/<appID>/env` (created from app env vars)
- **Network**: compose file should use `networks: panel_apps` or the agent will create the network
- **Port mapping**: compose YAML `ports` take precedence; agent parses the compose file to extract ports
- **Single container ID**: `app.ContainerID` points to the first compose service (for backward compatibility)
- **Multi-service**: `compose_services` table stores all services and their container IDs

## Files To Modify

| File | Changes |
|------|---------|
| `backend/internal/database/migrations/000004_compose.up.sql` | New migration |
| `backend/internal/database/models.go` | `ComposeConfig`, `ComposeService` types |
| `backend/internal/agent/compose.go` | New file: `ComposeManager` |
| `backend/internal/agent/agent.go` | New commands + handlers + client methods |
| `backend/internal/agent/health.go` | Handle compose apps |
| `backend/internal/handlers/apps/repository.go` | Compose service queries |
| `backend/internal/handlers/apps/apps.go` | Compose deployment, start/stop/restart/delete |
| `backend/cmd/api/main.go` | New routes |
| `frontend/src/app/apps/new/page.tsx` | Wire compose textarea |
| `frontend/src/lib/api.ts` | `compose_config` in CreateAppInput |
| `frontend/src/app/apps/[...slug]/AppDetailClient.tsx` | Compose UI |
| `frontend/src/app/apps/[...slug]/page.tsx` | Compose tab additions |

## Risks

- **Docker Compose CLI**: requires `docker-compose` plugin or `docker compose` CLI. If the server doesn't have it, the agent will fail. Should check at agent startup and report.
- **Complex YAML**: full compose YAML parsing is complex. The plan uses a simple subset for MVP (services, ports, environment, volumes, networks).
- **Multi-service health**: checking which service to health-check is ambiguous. Default to first service.
- **Port mapping**: compose `ports` can be complex (ranges, host-only). Parse the first simple port.
