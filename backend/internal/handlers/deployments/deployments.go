package deployments

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"panel-api/internal/agent"
	"panel-api/internal/database"
	"panel-api/internal/websocket"

	"github.com/gin-gonic/gin"
)

// Handler handles deployment-related HTTP requests
type Handler struct {
	repo  interface {
		GetDeploymentByID(ctx context.Context, id string) (*database.Deployment, error)
		GetDeploymentsByAppID(ctx context.Context, appID string, status string, page, perPage int) ([]database.Deployment, int, error)
		UpdateDeploymentStatus(ctx context.Context, id string, status string) error
		UpdateDeploymentLogs(ctx context.Context, id string, logs string, durationSeconds int) error
		GetUsernameByID(ctx context.Context, userID int) (string, error)
	}
	db   *database.DB
	agent *agent.Client
	wsHub *websocket.Hub
}

// NewHandler creates a new deployments handler
func NewHandler(db *database.DB, repo interface {
	GetDeploymentByID(ctx context.Context, id string) (*database.Deployment, error)
	GetDeploymentsByAppID(ctx context.Context, appID string, status string, page, perPage int) ([]database.Deployment, int, error)
	UpdateDeploymentStatus(ctx context.Context, id string, status string) error
	UpdateDeploymentLogs(ctx context.Context, id string, logs string, durationSeconds int) error
	GetUsernameByID(ctx context.Context, userID int) (string, error)
}, agentClient *agent.Client, ws *websocket.Hub) *Handler {
	return &Handler{
		repo:  repo,
		db:    db,
		agent: agentClient,
		wsHub: ws,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string      `json:"error"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// GetDeployment handles GET /deployments/:id
func (h *Handler) GetDeployment(c *gin.Context) {
	requestID := c.GetString("request_id")
	deploymentID := c.Param("id")
	ctx := context.Background()
	
	dep, err := h.repo.GetDeploymentByID(ctx, deploymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get deployment",
			RequestID: requestID,
		})
		return
	}
	
	if dep == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "deployment_not_found",
			Message:   "Deployment not found",
			RequestID: requestID,
		})
		return
	}
	
	triggeredByUser := "system"
	if dep.TriggeredByUserID != nil {
		triggeredByUser, _ = h.repo.GetUsernameByID(ctx, *dep.TriggeredByUserID)
	}
	
	detail := database.DeploymentDetail{
		ID:                   dep.ID,
		AppID:                dep.AppID,
		Status:               dep.Status,
		Commit:               dep.CommitSHA,
		CommitMessage:        dep.CommitMessage,
		CommitAuthor:         dep.CommitAuthor,
		Branch:               dep.Branch,
		BuildLogsURL:         "/api/v1/deployments/" + dep.ID + "/logs",
		BuildDurationSeconds: dep.BuildDurationSeconds,
		DeployDurationSeconds: dep.DeployDurationSeconds,
		StartedAt:            dep.StartedAt,
		CompletedAt:          dep.CompletedAt,
		TriggeredBy:          dep.TriggeredBy,
		TriggeredByUser:      triggeredByUser,
	}
	
	c.JSON(http.StatusOK, detail)
}

// GetDeploymentLogs handles GET /deployments/:id/logs
func (h *Handler) GetDeploymentLogs(c *gin.Context) {
	requestID := c.GetString("request_id")
	deploymentID := c.Param("id")
	ctx := context.Background()
	
	dep, err := h.repo.GetDeploymentByID(ctx, deploymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get deployment",
			RequestID: requestID,
		})
		return
	}
	
	if dep == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "deployment_not_found",
			Message:   "Deployment not found",
			RequestID: requestID,
		})
		return
	}
	
	// Parse logs if available
	logs := []database.DeploymentLogLine{}
	if dep.BuildLogs != nil && *dep.BuildLogs != "" {
		// Build logs are stored as text, parse each line
		logs = parseBuildLogs(*dep.BuildLogs)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"deployment_id": deploymentID,
		"lines":         logs,
	})
}

// parseBuildLogs parses build log text into structured log lines
func parseBuildLogs(logText string) []database.DeploymentLogLine {
	lines := []database.DeploymentLogLine{}
	
	// Simple line-by-line parsing
	// In a real implementation, logs would be structured JSON or have consistent format
	lineStrs := splitLines(logText)
	
	for _, line := range lineStrs {
		if line == "" {
			continue
		}
		
		level := "info"
		if contains(line, "ERROR") || contains(line, "error") || contains(line, "Failed") {
			level = "error"
		} else if contains(line, "WARN") || contains(line, "warning") {
			level = "warning"
		}
		
		lines = append(lines, database.DeploymentLogLine{
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     level,
			Message:   line,
		})
	}
	
	return lines
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// CancelDeployment handles POST /deployments/:id/cancel
func (h *Handler) CancelDeployment(c *gin.Context) {
	requestID := c.GetString("request_id")
	deploymentID := c.Param("id")
	ctx := context.Background()
	
	dep, err := h.repo.GetDeploymentByID(ctx, deploymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get deployment",
			RequestID: requestID,
		})
		return
	}
	
	if dep == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "deployment_not_found",
			Message:   "Deployment not found",
			RequestID: requestID,
		})
		return
	}
	
	// Can only cancel queued or in_progress deployments
	if dep.Status != "queued" && dep.Status != "in_progress" {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:     "deployment_not_cancellable",
			Message:   "Deployment cannot be cancelled in its current state",
			RequestID: requestID,
		})
		return
	}
	
	if err := h.repo.UpdateDeploymentStatus(ctx, deploymentID, "cancelled"); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to cancel deployment",
			RequestID: requestID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Deployment cancelled.",
		"status":  "cancelled",
	})
}

// DeploymentRepositoryAdapter adapts the app repository for deployment queries
type DeploymentRepositoryAdapter struct {
	db *database.DB
}

// NewDeploymentRepositoryAdapter creates a new adapter
func NewDeploymentRepositoryAdapter(db *database.DB) *DeploymentRepositoryAdapter {
	return &DeploymentRepositoryAdapter{db: db}
}

// GetDeploymentByID returns a deployment by ID
func (r *DeploymentRepositoryAdapter) GetDeploymentByID(ctx context.Context, id string) (*database.Deployment, error) {
	var dep database.Deployment
	err := r.db.GetContext(ctx, &dep, "SELECT * FROM deployments WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &dep, nil
}

// GetDeploymentsByAppID returns deployments for an app
func (r *DeploymentRepositoryAdapter) GetDeploymentsByAppID(ctx context.Context, appID string, status string, page, perPage int) ([]database.Deployment, int, error) {
	query := "SELECT * FROM deployments WHERE app_id = ?"
	countQuery := "SELECT COUNT(*) FROM deployments WHERE app_id = ?"
	var args []interface{}
	args = append(args, appID)
	
	if status != "" && status != "all" {
		query += " AND status = ?"
		countQuery += " AND status = ?"
		args = append(args, status)
	}
	
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	offset := (page - 1) * perPage
	args = append(args, perPage, offset)
	
	var deps []database.Deployment
	err = r.db.SelectContext(ctx, &deps, query, args...)
	if err != nil {
		return nil, 0, err
	}
	if deps == nil {
		deps = []database.Deployment{}
	}
	
	return deps, total, nil
}

// UpdateDeploymentStatus updates deployment status
func (r *DeploymentRepositoryAdapter) UpdateDeploymentStatus(ctx context.Context, id string, status string) error {
	query := "UPDATE deployments SET status = ?"
	args := []interface{}{status}
	
	if status == "in_progress" {
		query += ", started_at = ?"
		args = append(args, time.Now())
	} else if status == "success" || status == "failed" || status == "cancelled" {
		query += ", completed_at = ?"
		args = append(args, time.Now())
	}
	
	query += " WHERE id = ?"
	args = append(args, id)
	
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// UpdateDeploymentLogs updates deployment build logs
func (r *DeploymentRepositoryAdapter) UpdateDeploymentLogs(ctx context.Context, id string, logs string, durationSeconds int) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE deployments SET build_logs = ?, build_duration_seconds = ? WHERE id = ?",
		logs, durationSeconds, id)
	return err
}

// GetUsernameByID returns username by user ID
func (r *DeploymentRepositoryAdapter) GetUsernameByID(ctx context.Context, userID int) (string, error) {
	var username string
	err := r.db.GetContext(ctx, &username, "SELECT username FROM users WHERE id = ?", userID)
	if err == sql.ErrNoRows {
		return "unknown", nil
	}
	return username, err
}

// ListDeploymentsByStatus lists deployments with optional status filter
func (h *Handler) ListDeploymentsByStatus(c *gin.Context) {
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
	
	deps, total, err := h.repo.GetDeploymentsByAppID(ctx, appID, status, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to get deployments",
			RequestID: requestID,
		})
		return
	}
	
	type DeploymentListItem struct {
		ID                   string     `json:"id"`
		AppID                string     `json:"app_id"`
		Status               string     `json:"status"`
		Commit               *string    `json:"commit"`
		CommitMessage        *string    `json:"commit_message"`
		CommitAuthor         *string    `json:"commit_author"`
		Branch               *string    `json:"branch"`
		BuildDurationSeconds *int       `json:"build_duration_seconds"`
		DeployDurationSeconds *int      `json:"deploy_duration_seconds"`
		StartedAt            *time.Time `json:"started_at"`
		CompletedAt          *time.Time `json:"completed_at"`
		TriggeredBy          string     `json:"triggered_by"`
		TriggeredByUser      string     `json:"triggered_by_user"`
	}
	
	items := make([]DeploymentListItem, 0, len(deps))
	for _, d := range deps {
		triggeredByUser := "system"
		if d.TriggeredByUserID != nil {
			triggeredByUser, _ = h.repo.GetUsernameByID(ctx, *d.TriggeredByUserID)
		}
		
		items = append(items, DeploymentListItem{
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
