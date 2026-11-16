DROP TABLE IF EXISTS account_tokens CASCADE;
-- Falha se a conta de um parceiro ou cliente já estiver na auditoria (audit_log é só de inclusão).
DELETE FROM users WHERE role IN ('parceiro', 'cliente');
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_account_link_check;
DROP INDEX IF EXISTS users_customer_key;
DROP INDEX IF EXISTS users_partner_key;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS customer_id;
ALTER TABLE users DROP COLUMN IF EXISTS partner_id;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'central', 'legal_reviewer'));
