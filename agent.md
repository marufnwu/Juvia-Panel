# Agent Instructions
## Juvia Panel — AI Coding Agent Guide

**Version:** 1.0  
**Date:** 2026-06-01  
**Purpose:** Single source of truth for AI coding agents building this project. Read this first, then the relevant specification.

---

## 1. Project Overview

Build a **single-server self-hosted PaaS panel**. One VPS or dedicated server. One panel. Deploy apps from Git, manage databases, handle SSL automatically.

**Philosophy:**
- App-centric, not account-centric
- Git-native deployment
- Docker underneath, invisible to users
- Export on uninstall, then purge clean

---

## 2. Reference Specifications

Read these in order. Do not skip.

| # | File | Purpose | Read When |
|---|------|---------|-----------|
| 1 | `server-panel-specification.md` | Architecture, features, roadmap, security | Before any code |
| 2 | `database-schema-specification.md` | SQLite schema, indexes, triggers, Go structs | Before backend code |
| 3 | `api-specification.md` | All endpoints, payloads, WebSocket events, errors | Before API layer |
| 4 | `ui-ux-specification.md` | Every screen, component, user flow, design system | Before frontend code |

**Rule:** If a spec says X, build X. Do not substitute. Do not ask "what about Y instead?"

---

## 3. Technology Stack — Definitive

No alternatives. No "or." These exact technologies.

### Backend
| Layer | Technology | Version |
|-------|-----------|---------|
| Agent Daemon | Go | 1.22+ |
| API Server | Go + Gin | 1.22+ / v1.9+ |
| Database | SQLite (libsql) | 3.45+ via modernc.org/sqlite |
| SQL Library | sqlx | v1.3+ |
| Auth | JWT | golang-jwt/jwt/v5 |
| Real-time | WebSocket | gorilla/websocket |
| Proxy | Caddy | 2.7+ |
| Containers | Docker Engine + Compose | 24.0+ / v2 |
| Terminal | ttyd | 1.7+ |

### Frontend
| Layer | Technology | Version |
|-------|-----------|---------|
| Framework | Next.js | 14 (App Router) |
| Language | TypeScript | 5.3+ |
| Styling | Tailwind CSS | 3.4+ |
| Components | shadcn/ui | Latest |
| Global State | Zustand | 4.5+ |
| Server State | TanStack Query | 5.0+ |
| Real-time | Native WebSocket | Browser API |
| Terminal | xterm.js | 5.3+ |
| Code Editor | Monaco Editor | 0.45+ |
| Icons | Lucide React | Latest |
| Charts | Recharts | 2.10+ |
| Build | Static Export | `output: 'export'` |

### Infrastructure
| Component | Technology |
|-----------|-----------|
| Process Manager | systemd |
| Firewall | UFW |
| SSL | Let's Encrypt via Caddy |
| OS | Ubuntu 24.04 LTS (primary) |

---

## 4. Build Order — Phase by Phase

Do not build out of order. Each phase must be verified before the next.

### Phase 1: Foundation (Week 1)
**Goal:** Working backend with database and basic API.

1. **Database layer**
   - Create SQLite database at `/var/panel/panel.db`
   - Run all migrations from `database-schema-specification.md`
   - Verify tables: `users`, `apps`, `services`, `deployments`
   - Test: `sqlite3 /var/panel/panel.db ".tables"` shows all tables

2. **API server scaffold**
   - `go mod init panel-api`
   - Gin router with health check endpoint `GET /health`
   - Middleware: request logging, recovery, CORS
   - Test: `curl http://localhost:2053/health` returns `{"status":"ok"}`

3. **Authentication**
   - `POST /auth/login` — bcrypt password verify
   - `POST /auth/refresh` — refresh token from HTTP-only cookie
   - `POST /auth/logout` — invalidate session
   - JWT access token (15 min), refresh token (7 days)
   - Test: Login returns access token, refresh returns new access token

4. **User management**
   - `GET /users/me`
   - `POST /users/invite`
   - Role-based access control (owner, admin, developer, viewer)
   - Test: Create user, verify role restrictions

**Verification:** All API endpoints return correct JSON. JWT auth blocks unauthorized requests.

---

### Phase 2: Core API (Week 2)
**Goal:** Full CRUD for apps and services.

1. **Apps API**
   - `GET /apps` — list with filtering, sorting, pagination
   - `GET /apps/{id}` — full app detail
   - `POST /apps` — create from Git repo
   - `PUT /apps/{id}` — partial update
   - `DELETE /apps/{id}` — delete with volume check
   - Test: Create app, verify in database, delete app

2. **Services API**
   - `GET /services` — list databases/caches
   - `POST /services` — provision PostgreSQL, Redis, etc.
   - `GET /services/{id}` — connection strings
   - Test: Create PostgreSQL service, verify container running

3. **Deployments API**
   - `POST /apps/{id}/deploy` — trigger deployment
   - `GET /apps/{id}/deployments` — list history
   - `GET /deployments/{id}/logs` — build logs
   - `POST /apps/{id}/rollback` — rollback to previous
   - Test: Deploy app, verify deployment record created

4. **Environment variables**
   - `GET /apps/{id}/env` — list with secret masking
   - `PUT /apps/{id}/env` — update with encryption for secrets
   - Test: Add secret, verify masked in response

**Verification:** All endpoints match `api-specification.md` exactly. Response shapes identical.

---

### Phase 3: Docker Integration (Week 3)
**Goal:** Apps and services actually run in Docker containers.

1. **Agent daemon**
   - Go binary at `/usr/local/bin/panel-agent`
   - systemd service with auto-restart
   - Unix socket at `/var/run/panel/agent.sock`
   - Commands: build image, run container, stop, logs, exec
   - Test: Agent starts, responds to ping

2. **Build pipeline**
   - Git clone → detect runtime → build via Nixpacks → create image
   - Support: Node.js, Python, Go, PHP, Ruby, Static, Dockerfile
   - Build logs streamed to API
   - Test: Push to GitHub, verify auto-deploy triggers

3. **Container lifecycle**
   - Create container with env vars, volumes, port mapping
   - Health check endpoint polling
   - Restart on failure
   - Test: Deploy app, verify container running, health check passes

4. **Caddy integration**
   - Generate Caddyfile on app create/delete
   - Reload Caddy gracefully
   - SSL auto-provision via Let's Encrypt
   - Test: Create app with domain, verify HTTPS works

**Verification:** Deploy a real Node.js app from GitHub. It builds, runs, and serves traffic.

---

### Phase 4: Frontend Shell (Week 4)
**Goal:** Working UI with navigation and dashboard.

1. **Project setup**
   - `npx create-next-app@14 panel-ui --typescript --tailwind --app`
   - Add shadcn/ui: `npx shadcn-ui@latest init`
   - Add dependencies: zustand, @tanstack/react-query, recharts, lucide-react, xterm, @monaco-editor/react
   - Configure static export in `next.config.js`

2. **Design system**
   - Dark mode default (Slate 900 background, Slate 50 text)
   - Color tokens: Blue 600 primary, Green 600 success, Amber 600 warning, Red 600 danger
   - Font: Inter for UI, JetBrains Mono for code
   - Border radius: 8px cards, 6px buttons
   - Test: Storybook or simple page showing all components

3. **Global layout**
   - Top navigation: Dashboard, Apps, Services, Server
   - Command palette (Cmd+K) with search + actions
   - Notification bell with dropdown
   - User menu dropdown
   - Test: Navigate between pages, command palette opens

4. **Dashboard page**
   - Resource cards: CPU, RAM, Disk with sparklines
   - Running apps list (6 max)
   - Active services list (4 max)
   - Recent activity feed
   - Quick actions: Terminal, Logs, Updates, Restart
   - Test: Dashboard loads, shows real data from API

**Verification:** UI connects to backend. Dashboard shows actual server metrics.

---

### Phase 5: App Management UI (Week 5)
**Goal:** Full app CRUD in the browser.

1. **Apps list page**
   - Table with status dots, search, filters, sort
   - Hover actions: Deploy, Restart, Stop, Logs
   - Empty state with CTA
   - Test: Create 3 apps, verify list updates

2. **Create app wizard**
   - Step 1: Choose source (Git, Upload, Docker Compose, Templates)
   - Step 2: Configure source (repo URL, branch, build strategy)
   - Step 3: Basic config (name, domain, env vars)
   - Step 4: Review and deploy
   - Test: Create app from GitHub, verify deployment starts

3. **App detail page**
   - Tabs: Overview, Deployments, Logs, Environment, Settings
   - Overview: metrics, domains, connected services, volumes
   - Deployments: history table, rollback, view logs
   - Logs: real-time streaming with search, filter
   - Environment: key-value table, secret masking, import .env
   - Settings: build config, health checks, resource limits, danger zone
   - Test: Deploy app, view logs streaming, add env var, restart

**Verification:** Full app lifecycle managed entirely through UI. No CLI needed.

---

### Phase 6: Services & Infrastructure (Week 6)
**Goal:** Databases, backups, server management.

1. **Services list and detail**
   - Create PostgreSQL, MySQL, Redis, MongoDB
   - Connection strings with copy button
   - Backup schedule configuration
   - Test: Create database, connect app to it

2. **Backups**
   - Manual and scheduled backups
   - S3 destination configuration
   - Restore with overwrite or new instance
   - Test: Backup database, verify file exists, restore

3. **Server monitoring**
   - Metrics: CPU, RAM, disk, network, load average
   - Process list with kill action
   - Disk usage analyzer
   - Update checker and installer
   - Test: View real-time metrics, install security update

4. **Terminal and file manager**
   - Web terminal (ttyd + xterm.js)
   - File manager: browse, upload, edit, delete
   - Test: Open terminal, run `ls`, edit file

**Verification:** Server fully manageable through UI.

---

### Phase 7: Polish & Production (Week 7)
**Goal:** Ready for real users.

1. **Team management**
   - Invite users, role assignment
   - Activity audit log
   - API keys with scoped permissions

2. **Security hardening**
   - 2FA enforcement option
   - IP allowlist for panel access
   - Session management
   - Audit log export

3. **Error handling**
   - 404, 500, connection lost pages
   - Toast notifications for all actions
   - Retry logic for failed requests

4. **Responsive design**
   - Mobile layout with hamburger menu
   - Touch-friendly controls
   - Bottom sheet modals

5. **Performance**
   - Lazy load heavy components
   - Virtual scrolling for logs
   - Debounced search inputs
   - Optimistic UI updates

**Verification:** Deploy to fresh VPS, onboard in 5 minutes, deploy first app in 10 minutes.

---

## 5. Coding Standards

### Go (Backend)
- **No ORM.** Use sqlx with named queries. Raw SQL for complex queries.
- **Error handling:** Wrap all errors with context. Return structured errors matching API spec.
- **Logging:** Structured JSON logs. Include request_id in every log line.
- **Context:** Pass `context.Context` through all functions. Respect cancellation.
- **Timeouts:** All external calls (Docker, Git, HTTP) have timeouts.
- **Concurrency:** Use `errgroup` for parallel operations. Never leak goroutines.
- **Testing:** Table-driven tests. Mock Docker client for unit tests.

### TypeScript (Frontend)
- **Strict mode enabled.** No `any` types. Use `unknown` with type guards.
- **API client:** Generate from API spec or hand-write with shared types.
- **State management:** Server state in TanStack Query. UI state in Zustand. Never mix.
- **Components:** Functional components with hooks. No class components.
- **Styling:** Tailwind utilities only. No inline styles. No CSS-in-JS.
- **Accessibility:** All interactive elements keyboard accessible. ARIA labels where needed.

### General
- **Comments:** Explain "why," not "what." Code should be self-documenting.
- **Commits:** Conventional commits. `feat:`, `fix:`, `refactor:`, `docs:`.
- **No TODOs in production code.** Either fix now or create an issue.

---

## 6. Verification Checklist

Before declaring any phase complete, verify:

- [ ] All tests pass (`go test ./...`, `npm test`)
- [ ] API responses match `api-specification.md` exactly
- [ ] Database schema matches `database-schema-specification.md`
- [ ] UI matches `ui-ux-specification.md` layouts
- [ ] No secrets in code (use environment variables)
- [ ] No hardcoded URLs or paths (use config)
- [ ] Error handling covers all API spec error codes
- [ ] WebSocket reconnects automatically on disconnect
- [ ] Dark mode works throughout
- [ ] Mobile layout is usable

---

## 7. Common Pitfalls — Do Not Do These

| Pitfall | Why It's Wrong | What To Do Instead |
|---------|---------------|-------------------|
| Use PostgreSQL instead of SQLite | Spec says SQLite. Adds complexity. | Use `modernc.org/sqlite` as specified. |
| Use Nginx instead of Caddy | Spec says Caddy. No auto-HTTPS. | Use Caddy 2.7+ with JSON API for dynamic config. |
| Use Socket.IO instead of native WebSocket | Adds dependency, heavier protocol. | Use `gorilla/websocket` backend + native `WebSocket` frontend. |
| Use Redux instead of Zustand | Overkill for this project. | Use Zustand for global state, TanStack Query for server state. |
| Build SSR Next.js app | Spec says static export. Requires Node.js runtime in production. | Use `output: 'export'` and serve static files via Caddy. |
| Store JWT in localStorage | XSS vulnerability. | Store access token in memory, refresh token in HTTP-only cookie. |
| Run agent as root | Security risk. | Run as `panel` user with `docker` group membership. |
| Skip migrations | Database changes break production. | Use golang-migrate with versioned SQL files. |
| Hardcode ports | Port conflicts between apps. | Auto-assign ports from configurable range (3000-3999). |

---

## 8. Emergency Contacts

If the agent encounters something not covered by specs:

1. Check all four spec files for the answer.
2. If ambiguous, choose the simplest implementation that satisfies the requirement.
3. Document the decision in code comments.
4. Do not block. Build, verify, move on.

---

## 9. Quick Reference

### File Structure (Host)
```
/etc/panel/
├── config.yml              # Panel configuration
├── keys/                   # Encryption keys (600 permissions)
│   └── master.key
├── ssl/                    # SSL certificates
└── Caddyfile               # Auto-generated by panel

/var/panel/
├── panel.db                # SQLite database
├── panel.db-shm            # SQLite WAL shared memory
├── panel.db-wal            # SQLite WAL file
├── apps/                   # App persistent storage
│   └── {appId}/
│       ├── volumes/
│       └── env/
├── services/               # Service persistent storage
│   └── {serviceId}/
│       └── data/
├── backups/                # Local backup storage
├── logs/                   # Panel and system logs
├── tmp/                    # Temporary build artifacts
├── export/                 # Uninstall exports
└── migrations/             # Database migrations
    ├── 000001_init.up.sql
    └── 000001_init.down.sql

/usr/local/bin/
├── panel-agent             # Agent daemon binary
├── panel-api               # API server binary
└── panel-cli               # CLI management tool

/opt/panel/
└── ui/                     # Static Next.js build
    ├── _next/
    ├── index.html
    └── ...

/var/run/panel/
└── agent.sock              # Agent Unix socket
```

### Key Environment Variables
```bash
PANEL_ENV=production              # development, staging, production
PANEL_DATA_DIR=/var/panel         # Data directory
PANEL_CONFIG_DIR=/etc/panel       # Config directory
PANEL_AGENT_SOCKET=/var/run/panel/agent.sock
PANEL_API_PORT=2053               # Internal API port
PANEL_JWT_SECRET=                 # Generated on first run
PANEL_MASTER_KEY=                 # Generated on first run, AES-256 key
PANEL_DOMAIN=panel.example.com    # Panel domain
PANEL_LOG_LEVEL=info              # debug, info, warn, error
```

### One-Line Commands
```bash
# Check agent status
systemctl status panel-agent

# Check API status
curl http://localhost:2053/health

# View logs
journalctl -u panel-agent -f
journalctl -u panel-api -f

# Database inspection
sqlite3 /var/panel/panel.db ".tables"
sqlite3 /var/panel/panel.db "SELECT * FROM apps;"

# Caddy config check
caddy validate --config /etc/panel/Caddyfile

# Docker cleanup
docker system prune -f
docker volume prune -f
```

---

*End of Agent Instructions*## 8. Implementation State Tracking

The agent MUST maintain a `STATE.md` file in the project root. This file tracks what is built, what is in progress, and what is blocked.

### STATE.md Format

```markdown
# Implementation State

## Current Phase: [1-7 from Build Order]

## Completed
- [x] Database migrations (000001_init)
- [x] User model and bcrypt auth
- [x] JWT login/refresh/logout endpoints

## In Progress
- [ ] App CRUD endpoints (GET /apps done, POST /apps in progress)

## Blocked
- [ ] Docker integration — waiting on agent daemon scaffold

## Next Up
- [ ] Service CRUD endpoints
- [ ] Deployment trigger endpoint

## Decisions Made
- Using nanoid (12 chars) for all IDs, not UUID
- SQLite WAL mode enabled for better concurrency

## Known Issues
- JWT refresh token rotation not implemented yet
- CORS config allows all origins in dev (fix before production)
```

### Rules for STATE.md
1. **Update after every task** — before declaring a session complete, update STATE.md
2. **Check STATE.md first** — at the start of every session, read STATE.md to understand current state
3. **Never delete completed items** — move to "Completed" section
4. **Be specific** — "App endpoints" is wrong. "GET /apps with pagination" is right
5. **Flag blockers immediately** — if something is blocked, write why and what unblocks it

---

## 9. AI Agent Rules & Workflow

### 9.1 Session Start Protocol

At the beginning of every coding session, the agent MUST:

1. **Read STATE.md** — understand what was built, what is in progress, what is next
2. **Read relevant spec** — for the current task, read the matching section from the 4 spec files
3. **Verify environment** — check Go version, Node version, database state, running services
4. **State the plan** — before writing code, state: "Today I will build [X] because STATE.md says [Y] is next"

### 9.2 Task Execution Workflow

For every task, follow this loop:

```
PLAN → CODE → VERIFY → COMMIT → UPDATE STATE
```

**PLAN:**
- State what you are building and why
- Identify the files you will create or modify
- Identify dependencies on other components
- Estimate complexity (small / medium / large)

**CODE:**
- Write the smallest complete unit that works
- Follow the coding standards in Section 5
- Add comments explaining "why," not "what"
- No TODOs in code — either build it now or note it in STATE.md

**VERIFY:**
- Run tests: `go test ./...` or `npm test`
- Run linters: `golangci-lint` or `eslint`
- Verify the feature works manually if tests don't exist yet
- Check for security issues (hardcoded secrets, SQL injection, XSS)

**COMMIT:**
- Use conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`
- Commit message format: `type(scope): description`
- Example: `feat(apps): add POST /apps endpoint with validation`
- One logical change per commit

**UPDATE STATE:**
- Mark task as completed in STATE.md
- Update "In Progress" and "Next Up" sections
- Document any decisions or issues discovered

### 9.3 Code Review Rules (Self-Review Before Commit)

Before every commit, the agent MUST check:

- [ ] **No hallucinated APIs** — every function called exists in the codebase or standard library
- [ ] **No missing error handling** — every error is checked, wrapped, or explicitly ignored with comment
- [ ] **No SQL injection** — all SQL queries use parameterized statements (sqlx named queries)
- [ ] **No hardcoded secrets** — passwords, tokens, keys come from environment or config
- [ ] **No memory leaks** — goroutines have exit conditions, channels are closed
- [ ] **No race conditions** — shared state uses mutexes or channels
- [ ] **Tests exist** — new features have table-driven tests
- [ ] **No breaking changes** — API responses match the spec; if changed, spec is updated too
- [ ] **No dead code** — unused imports, variables, functions are removed
- [ ] **No console.log in production** — use structured logging

### 9.4 When Stuck or Uncertain

If the agent encounters something not covered by specs:

1. **Check all 4 spec files** — the answer may be in a different spec
2. **Check STATE.md** — a previous decision may apply
3. **Choose the simplest solution** that satisfies the requirement
4. **Document the decision** in STATE.md under "Decisions Made"
5. **Never guess on security** — if unsure about auth, encryption, or secrets, stop and ask

### 9.5 Multi-Agent Coordination (If Used)

If multiple AI agents work on this project:

- **Each agent owns one phase** — no two agents work on the same component
- **Agents communicate via STATE.md** — not chat, not comments, not DMs
- **Agent handoff protocol:**
  1. Agent A completes phase, updates STATE.md with "Phase X complete, ready for handoff"
  2. Agent B reads STATE.md, verifies completion, begins next phase
  3. If Agent B finds issues, opens a "Blocker" in STATE.md, tags Agent A
- **No direct code modification by another agent's component** — submit a PR-like patch description in STATE.md

### 9.6 Testing Strategy

**Backend (Go):**
- Unit tests for every handler, service, and utility
- Table-driven tests with named subtests
- Mock Docker client for container operations
- Mock filesystem for file manager operations
- Test database: in-memory SQLite (`:memory:`)

**Frontend (TypeScript):**
- Component tests for complex UI (shadcn/ui components are pre-tested)
- API client tests with mocked fetch
- Integration tests for critical user flows (login → create app → deploy)

**Test Commands:**
```bash
# Backend
go test ./... -race -cover
go test ./... -run TestAuth

# Frontend
npm test
npm run test:ui
npm run test:e2e
```

### 9.7 Debugging Protocol

When something breaks:

1. **Reproduce** — write a test that fails
2. **Isolate** — find the smallest code change that fixes it
3. **Understand** — don't just fix, understand why it broke
4. **Prevent** — add a test or assertion that would have caught it
5. **Document** — if it was a subtle bug, add a comment explaining the trap

**Never:**
- Comment out failing tests to "fix" the build
- Add `// TODO: fix later` without a STATE.md entry
- Change multiple things at once when debugging
- Delete code you don't understand — figure out why it exists first

---


