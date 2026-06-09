# Fix Login Redirect Issue

## Problem
After successful login, the user is redirected to the setup page instead of the dashboard. The login itself succeeds, but the post-login navigation goes wrong.

## Root Cause Analysis

### Root Cause #1: Auth store is not persisted
The `useAuthStore` in `frontend/src/stores/index.ts` does NOT use Zustand's `persist` middleware. Only the `useThemeStore` uses persist.

This means:
- `isAuthenticated`, `accessToken`, and `user` are in-memory only
- On full page reload (e.g. `window.location.href = '/'`), the state is lost
- After login, when the dashboard mounts, it sees no auth and redirects to `/login`
- The login page's setup check then runs and may redirect to `/setup`

### Root Cause #2: `checkUsersExist` returns false on ANY error
In `frontend/src/stores/index.ts` (line 86-100), `checkUsersExist` has overly aggressive error handling:

```javascript
checkUsersExist: async () => {
  try {
    const data = await api.auth.status()
    set({ usersExist: data.users_exist, setupCompleted: data.setup_completed })
    return { usersExist: data.users_exist, setupCompleted: data.setup_completed }
  } catch {
    return { usersExist: false, setupCompleted: false }  // BUG: returns false on network errors too
  }
}
```

This means if the `/api/v1/auth/status` call fails for ANY reason (network blip, timeout, CORS), the function returns `setupCompleted: false`, causing the login page to redirect to `/setup`.

### Root Cause #3: Login redirect chain
1. User submits login on `/login`
2. Login API succeeds → `setAuth` called
3. `window.location.href = '/'` causes full page reload
4. Auth store re-initializes (state lost) → `isAuthenticated: false`
5. Dashboard's useEffect: no auth → `router.push('/login')`
6. Login page mounts → `checkUsersExist()` runs
7. If API call fails or returns unexpected data → redirects to `/setup`

## Solution

### Fix 1: Add persist middleware to auth store
Wrap `useAuthStore` with Zustand's `persist` middleware so auth state survives page reloads.

**File**: `frontend/src/stores/index.ts`

Change from:
```javascript
export const useAuthStore = create<AuthState>((set) => ({...}))
```

To:
```javascript
export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({...}),
    { name: 'panel-auth' }
  )
)
```

### Fix 2: Make `checkUsersExist` error handling safer
Distinguish between "API explicitly says setup not done" and "API call failed".

**File**: `frontend/src/stores/index.ts`

Change the catch block to return `null` or throw, so the caller can decide what to do. Better yet, change the return type to include an error field.

```javascript
checkUsersExist: async () => {
  try {
    const data = await api.auth.status()
    set({ usersExist: data.users_exist, setupCompleted: data.setup_completed })
    return { usersExist: data.users_exist, setupCompleted: data.setup_completed, error: null }
  } catch (error) {
    return { usersExist: true, setupCompleted: true, error } // Assume setup done if can't verify
  }
}
```

**File**: `frontend/src/app/login/page.tsx`

Only redirect to `/setup` if we got a successful response indicating setup is not done:
```javascript
const { setupCompleted, error } = await checkUsersExist()
if (!error && setupCompleted === false) {
  window.location.href = '/setup'
  return
}
setIsChecking(false)
```

### Fix 3: Use `router.push` for client-side navigation
After login, use `router.push('/')` instead of `window.location.href = '/'`. With the persist middleware in place, the auth state will be available on the dashboard.

**File**: `frontend/src/app/login/page.tsx`

Change line 57:
```javascript
// Before
window.location.href = '/'
// After
router.push('/')
```

Also remove the setTimeout delay since it's no longer needed.

## Files to Modify
1. `frontend/src/stores/index.ts` - Add persist middleware to auth store, fix checkUsersExist error handling
2. `frontend/src/app/login/page.tsx` - Use router.push, fix setup check logic
3. (Optional) `frontend/src/app/setup/page.tsx` - Fix setupCompleted check similar to login

## Implementation Steps
1. Modify `frontend/src/stores/index.ts` to add `persist` to `useAuthStore`
2. Modify `frontend/src/stores/index.ts` to fix `checkUsersExist` error handling
3. Modify `frontend/src/app/login/page.tsx` to use `router.push` and fix setup check
4. Build the frontend: `cd frontend && npm run build`
5. Deploy `frontend/out/*` to server: `/opt/panel/ui/out/`
6. Test login flow

## Validation
After deployment:
1. User should be able to log in
2. After successful login, user should land on the dashboard (`/`)
3. Refreshing the page should keep the user logged in
4. Logout should clear auth state
