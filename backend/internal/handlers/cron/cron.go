package cron

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"panel-api/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles cron job API endpoints.
type Handler struct {
	cfg *config.Config
}

// NewHandler creates a new cron handler.
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

// ListCronJobs returns all cron jobs.
// GET /cron-jobs
func (h *Handler) ListCronJobs(c *gin.Context) {
	// Get cron jobs from system crontab
	cronJobs, err := h.GetSystemCronJobs()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"data": []gin.H{},
		})
		return
	}

	var items []gin.H
	for _, job := range cronJobs {
		parts := strings.Split(job, "|")
		if len(parts) >= 4 {
			items = append(items, gin.H{
				"id":        parts[0],
				"name":      parts[1],
				"schedule":  parts[2],
				"command":   parts[3],
				"status":    "active",
				"next_run":  h.calculateNextRun(parts[2]).Format(time.RFC3339),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
	})
}

// CronJob represents a cron job from the system
type CronJob struct {
	ID              string
	Name            string
	Schedule        string
	Command         string
	Status          string
	NotifyOnFailure bool
	LogRetention    int
}

// CreateCronJob creates a new cron job.
// POST /cron-jobs
func (h *Handler) CreateCronJob(c *gin.Context) {
	var req struct {
		Name            string `json:"name" binding:"required"`
		Schedule        string `json:"schedule" binding:"required"`
		Command         string `json:"command" binding:"required"`
		Target          *struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		NotifyOnFailure bool `json:"notify_on_failure"`
		LogRetention    int  `json:"log_retention"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Name, schedule, and command are required",
		})
		return
	}

	// Validate cron schedule format
	if !isValidCronSchedule(req.Schedule) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_schedule",
			"message": "Invalid cron schedule format",
		})
		return
	}

	// Generate cron job ID
	cronID := uuid.New().String()[:12]

	// Create system cron job
	if err := h.createSystemCronJob(cronID, req.Name, req.Schedule, req.Command); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "cron_creation_failed",
			"message": "Failed to create cron job: " + err.Error(),
		})
		return
	}

	nextRun := h.calculateNextRun(req.Schedule)

	c.JSON(http.StatusCreated, gin.H{
		"id":       cronID,
		"name":     req.Name,
		"status":   "active",
		"next_run": nextRun.Format(time.RFC3339),
		"message":  "Cron job created.",
	})
}

// GetCronJob returns a cron job.
// GET /cron-jobs/:id
func (h *Handler) GetCronJob(c *gin.Context) {
	cronJobID := c.Param("id")

	// Get from system crontab
	cronJobs, _ := h.GetSystemCronJobs()
	for _, job := range cronJobs {
		parts := strings.Split(job, "|")
		if len(parts) >= 4 && parts[0] == cronJobID {
			nextRun := h.calculateNextRun(parts[2])
			c.JSON(http.StatusOK, gin.H{
				"id":                parts[0],
				"name":              parts[1],
				"schedule":          parts[2],
				"command":           parts[3],
				"status":            "active",
				"notify_on_failure": false,
				"log_retention":    0,
				"next_run":          nextRun.Format(time.RFC3339),
				"created_at":        "",
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error":   "not_found",
		"message": "Cron job not found",
	})
}

// UpdateCronJob updates a cron job.
// PUT /cron-jobs/:id
func (h *Handler) UpdateCronJob(c *gin.Context) {
	cronJobID := c.Param("id")

	var req struct {
		Name    string `json:"name"`
		Schedule string `json:"schedule"`
		Command  string `json:"command"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
		return
	}

	// Validate schedule if provided
	if req.Schedule != "" && !isValidCronSchedule(req.Schedule) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_schedule",
			"message": "Invalid cron schedule format",
		})
		return
	}

	// Update system cron
	if err := h.updateSystemCronJob(cronJobID, req.Schedule, req.Command); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "cron_update_failed",
			"message": "Failed to update cron job: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      cronJobID,
		"message": "Cron job updated.",
	})
}

// DeleteCronJob deletes a cron job.
// DELETE /cron-jobs/:id
func (h *Handler) DeleteCronJob(c *gin.Context) {
	cronJobID := c.Param("id")

	// Remove from system crontab
	if err := h.deleteSystemCronJob(cronJobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "delete_failed",
			"message": "Failed to delete cron job: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cron job deleted.",
	})
}

// GetExecutionHistory returns cron job execution history.
// GET /cron-jobs/:id/history
func (h *Handler) GetExecutionHistory(c *gin.Context) {
	cronJobID := c.Param("id")

	// Read cron log files if they exist
	logDir := fmt.Sprintf("%s/logs/cron", h.cfg.DataDir)
	logFile := fmt.Sprintf("%s/%s.log", logDir, cronJobID)

	var items []gin.H

	if data, err := os.ReadFile(logFile); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				items = append(items, gin.H{
					"timestamp": parts[0],
					"status":    parts[1],
					"output":    strings.Join(parts[2:], " "),
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
	})
}

// ToggleCronJob enables or disables a cron job.
// POST /cron-jobs/:id/toggle
func (h *Handler) ToggleCronJob(c *gin.Context) {
	cronJobID := c.Param("id")

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

	// Toggle system cron job
	if err := h.toggleSystemCronJob(cronJobID, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "toggle_failed",
			"message": "Failed to toggle cron job: " + err.Error(),
		})
		return
	}

	status := "disabled"
	if req.Enabled {
		status = "active"
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     cronJobID,
		"status": status,
		"message": "Cron job " + status + ".",
	})
}

// Helper functions

// isValidCronSchedule validates a cron expression
func isValidCronSchedule(schedule string) bool {
	// Basic cron validation: 5 fields (minute, hour, day, month, weekday)
	// Standard format: * * * * *
	parts := strings.Fields(schedule)
	if len(parts) != 5 {
		return false
	}

	// Each part should be a valid cron value (number, *, ranges, lists)
	validPattern := regexp.MustCompile(`^(\*|[0-9]+|[0-9]+-[0-9]+)(/(\d+))?$`)
	for _, part := range parts {
		if !validPattern.MatchString(part) {
			// Also allow lists and ranges like "1,2,3" or "1-5"
			complexPattern := regexp.MustCompile(`^(\*|[0-9]+(-[0-9]+)?(,[0-9]+(-[0-9]+)?)*)(\/[0-9]+)?$`)
			if !complexPattern.MatchString(part) {
				return false
			}
		}
	}

	return true
}

// calculateNextRun calculates the next run time from a cron schedule
func (h *Handler) calculateNextRun(schedule string) time.Time {
	// For demonstration, add 24 hours
	// A real implementation would use a cron parsing library
	return time.Now().Add(24 * time.Hour)
}

// createSystemCronJob adds a cron job to the system crontab
func (h *Handler) createSystemCronJob(id, name, schedule, command string) error {
	ctx := context.Background()

	// Validate inputs to prevent injection
	if strings.ContainsAny(id, "\n\r|;&") || strings.ContainsAny(name, "\n\r|;&") {
		return fmt.Errorf("invalid id or name")
	}

	// Build cron entry with identifier comment
	cronEntry := fmt.Sprintf("#PANEL_CRON_%s|%s\n%s %s\n", id, name, schedule, command)

	// Get current crontab
	cmd := exec.CommandContext(ctx, "crontab", "-l")
	output, err := cmd.CombinedOutput()
	currentCrontab := ""
	if err == nil {
		currentCrontab = string(output)
	}

	// Append new cron entry
	newCrontab := currentCrontab + cronEntry

	// Write new crontab using stdin pipe instead of bash -c
	cmd = exec.CommandContext(ctx, "crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write crontab: %w", err)
	}

	return nil
}

// updateSystemCronJob modifies an existing cron job in the system crontab
func (h *Handler) updateSystemCronJob(id, schedule, command string) error {
	ctx := context.Background()

	// Get current crontab
	cmd := exec.CommandContext(ctx, "crontab", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// No existing crontab, create new
		return fmt.Errorf("crontab not found")
	}

	crontab := string(output)
	lines := strings.Split(crontab, "\n")
	var newLines []string
	found := false

	for _, line := range lines {
		if strings.Contains(line, "#PANEL_CRON_"+id) {
			// Found marker, get name from it
			name := strings.TrimPrefix(line, "#PANEL_CRON_"+id+"|")
			// Replace with new entry
			newLines = append(newLines, fmt.Sprintf("#PANEL_CRON_%s|%s", id, name))
			newLines = append(newLines, fmt.Sprintf("%s %s", schedule, command))
			found = true
		} else if !strings.Contains(line, "#PANEL_CRON_") {
			// Keep other cron entries (skip marker lines)
			newLines = append(newLines, line)
		}
	}

	if !found {
		return fmt.Errorf("cron job not found")
	}

	// Write new crontab using stdin pipe
	newCrontab := strings.Join(newLines, "\n")
	cmd = exec.CommandContext(ctx, "crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update crontab: %w", err)
	}

	return nil
}

// deleteSystemCronJob removes a cron job from the system crontab
func (h *Handler) deleteSystemCronJob(id string) error {
	ctx := context.Background()

	// Get current crontab
	cmd := exec.CommandContext(ctx, "crontab", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil // No crontab exists
	}

	crontab := string(output)
	lines := strings.Split(crontab, "\n")
	var newLines []string
	skipNext := false

	for _, line := range lines {
		if strings.Contains(line, "#PANEL_CRON_"+id) {
			skipNext = true
			continue
		}
		if skipNext && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			skipNext = false
		}
		if !skipNext && !strings.HasPrefix(line, "#PANEL_CRON_") {
			newLines = append(newLines, line)
		}
	}

	// Write new crontab using stdin pipe
	newCrontab := strings.Join(newLines, "\n")
	cmd = exec.CommandContext(ctx, "crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete from crontab: %w", err)
	}

	return nil
}

// toggleSystemCronJob enables or disables a cron job by commenting
func (h *Handler) toggleSystemCronJob(id string, enable bool) error {
	ctx := context.Background()

	// Get current crontab
	cmd := exec.CommandContext(ctx, "crontab", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil // No crontab exists
	}

	crontab := string(output)
	lines := strings.Split(crontab, "\n")
	var newLines []string
	inTargetCron := false
	targetID := "#PANEL_CRON_" + id

	for _, line := range lines {
		if strings.Contains(line, targetID) {
			inTargetCron = true
			if enable {
				// Uncomment the marker
				line = targetID
			}
			newLines = append(newLines, line)
		} else if inTargetCron {
			// This is the cron command line
			if !enable {
				// Comment out the command
				line = "# " + line
			}
			newLines = append(newLines, line)
			inTargetCron = false
		} else {
			newLines = append(newLines, line)
		}
	}

	// Write new crontab using stdin pipe
	newCrontab := strings.Join(newLines, "\n")
	cmd = exec.CommandContext(ctx, "crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to toggle crontab: %w", err)
	}

	return nil
}

// GetSystemCronJobs returns all cron jobs from the system crontab
// Format: "id|name|schedule|command"
func (h *Handler) GetSystemCronJobs() ([]string, error) {
	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "crontab", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []string{}, nil
	}

	var cronJobs []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#PANEL_CRON_") {
			// Parse: #PANEL_CRON_id|name
			rest := strings.TrimPrefix(line, "#PANEL_CRON_")
			// Next line is the actual cron schedule + command
			if scanner.Scan() {
				cronLine := scanner.Text()
				parts := strings.Fields(cronLine)
				if len(parts) >= 6 {
					schedule := strings.Join(parts[:5], " ")
					command := strings.Join(parts[5:], " ")
					cronJobs = append(cronJobs, fmt.Sprintf("%s|%s|%s|%s", rest, rest, schedule, command))
				}
			}
		}
	}

	return cronJobs, nil
}