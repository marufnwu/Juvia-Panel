# Plan: Fix App Features — Bugs & Missing Implementations

## Audit Findings Summary

Complete audit of all app-related backend and frontend code reveals **2 critical bugs**, **12 missing implementations**, and **5 code quality issues**. Combined, these prevent core app functionality from working end-to-end.

---

## Critical Bugs (Must Fix First)

### B1. CreateApp does NOT trigger deployment

**File:** `backend/internal/handlers/apps/apps.go:549-572`

The `CreateApp` handler creates a deployment record but **never calls `executeDeployment()`**. The app gets stuck in "deploying" status forever with no container created. The deployment record is created with status "queued" but no goroutine is spawned to actually build and run the container.

Compare with `TriggerDeployment` (line 1804) which correctly calls `go h.executeDeployment(app, deployment, buildConfig)`.

**Fix:** Add these lines after the deployment record creation at line 572:
```go
// Parse build config for deployment
var buildConfig database.BuildConfig
if app.BuildConfig != nil {
    database.ParseJSONField(app.BuildConfig, &buildConfig)
}
// Execute deployment asynchronously
var fullApp database.App
fullApp = *app
go h.executeDeployment(&fullApp, deployment, buildConfig)
```

### B2. Health check path inconsistency

**Files:** `apps.go:498` + `apps.go:1932`

`CreateApp` defaults health check path to `/health` (line 498), but `executeDeployment` defaults to `/` (line 1932). If no health check is configured, the deployment code can't reach the right path.

**Fix:** In `executeDeployment` (line 1932-1935), use the app's stored `HealthCheckPath` instead of only looking at `buildConfig.HealthCheck.Path`:
```go
healthCheckPath := app.HealthCheckPath
if healthCheckPath == "" {
    healthCheckPath = "/health"
}
// Override from build config if provided
if buildConfig.HealthCheck != nil && buildConfig.HealthCheck.Path != "" {
    healthCheckPath = buildConfig.HealthCheck.Path
}
```

---

## Missing Implementations

### M1. Live resource usage from agent

**File:** `backend/internal/handlers/apps/repository.go:169`

`ResourceUsage` is always `nil` in the app list response. Need to add a call to the agent to get live container stats.

**Fix:** In `ListApps` after building each list item, fetch usage from agent:
```go
// After building the item (line 172)
if app.ContainerID != nil {
    stats, err := h.agent.GetContainerStats(ctx, *app.ContainerID)
    if err == nil {
        item.ResourceUsage = &database.ResourceUsage{
            CPUPercent:    stats.CPUPercent,
            MemoryMB:      stats.MemoryMB,
            MemoryLimitMB: stats.MemoryLimitMB,
        }
    }
}
```

This requires the `AppRepository` to have access to the agent client, or `ListApps` to be moved to the Handler level (not repository). The simpler fix is to add a post-processing step in the `Handler.ListApps` method that enriches each item with stats.

Actually, the cleanest approach: move the resource usage enrichment to `Handler.ListApps`. After getting `apps` and `total` from the repository, iterate over the apps and enrich those with container IDs.

### M2. Add Domain endpoint — backend

**Need new routes:** `POST /apps/{id}/domains` and `DELETE /apps/{id}/domains/{domain}`

**New handler in apps.go:**
```go
func (h *Handler) AddDomain(c *gin.Context) { ... }
func (h *Handler) RemoveDomain(c *gin.Context) { ... }
```

**Register routes in main.go:**
```go
appsGroup.POST("/:id/domains", ...func(c *gin.Context) { appsHandler.AddDomain(c) })
appsGroup.DELETE("/:id/domains/:domain", ...func(c *gin.Context) { appsHandler.RemoveDomain(c) })
```

**AddDomain logic:**
- Validate app exists
- Validate domain format
- Check domain not already assigned to another app
- Insert into `app_domains` table
- Reload Caddy to pick up new domain route
- Return created domain

### M3. Wire up frontend "Add Domain" button

**File:** `frontend/src/app/apps/[...slug]/AppDetailClient.tsx:488`

Button has no `onClick` handler. Need to:
- Add state for domain modal
- Create domain input modal component
- Wire up API call to add domain
- Invalidate app query after adding

### M4. Wire up frontend "Add Volume" button

**File:** `frontend/src/app/apps/[...slug]/AppDetailClient.tsx:555`

Same pattern as Add Domain. Need:
- Volume creation modal (container_path input)
- API call to `POST /apps/{id}/volumes`
- Invalidate app query

### M5. Wire up frontend "Connect Service" button

**File:** `frontend/src/app/apps/[...slug]/AppDetailClient.tsx:528`

Need:
- Service selector modal showing available services
- API call (exists: `services.connectAppService`)  
- Invalidate app query

### M6. Wire up frontend "Import .env" button

**File:** `frontend/src/app/apps/[...slug]/AppDetailClient.tsx:696`

Need:
- File upload or textarea input for .env content
- API call to `POST /apps/{id}/env/import`
- Invalidate env vars query

### M7. Implement real-time logs (WebSocket or polling)

**File:** `frontend/src/app/apps/[...slug]/AppDetailClient.tsx:687`

Current: place holder text. Need:
- WebSocket subscription to `app.{appId}.logs`
- Real-time log streaming display
- Fallback polling for non-WebSocket scenarios

The backend already has `GetAppLogs` endpoint and the WebSocket hub supports app logs. The agent's container.go likely has log streaming support.

### M8. Implement app update (edit) UI

**File:** Need new or enhanced UI

Current "Settings" tab only shows read-only info. Need:
- Editable fields for build command, start command, build strategy
- Editable domain management (add/remove domains)
- Save button calling `PUT /apps/{id}`

### M9. Handle `docker_compose` and `upload` source types

**Files:** `backend/internal/handlers/apps/apps.go:302-303`, `backend/internal/agent/build.go`

The source types `upload` and `docker_compose` are accepted by validation but not handled by the build pipeline. Need:
- `upload` type: Accept file upload, extract to build directory, detect runtime
- `docker_compose` type: Use `docker compose up` instead of single-container deployment

### M10. Auto-deploy checkbox actually sets source config

**File:** `frontend/src/app/apps/new/page.tsx:469-472`

The "Auto-deploy on future pushes" checkbox is present but not wired to the API request. Need to pass `auto_deploy` in the source config:
```typescript
source: {
    type: ...,
    repo_url: ...,
    branch: ...,
    auto_deploy: true, // from checkbox
},
```

### M11. Build logs persistence

**File:** `backend/internal/handlers/apps/apps.go:1843`

In `executeDeployment`, after the build completes, the build logs from the agent result should be persisted to the database:
```go
// After successful build
buildLogJSON, _ := json.Marshal(buildResult.BuildLogs)
h.repo.UpdateDeploymentLogs(ctx, deploymentID, string(buildLogJSON), buildResult.Duration)
```

### M12. Volume delete confirmation

**File:** `frontend/src/app/apps/[...slug]/AppDetailClient.tsx` + `backend/handler`

Volumes shown in the overview don't have a delete button. Need to wire up the existing `DeleteVolume` endpoint with a trash icon on each volume item.

---

## Code Quality & Cleanup

### C1. Remove duplicated deployment listing code

`ListDeploymentsByStatus` in `deployments.go:344` duplicates almost identical code from `apps.go:1605`. Remove the deployment handler's version and route all deployment listing through the apps handler.

### C2. Handle RunParams memory defaults

`getMemoryLimitString()` returns empty string for nil/zero. Should default to a reasonable value like "512m" instead of unlimited.

### C3. Secret encryption for imported env vars

In `ImportEnvVars` (apps.go:1291), imported values are stored as-is without encryption even if they match secret patterns. Need to pass through the encryption check.

### C4. Deduplicate deployment execution code

`executeDeployment` and `executeRollback` share ~80% of the same container setup code (env vars, volumes, run params). Extract into a shared helper function.

### C5. Remove deprecated `frontend/out/` from .gitignore

No longer needed since migrating to server mode.

---

## Implementation Order

| Priority | Item | Complexity | Impact |
|----------|------|-----------|--------|
| 1 | B1 — Fix CreateApp deployment trigger | Low | **Critical** — apps stuck in deploying |
| 2 | B2 — Fix health check path | Low | High |
| 3 | M7 — Real-time logs | Medium | High — visible user feature |
| 4 | M1 — Live resource usage | Low | Medium |
| 5 | M3/M4/M5/M6 — Wire up dead frontend buttons | Medium | High — all visible features |
| 6 | M2 — Add Domain backend endpoint | Low | Medium |
| 7 | M8 — App update/edit UI | High | Medium |
| 8 | M9 — Upload/Docker Compose source types | High | Low |
| 9 | M11 — Build logs persistence | Low | Medium |
| 10 | C1-C4 — Code cleanup | Low | Low |
