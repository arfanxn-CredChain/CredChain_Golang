ALTER TABLE credentials DROP CONSTRAINT IF EXISTS fk_holder_user_id;
ALTER TABLE credentials DROP CONSTRAINT IF EXISTS fk_issuer_user_id;
ALTER TABLE credentials DROP CONSTRAINT IF EXISTS fk_revoker_user_id;

DROP TABLE IF EXISTS credentials;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS role;

-- Drop user_tokens table and indexes
DROP INDEX IF EXISTS idx_user_tokens_token;
DROP INDEX IF EXISTS idx_user_tokens_user_id;
DROP TABLE IF EXISTS user_tokens;
DROP TYPE IF EXISTS user_token_type;
