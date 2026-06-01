package activity

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"panel-api/internal/config"
	"panel-api/internal/database"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *database.DB
	cfg *config.Config
}

func NewHandler(db *database.DB, cfg *config.Config) *Handler {
	return &Handler{db: db, cfg: cfg}
}

type ActivityLogEntry struct {
	ID           string    `json:"id"`
	UserID       *int      `json:"user_id,omitempty"`
	UserUsername string    `json:"user_username"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   *string   `json:"resource_id,omitempty"`
	Details      *string   `json:"details,omitempty"`
	IPAddress    *string   `json:"ip_address,omitempty"`
	UserAgent    *string   `json:"user_agent,omitempty"`
	CreatedAt    string    `json:"created_at"`
}

type ListActivityParams struct {
	Page    int
	PerPage int
	Action  string
	UserID  int
	ResourceType string
	ResourceID   string
}

func (h *Handler) ListActivity(c *gin.Context) {
	params := ListActivityParams{
		Page:    1,
		PerPage: 50,
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

	params.Action = c.Query("action")
	params.ResourceType = c.Query("resource_type")
	params.ResourceID = c.Query("resource_id")

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := strconv.Atoi(userIDStr); err == nil {
			params.UserID = userID
		}
	}

	entries, total, err := h.getActivityLogs(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to fetch activity logs",
		})
		return
	}

	totalPages := (total + params.PerPage - 1) / params.PerPage

	c.JSON(http.StatusOK, gin.H{
		"data": entries,
		"meta": gin.H{
			"total":       total,
			"page":        params.Page,
			"per_page":    params.PerPage,
			"total_pages": totalPages,
		},
	})
}

func (h *Handler) getActivityLogs(ctx context.Context, params ListActivityParams) ([]ActivityLogEntry, int, error) {
	var entries []ActivityLogEntry
	var total int

	offset := (params.Page - 1) * params.PerPage

	whereClause := "1=1"
	args := []interface{}{}

	if params.Action != "" {
		whereClause += " AND action = ?"
		args = append(args, params.Action)
	}

	if params.UserID > 0 {
		whereClause += " AND user_id = ?"
		args = append(args, params.UserID)
	}

	if params.ResourceType != "" {
		whereClause += " AND resource_type = ?"
		args = append(args, params.ResourceType)
	}

	if params.ResourceID != "" {
		whereClause += " AND resource_id = ?"
		args = append(args, params.ResourceID)
	}

	countQuery := "SELECT COUNT(*) FROM activity_log WHERE " + whereClause
	err := h.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, user_username, action, resource_type, resource_id, details, ip_address, user_agent, created_at
		FROM activity_log
		WHERE ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, params.PerPage, offset)

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry ActivityLogEntry
		var userID *int
		var resourceID, details, ipAddress, userAgent *string

		err := rows.Scan(
			&entry.ID,
			&userID,
			&entry.UserUsername,
			&entry.Action,
			&entry.ResourceType,
			&resourceID,
			&details,
			&ipAddress,
			&userAgent,
			&entry.CreatedAt,
		)
		if err != nil {
			continue
		}

		entry.UserID = userID
		entry.ResourceID = resourceID
		entry.Details = details
		entry.IPAddress = ipAddress
		entry.UserAgent = userAgent

		entries = append(entries, entry)
	}

	if entries == nil {
		entries = []ActivityLogEntry{}
	}

	return entries, total, nil
}

type CreateActivityLogEntry struct {
	UserID       *int
	UserUsername string
	Action       string
	ResourceType string
	ResourceID   *string
	Details      *string
	IPAddress    *string
	UserAgent    *string
}

func (h *Handler) LogActivity(entry CreateActivityLogEntry) error {
	query := `
		INSERT INTO activity_log (user_id, user_username, action, resource_type, resource_id, details, ip_address, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`

	_, err := h.db.ExecContext(nil, query,
		entry.UserID,
		entry.UserUsername,
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		entry.Details,
		entry.IPAddress,
		entry.UserAgent,
	)

	return err
}

func ParseActionType(action string) (resourceType, actionName string) {
	parts := strings.SplitN(action, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "system", action
}