package settings

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"panel-api/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
)

// Handler handles settings-related requests
type Handler struct {
	db   *sqlx.DB
	cfg  *config.Config
	jwtSecret []byte
}

// New creates a new settings handler
func New(db *sqlx.DB, cfg *config.Config) *Handler {
	return &Handler{
		db:   db,
		cfg:  cfg,
		jwtSecret: []byte(cfg.JWTSecret),
	}
}

// PanelSettings represents panel configuration
type PanelSettings struct {
	PanelURL           string `json:"panel_url"`
	PanelPort          int    `json:"panel_port"`
	APIPort            int    `json:"api_port"`
	AgentSocket        string `json:"agent_socket"`
	DataDir            string `json:"data_dir"`
	LogLevel           string `json:"log_level"`
	LogFormat          string `json:"log_format"`
	MaxRequestSizeMB   int    `json:"max_request_size_mb"`
	RequestTimeoutSecs  int    `json:"request_timeout_secs"`
	MaintenanceMode    bool   `json:"maintenance_mode"`
	RegistrationEnabled bool   `json:"registration_enabled"`
	DefaultRole        string `json:"default_role"`
}

// ServerSettings represents server-level settings
type ServerSettings struct {
	Timezone          string `json:"timezone"`
	Hostname          string `json:"hostname"`
	SSHPort           int    `json:"ssh_port"`
	BackupPath        string `json:"backup_path"`
	AppDataPath       string `json:"app_data_path"`
	ServiceDataPath   string `json:"service_data_path"`
	TempPath          string `json:"temp_path"`
	AutoUpdatesEnabled bool  `json:"auto_updates_enabled"`
	UpdateChannel     string `json:"update_channel"`
	SSLEnabled        bool   `json:"ssl_enabled"`
	SSLProvider       string `json:"ssl_provider"`
	SSLEmail          string `json:"ssl_email"`
}

// NotificationSettings represents notification configuration
type NotificationSettings struct {
	EmailEnabled       bool   `json:"email_enabled"`
	EmailSMTPHost      string `json:"email_smtp_host"`
	EmailSMTPPort      int    `json:"email_smtp_port"`
	EmailSMTPUser      string `json:"email_smtp_user"`
	EmailSMTPPassword  string `json:"email_smtp_password"`
	EmailSMTPFrom      string `json:"email_smtp_from"`
	EmailFromAddress   string `json:"email_from_address"`
	SlackEnabled       bool   `json:"slack_enabled"`
	SlackWebhookURL    string `json:"slack_webhook_url"`
	DiscordEnabled     bool   `json:"discord_enabled"`
	DiscordWebhookURL  string `json:"discord_webhook_url"`
	WebhookEnabled     bool   `json:"webhook_enabled"`
	WebhookURL         string `json:"webhook_url"`
	DeploymentSuccess  bool   `json:"deployment_success"`
	DeploymentFailure  bool   `json:"deployment_failure"`
	BackupSuccess      bool   `json:"backup_success"`
	BackupFailure      bool   `json:"backup_failure"`
	ServerCritical     bool   `json:"server_critical"`
	ServerWarning      bool   `json:"server_warning"`
	SSLRenewal         bool   `json:"ssl_renewal"`
	SSLExpiryWarning   bool   `json:"ssl_expiry_warning"`
	DailySummary       bool   `json:"daily_summary"`
}

// Export represents an export job
type Export struct {
	ID          string    `db:"id" json:"id"`
	Status      string    `db:"status" json:"status"` // preparing, ready, failed
	Format      string    `db:"format" json:"format"`
	FilePath    string    `db:"file_path" json:"file_path"`
	SizeBytes   int64     `db:"size_bytes" json:"size_bytes"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	ExpiresAt   time.Time `db:"expires_at" json:"expires_at"`
}

// GetPanelSettings returns panel configuration settings
// GET /settings/panel
func (h *Handler) GetPanelSettings(c *gin.Context) {
	var settings PanelSettings
	
	// Try to get from database first
	var dbSettings *string
	err := h.db.Get(&dbSettings, "SELECT value FROM settings WHERE key = 'panel'")
	if err == nil && dbSettings != nil {
		json.Unmarshal([]byte(*dbSettings), &settings)
	} else {
		// Return defaults if not in database
		settings = PanelSettings{
			PanelURL:           "https://panel.example.com",
			PanelPort:          2053,
			APIPort:            2053,
			AgentSocket:        "/var/run/panel/agent.sock",
			DataDir:            "/var/panel",
			LogLevel:           "info",
			LogFormat:          "json",
			MaxRequestSizeMB:   10,
			RequestTimeoutSecs: 30,
			MaintenanceMode:    false,
			RegistrationEnabled: true,
			DefaultRole:        "viewer",
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": settings,
	})
}

// UpdatePanelSettings updates panel configuration settings
// PUT /settings/panel
func (h *Handler) UpdatePanelSettings(c *gin.Context) {
	var req PanelSettings

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	// Validate input
	if req.LogLevel != "" && !isValidLogLevel(req.LogLevel) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_LOG_LEVEL",
				"message": "Log level must be one of: debug, info, warn, error",
			},
		})
		return
	}

	if req.DefaultRole != "" && !isValidRole(req.DefaultRole) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ROLE",
				"message": "Default role must be one of: viewer, developer, admin",
			},
		})
		return
	}

	// Save to database
	settingsJSON, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to serialize settings",
			},
		})
		return
	}

	// Upsert settings
	_, err = h.db.Exec(`
		INSERT INTO settings (key, value, updated_at)
		VALUES ('panel', ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = datetime('now')
	`, string(settingsJSON), string(settingsJSON))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to save settings",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Panel settings updated successfully",
	})
}

// GetServerSettings returns server-level settings
// GET /settings/server
func (h *Handler) GetServerSettings(c *gin.Context) {
	var settings ServerSettings
	
	// Try to get from database first
	var dbSettings *string
	err := h.db.Get(&dbSettings, "SELECT value FROM settings WHERE key = 'server'")
	if err == nil && dbSettings != nil {
		json.Unmarshal([]byte(*dbSettings), &settings)
	} else {
		// Return defaults if not in database
		settings = ServerSettings{
			Timezone:          "UTC",
			Hostname:          "server-panel",
			SSHPort:           22,
			BackupPath:        "/var/panel/backups",
			AppDataPath:       "/var/panel/apps",
			ServiceDataPath:   "/var/panel/services",
			TempPath:          "/var/panel/tmp",
			AutoUpdatesEnabled: true,
			UpdateChannel:     "stable",
			SSLEnabled:        true,
			SSLProvider:       "letsencrypt",
			SSLEmail:          "admin@example.com",
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": settings,
	})
}

// UpdateServerSettings updates server-level settings
// PUT /settings/server
func (h *Handler) UpdateServerSettings(c *gin.Context) {
	var req ServerSettings

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	// Validate paths exist and are writable (skip in development)
	if h.cfg.Env == "production" {
		if req.BackupPath != "" {
			if err := validatePath(req.BackupPath); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": gin.H{
						"code":    "INVALID_PATH",
						"message": fmt.Sprintf("Backup path is not writable: %v", err),
					},
				})
				return
			}
		}
		if req.AppDataPath != "" {
			if err := validatePath(req.AppDataPath); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": gin.H{
						"code":    "INVALID_PATH",
						"message": fmt.Sprintf("App data path is not writable: %v", err),
					},
				})
				return
			}
		}
	}

	// Save to database
	settingsJSON, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to serialize settings",
			},
		})
		return
	}

	// Upsert settings
	_, err = h.db.Exec(`
		INSERT INTO settings (key, value, updated_at)
		VALUES ('server', ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = datetime('now')
	`, string(settingsJSON), string(settingsJSON))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to save settings",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Server settings updated successfully",
	})
}

// GetNotificationSettings returns notification configuration
// GET /settings/notifications
func (h *Handler) GetNotificationSettings(c *gin.Context) {
	var settings NotificationSettings
	
	// Try to get from database first
	var dbSettings *string
	err := h.db.Get(&dbSettings, "SELECT value FROM settings WHERE key = 'notifications'")
	if err == nil && dbSettings != nil {
		json.Unmarshal([]byte(*dbSettings), &settings)
		// Don't send password back
		settings.EmailSMTPPassword = ""
	} else {
		// Return defaults if not in database
		settings = NotificationSettings{
			EmailEnabled:       true,
			EmailSMTPHost:      "smtp.example.com",
			EmailSMTPPort:      587,
			EmailSMTPUser:      "notifications@example.com",
			EmailSMTPFrom:      "Server Panel <notifications@example.com>",
			EmailFromAddress:   "notifications@example.com",
			SlackEnabled:       false,
			SlackWebhookURL:    "",
			DiscordEnabled:     false,
			DiscordWebhookURL:  "",
			WebhookEnabled:     false,
			WebhookURL:         "",
			DeploymentSuccess:  true,
			DeploymentFailure:  true,
			BackupSuccess:      true,
			BackupFailure:      true,
			ServerCritical:     true,
			ServerWarning:      false,
			SSLRenewal:         true,
			SSLExpiryWarning:  true,
			DailySummary:       true,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": settings,
	})
}

// UpdateNotificationSettings updates notification configuration
// PUT /settings/notifications
func (h *Handler) UpdateNotificationSettings(c *gin.Context) {
	var req NotificationSettings

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	// Validate webhook URLs
	if req.SlackEnabled && req.SlackWebhookURL != "" && !isValidURL(req.SlackWebhookURL) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_URL",
				"message": "Invalid Slack webhook URL",
			},
		})
		return
	}
	if req.DiscordEnabled && req.DiscordWebhookURL != "" && !isValidURL(req.DiscordWebhookURL) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_URL",
				"message": "Invalid Discord webhook URL",
			},
		})
		return
	}
	if req.WebhookEnabled && req.WebhookURL != "" && !isValidURL(req.WebhookURL) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_URL",
				"message": "Invalid webhook URL",
			},
		})
		return
	}

	// Save to database (don't store password in plaintext - encrypt in production)
	settingsJSON, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to serialize settings",
			},
		})
		return
	}

	// Upsert settings
	_, err = h.db.Exec(`
		INSERT INTO settings (key, value, updated_at)
		VALUES ('notifications', ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = datetime('now')
	`, string(settingsJSON), string(settingsJSON))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to save settings",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notification settings updated successfully",
	})
}

// ExportPanelData exports all panel data for backup
// POST /settings/export
func (h *Handler) ExportPanelData(c *gin.Context) {
	var req struct {
		Format     string `json:"format"` // json, csv
		IncludeDB  bool   `json:"include_db"`
		IncludeLogs bool  `json:"include_logs"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Format = "json"
		req.IncludeDB = true
		req.IncludeLogs = false
	}

	// Generate export ID
	exportID := fmt.Sprintf("exp_%d", time.Now().Unix())

	// Create export directory
	exportDir := filepath.Join(h.cfg.DataDir, "exports", exportID)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "EXPORT_ERROR",
				"message": "Failed to create export directory",
			},
		})
		return
	}

	// Run export in background
	go h.performExport(exportID, req.Format, req.IncludeDB, req.IncludeLogs, exportDir)

	c.JSON(http.StatusOK, gin.H{
		"message": "Export started",
		"data": gin.H{
			"export_id":   exportID,
			"status":      "preparing",
			"download_url": fmt.Sprintf("/api/v1/settings/export/download/%s", exportID),
			"expires_at":  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		},
	})
}

// performExport does the actual export work
func (h *Handler) performExport(exportID, format string, includeDB, includeLogs bool, exportDir string) {
	status := "preparing"
	defer func() {
		// Update status in database
		var sizeBytes int64
		if status == "ready" {
			tarPath := filepath.Join(exportDir, "..", exportID+".tar.gz")
			if fi, err := os.Stat(tarPath); err == nil {
				sizeBytes = fi.Size()
			}
		}
		h.db.Exec(`
			INSERT INTO exports (id, status, format, file_path, size_bytes, created_at, expires_at)
			VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now', '+1 day'))
		`, exportID, status, format, filepath.Join(exportDir, "..", exportID+".tar.gz"), sizeBytes)
	}()

	// Export database tables
	if includeDB {
		tables := []string{"users", "apps", "services", "deployments", "backups", "cron_jobs", "domains"}
		for _, table := range tables {
			if err := h.exportTable(table, exportDir, format); err != nil {
				status = "failed"
				return
			}
		}
	}

	// Create tar.gz archive
	tarPath := filepath.Join(exportDir, "..", exportID+".tar.gz")
	if err := h.createTarGz(exportDir, tarPath); err != nil {
		status = "failed"
		return
	}

	// Clean up export directory
	os.RemoveAll(exportDir)
	status = "ready"
}

// exportTable exports a single table to a file
func (h *Handler) exportTable(table, exportDir, format string) error {
	// Whitelist allowed tables to prevent SQL injection
	allowedTables := map[string]bool{
		"users": true, "apps": true, "services": true, "deployments": true,
		"backups": true, "cron_jobs": true, "domains": true, "settings": true,
		"sessions": true, "api_keys": true, "notifications": true,
	}
	if !allowedTables[table] {
		return fmt.Errorf("table not allowed: %s", table)
	}

	rows, err := h.db.Query(fmt.Sprintf("SELECT * FROM %s", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	data := []map[string]interface{}{}

	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)

		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}
		data = append(data, row)
	}

	filename := filepath.Join(exportDir, table+".json")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(data)
}

// createTarGz creates a tar.gz archive of a directory
func (h *Handler) createTarGz(srcDir, destPath string) error {
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(filepath.Dir(srcDir), path)
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
}

// GetExportStatus returns the status of an export job
// GET /settings/export/:id
func (h *Handler) GetExportStatus(c *gin.Context) {
	exportID := c.Param("id")

	var exp Export
	err := h.db.Get(&exp, "SELECT * FROM exports WHERE id = ?", exportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Export not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"export_id":   exp.ID,
			"status":      exp.Status,
			"download_url": fmt.Sprintf("/api/v1/settings/export/download/%s", exp.ID),
			"size_bytes":  exp.SizeBytes,
			"created_at":  exp.CreatedAt.Format(time.RFC3339),
			"expires_at":  exp.ExpiresAt.Format(time.RFC3339),
		},
	})
}

// DownloadExport downloads an export file
// GET /settings/export/download/:id
func (h *Handler) DownloadExport(c *gin.Context) {
	exportID := c.Param("id")

	var exp Export
	err := h.db.Get(&exp, "SELECT * FROM exports WHERE id = ? AND status = 'ready'", exportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Export not found or not ready",
			},
		})
		return
	}

	// Check if file exists
	if _, err := os.Stat(exp.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "FILE_NOT_FOUND",
				"message": "Export file has expired or been deleted",
			},
		})
		return
	}

	// Stream the file
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=panel-export-%s.tar.gz", exportID))
	c.Header("Content-Type", "application/gzip")
	http.ServeFile(c.Writer, c.Request, exp.FilePath)
}

// TestEmailNotification sends a test email notification
// POST /settings/notifications/test/email
func (h *Handler) TestEmailNotification(c *gin.Context) {
	// Get notification settings
	var settings NotificationSettings
	var dbSettings *string
	err := h.db.Get(&dbSettings, "SELECT value FROM settings WHERE key = 'notifications'")
	if err != nil || dbSettings == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "NOT_CONFIGURED",
				"message": "Email notification not configured",
			},
		})
		return
	}
	json.Unmarshal([]byte(*dbSettings), &settings)

	if !settings.EmailEnabled || settings.EmailSMTPHost == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "NOT_CONFIGURED",
				"message": "Email notification not enabled",
			},
		})
		return
	}

	// Send test email
	auth := smtp.PlainAuth("", settings.EmailSMTPUser, settings.EmailSMTPPassword, settings.EmailSMTPHost)
	to := []string{settings.EmailFromAddress}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Test Email from Server Panel\r\n\r\nThis is a test email sent at %s\r\n", settings.EmailSMTPFrom, settings.EmailFromAddress, time.Now().Format(time.RFC3339))

	err = smtp.SendMail(fmt.Sprintf("%s:%d", settings.EmailSMTPHost, settings.EmailSMTPPort), auth, settings.EmailSMTPUser, to, []byte(msg))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "SMTP_ERROR",
				"message": fmt.Sprintf("Failed to send test email: %v", err),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Test email sent successfully",
	})
}

// TestWebhookNotification sends a test webhook notification
// POST /settings/notifications/test/webhook
func (h *Handler) TestWebhookNotification(c *gin.Context) {
	var req struct {
		Type string `json:"type"` // slack, discord, webhook
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	// Get notification settings
	var settings NotificationSettings
	var dbSettings *string
	err := h.db.Get(&dbSettings, "SELECT value FROM settings WHERE key = 'notifications'")
	if err != nil || dbSettings == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "NOT_CONFIGURED",
				"message": "Notification settings not found",
			},
		})
		return
	}
	json.Unmarshal([]byte(*dbSettings), &settings)

	var webhookURL string
	var payload map[string]interface{}

	switch req.Type {
	case "slack":
		if !settings.SlackEnabled || settings.SlackWebhookURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "NOT_CONFIGURED",
					"message": "Slack webhook not configured",
				},
			})
			return
		}
		webhookURL = settings.SlackWebhookURL
		payload = map[string]interface{}{
			"text": fmt.Sprintf("Test webhook from Server Panel at %s", time.Now().Format(time.RFC3339)),
		}

	case "discord":
		if !settings.DiscordEnabled || settings.DiscordWebhookURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "NOT_CONFIGURED",
					"message": "Discord webhook not configured",
				},
			})
			return
		}
		webhookURL = settings.DiscordWebhookURL
		payload = map[string]interface{}{
			"content": fmt.Sprintf("Test webhook from Server Panel at %s", time.Now().Format(time.RFC3339)),
		}

	case "webhook":
		if !settings.WebhookEnabled || settings.WebhookURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "NOT_CONFIGURED",
					"message": "Generic webhook not configured",
				},
			})
			return
		}
		webhookURL = settings.WebhookURL
		payload = map[string]interface{}{
			"event": "test",
			"timestamp": time.Now().Format(time.RFC3339),
			"message": "Test webhook from Server Panel",
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_TYPE",
				"message": "Invalid webhook type. Must be: slack, discord, or webhook",
			},
		})
		return
	}

	// Send webhook request
	payloadBytes, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "WEBHOOK_ERROR",
				"message": fmt.Sprintf("Failed to send webhook: %v", err),
			},
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"code":    "WEBHOOK_FAILED",
				"message": fmt.Sprintf("Webhook returned status %d", resp.StatusCode),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Test %s webhook sent successfully", req.Type),
	})
}

// Helper functions

func isValidLogLevel(level string) bool {
	validLevels := []string{"debug", "info", "warn", "error"}
	for _, l := range validLevels {
		if level == l {
			return true
		}
	}
	return false
}

func isValidRole(role string) bool {
	validRoles := []string{"viewer", "developer", "admin"}
	for _, r := range validRoles {
		if role == r {
			return true
		}
	}
	return false
}

func isValidURL(targetURL string) bool {
	parsed, err := url.Parse(targetURL)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

func validatePath(path string) error {
	// Check if directory exists and is writable
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create it
			return os.MkdirAll(path, 0755)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory")
	}
	// Check if writable
	testFile := filepath.Join(path, ".write_test")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("directory is not writable: %v", err)
	}
	file.Close()
	os.Remove(testFile)
	return nil
}

// ValidateJWT validates a JWT token and returns the claims
func (h *Handler) ValidateJWT(tokenString string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// AccessClaims for JWT validation
type AccessClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}
