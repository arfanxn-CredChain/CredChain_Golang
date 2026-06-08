-- Drop credentials first (FKs reference users)
DROP INDEX IF EXISTS idx_credentials_file_hash;
DROP INDEX IF EXISTS idx_credentials_extract_status;
DROP INDEX IF EXISTS idx_credentials_revoked_at;
DROP INDEX IF EXISTS idx_credentials_issuer_user_id;
DROP INDEX IF EXISTS idx_credentials_holder_user_id;
ALTER TABLE credentials DROP CONSTRAINT IF EXISTS fk_holder_user_id;
ALTER TABLE credentials DROP CONSTRAINT IF EXISTS fk_issuer_user_id;
ALTER TABLE credentials DROP CONSTRAINT IF EXISTS fk_revoker_user_id;
DROP TABLE IF EXISTS credentials;
DROP TYPE IF EXISTS credential_extract_status;

-- Drop user_tokens before users (FK fk_user_tokens_user_id references users)
DROP INDEX IF EXISTS idx_user_tokens_token;
DROP INDEX IF EXISTS idx_user_tokens_user_id;
DROP TABLE IF EXISTS user_tokens;
DROP TYPE IF EXISTS user_token_type;

-- Drop users last
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS role;
DROP TYPE IF EXISTS gender;
