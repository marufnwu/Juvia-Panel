# Fix Plan: UI Client-Side Navigation Not Working

## Problem Statement
- **Symptom**: Dashboard renders correctly, manual URL entry works, but clicking links (Navbar, Sidebar, etc.) does not navigate
- **Impact**: UI is essentially unusable - users can only navigate by typing URLs manually
- **Root Cause**: Client-side routing/hydration issue preventing Next.js App Router from handling link clicks

---

## Root Cause Analysis

### Primary Issue: Zustand Hydration Race Condition

**File**: `frontend/src/stores/index.ts` (lines 129-136)

```tsx
// Current (problematic):
onRehydrateStorage: () => (state) => {
  if (state) {
    state._hasHydrated = true  // Direct mutation
  }
}
```

**Problem**: Direct state mutation may not trigger React re-renders reliably, causing:
1. Hydration mismatch between server and client
2. React event listeners not attaching properly
3. Next.js router not initializing correctly

### Secondary Issue: AuthGuard Navigation Interference

**File**: `frontend/src/app/providers.tsx` (lines 86-106)

```tsx
useEffect(() => {
  // ...
  if (usersExist === true && !isAuthenticated && isProtectedRoute) {
    router.push('/login')  // Runs on EVERY pathname change
    return
  }
  // ...
}, [isLoading, _hasHydrated, usersExist, isAuthenticated, pathname, router])
```

**Problem**: Effect runs on every `pathname` change, potentially interfering with client-side navigation transitions.

### Tertiary Issue: WebSocket Connection Errors

**File**: `frontend/src/lib/websocket.ts`

If the API backend is unavailable, WebSocket connection errors may cause unhandled exceptions that break React's event system.

---

## Fix Plan

### Phase 1: Fix Zustand Hydration (Critical)

**File**: `frontend/src/stores/index.ts`

**Change**:
```tsx
// From:
{
  name: 'panel-auth',
  onRehydrateStorage: () => (state) => {
    if (state) {
      state._hasHydrated = true
    }
  },
}

// To:
{
  name: 'panel-auth',
  onRehydrateStorage: () => {
    return (state, error) => {
      if (state && !error) {
        // Use setState to ensure proper React re-render
        useAuthStore.setState({ _hasHydrated: true })
      }
    }
  },
}
```

**Why**: `setState()` triggers proper React state update and re-render, ensuring hydration completes correctly.

---

### Phase 2: Fix AuthGuard Effect Dependencies (Critical)

**File**: `frontend/src/app/providers.tsx`

**Change** (lines 86-106):
```tsx
// From:
useEffect(() => {
  if (isLoading || !_hasHydrated) return

  const isProtectedRoute = protectedRoutes.some(route => normalizedPathname === route || normalizedPathname.startsWith(route + '/'))
  const isAuthRoute = authRoutes.includes(normalizedPathname)

  if (usersExist === false && !isAuthRoute && pathname !== '/setup') {
    router.push('/setup')
    return
  }

  if (usersExist === true && !isAuthenticated && isProtectedRoute) {
    router.push('/login')
    return
  }

  if (isAuthenticated && isAuthRoute) {
    router.push('/')
    return
  }
}, [isLoading, _hasHydrated, usersExist, isAuthenticated, pathname, router])

// To:
useEffect(() => {
  if (isLoading || !_hasHydrated) return

  const isProtectedRoute = protectedRoutes.some(route => normalizedPathname === route || normalizedPathname.startsWith(route + '/'))
  const isAuthRoute = authRoutes.includes(normalizedPathname)

  // Use replace() instead of push() to avoid polluting history
  // Only redirect on auth state changes, not on every pathname change
  if (usersExist === false && !isAuthRoute && pathname !== '/setup') {
    router.replace('/setup')
    return
  }

  if (usersExist === true && !isAuthenticated && isProtectedRoute) {
    router.replace('/login')
    return
  }

  if (isAuthenticated && isAuthRoute) {
    router.replace('/')
    return
  }
}, [isLoading, _hasHydrated, usersExist, isAuthenticated]) // Removed pathname and router
```

**Why**: 
1. Removing `pathname` from deps prevents the effect from running on every navigation
2. Using `replace()` instead of `push()` prevents history pollution
3. Auth redirects should only happen on auth state changes, not route changes

---

### Phase 3: Improve WebSocket Error Handling (Defensive)

**File**: `frontend/src/lib/websocket.ts`

**Change** (around line 118):
```tsx
// From:
this.ws.onerror = (error) => {
  console.error('WebSocket error:', error)
}

// To:
this.ws.onerror = (error) => {
  console.error('WebSocket error:', error)
  // Don't let WebSocket errors crash the app
  // The onclose handler will handle reconnection
}
```

**Additional Change** (around line 84):
```tsx
// From:
connect(): void {
  if (this.ws?.readyState === WebSocket.OPEN || this.isConnecting) {
    return
  }
  // ...
}

// To:
connect(): void {
  if (this.ws?.readyState === WebSocket.OPEN || this.isConnecting) {
    return
  }

  // Check if WebSocket is in CLOSING state
  if (this.ws?.readyState === WebSocket.CLOSING) {
    return
  }

  // Clean up any existing closed connection
  if (this.ws?.readyState === WebSocket.CLOSED) {
    this.ws = null
  }
  // ...
}
```

**Why**: Prevents unhandled exceptions from breaking React's event system.

---

### Phase 4: Verify Caddyfile Configuration (Validation)

**File**: `backend/config/Caddyfile`

**Verify** the catch-all block uses correct syntax:
```caddy
handle {
    header {
        # ... security headers
    }
    root * /opt/panel/ui/out  # Note the * matcher
    try_files {path} {path}/index.html /index.html
    file_server
}
```

**Change if needed**:
```caddy
# From:
root /opt/panel/ui/out

# To:
root * /opt/panel/ui/out
```

**Why**: The `*` matcher ensures the root applies to all request phases correctly.

---

### Phase 5: Rebuild and Deploy

```bash
# 1. Clean previous build
cd frontend
rm -rf .next out

# 2. Install dependencies (if needed)
npm install

# 3. Build
npm run build

# 4. Verify output
ls -la out/
ls -la out/_next/static/chunks/

# 5. Copy to deployment location (on server)
# cp -r out/* /opt/panel/ui/out/

# 6. Reload Caddy (on server)
# systemctl reload caddy
```

---

## Verification Steps

### 1. Build Verification
- [ ] `npm run build` completes without errors
- [ ] `out/index.html` exists
- [ ] `out/_next/static/chunks/` contains JS files
- [ ] No hydration warnings in build output

### 2. Runtime Verification
- [ ] Open browser DevTools Console
- [ ] Navigate to dashboard - should render without "Hydration failed" errors
- [ ] Click "Apps" in navbar - should navigate to /apps
- [ ] Click "Services" in sidebar - should navigate to /services
- [ ] Click browser back button - should work
- [ ] Refresh page on /apps - should load correctly
- [ ] Manual URL entry (e.g., /settings) - should work

### 3. Error Checking
- [ ] No "Minified React error #418" (hydration mismatch)
- [ ] No "Text content did not match" warnings
- [ ] No uncaught exceptions in console
- [ ] WebSocket errors (if any) don't break navigation

---

## Rollback Plan

If fixes don't work:

1. **Revert changes**:
   ```bash
   git checkout -- frontend/src/stores/index.ts
   git checkout -- frontend/src/app/providers.tsx
   git checkout -- frontend/src/lib/websocket.ts
   ```

2. **Rebuild**:
   ```bash
   cd frontend && npm run build
   ```

3. **Investigate further**:
   - Check browser console for specific errors
   - Verify all JS chunks load correctly (Network tab)
   - Check if issue is browser-specific

---

## Files to Modify

| File | Change Type | Priority |
|------|-------------|----------|
| `frontend/src/stores/index.ts` | Fix Zustand hydration | Critical |
| `frontend/src/app/providers.tsx` | Fix AuthGuard effect | Critical |
| `frontend/src/lib/websocket.ts` | Improve error handling | Medium |
| `backend/config/Caddyfile` | Verify root directive | Low |

---

## Additional Notes

### Why Manual URL Works but Clicks Don't

1. **Manual URL**: Browser makes full page request → Caddy serves HTML → React hydrates fresh
2. **Link Click**: Next.js intercepts click → Client-side navigation → Router updates URL → Component renders

If React's event system is broken (due to hydration issues), link clicks aren't intercepted properly, so they fall back to default browser behavior (which may not work with SPA routing).

### Debug Commands

```bash
# Check for hydration errors in browser console
# Look for: "Warning: Text content did not match"
# Look for: "Minified React error #418"

# Check if JS chunks load correctly
curl -I http://localhost:2053/_next/static/chunks/webpack-*.js

# Check Caddy logs
journalctl -u caddy -f

# Validate Caddyfile
caddy validate --config /etc/panel/caddy/Caddyfile
```

---

## Estimated Time

- Phase 1-2 (Critical fixes): 10 minutes
- Phase 3 (Defensive fixes): 5 minutes
- Phase 4-5 (Verification): 10 minutes
- **Total**: ~25 minutes
