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
	EventType string
	ActorID   *uuid.UUID
	ClientID  *uuid.UUID
	Metadata  map[string]any
}
