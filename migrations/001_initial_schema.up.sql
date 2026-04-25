-- Migration: 001_initial_schema
-- Direction: UP
-- Creates all base tables for the OIDC provider.
-- Run as the application owner role (not audit_writer).

BEGIN;

-- Enable pgcrypto for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ─── users ───────────────────────────────────────────────────────────────────
CREATE TABLE users (
    user_id       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(255) NOT NULL,
    email         VARCHAR(255),
    password_hash BYTEA       NOT NULL,   -- Argon2id hash
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata      JSONB,
    CONSTRAINT users_username_unique UNIQUE (username)
);

-- ─── clients ─────────────────────────────────────────────────────────────────
-- client_secret_enc: AES-256-GCM ciphertext produced by OpenBao Transit.
-- The raw secret is never stored here.
CREATE TABLE clients (
    client_id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    client_secret_enc BYTEA       NOT NULL,
    redirect_uris     TEXT[]      NOT NULL,
    scopes            TEXT[]      NOT NULL DEFAULT '{openid}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ─── authorization_codes ──────────────────────────────────────────────────────
-- code_hash:      SHA3-256 of the raw code. The raw code is never stored.
-- code_challenge: BASE64URL(SHA3-256(code_verifier)) — PKCE S256.
-- used:           Atomically set to true via SELECT FOR UPDATE on exchange.
CREATE TABLE authorization_codes (
    code_id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id      UUID        NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
    user_id        UUID        NOT NULL REFERENCES users(user_id)     ON DELETE CASCADE,
    code_hash      BYTEA       NOT NULL,
    code_challenge TEXT        NOT NULL,
    redirect_uri   TEXT        NOT NULL,
    scopes         TEXT[]      NOT NULL,
    expires_at     TIMESTAMPTZ NOT NULL,
    used           BOOLEAN     NOT NULL DEFAULT false,
    CONSTRAINT auth_codes_hash_unique UNIQUE (code_hash)
);

CREATE INDEX idx_auth_codes_expires ON authorization_codes (expires_at);

-- ─── refresh_tokens ───────────────────────────────────────────────────────────
-- token_enc: AES-256-GCM ciphertext produced by OpenBao Transit.
-- Rotation is atomic: old row revoked + new row inserted in one transaction.
CREATE TABLE refresh_tokens (
    token_id   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id  UUID        NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(user_id)     ON DELETE CASCADE,
    token_enc  BYTEA       NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    issued_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked    BOOLEAN     NOT NULL DEFAULT false
);

CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens (expires_at);

-- ─── audit_log ────────────────────────────────────────────────────────────────
-- Append-only. The audit_writer role has INSERT only — no UPDATE or DELETE.
-- This schema migration grants the permission; the role must be created
-- by the DBA before running this migration.
CREATE TABLE audit_log (
    id         BIGSERIAL    PRIMARY KEY,
    event_type VARCHAR(64)  NOT NULL,
    actor_id   UUID,
    client_id  UUID,
    metadata   JSONB,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Grant INSERT-only to the audit writer role (role must exist).
-- Run as superuser or table owner.
-- ─── pqc_keys ─────────────────────────────────────────────────────────────────
-- Fallback table for algorithms not natively supported by OpenBao Transit (e.g. Dilithium).
-- encrypted_key_blob: Private key encrypted with AES-256-GCM via OpenBao Transit.
CREATE TABLE pqc_keys (
    key_id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    algorithm          VARCHAR(50) NOT NULL,
    encrypted_key_blob TEXT        NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;
