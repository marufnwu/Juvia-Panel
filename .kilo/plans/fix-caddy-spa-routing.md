# Fix Caddy SPA Routing Issue

## Problem
**Critical**: No client-side routing works in the UI panel. Clicking any link (Apps, Services, Settings, etc.) fails to navigate.

### Symptoms
- `/login` redirects to `/login/` (308 redirect)
- Client-side navigation (clicking links) doesn't work
- Direct URL access to routes like `/apps` may fail

### Root Cause
The Caddyfile uses incorrect Caddy v2 syntax for SPA routing. The `try_files` directive is placed inside a `handle` block, which doesn't work correctly in Caddy v2.

**Current (broken):**
```caddyfile
handle {
    try_files {path} {path}/index.html /index.html
    file_server {
        root /opt/panel/ui/out
    }
}
```

## Solution

### Fix: Correct Caddy v2 SPA Routing Syntax

**Option 1: Using `try_files` at site level (simplest)**
```caddyfile
:2053 {
    root /opt/panel/ui/out
    
    # Handle static assets first
    handle /_next/* {
        file_server
    }
    
    handle /static/* {
        file_server
    }
    
    # Proxy API to backend
    handle /api/* {
        reverse_proxy localhost:9090
    }
    
    handle /health {
        reverse_proxy localhost:9090
    }
    
    # SPA fallback - must be last
    try_files {path} {path}/index.html /index.html
    file_server
}
```

**Option 2: Using matcher (more explicit)**
```caddyfile
:2053 {
    root /opt/panel/ui/out
    
    handle /_next/* {
        file_server
    }
    
    handle /static/* {
        file_server
    }
    
    handle /api/* {
        reverse_proxy localhost:9090
    }
    
    handle /health {
        reverse_proxy localhost:9090
    }
    
    # SPA routing
    @spa {
        not path /api/*
        not path /health
        not path /_next/*
        not path /static/*
        not file {path}
    }
    handle @spa {
        rewrite /index.html
    }
    
    handle {
        file_server
    }
}
```

I recommend **Option 1** as it's simpler and proven to work.

## Files to Modify

1. **`backend/internal/proxy/caddy.go`** - Fix `writePanelUIBlock()` function (lines 108-150)
2. **`backend/config/Caddyfile`** - Update static config to match
3. **`Caddyfile`** - Update root Caddyfile
4. **`caddy-config/Caddyfile`** - Update caddy-config Caddyfile

## Implementation Steps

1. Update `writePanelUIBlock()` to generate correct Caddyfile syntax
2. Update all static Caddyfile copies
3. Build and test locally
4. Deploy to server
5. Verify all routes work

## Validation

After deployment, test:
```bash
# These should all return 200 (not 308 or 404)
curl -I http://localhost:2053/
curl -I http://localhost:2053/login
curl -I http://localhost:2053/setup
curl -I http://localhost:2053/apps
curl -I http://localhost:2053/services

# Client-side navigation should work without page reloads
```
