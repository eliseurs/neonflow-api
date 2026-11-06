-- Usuários desativados não entram no painel (gestão de usuários pelo admin, RN-05.2 e RN-05.6).
ALTER TABLE users ADD COLUMN active BOOLEAN NOT NULL DEFAULT true;
