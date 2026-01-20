package interfaces

import (
	"context"
	"time"
)

// User represents a system user
type User struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Never expose password hash in JSON
	FullName     string    `json:"full_name" db:"full_name"`
	RoleIDs      []string  `json:"role_ids" db:"role_ids"`
	Enabled      bool      `json:"enabled" db:"enabled"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at" db:"last_login_at"`
}

// Role represents a user role with permissions
type Role struct {
	ID          string   `json:"id" db:"id"`
	Name        string   `json:"name" db:"name"`
	Description string   `json:"description" db:"description"`
	Permissions []string `json:"permissions" db:"permissions"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Permission represents a specific permission
type Permission struct {
	ID          string `json:"id" db:"id"`
	Resource    string `json:"resource" db:"resource"`       // e.g., "rule", "analyzer", "processor", "pipeline"
	Action      string `json:"action" db:"action"`           // e.g., "create", "read", "update", "delete"
	Description string `json:"description" db:"description"`
}

// Session represents an active user session
type Session struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	IPAddress string    `json:"ip_address" db:"ip_address"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	User      *User     `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Username string   `json:"username" binding:"required"`
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=8"`
	FullName string   `json:"full_name"`
	RoleIDs  []string `json:"role_ids"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	Email    *string  `json:"email,omitempty"`
	FullName *string  `json:"full_name,omitempty"`
	RoleIDs  []string `json:"role_ids,omitempty"`
	Enabled  *bool    `json:"enabled,omitempty"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	// Login authenticates a user and returns a session token
	Login(ctx context.Context, username, password string) (*LoginResponse, error)

	// Logout invalidates a session token
	Logout(ctx context.Context, token string) error

	// ValidateToken validates a session token and returns the user
	ValidateToken(ctx context.Context, token string) (*User, error)

	// RefreshToken refreshes a session token
	RefreshToken(ctx context.Context, token string) (*LoginResponse, error)

	// HashPassword hashes a password
	HashPassword(password string) (string, error)

	// VerifyPassword verifies a password against a hash
	VerifyPassword(password, hash string) error
}

// UserRepository defines the interface for user persistence
type UserRepository interface {
	// User CRUD operations
	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, id string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	ListUsers(ctx context.Context, enabled *bool) ([]*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error
	ChangePassword(ctx context.Context, userID string, passwordHash string) error
	UpdateLastLogin(ctx context.Context, userID string) error

	// Role CRUD operations
	CreateRole(ctx context.Context, role *Role) error
	GetRole(ctx context.Context, id string) (*Role, error)
	ListRoles(ctx context.Context) ([]*Role, error)
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id string) error

	// Permission operations
	GetRolePermissions(ctx context.Context, roleID string) ([]string, error)
	GetUserPermissions(ctx context.Context, userID string) ([]string, error)

	// Session operations
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	CleanExpiredSessions(ctx context.Context) error
}

// AuthorizationService defines the interface for authorization operations
type AuthorizationService interface {
	// CheckPermission checks if a user has a specific permission
	CheckPermission(ctx context.Context, userID string, resource string, action string) (bool, error)

	// GetUserRoles gets all roles for a user
	GetUserRoles(ctx context.Context, userID string) ([]*Role, error)

	// AssignRoleToUser assigns a role to a user
	AssignRoleToUser(ctx context.Context, userID string, roleID string) error

	// RemoveRoleFromUser removes a role from a user
	RemoveRoleFromUser(ctx context.Context, userID string, roleID string) error
}

// Standard RBAC Errors
type ErrUnauthorized struct {
	Message string
}

func (e ErrUnauthorized) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "unauthorized"
}

type ErrForbidden struct {
	Resource string
	Action   string
}

func (e ErrForbidden) Error() string {
	return "forbidden: insufficient permissions for " + e.Action + " on " + e.Resource
}

type ErrUserNotFound struct {
	Identifier string
}

func (e ErrUserNotFound) Error() string {
	return "user not found: " + e.Identifier
}

type ErrUserExists struct {
	Username string
}

func (e ErrUserExists) Error() string {
	return "user already exists: " + e.Username
}

type ErrInvalidCredentials struct{}

func (e ErrInvalidCredentials) Error() string {
	return "invalid username or password"
}

type ErrSessionExpired struct{}

func (e ErrSessionExpired) Error() string {
	return "session expired"
}

type ErrRoleNotFound struct {
	ID string
}

func (e ErrRoleNotFound) Error() string {
	return "role not found: " + e.ID
}

type ErrRoleExists struct {
	Name string
}

func (e ErrRoleExists) Error() string {
	return "role already exists: " + e.Name
}

// Common permission constants
const (
	// Resource types
	ResourceRule      = "rule"
	ResourceAnalyzer  = "analyzer"
	ResourceProcessor = "processor"
	ResourcePipeline  = "pipeline"
	ResourceUser      = "user"
	ResourceRole      = "role"
	ResourceConfig    = "config"
	ResourceMetrics   = "metrics"
	ResourceTrace     = "trace"

	// Action types
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionExecute = "execute"
	ActionExport = "export"
	ActionImport = "import"

	// Predefined roles
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
	RoleEditor   = "editor"
)

// FormatPermission creates a permission string from resource and action
func FormatPermission(resource, action string) string {
	return resource + ":" + action
}

// ParsePermission parses a permission string into resource and action
func ParsePermission(permission string) (resource, action string) {
	// Simple split by colon
	parts := make([]string, 2)
	for i, part := range []byte(permission) {
		if part == ':' {
			parts[0] = permission[:i]
			parts[1] = permission[i+1:]
			break
		}
	}
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return permission, ""
}
