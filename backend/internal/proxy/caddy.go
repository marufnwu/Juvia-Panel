package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"panel-api/internal/agent"
)

// Caddy handles Caddy server configuration for app domains
type Caddy struct {
	configPath      string
	adminSocketPath string
	panelUIPort     int
	caddyfile       string
	mu              sync.RWMutex
}

// AppRoute represents a route configuration for an app
type AppRoute struct {
	AppID     string
	Domain    string
	Port      int
	Email     string // for Let's Encrypt
	ForceHTTPS bool
}

// New creates a new Caddy instance
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

// SetAdminSocketPath sets the admin API socket path
func (c *Caddy) SetAdminSocketPath(path string) {
	c.adminSocketPath = path
}

// SetConfigPath sets the path to the Caddyfile
func (c *Caddy) SetConfigPath(path string) {
	c.configPath = path
}

// SetPanelUIPort sets the port for the panel UI
func (c *Caddy) SetPanelUIPort(port int) {
	c.panelUIPort = port
}

// GenerateCaddyfile generates a Caddyfile from app routes
func (c *Caddy) GenerateCaddyfile(routes []AppRoute, globalEmail string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if globalEmail == "" {
		globalEmail = "admin@localhost"
	}

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

	// Write routes for each app
	for _, route := range routes {
		c.addRoute(&builder, route)
	}

	c.caddyfile = builder.String()

	// Ensure directory exists
	dir := c.configPath[:strings.LastIndex(c.configPath, "/")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write Caddyfile
	if err := os.WriteFile(c.configPath, []byte(c.caddyfile), 0644); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	return nil
}

// writePanelUIBlock writes the panel UI site block to the Caddyfile
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

// addRoute adds a route block for an app
func (c *Caddy) addRoute(builder *strings.Builder, route AppRoute) {
	builder.WriteString(fmt.Sprintf("%s {\n", route.Domain))

	// Reverse proxy to localhost:port
	builder.WriteString(fmt.Sprintf("  reverse_proxy localhost:%d\n", route.Port))

	// TLS configuration
	if route.Email != "" {
		builder.WriteString(fmt.Sprintf("  tls %s\n", route.Email))
	} else {
		builder.WriteString("  tls auto\n")
	}

	// Force HTTPS if enabled
	if route.ForceHTTPS {
		builder.WriteString("  redir https://{host}{uri}\n")
	}

	// Access log
	logFile := fmt.Sprintf("/var/panel/logs/%s-access.log", route.AppID)
	builder.WriteString(fmt.Sprintf("  log {\n"))
	builder.WriteString(fmt.Sprintf("    output file %s\n", logFile))
	builder.WriteString("    format json\n")
	builder.WriteString("  }\n")

	builder.WriteString("}\n\n")
}

// AddRoute adds or updates a route for an app
func (c *Caddy) AddRoute(route AppRoute) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Read existing Caddyfile
	existing := ""
	if data, err := os.ReadFile(c.configPath); err == nil {
		existing = string(data)
	}

	// Check if route for this domain already exists
	domainStart := fmt.Sprintf("%s {", route.Domain)
	if strings.Contains(existing, domainStart) {
		// Update existing route - remove old block and add new
		existing = c.removeRoute(existing, route.Domain)
	}

	// Extract header (global options + panel UI block)
	header := c.extractHeader(existing)

	// Add new route
	var builder strings.Builder
	builder.WriteString(header)
	c.addRoute(&builder, route)

	c.caddyfile = builder.String()

	// Write updated Caddyfile
	if err := os.WriteFile(c.configPath, []byte(c.caddyfile), 0644); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	return nil
}

// extractHeader extracts the global options and panel UI block from a Caddyfile
// Everything up to and including the panel UI block is considered the "header"
func (c *Caddy) extractHeader(content string) string {
	// Find the panel UI block start (e.g., ":2053 {")
	panelUIStart := fmt.Sprintf(":%d {", c.panelUIPort)
	idx := strings.Index(content, panelUIStart)
	if idx == -1 {
		// No panel UI block found, generate the header
		var builder strings.Builder
		// Write global options
		builder.WriteString("{\n")
		builder.WriteString(fmt.Sprintf("  admin unix%s\n", c.adminSocketPath))
		builder.WriteString("  log {\n")
		builder.WriteString("    level INFO\n")
		builder.WriteString("  }\n")
		builder.WriteString("}\n\n")
		// Write panel UI block
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
				// Return content up to and including the panel UI block
				return content[:i+1] + "\n\n"
			}
		}
	}

	// If we couldn't find the end, return what we have plus the panel UI block
	var builder strings.Builder
	builder.WriteString(content[:idx])
	c.writePanelUIBlock(&builder)
	return builder.String()
}

// RemoveRoute removes a route for an app
func (c *Caddy) RemoveRoute(domain string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Read existing Caddyfile
	existing := ""
	if data, err := os.ReadFile(c.configPath); err != nil {
		return fmt.Errorf("failed to read Caddyfile: %w", err)
	} else {
		existing = string(data)
	}

	// Remove route block
	updated := c.removeRoute(existing, domain)
	if updated == existing {
		// Route didn't exist, nothing to remove
		return nil
	}

	// Write updated Caddyfile
	if err := os.WriteFile(c.configPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	c.caddyfile = updated
	return nil
}

// removeRoute removes a route block from Caddyfile content
func (c *Caddy) removeRoute(content, domain string) string {
	domainBlock := fmt.Sprintf("%s {", domain)
	startIdx := strings.Index(content, domainBlock)
	if startIdx == -1 {
		return content // Not found
	}

	// Find the end of this block (next block start or end of file)
	endIdx := startIdx
	braceCount := 0
	inBlock := false

	for i := startIdx; i < len(content); i++ {
		if content[i] == '{' {
			braceCount++
			inBlock = true
		} else if content[i] == '}' {
			braceCount--
			if inBlock && braceCount == 0 {
				endIdx = i + 1
				break
			}
		}
	}

	// Remove the block
	return content[:startIdx] + content[endIdx:]
}

// ReloadCaddy reloads the Caddy server configuration
func (c *Caddy) ReloadCaddy() error {
	// Try admin API first
	if err := c.reloadViaAdminAPI(); err != nil {
		// Fall back to CLI reload
		return c.reloadViaCLI()
	}
	return nil
}

// reloadViaAdminAPI reloads Caddy using the admin API
func (c *Caddy) reloadViaAdminAPI() error {
	// Convert Caddyfile to JSON via caddy adapt
	adaptCmd := exec.Command("caddy", "adapt", "--config", c.configPath, "--pretty")
	jsonConfig, err := adaptCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Caddyfile adaptation failed: %w — %s", err, string(jsonConfig))
	}

	// POST the JSON config to the admin API
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
		return fmt.Errorf("admin API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("admin API returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// reloadViaCLI reloads Caddy using the CLI command
func (c *Caddy) reloadViaCLI() error {
	cmd := exec.Command("caddy", "reload", "--config", c.configPath, "--adapter", "caddyfile")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("CLI reload failed: %w — %s", err, string(output))
	}
	return nil
}

// ValidateConfig validates the Caddyfile
func (c *Caddy) ValidateConfig() error {
	cmd := exec.Command("caddy", "validate", "--config", c.configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("config validation failed: %w - %s", err, string(output))
	}
	return nil
}

// GetConfig returns the current Caddyfile content
func (c *Caddy) GetConfig() (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.caddyfile == "" {
		data, err := os.ReadFile(c.configPath)
		if err != nil {
			return "", err
		}
		c.caddyfile = string(data)
	}

	return c.caddyfile, nil
}

// ListRoutes returns all configured routes
func (c *Caddy) ListRoutes() ([]AppRoute, error) {
	content, err := c.GetConfig()
	if err != nil {
		return nil, err
	}

	var routes []AppRoute
	lines := strings.Split(content, "\n")

	var currentRoute *AppRoute
	braceCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is a domain line (not global config)
		if strings.HasSuffix(trimmed, " {") && !strings.HasPrefix(trimmed, "{") {
			// New route block
			if currentRoute != nil {
				routes = append(routes, *currentRoute)
			}

			domain := strings.TrimSuffix(trimmed, " {")
			currentRoute = &AppRoute{
				Domain:     domain,
				ForceHTTPS: false,
			}
			braceCount = 0
		}

		if currentRoute != nil {
			// Track braces
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// Check for TLS directive
			if strings.HasPrefix(trimmed, "tls") {
				parts := strings.Split(trimmed, " ")
				if len(parts) >= 2 {
					currentRoute.Email = parts[1]
				}
			}

			// Check for reverse proxy
			if strings.HasPrefix(trimmed, "reverse_proxy") {
				parts := strings.Split(trimmed, " ")
				if len(parts) >= 2 {
					addr := parts[1]
					if strings.HasPrefix(addr, "localhost:") {
						fmt.Sscanf(addr, "localhost:%d", &currentRoute.Port)
					}
				}
			}

			// Check for redir (force https)
			if strings.Contains(trimmed, "redir https://") {
				currentRoute.ForceHTTPS = true
			}

			// End of current block
			if braceCount == 0 && strings.TrimSpace(line) == "}" {
				routes = append(routes, *currentRoute)
				currentRoute = nil
			}
		}
	}

	return routes, nil
}

// CaddyManager provides a higher-level interface for managing Caddy
type CaddyManager struct {
	caddy        *Caddy
	agentClient  *agent.Client
}

// NewCaddyManager creates a new CaddyManager
func NewCaddyManager(caddy *Caddy, agentClient *agent.Client) *CaddyManager {
	return &CaddyManager{
		caddy:       caddy,
		agentClient: agentClient,
	}
}

// SetupAppDomain sets up domain routing for an app
func (cm *CaddyManager) SetupAppDomain(appID, domain string, port int, email string, forceHTTPS bool) error {
	// Validate domain
	if !isValidDomain(domain) {
		return fmt.Errorf("invalid domain: %s", domain)
	}

	// Add route
	route := AppRoute{
		AppID:      appID,
		Domain:     domain,
		Port:       port,
		Email:      email,
		ForceHTTPS: forceHTTPS,
	}

	if err := cm.caddy.AddRoute(route); err != nil {
		return fmt.Errorf("failed to add route: %w", err)
	}

	// Reload Caddy
	if err := cm.caddy.ReloadCaddy(); err != nil {
		return fmt.Errorf("failed to reload Caddy: %w", err)
	}

	return nil
}

// RemoveAppDomain removes domain routing for an app
func (cm *CaddyManager) RemoveAppDomain(domain string) error {
	// Remove route
	if err := cm.caddy.RemoveRoute(domain); err != nil {
		return fmt.Errorf("failed to remove route: %w", err)
	}

	// Reload Caddy
	if err := cm.caddy.ReloadCaddy(); err != nil {
		return fmt.Errorf("failed to reload Caddy: %w", err)
	}

	return nil
}

// UpdateAppDomain updates an existing domain route
func (cm *CaddyManager) UpdateAppDomain(appID, domain string, port int, email string, forceHTTPS bool) error {
	return cm.SetupAppDomain(appID, domain, port, email, forceHTTPS)
}

// isValidDomain performs basic domain validation
func isValidDomain(domain string) bool {
	if domain == "" {
		return false
	}

	// Check length
	if len(domain) > 253 {
		return false
	}

	// Check for valid characters
	validChars := "abcdefghijklmnopqrstuvwxyz0123456789-_."
	for _, c := range strings.ToLower(domain) {
		if !strings.Contains(validChars, string(c)) {
			return false
		}
	}

	// Check for consecutive dots
	if strings.Contains(domain, "..") {
		return false
	}

	// Check for leading/trailing dots
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	return true
}

// EnsureCaddy ensures Caddy is installed and configured
func (cm *CaddyManager) EnsureCaddy() error {
	// Check if Caddy is installed
	cmd := exec.Command("caddy", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Caddy is not installed: %w", err)
	}

	// Ensure config directory exists
	dir := cm.caddy.configPath[:strings.LastIndex(cm.caddy.configPath, "/")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Ensure log directory exists
	logDir := "/var/panel/logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Ensure admin socket directory exists
	socketDir := "/var/run/panel"
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	return nil
}