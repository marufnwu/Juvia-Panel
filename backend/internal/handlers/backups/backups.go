package backups

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"panel-api/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles backup-related API endpoints.
type Handler struct {
	cfg *config.Config
}

// NewHandler creates a new backups handler.
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

// ListBackups returns all backups.
// GET /backups
func (h *Handler) ListBackups(c *gin.Context) {
	backups, err := h.listBackupsFromDisk()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"data": []gin.H{},
			"meta": gin.H{
				"total":       0,
				"page":        1,
				"per_page":    20,
				"total_pages": 0,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": backups,
		"meta": gin.H{
			"total":       len(backups),
			"page":        1,
			"per_page":    20,
			"total_pages": 1,
		},
	})
}

// listBackupsFromDisk scans the backup directory and returns backup info
func (h *Handler) listBackupsFromDisk() ([]gin.H, error) {
	backupRoot := filepath.Join(h.cfg.DataDir, "backups")

	var backups []gin.H

	// Walk through backup directories
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		return backups, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		targetType := entry.Name()
		targetDir := filepath.Join(backupRoot, targetType)

		subEntries, err := os.ReadDir(targetDir)
		if err != nil {
			continue
		}

		for _, subEntry := range subEntries {
			if !subEntry.IsDir() {
				continue
			}

			targetID := subEntry.Name()
			backupDir := filepath.Join(targetDir, targetID)

			// Find backup files
			files, err := os.ReadDir(backupDir)
			if err != nil {
				continue
			}

			for _, file := range files {
				if file.IsDir() || filepath.Ext(file.Name()) != ".gz" {
					continue
				}

				info, _ := file.Info()
				backups = append(backups, gin.H{
					"id":              strings.TrimSuffix(file.Name(), ".tar.gz"),
					"target_type":     targetType,
					"target_id":       targetID,
					"status":          "completed",
					"size_mb":        info.Size() / (1024 * 1024),
					"destination":     "local",
					"destination_path": filepath.Join(backupDir, file.Name()),
					"created_at":      info.ModTime().Format(time.RFC3339),
				})
			}
		}
	}

	return backups, nil
}

// CreateBackup creates a new backup.
// POST /backups or POST /services/:id/backups (via context)
func (h *Handler) CreateBackup(c *gin.Context) {
	var req struct {
		TargetType   string `json:"target_type" binding:"required"`
		TargetID     string `json:"target_id" binding:"required"`
		Destination string `json:"destination"`
	}

	// Check if target_type and target_id were set via context (from /services/:id/backups)
	if ctxType, exists := c.Get("target_type"); exists {
		req.TargetType = ctxType.(string)
	}
	if ctxID, exists := c.Get("target_id"); exists {
		req.TargetID = ctxID.(string)
	}

	// If not in context, try to bind from JSON body
	if req.TargetType == "" || req.TargetID == "" {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "target_type and target_id are required",
			})
			return
		}
	}

	// Set default destination
	if req.Destination == "" {
		req.Destination = "local"
	}

	// Generate backup ID
	backupID := uuid.New().String()[:12]

	// Create backup directory
	backupDir := filepath.Join(h.cfg.DataDir, "backups", req.TargetType, req.TargetID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "backup_failed",
			"message": "Failed to create backup directory",
		})
		return
	}

	// Start backup based on target type
	go h.runBackup(backupID, req.TargetType, req.TargetID, backupDir)

	c.JSON(http.StatusAccepted, gin.H{
		"backup_id": backupID,
		"status":    "in_progress",
		"message":   "Backup started.",
	})
}

// runBackup executes the actual backup operation
func (h *Handler) runBackup(backupID, targetType, targetID, backupDir string) {
	ctx := context.Background()

	backupFile := filepath.Join(backupDir, backupID+".tar.gz")
	var err error

	switch targetType {
	case "app":
		err = h.backupApp(ctx, targetID, backupFile)
	case "service":
		err = h.backupService(ctx, targetID, backupFile)
	default:
		// Generic backup - just archive the directory
		sourceDir := filepath.Join(h.cfg.DataDir, "apps", targetID)
		if _, statErr := os.Stat(sourceDir); os.IsNotExist(statErr) {
			sourceDir = filepath.Join(h.cfg.DataDir, "services", targetID)
		}
		err = h.createArchive(ctx, sourceDir, backupFile)
	}

	// Log completion
	if err != nil {
		fmt.Printf("Backup %s failed: %v\n", backupID, err)
	} else {
		fmt.Printf("Backup %s completed: %s\n", backupID, backupFile)
	}
}

// backupApp backs up an application
func (h *Handler) backupApp(ctx context.Context, appID, backupFile string) error {
	appDir := filepath.Join(h.cfg.DataDir, "apps", appID)
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return fmt.Errorf("app directory not found")
	}
	return h.createArchive(ctx, appDir, backupFile)
}

// backupService backs up a service (database, etc.)
func (h *Handler) backupService(ctx context.Context, serviceID, backupFile string) error {
	serviceDir := filepath.Join(h.cfg.DataDir, "services", serviceID)
	if _, err := os.Stat(serviceDir); os.IsNotExist(err) {
		return fmt.Errorf("service directory not found")
	}
	return h.createArchive(ctx, serviceDir, backupFile)
}

// createArchive creates a tar.gz archive of a directory
func (h *Handler) createArchive(ctx context.Context, sourceDir, destFile string) error {
	cmd := exec.CommandContext(ctx, "tar", "-czf", destFile, "-C", sourceDir, ".")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	return nil
}

// RestoreBackup restores a backup.
// POST /backups/:id/restore or POST /services/:id/backups/:backupId/restore
func (h *Handler) RestoreBackup(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		if ctxID, exists := c.Get("backup_id"); exists {
			backupID = ctxID.(string)
		}
	}

	var req struct {
		TargetID  string `json:"target_id"`
		CreateNew bool   `json:"create_new"`
	}

	c.ShouldBindJSON(&req)

	// If target_id was set via context (from /services/:id/backups/:backupId/restore)
	if ctxTargetID, exists := c.Get("target_id"); exists && req.TargetID == "" {
		req.TargetID = ctxTargetID.(string)
	}

	// Find backup file
	backupFile := h.findBackupFile(backupID)
	if backupFile == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Backup not found",
		})
		return
	}

	// If TargetID not provided, extract from path
	if req.TargetID == "" {
		parts := strings.Split(backupFile, string(filepath.Separator))
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] == "backups" && i+2 < len(parts) {
				req.TargetID = parts[i+2]
				break
			}
		}
	}

	restoreID := uuid.New().String()[:12]

	go h.runRestore(restoreID, backupFile, req.TargetID, req.CreateNew)

	c.JSON(http.StatusAccepted, gin.H{
		"restore_id": restoreID,
		"status":     "in_progress",
		"message":    "Restoring backup.",
	})
}

// findBackupFile searches for a backup file by ID
func (h *Handler) findBackupFile(backupID string) string {
	backupRoot := filepath.Join(h.cfg.DataDir, "backups")

	var search func(root string) string
	search = func(root string) string {
		entries, err := os.ReadDir(root)
		if err != nil {
			return ""
		}

		for _, entry := range entries {
			if entry.IsDir() {
				if found := search(filepath.Join(root, entry.Name())); found != "" {
					return found
				}
			} else if strings.HasPrefix(entry.Name(), backupID) {
				return filepath.Join(root, entry.Name())
			}
		}
		return ""
	}

	return search(backupRoot)
}

// runRestore executes the actual restore operation
func (h *Handler) runRestore(restoreID, backupFile, targetID string, createNew bool) {
	ctx := context.Background()

	targetDir := filepath.Join(h.cfg.DataDir, "apps", targetID)
	if strings.Contains(backupFile, "/services/") {
		targetDir = filepath.Join(h.cfg.DataDir, "services", targetID)
	}

	if createNew {
		os.MkdirAll(targetDir, 0755)
	}

	// Extract backup
	cmd := exec.CommandContext(ctx, "tar", "-xzf", backupFile, "-C", targetDir)
	err := cmd.Run()

	if err != nil {
		fmt.Printf("Restore %s failed: %v\n", restoreID, err)
	} else {
		fmt.Printf("Restore %s completed: %s -> %s\n", restoreID, backupFile, targetDir)
	}
}

// DeleteBackup deletes a backup.
// DELETE /backups/:id
func (h *Handler) DeleteBackup(c *gin.Context) {
	backupID := c.Param("id")

	// Find and delete backup file
	backupFile := h.findBackupFile(backupID)
	if backupFile == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Backup not found",
		})
		return
	}

	// Delete backup file
	if err := os.Remove(backupFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "delete_failed",
			"message": "Failed to delete backup file",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Backup deleted.",
	})
}

// GetBackupSettings returns backup settings.
// GET /backup-settings
func (h *Handler) GetBackupSettings(c *gin.Context) {
	settings := h.loadBackupSettings()

	c.JSON(http.StatusOK, gin.H{
		"default_schedule": gin.H{
			"frequency":      settings.DefaultFrequency,
			"time":           settings.DefaultTime,
			"timezone":       settings.DefaultTimezone,
			"retention_days": settings.RetentionDays,
		},
		"default_destination": settings.DefaultDestination,
		"s3_config":           settings.S3Config,
	})
}

// BackupSettings represents backup configuration
type BackupSettings struct {
	DefaultFrequency   string     `json:"default_frequency"`
	DefaultTime        string     `json:"default_time"`
	DefaultTimezone    string     `json:"default_timezone"`
	RetentionDays      int        `json:"retention_days"`
	DefaultDestination string     `json:"default_destination"`
	S3Config           *S3Config  `json:"s3_config"`
}

// S3Config holds S3 backup destination configuration
type S3Config struct {
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	AccessKey string `json:"access_key_id"`
	SecretKey string `json:"secret_access_key"`
}

func (h *Handler) loadBackupSettings() BackupSettings {
	settings := BackupSettings{
		DefaultFrequency:   "daily",
		DefaultTime:        "02:00",
		DefaultTimezone:    "UTC",
		RetentionDays:      7,
		DefaultDestination: "local",
	}

	// Read from config file
	configPath := filepath.Join(h.cfg.ConfigDir, "backup-settings.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return settings
	}

	json.Unmarshal(data, &settings)
	return settings
}

// UpdateBackupSettings updates backup settings.
// PUT /backup-settings
func (h *Handler) UpdateBackupSettings(c *gin.Context) {
	var req struct {
		DefaultSchedule *struct {
			Frequency     string `json:"frequency"`
			Time          string `json:"time"`
			Timezone      string `json:"timezone"`
			RetentionDays int    `json:"retention_days"`
		} `json:"default_schedule"`
		DefaultDestination string    `json:"default_destination"`
		S3Config           *S3Config `json:"s3_config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
		return
	}

	// Load current settings
	settings := h.loadBackupSettings()

	// Update with new values
	if req.DefaultSchedule != nil {
		if req.DefaultSchedule.Frequency != "" {
			settings.DefaultFrequency = req.DefaultSchedule.Frequency
		}
		if req.DefaultSchedule.Time != "" {
			settings.DefaultTime = req.DefaultSchedule.Time
		}
		if req.DefaultSchedule.Timezone != "" {
			settings.DefaultTimezone = req.DefaultSchedule.Timezone
		}
		if req.DefaultSchedule.RetentionDays > 0 {
			settings.RetentionDays = req.DefaultSchedule.RetentionDays
		}
	}

	if req.DefaultDestination != "" {
		settings.DefaultDestination = req.DefaultDestination
	}

	if req.S3Config != nil {
		settings.S3Config = req.S3Config
	}

	// Save settings
	if err := h.saveBackupSettings(settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "save_failed",
			"message": "Failed to save backup settings",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Backup settings updated.",
	})
}

func (h *Handler) saveBackupSettings(settings BackupSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	configPath := filepath.Join(h.cfg.ConfigDir, "backup-settings.json")
	return os.WriteFile(configPath, data, 0644)
}

// ListServiceBackups returns backups for a specific service.
// GET /services/:id/backups
func (h *Handler) ListServiceBackups(c *gin.Context) {
	serviceID := c.Param("id")

	backupDir := filepath.Join(h.cfg.DataDir, "backups", "service", serviceID)
	files, err := os.ReadDir(backupDir)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"data": []gin.H{},
		})
		return
	}

	var items []gin.H
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".gz" {
			continue
		}

		info, _ := file.Info()
		items = append(items, gin.H{
			"id":          strings.TrimSuffix(file.Name(), ".tar.gz"),
			"target_type": "service",
			"target_id":   serviceID,
			"status":      "completed",
			"size_mb":     info.Size() / (1024 * 1024),
			"created_at":  info.ModTime().Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
	})
}