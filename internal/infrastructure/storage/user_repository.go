package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go_ProFiBus/pkg/interfaces"
)

// UserRepository implements the interfaces.UserRepository interface
type UserRepository struct {
	store *PostgresStore
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(store *PostgresStore) *UserRepository {
	return &UserRepository{
		store: store,
	}
}

// ========== User Operations ==========

// CreateUser creates a new user
func (r *UserRepository) CreateUser(ctx context.Context, user *interfaces.User) error {
	// Check if user already exists
	var exists bool
	err := r.store.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", user.Username).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return interfaces.ErrUserExists{Username: user.Username}
	}

	query := `
		INSERT INTO users (id, username, email, password_hash, full_name, enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = r.store.pool.Exec(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Enabled,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Assign roles if provided
	for _, roleID := range user.RoleIDs {
		if err := r.assignRoleToUser(ctx, user.ID, roleID); err != nil {
			return fmt.Errorf("failed to assign role: %w", err)
		}
	}

	return nil
}

// GetUser retrieves a user by ID
func (r *UserRepository) GetUser(ctx context.Context, id string) (*interfaces.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, enabled, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`

	var user interfaces.User
	err := r.store.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Enabled,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err == pgx.ErrNoRows {
		return nil, interfaces.ErrUserNotFound{Identifier: id}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get user roles
	user.RoleIDs, err = r.getUserRoleIDs(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*interfaces.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, enabled, created_at, updated_at, last_login_at
		FROM users
		WHERE username = $1
	`

	var user interfaces.User
	err := r.store.pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Enabled,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err == pgx.ErrNoRows {
		return nil, interfaces.ErrUserNotFound{Identifier: username}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get user roles
	user.RoleIDs, err = r.getUserRoleIDs(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*interfaces.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, enabled, created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1
	`

	var user interfaces.User
	err := r.store.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Enabled,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err == pgx.ErrNoRows {
		return nil, interfaces.ErrUserNotFound{Identifier: email}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get user roles
	user.RoleIDs, err = r.getUserRoleIDs(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	return &user, nil
}

// ListUsers lists all users with optional filtering
func (r *UserRepository) ListUsers(ctx context.Context, enabled *bool) ([]*interfaces.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, enabled, created_at, updated_at, last_login_at
		FROM users
	`

	var args []interface{}
	if enabled != nil {
		query += " WHERE enabled = $1"
		args = append(args, *enabled)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.store.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*interfaces.User
	for rows.Next() {
		var user interfaces.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.FullName,
			&user.Enabled,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.LastLoginAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		// Get user roles
		user.RoleIDs, err = r.getUserRoleIDs(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user roles: %w", err)
		}

		users = append(users, &user)
	}

	return users, nil
}

// UpdateUser updates an existing user
func (r *UserRepository) UpdateUser(ctx context.Context, user *interfaces.User) error {
	query := `
		UPDATE users
		SET email = $2, full_name = $3, enabled = $4
		WHERE id = $1
	`

	result, err := r.store.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.FullName,
		user.Enabled,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrUserNotFound{Identifier: user.ID}
	}

	// Update roles - remove all existing roles and reassign
	if _, err := r.store.pool.Exec(ctx, "DELETE FROM user_roles WHERE user_id = $1", user.ID); err != nil {
		return fmt.Errorf("failed to remove old roles: %w", err)
	}

	for _, roleID := range user.RoleIDs {
		if err := r.assignRoleToUser(ctx, user.ID, roleID); err != nil {
			return fmt.Errorf("failed to assign role: %w", err)
		}
	}

	return nil
}

// DeleteUser deletes a user
func (r *UserRepository) DeleteUser(ctx context.Context, id string) error {
	query := "DELETE FROM users WHERE id = $1"
	result, err := r.store.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrUserNotFound{Identifier: id}
	}

	return nil
}

// ChangePassword updates a user's password
func (r *UserRepository) ChangePassword(ctx context.Context, userID string, passwordHash string) error {
	query := "UPDATE users SET password_hash = $2 WHERE id = $1"
	result, err := r.store.pool.Exec(ctx, query, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("failed to change password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrUserNotFound{Identifier: userID}
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp for a user
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	query := "UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = $1"
	_, err := r.store.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// ========== Role Operations ==========

// CreateRole creates a new role
func (r *UserRepository) CreateRole(ctx context.Context, role *interfaces.Role) error {
	var exists bool
	err := r.store.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM roles WHERE name = $1)", role.Name).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check role existence: %w", err)
	}
	if exists {
		return interfaces.ErrRoleExists{Name: role.Name}
	}

	query := `
		INSERT INTO roles (id, name, description, permissions)
		VALUES ($1, $2, $3, $4)
	`

	_, err = r.store.pool.Exec(ctx, query,
		role.ID,
		role.Name,
		role.Description,
		role.Permissions,
	)

	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// GetRole retrieves a role by ID
func (r *UserRepository) GetRole(ctx context.Context, id string) (*interfaces.Role, error) {
	query := `
		SELECT id, name, description, permissions, created_at, updated_at
		FROM roles
		WHERE id = $1
	`

	var role interfaces.Role
	err := r.store.pool.QueryRow(ctx, query, id).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.Permissions,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, interfaces.ErrRoleNotFound{ID: id}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &role, nil
}

// ListRoles lists all roles
func (r *UserRepository) ListRoles(ctx context.Context) ([]*interfaces.Role, error) {
	query := `
		SELECT id, name, description, permissions, created_at, updated_at
		FROM roles
		ORDER BY name
	`

	rows, err := r.store.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []*interfaces.Role
	for rows.Next() {
		var role interfaces.Role
		err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.Permissions,
			&role.CreatedAt,
			&role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}

		roles = append(roles, &role)
	}

	return roles, nil
}

// UpdateRole updates an existing role
func (r *UserRepository) UpdateRole(ctx context.Context, role *interfaces.Role) error {
	query := `
		UPDATE roles
		SET name = $2, description = $3, permissions = $4
		WHERE id = $1
	`

	result, err := r.store.pool.Exec(ctx, query,
		role.ID,
		role.Name,
		role.Description,
		role.Permissions,
	)

	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrRoleNotFound{ID: role.ID}
	}

	return nil
}

// DeleteRole deletes a role
func (r *UserRepository) DeleteRole(ctx context.Context, id string) error {
	query := "DELETE FROM roles WHERE id = $1"
	result, err := r.store.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return interfaces.ErrRoleNotFound{ID: id}
	}

	return nil
}

// ========== Permission Operations ==========

// GetRolePermissions gets all permissions for a role
func (r *UserRepository) GetRolePermissions(ctx context.Context, roleID string) ([]string, error) {
	query := "SELECT permissions FROM roles WHERE id = $1"

	var permissions []string
	err := r.store.pool.QueryRow(ctx, query, roleID).Scan(&permissions)
	if err == pgx.ErrNoRows {
		return nil, interfaces.ErrRoleNotFound{ID: roleID}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	return permissions, nil
}

// GetUserPermissions gets all permissions for a user (aggregated from all roles)
func (r *UserRepository) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT DISTINCT unnest(r.permissions) AS permission
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
	`

	rows, err := r.store.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// ========== Session Operations ==========

// CreateSession creates a new session
func (r *UserRepository) CreateSession(ctx context.Context, session *interfaces.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, token, expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.store.pool.Exec(ctx, query,
		session.ID,
		session.UserID,
		session.Token,
		session.ExpiresAt,
		session.IPAddress,
		session.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSession retrieves a session by token
func (r *UserRepository) GetSession(ctx context.Context, token string) (*interfaces.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, ip_address, user_agent
		FROM sessions
		WHERE token = $1 AND expires_at > CURRENT_TIMESTAMP
	`

	var session interfaces.Session
	err := r.store.pool.QueryRow(ctx, query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.IPAddress,
		&session.UserAgent,
	)

	if err == pgx.ErrNoRows {
		return nil, interfaces.ErrSessionExpired{}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// DeleteSession deletes a session
func (r *UserRepository) DeleteSession(ctx context.Context, token string) error {
	query := "DELETE FROM sessions WHERE token = $1"
	_, err := r.store.pool.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// CleanExpiredSessions removes all expired sessions
func (r *UserRepository) CleanExpiredSessions(ctx context.Context) error {
	query := "DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP"
	_, err := r.store.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clean expired sessions: %w", err)
	}
	return nil
}

// ========== Helper Methods ==========

// getUserRoleIDs gets role IDs for a user
func (r *UserRepository) getUserRoleIDs(ctx context.Context, userID string) ([]string, error) {
	query := "SELECT role_id FROM user_roles WHERE user_id = $1"

	rows, err := r.store.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roleIDs []string
	for rows.Next() {
		var roleID string
		if err := rows.Scan(&roleID); err != nil {
			return nil, err
		}
		roleIDs = append(roleIDs, roleID)
	}

	return roleIDs, nil
}

// assignRoleToUser assigns a role to a user
func (r *UserRepository) assignRoleToUser(ctx context.Context, userID, roleID string) error {
	query := "INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING"
	_, err := r.store.pool.Exec(ctx, query, userID, roleID)
	return err
}
