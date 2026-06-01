# Stores

Zustand stores for global state management.

## Stores Overview

- `index.ts` - All store exports
- `auth.ts` - Authentication state (access token, user)
- `theme.ts` - Theme preference (dark/light/system)
- `toast.ts` - Toast notifications
- `commandPalette.ts` - Command palette open state
- `sidebar.ts` - Mobile sidebar state
- `notifications.ts` - Notification panel and unread count

## Usage

```typescript
import { useAuthStore, useToastStore } from '@/stores'

// Auth
const { isAuthenticated, user } = useAuthStore()

// Toast
const { addToast } = useToastStore()
addToast({ type: 'success', title: 'App deployed' })
```

## Server State

Server state (apps, services, metrics) is managed by TanStack Query, not Zustand.
See `src/lib/api.ts` for API client and query hooks.