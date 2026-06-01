package users

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"panel-api/internal/database"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// generateInviteToken creates a unique invite token
func generateInviteToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetCurrentUser handles GET /users/me
func GetCurrentUser(c *gin.Context) {
	requestID := c.GetString("request_id")

	db := c.MustGet("db").(*database.DB)
	userID := c.GetInt("user_id")

	ctx := context.Background()
	var user database.User
	err := db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = ?", userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "user_not_found",
			Message:   "User not found",
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  user.ID,
		"username":            user.Username,
		"email":               user.Email,
		"role":                user.Role,
		"two_factor_enabled":  user.TwoFactorEnabled,
		"avatar_url":          user.AvatarURL,
		"last_login_at":       user.LastLoginAt,
		"last_login_ip":       user.LastLoginIP,
		"created_at":          user.CreatedAt,
		"updated_at":          user.UpdatedAt,
		"request_id":          requestID,
	})
}

// ListUsers handles GET /users
func ListUsers(c *gin.Context) {
	requestID := c.GetString("request_id")
	db := c.MustGet("db").(*database.DB)

	ctx := context.Background()
	var users []database.User
	err := db.SelectContext(ctx, &users, "SELECT * FROM users ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to fetch users",
			RequestID: requestID,
		})
		return
	}

	// Map to response format
	response := make([]gin.H, len(users))
	for i, user := range users {
		response[i] = gin.H{
			"id":                 user.ID,
			"username":           user.Username,
			"email":              user.Email,
			"role":               user.Role,
			"two_factor_enabled": user.TwoFactorEnabled,
			"avatar_url":         user.AvatarURL,
			"last_login_at":      user.LastLoginAt,
			"created_at":         user.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"users":      response,
		"total":      len(response),
		"request_id": requestID,
	})
}

// InviteUser handles POST /users/invite
func InviteUser(c *gin.Context) {
	requestID := c.GetString("request_id")

	type InviteRequest struct {
		Email string `json:"email" binding:"required,email"`
		Role  string `json:"role" binding:"required,oneof=admin developer viewer"`
	}

	var req InviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Valid email and role are required (admin, developer, viewer)",
			RequestID: requestID,
		})
		return
	}

	db := c.MustGet("db").(*database.DB)
	inviterID := c.GetInt("user_id")
	_ = inviterID // Used in invite creation
	cfg := c.MustGet("config").(*gin.Context).Value("config")
	if cfg == nil {
		cfg = c.MustGet("config")
	}

	ctx := context.Background()

	// Check if user already exists
	var existingUser database.User
	err := db.GetContext(ctx, &existingUser, "SELECT * FROM users WHERE email = ?", req.Email)
	if err == nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:     "user_exists",
			Message:   "A user with this email already exists",
			RequestID: requestID,
		})
		return
	}

	// Check if invite already exists
	var existingInvite database.User
	_ = db.GetContext(ctx, &existingInvite, "SELECT * FROM user_invites WHERE email = ? AND accepted_at IS NULL", req.Email)
	if err == nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:     "invite_exists",
			Message:   "An invite for this email already exists",
			RequestID: requestID,
		})
		return
	}

	// Generate invite token
	token, err := generateInviteToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to generate invite token",
			RequestID: requestID,
		})
		return
	}

	// Hash token for storage
	// In production, use bcrypt or similar

	// Set expiration to 7 days
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// Create invite
	_, err = db.ExecContext(ctx,
		`INSERT INTO user_invites (id, email, role, token_hash, invited_by, expires_at, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		token[:12], req.Email, req.Role, token, inviterID, expiresAt, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create invite",
			RequestID: requestID,
		})
		return
	}

	// Build invite URL
	inviteURL := fmt.Sprintf("/auth/register?token=%s", token[:12])

	c.JSON(http.StatusCreated, gin.H{
		"id":        token[:12],
		"email":     req.Email,
		"role":      req.Role,
		"expires_at": expiresAt,
		"invite_url": inviteURL,
		"request_id": requestID,
	})
}

// UpdateUserRole handles PUT /users/:id/role
func UpdateUserRole(c *gin.Context) {
	requestID := c.GetString("request_id")

	_ = c.GetInt("user_id") // not needed, only role is checked
	requestingUserRole := c.GetString("role")

	type UpdateRoleRequest struct {
		Role string `json:"role" binding:"required,oneof=admin developer viewer owner"`
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Valid role is required (admin, developer, viewer, owner)",
			RequestID: requestID,
		})
		return
	}

	// Only owner can change roles
	if requestingUserRole != "owner" {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:     "forbidden",
			Message:   "Only the owner can change user roles",
			RequestID: requestID,
		})
		return
	}

	targetUserID := c.Param("id")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "User ID is required",
			RequestID: requestID,
		})
		return
	}

	db := c.MustGet("db").(*database.DB)
	ctx := context.Background()

	// Get target user
	var targetUser database.User
	err := db.GetContext(ctx, &targetUser, "SELECT * FROM users WHERE id = ?", targetUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "user_not_found",
			Message:   "User not found",
			RequestID: requestID,
		})
		return
	}

	// Cannot change owner's role
	if targetUser.Role == "owner" {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:     "forbidden",
			Message:   "Cannot change owner's role",
			RequestID: requestID,
		})
		return
	}

	// Cannot remove last admin
	if targetUser.Role == "admin" && req.Role != "admin" {
		var adminCount int
		db.GetContext(ctx, &adminCount, "SELECT COUNT(*) FROM users WHERE role = 'admin'")
		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:     "last_admin",
				Message:   "Cannot remove the last admin",
				RequestID: requestID,
			})
			return
		}
	}

	// Update role
	_, err = db.ExecContext(ctx, "UPDATE users SET role = ?, updated_at = ? WHERE id = ?", req.Role, time.Now(), targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to update user role",
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         targetUserID,
		"role":       req.Role,
		"updated_at": time.Now(),
		"request_id": requestID,
	})
}

// DeleteUser handles DELETE /users/:id
func DeleteUser(c *gin.Context) {
	requestID := c.GetString("request_id")

	requestingUserRole := c.GetString("role")
	if requestingUserRole != "owner" {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:     "forbidden",
			Message:   "Only the owner can delete users",
			RequestID: requestID,
		})
		return
	}

	targetUserID := c.Param("id")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "User ID is required",
			RequestID: requestID,
		})
		return
	}

	db := c.MustGet("db").(*database.DB)
	ctx := context.Background()

	// Get target user
	var targetUser database.User
	err := db.GetContext(ctx, &targetUser, "SELECT * FROM users WHERE id = ?", targetUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "user_not_found",
			Message:   "User not found",
			RequestID: requestID,
		})
		return
	}

	// Cannot delete owner
	if targetUser.Role == "owner" {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:     "forbidden",
			Message:   "Cannot delete the owner",
			RequestID: requestID,
		})
		return
	}

	// Delete user (cascades to sessions, api_keys, etc.)
	_, err = db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to delete user",
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "User deleted successfully",
		"request_id": requestID,
	})
}

// GetUserInvites handles GET /users/invites
func GetUserInvites(c *gin.Context) {
	requestID := c.GetString("request_id")
	db := c.MustGet("db").(*database.DB)

	ctx := context.Background()
	var invites []struct {
		ID          string     `db:"id"`
		Email       string     `db:"email"`
		Role        string     `db:"role"`
		InvitedBy   int        `db:"invited_by"`
		ExpiresAt   time.Time  `db:"expires_at"`
		AcceptedAt  *time.Time `db:"accepted_at"`
		CreatedAt   time.Time  `db:"created_at"`
	}

	err := db.SelectContext(ctx, &invites, `
		SELECT id, email, role, invited_by, expires_at, accepted_at, created_at 
		FROM user_invites 
		WHERE accepted_at IS NULL 
		ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to fetch invites",
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invites":    invites,
		"total":      len(invites),
		"request_id": requestID,
	})
}

// CancelInvite handles DELETE /users/invites/:id
func CancelInvite(c *gin.Context) {
	requestID := c.GetString("request_id")
	inviteID := c.Param("id")

	if inviteID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Invite ID is required",
			RequestID: requestID,
		})
		return
	}

	db := c.MustGet("db").(*database.DB)
	ctx := context.Background()

	result, err := db.ExecContext(ctx, "DELETE FROM user_invites WHERE id = ? AND accepted_at IS NULL", inviteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to cancel invite",
			RequestID: requestID,
		})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "not_found",
			Message:   "Invite not found or already accepted",
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Invite cancelled successfully",
		"request_id": requestID,
	})
}

// AcceptInvite handles POST /users/invites/:id/accept
func AcceptInvite(c *gin.Context) {
	requestID := c.GetString("request_id")
	inviteID := c.Param("id")

	type AcceptInviteRequest struct {
		Username string `json:"username" binding:"required,min=3,max=32"`
		Password string `json:"password" binding:"required,min=8"`
	}

	var req AcceptInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Username and password are required",
			RequestID: requestID,
		})
		return
	}

	db := c.MustGet("db").(*database.DB)
	ctx := context.Background()

	// Get invite
	var invite struct {
		ID         string    `db:"id"`
		Email      string    `db:"email"`
		Role       string    `db:"role"`
		InvitedBy  int       `db:"invited_by"`
		ExpiresAt  time.Time `db:"expires_at"`
		AcceptedAt *time.Time `db:"accepted_at"`
	}

	err := db.GetContext(ctx, &invite, "SELECT * FROM user_invites WHERE id = ?", inviteID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "invite_not_found",
			Message:   "Invite not found",
			RequestID: requestID,
		})
		return
	}

	if invite.AcceptedAt != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invite_used",
			Message:   "Invite has already been used",
			RequestID: requestID,
		})
		return
	}

	if time.Now().After(invite.ExpiresAt) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invite_expired",
			Message:   "Invite has expired",
			RequestID: requestID,
		})
		return
	}

	// Validate username
	if strings.Contains(req.Username, "@") {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_username",
			Message:   "Username cannot contain @",
			RequestID: requestID,
		})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to hash password",
			RequestID: requestID,
		})
		return
	}

	// Create user
	result, err := db.ExecContext(ctx,
		`INSERT INTO users (username, email, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		req.Username, invite.Email, hashedPassword, invite.Role, time.Now(), time.Now(),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:     "username_taken",
				Message:   "Username is already taken",
				RequestID: requestID,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create user",
			RequestID: requestID,
		})
		return
	}

	userID, _ := result.LastInsertId()

	// Mark invite as accepted
	db.ExecContext(ctx,
		"UPDATE user_invites SET accepted_at = ?, accepted_by_user_id = ? WHERE id = ?",
		time.Now(), userID, inviteID,
	)

	c.JSON(http.StatusCreated, gin.H{
		"id":         userID,
		"username":   req.Username,
		"email":      invite.Email,
		"role":       invite.Role,
		"request_id": requestID,
	})
}

// ListAPIKeys handles GET /users/me/api-keys
func ListAPIKeys(c *gin.Context) {
	requestID := c.GetString("request_id")
	db := c.MustGet("db").(*database.DB)
	userID := c.GetInt("user_id")
	ctx := context.Background()

	var keys []database.APIKey
	err := db.SelectContext(ctx, &keys,
		"SELECT id, user_id, name, scopes, last_used_at, expires_at, created_at FROM api_keys WHERE user_id = ? AND revoked_at IS NULL ORDER BY created_at DESC",
		userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to fetch API keys",
			RequestID: requestID,
		})
		return
	}

	// Mask token in response (only show last 4 chars)
	response := make([]gin.H, len(keys))
	for i, key := range keys {
		response[i] = gin.H{
			"id":           key.ID,
			"name":         key.Name,
			"scopes":       key.Scopes,
			"last_used_at": key.LastUsedAt,
			"expires_at":   key.ExpiresAt,
			"created_at":   key.CreatedAt,
			"masked_token": "sk_live_****" + key.TokenHash[len(key.TokenHash)-4:],
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

// CreateAPIKey handles POST /users/me/api-keys
func CreateAPIKey(c *gin.Context) {
	requestID := c.GetString("request_id")
	db := c.MustGet("db").(*database.DB)
	userID := c.GetInt("user_id")
	ctx := context.Background()

	var req struct {
		Name      string `json:"name" binding:"required"`
		Scopes    string `json:"scopes" binding:"required"`
		ExpiresIn int    `json:"expires_in"` // days, 0 = no expiry
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Name and scopes are required",
			RequestID: requestID,
		})
		return
	}

	// Generate token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to generate token",
			RequestID: requestID,
		})
		return
	}
	token := "sk_live_" + hex.EncodeToString(tokenBytes)

	// Hash token for storage
	h := sha256.New()
	h.Write([]byte(token))
	tokenHash := hex.EncodeToString(h.Sum(nil))

	// Generate key ID
	keyIDBytes := make([]byte, 8)
	if _, err := rand.Read(keyIDBytes); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to generate key ID",
			RequestID: requestID,
		})
		return
	}
	keyID := "key_" + hex.EncodeToString(keyIDBytes)

	// Calculate expiry
	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		t := time.Now().AddDate(0, 0, req.ExpiresIn)
		expiresAt = &t
	}

	var insertErr error
	_, insertErr = db.ExecContext(ctx,
		"INSERT INTO api_keys (id, user_id, name, token_hash, scopes, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		keyID, userID, req.Name, tokenHash, req.Scopes, expiresAt, time.Now())
	if insertErr != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create API key",
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        keyID,
		"name":      req.Name,
		"scopes":    req.Scopes,
		"token":     token, // Only shown once!
		"expires_at": expiresAt,
		"created_at": time.Now(),
	})
}

// RevokeAPIKey handles DELETE /users/me/api-keys/:id
func RevokeAPIKey(c *gin.Context) {
	requestID := c.GetString("request_id")
	db := c.MustGet("db").(*database.DB)
	userID := c.GetInt("user_id")
	keyID := c.Param("id")
	ctx := context.Background()

	result, err := db.ExecContext(ctx,
		"UPDATE api_keys SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL",
		time.Now(), keyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to revoke API key",
			RequestID: requestID,
		})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:     "not_found",
			Message:   "API key not found",
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key revoked successfully",
	})
}
