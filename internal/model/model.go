// Package model defines the shared domain types for the go-oidc server.
package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated end-user.
type User struct {
	ID           uuid.UUID
	Username     string
	Email        string
	PasswordHash []byte
	CreatedAt    time.Time
	Metadata     map[string]any
}

// Client represents a registered OAuth2 client application.
type Client struct {
	ID               uuid.UUID
	ClientSecretHash []byte // Argon2id hash; raw secret never stored
	RedirectURIs     []string
	Scopes           []string
	CreatedAt        time.Time
}

type TokenClaims struct {
	Subject   string         `json:"sub"`
	Issuer    string         `json:"iss"`
	Audience  []string       `json:"aud"`
	ExpiresAt int64          `json:"exp"`
	IssuedAt  int64          `json:"iat"`
	ClientID  string         `json:"client_id"`
	Scope     string         `json:"scope"`
	Nonce     string         `json:"nonce"`
	RequestID string         `json:"request_id,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
}

// AuthCode represents a single-use, short-lived authorization code.
type AuthCode struct {
	ID            uuid.UUID
	ClientID      uuid.UUID
	UserID        uuid.UUID
	CodeHash      []byte // SHA-3 hash of the raw code
	CodeChallenge string // S256 PKCE challenge (SHA3-256 of verifier)
	RedirectURI   string
	Scopes        []string
	ExpiresAt     time.Time
	Used          bool
}

// RefreshToken represents a rotatable, long-lived refresh token.
type RefreshToken struct {
	ID        uuid.UUID
	ClientID  uuid.UUID
	UserID    uuid.UUID
	TokenEnc  []byte // AES-256-GCM encrypted via OpenBao
	ExpiresAt time.Time
	IssuedAt  time.Time
	Revoked   bool
}

// AuditEvent is an append-only record of a security-relevant action.
type AuditEvent struct {
	ID        uuid.UUID      `json:"id"`
	RequestID string         `json:"request_id"`
	EventType string         `json:"event_type"`
	ActorID   uuid.UUID      `json:"actor_id"`
	ClientID  uuid.UUID      `json:"client_id"`
	Metadata  map[string]any `json:"metadata"`
	Timestamp time.Time      `json:"timestamp"`
}
