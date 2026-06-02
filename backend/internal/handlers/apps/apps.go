package apps

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"panel-api/internal/agent"
	"panel-api/internal/config"
	"panel-api/internal/database"
	"panel-api/internal/services"
	"panel-api/internal/websocket"

	"github.com/gin-gonic/gin"
)

// Handler handles app-related HTTP requests
type Handler struct {
	repo       *AppRepository
	config     *config.Config
	agent      *agent.Client
	wsHub      *websocket.Hub
}

// NewHandler creates a new apps handler
func NewHandler(db *database.DB, cfg *config.Config, agentClient *agent.Client, ws *websocket.Hub) *Handler {
	return &Handler{
		repo:   NewAppRepository(db),
		config: cfg,
		agent:  agentClient,
		wsHub:  ws,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string      `json:"error"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// ListApps handles GET /apps
func (h *Handler) ListApps(c *gin.Context) {
	requestID := c.GetString("request_id")
	ctx := context.Background()
	
	// Parse query parameters
	params := ListAppsParams{
		Status:  c.DefaultQuery("status", "all"),
		Runtime: c.Query("runtime"),
		Search:  c.Query("search"),
		Sort:    c.DefaultQuery("sort", "updated_at"),
		Order:   c.DefaultQuery("order", "desc"),
		Page:    1,
		PerPage: 20,
	}
	
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		}
	}
	
	if perPageStr := c.Query("per_page"); perPageStr != "" {
		if perPage, err := strconv.Atoi(perPageStr); err == nil && perPage > 0 && perPage <= 100 {
			params.PerPage = perPage
		}
	}
	
	apps, total, err := h.repo.ListApps(ctx, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to list apps",
			RequestID: requestID,
		})
		return
	}
	
	totalPages := (total + params.PerPage - 1) / params.PerPage
	
	c.JSON(http.StatusOK, gin.H{
		"data": apps,
		"meta": gin.H{
			"total":       total,
			"page":        params.Page,
			"per_page":    params.PerPage,
			"total_pages": totalPages,
		},
	})
}

// GetApp handles GET /apps/:id
func (h *Handler) GetApp(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()

	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}

	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}

	// Get domains
	domainsDetail, err := h.repo.GetAppDomainsDetail(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app domains",
			RequestID: requestID,
		})
		return
	}

	// Parse source config
	var sourceConfig database.SourceConfig
	database.ParseJSONField(&app.SourceConfig, &sourceConfig)

	// Parse build config
	var buildConfig database.BuildConfig
	if app.BuildConfig != nil {
		database.ParseJSONField(app.BuildConfig, &buildConfig)
	}

	// Get volumes
	volumes, err := h.repo.GetAppVolumes(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app volumes",
			RequestID: requestID,
		})
		return
	}

	// Get connected services
	connectedSvcs, err := h.repo.GetConnectedServices(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get connected services",
			RequestID: requestID,
		})
		return
	}

	// Get primary domain
	var primaryDomain *string
	domainItems := make([]database.AppDomainItem, 0, len(domainsDetail))
	for _, d := range domainsDetail {
		item := database.AppDomainItem{
			Domain:       d.Domain,
			SSLStatus:    d.SSLStatus,
			SSLExpiresAt: d.SSLExpiresAt,
			ForceHTTPS:   d.ForceHTTPS,
		}
		domainItems = append(domainItems, item)

		if d.IsPrimary {
			primaryDomain = &d.Domain
		}
	}

	// Build response
	volumeItems := make([]database.VolumeItem, 0, len(volumes))
	for _, v := range volumes {
		volumeItems = append(volumeItems, database.VolumeItem{
			ID:            v.ID,
			HostPath:      v.HostPath,
			ContainerPath: v.ContainerPath,
			SizeMB:        v.SizeMB,
		})
	}

	// Container info
	containerInfo := database.ContainerInfo{
		ID:            app.ContainerID,
		Image:         app.ContainerImage,
		Status:        app.Status,
		RestartPolicy: app.RestartPolicy,
		Ports:         []int{app.InternalPort},
		Network:       "panel_apps", // Default network name
	}

	detail := database.AppDetail{
		ID:                app.ID,
		Name:              app.Name,
		Status:            app.Status,
		HealthStatus:      app.HealthStatus,
		Runtime:           app.Runtime,
		RuntimeVersion:    app.RuntimeVersion,
		PrimaryDomain:     primaryDomain,
		Domains:           domainItems,
		Source:            sourceConfig,
		Build:             buildConfig,
		Resources: database.Resources{
			CPULimit:      app.CPULimit,
			MemoryLimitMB: app.MemoryLimitMB,
			MemorySwapMB:  app.MemorySwapMB,
		},
		Container:         containerInfo,
		Volumes:           volumeItems,
		ConnectedServices: connectedSvcs,
		CreatedAt:         app.CreatedAt,
		UpdatedAt:         app.UpdatedAt,
	}

	c.JSON(http.StatusOK, detail)
}

// GetAppLogs handles GET /apps/:id/logs
func (h *Handler) GetAppLogs(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()

	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}

	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}

	if app.ContainerID == nil || *app.ContainerID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "no_container",
			Message:   "App has no container",
			RequestID: requestID,
		})
		return
	}

	stream := c.DefaultQuery("stream", "stdout")
	tailStr := c.DefaultQuery("tail", "100")
	tail := 100
	if t, err := strconv.Atoi(tailStr); err == nil && t > 0 {
		tail = t
	}

	logLines, err := h.agent.GetLogs(ctx, *app.ContainerID, stream, tail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "logs_error",
			Message:   "Failed to get container logs: " + err.Error(),
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"app_id":  appID,
		"stream":  stream,
		"lines":   logLines,
	})
}

// CreateAppRequest represents the request body for creating an app
type CreateAppRequest struct {
	Name   string                `json:"name" binding:"required"`
	Source CreateAppSourceConfig `json:"source" binding:"required"`
	Build  *CreateAppBuildConfig `json:"build,omitempty"`
	Domain *CreateAppDomain      `json:"domain,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Resources *CreateAppResources `json:"resources,omitempty"`
}

// CreateAppSourceConfig represents source configuration for creating an app
type CreateAppSourceConfig struct {
	Type       string `json:"type" binding:"required"` // git, upload, docker_compose
	RepoURL    string `json:"repo_url,omitempty"`
	Branch     string `json:"branch,omitempty"`
	AutoDeploy bool   `json:"auto_deploy,omitempty"`
}

// CreateAppBuildConfig represents build configuration for creating an app
type CreateAppBuildConfig struct {
	Strategy     string            `json:"strategy,omitempty"`
	BuildCommand string            `json:"build_command,omitempty"`
	StartCommand string            `json:"start_command,omitempty"`
	HealthCheck  *HealthCheckConfig `json:"health_check,omitempty"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Path     string `json:"path,omitempty"`
	Interval int    `json:"interval,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
	Retries  int    `json:"retries,omitempty"`
}

// CreateAppDomain represents domain configuration for creating an app
type CreateAppDomain struct {
	Primary    string `json:"primary,omitempty"`
	ForceHTTPS bool   `json:"force_https,omitempty"`
}

// CreateAppResources represents resource configuration for creating an app
type CreateAppResources struct {
	CPULimit      *float64 `json:"cpu_limit,omitempty"`
	MemoryLimitMB *int     `json:"memory_limit_mb,omitempty"`
}

// CreateApp handles POST /apps
func (h *Handler) CreateApp(c *gin.Context) {
	requestID := c.GetString("request_id")
	ctx := context.Background()
	
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Invalid request body: " + err.Error(),
			RequestID: requestID,
		})
		return
	}
	
	// Validate name (alphanumeric, dashes, underscores)
	if !isValidAppName(req.Name) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_name",
			Message:   "App name must be alphanumeric with dashes and underscores only",
			RequestID: requestID,
		})
		return
	}
	
	// Check if app name already exists
	exists, err := h.repo.AppExists(ctx, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to check app name",
			RequestID: requestID,
		})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:     "app_name_exists",
			Message:   "An app with the name '" + req.Name + "' already exists.",
			RequestID: requestID,
		})
		return
	}
	
	// Validate Git URL if source type is git
	if req.Source.Type == "git" {
		if req.Source.RepoURL == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:     "missing_git_url",
				Message:   "Repository URL is required when source type is git.",
				RequestID: requestID,
			})
			return
		}
		if !ValidateGitURL(req.Source.RepoURL) {
			c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
				Error:     "invalid_git_url",
				Message:   "Repository URL is not accessible. Ensure it is public or add an SSH key.",
				RequestID: requestID,
			})
			return
		}
	}
	
	// Generate app ID
	appID, err := generateID("app_")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to generate app ID",
			RequestID: requestID,
		})
		return
	}
	
	// Determine runtime from source or default
	runtime := detectRuntime(req.Source.RepoURL, req.Source.Type)
	
	// Parse source config
	provider, _ := ParseRepoURL(req.Source.RepoURL)
	sourceConfig := database.SourceConfig{
		Type:       req.Source.Type,
		Provider:   provider,
		RepoURL:    req.Source.RepoURL,
		Branch:     req.Source.Branch,
		AutoDeploy: req.Source.AutoDeploy,
	}
	
	// Build config
	buildStrategy := "nixpacks"
	if req.Build != nil && req.Build.Strategy != "" {
		buildStrategy = req.Build.Strategy
	}
	
	var buildConfig database.BuildConfig
	if req.Build != nil {
		buildConfig.Strategy = buildStrategy
		buildConfig.BuildCommand = req.Build.BuildCommand
		buildConfig.StartCommand = req.Build.StartCommand
		if req.Build.HealthCheck != nil {
			buildConfig.HealthCheck = &database.HealthCheck{
				Path:     req.Build.HealthCheck.Path,
				Interval: req.Build.HealthCheck.Interval,
				Timeout:  req.Build.HealthCheck.Timeout,
				Retries:  req.Build.HealthCheck.Retries,
			}
		}
	} else {
		// Default health check
		buildConfig.Strategy = buildStrategy
		buildConfig.HealthCheck = &database.HealthCheck{
			Path:     "/health",
			Interval: 30,
			Timeout:  5,
			Retries:  3,
		}
	}
	
	// Create app
	userID := c.GetInt("user_id")
	now := time.Now()
	
	app := &database.App{
		ID:                  appID,
		Name:                req.Name,
		Status:              "deploying",
		HealthStatus:        "unknown",
		Runtime:             runtime,
		RuntimeVersion:      nil,
		SourceType:          req.Source.Type,
		SourceConfig:        sourceConfig.ToJSON(),
		BuildStrategy:       buildStrategy,
		BuildConfig:         strPtr(buildConfig.ToJSON()),
		ContainerID:         nil,
		ContainerImage:      nil,
		InternalPort:        3000,
		RestartPolicy:       "unless-stopped",
		HealthCheckPath:     "/health",
		HealthCheckInterval: 30,
		HealthCheckTimeout:  5,
		HealthCheckRetries: 3,
		CPULimit:           nil,
		MemoryLimitMB:       nil,
		MemorySwapMB:        nil,
		CreatedBy:          userID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	
	// Apply resource limits
	if req.Resources != nil {
		app.CPULimit = req.Resources.CPULimit
		app.MemoryLimitMB = req.Resources.MemoryLimitMB
	}
	
	// Apply build config to app fields
	if buildConfig.HealthCheck != nil {
		app.HealthCheckPath = buildConfig.HealthCheck.Path
		app.HealthCheckInterval = buildConfig.HealthCheck.Interval
		app.HealthCheckTimeout = buildConfig.HealthCheck.Timeout
		app.HealthCheckRetries = buildConfig.HealthCheck.Retries
	}
	
	if err := h.repo.CreateApp(ctx, app); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create app",
			RequestID: requestID,
		})
		return
	}
	
	// Create primary domain if provided
	if req.Domain != nil && req.Domain.Primary != "" {
		domain := &database.AppDomain{
			AppID:      appID,
			Domain:     req.Domain.Primary,
			IsPrimary:  true,
			ForceHTTPS: req.Domain.ForceHTTPS,
			SSLStatus:  "pending",
			CreatedAt:  now,
		}
		if err := h.repo.CreateAppDomain(ctx, domain); err != nil {
			// Log error but don't fail
		}
	}
	
	// Create deployment record
	deploymentID, _ := generateID("dep_")
	deployment := &database.Deployment{
		ID:          deploymentID,
		AppID:       appID,
		Status:      "queued",
		CommitSHA:   nil,
		CommitMessage: nil,
		CommitAuthor: nil,
		Branch:      strPtr(req.Source.Branch),
		BuildStrategy: &buildStrategy,
		BuildLogs:   nil,
		TriggeredBy: "manual",
		TriggeredByUserID: &userID,
		StartedAt:   nil,
		CompletedAt: nil,
		CreatedAt:  now,
	}
	h.repo.CreateDeployment(ctx, deployment)
	
	c.JSON(http.StatusCreated, gin.H{
		"id":            appID,
		"name":          req.Name,
		"status":        "deploying",
		"message":       "App created. Deployment started.",
		"deployment_id": deploymentID,
	})
}

// UpdateAppRequest represents the request body for updating an app
type UpdateAppRequest struct {
	Domain *UpdateAppDomain `json:"domain,omitempty"`
	Build  *UpdateBuild     `json:"build,omitempty"`
}

// UpdateAppDomain represents domain update configuration
type UpdateAppDomain struct {
	Primary    string   `json:"primary,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
	ForceHTTPS *bool    `json:"force_https,omitempty"`
}

// UpdateBuild represents build configuration update
type UpdateBuild struct {
	BuildCommand string `json:"build_command,omitempty"`
	StartCommand string `json:"start_command,omitempty"`
}

// UpdateApp handles PUT /apps/:id
func (h *Handler) UpdateApp(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	var req UpdateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Invalid request body",
			RequestID: requestID,
		})
		return
	}
	
	// Get existing app
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	requiresRestart := false
	
	// Update build config if provided
	if req.Build != nil {
		var buildConfig database.BuildConfig
		if app.BuildConfig != nil {
			database.ParseJSONField(app.BuildConfig, &buildConfig)
		}
		
		if req.Build.BuildCommand != "" {
			buildConfig.BuildCommand = req.Build.BuildCommand
			requiresRestart = true
		}
		if req.Build.StartCommand != "" {
			buildConfig.StartCommand = req.Build.StartCommand
			requiresRestart = true
		}
		
		app.BuildConfig = strPtr(buildConfig.ToJSON())
	}
	
	// Update app
	if err := h.repo.UpdateApp(ctx, app); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to update app",
			RequestID: requestID,
		})
		return
	}
	
	message := "App updated."
	if requiresRestart {
		message = "App updated. Restart required for some changes to take effect."
	}
	
	c.JSON(http.StatusOK, gin.H{
		"id":               appID,
		"name":             app.Name,
		"message":          message,
		"requires_restart": requiresRestart,
	})
}

// DeleteApp handles DELETE /apps/:id
func (h *Handler) DeleteApp(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	force := c.Query("force") == "true"
	deleteVolumes := c.Query("delete_volumes") == "true"
	
	// Get app
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	// Check for volumes if not deleting volumes and not forcing
	if !force && !deleteVolumes {
		volumes, err := h.repo.GetAppVolumes(ctx, appID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:     "internal_error",
				Message:   "Failed to check volumes",
				RequestID: requestID,
			})
			return
		}
		
		if len(volumes) > 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:     "volumes_exist",
				Message:   "App has persistent volumes. Set delete_volumes=true to remove them, or export data first.",
				RequestID: requestID,
			})
			return
		}
	}
	
	// Delete app (cascade will handle related records)
	if err := h.repo.DeleteApp(ctx, appID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to delete app",
			RequestID: requestID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "App '" + app.Name + "' and all associated resources deleted.",
		"deleted_resources": gin.H{
			"container":        true,
			"volumes":          deleteVolumes,
			"domains":          true,
			"ssl_certificates": true,
			"nginx_config":      true,
		},
	})
}

// RestartApp handles POST /apps/:id/restart
func (h *Handler) RestartApp(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	// Update status to restarting
	if err := h.repo.UpdateAppStatus(ctx, appID, "restarting"); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to restart app",
			RequestID: requestID,
		})
		return
	}
	
	// Emit status change event
	if h.wsHub != nil {
		websocket.EmitAppStatusChanged(h.wsHub, appID, app.Status, "restarting", app.HealthStatus)
	}
	
	// Restart container via agent
	go func() {
		if app.ContainerID != nil && *app.ContainerID != "" {
			h.agent.Restart(ctx, *app.ContainerID)
		}
		h.repo.UpdateAppStatus(context.Background(), appID, "running")
		if h.wsHub != nil {
			websocket.EmitAppStatusChanged(h.wsHub, appID, "restarting", "running", "healthy")
		}
	}()
	
	c.JSON(http.StatusOK, gin.H{
		"message": "App '" + app.Name + "' is restarting.",
		"status":  "restarting",
	})
}

// StopApp handles POST /apps/:id/stop
func (h *Handler) StopApp(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	// Stop container via agent if running
	if app.ContainerID != nil && *app.ContainerID != "" {
		go func() {
			h.agent.Stop(ctx, *app.ContainerID, 10)
			h.repo.UpdateAppStatus(context.Background(), appID, "stopped")
			if h.wsHub != nil {
				websocket.EmitAppStatusChanged(h.wsHub, appID, app.Status, "stopped", app.HealthStatus)
			}
		}()
	} else {
		h.repo.UpdateAppStatus(ctx, appID, "stopped")
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "App '" + app.Name + "' stopped.",
		"status":  "stopped",
	})
}

// StartApp handles POST /apps/:id/start
func (h *Handler) StartApp(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	if app.ContainerID == nil || *app.ContainerID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "no_container",
			Message:   "App has no container to start",
			RequestID: requestID,
		})
		return
	}
	
	if err := h.repo.UpdateAppStatus(ctx, appID, "deploying"); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to start app",
			RequestID: requestID,
		})
		return
	}
	
	// Start container via agent
	go func() {
		if err := h.agent.Start(context.Background(), *app.ContainerID); err != nil {
			h.repo.UpdateAppStatus(context.Background(), appID, "failed")
			return
		}
		h.repo.UpdateAppStatus(context.Background(), appID, "running")
		if h.wsHub != nil {
			websocket.EmitAppStatusChanged(h.wsHub, appID, "deploying", "running", "healthy")
		}
	}()
	
	c.JSON(http.StatusOK, gin.H{
		"message": "App '" + app.Name + "' is starting.",
		"status":  "deploying",
	})
}

// Helper functions

func generateID(prefix string) (string, error) {
	bytes := make([]byte, 9)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + encode62(bytes), nil
}

func encode62(bytes []byte) string {
	const chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	result := make([]byte, len(bytes))
	for i, b := range bytes {
		result[i] = chars[int(b)%62]
	}
	return string(result)
}

func strPtr(s string) *string {
	return &s
}

func isValidAppName(name string) bool {
	if len(name) < 1 || len(name) > 64 {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func detectRuntime(repoURL, sourceType string) string {
	if sourceType == "docker_compose" || sourceType == "docker" {
		return "docker"
	}
	
	lowerURL := strings.ToLower(repoURL)
	if strings.Contains(lowerURL, "node") || strings.Contains(lowerURL, "next") || strings.Contains(lowerURL, "react") || strings.Contains(lowerURL, "vue") {
		return "nodejs"
	}
	if strings.Contains(lowerURL, "python") || strings.Contains(lowerURL, "django") || strings.Contains(lowerURL, "flask") {
		return "python"
	}
	if strings.Contains(lowerURL, "go") || strings.Contains(lowerURL, "golang") {
		return "go"
	}
	if strings.Contains(lowerURL, "php") || strings.Contains(lowerURL, "laravel") || strings.Contains(lowerURL, "wordpress") {
		return "php"
	}
	if strings.Contains(lowerURL, "ruby") || strings.Contains(lowerURL, "rails") {
		return "ruby"
	}
	
	return "static"
}

func parseEnvVars(envVars map[string]string) ([]database.AppEnvVar, error) {
	result := make([]database.AppEnvVar, 0, len(envVars))
	now := time.Now()
	
	for key, value := range envVars {
		isSecret := isSecretKey(key)
		result = append(result, database.AppEnvVar{
			Key:       key,
			Value:     value,
			IsSecret:  isSecret,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	
	return result, nil
}

// EnvVars section

// GetEnvVars handles GET /apps/:id/env
func (h *Handler) GetEnvVars(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	// Check app exists
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	envVars, err := h.repo.GetAppEnvVars(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get environment variables",
			RequestID: requestID,
		})
		return
	}
	
	type EnvVarItem struct {
		Key       string    `json:"key"`
		Value     string    `json:"value"`
		IsSecret  bool      `json:"is_secret"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	
	items := make([]EnvVarItem, 0, len(envVars))
	for _, env := range envVars {
		value := env.Value
		// Mask secret values
		if env.IsSecret {
			value = maskSecretValue(env.Value)
		}
		items = append(items, EnvVarItem{
			Key:       env.Key,
			Value:     value,
			IsSecret:  env.IsSecret,
			CreatedAt: env.CreatedAt,
			UpdatedAt: env.UpdatedAt,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"app_id":    appID,
		"variables": items,
	})
}

func maskSecretValue(value string) string {
	if len(value) <= 8 {
		return "********"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func isSecretKey(key string) bool {
	upperKey := strings.ToUpper(key)
	secretIndicators := []string{"SECRET", "KEY", "PASSWORD", "TOKEN", "PRIVATE", "CREDENTIAL", "AUTH", "CRED"}
	for _, indicator := range secretIndicators {
		if strings.Contains(upperKey, indicator) {
			return true
		}
	}
	return false
}

// UpdateEnvVarsRequest represents the request body for updating env vars
type UpdateEnvVarsRequest struct {
	Variables  []EnvVarInput `json:"variables"`
	DeleteKeys []string      `json:"delete_keys"`
}

// EnvVarInput represents an environment variable input
type EnvVarInput struct {
	Key      string `json:"key" binding:"required"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
}

// UpdateEnvVars handles PUT /apps/:id/env
func (h *Handler) UpdateEnvVars(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	var req UpdateEnvVarsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Invalid request body",
			RequestID: requestID,
		})
		return
	}
	
	// Check app exists
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	updatedCount := 0
	deletedCount := 0
	masterKey := h.config.MasterKey
	
	// Update or insert variables
	for _, v := range req.Variables {
		// Encrypt value if it's a secret
		value := v.Value
		if v.IsSecret && masterKey != "" {
			encrypted, err := encryptValue(value, masterKey)
			if err == nil {
				value = encrypted
			}
		}
		
		// Check if exists
		var existing database.AppEnvVar
		err := h.repo.GetEnvVar(ctx, appID, v.Key, &existing)
		
		if err == sql.ErrNoRows {
			// Insert new
			now := time.Now()
			_, err = h.repo.InsertEnvVar(ctx, &database.AppEnvVar{
				AppID:     appID,
				Key:       v.Key,
				Value:     value,
				IsSecret:  v.IsSecret,
				CreatedAt: now,
				UpdatedAt: now,
			})
			if err == nil {
				updatedCount++
			}
		} else if err == nil {
			// Update existing
			err = h.repo.UpdateEnvVar(ctx, appID, v.Key, value, v.IsSecret)
			if err == nil {
				updatedCount++
			}
		}
	}
	
	// Delete specified keys
	for _, key := range req.DeleteKeys {
		err := h.repo.DeleteEnvVar(ctx, appID, key)
		if err == nil {
			deletedCount++
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":       "Environment variables updated. Restart app to apply changes.",
		"updated_count": updatedCount,
		"deleted_count": deletedCount,
	})
}

// ImportEnvVarsRequest represents the request body for importing env vars
type ImportEnvVarsRequest struct {
	Content string `json:"content" binding:"required"`
}

// ImportEnvVars handles POST /apps/:id/env/import
func (h *Handler) ImportEnvVars(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	var req ImportEnvVarsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Invalid request body",
			RequestID: requestID,
		})
		return
	}
	
	// Check app exists
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	// Parse .env content
	envVars, err := parseEnvFile(req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_env_file",
			Message:   "Failed to parse .env file",
			RequestID: requestID,
		})
		return
	}
	
	imported := 0
	skipped := 0
	now := time.Now()
	
	for key, value := range envVars {
		// Check if exists
		var existing database.AppEnvVar
		err := h.repo.GetEnvVar(ctx, appID, key, &existing)
		
		isSecret := isSecretKey(key)
		
		if err == sql.ErrNoRows {
			// Insert new
			_, err = h.repo.InsertEnvVar(ctx, &database.AppEnvVar{
				AppID:     appID,
				Key:       key,
				Value:     value,
				IsSecret:  isSecret,
				CreatedAt: now,
				UpdatedAt: now,
			})
			if err == nil {
				imported++
			} else {
				skipped++
			}
		} else {
			skipped++
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":  "Imported " + strconv.Itoa(imported) + " variables.",
		"imported": imported,
		"skipped":  skipped,
	})
}

func parseEnvFile(content string) (map[string]string, error) {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Find the first = sign
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}
		
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		
		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}
		
		result[key] = value
	}
	
	return result, nil
}

// AddToRepo adds missing methods to AppRepository
func (r *AppRepository) GetEnvVar(ctx context.Context, appID, key string, env *database.AppEnvVar) error {
	return r.db.GetContext(ctx, env, 
		"SELECT * FROM app_env_vars WHERE app_id = ? AND key = ?", appID, key)
}

func (r *AppRepository) InsertEnvVar(ctx context.Context, env *database.AppEnvVar) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO app_env_vars (app_id, key, value, is_secret, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		env.AppID, env.Key, env.Value, env.IsSecret, env.CreatedAt, env.UpdatedAt)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *AppRepository) UpdateEnvVar(ctx context.Context, appID, key, value string, isSecret bool) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE app_env_vars SET value = ?, is_secret = ?, updated_at = ? WHERE app_id = ? AND key = ?",
		value, isSecret, time.Now(), appID, key)
	return err
}

func (r *AppRepository) DeleteEnvVar(ctx context.Context, appID, key string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM app_env_vars WHERE app_id = ? AND key = ?", appID, key)
	return err
}

func encryptValue(value, masterKey string) (string, error) {
	// Use the real AES-256-GCM encryption service
	return services.Encrypt(value, masterKey)
}

// Volume handlers

// GetVolumes handles GET /apps/:id/volumes
func (h *Handler) GetVolumes(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	// Check app exists
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	volumes, err := h.repo.GetAppVolumes(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get volumes",
			RequestID: requestID,
		})
		return
	}
	
	type VolumeItem struct {
		ID            string    `json:"id"`
		HostPath      string    `json:"host_path"`
		ContainerPath string    `json:"container_path"`
		SizeMB        int       `json:"size_mb"`
		CreatedAt     time.Time `json:"created_at"`
	}
	
	items := make([]VolumeItem, 0, len(volumes))
	for _, v := range volumes {
		items = append(items, VolumeItem{
			ID:            v.ID,
			HostPath:      v.HostPath,
			ContainerPath: v.ContainerPath,
			SizeMB:        v.SizeMB,
			CreatedAt:     v.CreatedAt,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"app_id":  appID,
		"volumes": items,
	})
}

// CreateVolumeRequest represents the request body for creating a volume
type CreateVolumeRequest struct {
	ContainerPath string `json:"container_path" binding:"required"`
	Name          string `json:"name"`
}

// CreateVolume handles POST /apps/:id/volumes
func (h *Handler) CreateVolume(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	var req CreateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Invalid request body",
			RequestID: requestID,
		})
		return
	}
	
	// Check app exists
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	// Generate volume ID
	volumeID, err := generateID("vol_")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to generate volume ID",
			RequestID: requestID,
		})
		return
	}
	
	// Determine name if not provided
	name := req.Name
	if name == "" {
		// Use container path basename
		parts := strings.Split(strings.Trim(req.ContainerPath, "/"), "/")
		name = parts[len(parts)-1]
	}
	
	// Create host path
	hostPath := "/var/panel/apps/" + appID + "/volumes/" + name
	now := time.Now()
	
	volume := &database.AppVolume{
		ID:            volumeID,
		AppID:         appID,
		Name:          name,
		HostPath:      hostPath,
		ContainerPath: req.ContainerPath,
		SizeMB:        0,
		CreatedAt:     now,
	}
	
	if err := h.repo.CreateVolume(ctx, volume); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create volume",
			RequestID: requestID,
		})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"id":            volumeID,
		"host_path":     hostPath,
		"container_path": req.ContainerPath,
		"size_mb":       0,
		"created_at":    now,
	})
}

// DeleteVolume handles DELETE /apps/:id/volumes/:volume_id
func (h *Handler) DeleteVolume(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	volumeID := c.Param("volume_id")
	ctx := context.Background()
	
	deleteData := c.Query("delete_data") == "true"
	
	// Check volume exists and belongs to app
	volume, err := h.repo.GetAppVolumeByID(ctx, volumeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get volume",
			RequestID: requestID,
		})
		return
	}
	if volume == nil || volume.AppID != appID {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "volume_not_found",
			Message:   "Volume not found",
			RequestID: requestID,
		})
		return
	}
	
	// Delete from database
	if err := h.repo.DeleteVolume(ctx, volumeID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to delete volume",
			RequestID: requestID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":      "Volume removed from app configuration.",
		"data_deleted": deleteData,
	})
}

// AddToRepo adds missing volume methods to AppRepository
func (r *AppRepository) CreateVolume(ctx context.Context, volume *database.AppVolume) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO app_volumes (id, app_id, name, host_path, container_path, size_mb, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		volume.ID, volume.AppID, volume.Name, volume.HostPath, volume.ContainerPath, volume.SizeMB, volume.CreatedAt)
	return err
}

func (r *AppRepository) DeleteVolume(ctx context.Context, volumeID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM app_volumes WHERE id = ?", volumeID)
	return err
}

// Deployment handlers

// ListDeployments handles GET /apps/:id/deployments
func (h *Handler) ListDeployments(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	status := c.DefaultQuery("status", "all")
	page := 1
	perPage := 20
	
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	
	if perPageStr := c.Query("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}
	
	// Check app exists
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	deps, total, err := h.repo.GetDeploymentsByAppID(ctx, appID, status, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get deployments",
			RequestID: requestID,
		})
		return
	}
	
	items := make([]database.DeploymentListItem, 0, len(deps))
	for _, d := range deps {
		triggeredByUser := "system"
		if d.TriggeredByUserID != nil {
			triggeredByUser, _ = h.repo.GetUsernameByID(ctx, *d.TriggeredByUserID)
		}
		
		items = append(items, database.DeploymentListItem{
			ID:                   d.ID,
			AppID:                d.AppID,
			Status:               d.Status,
			Commit:               d.CommitSHA,
			CommitMessage:        d.CommitMessage,
			CommitAuthor:         d.CommitAuthor,
			Branch:               d.Branch,
			BuildDurationSeconds: d.BuildDurationSeconds,
			DeployDurationSeconds: d.DeployDurationSeconds,
			StartedAt:            d.StartedAt,
			CompletedAt:          d.CompletedAt,
			TriggeredBy:          d.TriggeredBy,
			TriggeredByUser:      triggeredByUser,
		})
	}
	
	totalPages := (total + perPage - 1) / perPage
	
	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{
			"total":       total,
			"page":        page,
			"per_page":    perPage,
			"total_pages": totalPages,
		},
	})
}

// TriggerDeploymentRequest represents the request body for triggering a deployment
type TriggerDeploymentRequest struct {
	Branch string `json:"branch,omitempty"`
	Commit string `json:"commit,omitempty"`
	Force  bool   `json:"force,omitempty"`
}

// TriggerDeployment handles POST /apps/:id/deploy
func (h *Handler) TriggerDeployment(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	var req TriggerDeploymentRequest
	c.ShouldBindJSON(&req) // Optional body
	
	// Check app exists
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	// Check for existing in-progress deployment
	deps, _, err := h.repo.GetDeploymentsByAppID(ctx, appID, "in_progress", 1, 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to check deployments",
			RequestID: requestID,
		})
		return
	}
	if len(deps) > 0 && !req.Force {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:     "deployment_in_progress",
			Message:   "A deployment is already in progress. Use force=true to override.",
			RequestID: requestID,
		})
		return
	}
	
	// Create deployment record
	deploymentID, _ := generateID("dep_")
	userID := c.GetInt("user_id")
	now := time.Now()
	
	branch := req.Branch
	if branch == "" {
		// Get branch from app source config
		var sourceConfig database.SourceConfig
		database.ParseJSONField(&app.SourceConfig, &sourceConfig)
		branch = sourceConfig.Branch
	}
	
	// Get build config
	var buildConfig database.BuildConfig
	if app.BuildConfig != nil {
		database.ParseJSONField(app.BuildConfig, &buildConfig)
	}
	
	deployment := &database.Deployment{
		ID:             deploymentID,
		AppID:          appID,
		Status:         "queued",
		CommitSHA:      strPtr(req.Commit),
		Branch:         strPtr(branch),
		BuildStrategy:  &app.BuildStrategy,
		TriggeredBy:    "manual",
		TriggeredByUserID: &userID,
		CreatedAt:      now,
	}
	
	if err := h.repo.CreateDeployment(ctx, deployment); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create deployment",
			RequestID: requestID,
		})
		return
	}
	
	// Update app status
	h.repo.UpdateAppStatus(ctx, appID, "deploying")
	
	// Emit deployment started event
	if h.wsHub != nil {
		websocket.EmitAppDeployStarted(h.wsHub, appID, deploymentID, req.Commit, branch)
		websocket.EmitAppDeployProgress(h.wsHub, appID, deploymentID, "queued", "Deployment queued", 0)
	}
	
	// Execute deployment asynchronously
	go h.executeDeployment(app, deployment, buildConfig)
	
	c.JSON(http.StatusAccepted, gin.H{
		"deployment_id": deploymentID,
		"status":       "queued",
		"message":      "Deployment started.",
	})
}

// executeDeployment performs the actual deployment process
func (h *Handler) executeDeployment(app *database.App, deployment *database.Deployment, buildConfig database.BuildConfig) {
	ctx := context.Background()
	deploymentID := deployment.ID
	appID := app.ID
	
	// Update deployment status to in_progress
	h.repo.UpdateDeploymentStatus(ctx, deploymentID, "in_progress")
	
	if h.wsHub != nil {
		websocket.EmitAppDeployProgress(h.wsHub, appID, deploymentID, "cloning", "Cloning repository...", 10)
	}
	
	// Parse source config to get repo URL
	var sourceConfig database.SourceConfig
	database.ParseJSONField(&app.SourceConfig, &sourceConfig)
	
	// Build parameters
	buildParams := agent.BuildParams{
		AppID:         appID,
		AppName:       app.Name,
		RepoURL:       sourceConfig.RepoURL,
		Branch:        *deployment.Branch,
		Commit:        "",
		BuildStrategy: app.BuildStrategy,
		BuildCommand:  buildConfig.BuildCommand,
		StartCommand:  buildConfig.StartCommand,
	}
	
	// Execute build via agent client
	buildResult, err := h.agent.Build(ctx, buildParams)
	if err != nil {
		h.handleDeploymentFailure(ctx, appID, deploymentID, err.Error(), "build")
		return
	}
	
	// Check build success
	if !buildResult.Success {
		h.handleDeploymentFailure(ctx, appID, deploymentID, buildResult.Error, "build")
		return
	}
	
	if h.wsHub != nil {
		websocket.EmitAppDeployProgress(h.wsHub, appID, deploymentID, "building", "Image built successfully", 50)
	}
	
	// Get environment variables
	envVars, err := h.repo.GetAppEnvVars(ctx, appID)
	if err != nil {
		envVars = []database.AppEnvVar{}
	}
	
	// Prepare env vars map for container
	envMap := make(map[string]string)
	for _, env := range envVars {
		// Decrypt secrets if needed
		value := env.Value
		if env.IsSecret && h.config.MasterKey != "" {
			// Decrypt value
			decrypted, err := h.decryptEnvValue(env.Value, h.config.MasterKey)
			if err == nil {
				value = decrypted
			}
		}
		envMap[env.Key] = value
	}
	
	// Add app-specific environment variables
	envMap["APP_ID"] = appID
	envMap["APP_NAME"] = app.Name
	envMap["APP_RUNTIME"] = app.Runtime
	
	// Get volumes
	volumes, _ := h.repo.GetAppVolumes(ctx, appID)
	volumeMounts := make([]agent.VolumeMount, 0, len(volumes))
	for _, v := range volumes {
		volumeMounts = append(volumeMounts, agent.VolumeMount{
			HostPath:      v.HostPath,
			ContainerPath: v.ContainerPath,
			ReadOnly:      false,
		})
	}
	
	// Add app data volume
	appVolumePath := "/var/panel/apps/" + appID + "/data"
	volumeMounts = append(volumeMounts, agent.VolumeMount{
		HostPath:      appVolumePath,
		ContainerPath: "/app/data",
		ReadOnly:      false,
	})
	
	// Run container
	if h.wsHub != nil {
		websocket.EmitAppDeployProgress(h.wsHub, appID, deploymentID, "running", "Starting container...", 70)
	}
	
	runParams := agent.RunParams{
		AppID:   appID,
		Image:   buildResult.ImageName,
		EnvVars: envMap,
		Volumes: volumeMounts,
		Ports: []agent.PortMapping{
			{Internal: app.InternalPort, External: 0}, // auto-assign external port
		},
		Network:     "panel_apps",
		Restart:     app.RestartPolicy,
		MemoryLimit: getMemoryLimitString(app.MemoryLimitMB),
		CPUQuota:    getCPUQuota(app.CPULimit),
	}

	runResult, err := h.agent.Run(ctx, runParams)
	if err != nil {
		h.handleDeploymentFailure(ctx, appID, deploymentID, err.Error(), "container")
		return
	}

	// Update app with container info
	h.repo.UpdateAppContainer(ctx, appID, runResult.ContainerID, buildResult.ImageName, runResult.Port)

	// Start health check for the container
	h.agent.StartHealthCheck(ctx, agent.HealthCheckParams{
		AppID:       appID,
		ContainerID: runResult.ContainerID,
		Port:        runResult.Port,
		Path:        "/",
		Interval:    30,
		Timeout:     5,
		Retries:     3,
	})

	// Update deployment status to success
	h.repo.UpdateDeploymentStatus(ctx, deploymentID, "success")
	
	// Calculate duration
	durationSeconds := int(time.Since(deployment.CreatedAt).Seconds())
	
	if h.wsHub != nil {
		websocket.EmitAppDeploySuccess(h.wsHub, appID, deploymentID, durationSeconds)
	}
	
	// Update app status to running
	h.repo.UpdateAppStatus(ctx, appID, "running")
}

// handleDeploymentFailure handles a failed deployment
func (h *Handler) handleDeploymentFailure(ctx context.Context, appID, deploymentID, errorMsg, step string) {
	h.repo.UpdateDeploymentStatus(ctx, deploymentID, "failed")
	h.repo.UpdateAppStatus(ctx, appID, "failed")
	
	if h.wsHub != nil {
		websocket.EmitAppDeployFailed(h.wsHub, appID, deploymentID, errorMsg, step)
	}
}

// decryptEnvValue decrypts an encrypted environment variable value
func (h *Handler) decryptEnvValue(encryptedValue, masterKey string) (string, error) {
	// Use the real AES-256-GCM decryption service
	return services.Decrypt(encryptedValue, masterKey)
}

// getMemoryLimitString converts memory limit MB to Docker memory string format
func getMemoryLimitString(memoryLimitMB *int) string {
	if memoryLimitMB == nil || *memoryLimitMB == 0 {
		return ""
	}
	return fmt.Sprintf("%dm", *memoryLimitMB)
}

// getCPUQuota converts CPU limit to Docker CPU quota (microseconds)
func getCPUQuota(cpuLimit *float64) int64 {
	if cpuLimit == nil || *cpuLimit == 0 {
		return 0
	}
	// Convert CPU cores to quota (e.g., 0.5 cores = 50000 microseconds per 100ms)
	return int64(*cpuLimit * 100000)
}

// RollbackRequest represents the request body for rollback
type RollbackRequest struct {
	DeploymentID string `json:"deployment_id" binding:"required"`
}

// Rollback handles POST /apps/:id/rollback
func (h *Handler) Rollback(c *gin.Context) {
	requestID := c.GetString("request_id")
	appID := c.Param("id")
	ctx := context.Background()
	
	var req RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Deployment ID required",
			RequestID: requestID,
		})
		return
	}
	
	// Check app exists
	app, err := h.repo.GetAppByID(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get app",
			RequestID: requestID,
		})
		return
	}
	if app == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "app_not_found",
			Message:   "App not found",
			RequestID: requestID,
		})
		return
	}
	
	// Check target deployment exists and belongs to app
	targetDep, err := h.repo.GetDeploymentByID(ctx, req.DeploymentID)
	if err != nil || targetDep == nil || targetDep.AppID != appID {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "deployment_not_found",
			Message:   "Deployment not found",
			RequestID: requestID,
		})
		return
	}
	
	// Create new deployment for rollback
	newDeploymentID, _ := generateID("dep_")
	userID := c.GetInt("user_id")
	now := time.Now()
	
	// Get build config
	var buildConfig database.BuildConfig
	if app.BuildConfig != nil {
		database.ParseJSONField(app.BuildConfig, &buildConfig)
	}
	
	newDep := &database.Deployment{
		ID:              newDeploymentID,
		AppID:           appID,
		Status:          "queued",
		CommitSHA:       targetDep.CommitSHA,
		CommitMessage:   strPtr("Rollback to " + req.DeploymentID),
		Branch:          targetDep.Branch,
		BuildStrategy:   targetDep.BuildStrategy,
		TriggeredBy:     "rollback",
		TriggeredByUserID: &userID,
		RollbackOfID:    strPtr(req.DeploymentID),
		CreatedAt:       now,
	}
	
	if err := h.repo.CreateDeployment(ctx, newDep); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create rollback deployment",
			RequestID: requestID,
		})
		return
	}
	
	// Update app status
	h.repo.UpdateAppStatus(ctx, appID, "deploying")
	
	// Emit deployment started event
	if h.wsHub != nil {
		websocket.EmitAppDeployStarted(h.wsHub, appID, newDeploymentID, "", "")
		websocket.EmitAppDeployProgress(h.wsHub, appID, newDeploymentID, "queued", "Rollback deployment queued", 0)
	}
	
	// Execute rollback asynchronously
	go h.executeRollback(app, newDep, targetDep, buildConfig)
	
	c.JSON(http.StatusAccepted, gin.H{
		"message":             "Rollback initiated.",
		"new_deployment_id":   newDeploymentID,
		"target_deployment_id": req.DeploymentID,
	})
}

// executeRollback performs a rollback deployment
func (h *Handler) executeRollback(app *database.App, deployment *database.Deployment, targetDep *database.Deployment, buildConfig database.BuildConfig) {
	ctx := context.Background()
	deploymentID := deployment.ID
	appID := app.ID
	
	// Update deployment status to in_progress
	h.repo.UpdateDeploymentStatus(ctx, deploymentID, "in_progress")
	
	if h.wsHub != nil {
		websocket.EmitAppDeployProgress(h.wsHub, appID, deploymentID, "rollback", "Starting rollback...", 10)
	}
	
	// Parse source config to get repo URL
	var sourceConfig database.SourceConfig
	database.ParseJSONField(&app.SourceConfig, &sourceConfig)
	
	// Build parameters
	buildParams := agent.BuildParams{
		AppID:         appID,
		AppName:       app.Name,
		RepoURL:       sourceConfig.RepoURL,
		Branch:        *deployment.Branch,
		Commit:        "",
		BuildStrategy: app.BuildStrategy,
		BuildCommand:  buildConfig.BuildCommand,
		StartCommand:  buildConfig.StartCommand,
	}
	
	// Execute build via agent client
	buildResult, err := h.agent.Build(ctx, buildParams)
	if err != nil {
		h.handleDeploymentFailure(ctx, appID, deploymentID, err.Error(), "build")
		return
	}
	
	if !buildResult.Success {
		h.handleDeploymentFailure(ctx, appID, deploymentID, buildResult.Error, "build")
		return
	}
	
	if h.wsHub != nil {
		websocket.EmitAppDeployProgress(h.wsHub, appID, deploymentID, "building", "Rollback image built", 50)
	}
	
	// Get environment variables
	envVars, err := h.repo.GetAppEnvVars(ctx, appID)
	if err != nil {
		envVars = []database.AppEnvVar{}
	}
	
	envMap := make(map[string]string)
	for _, env := range envVars {
		value := env.Value
		if env.IsSecret && h.config.MasterKey != "" {
			decrypted, err := h.decryptEnvValue(env.Value, h.config.MasterKey)
			if err == nil {
				value = decrypted
			}
		}
		envMap[env.Key] = value
	}
	
	envMap["APP_ID"] = appID
	envMap["APP_NAME"] = app.Name
	envMap["APP_RUNTIME"] = app.Runtime
	
	// Get volumes
	volumes, _ := h.repo.GetAppVolumes(ctx, appID)
	volumeMounts := make([]agent.VolumeMount, 0, len(volumes))
	for _, v := range volumes {
		volumeMounts = append(volumeMounts, agent.VolumeMount{
			HostPath:      v.HostPath,
			ContainerPath: v.ContainerPath,
			ReadOnly:      false,
		})
	}
	
	appVolumePath := "/var/panel/apps/" + appID + "/data"
	volumeMounts = append(volumeMounts, agent.VolumeMount{
		HostPath:      appVolumePath,
		ContainerPath: "/app/data",
		ReadOnly:      false,
	})
	
	// Run container
	if h.wsHub != nil {
		websocket.EmitAppDeployProgress(h.wsHub, appID, deploymentID, "running", "Starting container...", 70)
	}
	
	runParams := agent.RunParams{
		AppID:   appID,
		Image:   buildResult.ImageName,
		EnvVars: envMap,
		Volumes: volumeMounts,
		Ports: []agent.PortMapping{
			{Internal: app.InternalPort, External: 0},
		},
		Network:     "panel_apps",
		Restart:     app.RestartPolicy,
		MemoryLimit: getMemoryLimitString(app.MemoryLimitMB),
		CPUQuota:    getCPUQuota(app.CPULimit),
	}
	
	runResult, err := h.agent.Run(ctx, runParams)
	if err != nil {
		h.handleDeploymentFailure(ctx, appID, deploymentID, err.Error(), "container")
		return
	}
	
	// Update app with container info
	h.repo.UpdateAppContainer(ctx, appID, runResult.ContainerID, buildResult.ImageName, runResult.Port)
	
	// Update deployment status to success
	h.repo.UpdateDeploymentStatus(ctx, deploymentID, "success")
	
	durationSeconds := int(time.Since(deployment.CreatedAt).Seconds())
	
	if h.wsHub != nil {
		websocket.EmitAppDeploySuccess(h.wsHub, appID, deploymentID, durationSeconds)
	}
	
	// Update app status to running
	h.repo.UpdateAppStatus(ctx, appID, "running")
}
