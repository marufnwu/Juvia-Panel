# Plan: Migrate Frontend from Static Export to Next.js Server Mode

## Goal

Replace the current Next.js `output: 'export'` (static HTML/CSS/JS served by Caddy) with `next start` (Node.js server process proxied by Caddy). This eliminates all dynamic routing issues, enables server-side auth middleware, and removes the need for Caddy rewrite hacks.

---

## Current Architecture (Before)

```
                    ┌─────────────────────────────────────────────┐
                    │              Caddy (:2053)                   │
                    │                                              │
Browser ────────────┤  /_next/*     → file_server /opt/panel/ui/out│
                    │  /api/*       → reverse_proxy :9090          │
                    │  /apps/*      → rewrite /apps/_/index.html   │
                    │  /services/*  → rewrite /services/_/index.html│
                    │  /cron/*      → rewrite /cron/_/index.html   │
                    │  /*           → try_files ... /index.html     │
                    └─────────────────────────────────────────────┘
                                     │                    │
                    ┌────────────────┘                    └──────────────┐
                    ▼                                                    ▼
        ┌─────────────────────────┐                    ┌─────────────────────────┐
        │   Go Backend (:9090)     │                    │   Static UI Files        │
        │   gin-go REST API        │                    │   /opt/panel/ui/out/     │
        │   WebSocket (:9090)      │                    │   (Next.js static export) │
        │   SQLite DB              │                    └─────────────────────────┘
        │   Caddyfile management   │
        └─────────────────────────┘
```

**Problems:**
1. Dynamic routes (`/apps/{id}`, `/services/{id}`, `/cron/{id}`) need explicit Caddy rewrite rules per route
2. Each new dynamic route requires Caddyfile updates in Go code and manual Caddyfile
3. Refreshing a dynamic route page falls back to `/index.html`, losing routing context
4. `try_files` SPA fallback breaks nested client-side routes
5. Cannot use Next.js middleware, server-side redirects, or SSR
6. Images must be unoptimized, increasing bandwidth

---

## Target Architecture (After)

```
                    ┌─────────────────────────────────────────────┐
                    │              Caddy (:2053)                   │
                    │                                              │
Browser ────────────┤  /_next/static/* → file_server (performance) │
                    │  /api/*          → reverse_proxy :9090       │
                    │  /api/v1/stream  → reverse_proxy :9090 (WS)  │
                    │  /health         → reverse_proxy :9090       │
                    │  /*              → reverse_proxy :3000       │
                    └─────────────────────────────────────────────┘
                                     │                    │
                    ┌────────────────┘                    └──────────────┐
                    ▼                                                    ▼
        ┌─────────────────────────┐                    ┌─────────────────────────┐
        │   Go Backend (:9090)     │                    │   Next.js Server (:3000) │
        │   (unchanged)            │                    │   node_modules/.bin/     │
        │                          │                    │   next start -p 3000     │
        └─────────────────────────┘                    │   SSR / dynamic routes   │
                                                       │   Middleware (auth)       │
                                                       │   Built-in image opt      │
                                                       └─────────────────────────┘
```

**Benefits:**
- Zero Caddy rewrite rules for UI routes — everything goes to Next.js
- Native support for dynamic routes on refresh/direct navigation
- Server-side auth middleware to prevent flash of unauthorized content
- Built-in Next.js image optimization
- Future-proof: ISR, server-side redirects, API route BFF pattern all possible
- Caddyfile becomes trivially simple

---

## Implementation Steps

### Phase 1: Frontend Configuration Changes

#### 1.1 Remove Static Export from `next.config.js`

**File:** `frontend/next.config.js`

```js
/** @type {import('next').NextConfig} */
const nextConfig = {
  // Remove: output: 'export'     ← enables server mode
  // Remove: images: { unoptimized: true }  ← re-enable optimization
  trailingSlash: true,

  // Security: Prevent response header leakage
  poweredByHeader: false,

  // Optional: Compress responses
  compress: true,

  // Optional: API rewrite for BFF pattern (future use)
  // async rewrites() {
  //   return [
  //     { source: '/api/v1/:path*', destination: 'http://127.0.0.1:9090/api/v1/:path*' },
  //   ]
  // },
}

module.exports = nextConfig
```

#### 1.2 Add Next.js Middleware for Server-Side Auth Guard

**New file:** `frontend/src/middleware.ts`

```typescript
import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

const PUBLIC_PATHS = ['/login', '/setup', '/_next', '/api', '/favicon.ico']
const STATIC_EXTS = /\.(ico|png|svg|jpg|jpeg|gif|webp|css|js|woff2?|ttf|eot)$/

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  // Allow static files and Next.js internals
  if (STATIC_EXTS.test(pathname) || pathname.startsWith('/_next/')) {
    return NextResponse.next()
  }

  // Allow public paths
  if (PUBLIC_PATHS.some(p => pathname === p || pathname.startsWith(p + '/'))) {
    return NextResponse.next()
  }

  // Read session cookie (set by Go backend on login)
  const sessionToken = request.cookies.get('panel_token')?.value
  const authHeader = request.cookies.get('panel_auth')?.value

  // Redirect to login if not authenticated
  if (!sessionToken && !authHeader) {
    const loginUrl = new URL('/login', request.url)
    loginUrl.searchParams.set('redirect', pathname)
    return NextResponse.redirect(loginUrl)
  }

  // Set auth cookie as header for API calls (future BFF pattern)
  const response = NextResponse.next()
  if (sessionToken) {
    response.headers.set('X-Auth-Token', sessionToken)
  }

  return response
}

export const config = {
  matcher: [
    '/((?!_next|api/v1|health|favicon.ico).*)',
  ],
}
```

> **Decision:** Keep current client-side auth guard as the primary mechanism. The middleware serves as a **_defense-in-depth_ layer** to prevent unauthenticated page loads before JS hydrates, but does NOT block rendering — it issues a soft redirect. The existing `AuthGuard` in `providers.tsx` remains the authoritative auth check.

#### 1.3 Add Environment Configuration Placeholder

**New file:** `frontend/.env.production`

```env
# The API is served on the same origin by Caddy, so relative URLs work.
NEXT_PUBLIC_API_URL=/api/v1
```

> This already matches the fallback in `src/lib/api.ts` ("`/api/v1`"). No behavior change.

#### 1.4 Update `package.json` Scripts

**File:** `frontend/package.json`

```diff
   "scripts": {
     "dev": "next dev",
     "build": "next build",
-    "start": "next start",
+    "start": "next start --port 3000",
     "lint": "next lint"
   },
```

---

### Phase 2: Caddy Configuration Overhaul

#### 2.1 Simplify `backend/config/Caddyfile`

**File:** `backend/config/Caddyfile` (serves as the template/default)

```caddy
{
    admin unix//var/run/panel/caddy-admin.sock
    log {
        level INFO
    }
}

:2053 {
    # WebSocket — must come before the generic /api handler
    handle /api/v1/stream {
        reverse_proxy localhost:9090 {
            header_up X-Real-IP {remote_host}
            header_up X-Forwarded-For {remote_host}
            header_up X-Forwarded-Proto {scheme}
        }
    }

    # API routes — proxy to Go backend
    handle /api* {
        reverse_proxy localhost:9090
    }

    # Health check
    handle /health {
        reverse_proxy localhost:9090
    }

    # Everything else — proxy to Next.js server
    handle {
        reverse_proxy localhost:3000 {
            header_up Host {host}
            header_up X-Real-IP {remote_host}
            header_up X-Forwarded-For {remote_host}
            header_up X-Forwarded-Proto {scheme}
        }
    }
}
```

> **Rationale:** Let Next.js serve `/_next/static/*` bundles. Next.js already sets proper immutable cache headers for these assets. The performance gain from serving them directly via Caddy is negligible for a management panel (<10 concurrent users). This keeps the Caddyfile trivially simple and eliminates `handle_path` path-stripping complexity.

#### 2.2 Update Go's Caddyfile Generator

**File:** `backend/internal/proxy/caddy.go`

Remove the hardcoded rewrite rules and simplify the `writePanelUIBlock` method:

**Remove these sections from `writePanelUIBlock`:**
- The entire `# Rewrite rules for Next.js static export catch-all routes` section (lines 164-185)
- The `try_files` SPA fallback in the final `handle` block (line 199)
- The `root /opt/panel/ui/out` directive (lines 198)
- The custom `file_server` blocks for `/_next/*` and `/static/*` (handlers need to point to `.next/static`)

**Replace with:**
```go
func (c *Caddy) writePanelUIBlock(builder *strings.Builder) {
    port := c.panelUIPort
    builder.WriteString(fmt.Sprintf(":%d {\n", port))

    // WebSocket endpoint (must be before generic /api handler)
    builder.WriteString("    handle /api/v1/stream {\n")
    builder.WriteString("        reverse_proxy localhost:9090 {\n")
    builder.WriteString("            header_up X-Real-IP {remote_host}\n")
    builder.WriteString("            header_up X-Forwarded-For {remote_host}\n")
    builder.WriteString("            header_up X-Forwarded-Proto {scheme}\n")
    builder.WriteString("        }\n")
    builder.WriteString("    }\n\n")

    // API proxy
    builder.WriteString("    handle /api* {\n")
    builder.WriteString("        reverse_proxy localhost:9090\n")
    builder.WriteString("    }\n\n")

    // Health
    builder.WriteString("    handle /health {\n")
    builder.WriteString("        reverse_proxy localhost:9090\n")
    builder.WriteString("    }\n\n")

    // Next.js server (all remaining requests including _next/static)
    builder.WriteString("    handle {\n")
    builder.WriteString("        reverse_proxy localhost:3000 {\n")
    builder.WriteString("            header_up Host {host}\n")
    builder.WriteString("            header_up X-Real-IP {remote_host}\n")
    builder.WriteString("            header_up X-Forwarded-For {remote_host}\n")
    builder.WriteString("            header_up X-Forwarded-Proto {scheme}\n")
    builder.WriteString("        }\n")
    builder.WriteString("    }\n")
    builder.WriteString("}\n\n")
}
```

The CSP and security headers (X-Frame-Options, etc.) can optionally be added back to the final `handle` block if desired, but Caddy applies them to all proxied responses by default. For now, simplicity takes priority.

#### 2.3 Remove Obsolete Caddyfiles

**Files to archive/delete:**
- `Caddyfile` (root) — was never the deployed version; superseded by generated config
- `caddy-config/Caddyfile` — same as root
- `backend/config/Caddyfile.new` — was a simplified version, now obsolete

Keep `backend/config/Caddyfile` as the single source of truth for the default template.

---

### Phase 3: Build & Deployment Changes

#### 3.1 Update `Makefile`

**File:** `Makefile`

```diff
 build-ui:
 	@echo "Building UI..."
-	cd $(FRONTEND_DIR) && npm run build
+	cd $(FRONTEND_DIR) && npm install --legacy-peer-deps && npm run build

 clean:
 	@echo "Cleaning..."
 	@rm -rf $(BUILD_DIR)
 	@rm -rf $(FRONTEND_DIR)/.next
-	@rm -rf $(FRONTEND_DIR)/out

+start-ui:
+	@echo "Starting Next.js server..."
+	cd $(FRONTEND_DIR) && npm run start
```

#### 3.2 Update Install Script

**File:** `scripts/install.sh`

Key changes needed (around lines 270-305):

**a) Install Node.js permanently (not just for build):**
```bash
# Node.js is now a RUNTIME dependency, not just build-time
if ! command -v node &> /dev/null; then
    log_info "Installing Node.js 20.x..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
    apt-get install -y nodejs
fi
```

**b) Build frontend for server mode (not static export):**
```bash
cd "$TEMP_CLONE_DIR/frontend"
npm install --legacy-peer-deps

# Copy .env.production for runtime
if [[ -f "$TEMP_CLONE_DIR/frontend/.env.production" ]]; then
    cp "$TEMP_CLONE_DIR/frontend/.env.production" "$INSTALL_DIR/ui/"
fi

npm run build   # This now produces .next/ instead of out/
```

**c) Copy .next instead of out:**
```bash
# Install UI (server mode)
if [[ -d "$TEMP_CLONE_DIR/frontend/.next" ]]; then
    rm -rf "$INSTALL_DIR/ui/.next"
    mkdir -p "$INSTALL_DIR/ui"
    cp -r "$TEMP_CLONE_DIR/frontend/.next" "$INSTALL_DIR/ui/"
    cp "$TEMP_CLONE_DIR/frontend/package.json" "$INSTALL_DIR/ui/"
    cp -r "$TEMP_CLONE_DIR/frontend/node_modules" "$INSTALL_DIR/ui/" 2>/dev/null || \
        (cd "$INSTALL_DIR/ui" && npm install --production --legacy-peer-deps)
    cp "$TEMP_CLONE_DIR/frontend/next.config.js" "$INSTALL_DIR/ui/"
    chown -R juvia:juvia "$INSTALL_DIR/ui"
    log_info "UI installed to $INSTALL_DIR/ui (server mode)"
fi
```

> **Decision:** Copy `node_modules` from the build environment to avoid re-installing on the target. If the target architecture differs, fall back to `npm install --production`.

#### 3.3 Create Systemd Service for Next.js

**New file created by install script:** `/etc/systemd/system/juvia-ui.service`

```ini
[Unit]
Description=Juvia Panel UI (Next.js Server)
After=network.target juvia-api.service
Requires=juvia-api.service

[Service]
Type=simple
User=juvia
Group=juvia
WorkingDirectory=/opt/panel/ui
ExecStart=/usr/bin/node /opt/panel/ui/node_modules/.bin/next start --port 3000
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=juvia-ui
Environment=NODE_ENV=production
Environment=PORT=3000
EnvironmentFile=/etc/panel/env

# Resource limits (management panel is low-traffic)
LimitNOFILE=65536
TimeoutStartSec=30
TimeoutStopSec=10

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/panel/ui/.next/cache

[Install]
WantedBy=multi-user.target
```

**Also update `juvia-caddy.service`** to depend on `juvia-ui.service`:
```ini
After=network.target juvia-api.service juvia-ui.service
Requires=juvia-api.service juvia-ui.service
```

**Enable and start:**
```bash
systemctl daemon-reload
systemctl enable juvia-ui
systemctl start juvia-ui
```

#### 3.4 Update the Update Script

**File:** `scripts/update.sh`

Ensure the update script also:
1. Rebuilds frontend (`npm run build`)
2. Copies `.next` output instead of `out`
3. Restarts `juvia-ui` service instead of just reloading Caddy

---

### Phase 4: CORS & Security Adjustments

#### 4.1 Update Go CORS Middleware

Since Next.js server will proxy requests from `localhost:3000` to `localhost:9090` (if BFF pattern is used) OR the browser will make direct calls (current pattern, same-origin), the CORS configuration needs adjustment:

**File:** `backend/internal/middleware/cors.go`

When `origin` is empty (same-origin request through Caddy), the current behavior already works (`allowed = true`). No changes required for the current pattern.

If BFF pattern is added in the future, add `localhost:3000` to allowed origins.

#### 4.2 Security Headers

CSS and security headers currently set by Caddy (CSP, X-Frame-Options, etc.) need to either:
- **Option A:** Keep in Caddy (applied to all responses via `header` directive)
- **Option B:** Move to Next.js `next.config.js` headers configuration (more granular)

**Recommendation:** Keep in Caddy for simplicity. Headers apply globally and don't depend on the UI server.

---

### Phase 5: Testing & Validation

#### 5.1 Manual Test Checklist

- [ ] Build: `make build-ui` succeeds and produces `.next/` directory
- [ ] Start: `npm run start` in `frontend/` starts on port 3000
- [ ] Direct URL: `http://localhost:3000/` renders dashboard
- [ ] Static route: `http://localhost:3000/apps` renders app list
- [ ] Dynamic route: `http://localhost:3000/apps/test-app` renders app detail
- [ ] Dynamic route refresh: hit F5 on `/apps/test-app` — page reloads correctly
- [ ] Login flow: navigate to `/login`, authenticate, redirected to `/`
- [ ] Auth redirect: navigate to `/` without auth → redirected to `/login`
- [ ] API calls: all API features work (apps CRUD, services, cron, etc.)
- [ ] WebSocket: real-time metrics update, deployment notifications work
- [ ] Caddy proxy: visit `http://localhost:2053/` → proxied to Next.js (:3000)
- [ ] Caddy + API: `http://localhost:2053/api/v1/apps` → proxied to Go backend (:9090)
- [ ] Caddy + WS: `ws://localhost:2053/api/v1/stream` → proxied to Go backend
- [ ] Install script: fresh install produces working system
- [ ] Systemd: all 4 services start (`juvia-agent`, `juvia-api`, `juvia-ui`, `juvia-caddy`)

#### 5.2 Automated Tests

No new automated tests needed (existing Go tests remain unchanged, frontend has no test suite).

---

## Files Changed (Summary)

| File | Action | Description |
|------|--------|-------------|
| `frontend/next.config.js` | **Modify** | Remove `output: 'export'`, add `poweredByHeader: false`, `compress: true` |
| `frontend/src/middleware.ts` | **Create** | Server-side auth redirect middleware (defense-in-depth) |
| `frontend/.env.production` | **Create** | Runtime env vars for Next.js server |
| `frontend/package.json` | **Modify** | Update `start` script with port flag |
| `backend/internal/proxy/caddy.go` | **Modify** | Simplify `writePanelUIBlock` — remove rewrite rules, remove static file_server, add reverse_proxy to Next.js |
| `backend/config/Caddyfile` | **Modify** | Replace entire content with simplified config |
| `Makefile` | **Modify** | Add `start-ui` target, update `build-ui` to run npm install |
| `scripts/install.sh` | **Modify** | Install Node.js as runtime, copy `.next` not `out`, create `juvia-ui.service`, copy `.env`, start UI service |
| `scripts/update.sh` | **Modify** | Update build/deploy logic for server mode |

| File | Action | Description |
|------|--------|-------------|
| `Caddyfile` (root) | **Delete** | Never used in deployment; superseded |
| `caddy-config/Caddyfile` | **Delete** | Duplicate; superseded |
| `backend/config/Caddyfile.new` | **Delete** | Was a simplified test version |

---

## Rollback Plan

If the migration fails in production:

1. Revert `next.config.js` changes (restore `output: 'export'`)
2. Revert `proxy/caddy.go` changes (restore rewrite rules and static file_server)
3. Revert `install.sh` to copy `out/` instead of `.next/`
4. Remove `juvia-ui.service` and restart `juvia-caddy`
5. Run `npm run build` to regenerate static export
6. All Go backend code remains unchanged, no database migration needed

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Node.js runtime required on production server | Higher resource usage (~100MB RAM) | Already required for build; panel targets 4GB+ servers |
| Next.js server crashes | UI becomes unavailable | systemd auto-restart (`Restart=always`) |
| Port 3000 conflict | Startup fails | Configurable via env var; documented in install |
| Build artifacts change (`.next` vs `out`) | Update scripts may fail | Explicitly tested in Phase 5 |
| CSP headers removed from Go generator | Losing security headers | Keep CSP in Caddy global config, not Go code |
| Memory leak in long-running Next.js | Gradual degradation | systemd MemoryMax limit (optional, add later) |
