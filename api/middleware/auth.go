package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go_ProFiBus/pkg/interfaces"
)

// ContextKey type for context keys
type ContextKey string

const (
	// UserContextKey is the key for storing user in context
	UserContextKey ContextKey = "user"
)

// AuthMiddleware creates authentication middleware
func AuthMiddleware(authService interfaces.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		user, err := authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			if _, ok := err.(interfaces.ErrSessionExpired); ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			}
			c.Abort()
			return
		}

		// Store user in context
		ctx := context.WithValue(c.Request.Context(), UserContextKey, user)
		c.Request = c.Request.WithContext(ctx)

		// Also set in Gin context for easier access
		c.Set("user", user)
		c.Set("user_id", user.ID)

		c.Next()
	}
}

// RequirePermission creates middleware that requires specific permission
func RequirePermission(authzService interfaces.AuthorizationService, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
			c.Abort()
			return
		}

		user, ok := userInterface.(*interfaces.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user type in context"})
			c.Abort()
			return
		}

		// Check permission
		hasPermission, err := authzService.CheckPermission(c.Request.Context(), user.ID, resource, action)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check permissions"})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error":    "forbidden",
				"message":  "insufficient permissions",
				"required": interfaces.FormatPermission(resource, action),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission creates middleware that requires any of the specified permissions
func RequireAnyPermission(authzService interfaces.AuthorizationService, permissions map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
			c.Abort()
			return
		}

		user, ok := userInterface.(*interfaces.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user type in context"})
			c.Abort()
			return
		}

		// Check if user has any of the required permissions
		for resource, action := range permissions {
			hasPermission, err := authzService.CheckPermission(c.Request.Context(), user.ID, resource, action)
			if err != nil {
				continue
			}
			if hasPermission {
				c.Next()
				return
			}
		}

		// No permission found
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "insufficient permissions",
		})
		c.Abort()
	}
}

// RequireRole creates middleware that requires a specific role
func RequireRole(authzService interfaces.AuthorizationService, roleID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
			c.Abort()
			return
		}

		user, ok := userInterface.(*interfaces.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user type in context"})
			c.Abort()
			return
		}

		// Check if user has the required role
		hasRole := false
		for _, userRoleID := range user.RoleIDs {
			if userRoleID == roleID {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error":    "forbidden",
				"message":  "required role not found",
				"required": roleID,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth creates optional authentication middleware (doesn't block if no auth provided)
func OptionalAuth(authService interfaces.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// Check if it's a Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]

		// Validate token
		user, err := authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.Next()
			return
		}

		// Store user in context
		ctx := context.WithValue(c.Request.Context(), UserContextKey, user)
		c.Request = c.Request.WithContext(ctx)
		c.Set("user", user)
		c.Set("user_id", user.ID)

		c.Next()
	}
}

// GetUserFromContext extracts user from Gin context
func GetUserFromContext(c *gin.Context) (*interfaces.User, bool) {
	userInterface, exists := c.Get("user")
	if !exists {
		return nil, false
	}

	user, ok := userInterface.(*interfaces.User)
	return user, ok
}

// GetUserIDFromContext extracts user ID from Gin context
func GetUserIDFromContext(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}

	userIDStr, ok := userID.(string)
	return userIDStr, ok
}
