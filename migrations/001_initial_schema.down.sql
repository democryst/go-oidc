-- Migration: 001_initial_schema
-- Direction: DOWN
-- Drops all tables created by 001_initial_schema.up.sql

BEGIN;

DROP TABLE IF EXISTS audit_log          CASCADE;
DROP TABLE IF EXISTS refresh_tokens     CASCADE;
DROP TABLE IF EXISTS authorization_codes CASCADE;
DROP TABLE IF EXISTS clients            CASCADE;
DROP TABLE IF EXISTS users              CASCADE;

COMMIT;
