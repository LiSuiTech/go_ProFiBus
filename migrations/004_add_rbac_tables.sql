-- Migration: Add RBAC (Role-Based Access Control) tables
-- Description: Creates tables for users, roles, permissions, and sessions

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE
);

-- Roles table
CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    permissions TEXT[], -- Array of permission strings (resource:action)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User-Role mapping table (many-to-many)
CREATE TABLE IF NOT EXISTS user_roles (
    user_id VARCHAR(255) REFERENCES users(id) ON DELETE CASCADE,
    role_id VARCHAR(255) REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);

-- Sessions table for authentication tokens
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(512) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(50),
    user_agent TEXT
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_enabled ON users(enabled);
CREATE INDEX IF NOT EXISTS idx_users_last_login ON users(last_login_at DESC);

CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);

CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Create trigger function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_rbac_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for users and roles tables
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_rbac_updated_at_column();

CREATE TRIGGER update_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW
    EXECUTE FUNCTION update_rbac_updated_at_column();

-- Insert default roles with predefined permissions

-- Admin role - full access to everything
INSERT INTO roles (id, name, description, permissions) VALUES
(
    'role-admin',
    'Admin',
    'System administrator with full access to all resources',
    ARRAY[
        'rule:create', 'rule:read', 'rule:update', 'rule:delete',
        'analyzer:create', 'analyzer:read', 'analyzer:update', 'analyzer:delete',
        'processor:create', 'processor:read', 'processor:update', 'processor:delete',
        'pipeline:create', 'pipeline:read', 'pipeline:update', 'pipeline:delete', 'pipeline:execute',
        'user:create', 'user:read', 'user:update', 'user:delete',
        'role:create', 'role:read', 'role:update', 'role:delete',
        'config:create', 'config:read', 'config:update', 'config:delete', 'config:export', 'config:import',
        'metrics:read',
        'trace:read'
    ]
),
(
    'role-operator',
    'Operator',
    'Operator with ability to manage pipelines and configurations',
    ARRAY[
        'rule:create', 'rule:read', 'rule:update',
        'analyzer:create', 'analyzer:read', 'analyzer:update',
        'processor:create', 'processor:read', 'processor:update',
        'pipeline:read', 'pipeline:execute',
        'config:read', 'config:update',
        'metrics:read',
        'trace:read'
    ]
),
(
    'role-editor',
    'Editor',
    'Editor with ability to create and modify configurations',
    ARRAY[
        'rule:create', 'rule:read', 'rule:update',
        'analyzer:create', 'analyzer:read', 'analyzer:update',
        'processor:create', 'processor:read', 'processor:update',
        'pipeline:read',
        'config:read', 'config:update',
        'metrics:read',
        'trace:read'
    ]
),
(
    'role-viewer',
    'Viewer',
    'Read-only access to view all resources',
    ARRAY[
        'rule:read',
        'analyzer:read',
        'processor:read',
        'pipeline:read',
        'config:read',
        'metrics:read',
        'trace:read'
    ]
);

-- Insert default admin user
-- Default username: admin
-- Default password: admin123 (hashed using bcrypt)
-- Password hash for 'admin123': $2a$10$Z6r1y7eDJZ9XQN.Q9H.7GOF3iWXB5x7tXY/Eh/k5WvBJG/8p0RJ5K
INSERT INTO users (id, username, email, password_hash, full_name, enabled) VALUES
(
    'user-admin',
    'admin',
    'admin@profibus.local',
    '$2a$10$Z6r1y7eDJZ9XQN.Q9H.7GOF3iWXB5x7tXY/Eh/k5WvBJG/8p0RJ5K',
    'System Administrator',
    true
);

-- Assign admin role to default admin user
INSERT INTO user_roles (user_id, role_id) VALUES
('user-admin', 'role-admin');

-- Add comments to tables
COMMENT ON TABLE users IS 'System users with authentication credentials';
COMMENT ON TABLE roles IS 'Roles with associated permissions';
COMMENT ON TABLE user_roles IS 'Many-to-many mapping between users and roles';
COMMENT ON TABLE sessions IS 'Active user sessions with authentication tokens';

COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password';
COMMENT ON COLUMN roles.permissions IS 'Array of permission strings in format resource:action';
COMMENT ON COLUMN sessions.token IS 'JWT or random token for session authentication';
COMMENT ON COLUMN sessions.expires_at IS 'Session expiration timestamp';

-- Function to clean up expired sessions
CREATE OR REPLACE FUNCTION clean_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP;
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION clean_expired_sessions() IS 'Removes all expired sessions and returns the count of deleted sessions';
