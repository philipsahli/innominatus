-- Migration: Create user_api_keys table for OIDC users
-- This table stores API keys for users authenticated via OIDC
-- File-based users (users.yaml) continue using the existing system

CREATE TABLE IF NOT EXISTS user_api_keys (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    key_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_username_keyname UNIQUE(username, key_name)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_user_api_keys_username ON user_api_keys(username);
CREATE INDEX IF NOT EXISTS idx_user_api_keys_hash ON user_api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_user_api_keys_expires ON user_api_keys(expires_at);

-- Comments for documentation
COMMENT ON TABLE user_api_keys IS 'API keys for OIDC-authenticated users';
COMMENT ON COLUMN user_api_keys.key_hash IS 'SHA-256 hash of the API key (never store plain keys)';
COMMENT ON COLUMN user_api_keys.username IS 'Username from OIDC claims (preferred_username or email)';
COMMENT ON COLUMN user_api_keys.key_name IS 'User-provided name for the API key';
