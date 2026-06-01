package firewall

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles firewall-related API endpoints.
type Handler struct {
}

// NewHandler creates a new firewall handler.
func NewHandler() *Handler {
	return &Handler{}
}

// GetFirewallStatus returns firewall status and rules.
// GET /firewall
func (h *Handler) GetFirewallStatus(c *gin.Context) {
	ctx := context.Background()

	status := h.getUFWStatus(ctx)
	rules := h.getUFWRules(ctx)
	recentBlocks := h.getRecentBlocks(ctx)

	c.JSON(http.StatusOK, gin.H{
		"enabled":        status["enabled"],
		"backend":       "ufw",
		"default_policy": status["default_policy"],
		"rules":          rules,
		"recent_blocks":  recentBlocks,
	})
}

// getUFWStatus retrieves UFW status information
func (h *Handler) getUFWStatus(ctx context.Context) map[string]interface{} {
	result := map[string]interface{}{
		"enabled":        false,
		"default_policy": map[string]string{},
	}

	cmd := exec.CommandContext(ctx, "which", "ufw")
	if err := cmd.Run(); err != nil {
		result["error"] = "ufw not installed"
		return result
	}

	cmd = exec.CommandContext(ctx, "ufw", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		result["error"] = string(output)
		return result
	}

	outputStr := string(output)

	if strings.Contains(outputStr, "Status: active") {
		result["enabled"] = true
	} else {
		result["enabled"] = false
	}

	if strings.Contains(outputStr, "Default: deny incoming") {
		result["default_policy"] = map[string]string{
			"incoming": "deny",
			"outgoing": "allow",
		}
	} else {
		result["default_policy"] = map[string]string{
			"incoming": "allow",
			"outgoing": "allow",
		}
	}

	return result
}

// getUFWRules retrieves all UFW rules
func (h *Handler) getUFWRules(ctx context.Context) []gin.H {
	var rules []gin.H

	// Parse 'ufw status numbered' output
	cmd := exec.CommandContext(ctx, "ufw", "status", "numbered")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return rules
	}

	rules = h.parseUFWStatus(string(output))
	return rules
}

// parseUFWStatus parses UFW status output into rules
func (h *Handler) parseUFWStatus(output string) []gin.H {
	var rules []gin.H

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		// Skip header lines
		if i < 3 {
			continue
		}

		// Parse numbered output like:
		// [ 1] 22/tcp                     ALLOW       Anywhere
		re := regexp.MustCompile(`\[\s*(\d+)\]\s+(\d+)/(\w+)\s+(\w+)\s*(.*)`)
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 5 {
			id := fmt.Sprintf("fw_%s", matches[1])
			port, _ := strconv.Atoi(matches[2])
			protocol := matches[3]
			action := strings.ToLower(matches[4])
			source := "any"
			if strings.Contains(matches[5], "Anywhere") {
				source = "any"
			} else {
				source = strings.TrimSpace(matches[5])
			}

			rules = append(rules, gin.H{
				"id":          id,
				"port":        port,
				"protocol":    protocol,
				"source":      source,
				"action":      action,
				"description": "",
			})
		}
	}

	return rules
}

// getRecentBlocks retrieves recent firewall blocks (from logs)
func (h *Handler) getRecentBlocks(ctx context.Context) []gin.H {
	var blocks []gin.H

	cmd := exec.CommandContext(ctx, "journalctl", "-u", "ufw", "--no-pager", "-n", "50", "--output=cat")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return blocks
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "[UFW BLOCK]") {
			blocks = append(blocks, gin.H{
				"message": line,
			})
		}
	}

	return blocks
}

// AddRule adds a firewall rule.
// POST /firewall/rules
func (h *Handler) AddRule(c *gin.Context) {
	var req struct {
		Port        int    `json:"port" binding:"required"`
		Protocol    string `json:"protocol"`
		Source      string `json:"source"`
		Action      string `json:"action"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Port is required",
		})
		return
	}

	if req.Protocol == "" {
		req.Protocol = "tcp"
	}
	if req.Source == "" {
		req.Source = "any"
	}
	if req.Action == "" {
		req.Action = "allow"
	}

	ctx := context.Background()

	var ufwCmd string
	if req.Action == "allow" {
		ufwCmd = fmt.Sprintf("allow from %s to any port %d proto %s", req.Source, req.Port, req.Protocol)
	} else {
		ufwCmd = fmt.Sprintf("deny from %s to any port %d proto %s", req.Source, req.Port, req.Protocol)
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", fmt.Sprintf("echo 'y' | ufw %s", ufwCmd))
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "already") {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "ufw_failed",
			"message": "Failed to add rule: " + string(output),
		})
		return
	}

	ruleID := uuid.New().String()[:8]

	c.JSON(http.StatusCreated, gin.H{
		"id":          ruleID,
		"port":        req.Port,
		"protocol":    req.Protocol,
		"source":      req.Source,
		"action":      req.Action,
		"description": req.Description,
		"message":     "Firewall rule added.",
	})
}

// DeleteRule deletes a firewall rule.
// DELETE /firewall/rules/:id
func (h *Handler) DeleteRule(c *gin.Context) {
	ruleID := c.Param("id")

	ctx := context.Background()

	numStr := strings.TrimPrefix(ruleID, "fw_")
	cmd := exec.CommandContext(ctx, "ufw", "delete", numStr)
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "not found") {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "delete_failed",
			"message": "Failed to delete rule: " + string(output),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Firewall rule deleted.",
	})
}

// ToggleFirewall enables or disables the firewall.
// POST /firewall/toggle
func (h *Handler) ToggleFirewall(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request",
		})
		return
	}

	ctx := context.Background()

	var cmd *exec.Cmd
	if req.Enabled {
		cmd = exec.CommandContext(ctx, "bash", "-c", "echo 'y' | ufw enable")
	} else {
		cmd = exec.CommandContext(ctx, "ufw", "disable")
	}

	cmdOutput, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(cmdOutput), "already") {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "ufw_failed",
			"message": "Failed to toggle firewall",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled": req.Enabled,
		"message": "Firewall " + map[bool]string{true: "enabled", false: "disabled"}[req.Enabled],
	})
}

// SetDefaultPolicy sets the default incoming/outgoing policy.
// POST /firewall/default-policy
func (h *Handler) SetDefaultPolicy(c *gin.Context) {
	var req struct {
		Direction string `json:"direction"`
		Policy    string `json:"policy"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request",
		})
		return
	}

	if req.Direction != "incoming" && req.Direction != "outgoing" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_direction",
			"message": "Direction must be 'incoming' or 'outgoing'",
		})
		return
	}

	if req.Policy != "allow" && req.Policy != "deny" && req.Policy != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_policy",
			"message": "Policy must be 'allow', 'deny', or 'reject'",
		})
		return
	}

	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "ufw", "default", req.Policy, req.Direction)
	_, err := cmd.CombinedOutput()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "ufw_failed",
			"message": "Failed to set default policy",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"direction": req.Direction,
		"policy":    req.Policy,
		"message":   "Default policy updated.",
	})
}

// GetLogs returns firewall logs.
// GET /firewall/logs
func (h *Handler) GetLogs(c *gin.Context) {
	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "journalctl", "-u", "ufw", "--no-pager", "-n", "100")
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"logs": []gin.H{},
		})
		return
	}

	var logs []gin.H
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "[UFW") {
			logs = append(logs, gin.H{
				"message": line,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
	})
}