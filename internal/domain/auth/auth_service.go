package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/yourusername/go_profibus/pkg/interfaces"
	"golang.org/x/crypto/bcrypt"
)

// AuthService implements the interfaces.AuthService interface
type AuthService struct {
	userRepo          interfaces.UserRepository
	sessionDuration   time.Duration
	bcryptCost        int
}

// NewAuthService creates a new AuthService
func NewAuthService(userRepo interfaces.UserRepository) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		sessionDuration: 24 * time.Hour, // Default 24 hours
		bcryptCost:      bcrypt.DefaultCost,
	}
}

// SetSessionDuration sets the session duration
func (s *AuthService) SetSessionDuration(duration time.Duration) {
	s.sessionDuration = duration
}

// Login authenticates a user and returns a session token
func (s *AuthService) Login(ctx context.Context, username, password string) (*interfaces.LoginResponse, error) {
	// Get user by username
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		if _, ok := err.(interfaces.ErrUserNotFound); ok {
			return nil, interfaces.ErrInvalidCredentials{}
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is enabled
	if !user.Enabled {
		return nil, interfaces.ErrUnauthorized{Message: "user account is disabled"}
	}

	// Verify password
	if err := s.VerifyPassword(password, user.PasswordHash); err != nil {
		return nil, interfaces.ErrInvalidCredentials{}
	}

	// Generate session token
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Create session
	expiresAt := time.Now().Add(s.sessionDuration)
	session := &interfaces.Session{
		ID:        generateID("session"),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	if err := s.userRepo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login time
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	// Don't return password hash
	user.PasswordHash = ""

	return &interfaces.LoginResponse{
		Token:     token,
		User:      user,
		ExpiresAt: expiresAt,
	}, nil
}

// Logout invalidates a session token
func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.userRepo.DeleteSession(ctx, token)
}

// ValidateToken validates a session token and returns the user
func (s *AuthService) ValidateToken(ctx context.Context, token string) (*interfaces.User, error) {
	// Get session
	session, err := s.userRepo.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		_ = s.userRepo.DeleteSession(ctx, token)
		return nil, interfaces.ErrSessionExpired{}
	}

	// Get user
	user, err := s.userRepo.GetUser(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	// Check if user is still enabled
	if !user.Enabled {
		_ = s.userRepo.DeleteSession(ctx, token)
		return nil, interfaces.ErrUnauthorized{Message: "user account is disabled"}
	}

	// Don't return password hash
	user.PasswordHash = ""

	return user, nil
}

// RefreshToken refreshes a session token
func (s *AuthService) RefreshToken(ctx context.Context, token string) (*interfaces.LoginResponse, error) {
	// Validate current token
	user, err := s.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Delete old session
	_ = s.userRepo.DeleteSession(ctx, token)

	// Generate new token
	newToken, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Create new session
	expiresAt := time.Now().Add(s.sessionDuration)
	session := &interfaces.Session{
		ID:        generateID("session"),
		UserID:    user.ID,
		Token:     newToken,
		ExpiresAt: expiresAt,
	}

	if err := s.userRepo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &interfaces.LoginResponse{
		Token:     newToken,
		User:      user,
		ExpiresAt: expiresAt,
	}, nil
}

// HashPassword hashes a password using bcrypt
func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword verifies a password against a hash
func (s *AuthService) VerifyPassword(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return fmt.Errorf("password verification failed: %w", err)
	}
	return nil
}

// AuthorizationService implements the interfaces.AuthorizationService interface
type AuthorizationService struct {
	userRepo interfaces.UserRepository
}

// NewAuthorizationService creates a new AuthorizationService
func NewAuthorizationService(userRepo interfaces.UserRepository) *AuthorizationService {
	return &AuthorizationService{
		userRepo: userRepo,
	}
}

// CheckPermission checks if a user has a specific permission
func (s *AuthorizationService) CheckPermission(ctx context.Context, userID string, resource string, action string) (bool, error) {
	// Get user permissions
	permissions, err := s.userRepo.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Check if user has the required permission
	requiredPermission := interfaces.FormatPermission(resource, action)
	for _, permission := range permissions {
		if permission == requiredPermission {
			return true, nil
		}
	}

	return false, nil
}

// GetUserRoles gets all roles for a user
func (s *AuthorizationService) GetUserRoles(ctx context.Context, userID string) ([]*interfaces.Role, error) {
	// Get user
	user, err := s.userRepo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get roles
	var roles []*interfaces.Role
	for _, roleID := range user.RoleIDs {
		role, err := s.userRepo.GetRole(ctx, roleID)
		if err != nil {
			continue // Skip roles that don't exist
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// AssignRoleToUser assigns a role to a user
func (s *AuthorizationService) AssignRoleToUser(ctx context.Context, userID string, roleID string) error {
	// Verify role exists
	_, err := s.userRepo.GetRole(ctx, roleID)
	if err != nil {
		return err
	}

	// Get user
	user, err := s.userRepo.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Check if role is already assigned
	for _, existingRoleID := range user.RoleIDs {
		if existingRoleID == roleID {
			return nil // Already assigned
		}
	}

	// Add role
	user.RoleIDs = append(user.RoleIDs, roleID)

	// Update user
	return s.userRepo.UpdateUser(ctx, user)
}

// RemoveRoleFromUser removes a role from a user
func (s *AuthorizationService) RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error {
	// Get user
	user, err := s.userRepo.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Remove role
	var newRoleIDs []string
	for _, existingRoleID := range user.RoleIDs {
		if existingRoleID != roleID {
			newRoleIDs = append(newRoleIDs, existingRoleID)
		}
	}

	user.RoleIDs = newRoleIDs

	// Update user
	return s.userRepo.UpdateUser(ctx, user)
}

// ========== Helper Functions ==========

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateID generates a unique ID with a prefix
func generateID(prefix string) string {
	token, _ := generateSecureToken(8)
	return prefix + "-" + token
}
