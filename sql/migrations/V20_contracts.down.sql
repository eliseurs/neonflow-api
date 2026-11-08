DROP TABLE IF EXISTS contract_comments CASCADE;
DROP TABLE IF EXISTS contract_events CASCADE;
DROP TABLE IF EXISTS contract_versions CASCADE;
DROP TABLE IF EXISTS contract_documents CASCADE;
DROP FUNCTION IF EXISTS contract_append_only();
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'central'));
