CREATE TYPE role AS ENUM (
    'super_admin',
    'admin',
    'issuer',
    'holder'
);

CREATE TABLE users (
    id CHAR(26) PRIMARY KEY,
    name VARCHAR(256),
    number VARCHAR(256) UNIQUE,
    phone_number VARCHAR(18) UNIQUE,
    email VARCHAR(256) UNIQUE NOT NULL,
    birth_date DATE,
    meta JSONB,
    role role NOT NULL,
    wallet_address CHAR(42) UNIQUE NOT NULL,
    wallet_private_key VARCHAR(256) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE credentials (
    id CHAR(26) PRIMARY KEY,
    holder_user_id CHAR(26) NOT NULL,
    issuer_user_id CHAR(26) NOT NULL,
    revoker_user_id CHAR(26),
    name VARCHAR(256) NOT NULL,
    meta JSONB,
    token_id VARCHAR(256) UNIQUE,
    file_hash CHAR(66) NOT NULL,
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_holder_user_id FOREIGN KEY (holder_user_id) REFERENCES users(id),
    CONSTRAINT fk_issuer_user_id FOREIGN KEY (issuer_user_id) REFERENCES users(id),
    CONSTRAINT fk_revoker_user_id FOREIGN KEY (revoker_user_id) REFERENCES users(id)
);

-- Token types for user refresh tokens
CREATE TYPE user_token_type AS ENUM ('refresh');

-- User tokens table for managing refresh tokens
CREATE TABLE user_tokens (
    id CHAR(26) PRIMARY KEY,
    user_id CHAR(26) NOT NULL,
    type user_token_type NOT NULL,
    token VARCHAR(512) NOT NULL UNIQUE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user_tokens_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Index for fast user-based lookups
CREATE INDEX idx_user_tokens_user_id ON user_tokens(user_id);

-- Index for token lookups (refresh token validation)
CREATE INDEX idx_user_tokens_token ON user_tokens(token);
