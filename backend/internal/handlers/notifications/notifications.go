package notifications

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"panel-api/internal/database"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *database.DB
}

func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

type Notification struct {
	ID        string     `json:"id"`
	UserID    int        `json:"user_id"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	Severity  string     `json:"severity"`
	Link      *string    `json:"link,omitempty"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type ListNotificationsParams struct {
	Page    int
	PerPage int
	UnreadOnly bool
}

func (h *Handler) ListNotifications(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := c.GetInt("user_id")
	ctx := context.Background()

	params := ListNotificationsParams{
		Page:    1,
		PerPage: 50,
		UnreadOnly: c.Query("unread") == "true",
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

	whereClause := "user_id = ?"
	args := []interface{}{userID}

	if params.UnreadOnly {
		whereClause += " AND read_at IS NULL"
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM notifications WHERE " + whereClause
	err := h.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to count notifications",
			"request_id": requestID,
		})
		return
	}

	offset := (params.Page - 1) * params.PerPage
	query := `
		SELECT id, user_id, title, message, severity, link, read_at, created_at
		FROM notifications
		WHERE ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, params.PerPage, offset)

	var notifications []Notification
	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to fetch notifications",
			"request_id": requestID,
		})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var n Notification
		err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.Severity, &n.Link, &n.ReadAt, &n.CreatedAt)
		if err != nil {
			continue
		}
		notifications = append(notifications, n)
	}

	if notifications == nil {
		notifications = []Notification{}
	}

	totalPages := (total + params.PerPage - 1) / params.PerPage

	c.JSON(http.StatusOK, gin.H{
		"data": notifications,
		"meta": gin.H{
			"total":       total,
			"page":        params.Page,
			"per_page":    params.PerPage,
			"total_pages": totalPages,
		},
	})
}

func (h *Handler) MarkAsRead(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := c.GetInt("user_id")
	notificationID := c.Param("id")
	ctx := context.Background()

	result, err := h.db.ExecContext(ctx,
		"UPDATE notifications SET read_at = ? WHERE id = ? AND user_id = ? AND read_at IS NULL",
		time.Now(), notificationID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to mark notification as read",
			"request_id": requestID,
		})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Notification not found",
			"request_id": requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notification marked as read",
	})
}

func (h *Handler) MarkAllAsRead(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := c.GetInt("user_id")
	ctx := context.Background()

	_, err := h.db.ExecContext(ctx,
		"UPDATE notifications SET read_at = ? WHERE user_id = ? AND read_at IS NULL",
		time.Now(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to mark all notifications as read",
			"request_id": requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All notifications marked as read",
	})
}

func (h *Handler) DeleteNotification(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := c.GetInt("user_id")
	notificationID := c.Param("id")
	ctx := context.Background()

	result, err := h.db.ExecContext(ctx,
		"DELETE FROM notifications WHERE id = ? AND user_id = ?",
		notificationID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete notification",
			"request_id": requestID,
		})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Notification not found",
			"request_id": requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notification deleted",
	})
}

func (h *Handler) GetUnreadCount(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := c.GetInt("user_id")
	ctx := context.Background()

	var count int
	err := h.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at IS NULL", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get unread count",
			"request_id": requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unread_count": count,
	})
}