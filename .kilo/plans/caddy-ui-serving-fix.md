# Fix Caddy UI Serving & Architecture Bugs

## Problem

After fresh install, the panel UI is not accessible at `http://192.168.0.211:2053/`. Two root causes:

1. **`GenerateCaddyfile` overwrites panel UI routes**: When the API generates/updates the Caddyfile for app domains, it completely overwrites the static Caddyfile that contains the panel UI site block on `:2053`. The panel UI routes are lost.

2. **`--data` flag invalid in Caddy v2.x**: The `juvia-caddy.service` uses `caddy run --config ... --data ...` but `--data` doesn't exist in Caddy v2.11.3. Caddy crashes in a restart loop.

3. **Admin API mismatch**: The initial Caddyfile has `admin off`, but `ReloadCaddy()` tries to use the admin API socket. On first boot, the reload will fail.

---

## Architecture Overview

```
Initial boot:
  backend/config/Caddyfile → /etc/panel/caddy/Caddyfile
  juvia-caddy starts → serves panel UI on :2053, proxies /api/* → localhost:9090

API startup:
  proxy.New("/etc/panel/caddy/Caddyfile") → Caddy instance
  When app domains are added → GenerateCaddyfile() or AddRoute()
  PROBLEM: These overwrite the ENTIRE Caddyfile, losing panel UI routes
```

---

## Fixes

### Fix 1: `proxy/caddy.go` — Generate complete Caddyfile with panel UI routes

**File:** `backend/internal/proxy/caddy.go`

**Problem:** `GenerateCaddyfile()` generates a Caddyfile with ONLY app domain routes, overwriting the panel UI site block.

**Solution:** Add a `panelUIPort` field to the `Caddy` struct. `GenerateCaddyfile()` writes the global options block, then the panel UI site block, then app domain routes.

```go
type Caddy struct {
    configPath      string
    adminSocketPath string
    panelUIPort     int
    caddyfile       string
    mu              sync.RWMutex
}

func New(configPath string) *Caddy {
    if configPath == "" {
        configPath = "/etc/panel/caddy/Caddyfile"
    }
    return &Caddy{
        configPath:      configPath,
        adminSocketPath: "/var/run/panel/caddy-admin.sock",
        panelUIPort:     2053,
        caddyfile:       "",
    }
}

func (c *Caddy) SetPanelUIPort(port int) {
    c.panelUIPort = port
}
```

Modify `GenerateCaddyfile()` to write:
1. Global options block with admin socket enabled
2. Panel UI site block on `:2053` (same as `backend/config/Caddyfile`)
3. App domain site blocks

```go
func (c *Caddy) GenerateCaddyfile(routes []AppRoute, globalEmail string) error {
    // ... existing lock/email logic ...

    var builder strings.Builder

    // Global options - admin socket ENABLED for dynamic reloads
    builder.WriteString("{\n")
    builder.WriteString(fmt.Sprintf("  email %s\n", globalEmail))
    builder.WriteString(fmt.Sprintf("  admin unix%s\n", c.adminSocketPath))
    builder.WriteString("  log {\n")
    builder.WriteString("    level INFO\n")
    builder.WriteString("  }\n")
    builder.WriteString("}\n\n")

    // Panel UI site block
    c.writePanelUIBlock(&builder)

    // App domain routes
    for _, route := range routes {
        c.addRoute(&builder, route)
    }

    // ... write file ...
}

func (c *Caddy) writePanelUIBlock(builder *strings.Builder) {
    port := c.panelUIPort
    builder.WriteString(fmt.Sprintf(":%d {\n", port))
    builder.WriteString("    header {\n")
    builder.WriteString("        X-Frame-Options \"DENY\"\n")
    builder.WriteString("        X-Content-Type-Options \"nosniff\"\n")
    builder.WriteString("        X-XSS-Protection \"1; mode=block\"\n")
    builder.WriteString("        Referrer-Policy \"strict-origin-when-cross-origin\"\n")
    builder.WriteString("        Permissions-Policy \"camera=(), microphone=(), geolocation=()\"\n")
    builder.WriteString("        -Server\n")
    builder.WriteString("        Content-Security-Policy \"default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self' ws: wss:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'\"\n")
    builder.WriteString("    }\n\n")
    builder.WriteString("    handle /_next/* {\n")
    builder.WriteString("        file_server {\n")
    builder.WriteString("            root /opt/panel/ui/out\n")
    builder.WriteString("        }\n")
    builder.WriteString("    }\n\n")
    builder.WriteString("    handle /static/* {\n")
    builder.WriteString("        file_server {\n")
    builder.WriteString("            root /opt/panel/ui/out\n")
    builder.WriteString("        }\n")
    builder.WriteString("    }\n\n")
    builder.WriteString("    handle /api/* {\n")
    builder.WriteString("        reverse_proxy localhost:9090\n")
    builder.WriteString("    }\n\n")
    builder.WriteString("    handle /health {\n")
    builder.WriteString("        reverse_proxy localhost:9090\n")
    builder.WriteString("    }\n\n")
    builder.WriteString("    handle {\n")
    builder.WriteString("        try_files {path} {path}/index.html /index.html\n")
    builder.WriteString("        file_server {\n")
    builder.WriteString("            root /opt/panel/ui/out\n")
    builder.WriteString("        }\n")
    builder.WriteString("    }\n")
    builder.WriteString("}\n\n")
}
```

Also update `AddRoute()` and `RemoveRoute()` to preserve the panel UI block. The simplest approach: after modifying routes, re-read the file and ensure the panel UI block exists. If not, prepend it.

Alternative (cleaner): Store app routes in memory, regenerate the complete Caddyfile on every change.

```go
func (c *Caddy) AddRoute(route AppRoute) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Read existing Caddyfile
    existing := ""
    if data, err := os.ReadFile(c.configPath); err == nil {
        existing = string(data)
    }

    // Extract global options and panel UI block (everything before first app domain)
    // App domains start with a domain name like "app.example.com {"
    // Panel UI block starts with ":2053 {"
    // Global options start with "{"

    // Find where app domain routes start (after panel UI block)
    header := c.extractHeader(existing)

    // Remove old route for this domain if exists
    header = c.removeRoute(header, route.Domain)

    // Add new route
    var builder strings.Builder
    builder.WriteString(header)
    c.addRoute(&builder, route)

    c.caddyfile = builder.String()

    if err := os.WriteFile(c.configPath, []byte(c.caddyfile), 0644); err != nil {
        return fmt.Errorf("failed to write Caddyfile: %w", err)
    }

    return nil
}

func (c *Caddy) extractHeader(content string) string {
    // Find the panel UI block end (":2053 {" ... "}")
    // Everything up to and including the panel UI block is the "header"
    panelUIStart := fmt.Sprintf(":%d {", c.panelUIPort)
    idx := strings.Index(content, panelUIStart)
    if idx == -1 {
        // No panel UI block found, generate it
        var builder strings.Builder
        c.writePanelUIBlock(&builder)
        return builder.String()
    }

    // Find the end of the panel UI block
    braceCount := 0
    inBlock := false
    for i := idx; i < len(content); i++ {
        if content[i] == '{' {
            braceCount++
            inBlock = true
        } else if content[i] == '}' {
            braceCount--
            if inBlock && braceCount == 0 {
                return content[:i+1] + "\n\n"
            }
        }
    }

    // If we couldn't find the end, return what we have
    return content[:idx] + panelUIStart + "\n"
}
```

### Fix 2: `install.sh` — Remove `--data` flag from Caddy systemd service

**File:** `scripts/install.sh` (line 505)

**Problem:** `caddy run --config ... --data ...` — `--data` doesn't exist in Caddy v2.x.

**Solution:** Remove `--data` flag. Caddy v2 uses `XDG_DATA_HOME` or the `storage` directive in the Caddyfile for data directory.

```diff
- ExecStart=/usr/bin/caddy run --config $CONFIG_DIR/caddy/Caddyfile --adapter caddyfile --data $DATA_DIR/caddy
+ ExecStart=/usr/bin/caddy run --config $CONFIG_DIR/caddy/Caddyfile --adapter caddyfile
```

If we need to set the data directory, add it to the Caddyfile global options:
```
{
    storage file_system /var/panel/caddy
    ...
}
```

### Fix 3: `proxy/caddy.go` — Add CLI reload fallback for first boot

**File:** `backend/internal/proxy/caddy.go`

**Problem:** `ReloadCaddy()` uses the admin API, but on first boot the admin API isn't available (initial Caddyfile has `admin off`).

**Solution:** Try admin API first, fall back to CLI reload if it fails.

```go
func (c *Caddy) ReloadCaddy() error {
    // Try admin API first
    err := c.reloadViaAdminAPI()
    if err == nil {
        return nil
    }

    // Fall back to CLI reload
    return c.reloadViaCLI()
}

func (c *Caddy) reloadViaAdminAPI() error {
    adaptCmd := exec.Command("caddy", "adapt", "--config", c.configPath, "--pretty")
    jsonConfig, err := adaptCmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("Caddyfile adaptation failed: %w — %s", err, string(jsonConfig))
    }

    adminURL := fmt.Sprintf("http://unix%s/load", c.adminSocketPath)
    req, err := http.NewRequest(http.MethodPost, adminURL, strings.NewReader(string(jsonConfig)))
    if err != nil {
        return fmt.Errorf("failed to create reload request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
                var d net.Dialer
                return d.DialContext(ctx, "unix", c.adminSocketPath)
            },
        },
        Timeout: 10 * time.Second,
    }

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("admin API returned %d: %s", resp.StatusCode, string(body))
    }

    return nil
}

func (c *Caddy) reloadViaCLI() error {
    cmd := exec.Command("caddy", "reload", "--config", c.configPath, "--adapter", "caddyfile")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("CLI reload failed: %w — %s", err, string(output))
    }
    return nil
}
```

### Fix 4: `cmd/api/main.go` — Generate initial Caddyfile on startup

**File:** `backend/cmd/api/main.go`

**Problem:** On first boot, the Caddyfile may not have the admin socket enabled or panel UI routes (if `GenerateCaddyfile` was never called).

**Solution:** After initializing the Caddy manager, generate the initial Caddyfile with panel UI routes and reload Caddy.

```go
// After initializing Caddy manager (around line 106):
caddyMgr = proxy.NewCaddyManager(caddy, agentClient)

// Generate initial Caddyfile with panel UI routes
panelEmail := cfg.ServerEmail
if panelEmail == "" {
    panelEmail = "admin@localhost"
}
if err := caddy.GenerateCaddyfile([]proxy.AppRoute{}, panelEmail); err != nil {
    log.Printf("WARNING: Failed to generate initial Caddyfile: %v", err)
} else {
    if err := caddy.ReloadCaddy(); err != nil {
        log.Printf("WARNING: Failed to reload Caddy: %v", err)
    } else {
        log.Println("Caddy reloaded with panel UI routes")
    }
}
```

### Fix 5: `config/config.go` — Add `ServerEmail` field

**File:** `backend/internal/config/config.go`

**Problem:** No way to configure the email for Let's Encrypt / Caddy global options.

**Solution:** Add `ServerEmail` to the Config struct, loaded from config.yml or env var.

```go
type Config struct {
    // ... existing fields ...
    ServerEmail  string
}

// In LoadWithVersion, add:
if v := os.Getenv("PANEL_SERVER_EMAIL"); v != "" {
    cfg.ServerEmail = v
}

// And from config.yml:
if cf.Server.Email != "" {
    cfg.ServerEmail = cf.Server.Email
}
```

### Fix 6: Update static Caddyfile to match generated output

**File:** `backend/config/Caddyfile`

**Problem:** The static Caddyfile has `admin off` which conflicts with the generated Caddyfile's admin socket.

**Solution:** Update the static Caddyfile to match what `GenerateCaddyfile` produces, so it's consistent. This is the fallback config used before the API generates the complete one.

```
{
    admin unix//var/run/panel/caddy-admin.sock
    log {
        level INFO
    }
}

:2053 {
    # ... same panel UI routes ...
}
```

---

## Files to Modify

| File | Fix | Description |
|------|-----|-------------|
| `backend/internal/proxy/caddy.go` | 1, 3 | Add `panelUIPort`, `writePanelUIBlock()`, `extractHeader()`, CLI reload fallback |
| `scripts/install.sh` | 2 | Remove `--data` flag from Caddy systemd service |
| `backend/cmd/api/main.go` | 4 | Generate initial Caddyfile on startup |
| `backend/internal/config/config.go` | 5 | Add `ServerEmail` field |
| `backend/config/Caddyfile` | 6 | Update to match generated output |

## Validation

After implementation:
1. `go build ./... && go vet ./...` — must pass
2. `go test ./...` — must pass
3. Fresh install on server:
   - `curl http://192.168.0.211:2053/` — returns panel UI HTML
   - `curl http://192.168.0.211:2053/health` — returns `{"status":"ok"}`
   - `curl http://192.168.0.211:2053/api/v1/auth/status` — returns auth status
   - `systemctl status juvia-caddy` — active (running)
   - Adding an app domain → Caddy reloads successfully
