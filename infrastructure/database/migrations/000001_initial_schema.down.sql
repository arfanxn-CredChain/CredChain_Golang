ALTER TABLE credentials DROP CONSTRAINT IF EXISTS fk_holder_user_id;
ALTER TABLE credentials DROP CONSTRAINT IF EXISTS fk_issuer_user_id;
ALTER TABLE credentials DROP CONSTRAINT IF EXISTS fk_revoker_user_id;

DROP TABLE IF EXISTS credentials;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS role;
