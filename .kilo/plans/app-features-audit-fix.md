# Juvia Panel — Full Codebase Audit & Fix Plan

**Date:** 2026-06-10  
**Scope:** Backend (Go/Gin) + Frontend (Next.js 14) + Agent (Docker)

---

## A. Bug Fixes

### BUG-1: Health check never updates app status in DB → apps stuck at "deploying"
**File:** `backend/internal/agent/health.go:80-125` — `monitor()`  
**Root cause:** HealthChecker has no reference to the database or any callback to update app status; it only restarts containers and logs to stdout.  
**Impact:** After deployment, if the container starts but health check passes, the app stays at "deploying" forever because nothing sets it to "running".

**Fix:**
1. Add a `StatusCallback func(appID, status, healthStatus string)` field to `HealthChecker` struct (health.go:13)
2. In `monitor()`, after successful health check (line 111), call `callback(appID, "running", "healthy")` on the FIRST success after deployment
3. After `restartContainer` failure (line 119), call `callback(appID, "failed", "unhealthy")`
4. In `apps.go:executeDeployment()` (line ~2006), pass a callback to `StartHealthCheck` that calls `h.repo.UpdateAppStatus()`
5. Add a `firstCheck` flag to `healthMonitor` so status is only updated once after deployment

**Complexity:** M  
**Depends on:** nothing  
**Blocks:** BUG-3 (deploy flow completeness)

---

### BUG-2: Port allocation random on every deployment, not reused
**File:** `backend/internal/agent/container.go:66-73` — `allocatePort()`  
**Root cause:** `allocatePort()` finds the first unused port 3000-3999 each time; when redeploying, the old port is released (old container stopped) and a new one is allocated, breaking any DNS/proxy config pointing to the old port.

**Fix:**
1. In `executeDeployment()` (`apps.go:1969-1981`), read the app's previously assigned port from `app.InternalPort` or a new `ExternalPort` DB field
2. Pass `External: previousPort` in `RunParams.Ports` instead of `External: 0`
3. Add `external_port` column to `apps` table (migration)
4. After first successful deployment, persist the assigned external port to the app record
5. On subsequent deployments, reuse that port

**Complexity:** M  
**Depends on:** nothing  
**Blocks:** nothing

---

### BUG-3: Caddy config not regenerated when domain is added via AddDomain()
**File:** `backend/internal/handlers/apps/apps.go:2037-2129` — `AddDomain()`  
**Root cause:** The handler creates the domain DB record but never calls `caddyMgr.SetupAppDomain()` or `caddy.ReloadCaddy()`. The domain is saved but has no proxy route.

**Fix:**
1. After `h.repo.CreateAppDomain()` succeeds (line 2113), look up the app's external port
2. Call `caddyMgr.SetupAppDomain(appID, domain, port, email, forceHTTPS)`
3. Add a `caddyMgr *proxy.CaddyManager` field to `Handler` struct
4. Pass `caddyMgr` from `main.go` when creating the handler

**Complexity:** S  
**Depends on:** nothing  
**Blocks:** nothing

---

### BUG-4: Deployment logs endpoint route mismatch — frontend gets 404
**File:** `frontend/src/lib/api.ts:263` calls `/apps/${appId}/deployments/${deploymentId}/logs`  
**Root cause:** No such route exists in `main.go`. The actual route is `GET /deployments/:id/logs` (main.go:353). The frontend calls a non-existent nested route.

**Fix (two options):**
- **Option A (backend):** Add route `appsGroup.GET("/:id/deployments/:deploymentId/logs", ...)` in `main.go` that delegates to `deploymentsHandler.GetDeploymentLogs`
- **Option B (frontend):** Change api.ts:263 to `/deployments/${deploymentId}/logs`

**Recommended:** Option A (add backend route) — keeps RESTful nesting and avoids changing existing frontend consumers.

**Complexity:** S  
**Depends on:** nothing  
**Blocks:** nothing

---

### BUG-5: Git clone has no SSH key support — private repos fail silently
**File:** `backend/internal/agent/build.go:104-127` — `CloneRepo()`  
**Root cause:** Uses plain `git clone` with no SSH key handling. Private repos require authentication.

**Fix:**
1. Add `SSHKey string` field to `BuildParams` (agent.go:414)
2. In `CloneRepo()`, if SSHKey is non-empty:
   - Write key to temp file with `0600` permissions
   - Set `GIT_SSH_COMMAND=ssh -i /path/to/key -o StrictHostKeyChecking=no`
   - Clean up key file after clone
3. Add SSH key storage to app config (migration + model)
4. Add SSH key field to CreateApp API request

**Complexity:** M  
**Depends on:** nothing  
**Blocks:** nothing

---

### BUG-6: StartApp sets status to "deploying" instead of "starting"
**File:** `backend/internal/handlers/apps/apps.go:974`  
**Root cause:** `StartApp` sets status to "deploying" which is misleading; it should be "starting" since no build occurs.

**Fix:** Change `"deploying"` to `"starting"` at line 974, and update the WebSocket event accordingly.

**Complexity:** S  
**Depends on:** nothing  
**Blocks:** nothing

---

### BUG-7: DeleteApp doesn't actually remove the Docker container
**File:** `backend/internal/handlers/apps/apps.go:800-821` — `DeleteApp()`  
**Root cause:** The handler deletes the DB record but never calls `h.agent.Remove()` to destroy the Docker container. The container stays running as an orphan.

**Fix:**
1. Before `h.repo.DeleteApp()`, get the app's container ID
2. Call `h.agent.Remove(ctx, containerID, true)` to force-remove the container
3. Also stop any health checks: `h.agent.StopHealthCheck(appID)`

**Complexity:** S  
**Depends on:** nothing  
**Blocks:** nothing

---

### BUG-8: RestartApp doesn't wait for container restart to complete
**File:** `backend/internal/handlers/apps/apps.go:864-880`  
**Root cause:** Restart runs in a goroutine but immediately returns "restarting" to the client. If the restart fails, the status is set to "failed" but the client already got 200 OK. Also, the health check isn't re-triggered after restart.

**Fix:**
1. After `h.agent.Restart()` succeeds, start a health check for the container
2. Set status to "running" only after health check passes (or set immediately if no health check configured)

**Complexity:** S  
**Depends on:** BUG-1  
**Blocks:** nothing

---

## B. Missing Implementations

### MISS-1: Upload flow — no file upload UI in app detail page
**File:** `frontend/src/app/apps/[...slug]/AppDetailClient.tsx`  
**What is missing:** The app detail page has no upload source files functionality. After creating an upload-type app, users cannot upload files. The existing `handleFileUpload` (line 537) is for .env import, not source code.

**Implementation:**
1. Add `uploadSource` mutation using `api.apps.uploadSource()`
2. Add file picker (accept `.zip,.tar.gz,.tar`) in the overview tab, visible when `app.source?.type === 'upload'`
3. Show upload progress and trigger deployment after successful upload
4. Add "Last uploaded" info display

**Complexity:** M  
**Depends on:** nothing  
**Blocks:** nothing

---

### MISS-2: Docker Compose source type — no backend support
**File:** `backend/internal/handlers/apps/apps.go:433` — validates `docker_compose` as source type  
**What is missing:** CreateApp accepts `docker_compose` but `executeDeployment` has no logic to handle compose files. The build pipeline treats it like any other app and fails.

**Implementation:**
1. In `executeDeployment()`, detect `sourceType == "docker_compose"`
2. Call `docker compose up -d` instead of the standard build pipeline
3. Store compose file content in a new `compose_config` column or in `source_config`
4. Add compose-specific start/stop/restart logic

**Complexity:** L  
**Depends on:** nothing  
**Blocks:** nothing

---

### MISS-3: Build logs not streamed in real-time during deployment
**File:** `frontend/src/app/apps/[...slug]/AppDetailClient.tsx:417-470` — live logs section  
**What is missing:** The frontend subscribes to `app.logs` WebSocket events for container logs, but build logs during deployment are not streamed. The backend emits `app.deploy.progress` events with step/message, but the frontend doesn't display build log lines in real-time.

**Implementation:**
1. In `executeDeployment()`, emit WebSocket events for each build log line (already partially done via progress events)
2. In AppDetailClient, listen for `app.deploy.progress` events and display them in a live build log panel
3. Add a "Build Output" section that appears during deployment

**Complexity:** M  
**Depends on:** nothing  
**Blocks:** nothing

---

### MISS-4: `getDeploymentLogs` API response format mismatch
**File:** `backend/internal/handlers/deployments/deployments.go:106-141` — `GetDeploymentLogs()`  
**What is missing:** Backend returns `{ deployment_id, lines: [{timestamp, level, message}] }` but frontend expects `{ logs: string }` (api.ts:263). The frontend tries to access `data.logs` which is undefined.

**Fix:**
1. Either change frontend to parse `response.lines` and join them into a string
2. Or change backend to also return a `logs` field with the joined string

**Complexity:** S  
**Depends on:** BUG-4 (route must exist first)  
**Blocks:** nothing

---

### MISS-5: Overview tab doesn't show upload-type app info
**File:** `frontend/src/app/apps/[...slug]/AppDetailClient.tsx:687-709`  
**What is missing:** The overview tab only shows repo URL info when `app.source?.repo_url` is truthy. Upload-type apps show no source info at all.

**Fix:** Add an `app.source?.type === 'upload'` branch that shows:
- Source type badge ("Upload")
- Upload status and file info
- "Upload Source" button

**Complexity:** S  
**Depends on:** MISS-1  
**Blocks:** nothing

---

## C. Enhancements

### ENH-1: Deployment list should show error message for failed deployments
**Current:** Failed deployments show "failed" status but no error detail. Users must click "View logs" (which is broken per BUG-4).  
**Better:** Show the last line of build_logs as a tooltip or inline error message.  
**Why:** Users need to know WHY a deployment failed without extra clicks.

**Complexity:** S  
**Depends on:** BUG-4

---

### ENH-2: Auto-deploy on upload should be optional
**Current:** After uploading files via `apps/new/page.tsx`, the app is auto-deployed.  
**Better:** Add a toggle "Auto-deploy after upload" or let users manually trigger deployment.  
**Why:** Users may want to upload files first and configure env vars before deploying.

**Complexity:** S  
**Depends on:** MISS-1

---

### ENH-3: App status should reflect actual container state
**Current:** App status is set during deployment lifecycle but never reconciled with actual Docker container state. If a container crashes, the app still shows "running".  
**Better:** Periodically check container status via agent and update DB.  
**Why:** Status accuracy is critical for a PaaS dashboard.

**Complexity:** M  
**Depends on:** BUG-1

---

### ENH-4: Add "Redeploy" button for quick re-deployment
**Current:** Users must go to deployments tab → find a successful deployment → click rollback.  
**Better:** Add a "Redeploy" button in the header that re-triggers the last deployment.  
**Why:** Common operation should be one click.

**Complexity:** S  
**Depends on:** nothing

---

## D. New Endpoints Needed

### EP-1: `GET /apps/:id/domains` — List domains for an app
**Handler:** Already exists internally (`GetAppDomainsDetail`) but not exposed as a standalone endpoint.  
**Route registration:** `appsGroup.GET("/:id/domains", ...)` in main.go after line 339  
**Response:** Array of domain objects with SSL status.

**Complexity:** S

---

### EP-2: `POST /apps/:id/redeploy` — Quick redeploy
**Handler:** Reuses existing `TriggerDeployment` logic with `force: true`.  
**Route registration:** `appsGroup.POST("/:id/redeploy", ...)` in main.go  
**Response:** Same as deploy endpoint.

**Complexity:** S

---

### EP-3: `GET /apps/:id/metrics` — App resource metrics
**Handler:** Calls `agent.GetStats(containerID)` and returns CPU/memory/network.  
**Route registration:** `appsGroup.GET("/:id/metrics", ...)` in main.go  
**Response:** `{ cpu_percent, memory_mb, memory_limit_mb, network_rx, network_tx }`

**Complexity:** S

---

### EP-4: `GET /apps/:id/health` — App health status
**Handler:** Calls `agent.GetHealthStatus(appID)` and returns health info.  
**Route registration:** `appsGroup.GET("/:id/health", ...)` in main.go  
**Response:** `{ healthy: bool, fail_count: int, last_check: string }`

**Complexity:** S

---

### EP-5: `PUT /apps/:id/resources` — Update resource limits
**Handler:** Updates CPU/memory limits and restarts container with new limits.  
**Route registration:** `appsGroup.PUT("/:id/resources", ...)` in main.go  
**Request:** `{ cpu_limit: float, memory_limit_mb: int }`

**Complexity:** M

---

## E. Frontend Gaps

### FE-1: No upload UI in app detail page
**Component:** `AppDetailClient.tsx`  
**Missing:** File picker for source upload, upload progress indicator, upload status display.  
**Implementation:** See MISS-1.

---

### FE-2: Deployment logs viewer broken
**Component:** `AppDetailClient.tsx:548-556` — `fetchDeploymentLogs()`  
**Missing:** Calls wrong API endpoint (BUG-4), and response format doesn't match (MISS-4).  
**Implementation:** Fix endpoint URL and parse `lines` array instead of `logs` string.

---

### FE-3: No real-time deployment progress display
**Component:** `AppDetailClient.tsx`  
**Missing:** Build progress is not shown during deployment. The `deployments` query polls every 5s but doesn't show build log output.  
**Implementation:** See MISS-3.

---

### FE-4: No metrics tab in app detail
**Component:** `AppDetailClient.tsx:655` — tabs list  
**Missing:** No "metrics" or "monitoring" tab showing CPU/memory graphs.  
**Implementation:** Add tab, fetch from `GET /apps/:id/metrics`, display with simple charts.

---

### FE-5: No health status display
**Component:** `AppDetailClient.tsx` overview section  
**Missing:** Health status is shown in the header badge but no detailed health info (last check, fail count, uptime).  
**Implementation:** Fetch from `GET /apps/:id/health` and display in overview.

---

## F. Prioritized Execution Order

| # | Task | Est | Depends On | Blocks |
|---|------|-----|------------|--------|
| 1 | **BUG-4:** Add deployment logs route in backend | S | — | FE-2, MISS-4 |
| 2 | **BUG-3:** Wire AddDomain to Caddy | S | — | — |
| 3 | **BUG-7:** DeleteApp removes Docker container | S | — | — |
| 4 | **BUG-6:** StartApp status "starting" not "deploying" | S | — | — |
| 5 | **BUG-1:** Health check updates app status in DB | M | — | BUG-8, ENH-3 |
| 6 | **BUG-2:** Persist and reuse external port | M | — | — |
| 7 | **BUG-5:** SSH key support for private repos | M | — | — |
| 8 | **BUG-8:** RestartApp re-triggers health check | S | BUG-1 | — |
| 9 | **MISS-4:** Fix deployment logs response format | S | BUG-4 | FE-2 |
| 10 | **FE-2:** Fix frontend deployment logs viewer | S | BUG-4, MISS-4 | — |
| 11 | **MISS-1:** Upload UI in app detail page | M | — | MISS-5, FE-1 |
| 12 | **MISS-5:** Overview tab upload-type info | S | MISS-1 | — |
| 13 | **ENH-1:** Show error message for failed deployments | S | BUG-4 | — |
| 14 | **EP-1:** GET /apps/:id/domains endpoint | S | — | — |
| 15 | **EP-3:** GET /apps/:id/metrics endpoint | S | — | FE-4 |
| 16 | **EP-4:** GET /apps/:id/health endpoint | S | — | FE-5 |
| 17 | **EP-2:** POST /apps/:id/redeploy endpoint | S | — | — |
| 18 | **EP-5:** PUT /apps/:id/resources endpoint | M | — | — |
| 19 | **MISS-3:** Real-time build log streaming | M | — | FE-3 |
| 20 | **ENH-3:** Periodic container state reconciliation | M | BUG-1 | — |
| 21 | **ENH-4:** Quick redeploy button | S | EP-2 | — |
| 22 | **MISS-2:** Docker Compose backend support | L | — | — |
| 23 | **FE-4:** Metrics tab in app detail | M | EP-3 | — |
| 24 | **FE-5:** Health status display | S | EP-4 | — |

---

## Implementation Notes

- **Priority 1-4:** Quick wins (S complexity), fix critical data integrity issues
- **Priority 5-8:** Core deployment flow fixes (M complexity), unblocks reliability
- **Priority 9-12:** Frontend fixes that make the UI functional again
- **Priority 13-18:** Missing API endpoints and frontend enhancements
- **Priority 19-24:** Nice-to-have features and hardening

Each task should be implemented, tested locally, and deployed before moving to the next. The agent restart after each backend change requires: `ssh maruf@192.168.0.211 "sudo systemctl restart juvia-api"`.
