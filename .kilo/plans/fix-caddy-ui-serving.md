# Fix Caddy UI Serving - Duplicate Content in Caddyfile

## Problem

The `backend/config/Caddyfile` has **duplicate content** after line 67. Lines 69-97 contain extra `handle` blocks that are **outside any site block** - this is invalid Caddyfile syntax.

### File Comparison

| File | Status | Lines |
|------|--------|-------|
| `caddy-config/Caddyfile` | Clean | 67 |
| `Caddyfile` (root) | Clean | 67 |
| `backend/config/Caddyfile` | **CORRUPTED** | 97 (lines 69-97 are invalid duplicates) |

### What's Broken in `backend/config/Caddyfile`

Lines 1-67: Valid Caddyfile with `:2053 { ... }` site block (correct)
Lines 69-97: **Duplicate `handle` blocks outside any site block** (invalid syntax)

```
# Lines 68-97 (INVALID - outside any site block):
    # Next.js static files (_next/static, etc.)
    handle /_next/* {
        file_server {
            root /opt/panel/ui/out
        }
    }
    ... more duplicate blocks ...
}   # <-- stray closing brace
```

This causes Caddy to fail parsing the config, breaking all UI routes except `/login` (which may work due to caching or coincidental fallback).

## Fix

### Remove duplicate lines 68-97 from `backend/config/Caddyfile`

The file should end at line 67 (the closing `}` of the `:2053` site block).

**Before:** 97 lines (with duplicate content after line 67)
**After:** 67 lines (clean, matches `caddy-config/Caddyfile`)

## Validation

1. `backend/config/Caddyfile` should be identical to `caddy-config/Caddyfile`
2. `caddy validate --config backend/config/Caddyfile` should pass
3. After deploy: all UI routes should work (`/`, `/login`, `/setup`, `/apps`, etc.)
