# Port Architecture Redesign - Architectural Plan

## Problem Statement

Current implementation has two security/design issues:

1. **Panel UI on port 8080** - Too common, can conflict with other services
2. **API port publicly accessible** - Should only be accessed by frontend internally via reverse proxy

## Current Architecture

```
Internet → Caddy :443 (HTTPS)
                ├── panel.example.com/* → /opt/panel/ui/ (static)
                └── api.example.com/*   → localhost:8080 (API)

Internal only: Nothing (8080 is exposed on 0.0.0.0)
```

## Proposed Architecture

```
Internet → Caddy :443 (HTTPS only)
                └── *.{domain}/* → everything via Caddy proxy

Internal (localhost only):
                localhost:{UNCOMMON_UI_PORT} → Go API server
```

Caddy acts as the **single public entry point**. All traffic (UI + API) flows through Caddy. The API server is bound to `localhost` only and never directly exposed.

## New Port Design

| Service | Bind Address | Port | Purpose |
|---------|-------------|------|---------|
| Panel UI | `localhost` | `2053` (uncommon) | Caddy → API internal communication |
| Go API Server | `localhost` | `2053` | Receives requests from Caddy only |
| Caddy | `0.0.0.0` | 80, 443 | Public HTTPS entry point |
| SSH | `0.0.0.0` | 22 | Server access |

**Note:** UI port (2053) is for Caddy's internal proxy to the API - not directly user-accessible.

## Caddyfile Changes

### Current (Problematic)
```caddy
# API server - proxy /api/* requests to Go API on localhost:8080
api.{$DOMAIN}, {$DOMAIN}:8080 {
    reverse_proxy localhost:8080
    ...
}
```

### Proposed (Fixed)
```caddy
# Panel domain - serves both static UI and proxies API
# Everything under one domain, API is /api/* path
panel.{$DOMAIN} {
    # Static UI files
    root * /opt/panel/ui
    file_server

    # SPA routing
    @spa {
        not {
            file {
                try_files {path} {path}/ /index.html
            }
        }
    }
    rewrite @spa /index.html

    # API proxy - localhost only, not publicly accessible
    handle /api/* {
        reverse_proxy localhost:2053
    }

    # WebSocket support
    @ws {
        header Connection *Upgrade*
        header Upgrade websocket
    }
    reverse_proxy @ws localhost:2053
}
```

### Key Changes:
1. **Remove `api.*` subdomain** - API is accessed via `/api/*` path
2. **API on uncommon port** - Use `localhost:2053` instead of `8080`
3. **Single domain for panel** - `panel.{domain}` serves UI + API

## config.yaml Changes

```yaml
server:
  host: "127.0.0.1"   # Changed from "0.0.0.0" - localhost only!
  port: 2053          # Changed from 8080 to uncommon port
```

## install.sh Changes

### Port Configuration
- Default UI/API port: `2053`
- Add `--port` flag for customization
- Validate port is not well-known (1-1023 reserved)

### Firewall Configuration
- Remove external access to port 8080
- Only allow 80, 443 externally

### Caddyfile Generation
- Generate config with single `panel.{domain}` block
- Remove `api.*` subdomain handling

## Access Flow (New)

```
User types: https://panel.example.com
    │
    ▼
Caddy :443 (handles HTTPS)
    │
    ├── /* (static files) → /opt/panel/ui/
    │
    └── /api/* → reverse_proxy localhost:2053
                    │
                    ▼
                 Go API Server (localhost:2053 only)
```

## User Access URL

| Scenario | URL |
|----------|-----|
| With domain | `https://panel.example.com` |
| Without domain (IP) | `http://{ip}` (Caddy on 80) |

## Benefits

1. **No port conflicts** - Using uncommon port 2053
2. **API never exposed** - Bound to localhost only, proxied via Caddy
3. **Single entry point** - Users visit one domain, Caddy routes internally
4. **Automatic SSL** - Works with Let's Encrypt via Caddy
5. **Simpler UX** - One URL for everything

## Files to Modify

1. `backend/config/config.yaml` - Change port to 2053, host to 127.0.0.1
2. `backend/config/Caddyfile` - Simplify to single panel domain, proxy to localhost:2053
3. `scripts/install.sh` - Update port configuration, firewall rules
4. `lifecycle-scripts-specification.md` - Update documentation
5. `api-specification.md` - Update base URL documentation

## Rollout Considerations

1. Existing installations will need migration
2. Database stores settings, so config changes apply on restart
3. Caddyfile is auto-generated from template on install/update
4. No database schema changes needed

## Migration Path for Existing Installations

For existing installations updating:
1. Stop services
2. Update Caddyfile
3. Update config.yaml
4. Restart services
5. Verify health check passes