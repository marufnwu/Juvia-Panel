# Components

Components are organized by type:

- `ui/` - Base UI components (shadcn/ui base)
- `layout/` - Layout components (Navbar, Sidebar, etc.)
- `dashboard/` - Dashboard-specific components

## UI Components (shadcn/ui)

These are typically copied from shadcn/ui rather than installed via npm.
See: https://ui.shadcn.com/

Copy components into `src/components/ui/` and customize as needed.

## Layout Components

- `Navbar.tsx` - Top navigation bar
- `Sidebar.tsx` - Side navigation (mobile)
- `CommandPalette.tsx` - Cmd+K command palette
- `NotificationPanel.tsx` - Notification dropdown

## Dashboard Components

- `ResourceCard.tsx` - CPU/RAM/Disk metric cards
- `AppList.tsx` - Running apps list
- `ServiceList.tsx` - Active services list
- `ActivityFeed.tsx` - Recent activity feed
- `QuickActions.tsx` - Quick action buttons