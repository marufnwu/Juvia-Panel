package middleware

import (
	"errors"
	"net/http"
	"strings"

	"panel-api/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Auth is JWT authentication middleware
// Reads token from (in order):
//  1. HttpOnly cookie "access_token"
//  2. Authorization: Bearer <token> header
func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		var tokenString string

		// Try cookie first (more secure - not accessible via JS)
		if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
			tokenString = cookie
		}

		// Fallback to Authorization header (for API clients)
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":      "unauthorized",
					"message":    "Authentication required",
					"request_id": requestID,
				})
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":      "invalid_token",
					"message":    "Invalid authorization header format. Use: Bearer <token>",
					"request_id": requestID,
				})
				return
			}
			tokenString = parts[1]
		}

		// Parse and validate token
		claims := &AccessClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":      "invalid_token",
				"message":    "Invalid or expired access token",
				"request_id": requestID,
			})
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AccessClaims for JWT validation
type AccessClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// RequireRole creates middleware that checks if user has one of the allowed roles
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		userRole := c.GetString("role")

		if userRole == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":      "unauthorized",
				"message":    "Authentication required",
				"request_id": requestID,
			})
			return
		}

		// Check if user's role is in allowed list
		allowed := false
		for _, role := range allowedRoles {
			if userRole == role {
				allowed = true
				break
			}
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":      "forbidden",
				"message":    "Insufficient permissions",
				"request_id": requestID,
			})
			return
		}

		c.Next()
	}
}

// RequireAuth is a convenience alias for RequireRole with all roles
// This ensures the user is authenticated but any role is allowed
func RequireAuth() gin.HandlerFunc {
	return RequireRole("owner", "admin", "developer", "viewer")
}