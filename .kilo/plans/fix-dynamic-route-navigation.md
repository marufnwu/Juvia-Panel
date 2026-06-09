# Fix Plan: Simplify Caddy + Fix Frontend Dynamic Routes

## Problem Summary

The current approach has two issues:
1. **Frontend bug**: `useParams()` returns pre-generated static params (`_`) instead of actual URL params when using Next.js static export with `generateStaticParams()`
2. **Over-engineered Caddy**: We added route-specific rewrites for `/apps/*`, `/services/*`, `/cron/*` which is unnecessary

## Root Cause

In Next.js static export, `generateStaticParams()` returns `[{ slug: ['_'] }]` as a placeholder. When the page loads, `useParams()` returns this placeholder instead of the actual URL segment. This causes API calls to go to `/api/v1/apps/_` instead of `/api/v1/apps/app_Eip4HNSkL`.

## Solution

### Part 1: Fix Frontend (3 files)

Replace `useParams()` with `window.location.pathname` extraction in:

1. **`frontend/src/app/apps/[...slug]/AppDetailClient.tsx`** (line 145, 151)
   ```tsx
   // Change from:
   const params = useParams()
   const appId = (params.slug as string[])?.[0] as string
   
   // Change to:
   const pathname = typeof window !== 'undefined' ? window.location.pathname : ''
   const pathSegments = pathname.split('/').filter(Boolean)
   const appId = pathSegments[1] || '' // /apps/{appId} -> index 1
   ```
   Also remove `useParams` from imports.

2. **`frontend/src/app/services/[...slug]/ServiceDetailClient.tsx`** (line 113, 118)
   ```tsx
   // Change from:
   const params = useParams()
   const serviceId = (params.slug as string[])?.[0] as string
   
   // Change to:
   const pathname = typeof window !== 'undefined' ? window.location.pathname : ''
   const pathSegments = pathname.split('/').filter(Boolean)
   const serviceId = pathSegments[1] || '' // /services/{serviceId} -> index 1
   ```
   Also remove `useParams` from imports.

3. **`frontend/src/app/cron/[...slug]/CronJobDetailClient.tsx`** (line 132, 134)
   ```tsx
   // Change from:
   const params = useParams()
   const cronJobId = (params.slug as string[])?.[0] as string
   
   // Change to:
   const pathname = typeof window !== 'undefined' ? window.location.pathname : ''
   const pathSegments = pathname.split('/').filter(Boolean)
   const cronJobId = pathSegments[1] || '' // /cron/{cronJobId} -> index 1
   ```
   Also remove `useParams` from imports.

### Part 2: Simplify Caddyfile

Remove all route-specific rewrite rules. Caddy should only:
- Serve static files (`/_next/*`, `/static/*`)
- Reverse proxy API requests (`/api/*`) to Go backend
- SPA fallback for client-side routing

**`backend/config/Caddyfile`** - Remove lines 61-84 (rewrite rules)

**`backend/internal/proxy/caddy.go`** - Remove lines 164-185 (rewrite rules in `writePanelUIBlock`)

### Part 3: Rebuild and Deploy

1. Rebuild frontend: `cd frontend && npm run build`
2. Rebuild backend: `cd backend && go build -o panel-api ./cmd/api`
3. Deploy to server

## Final Caddyfile Structure

```caddy
{
    admin unix//var/run/panel/caddy-admin.sock
    log { level INFO }
}

:2053 {
    # Next.js static assets (immutable, cached)
    handle /_next/* {
        header { /* security headers + CSP with blob: */ }
        file_server { root /opt/panel/ui/out }
    }

    # Static assets
    handle /static/* {
        header { /* security headers + CSP with blob: */ }
        file_server { root /opt/panel/ui/out }
    }

    # API proxy (includes WebSocket at /api/v1/stream)
    handle /api* {
        reverse_proxy localhost:9090
    }

    # Health check
    handle /health {
        reverse_proxy localhost:9090
    }

    # SPA fallback - serve index.html for client-side routing
    handle {
        header { /* security headers + CSP with blob: */ }
        root * /opt/panel/ui/out
        try_files {path} {path}/index.html /index.html
        file_server
    }
}
```

## Files to Modify

| File | Change |
|------|--------|
| `frontend/src/app/apps/[...slug]/AppDetailClient.tsx` | Replace `useParams()` with pathname extraction |
| `frontend/src/app/services/[...slug]/ServiceDetailClient.tsx` | Replace `useParams()` with pathname extraction |
| `frontend/src/app/cron/[...slug]/CronJobDetailClient.tsx` | Replace `useParams()` with pathname extraction |
| `backend/config/Caddyfile` | Remove route-specific rewrite rules |
| `backend/internal/proxy/caddy.go` | Remove route-specific rewrite rules from `writePanelUIBlock` |

## Verification

After deployment:
1. Visit `http://192.168.0.211:2053/apps/app_Eip4HNSkL`
2. Check Network tab - API calls should go to `/api/v1/apps/app_Eip4HNSkL` (not `/api/v1/apps/_`)
3. Click links in navbar - should navigate correctly
4. Refresh page - should load correctly
