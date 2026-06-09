# Fix Plan: UI Navigation & CSP/WebSocket Issues

## Problem Summary

After the previous fixes, the UI still has critical issues:
- Dashboard renders but clicking links doesn't navigate
- Console errors show:
  1. **CSP blocking blob workers**: `Content Security Policy directive: "script-src 'self' 'unsafe-inline' 'unsafe-eval'"` - blocking React's web workers
  2. **WebSocket connection failures**: `WebSocket connection to 'ws://192.168.0.211:2053/api/v1/stream?token=...' failed`
  3. **Worker creation blocked**: `Creating a worker from 'blob:...' violates the following Content Security Policy directive`

## Root Cause Analysis

### Issue 1: CSP Headers Too Restrictive
The CSP headers in both `backend/config/Caddyfile` and `backend/internal/proxy/caddy.go` are missing:
- `blob:` in `script-src` - required for web workers created from blob URLs
- `worker-src` directive - needed to explicitly allow web workers

React uses web workers for Monaco Editor, xterm.js, and other components. When CSP blocks these, React fails during initialization, breaking client-side routing and navigation.

### Issue 2: WebSocket Proxy Configuration
The WebSocket endpoint `/api/v1/stream` is handled by the generic `/api*` handler. While Caddy's `reverse_proxy` should handle WebSocket upgrades automatically, we need to ensure:
- Explicit WebSocket handling for reliability
- Proper header forwarding for WebSocket upgrades

## Fix Plan

### Phase 1: Update CSP Headers (Critical)

**Files to modify:**
1. `backend/config/Caddyfile` (static template)
2. `backend/internal/proxy/caddy.go` (dynamic generation)

**Changes:**
Update the CSP directive from:
```
Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self' ws: wss:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
```

To:
```
Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' blob:; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self' ws: wss:; worker-src 'self' blob:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
```

**Key changes:**
1. Add `blob:` to `script-src` - allows blob-based web workers
2. Add `worker-src 'self' blob:` - explicitly allows web workers from blob URLs

### Phase 2: Add Explicit WebSocket Handler (Critical)

**File:** `backend/internal/proxy/caddy.go`

Add a dedicated WebSocket handler before the generic `/api*` handler:

```go
// WebSocket endpoint - explicit handling for upgrades
handle /api/v1/stream {
    reverse_proxy localhost:9090 {
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }
}
```

This ensures WebSocket connections get proper upgrade handling.

### Phase 3: Update Static Caddyfile (Validation)

**File:** `backend/config/Caddyfile`

Apply the same CSP and WebSocket fixes to the static template.

### Phase 4: Rebuild and Deploy

**Steps:**
1. Rebuild backend:
   ```bash
   cd backend
   go build -o panel-api ./cmd/api
   go build -o panel-agent ./cmd/agent
   ```

2. Rebuild frontend (if needed):
   ```bash
   cd frontend
   npm run build
   ```

3. Deploy:
   ```bash
   sudo systemctl stop juvia-api juvia-agent
   sudo cp panel-api /usr/local/bin/
   sudo cp panel-agent /usr/local/bin/
   sudo systemctl start juvia-api juvia-agent
   sudo systemctl reload caddy
   ```

## Verification Steps

### 1. Check CSP Headers
```bash
curl -I http://localhost:2053/ | grep -i content-security-policy
```
Should show:
```
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' blob:; ... worker-src 'self' blob:; ...
```

### 2. Check WebSocket Connection
Open browser DevTools → Network → WS tab:
- Should see WebSocket connection to `/api/v1/stream`
- Connection should show "101 Switching Protocols"
- No errors in console

### 3. Check Navigation
- Click "Apps" in navbar → should navigate to /apps
- Click "Services" in sidebar → should navigate to /services
- Click browser back button → should work
- Refresh page on any route → should load correctly

### 4. Check Console for Errors
- No CSP violations about blob workers
- No WebSocket connection errors
- No React hydration errors

## Files to Modify

| File | Change Type | Priority |
|------|-------------|----------|
| `backend/internal/proxy/caddy.go` | Update CSP + add WebSocket handler | Critical |
| `backend/config/Caddyfile` | Update CSP + add WebSocket handler | Critical |

## Estimated Time
- Phase 1-2: 10 minutes
- Phase 3-4: 10 minutes
- **Total**: ~20 minutes

## Rollback Plan

If fixes don't work:
1. Revert changes to `caddy.go` and `Caddyfile`
2. Rebuild backend
3. Restart services
4. Investigate further with browser DevTools
