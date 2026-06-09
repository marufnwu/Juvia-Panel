# Juvia Panel - Comprehensive Issue Fix Plan

## Summary of Issues Found

After analyzing the codebase, I've identified **10 issues** across authentication, routing, and security. The critical issue causing the login redirect problem is that the `AuthGuard` component doesn't wait for Zustand persist rehydration before making auth decisions.

---

## Issue #1: AuthGuard doesn't wait for persist rehydration (CRITICAL)

**Location:** `frontend/src/app/providers.tsx:66-116`

**Problem:** The `AuthGuard` component checks `isAuthenticated` and `usersExist` but doesn't wait for Zustand's persist middleware to rehydrate from localStorage. On page load:
- `_hasHydrated` is `false`
- `isAuthenticated` is `false` (initial state, even if user was logged in)
- `usersExist` is `null` (initial state)

This causes the guard to redirect to `/login` even when the user has a valid session stored in localStorage.

**Fix:**
```typescript
const { isAuthenticated, checkUsersExist, usersExist, _hasHydrated } = useAuthStore()

useEffect(() => {
  if (!_hasHydrated) return  // Wait for rehydration
  async function checkAuth() {
    if (usersExist === null) {
      await checkUsersExist()
    }
    setIsLoading(false)
  }
  checkAuth()
}, [checkUsersExist, usersExist, _hasHydrated])
```

---

## Issue #2: AuthGuard and Dashboard have conflicting auth checks

**Location:** `frontend/src/app/providers.tsx:66-116` and `frontend/src/app/page.tsx:242-260`

**Problem:** Both `AuthGuard` (global) and `DashboardPage` have their own auth checks. This can cause:
- Double redirects
- Race conditions
- Inconsistent behavior

**Fix:** Remove the auth check from `DashboardPage` since `AuthGuard` already handles it globally. The dashboard should only check `_hasHydrated` to wait for the store to be ready.

---

## Issue #3: Login page uses raw fetch instead of api client

**Location:** `frontend/src/app/login/page.tsx:42-57`

**Problem:** The login page uses raw `fetch` instead of `api.auth.login()`. This is inconsistent and bypasses the API client's error handling and retry logic.

**Fix:** Use `api.auth.login()` instead:
```typescript
const data = await api.auth.login(email, password)
useAuthStore.getState().setAuth(data.access_token, data.user)
```

---

## Issue #4: Setup page doesn't wait for hydration

**Location:** `frontend/src/app/setup/page.tsx:45-49`

**Problem:** The setup page checks `isAuthenticated` without waiting for `_hasHydrated`. This could cause incorrect redirects if the user was previously logged in.

**Fix:**
```typescript
const { isAuthenticated, _hasHydrated } = useAuthStore()

useEffect(() => {
  if (_hasHydrated && isAuthenticated) {
    router.push('/')
  }
}, [isAuthenticated, _hasHydrated, router])
```

---

## Issue #5: Login page doesn't wait for hydration before checking setup

**Location:** `frontend/src/app/login/page.tsx:24-34`

**Problem:** The login page calls `checkUsersExist()` immediately on mount without waiting for `_hasHydrated`. This could cause unnecessary API calls or incorrect behavior.

**Fix:**
```typescript
const { _hasHydrated } = useAuthStore()

useEffect(() => {
  if (!_hasHydrated) return
  async function check() {
    const { setupCompleted, error } = await checkUsersExist()
    if (!error && setupCompleted === false) {
      window.location.href = '/setup'
      return
    }
    setIsChecking(false)
  }
  check()
}, [checkUsersExist, _hasHydrated])
```

---

## Issue #6: Missing SameSite attribute on refresh token cookie

**Location:** `backend/internal/handlers/auth/auth.go:232-240` and `771-779`

**Problem:** The refresh token cookie is set with `Secure` and `HttpOnly` but no `SameSite` attribute. This could cause issues with cross-origin requests or CSRF attacks.

**Fix:** Add `SameSite=Strict` or `SameSite=Lax`:
```go
c.SetCookie(
    "refresh_token",
    refreshToken,
    int(cfg.RefreshExpiry.Seconds()),
    "/",
    "",
    cfg.Env == "production",
    true, // httpOnly
)
// Note: Need to use http.SetCookie for SameSite
```

Actually, Gin's `SetCookie` doesn't support `SameSite` directly. Need to use `http.SetCookie`:
```go
cookie := &http.Cookie{
    Name:     "refresh_token",
    Value:    refreshToken,
    Path:     "/",
    MaxAge:   int(cfg.RefreshExpiry.Seconds()),
    Secure:   cfg.Env == "production",
    HttpOnly: true,
    SameSite: http.SameSiteStrictMode,
}
http.SetCookie(c.Writer, cookie)
```

---

## Issue #7: CORS configuration too permissive in development

**Location:** `backend/internal/middleware/cors.go:32-34`

**Problem:** When `cfg.AllowedOrigins` and `cfg.PanelDomain` are both empty, it allows all origins. This is a security concern.

**Fix:** In production, require explicit origin configuration:
```go
} else if cfg.AllowedOrigins == "" && cfg.PanelDomain == "" {
    if cfg.Env == "production" {
        // In production, don't allow all origins
        c.AbortWithStatusJSON(403, gin.H{"error": "origin not allowed"})
        return
    }
    // In development, allow all origins
    c.Header("Access-Control-Allow-Origin", origin)
    allowed = true
}
```

---

## Issue #8: WebSocket endpoint doesn't require authentication

**Location:** `backend/cmd/api/main.go:136-138`

**Problem:** The WebSocket endpoint `/api/v1/stream` doesn't use the auth middleware. This means anyone can connect and receive real-time updates.

**Fix:** Add authentication to WebSocket:
```go
router.GET("/api/v1/stream", middleware.Auth(cfg), func(c *gin.Context) {
    wsHub.ServeWs(c.Writer, c.Request)
})
```

---

## Issue #9: Potential race condition in login flow

**Location:** `frontend/src/app/login/page.tsx:54-57`

**Problem:** After login, `setAuth` is called, then `router.push('/')`. But the `AuthGuard` might still be processing and could redirect back to login if it hasn't seen the updated auth state yet.

**Fix:** Add a small delay or use `window.location.href` for a full page reload to ensure the auth state is persisted before navigation:
```typescript
useAuthStore.getState().setAuth(data.access_token, data.user)
// Wait for persist to write to localStorage
await new Promise(resolve => setTimeout(resolve, 100))
router.push('/')
```

Or better, use `window.location.href = '/'` for a full page reload which will trigger rehydration.

---

## Issue #10: AuthGuard doesn't handle loading state properly

**Location:** `frontend/src/app/providers.tsx:82-102`

**Problem:** The `AuthGuard` has complex logic that can cause issues:
- It checks `usersExist` and `isAuthenticated` in the same effect
- It doesn't properly handle the case where `usersExist` is `null` (loading)
- It can cause flash of content or incorrect redirects

**Fix:** Simplify the logic and ensure proper loading states:
```typescript
useEffect(() => {
  if (isLoading || !_hasHydrated) return

  const isProtectedRoute = protectedRoutes.some(route => pathname.startsWith(route))
  const isAuthRoute = authRoutes.includes(pathname)

  // If setup not completed and not on setup page, redirect to setup
  if (usersExist === false && !isAuthRoute && pathname !== '/setup') {
    router.push('/setup')
    return
  }

  // If setup completed but not authenticated and on protected route, redirect to login
  if (usersExist === true && !isAuthenticated && isProtectedRoute) {
    router.push('/login')
    return
  }

  // If authenticated and on auth route, redirect to dashboard
  if (isAuthenticated && isAuthRoute) {
    router.push('/')
    return
  }
}, [isLoading, _hasHydrated, usersExist, isAuthenticated, pathname, router])
```

---

## Implementation Order

1. **Fix AuthGuard to wait for hydration** (Issue #1) - This is the root cause
2. **Fix login page to wait for hydration** (Issue #5)
3. **Fix setup page to wait for hydration** (Issue #4)
4. **Remove duplicate auth check from Dashboard** (Issue #2)
5. **Fix login page to use api client** (Issue #3)
6. **Fix race condition in login flow** (Issue #9)
7. **Fix AuthGuard loading state** (Issue #10)
8. **Add SameSite to cookie** (Issue #6)
9. **Fix CORS configuration** (Issue #7)
10. **Add auth to WebSocket** (Issue #8)

---

## Files to Modify

1. `frontend/src/app/providers.tsx` - Fix AuthGuard
2. `frontend/src/app/login/page.tsx` - Fix login page
3. `frontend/src/app/setup/page.tsx` - Fix setup page
4. `frontend/src/app/page.tsx` - Remove duplicate auth check
5. `backend/internal/handlers/auth/auth.go` - Fix cookie SameSite
6. `backend/internal/middleware/cors.go` - Fix CORS
7. `backend/cmd/api/main.go` - Add auth to WebSocket

---

## Testing Checklist

After implementing fixes, verify:
- [ ] Login redirects to dashboard correctly
- [ ] Page refresh keeps user logged in
- [ ] Logout clears auth state
- [ ] Setup page redirects to login if setup completed
- [ ] Login page redirects to setup if setup not completed
- [ ] Protected routes redirect to login if not authenticated
- [ ] Authenticated users redirected to dashboard from login/setup
- [ ] WebSocket requires authentication
- [ ] CORS works correctly in production
- [ ] Refresh token cookie has SameSite attribute
