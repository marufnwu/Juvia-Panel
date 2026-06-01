# Juvia Panel — UI/UX Specification
## Complete Pages, Features & User Flows

**Version:** 1.0  
**Date:** 2026-06-01  
**Status:** Draft  
**Target:** Single-Server Self-Hosted PaaS for Developers

---

## Table of Contents

1. [Design System & Principles](#1-design-system--principles)
2. [Global Navigation & Layout](#2-global-navigation--layout)
3. [Onboarding Flow](#3-onboarding-flow)
4. [Dashboard (Home)](#4-dashboard-home)
5. [Apps Management](#5-apps-management)
6. [App Detail View](#6-app-detail-view)
7. [Create New App Flow](#7-create-new-app-flow)
8. [Services & Databases](#8-services--databases)
9. [Service Detail View](#9-service-detail-view)
10. [Server & Monitoring](#10-server--monitoring)
11. [Terminal](#11-terminal)
12. [File Manager](#12-file-manager)
13. [Backups](#13-backups)
14. [Cron Jobs](#14-cron-jobs)
15. [Firewall](#15-firewall)
16. [Domains & SSL](#16-domains--ssl)
17. [Settings](#17-settings)
18. [Team Management](#18-team-management)
19. [Notifications & Activity](#19-notifications--activity)
20. [Error Pages & Empty States](#20-error-pages--empty-states)
21. [Responsive Behavior](#21-responsive-behavior)
22. [Keyboard Shortcuts](#22-keyboard-shortcuts)

---

## 1. Design System & Principles

### 1.1 Visual Identity
| Element | Specification |
|---------|---------------|
| **Primary color** | `#2563EB` (Blue 600) — actions, links, active states |
| **Success color** | `#16A34A` (Green 600) — healthy, deployed, online |
| **Warning color** | `#D97706` (Amber 600) — deploying, pending, attention |
| **Danger color** | `#DC2626` (Red 600) — failed, error, offline, destructive |
| **Neutral color** | `#6B7280` (Gray 500) — inactive, secondary text |
| **Background** | `#0F172A` (Slate 900) — dark mode default |
| **Surface** | `#1E293B` (Slate 800) — cards, panels, inputs |
| **Border** | `#334155` (Slate 700) — dividers, borders |
| **Text primary** | `#F8FAFC` (Slate 50) — headings, primary content |
| **Text secondary** | `#94A3B8` (Slate 400) — descriptions, metadata |
| **Font family** | `Inter` for UI, `JetBrains Mono` for code/logs/terminal |
| **Border radius** | `8px` for cards, `6px` for buttons/inputs, `4px` for tags |
| **Shadows** | Minimal, subtle glow for focused elements only |

### 1.2 Core Design Principles
1. **Information density over whitespace** — Developers want data. Compact tables, tight spacing, monospace for technical content.
2. **Status at a glance** — Every app, service, and server metric has a color-coded health indicator visible without clicking.
3. **Progressive disclosure** — Simple defaults upfront. Advanced settings (custom Nginx, resource limits, pre-deploy hooks) behind collapsible "Advanced" sections.
4. **Action proximity** — The button to do something is always within one click of the thing you are looking at.
5. **No dead ends** — Every empty state has a primary action button. Every error has a recovery path.
6. **Dark mode default** — Developers prefer dark interfaces. Light mode is optional, not primary.

### 1.3 Iconography
- **Library:** Lucide React (consistent, lightweight, developer-friendly)
- **Size:** 16px inline, 20px for navigation, 24px for empty states
- **Rules:**
  - Icons are always accompanied by text labels in navigation
  - Status icons use color (green dot, red dot, amber spinner)
  - Action icons in tables show on hover to reduce visual noise

### 1.4 Animation & Motion
- **Transitions:** 150ms ease for hover states, 200ms ease-in-out for modals/drawers
- **Loading:** Skeleton screens for initial data load, spinners for actions, progress bars for deployments
- **Real-time indicators:** Subtle pulse animation for "deploying" status, steady glow for "live"
- **No motion for:** Status changes that should be instant (health check failures, errors)

---

### 1.5 Technology Stack — Definitive

The UI is built with the following stack. No alternatives, no "or" choices.

| Layer | Technology | Version | Purpose |
|-------|-----------|---------|---------|
| **Framework** | Next.js | 14 (App Router) | React framework with static export for production. No Node.js runtime required in production. |
| **Language** | TypeScript | 5.3+ | Type safety across all components and API client. |
| **Styling** | Tailwind CSS | 3.4+ | Utility-first CSS with custom design tokens matching the color system above. |
| **Components** | shadcn/ui | Latest | Accessible, composable components built on Radix UI primitives. Copy-paste, not installed via npm. |
| **Global State** | Zustand | 4.5+ | Lightweight global state for UI state (theme, sidebar, command palette). |
| **Server State** | TanStack Query (React Query) | 5.0+ | Data fetching, caching, background refetching, and optimistic updates. |
| **Real-time** | Native WebSocket | Browser API | Connects to Go API WebSocket hub at `/api/v1/stream`. No Socket.IO, no abstraction library. |
| **Terminal** | xterm.js | 5.3+ | Terminal emulator in the browser. Backend is ttyd (Go/C) which the UI connects to via WebSocket. |
| **Code Editor** | Monaco Editor | 0.45+ | VS Code core editor for the file manager. Package: `@monaco-editor/react`. |
| **Icons** | Lucide React | Latest | Consistent, lightweight SVG icons. One icon library for the entire UI. |
| **Charts** | Recharts | 2.10+ | Composable React charts for server metrics on the Dashboard and Server pages. |
| **HTTP Client** | Native `fetch` | Browser API | No Axios, no wrapper. Uses TanStack Query for request lifecycle. |
| **Build Output** | Static Export | `output: 'export'` | Next.js builds to static HTML/JS/CSS. Caddy serves these files. No Next.js server in production. |

**Backend Connection:**
- The UI talks to the **Go API** (Gin framework) running on `127.0.0.1:2053`
- Caddy reverse-proxies `/api/*` and `/api/v1/stream` to the API server
- All other routes serve the static Next.js files
- Authentication: JWT access token in memory, refresh token in HTTP-only cookie

**Why This Stack:**
- **Single framework:** Next.js handles routing, static generation, and API client structure
- **No runtime dependency:** Static export means Caddy serves files directly. If Next.js has a vulnerability, production is unaffected.
- **Type safety:** TypeScript + Go API = end-to-end type safety via shared types or generated client
- **Lightweight:** No Redux, no GraphQL, no heavy abstraction layers

---

## 2. Global Navigation & Layout

### 2.1 Shell Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  [Logo]  Dashboard    Apps    Services    Server    [Search] [Bell] [User] │  ← Top Bar (56px)
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  [Breadcrumb: Home > Apps > api-prod]                               │  ← Breadcrumb Bar (40px)
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                                                             │   │  ← Main Content Area
│  │                    Page Content                             │   │     (scrollable)
│  │                                                             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 Top Navigation Bar
| Item | Icon | Behavior |
|------|------|----------|
| **Logo** | Custom mark | Click → Dashboard. Hover shows version number. |
| **Dashboard** | LayoutDashboard | Click → Dashboard. Active when on Dashboard. |
| **Apps** | Boxes | Click → Apps list. Shows badge with active app count. |
| **Services** | Database | Click → Services list. Shows badge with active service count. |
| **Server** | Server | Click → Server monitoring. Shows red dot if server has alerts. |
| **Search** | Search | Click or `Cmd+K` opens command palette. |
| **Notifications** | Bell | Badge shows unread count. Click opens dropdown panel. |
| **User menu** | Avatar | Dropdown: Profile, Team, Settings, API Keys, Documentation, Sign Out. |

### 2.3 Command Palette (Cmd+K)
Universal search and action launcher. Opens as a centered modal overlay.

**Sections:**
- **Recent** — Last 5 visited pages/actions
- **Apps** — Type app name → jump to app detail
- **Services** — Type service name → jump to service detail
- **Actions** — "Create app", "Deploy api-prod", "Restart redis", "Open terminal"
- **Settings** — "Change password", "Add team member", "View logs"

**Keyboard navigation:**
- `↑` `↓` to navigate results
- `Enter` to select
- `Esc` to close
- `Tab` to switch between sections

### 2.4 Notification Panel
Dropdown from bell icon. Shows recent events with timestamps.

| Element | Behavior |
|---------|----------|
| **Unread dot** | Blue dot on bell icon. Clears when panel opened. |
| **Event item** | Icon + title + timestamp + app/service link. Click → navigate to relevant page. |
| **Mark all read** | Button at top right of panel. |
| **View all** | Link at bottom → Activity log page. |

**Event types displayed:**
- Deployment started/failed/succeeded
- SSL certificate renewed/expiring
- Service backup completed/failed
- Server resource alert (CPU > 80%)
- Team member invited/accepted

---

## 3. Onboarding Flow

### 3.1 First-Run Wizard
After installation script completes, the user opens the panel URL.

**Step 1: Welcome Screen**
```
┌──────────────────────────────────────────────┐
│                                              │
│     [Logo Large]                             │
│                                              │
│     Welcome to Panel                         │
│     Your server is ready. Let's set it up.   │
│                                              │
│     [Get Started →]                          │
│                                              │
└──────────────────────────────────────────────┘
```

**Step 2: Create Admin Account**
- Username (default: `admin`)
- Email address (for alerts and notifications)
- Password (with strength meter: weak/fair/strong)
- Confirm password
- Checkbox: "Enable 2FA" (optional, can skip)
- If 2FA checked: show QR code, require TOTP verification

**Step 3: Server Configuration**
- Server name (default: hostname)
- Timezone selector (detected from browser, editable)
- Automatic security updates: [Enabled / Disabled]
- Backup destination: [Local only / Connect S3]
- If S3: fields for endpoint, bucket, key, secret, region

**Step 4: Domain Setup (Optional)**
- "Do you have a domain for the panel?"
- Input: `panel.example.com` or skip
- If provided: validate DNS A record points to server IP
- Auto-provision Let's Encrypt SSL for panel domain

**Step 5: Completion**
```
┌──────────────────────────────────────────────┐
│     🎉 You're all set!                       │
│                                              │
│     Server: my-vps-01                        │
│     Panel URL: https://panel.example.com     │
│     Apps ready: 0                            │
│                                              │
│     [Create Your First App →]                │
│     [Go to Dashboard]                        │
│                                              │
└──────────────────────────────────────────────┘
```

### 3.2 Post-Onboarding Empty State Dashboard
If the user clicks "Go to Dashboard" without creating apps:
- Dashboard shows empty state cards with "Create App" and "Create Service" CTAs
- Quick-start guide banner: "Deploy your first app in 3 minutes"

---

## 4. Dashboard (Home)

### 4.1 Purpose
The command center. Shows server health, recent activity, and quick access to everything.

### 4.2 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Dashboard                                          [+ New App]     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐       │
│  │  CPU Usage      │ │  RAM Usage      │ │  Disk Usage     │       │
│  │    34%          │ │    62%          │ │    45%          │       │
│  │  [Sparkline]    │ │  [Sparkline]    │ │  [Sparkline]    │       │
│  │  4 cores        │ │  4.8 / 8 GB     │ │  45 / 100 GB    │       │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘       │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Network I/O                    Uptime: 14 days 3 hours      │   │
│  │ [Line chart: 24h inbound/outbound]                           │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌──────────────────────────────┐  ┌────────────────────────────┐ │
│  │ Running Apps (6)            │  │ Active Services (4)        │ │
│  │ ──────────────────────────  │  │ ──────────────────────────   │ │
│  │ ● api-prod      Node.js     │  │ ● main-pg    PostgreSQL    │ │
│  │ ● web-client    React       │  │ ● redis      Redis         │ │
│  │ ● worker        Python      │  │ ● minio      MinIO         │ │
│  │ ● blog          Ghost       │  │                            │ │
│  │ ● api-staging   Node.js     │  │                            │ │
│  │ ● docs          Static      │  │                            │ │
│  │                             │  │                            │ │
│  │ [View All Apps →]           │  │ [View All Services →]      │ │
│  └──────────────────────────────┘  └────────────────────────────┘ │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Recent Activity                                              │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ 🟢 api-prod deployed successfully    2 min ago    [View]   │   │
│  │ 🟢 SSL renewed for example.com       1 hr ago     [View]   │   │
│  │ 🟡 main-pg backup in progress        1 hr ago     [View]   │   │
│  │ 🔴 worker failed health check        3 hrs ago    [View]   │   │
│  │ 🟢 web-client deployed               5 hrs ago     [View]   │   │
│  │                                                             │   │
│  │ [View Full Activity Log →]                                   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Quick Actions                                                │   │
│  │ [Open Terminal] [View Logs] [Check Updates] [Restart Server]  │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.3 Features
| Feature | Description |
|---------|-------------|
| **Resource cards** | CPU, RAM, Disk with current percentage, sparkline (last 60 minutes), and absolute values. Click any card → Server monitoring page with that metric focused. |
| **Network chart** | 24-hour line chart showing inbound/outbound traffic. Auto-updates every 30 seconds. |
| **Uptime display** | Days, hours, minutes since last reboot. Tooltip shows exact boot timestamp. |
| **App list** | Up to 6 most recently active apps. Shows status dot, name, runtime. Click row → App detail. |
| **Service list** | Up to 4 active services. Shows status dot, name, type. Click row → Service detail. |
| **Activity feed** | Last 5 events with icon, description, relative timestamp, and link. Auto-refreshes. |
| **Quick actions** | One-click buttons for common tasks. "Restart Server" has confirmation modal. |

### 4.4 Empty State (No Apps)
```
┌─────────────────────────────────────────────────────────────────────┐
│  Welcome to your server!                                            │
│                                                                     │
│  ┌────────────────────────┐  ┌────────────────────────┐            │
│  │    [App Icon]          │  │    [Database Icon]     │            │
│  │                        │  │                        │            │
│  │  No apps yet           │  │  No services yet       │            │
│  │                        │  │                        │            │
│  │  Deploy your first     │  │  Add a database for    │            │
│  │  application from Git  │  │  your apps             │            │
│  │                        │  │                        │            │
│  │  [Create App →]        │  │  [Create Service →]    │            │
│  └────────────────────────┘  └────────────────────────┘            │
│                                                                     │
│  💡 Quick Start: Deploy a Node.js app from GitHub in 3 minutes    │
│     [Show me how]                                                   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5. Apps Management

### 5.1 Purpose
List, search, filter, and manage all deployed applications.

### 5.2 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Apps                                    [+ New App]  [Import]      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  [Search apps...]  [Filter ▼]  [Runtime ▼]  [Status ▼]            │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Status │ Name        │ Runtime   │ Domain              │ Updated │
│  │ ───────────────────────────────────────────────────────────│   │
│  │ 🟢     │ api-prod    │ Node 20   │ api.example.com     │ 2m    │
│  │ 🟢     │ web-client  │ React     │ app.example.com     │ 5h    │
│  │ 🟡     │ worker      │ Python 3  │ —                   │ 1d    │
│  │ 🟢     │ blog        │ Ghost     │ blog.example.com    │ 3d    │
│  │ 🔴     │ api-staging │ Node 20   │ staging.example.com │ 1w    │
│  │ ⚪     │ old-project │ Static    │ —                   │ 2mo   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  Showing 6 of 6 apps                                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.3 Features
| Feature | Description |
|---------|-------------|
| **Search** | Real-time filter by app name, domain, or Git repo URL. Debounced 200ms. |
| **Filter dropdown** | Filter by: All, Running, Stopped, Failed, Deploying |
| **Runtime dropdown** | Filter by runtime: Node.js, Python, Go, PHP, Ruby, Static, Docker |
| **Status column** | Color dot: 🟢 live/healthy, 🟡 deploying/restarting, 🔴 failed/unhealthy, ⚪ stopped |
| **Name column** | Clickable link to app detail. Hover shows quick actions: Deploy, Restart, Stop, Logs. |
| **Domain column** | Primary domain or "—" if none. Click → open domain in new tab. |
| **Updated column** | Relative time of last deployment. Tooltip shows exact timestamp. |
| **Bulk actions** (future) | Checkbox per row, bulk restart/stop/delete |
| **Sort** | Click column headers to sort by name, status, updated time |

### 5.4 Row Hover Actions
When hovering a row, action buttons appear on the right:
- **Deploy** (arrow-up-circle) — triggers new deployment
- **Restart** (refresh-cw) — restarts app container
- **Stop** (square) — stops app (if running)
- **Logs** (file-text) — opens app detail on Logs tab
- **More** (three dots) — dropdown: Settings, Environment, Delete

### 5.5 Empty State
```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│              [Large App Icon]                                       │
│                                                                     │
│              No apps yet                                            │
│                                                                     │
│              Deploy your first application from GitHub,             │
│              GitLab, or upload files directly.                      │
│                                                                     │
│              [Create App →]  [Import from Docker Compose →]         │
│                                                                     │
│              💡 Or try a one-click template:                        │
│              [WordPress] [Ghost] [Plausible] [Nextcloud]              │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 6. App Detail View

### 6.1 Purpose
The single most important page. Manage everything about one application.

### 6.2 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Apps > api-prod                                                    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  🟢 api-prod          [Deploy] [Restart] [Stop] [⋮]        │   │
│  │  Node.js 20 • api.example.com • Last deployed 2 min ago     │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [Overview] [Deployments] [Logs] [Environment] [Settings]           │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    TAB CONTENT                              │   │
│  │                                                             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 6.3 Header Actions
| Button | Behavior |
|--------|----------|
| **Deploy** | Opens deploy modal: select branch/tag, view last commit, confirm. |
| **Restart** | One-click restart. Shows confirmation toast. Spinner on button during restart. |
| **Stop** | Stops the app container. Button changes to "Start" when stopped. |
| **More (⋮)** | Dropdown: Clone app, Create staging environment, Export compose, Delete app |

### 6.4 Tab: Overview
```
┌─────────────────────────────────────────────────────────────────────┐
│  Overview                                                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────┐  ┌─────────────────────────────────────┐ │
│  │ App Info             │  │ Resource Usage (Last 24h)          │ │
│  │ ───────────────────  │  │ ───────────────────────────────────  │ │
│  │ Status: 🟢 Healthy   │  │ [CPU line chart]  CPU: 12%         │ │
│  │ Runtime: Node.js 20  │  │ [RAM line chart]  RAM: 256 MB      │ │
│  │ Build: 45s           │  │ [Network chart]   Net: 2.4 MB/s    │ │
│  │ Uptime: 14 days      │  │                                     │ │
│  │                      │  │                                     │ │
│  │ GitHub:              │  │                                     │ │
│  │ user/repo            │  │                                     │ │
│  │ Branch: main         │  │                                     │ │
│  │ Commit: abc1234      │  │                                     │ │
│  │                      │  │                                     │ │
│  │ [View on GitHub →]   │  │                                     │ │
│  └──────────────────────┘  └─────────────────────────────────────┘ │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Domains                                                      │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ Primary: api.example.com 🟢 SSL valid until 2026-09-01      │   │
│  │ Aliases: www.api.example.com, api.example.com              │   │
│  │ [Add Domain]                                                │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Connected Services                                           │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ ● main-pg (PostgreSQL)  ● redis-cache (Redis)              │   │
│  │ [Connect Service →]                                         │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Persistent Volumes                                           │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ /app/data → /var/panel/apps/api-prod/volumes/data (2.3 GB) │   │
│  │ /app/uploads → /var/panel/apps/api-prod/volumes/uploads    │   │
│  │ [Add Volume]                                                │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 6.5 Tab: Deployments
```
┌─────────────────────────────────────────────────────────────────────┐
│  Deployments                                      [Deploy Now]      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Status │ Commit    │ Branch │ Author     │ Duration │ Time  │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ 🟢     │ abc1234   │ main   │ john       │ 45s      │ 2m    │   │
│  │ 🟢     │ def5678   │ main   │ john       │ 38s      │ 1h    │   │
│  │ 🟢     │ ghi9012   │ main   │ sarah      │ 52s      │ 3h    │   │
│  │ 🔴     │ jkl3456   │ dev    │ john       │ 12s      │ 5h    │   │
│  │ 🟢     │ mno7890   │ main   │ sarah      │ 41s      │ 1d    │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  Click any row to view build logs.                                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Row click behavior:** Opens a drawer from the right showing full build logs for that deployment, with option to "Rollback to this deployment" or "Redeploy this commit".

**Failed deployment row:**
- Red background tint on row
- Expandable inline to show error summary
- "View logs" button opens full build log with error highlighted

### 6.6 Tab: Logs
```
┌─────────────────────────────────────────────────────────────────────┐
│  Logs                                               [Download]      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  [App Logs ▼] [stdout] [stderr] [Auto-scroll ▼] [Search...]       │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ 2024-06-01 12:34:56  GET /api/users 200 45ms              │   │
│  │ 2024-06-01 12:34:57  POST /api/login 200 120ms            │   │
│  │ 2024-06-01 12:34:58  ERROR: Connection timeout to redis   │   │
│  │ 2024-06-01 12:34:59  GET /health 200 2ms                  │   │
│  │ 2024-06-01 12:35:01  Worker job completed: email-send     │   │
│  │ ...                                                         │   │
│  │                                                             │   │
│  │ [Streaming...]                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Features:**
| Feature | Behavior |
|---------|----------|
| **Log type selector** | App logs (stdout/stderr), Nginx access logs, Nginx error logs |
| **Stream toggle** | stdout, stderr, or both. Default: both. |
| **Auto-scroll** | On by default. Follows new log entries. Pause when user scrolls up. |
| **Search** | Filter logs by keyword. Highlights matches. Works on buffered logs. |
| **Download** | Download current log buffer as `.log` file |
| **Timestamp** | ISO 8601 format. Toggle relative time ("2s ago") in settings. |
| **Log highlighting** | Errors in red, warnings in amber, success in green |
| **WebSocket streaming** | Real-time updates. Reconnects automatically on disconnect. |

### 6.7 Tab: Environment
```
┌─────────────────────────────────────────────────────────────────────┐
│  Environment Variables                              [Import .env]     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ KEY                    │ VALUE              │ Actions        │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ NODE_ENV               │ production         │ [Edit] [Del]   │   │
│  │ DATABASE_URL           │ ************       │ [Edit] [Del]   │   │
│  │ REDIS_URL              │ redis://redis:6379 │ [Edit] [Del]   │   │
│  │ API_SECRET_KEY         │ ************       │ [Edit] [Del]   │   │
│  │ PORT                   │ 3000               │ [Edit] [Del]   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [+ Add Variable]                                                   │
│                                                                     │
│  ⚠️ Changes require app restart to take effect.                     │
│                                                                     │
│  [Restart App with New Variables]                                   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Features:**
| Feature | Behavior |
|---------|----------|
| **Secret masking** | Values containing "SECRET", "KEY", "PASSWORD", "TOKEN" are masked as `************`. Click eye icon to reveal. |
| **Bulk import** | Paste `.env` file content. Parses KEY=VALUE pairs. Validates for duplicates. |
| **Add variable** | Inline form: key input, value input (textarea for multiline), secret toggle. |
| **Edit** | Inline edit. Save or cancel. |
| **Delete** | Confirm with "Delete" button (not just OK/Cancel). |
| **Restart prompt** | Banner appears when unsaved changes exist. One-click restart applies changes. |

### 6.8 Tab: Settings
```
┌─────────────────────────────────────────────────────────────────────┐
│  Settings                                                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ General                                                      │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ App Name: [api-prod          ]                                │   │
│  │ Primary Domain: [api.example.com] [Validate DNS]             │   │
│  │ Force HTTPS: [✓]                                             │   │
│  │                                                             │   │
│  │ Build & Deploy                                               │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ Git Repository: [https://github.com/user/repo]             │   │
│  │ Branch: [main ▼]                                             │   │
│  │ Auto-deploy on push: [✓]                                    │   │
│  │ Build Command: [npm run build          ]                     │   │
│  │ Start Command: [npm start            ]                       │   │
│  │                                                             │   │
│  │ [Advanced ▼]                                                  │   │
│  │   Health Check: /health                                       │   │
│  │   Health Interval: 30s                                        │   │
│  │   Pre-deploy Hook: [npm run migrate      ]                   │   │
│  │   Post-deploy Hook: [npm run notify      ]                   │   │
│  │   Resource Limits: CPU [2] cores, RAM [512] MB               │   │
│  │   Custom Nginx Config: [Edit →]                              │   │
│  │                                                             │   │
│  │ Danger Zone                                                  │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ [Delete App] — This cannot be undone. All data will be lost. │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Danger Zone behavior:**
- Delete button is gray, not red, until clicked
- First click: button turns red, text changes to "Confirm Delete"
- Second click: opens modal requiring app name typed for confirmation
- Modal shows: "This will delete the app container, all volumes, domains, and SSL certificates. This action cannot be undone."

---

## 7. Create New App Flow

### 7.1 Purpose
The primary user journey. Must be fast, confident, and never leave the user stuck.

### 7.2 Step-by-Step Wizard

**Step 1: Choose Source**
```
┌─────────────────────────────────────────────────────────────────────┐
│  Create New App                                     Step 1 of 4      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  How do you want to deploy your app?                                │
│                                                                     │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐     │
│  │   [Git Icon]    │  │  [Upload Icon]  │  │  [Docker Icon]  │     │
│  │                 │  │                 │  │                 │     │
│  │  Git Repository │  │  Upload Files   │  │  Docker Compose │     │
│  │                 │  │                 │  │                 │     │
│  │  GitHub, GitLab,│  │  ZIP or tar   │  │  Custom Docker  │     │
│  │  Bitbucket      │  │  archive        │  │  configuration  │     │
│  │                 │  │                 │  │                 │     │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘     │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Or choose a template:                                       │   │
│  │ [WordPress] [Ghost] [Next.js] [Laravel] [Django] [Plausible]│   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Step 2: Configure Source**

*If Git selected:*
```
┌─────────────────────────────────────────────────────────────────────┐
│  Create New App                                     Step 2 of 4      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Git Repository                                                     │
│  [https://github.com/user/repo                    ]                 │
│                                                                     │
│  Branch                                                             │
│  [main ▼]  (auto-detected from repo)                              │
│                                                                     │
│  Build Strategy                                                     │
│  [Auto-detect ▼]  Detected: Node.js (package.json found)          │
│                                                                     │
│  Options: Nixpacks, Dockerfile, Static Site, Custom               │
│                                                                     │
│  [Test Connection] → Shows green checkmark if repo is accessible   │
│                                                                     │
│  [Back]  [Continue →]                                             │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

*If Upload selected:*
```
│  Drop files here or click to upload                                 │
│  [Browse files]                                                     │
│  Supported: .zip, .tar.gz, .tar                                    │
│                                                                     │
│  Build Strategy: [Auto-detect ▼]                                    │
```

*If Docker Compose selected:*
```
│  Paste docker-compose.yml content or upload file                    │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ version: "3.8"                                               │   │
│  │ services:                                                    │   │
│  │   app: ...                                                   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [Validate] → Checks syntax and extracts service names               │
```

**Step 3: Basic Configuration**
```
┌─────────────────────────────────────────────────────────────────────┐
│  Create New App                                     Step 3 of 4      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  App Name                                                           │
│  [my-app             ]  (lowercase, alphanumeric, hyphens)        │
│                                                                     │
│  Domain (optional)                                                │
│  [my-app.example.com ]  [Validate DNS]                             │
│  Skip for now → panel will generate a subdomain                    │
│                                                                     │
│  Environment Variables (optional)                                   │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ KEY                    │ VALUE                              │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ [NODE_ENV            ] │ [production        ]                 │   │
│  │ [PORT                ] │ [3000              ]                 │   │
│  └─────────────────────────────────────────────────────────────┘   │
│  [+ Add Variable]                                                   │
│                                                                     │
│  [Back]  [Continue →]                                               │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Step 4: Review & Deploy**
```
┌─────────────────────────────────────────────────────────────────────┐
│  Create New App                                     Step 4 of 4      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Review your configuration                                         │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Source:        GitHub — user/repo (main branch)              │   │
│  │ Build:         Node.js 20 via Nixpacks                      │   │
│  │ Name:          my-app                                        │   │
│  │ Domain:        my-app.panel.example.com (auto-generated)     │   │
│  │ Env vars:      2 variables set                               │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [✓] Auto-deploy on future pushes to main branch                   │
│                                                                     │
│  [Back]  [Deploy App →]                                            │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 7.3 Post-Create Flow
After clicking "Deploy App":
1. Modal closes, user is redirected to App Detail → Deployments tab
2. A new deployment row appears with 🟡 "Building" status
3. Build logs stream in real-time in an expandable panel
4. Progress indicator shows: Cloning → Building → Starting → Health Check → Live
5. On success: status changes to 🟢, notification appears, domain link becomes clickable
6. On failure: status changes to 🔴, error summary shown, "View logs" button highlighted

### 7.4 Error Handling During Creation
| Error | UI Response |
|-------|-------------|
| Invalid Git URL | Inline error: "Repository not found or not accessible. Check the URL and ensure it's public or add SSH key." |
| DNS not pointing to server | Warning banner: "Domain DNS does not point to this server. SSL will be provisioned once DNS is updated." |
| Build fails | Deployment tab shows red status. Build log highlights the error line. Suggested fix based on error type. |
| Port conflict | Auto-increment port (3000 → 3001). Notify user in app settings. |
| Name already exists | Inline error: "An app with this name already exists." Suggest alternatives. |

---

## 8. Services & Databases

### 8.1 Purpose
Manage databases, caches, and other backing services.

### 8.2 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Services                                          [+ New Service]  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  [Search services...]  [Type ▼]  [Status ▼]                       │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Status │ Name        │ Type        │ Apps Using │ Size   │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ 🟢     │ main-pg     │ PostgreSQL  │ 3 apps     │ 2.3 GB │   │
│  │ 🟢     │ redis-cache │ Redis       │ 2 apps     │ 128 MB │   │
│  │ 🟢     │ minio-store │ MinIO       │ 1 app      │ 5.1 GB │   │
│  │ 🟢     │ analytics-db│ PostgreSQL  │ 1 app      │ 890 MB │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 8.3 Features
| Feature | Description |
|---------|-------------|
| **Type filter** | PostgreSQL, MySQL, MariaDB, MongoDB, Redis, Memcached, MinIO, Custom |
| **Apps using** | Clickable count → shows list of connected apps |
| **Size** | Current data directory size. Updates every 5 minutes. |
| **Row hover** | Quick actions: Connect, Backup, Restart, Logs, Delete |
| **Status** | 🟢 running, 🟡 starting, 🔴 failed, ⚪ stopped |

### 8.4 Empty State
```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│              [Database Icon]                                        │
│                                                                     │
│              No services yet                                        │
│                                                                     │
│              Add a database or cache for your apps.                 │
│              One-click setup with automatic backups.              │
│                                                                     │
│              [Create Service →]                                   │
│                                                                     │
│              💡 Popular choices:                                    │
│              [PostgreSQL] [MySQL] [Redis] [MongoDB]                 │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 9. Service Detail View

### 9.1 Purpose
Manage a single database or service instance.

### 9.2 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Services > main-pg                                                 │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  🟢 main-pg         [Restart] [Backup] [Connect] [⋮]        │   │
│  │  PostgreSQL 15 • 3 apps connected • 2.3 GB data           │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [Overview] [Backups] [Logs] [Settings]                            │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    TAB CONTENT                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 9.3 Tab: Overview
```
┌─────────────────────────────────────────────────────────────────────┐
│  Overview                                                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────┐  ┌─────────────────────────────────────┐ │
│  │ Service Info         │  │ Connection Details                   │ │
│  │ ───────────────────  │  │ ───────────────────────────────────  │ │
│  │ Type: PostgreSQL 15  │  │ Host: localhost                      │ │
│  │ Status: 🟢 Running   │  │ Port: 5432                           │ │
│  │ Uptime: 14 days      │  │ Database: main-pg                    │ │
│  │ Data size: 2.3 GB    │  │ Username: main-pg-user               │ │
│  │                      │  │ Password: [••••••••] [Copy] [Show]   │ │
│  │                      │  │                                      │ │
│  │                      │  │ Connection String:                   │ │
│  │                      │  │ postgres://user:pass@localhost:5432  │ │
│  │                      │  │ [Copy] [Test Connection]             │ │
│  └──────────────────────┘  └─────────────────────────────────────┘ │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Connected Apps                                               │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ ● api-prod    │ ● web-client    │ ● worker                    │   │
│  │ [Manage Connections →]                                        │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Resource Usage                                               │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ [CPU sparkline]  CPU: 5%                                    │   │
│  │ [RAM sparkline]  RAM: 256 MB                                │   │
│  │ [Connections]    Active: 12 / Max: 100                      │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [Open Adminer/pgAdmin]  [View Database Users]  [Run Query]        │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Connection details:**
- Password is masked by default. Click eye to reveal.
- "Copy" button copies to clipboard with toast confirmation.
- "Test Connection" runs a quick `psql` or `mysql` command and shows green checkmark or red error.

### 9.4 Tab: Backups
```
┌─────────────────────────────────────────────────────────────────────┐
│  Backups                                         [Backup Now]     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Backup Schedule: [Daily at 02:00 ▼]  Retention: [7 days ▼]       │
│  Destination: [Local + S3 (my-backup-bucket) ▼]                    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Status │ Size    │ Type     │ Location  │ Time    │ Actions   │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ 🟢     │ 234 MB  │ Auto     │ S3        │ 2h ago  │ [Restore] │   │
│  │ 🟢     │ 231 MB  │ Auto     │ S3        │ 1d ago  │ [Restore] │   │
│  │ 🟢     │ 228 MB  │ Auto     │ Local     │ 2d ago  │ [Restore] │   │
│  │ 🟢     │ 225 MB  │ Manual   │ S3        │ 5d ago  │ [Restore] │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [Restore] opens confirmation: "This will overwrite current data.   │
│   A backup of current state will be created first."               │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 9.5 Tab: Logs
Same pattern as App Logs, but shows database service logs (PostgreSQL logs, Redis logs, etc.).

### 9.6 Tab: Settings
- Service name
- Version (with upgrade option if newer version available)
- Port
- Resource limits (CPU, RAM)
- Advanced: custom config flags, replication settings (future)
- Danger Zone: delete service (with confirmation)

---

## 10. Server & Monitoring

### 10.1 Purpose
Monitor server health, manage system resources, and perform maintenance.

### 10.2 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Server                                                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ my-vps-01                                                   │   │
│  │ Ubuntu 24.04 LTS • 4 CPU • 8 GB RAM • 100 GB SSD             │   │
│  │ Uptime: 14 days 3 hours                                     │   │
│  │ Panel version: 1.2.3  [Check for Updates]                   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [Overview] [Processes] [Disks] [Network] [Updates] [Firewall]     │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    TAB CONTENT                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 10.3 Tab: Overview
```
┌─────────────────────────────────────────────────────────────────────┐
│  Overview                                                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐       │
│  │  CPU            │ │  RAM            │ │  Disk           │       │
│  │  [Donut: 34%]   │ │  [Donut: 62%]   │ │  [Donut: 45%]   │       │
│  │  1.36 / 4 cores │ │  4.96 / 8 GB    │ │  45 / 100 GB    │       │
│  │  [1h] [6h] [24h]│  [1h] [6h] [24h]│  [1h] [6h] [24h]│       │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘       │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Load Average                                                 │   │
│  │ 1 min: 0.45  │  5 min: 0.52  │  15 min: 0.38               │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Top Processes by CPU                                         │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ PID   │ Name        │ CPU% │ RAM% │ User     │ Time         │   │
│  │ 1234  │ node        │ 12%  │ 8%   │ app_001  │ 2d 04:12     │   │
│  │ 5678  │ postgres    │ 5%   │ 15%  │ postgres │ 14d 00:00    │   │
│  │ 9012  │ nginx       │ 2%   │ 1%   │ www-data │ 14d 00:00    │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Alerts & Thresholds                                          │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ CPU > 80% for 5 min: [Notify ▼]  [Email: admin@example.com]│   │
│  │ RAM > 90%: [Notify ▼]  [Slack webhook]                      │   │
│  │ Disk > 85%: [Notify + Auto-cleanup ▼]                       │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Time range buttons:** Toggle between 1h, 6h, 24h, 7d views. Charts update with smooth transition.

### 10.4 Tab: Processes
- Full process tree with search
- Filter by user, CPU usage, memory usage
- Kill process button (with confirmation)
- Auto-refresh every 5 seconds (toggleable)

### 10.5 Tab: Disks
- Disk usage per mount point
- I/O statistics (read/write throughput)
- Largest directories (disk usage analyzer)
- Cleanup suggestions (Docker prune, log rotation)

### 10.6 Tab: Network
- Bandwidth usage (inbound/outbound)
- Active connections table
- Open ports table
- Firewall rule summary

### 10.7 Tab: Updates
- Available OS security updates list
- One-click "Install Security Updates" button
- Panel update availability
- Changelog display
- "Update Panel" button with rollback option

---

## 11. Terminal

### 11.1 Purpose
Web-based terminal for running commands on the server or inside containers.

### 11.2 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Terminal                                                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  [Host Server ▼]  [api-prod ▼]  [web-client ▼]  [main-pg ▼]       │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ root@my-vps-01:~$ ls -la                                    │   │
│  │ total 128                                                   │   │
│  │ drwxr-xr-x  4 root root  4096 Jun 01 12:00 .              │   │
│  │ drwxr-xr-x 20 root root  4096 May 15 08:30 ..             │   │
│  │ -rw-r--r--  1 root root  2200 May 20 10:15 .bashrc        │   │
│  │ root@my-vps-01:~$ █                                         │   │
│  │                                                             │   │
│  │                                                             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [New Tab]  [Copy]  [Paste]  [Clear]  [Full Screen]               │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 11.3 Features
| Feature | Behavior |
|---------|----------|
| **Target selector** | Dropdown to switch between host server or any running app/service container |
| **Multiple tabs** | Open multiple terminals side by side or in tabs |
| **Copy/paste** | Right-click menu or keyboard shortcuts (Ctrl+C/V, Ctrl+Shift+C/V for terminal) |
| **Full screen** | Toggle full-screen terminal mode (F11) |
| **Session persistence** | Terminal session survives page refresh (reconnects automatically) |
| **Command history** | Up/down arrow navigates command history |
| **Auto-complete** | Tab completion for file paths and commands |
| **Color themes** | Default: dark. Options: light, solarized, monokai |

---

## 12. File Manager

### 12.1 Purpose
Browse, edit, upload, and manage files without leaving the browser.

### 12.2 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  File Manager                                                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  [Host Server ▼]  [api-prod ▼]  [web-client ▼]                     │
│                                                                     │
│  /var/panel/apps/api-prod/  [Breadcrumb navigation]                │
│                                                                     │
│  [Upload] [New Folder] [New File] [Download] [Delete] [Permissions] │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ [☐] Name          │ Size    │ Modified     │ Permissions   │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ 📁 src            │ —       │ Jun 01 10:00 │ drwxr-xr-x    │   │
│  │ 📁 public         │ —       │ Jun 01 10:00 │ drwxr-xr-x    │   │
│  │ 📁 node_modules   │ —       │ Jun 01 10:00 │ drwxr-xr-x    │   │
│  │ 📄 package.json   │ 2.3 KB  │ Jun 01 10:00 │ -rw-r--r--    │   │
│  │ 📄 server.js      │ 5.1 KB  │ Jun 01 10:00 │ -rw-r--r--    │   │
│  │ 📄 .env           │ 512 B   │ Jun 01 10:00 │ -rw-------    │   │
│  │ 📄 README.md      │ 1.2 KB  │ Jun 01 10:00 │ -rw-r--r--    │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  7 items, 2 folders, 5 files                                         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 12.3 Features
| Feature | Behavior |
|---------|----------|
| **Target selector** | Browse host filesystem or any app/service container |
| **Breadcrumb** | Click any path segment to navigate up. Editable path bar. |
| **Upload** | Drag & drop or file picker. Multiple files allowed. Progress bar. |
| **New folder/file** | Inline creation with name input. Default permissions 755/644. |
| **Download** | Single file or ZIP of selected items. |
| **Delete** | Move to trash (recoverable for 24h) or permanent delete. Confirmation. |
| **Permissions** | Modal with owner/group/others checkboxes or octal input. |
| **Edit** | Click file → opens code editor modal with syntax highlighting. Save or discard. |
| **Archive** | Extract .zip, .tar.gz, .tar. Create archive from selected files. |
| **Search** | Search filenames within current directory and subdirectories. |
| **Image preview** | Click image → lightbox preview. |
| **Bulk select** | Checkbox per row. Shift-click for range select. |

**Code editor:**
- Monaco Editor (VS Code core) or CodeMirror 6
- Syntax highlighting for 50+ languages
- Line numbers, minimap, find/replace
- Auto-save draft, manual save to apply

---

## 13. Backups

### 13.1 Purpose
Centralized backup management for all apps and services.

### 13.2 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Backups                                                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Backup Settings                                              │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ Default Schedule: [Daily at 02:00 UTC ▼]                   │   │
│  │ Default Retention: [7 days ▼]                              │   │
│  │ Default Destination: [S3 (my-backup-bucket) ▼]             │   │
│  │ [Test Connection] [Save Settings]                          │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ All Backups                                                  │   │
│  │ [Filter by app/service] [Date range] [Status ▼]            │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ Status │ Name        │ Type   │ Size   │ Destination │ Time   │   │
│  │ 🟢     │ api-prod    │ App    │ 45 MB  │ S3          │ 2h     │   │
│  │ 🟢     │ main-pg     │ DB     │ 234 MB │ S3          │ 2h     │   │
│  │ 🟢     │ web-client  │ App    │ 12 MB  │ Local       │ 1d     │   │
│  │ 🔴     │ redis-cache │ DB     │ —      │ S3          │ 1d     │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [Restore] on any row opens target selection: restore to original   │
│  or create new app/service from backup.                             │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 14. Cron Jobs

### 14.1 Purpose
Schedule and manage automated tasks.

### 14.2 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Cron Jobs                                         [+ New Cron Job] │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Status │ Name        │ Schedule    │ Command      │ Last Run │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ 🟢     │ db-cleanup  │ 0 2 * * *   │ npm run      │ 2h ago   │   │
│  │        │             │ (Daily 2AM) │ cleanup      │ Success  │   │
│  │ 🟢     │ sitemap     │ 0 */6 * * * │ php artisan  │ 4h ago   │   │
│  │        │             │ (Every 6h)  │ sitemap:gen  │ Success  │   │
│  │ 🔴     │ old-report  │ 0 0 * * 0   │ python       │ 1d ago   │   │
│  │        │             │ (Weekly)    │ report.py    │ Failed   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  Click row to view execution history and logs.                      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 14.3 Create Cron Job Modal
```
┌─────────────────────────────────────────────────────────────────────┐
│  New Cron Job                                                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Name: [db-cleanup          ]                                       │
│                                                                     │
│  Target: [Host Server ▼]  [api-prod ▼]                            │
│                                                                     │
│  Schedule:                                                          │
│  [Custom (cron expression) ▼]                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Minute  │ Hour  │ Day  │ Month  │ Weekday                    │   │
│  │ [0    ] │ [2  ] │ [*  ] │ [*   ] │ [*     ]                  │   │
│  │ 0 2 * * * → Daily at 2:00 AM                                  │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  Presets: [Every minute] [Hourly] [Daily] [Weekly] [Monthly]      │
│                                                                     │
│  Command:                                                           │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ npm run cleanup                                             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [✓] Notify on failure                                            │
│  [✓] Log output (retain last 10 runs)                             │
│                                                                     │
│  [Cancel]  [Create Cron Job]                                        │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 15. Firewall

### 15.1 Purpose
Simple visual interface for managing server firewall rules.

### 15.2 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Firewall                                                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Status: 🟢 Active (UFW)  [Disable Firewall] (danger style)       │
│                                                                     │
│  Default Policy: [Drop incoming ▼]  [Allow outgoing ▼]             │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Port │ Protocol │ Source      │ Action   │ App       │ Del  │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ 22   │ TCP      │ Anywhere    │ Allow    │ SSH       │ [×]  │   │
│  │ 80   │ TCP      │ Anywhere    │ Allow    │ HTTP      │ [×]  │   │
│  │ 443  │ TCP      │ Anywhere    │ Allow    │ HTTPS     │ [×]  │   │
│  │ 3000 │ TCP      │ 10.0.0.0/8  │ Allow    │ api-prod  │ [×]  │   │
│  │ 5432 │ TCP      │ 127.0.0.1   │ Allow    │ main-pg   │ [×]  │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [+ Add Rule]                                                       │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Recent Blocks (Last 24h)                                     │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ Time     │ Source IP    │ Port │ Protocol │ Reason         │   │
│  │ 12:34:56 │ 192.168.1.45 │ 22   │ TCP      │ Brute force    │   │
│  │ 11:22:33 │ 10.0.0.99    │ 3389 │ TCP      │ Port scan      │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [IP Blocklist]  [Geo-blocking]  [Rate Limiting] (future tabs)     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 15.3 Add Rule Modal
```
│  Port: [3000          ]  or  [Select App ▼]                        │
│  Protocol: [TCP ▼]  [UDP ▼]  [TCP/UDP ▼]                          │
│  Source: [Anywhere ▼]  [My IP ▼]  [Custom: [10.0.0.0/8]]         │
│  Action: [Allow ▼]  [Deny ▼]                                      │
│  Description: [API internal access]                               │
```

---

## 16. Domains & SSL

### 16.1 Purpose
Manage all domains and SSL certificates in one place.

### 16.2 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Domains & SSL                                                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Domain            │ Apps        │ SSL Status │ Expiry      │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ api.example.com   │ api-prod    │ 🟢 Valid   │ 2026-09-01 │   │
│  │ app.example.com   │ web-client  │ 🟢 Valid   │ 2026-09-01 │   │
│  │ blog.example.com  │ blog        │ 🟡 Expiring│ 2026-06-15 │   │
│  │ staging.example   │ api-staging │ 🟢 Valid   │ 2026-09-01 │   │
│  │ old.example.com   │ —           │ 🔴 Failed  │ —          │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  [+ Add Domain]  [Renew All]  [Force HTTPS All]                   │
│                                                                     │
│  Click any domain to view DNS records, validate configuration,     │
│  and view certificate details.                                      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 17. Settings

### 17.1 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Settings                                                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  [Panel] [Server] [User] [API] [Notifications] [Security]         │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    TAB CONTENT                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 17.2 Tab: Panel Settings
- Panel name
- Panel domain and SSL
- Default app subdomain pattern (`{app}.panel.example.com`)
- Default build strategy
- Backup defaults
- Update preferences (auto-check, auto-update, maintenance window)
- Export all data (JSON dump of panel database)

### 17.3 Tab: Server Settings
- Server name
- Timezone
- Swap configuration
- Kernel parameters (advanced, collapsible)
- OS update schedule
- Reboot server button (with confirmation)

### 17.4 Tab: User Settings
- Profile: username, email, avatar
- Password change (current + new + confirm)
- 2FA: enable/disable, regenerate backup codes
- Session management: list active sessions, revoke any
- Theme: dark / light / system
- Language
- Time format: 12h / 24h
- Date format

### 17.5 Tab: API Keys
- List of active keys with name, scopes, last used, created date
- Create new key: name + scope selection (read, deploy, manage, admin)
- Copy key once (shown only at creation, never again)
- Revoke key

### 17.6 Tab: Notifications
- Email configuration (SMTP or panel-managed)
- Slack/Discord webhook URLs
- Notification preferences per event type:
  - Deployment failed/succeeded
  - SSL expiring
  - Backup failed
  - Server resource alert
  - Team member action
- Quiet hours (no non-critical notifications)

### 17.7 Tab: Security
- Login attempts log
- Failed 2FA attempts
- IP allowlist for panel access
- Session timeout duration
- Password policy (min length, complexity)
- Audit log export

---

## 18. Team Management

### 18.1 Layout
```
┌─────────────────────────────────────────────────────────────────────┐
│  Team                                              [+ Invite Member]│
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Member          │ Role        │ Status   │ Last Active │ Del │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ You (admin)     │ Owner       │ Active   │ Now         │ —   │   │
│  │ john@example    │ Developer   │ Active   │ 2h ago      │ [×] │   │
│  │ sarah@example   │ Admin       │ Active   │ 1d ago      │ [×] │   │
│  │ mike@example    │ Viewer      │ Pending  │ —           │ [×] │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  Roles:                                                             │
│  • Owner — Full control, billing, delete server                     │
│  • Admin — Manage apps, services, users. Cannot delete server.      │
│  • Developer — Deploy apps, manage env vars, view logs.            │
│  • Viewer — Read-only access to all resources.                       │
│                                                                     │
│  [Role Permissions Matrix →] (expandable detailed table)          │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 18.2 Invite Member Modal
```
│  Email: [newmember@example.com        ]                             │
│  Role: [Developer ▼]                                                │
│  [✓] Send welcome email with setup instructions                   │
│  [✓] Notify existing team members                                 │
│  [Cancel]  [Send Invitation]                                      │
```

---

## 19. Notifications & Activity

### 19.1 Activity Log Page
```
┌─────────────────────────────────────────────────────────────────────┐
│  Activity Log                                                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  [Filter: All ▼] [App: All ▼] [Date range] [Search]               │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Time     │ User    │ Action              │ Target    │ IP    │   │
│  │ ───────────────────────────────────────────────────────────  │   │
│  │ 12:34:56 │ You     │ Deployed app        │ api-prod  │ local │   │
│  │ 12:30:01 │ john    │ Restarted service   │ main-pg   │ 10.0..│   │
│  │ 11:15:22 │ sarah   │ Added env var       │ web-client│ 10.0..│   │
│  │ 10:00:00 │ system  │ SSL auto-renewed    │ blog      │ local │   │
│  │ 09:45:12 │ You     │ Created backup      │ main-pg   │ local │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  Pagination: 1 2 3 ... 50  [Export CSV]                            │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 20. Error Pages & Empty States

### 20.1 404 — Page Not Found
```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│              [404 Illustration]                                     │
│                                                                     │
│              Page not found                                         │
│                                                                     │
│              The page you're looking for doesn't exist or            │
│              has been moved.                                        │
│                                                                     │
│              [Go to Dashboard]  [Open Command Palette (Cmd+K)]        │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 20.2 500 — Server Error
```
│              [Error Illustration]                                   │
│                                                                     │
│              Something went wrong                                   │
│                                                                     │
│              The panel encountered an unexpected error.              │
│              The error has been logged and the team notified.        │
│                                                                     │
│              Error ID: err-20240601-123456-abc                      │
│                                                                     │
│              [Reload Page]  [View Logs]  [Contact Support]           │
```

### 20.3 Connection Lost
```
│              [Disconnected Icon]                                    │
│                                                                     │
│              Connection lost                                        │
│                                                                     │
│              Real-time updates are unavailable.                    │
│              Check your network connection.                         │
│                                                                     │
│              [Retry Connection]                                     │
│                                                                     │
│  Auto-retrying in 5... 4... 3... (countdown with cancel)          │
```

### 20.4 Permission Denied
```
│              [Lock Icon]                                            │
│                                                                     │
│              Access denied                                          │
│                                                                     │
│              You don't have permission to view this resource.       │
│              Contact your team administrator.                       │
│                                                                     │
│              [Go Back]  [Request Access]                            │
```

---

## 21. Responsive Behavior

### 21.1 Breakpoints
| Breakpoint | Width | Behavior |
|------------|-------|----------|
| **Mobile** | < 640px | Single column, hamburger menu, stacked cards, bottom sheet modals |
| **Tablet** | 640–1024px | Two-column grids, collapsible sidebar, medium modals |
| **Desktop** | > 1024px | Full layout, three-column dashboard, side drawer modals |

### 21.2 Mobile Adaptations
- **Navigation:** Hamburger menu with slide-out drawer. Bottom tab bar for quick access (Dashboard, Apps, Services, Terminal).
- **Tables:** Horizontal scroll with sticky first column (name). Or card-based list view.
- **Terminal:** Full-screen modal. Swipe up to expand.
- **Logs:** Full-screen view with floating pause/play button.
- **Modals:** Bottom sheet on mobile, centered modal on desktop.
- **Forms:** Single column, full-width inputs, floating labels.

### 21.3 Touch Interactions
- Swipe right on table rows for quick actions (deploy, restart, delete)
- Pull to refresh on lists
- Pinch to zoom on charts
- Long press for context menus

---

## 22. Keyboard Shortcuts

### 22.1 Global Shortcuts
| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl + K` | Open command palette |
| `Cmd/Ctrl + /` | Show keyboard shortcuts help modal |
| `Esc` | Close modal / drawer / dropdown |
| `Cmd/Ctrl + 1` | Go to Dashboard |
| `Cmd/Ctrl + 2` | Go to Apps |
| `Cmd/Ctrl + 3` | Go to Services |
| `Cmd/Ctrl + 4` | Go to Server |
| `Cmd/Ctrl + T` | Open Terminal (new tab) |
| `Cmd/Ctrl + N` | Create new app (opens wizard) |
| `Cmd/Ctrl + B` | Toggle sidebar collapse |

### 22.2 App Detail Shortcuts
| Shortcut | Action |
|----------|--------|
| `D` | Deploy |
| `R` | Restart |
| `L` | Focus logs tab |
| `E` | Focus environment tab |
| `S` | Focus settings tab |
| `Shift + D` | Open deploy modal |

### 22.3 Terminal Shortcuts
| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl + C` | Copy selection |
| `Cmd/Ctrl + V` | Paste |
| `Cmd/Ctrl + Shift + C` | Copy terminal text |
| `Cmd/Ctrl + Shift + V` | Paste into terminal |
| `Cmd/Ctrl + T` | New terminal tab |
| `Cmd/Ctrl + W` | Close terminal tab |
| `F11` | Full screen |

### 22.4 Shortcuts Help Modal
Accessible via `Cmd/Ctrl + /` or "Keyboard Shortcuts" in user menu.
- Categorized list with shortcut and description
- Searchable
- Option to disable shortcuts (for accessibility)

---

*End of UI/UX Specification*
