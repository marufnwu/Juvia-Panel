# Fix Panel UI Navigation - Trailing Slash Mismatch

## Root Cause

Next.js is configured with `trailingSlash: true` in `next.config.js`, which means:
- URLs have trailing slashes: `/login/`, `/apps/`, `/setup/`
- `usePathname()` returns paths with trailing slashes: `/login/`

But the route arrays in `providers.tsx` don't have trailing slashes:
```javascript
const authRoutes = ['/login', '/setup']
```

So when checking `authRoutes.includes(pathname)` with `pathname = '/login/'`, it returns `false` because `/login/` !== `/login`.

This breaks:
1. **Login redirect**: After login, `isAuthRoute` is `false` so the redirect to `/` doesn't trigger
2. **Auth route detection**: Authenticated users aren't redirected away from login/setup
3. **Protected route detection**: Some route checks may fail

## Fix

### 1. Normalize pathname in AuthGuard (providers.tsx)

Add a helper to strip trailing slashes before route matching:

```javascript
// Normalize pathname by removing trailing slash (except for root "/")
const normalizedPathname = pathname === '/' ? '/' : pathname.replace(/\/$/, '')
```

Then use `normalizedPathname` for all route comparisons:

```javascript
const isProtectedRoute = protectedRoutes.some(route => normalizedPathname === route || normalizedPathname.startsWith(route + '/'))
const isAuthRoute = authRoutes.includes(normalizedPathname)
```

### 2. Fix login page redirect (login/page.tsx)

After successful login, use `window.location.href` instead of `router.push` to ensure a full page reload that picks up the new auth state:

```javascript
await login(email, password)
addToast({ type: 'success', title: 'Welcome back!' })
window.location.href = '/'
```

### 3. Fix setup page redirect (setup/page.tsx)

Same fix - use `window.location.href` for redirects after authentication changes.

## Files to Modify

1. `frontend/src/app/providers.tsx` - Normalize pathname in AuthGuard
2. `frontend/src/app/login/page.tsx` - Use `window.location.href` for post-login redirect
3. `frontend/src/app/setup/page.tsx` - Use `window.location.href` for post-setup redirect

## Validation

After fix:
1. Login → should redirect to dashboard (`/`)
2. Setup → should redirect to dashboard after account creation
3. Clicking nav links → should navigate correctly
4. Refreshing page → should maintain auth state and not redirect to login
