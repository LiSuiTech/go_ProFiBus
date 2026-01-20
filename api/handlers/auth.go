package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/go_profibus/api/middleware"
	"github.com/yourusername/go_profibus/pkg/interfaces"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService  interfaces.AuthService
	authzService interfaces.AuthorizationService
	userRepo     interfaces.UserRepository
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(
	authService interfaces.AuthService,
	authzService interfaces.AuthorizationService,
	userRepo interfaces.UserRepository,
) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		authzService: authzService,
		userRepo:     userRepo,
	}
}

// ========== Authentication Endpoints ==========

// Login handles user login
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req interfaces.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	loginResp, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if _, ok := err.(interfaces.ErrInvalidCredentials); ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		if _, ok := err.(interfaces.ErrUnauthorized); ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, loginResp)
}

// Logout handles user logout
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get token from Authorization header
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"})
		return
	}

	if err := h.authService.Logout(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "logout failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// RefreshToken handles token refresh
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"})
		return
	}

	loginResp, err := h.authService.RefreshToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token refresh failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, loginResp)
}

// GetMe returns the current user's information
// GET /api/v1/auth/me
func (h *AuthHandler) GetMe(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// ChangePassword handles password change for current user
// POST /api/v1/auth/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	var req interfaces.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	// Get full user (with password hash)
	fullUser, err := h.userRepo.GetUser(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user: " + err.Error()})
		return
	}

	// Verify old password
	if err := h.authService.VerifyPassword(req.OldPassword, fullUser.PasswordHash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid old password"})
		return
	}

	// Hash new password
	newHash, err := h.authService.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password: " + err.Error()})
		return
	}

	// Update password
	if err := h.userRepo.ChangePassword(c.Request.Context(), user.ID, newHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to change password: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

// ========== User Management Endpoints ==========

// CreateUser creates a new user (admin only)
// POST /api/v1/users
func (h *AuthHandler) CreateUser(c *gin.Context) {
	var req interfaces.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	// Hash password
	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password: " + err.Error()})
		return
	}

	// Create user
	user := &interfaces.User{
		ID:           generateUserID(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		FullName:     req.FullName,
		RoleIDs:      req.RoleIDs,
		Enabled:      true,
	}

	if err := h.userRepo.CreateUser(c.Request.Context(), user); err != nil {
		if _, ok := err.(interfaces.ErrUserExists); ok {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user: " + err.Error()})
		return
	}

	// Don't return password hash
	user.PasswordHash = ""

	c.JSON(http.StatusCreated, user)
}

// ListUsers lists all users (admin only)
// GET /api/v1/users
func (h *AuthHandler) ListUsers(c *gin.Context) {
	// Parse query parameters
	var enabled *bool
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabledVal := enabledStr == "true"
		enabled = &enabledVal
	}

	users, err := h.userRepo.ListUsers(c.Request.Context(), enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users: " + err.Error()})
		return
	}

	// Remove password hashes
	for _, user := range users {
		user.PasswordHash = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

// GetUser retrieves a user by ID
// GET /api/v1/users/:id
func (h *AuthHandler) GetUser(c *gin.Context) {
	id := c.Param("id")

	user, err := h.userRepo.GetUser(c.Request.Context(), id)
	if err != nil {
		if _, ok := err.(interfaces.ErrUserNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user: " + err.Error()})
		return
	}

	// Don't return password hash
	user.PasswordHash = ""

	c.JSON(http.StatusOK, user)
}

// UpdateUser updates an existing user
// PUT /api/v1/users/:id
func (h *AuthHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var req interfaces.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	// Get existing user
	user, err := h.userRepo.GetUser(c.Request.Context(), id)
	if err != nil {
		if _, ok := err.(interfaces.ErrUserNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user: " + err.Error()})
		return
	}

	// Update fields if provided
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	if req.RoleIDs != nil {
		user.RoleIDs = req.RoleIDs
	}
	if req.Enabled != nil {
		user.Enabled = *req.Enabled
	}

	if err := h.userRepo.UpdateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user: " + err.Error()})
		return
	}

	// Don't return password hash
	user.PasswordHash = ""

	c.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user
// DELETE /api/v1/users/:id
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// Prevent deleting self
	currentUser, ok := middleware.GetUserFromContext(c)
	if ok && currentUser.ID == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete yourself"})
		return
	}

	if err := h.userRepo.DeleteUser(c.Request.Context(), id); err != nil {
		if _, ok := err.(interfaces.ErrUserNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

// ========== Role Management Endpoints ==========

// CreateRole creates a new role (admin only)
// POST /api/v1/roles
func (h *AuthHandler) CreateRole(c *gin.Context) {
	var role interfaces.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if role.ID == "" {
		role.ID = generateRoleID()
	}

	if err := h.userRepo.CreateRole(c.Request.Context(), &role); err != nil {
		if _, ok := err.(interfaces.ErrRoleExists); ok {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create role: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, role)
}

// ListRoles lists all roles
// GET /api/v1/roles
func (h *AuthHandler) ListRoles(c *gin.Context) {
	roles, err := h.userRepo.ListRoles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list roles: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
		"count": len(roles),
	})
}

// GetRole retrieves a role by ID
// GET /api/v1/roles/:id
func (h *AuthHandler) GetRole(c *gin.Context) {
	id := c.Param("id")

	role, err := h.userRepo.GetRole(c.Request.Context(), id)
	if err != nil {
		if _, ok := err.(interfaces.ErrRoleNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get role: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, role)
}

// UpdateRole updates an existing role
// PUT /api/v1/roles/:id
func (h *AuthHandler) UpdateRole(c *gin.Context) {
	id := c.Param("id")

	var role interfaces.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	role.ID = id

	if err := h.userRepo.UpdateRole(c.Request.Context(), &role); err != nil {
		if _, ok := err.(interfaces.ErrRoleNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, role)
}

// DeleteRole deletes a role
// DELETE /api/v1/roles/:id
func (h *AuthHandler) DeleteRole(c *gin.Context) {
	id := c.Param("id")

	// Prevent deleting predefined roles
	if id == "role-admin" || id == "role-operator" || id == "role-viewer" || id == "role-editor" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete predefined roles"})
		return
	}

	if err := h.userRepo.DeleteRole(c.Request.Context(), id); err != nil {
		if _, ok := err.(interfaces.ErrRoleNotFound); ok {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete role: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role deleted successfully"})
}

// ========== User-Role Assignment Endpoints ==========

// AssignRoleToUser assigns a role to a user
// POST /api/v1/users/:id/roles/:role_id
func (h *AuthHandler) AssignRoleToUser(c *gin.Context) {
	userID := c.Param("id")
	roleID := c.Param("role_id")

	if err := h.authzService.AssignRoleToUser(c.Request.Context(), userID, roleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign role: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role assigned successfully"})
}

// RemoveRoleFromUser removes a role from a user
// DELETE /api/v1/users/:id/roles/:role_id
func (h *AuthHandler) RemoveRoleFromUser(c *gin.Context) {
	userID := c.Param("id")
	roleID := c.Param("role_id")

	if err := h.authzService.RemoveRoleFromUser(c.Request.Context(), userID, roleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove role: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role removed successfully"})
}

// GetUserRoles gets all roles for a user
// GET /api/v1/users/:id/roles
func (h *AuthHandler) GetUserRoles(c *gin.Context) {
	userID := c.Param("id")

	roles, err := h.authzService.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user roles: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
		"count": len(roles),
	})
}

// ========== Helper Functions ==========

// extractToken extracts the Bearer token from Authorization header
func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

// generateUserID generates a unique user ID
func generateUserID() string {
	return "user-" + uuid.New().String()[:8]
}

// generateRoleID generates a unique role ID
func generateRoleID() string {
	return "role-" + uuid.New().String()[:8]
}
