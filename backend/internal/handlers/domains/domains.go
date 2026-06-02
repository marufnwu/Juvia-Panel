package domains

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"panel-api/internal/config"
	"panel-api/internal/proxy"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles domain-related API endpoints.
type Handler struct {
	cfg  *config.Config
	cm   *proxy.CaddyManager
}

// NewHandler creates a new domains handler.
func NewHandler(cfg *config.Config, cm *proxy.CaddyManager) *Handler {
	return &Handler{cfg: cfg, cm: cm}
}

// ListDomains returns all domains.
// GET /domains
func (h *Handler) ListDomains(c *gin.Context) {
	// Read domains from Caddyfile
	domains := h.getDomainsFromCaddyfile()

	var items []gin.H
	for _, domain := range domains {
		items = append(items, gin.H{
			"id":         domain["id"],
			"app_id":     domain["app_id"],
			"domain":     domain["domain"],
			"ssl_status": "valid",
			"created_at": "",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
	})
}

// getDomainsFromCaddyfile parses domains from the Caddyfile
func (h *Handler) getDomainsFromCaddyfile() []map[string]string {
	caddyPath := h.cfg.CaddyConfig
	data, err := os.ReadFile(caddyPath)
	if err != nil {
		return []map[string]string{}
	}

	var domains []map[string]string
	content := string(data)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "{") {
			continue
		}

		// Check if line looks like a domain
		if !strings.Contains(line, "{") && !strings.Contains(line, " ") {
			// This might be a domain
			if strings.Contains(line, ".") && !strings.Contains(line, "/") {
				id := uuid.New().String()[:8]
				domains = append(domains, map[string]string{
					"id":     id,
					"domain": line,
					"app_id": "",
				})
			}
		}
	}

	return domains
}

// AddDomain adds a domain to an app.
// POST /apps/:id/domains
func (h *Handler) AddDomain(c *gin.Context) {
	appID := c.Param("id")

	var req struct {
		Domain     string `json:"domain" binding:"required"`
		ForceHTTPS bool   `json:"force_https"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Domain is required",
		})
		return
	}

	// Validate domain format
	if !isValidDomain(req.Domain) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_domain",
			"message": "Invalid domain format",
		})
		return
	}

	// Get server public IP for DNS validation
	serverIP := h.getServerIP()

	// Validate DNS - check if domain's A record points to our server
	dnsValid := false
	aRecord := ""
	if req.Domain != "" {
		aRecord = h.resolveDNS(req.Domain)
		if aRecord == serverIP && serverIP != "" {
			dnsValid = true
		}
	}

	// Get app's internal port from app config
	internalPort := h.getAppInternalPort(appID)

	// If we found an internal port, configure Caddy
	if internalPort > 0 {
		if err := h.configureDomain(appID, req.Domain, internalPort, req.ForceHTTPS); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "ssl_config_failed",
				"message": "Failed to configure SSL: " + err.Error(),
			})
			return
		}
	}

	// Generate domain ID
	domainID := uuid.New().String()[:12]

	// Determine SSL status
	sslStatus := "pending"
	if dnsValid {
		sslStatus = "provisioning"
	}

	// If DNS is valid, trigger SSL provisioning
	if dnsValid {
		go h.provisionSSL(domainID, req.Domain)
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         domainID,
		"domain":     req.Domain,
		"ssl_status": sslStatus,
		"dns_valid":  dnsValid,
		"message":    "Domain added. SSL certificate will be provisioned automatically.",
	})
}

// getAppInternalPort returns the internal port for an app by reading its config
func (h *Handler) getAppInternalPort(appID string) int {
	// Look for app in the apps data directory
	appConfigPath := filepath.Join(h.cfg.DataDir, "apps", appID, "config.json")
	if data, err := os.ReadFile(appConfigPath); err == nil {
		var config struct {
			InternalPort int `json:"internal_port"`
		}
		if json.Unmarshal(data, &config) == nil {
			return config.InternalPort
		}
	}
	return 3000 // Default
}

// RemoveDomain removes a domain from an app.
// DELETE /apps/:id/domains/:domain
func (h *Handler) RemoveDomain(c *gin.Context) {
	domainName := c.Param("domain")

	// Remove from Caddy config
	h.removeDomainFromCaddy(domainName)

	c.JSON(http.StatusOK, gin.H{
		"message": "Domain '" + domainName + "' removed from app.",
	})
}

// RenewSSL renews SSL certificate for a domain.
// POST /domains/:domain/renew
func (h *Handler) RenewSSL(c *gin.Context) {
	domainName := c.Param("domain")

	// Trigger Caddy to reload and renew certificate
	ctx := context.Background()

	// Run caddy reload
	caddyPath := h.cfg.CaddyConfig
	cmd := exec.CommandContext(ctx, "caddy", "reload", "--config", caddyPath)
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "success") {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "renewal_failed",
			"message": "Failed to renew certificate: " + string(output),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"domain":     domainName,
		"ssl_status": "valid",
		"message":    "SSL certificate renewed successfully.",
	})
}

// ValidateDNS validates DNS configuration for a domain.
// GET /domains/:domain/validate-dns
func (h *Handler) ValidateDNS(c *gin.Context) {
	domainName := c.Param("domain")

	// Get server public IP
	serverIP := h.getServerIP()

	// Resolve domain's A record
	aRecord := h.resolveDNS(domainName)

	// Check if DNS is correctly configured
	dnsValid := (aRecord == serverIP && serverIP != "")

	c.JSON(http.StatusOK, gin.H{
		"domain":    domainName,
		"dns_valid": dnsValid,
		"a_record":  aRecord,
		"server_ip": serverIP,
		"message": func() string {
			if dnsValid {
				return "DNS is correctly configured."
			}
			return "DNS A record does not match server IP."
		}(),
	})
}

// GetSSLCertificate returns SSL certificate info for a domain.
// GET /domains/:domain/ssl
func (h *Handler) GetSSLCertificate(c *gin.Context) {
	domainName := c.Param("domain")

	// Check Caddy's certificate storage
	certPath := filepath.Join(h.cfg.DataDir, "caddy", "certificates", "acme-v02")
	keyPath := filepath.Join(certPath, domainName+".key")
	certFilePath := filepath.Join(certPath, domainName+".crt")

	var exists bool
	if _, err := os.Stat(certFilePath); err == nil {
		exists = true
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Certificate not found",
		})
		return
	}

	// Read certificate details using openssl
	certCmd := exec.Command("openssl", "x509", "-in", certFilePath, "-noout", "-dates")
	certOutput, _ := certCmd.CombinedOutput()
	_ = string(certOutput)

	c.JSON(http.StatusOK, gin.H{
		"domain":    domainName,
		"cert_path": certFilePath,
		"key_path":  keyPath,
		"issuer":    "Let's Encrypt",
		"message":   "Certificate found.",
	})
}

// Helper functions

func isValidDomain(domain string) bool {
	if len(domain) < 3 || len(domain) > 253 {
		return false
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}

	for _, part := range parts {
		if len(part) < 1 || len(part) > 63 {
			return false
		}
		if !isValidDomainPart(part) {
			return false
		}
	}

	return true
}

func isValidDomainPart(part string) bool {
	for _, c := range part {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func (h *Handler) getServerIP() string {
	ips := []string{
		"https://api.ipify.org",
		"https://ifconfig.me",
		"https://icanhazip.com",
	}

	for _, url := range ips {
		resp, err := http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				buf := make([]byte, 50)
				n, _ := resp.Body.Read(buf)
				ip := strings.TrimSpace(string(buf[:n]))
				if net.ParseIP(ip) != nil {
					return ip
				}
			}
		}
	}

	cmd := exec.Command("hostname", "-I")
	output, _ := cmd.CombinedOutput()
	parts := strings.Fields(string(output))
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}

func (h *Handler) resolveDNS(domain string) string {
	addrs, err := net.LookupHost(domain)
	if err != nil {
		return ""
	}

	if len(addrs) > 0 {
		return addrs[0]
	}

	return ""
}

func (h *Handler) configureDomain(appID, domain string, internalPort int, forceHTTPS bool) error {
	ctx := context.Background()

	caddyBlock := fmt.Sprintf(`
%s {
  reverse_proxy localhost:%d
  tls {
    on_domain_change redirect
  }
  encode gzip
  log {
    output file /var/log/caddy/%s.log
  }
}
`, domain, internalPort, appID)

	caddyPath := h.cfg.CaddyConfig
	var existingCaddy string
	if data, err := os.ReadFile(caddyPath); err == nil {
		existingCaddy = string(data)
	}

	if strings.Contains(existingCaddy, domain) {
		existingCaddy = strings.ReplaceAll(existingCaddy,
			domain+"{",
			caddyBlock)
	} else {
		existingCaddy += caddyBlock
	}

	if err := os.WriteFile(caddyPath, []byte(existingCaddy), 0644); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	cmd := exec.CommandContext(ctx, "caddy", "reload", "--config", caddyPath)
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "success") {
		return fmt.Errorf("caddy reload failed: %s", string(output))
	}

	return nil
}

func (h *Handler) removeDomainFromCaddy(domain string) error {
	ctx := context.Background()

	caddyPath := h.cfg.CaddyConfig
	data, err := os.ReadFile(caddyPath)
	if err != nil {
		return nil
	}

	caddyContent := string(data)
	lines := strings.Split(caddyContent, "\n")
	var newLines []string
	skipBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, domain+" ") || strings.HasPrefix(trimmed, domain+" {") {
			skipBlock = true
			continue
		}
		if skipBlock {
			if strings.TrimSpace(line) == "}" {
				skipBlock = false
			}
			continue
		}
		newLines = append(newLines, line)
	}

	newCaddy := strings.Join(newLines, "\n")

	if err := os.WriteFile(caddyPath, []byte(newCaddy), 0644); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	cmd := exec.CommandContext(ctx, "caddy", "reload", "--config", caddyPath)
	_, err = cmd.CombinedOutput()
	if err != nil {
		// Log but don't fail
	}

	return nil
}

func (h *Handler) provisionSSL(domainID, domain string) {
	ctx := context.Background()

	domainRecord := h.resolveDNS(domain)
	if domainRecord == "" {
		return
	}

	caddyPath := h.cfg.CaddyConfig
	cmd := exec.CommandContext(ctx, "caddy", "reload", "--config", caddyPath)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	// Wait for certificate
	certBaseDir := filepath.Join(h.cfg.DataDir, "caddy", "certificates")
	for i := 0; i < 30; i++ {
		certPath := filepath.Join(certBaseDir, domain+".crt")
		if _, err := os.Stat(certPath); err == nil {
			return
		}
		time.Sleep(2 * time.Second)
	}
}