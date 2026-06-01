package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"panel-api/internal/config"
	"panel-api/internal/database"
	"panel-api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

// JWT claims for access tokens
type AccessClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWT claims for refresh tokens
type RefreshClaims struct {
	SessionID string `json:"session_id"`
	UserID   int    `json:"user_id"`
	jwt.RegisteredClaims
}

// Login request body
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is returned on successful login
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	User         *UserInfo `json:"user,omitempty"`
}

// 2FA Setup response
type TwoFactorSetupResponse struct {
	Secret     string   `json:"secret"`
	QRCodeURL  string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// Error response
type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// generateAccessToken creates a new JWT access token
func generateAccessToken(user *database.User, cfg *config.Config) (string, error) {
	claims := AccessClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.JWTExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// generateRefreshToken creates a new refresh token
func generateRefreshToken(userID int, sessionID string, cfg *config.Config) (string, error) {
	claims := RefreshClaims{
		SessionID: sessionID,
		UserID:    userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.RefreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// generateSessionID creates a unique session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashToken hashes a token for storage using SHA-256
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// generateBackupCodes generates backup codes for 2FA
func generateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		bytes := make([]byte, 8)
		if _, err := rand.Read(bytes); err != nil {
			return nil, err
		}
		codes[i] = strings.ToUpper(hex.EncodeToString(bytes)[:8])
	}
	return codes, nil
}

// Login handles POST /auth/login
func Login(c *gin.Context) {
	requestID := c.GetString("request_id")
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Invalid request body",
			RequestID: requestID,
		})
		return
	}

	cfg := c.MustGet("config").(*config.Config)
	db := c.MustGet("db").(*database.DB)

	// Find user by username or email
	ctx := context.Background()
	var user database.User
	err := db.GetContext(ctx, &user, "SELECT * FROM users WHERE username = ? OR email = ?", req.Username, req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:     "invalid_credentials",
			Message:   "Invalid username or password",
			RequestID: requestID,
		})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:     "invalid_credentials",
			Message:   "Invalid username or password",
			RequestID: requestID,
		})
		return
	}

	// Check if 2FA is enabled
	if user.TwoFactorEnabled {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:     "2fa_required",
			Message:   "Two-factor authentication is required",
			RequestID: requestID,
		})
		return
	}

	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create session",
			RequestID: requestID,
		})
		return
	}

	// Generate access token
	accessToken, err := generateAccessToken(&user, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to generate access token",
			RequestID: requestID,
		})
		return
	}

	// Generate refresh token
	refreshToken, err := generateRefreshToken(user.ID, sessionID, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to generate refresh token",
			RequestID: requestID,
		})
		return
	}

	// Store session in database
	expiresAt := time.Now().Add(cfg.RefreshExpiry)
	_, err = db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		sessionID, user.ID, hashToken(refreshToken), expiresAt, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create session",
			RequestID: requestID,
		})
		return
	}

	// Update last login
	db.ExecContext(ctx, "UPDATE users SET last_login_at = ?, last_login_ip = ? WHERE id = ?",
		time.Now(), c.ClientIP(), user.ID)

	// Set refresh token cookie
	c.SetCookie(
		"refresh_token",
		refreshToken,
		int(cfg.RefreshExpiry.Seconds()),
		"/",
		"",
		cfg.Env == "production",
		true, // httpOnly
	)

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:    int(cfg.JWTExpiry.Seconds()),
		User: &UserInfo{
			ID:       user.ID,
			Email:    user.Email,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

// Refresh handles POST /auth/refresh
func Refresh(c *gin.Context) {
	requestID := c.GetString("request_id")

	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:     "invalid_token",
			Message:   "Refresh token required",
			RequestID: requestID,
		})
		return
	}

	cfg := c.MustGet("config").(*config.Config)
	db := c.MustGet("db").(*database.DB)

	// Parse and validate refresh token
	claims := &RefreshClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:     "invalid_token",
			Message:   "Invalid or expired refresh token",
			RequestID: requestID,
		})
		return
	}

	// Verify session exists and is not expired
	ctx := context.Background()
	var session database.Session
	err = db.GetContext(ctx, &session, "SELECT * FROM sessions WHERE id = ? AND expires_at > ?",
		claims.SessionID, time.Now())
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:     "invalid_session",
			Message:   "Session not found or expired",
			RequestID: requestID,
		})
		return
	}

	// Verify refresh token hash matches stored hash
	tokenHash := hashToken(refreshToken)
	if tokenHash != session.RefreshTokenHash {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:     "invalid_token",
			Message:   "Refresh token mismatch",
			RequestID: requestID,
		})
		return
	}

	// Get user
	var user database.User
	err = db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = ?", claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:     "user_not_found",
			Message:   "User not found",
			RequestID: requestID,
		})
		return
	}

	// Generate new access token
	accessToken, err := generateAccessToken(&user, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to generate access token",
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(cfg.JWTExpiry.Seconds()),
	})
}

// Logout handles POST /auth/logout
func Logout(c *gin.Context) {
	requestID := c.GetString("request_id")

	cfg := c.MustGet("config").(*config.Config)
	db := c.MustGet("db").(*database.DB)

	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err == nil && refreshToken != "" {
		ctx := context.Background()

		// Get claims to find session
		claims := &RefreshClaims{}
		token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})

		if err == nil && token.Valid {
			// Delete session
			db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", claims.SessionID)
		}
	}

	// Clear refresh token cookie
	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/",
		"",
		cfg.Env == "production",
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Logged out successfully",
		"request_id": requestID,
	})
}

// Setup2FA handles POST /auth/2fa/setup
func Setup2FA(c *gin.Context) {
	requestID := c.GetString("request_id")

	cfg := c.MustGet("config").(*config.Config)
	db := c.MustGet("db").(*database.DB)

	// Get user from context (set by auth middleware)
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:     "unauthorized",
			Message:   "Authentication required",
			RequestID: requestID,
		})
		return
	}

	ctx := context.Background()

	// Get user
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

	// Generate TOTP secret
	secret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "JuviaPanel",
		AccountName: user.Email,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to generate 2FA secret",
			RequestID: requestID,
		})
		return
	}

	// Generate backup codes
	backupCodes, err := generateBackupCodes(8)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to generate backup codes",
			RequestID: requestID,
		})
		return
	}

	// Store 2FA secret and backup codes (not enabled yet)
	// Encrypt secret and backup codes if master key is available
	secretToStore := secret.Secret()
	backupCodesToStore := strings.Join(backupCodes, ",")
	if cfg.MasterKey != "" {
		if enc, err := services.Encrypt(secret.Secret(), cfg.MasterKey); err == nil {
			secretToStore = enc
		}
		if enc, err := services.Encrypt(strings.Join(backupCodes, ","), cfg.MasterKey); err == nil {
			backupCodesToStore = enc
		}
	}
	_, err = db.ExecContext(ctx,
		"UPDATE users SET two_factor_secret = ?, two_factor_backup_codes = ? WHERE id = ?",
		secretToStore, backupCodesToStore, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to store 2FA secret",
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusOK, TwoFactorSetupResponse{
		Secret:      secret.Secret(),
		QRCodeURL:   secret.URL(),
		BackupCodes: backupCodes,
	})
}

// Verify2FA handles POST /auth/2fa/verify
func Verify2FA(c *gin.Context) {
	requestID := c.GetString("request_id")

	type VerifyRequest struct {
		Code string `json:"code" binding:"required"`
	}

	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Verification code required",
			RequestID: requestID,
		})
		return
	}

	db := c.MustGet("db").(*database.DB)

	userID := c.GetInt("user_id")
	ctx := context.Background()

	// Get user
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

	if user.TwoFactorSecret == nil || *user.TwoFactorSecret == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "2fa_not_setup",
			Message:   "2FA not set up. Please call setup first.",
			RequestID: requestID,
		})
		return
	}

	// Decrypt 2FA secret if master key is available
	twoFactorSecret := *user.TwoFactorSecret
	cfg := c.MustGet("config").(*config.Config)
	if cfg != nil && cfg.MasterKey != "" {
		if dec, err := services.Decrypt(twoFactorSecret, cfg.MasterKey); err == nil {
			twoFactorSecret = dec
		}
	}

	// Verify TOTP code
	if !totp.Validate(req.Code, twoFactorSecret) {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:     "invalid_code",
			Message:   "Invalid verification code",
			RequestID: requestID,
		})
		return
	}

	// Enable 2FA
	_, err = db.ExecContext(ctx, "UPDATE users SET two_factor_enabled = 1 WHERE id = ?", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to enable 2FA",
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Two-factor authentication enabled",
		"request_id": requestID,
	})
}

// RegisterRequest is the request body for registration
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=8"`
}

// RegisterResponse is returned on successful registration
type RegisterResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	User         *UserInfo `json:"user,omitempty"`
}

// UserInfo represents user data returned in responses
type UserInfo struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Register handles POST /auth/register - Create first user (only works when no users exist)
func Register(c *gin.Context) {
	requestID := c.GetString("request_id")
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Invalid request body: " + err.Error(),
			RequestID: requestID,
		})
		return
	}

	cfg := c.MustGet("config").(*config.Config)
	db := c.MustGet("db").(*database.DB)

	ctx := context.Background()

	// Check if any users exist in the database
	var count int
	err := db.GetContext(ctx, &count, "SELECT COUNT(*) FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to check existing users",
			RequestID: requestID,
		})
		return
	}

	// Only allow registration when no users exist
	if count > 0 {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:     "registration_closed",
			Message:   "Registration is only allowed when no users exist. Please login instead.",
			RequestID: requestID,
		})
		return
	}

	// Validate username format (alphanumeric and underscores only)
	validUsername := true
	for _, ch := range req.Username {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			validUsername = false
			break
		}
	}
	if !validUsername {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_username",
			Message:   "Username can only contain letters, numbers, and underscores",
			RequestID: requestID,
		})
		return
	}

	// Check if username or email already exists
	var existingCount int
	err = db.GetContext(ctx, &existingCount, "SELECT COUNT(*) FROM users WHERE username = ? OR email = ?", req.Username, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to check existing user",
			RequestID: requestID,
		})
		return
	}
	if existingCount > 0 {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:     "user_exists",
			Message:   "Username or email already registered",
			RequestID: requestID,
		})
		return
	}

	// Hash password with bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to hash password",
			RequestID: requestID,
		})
		return
	}

	// Create user with 'owner' role (first user is always owner)
	now := time.Now()
	var userID int
	err = db.GetContext(ctx, &userID,
		`INSERT INTO users (username, email, password_hash, role, created_at, updated_at) 
		 VALUES (?, ?, ?, 'owner', ?, ?) 
		 RETURNING id`,
		req.Username, req.Email, string(hashedPassword), now, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create user",
			RequestID: requestID,
		})
		return
	}

	// Get the created user
	var user database.User
	err = db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = ?", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to fetch created user",
			RequestID: requestID,
		})
		return
	}

	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create session",
			RequestID: requestID,
		})
		return
	}

	// Generate access token
	accessToken, err := generateAccessToken(&user, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to generate access token",
			RequestID: requestID,
		})
		return
	}

	// Generate refresh token
	refreshToken, err := generateRefreshToken(user.ID, sessionID, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to generate refresh token",
			RequestID: requestID,
		})
		return
	}

	// Store session in database
	expiresAt := time.Now().Add(cfg.RefreshExpiry)
	_, err = db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		sessionID, user.ID, hashToken(refreshToken), expiresAt, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to create session",
			RequestID: requestID,
		})
		return
	}

	// Set refresh token cookie
	c.SetCookie(
		"refresh_token",
		refreshToken,
		int(cfg.RefreshExpiry.Seconds()),
		"/",
		"",
		cfg.Env == "production",
		true, // httpOnly
	)

	c.JSON(http.StatusCreated, RegisterResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(cfg.JWTExpiry.Seconds()),
		User: &UserInfo{
			ID:       user.ID,
			Email:    user.Email,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

// CheckUsersExists checks if any users exist in the database
func CheckUsersExists(c *gin.Context) {
	requestID := c.GetString("request_id")
	db := c.MustGet("db").(*database.DB)

	ctx := context.Background()
	var count int
	err := db.GetContext(ctx, &count, "SELECT COUNT(*) FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to check users",
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users_exist": count > 0,
		"count":       count,
	})
}

// Disable2FA handles POST /auth/2fa/disable
func Disable2FA(c *gin.Context) {
	requestID := c.GetString("request_id")

	type DisableRequest struct {
		Password string `json:"password" binding:"required"`
		Code     string `json:"code" binding:"required"`
	}

	var req DisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "invalid_request",
			Message:   "Password and code required",
			RequestID: requestID,
		})
		return
	}

	db := c.MustGet("db").(*database.DB)

	userID := c.GetInt("user_id")
	ctx := context.Background()

	// Get user
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

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:     "invalid_password",
			Message:   "Invalid password",
			RequestID: requestID,
		})
		return
	}

	// Verify TOTP code
	twoFactorSecret := ""
	if user.TwoFactorSecret != nil {
		twoFactorSecret = *user.TwoFactorSecret
	}
	cfg2fa := c.MustGet("config").(*config.Config)
	if cfg2fa != nil && cfg2fa.MasterKey != "" {
		if dec, err := services.Decrypt(twoFactorSecret, cfg2fa.MasterKey); err == nil {
			twoFactorSecret = dec
		}
	}
	if !totp.Validate(req.Code, twoFactorSecret) {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:     "invalid_code",
			Message:   "Invalid verification code",
			RequestID: requestID,
		})
		return
	}

	// Disable 2FA
	_, err = db.ExecContext(ctx,
		"UPDATE users SET two_factor_enabled = 0, two_factor_secret = NULL, two_factor_backup_codes = NULL WHERE id = ?",
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:     "internal_error",
			Message:   "Failed to disable 2FA",
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Two-factor authentication disabled",
		"request_id": requestID,
	})
}
