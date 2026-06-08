# Juvia Panel: App-Related Functions Fix Plan

## Problem

The user reports "app related functions not working." After thorough investigation of the entire codebase, I've identified multiple bugs across the backend that would cause app CRUD, deployment, and related operations to fail.

---

## Issues Found

### Critical (will cause immediate failures)

#### 1. `services.go:143-148` — `ListServices` scan mismatch
**File:** `backend/internal/handlers/services/services.go:143-148`

The query selects `s.*` (20 columns from services table) + `connected_apps_count` (1 computed) = **21 columns**.

But `ServiceRow` embeds `database.Service` (20 fields) + adds `ConnectedAppsCount` + the scan also tries to read `lastBackupAt` (line 147). The `services` table has **no `last_backup_at` column** (confirmed in migration `000001_init.up.sql`).

**Result:** `rows.Scan()` fails with "sql: expected 21 destination arguments in Scan, not 22" or similar column mismatch. The entire services list endpoint is broken.

**Fix:** Remove `lastBackupAt` from the scan and the `ServiceRow` struct. The services table doesn't have this column.

#### 2. `services.go:987` — `strings.Title()` is deprecated and removed in Go 1.26
**File:** `backend/internal/handlers/services/services.go:987`

```go
fmt.Sprintf("Connected to %s %s.", strings.Title(svc.Type), svc.Version)
```

`strings.Title` was deprecated in Go 1.18 and **removed in Go 1.26**. The project uses Go 1.26.1. This will cause a **compile error**.

**Fix:** Replace with `cases.Title(language.English, cases.NoLower).String(svc.Type)` from `golang.org/x/text/cases`, or simpler: just capitalize manually or use `strings.ToUpper(svc.Type[:1]) + svc.Type[1:]`.

#### 3. `services.go:1118-1122` — `ON CONFLICT` on table without unique constraint
**File:** `backend/internal/handlers/services/services.go:1118-1122`

```sql
INSERT INTO service_app_links (service_id, app_id, connection_env_key, created_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(service_id, app_id) DO UPDATE SET connection_env_key = ?
```

The migration creates `service_app_links` with `PRIMARY KEY (service_id, app_id)` (line 270 of migration). This **does** support `ON CONFLICT` in SQLite. However, the `created_at` column is NOT updated on conflict — this means if you reconnect the same app, the `created_at` stays stale. Not a crash bug, but a data integrity issue.

**Fix:** Add `created_at = excluded.created_at` to the `DO UPDATE` clause.

#### 4. `services.go:267-330` — Manual JSON parsing instead of `json.Unmarshal`
**File:** `backend/internal/handlers/services/services.go:267-330`

The `parseCredentials` and `extractJSONField` functions manually parse JSON strings with string operations. This is fragile and error-prone (e.g., doesn't handle escaped quotes, nested objects, or numeric values correctly). The `encoding/json` package is already imported in `models.go`.

**Fix:** Use `json.Unmarshal` to parse the credentials JSON string.

### High (will cause deployment failures)

#### 5. `apps.go:1807-1948` — `executeDeployment` doesn't set `started_at`
**File:** `backend/internal/handlers/apps/apps.go:1807`

The deployment is created with status `"queued"`, then `executeDeployment` calls `UpdateDeploymentStatus(ctx, deploymentID, "in_progress")` which sets `started_at`. However, this happens **before** the async goroutine starts (line 1797), so the status is updated to `in_progress` but the goroutine may not have started yet. If the goroutine fails immediately, the deployment shows "in_progress" with a `started_at` but no actual work done.

**Fix:** Move the `in_progress` status update to the **first line inside** the goroutine, not before it.

#### 6. `apps.go:1828` — `*deployment.Branch` could be nil
**File:** `backend/internal/handlers/apps/apps.go:1828`

```go
Branch: *deployment.Branch,
```

`deployment.Branch` is a `*string`. If it's nil (which it is when created at line 1771 with `Branch: strPtr(branch)` — wait, that's actually set). But in `TriggerDeployment` at line 1752-1758, if `req.Branch` is empty and `sourceConfig.Branch` is also empty, `branch` is `""` and `strPtr("")` gives a pointer to empty string, not nil. So this is actually safe. But it's fragile.

**Fix:** Add a nil check: `Branch: derefOrEmpty(deployment.Branch)`.

#### 7. `build.go:116` — `git clone` discards stdout/stderr
**File:** `backend/internal/agent/build.go:116-122`

```go
cmd.Stdout = nil
cmd.Stderr = nil
```

When git clone fails, the error message is lost. The caller only gets "git clone failed: exit status 128" with no details about why (auth failure, repo not found, network error, etc.).

**Fix:** Capture stderr and include it in the error message.

#### 8. `build.go:254-278` — `nixpacks` build doesn't capture error output properly
**File:** `backend/internal/agent/build.go:254-278`

The nixpacks build uses `cmd.StdoutPipe()` and `cmd.StderrPipe()` with goroutines to read output. But `cmd.Wait()` returns the error, and the goroutines may not have finished reading by then. The error message from nixpacks may be incomplete.

**Fix:** Wait for goroutines to finish before checking `cmd.Wait()` error, or use `cmd.CombinedOutput()` for simplicity.

#### 9. `container.go:98` — Port mapping uses `params.Ports[0]` without bounds check
**File:** `backend/internal/agent/container.go:98`

```go
internalPort := params.Ports[0].Internal
```

If `params.Ports` is empty, this panics. The `CreateAndStart` function at line 54 checks `params.Ports[0].External` at line 62, but doesn't verify the slice has elements first.

**Fix:** Add a bounds check: `if len(params.Ports) == 0 { return nil, fmt.Errorf("no ports specified") }`.

#### 10. `health.go:129` — Health check uses `localhost` instead of container IP
**File:** `backend/internal/agent/health.go:129`

```go
url := fmt.Sprintf("http://localhost:%d%s", port, path)
```

The health checker runs inside the **agent process**, which is on the host machine. Docker containers are on the `panel_apps` network. The container's port is mapped to a host port (e.g., 3000-3999), so `localhost:port` should work. But the `port` parameter passed to `StartHealthCheck` is the **internal container port** (e.g., 3000), not the **host-mapped port**. The actual host port is `runResult.Port` from `CreateAndStart`.

Looking at `apps.go:1926-1934`:
```go
h.agent.StartHealthCheck(ctx, agent.HealthCheckParams{
    Port: runResult.Port,  // This IS the host-mapped port
    ...
})
```

So this is actually correct — `runResult.Port` is the external port. But the health check path defaults to `/` (line 1931), while the app's health check path is configured in `buildConfig.HealthCheck.Path` (usually `/health`). The health check will hit `/` instead of the configured path.

**Fix:** Use the app's configured health check path instead of hardcoding `/`.

### Medium (will cause incorrect behavior)

#### 11. `repository.go:127` — `ListApps` skips loading domains
**File:** `backend/internal/handlers/apps/repository.go:127`

```go
// Get domains for this app - skip for now to avoid timeout, domains can be loaded per-app
domains := []string{}
```

Domains are never loaded in the list response. The frontend shows `app.primary_domain` which comes from the subquery in the SQL (line 39), so that works. But `app.domains` is always empty.

**Fix:** This is intentional for performance. No change needed unless the frontend needs the full domain list.

#### 12. `apps.go:391` — `ValidateGitURL` is synchronous and blocks the request
**File:** `backend/internal/handlers/apps/apps.go:391`

```go
if !ValidateGitURL(req.Source.RepoURL) {
```

`ValidateGitURL` only checks the URL format (regex match). It does NOT make a network request. So this is actually fine — it's fast. The comment in the plan said it might be slow, but looking at the implementation (repository.go:418-435), it's just regex matching.

**No fix needed.**

#### 13. `apps.go:480` — `userID := c.GetInt("user_id")` may be 0
**File:** `backend/internal/handlers/apps/apps.go:480`

The auth middleware (`middleware/auth.go`) sets `user_id` in the context. If the middleware fails or the token is invalid, the request would be rejected before reaching the handler. So `userID` should always be set. But if it's 0 (unlikely), the `created_by` foreign key would fail.

**No fix needed** — the auth middleware ensures this is set.

#### 14. `frontend/src/app/apps/new/page.tsx:151` — App name sanitization may produce empty string
**File:** `frontend/src/app/apps/new/page.tsx:151`

```typescript
name: config.appName.toLowerCase().replace(/[^a-z0-9-]/g, '-'),
```

If the user enters only special characters, the name becomes `-----` or similar. The backend validates with `isValidAppName` which requires at least 1 character and alphanumeric/dash/underscore. `-----` would pass.

**Low priority** — the backend regex validation is sufficient.

#### 15. `services.go:530-556` — Service provisioning goroutine doesn't update `container_id`
**File:** `backend/internal/handlers/services/services.go:530-556`

The provisioning goroutine calls `h.provisioner.ProvisionPostgreSQL(...)` etc., which creates a container. But the service record in the database is never updated with the `container_id`, `container_image`, or actual port from the provisioned container. The service stays with `container_id = NULL` in the database.

**Fix:** After successful provisioning, update the service record with `container_id`, `container_image`, and `status = 'running'`.

#### 16. `services.go:860-909` — `RestartService` doesn't wait for container to be ready
**File:** `backend/internal/handlers/services/services.go:860-909`

The restart goroutine calls `h.provisioner.RestartService()` and immediately sets status to `running`. But the container may take several seconds to become healthy.

**Fix:** Add a health check wait after restart before setting status to `running`.

#### 17. `services.go:912-944` — `GetServiceLogs` returns empty logs
**File:** `backend/internal/handlers/services/services.go:912-944`

```go
// In a real implementation, fetch logs from Docker container
// For now, return empty logs
c.JSON(http.StatusOK, gin.H{
    "service_id":  serviceID,
    "lines":        []interface{}{},
    "total_lines":  0,
})
```

This is a stub. Service logs are never fetched from Docker.

**Fix:** Use `h.provisioner.GetServiceLogs(ctx, serviceID, 100)` to fetch actual logs.

#### 18. `services.go:947-988` — `TestConnection` is a stub
**File:** `backend/internal/handlers/services/services.go:947-988`

```go
// In a real implementation, test the connection using the service type
// For now, simulate success based on status
```

Always returns success if status is "running", without actually testing the connection.

**Fix:** Implement actual connection testing (e.g., TCP dial to the service port, or a simple query like `SELECT 1` for databases).

#### 19. `caddy.go:59` — Caddy admin socket path is hardcoded
**File:** `backend/internal/proxy/caddy.go:59`

```go
builder.WriteString("  admin unix//var/run/panel/caddy-admin.sock\n")
```

The admin socket path is hardcoded. If `/var/run/panel` doesn't exist or has wrong permissions, Caddy will fail to start.

**Fix:** Make the admin socket path configurable via config, or ensure the directory exists before writing the Caddyfile.

#### 20. `caddy.go:206-218` — `ReloadCaddy` uses `caddy` CLI, not API
**File:** `backend/internal/proxy/caddy.go:206-218`

The reload uses `caddy validate` and `caddy reload` CLI commands. This requires the `caddy` binary to be in PATH and the Caddyfile to be valid. If Caddy is running as a systemd service, the CLI reload may not work correctly (it starts a new Caddy process instead of reloading the running one).

**Fix:** Use the Caddy admin API (`POST /load`) to reload configuration instead of CLI commands.

---

## Fix Priority

| Priority | # | Description | File |
|----------|---|-------------|------|
| **P0** | 2 | `strings.Title` removed in Go 1.26 — compile error | services.go:987 |
| **P0** | 1 | `ListServices` scan column mismatch — runtime panic | services.go:143-148 |
| **P1** | 9 | `params.Ports[0]` without bounds check — panic | container.go:98 |
| **P1** | 7 | Git clone error output discarded — debugging impossible | build.go:116 |
| **P1** | 15 | Service provisioning doesn't update `container_id` | services.go:530-556 |
| **P1** | 17 | `GetServiceLogs` returns empty — no logs visible | services.go:912-944 |
| **P1** | 10 | Health check uses wrong path `/` instead of configured | health.go:129 |
| **P2** | 5 | Deployment `started_at` set before goroutine starts | apps.go:1807 |
| **P2** | 8 | Nixpacks build error output may be incomplete | build.go:254-278 |
| **P2** | 3 | `ON CONFLICT` doesn't update `created_at` | services.go:1118 |
| **P2** | 4 | Manual JSON parsing is fragile | services.go:267-330 |
| **P2** | 16 | Service restart doesn't wait for healthy | services.go:860-909 |
| **P2** | 18 | `TestConnection` is a stub | services.go:947-988 |
| **P2** | 19 | Caddy admin socket path hardcoded | caddy.go:59 |
| **P2** | 20 | Caddy reload uses CLI instead of API | caddy.go:206-218 |
| **P3** | 6 | `*deployment.Branch` nil safety | apps.go:1828 |
| **P3** | 11 | `ListApps` skips loading domains (intentional) | repository.go:127 |
| **P3** | 12 | `ValidateGitURL` is fast (no fix needed) | apps.go:391 |
| **P3** | 13 | `userID` is guaranteed by middleware (no fix needed) | apps.go:480 |
| **P3** | 14 | App name sanitization edge case (low risk) | frontend new/page.tsx:151 |

---

## Implementation Plan

### Phase 1: P0 — Fix compile errors and runtime panics

1. **services.go:987** — Replace `strings.Title(svc.Type)` with manual capitalization:
   ```go
   func titleCase(s string) string {
       if len(s) == 0 { return s }
       return strings.ToUpper(s[:1]) + s[1:]
   }
   ```

2. **services.go:143-148** — Remove `lastBackupAt` from the scan:
   - Remove `var lastBackupAt sql.NullTime` declaration
   - Remove `&lastBackupAt` from the `rows.Scan()` call
   - Remove the `if lastBackupAt.Valid` block

3. **container.go:98** — Add bounds check for `params.Ports`:
   ```go
   if len(params.Ports) == 0 {
       return nil, fmt.Errorf("no ports specified")
   }
   ```

### Phase 2: P1 — Fix deployment and service provisioning

4. **build.go:116** — Capture git clone stderr:
   ```go
   var stderr bytes.Buffer
   cmd.Stderr = &stderr
   if err := cmd.Run(); err != nil {
       return "", fmt.Errorf("git clone failed: %w — %s", err, stderr.String())
   }
   ```

5. **services.go:530-556** — Update service record after provisioning:
   ```go
   if provisionErr == nil {
       h.db.ExecContext(context.Background(),
           "UPDATE services SET status = 'running', container_id = ?, container_image = ? WHERE id = ?",
           info.ContainerID, info.ContainerImage, serviceID)
   }
   ```

6. **services.go:912-944** — Implement `GetServiceLogs`:
   ```go
   logs, err := h.provisioner.GetServiceLogs(ctx, serviceID, 100)
   if err != nil {
       // return error or empty logs
   }
   c.JSON(http.StatusOK, gin.H{"service_id": serviceID, "lines": logs})
   ```

7. **health.go:129** — Use configured health check path:
   - Pass the path from `HealthCheckParams` (already available)
   - The caller in `apps.go:1926-1934` should pass `buildConfig.HealthCheck.Path` instead of `/`

### Phase 3: P2 — Fix correctness issues

8. **apps.go:1807** — Move `in_progress` status update inside goroutine
9. **build.go:254-278** — Wait for goroutines before checking error
10. **services.go:1118** — Add `created_at = excluded.created_at` to ON CONFLICT
11. **services.go:267-330** — Use `json.Unmarshal` for credentials parsing
12. **services.go:860-909** — Add health check wait after restart
13. **services.go:947-988** — Implement actual connection testing
14. **caddy.go:59** — Make admin socket path configurable
15. **caddy.go:206-218** — Use Caddy admin API for reload

### Phase 4: P3 — Minor fixes and cleanup

16. **apps.go:1828** — Add nil check for `*deployment.Branch`
17. **frontend** — Add validation for app name edge cases

---

## Files to Modify

- `backend/internal/handlers/services/services.go` — P0 #1, #2; P1 #5, #6; P2 #10, #11, #12, #13
- `backend/internal/agent/container.go` — P0 #3
- `backend/internal/agent/build.go` — P1 #4, #7; P2 #8
- `backend/internal/handlers/apps/apps.go` — P2 #7; P3 #15
- `backend/internal/agent/health.go` — P1 #6
- `backend/internal/proxy/caddy.go` — P2 #13, #14

## Validation

After implementation:
1. `cd backend && go build ./... && go vet ./...` — must pass
2. `go test ./...` — must pass
3. Manual testing:
   - Create an app via API → should succeed
   - Deploy the app → should clone, build, and run
   - List services → should return without error
   - Get service logs → should return actual Docker logs
   - Test service connection → should actually test connectivity
