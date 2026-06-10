# Fix: Login Redirect Loop (308/307 Redirects)

## Root Cause

Two interacting issues cause a redirect loop after login:

### Issue 1: `trailingSlash: true` causes 308 redirects on every page

In `frontend/next.config.js`, `trailingSlash: true` makes Next.js respond with 308 Permanent Redirect for any URL missing a trailing slash (e.g., `/login` → `/login/`, `/apps` → `/apps/`). This was needed for static export (`output: 'export'`) where trailing slashes mapped to `index.html` files inside directories. In server mode, it's unnecessary and adds redirects.

### Issue 2: Secure cookie never sent over HTTP → middleware infinite redirect

The `refresh_token` cookie is set with `Secure: true` in production (`auth.go:237`: `Secure: cfg.Env == "production"`). The panel is accessed via **HTTP** on port 2053. Browsers will **never** send a `Secure` cookie over HTTP. Since `src/middleware.ts` checks for this cookie at line 18 and redirects to `/login` if absent (line 21-24), every page navigation triggers a 307 redirect to `/login`, even immediately after successful login.

**Why it worked before:** The old static export had no middleware. The `AuthGuard` in `providers.tsx` manages auth client-side via `access_token` in localStorage — this still works correctly and needs no changes.

---

## Fix Plan

### Step 1: Remove `trailingSlash: true` from `next.config.js`

**File:** `frontend/next.config.js`

**Change:**
```diff
 const nextConfig = {
-  trailingSlash: true,
   poweredByHeader: false,
   compress: true,
 }
```

**Rationale:** Eliminates all 308 Permanent Redirects. All client-side `<Link>` and `router.push()` in the codebase already use paths without trailing slashes (e.g., `/apps`, `/login`, `/services`). No other changes needed.

### Step 2: Remove redirect logic from middleware

**File:** `frontend/src/middleware.ts`

**Change:** Replace the auth redirect logic with a simple pass-through. Keep the middleware file for future security headers.

```typescript
import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

export function middleware(request: NextRequest) {
  return NextResponse.next()
}

export const config = {
  matcher: [
    '/((?!_next|api|health|favicon.ico).*)',
  ],
}
```

**Rationale:** The `refresh_token` HttpOnly cookie cannot be read by the middleware over HTTP due to its `Secure` flag. Attempting to validate auth server-side is futile. The client-side `AuthGuard` in `providers.tsx` already handles auth via localStorage `accessToken`. This component:

1. Checks `isAuthenticated` from Zustand store (`stores/index.ts:46`)
2. Checks `usersExist` via API call to determine if setup is needed
3. Redirects unauthenticated users to `/login` 
4. Redirects authenticated users away from `/login`/`setup`

This all works correctly and needs no changes. Removing the middleware redirect eliminates all 307 redirects.

### Step 3: Rebuild and test

1. Build frontend: `cd frontend && npm run build`
2. Copy `.next` to server
3. Restart `juvia-ui` service
4. Verify login flow works without redirects

---

## Verification Checklist

- [ ] `GET /login` returns 200 directly (no 308 trailing-slash redirect)
- [ ] `GET /apps` returns 200 (after login, no 308 redirect)
- [ ] `GET /` loads dashboard after login (not redirected to `/login`)
- [ ] Login form submission works and redirects to dashboard
- [ ] No 307 or 308 redirects in network tab
- [ ] Dynamic route `/apps/test-app` works without redirects
