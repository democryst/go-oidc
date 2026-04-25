// Package interfaces defines the contracts that decouple core logic from
// concrete implementations (database, crypto, KMS). All dependencies flow
// inward through these interfaces.
package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/democryst/go-oidc/internal/model"
)

// Repository is the persistence contract for all OIDC data.
// Implementations must be safe for concurrent use.
type Repository interface {
	// --- Client ---
	GetClient(ctx context.Context, clientID uuid.UUID) (*model.Client, error)

	// --- Authorization codes ---

	// SaveAuthCode persists a new, unused authorization code.
	SaveAuthCode(ctx context.Context, code *model.AuthCode) error
	// GetAuthCode retrieves a code by its SHA-3 hash.
	// Returns an error if the code is not found, expired, or already used.
	GetAuthCode(ctx context.Context, codeHash []byte) (*model.AuthCode, error)
	// MarkAuthCodeUsed atomically marks a code as used within a transaction
	// to prevent replay attacks. Uses SELECT FOR UPDATE.
	MarkAuthCodeUsed(ctx context.Context, codeID uuid.UUID) error

	// --- Refresh tokens ---

	// SaveRefreshToken persists a new refresh token (AES-256-GCM encrypted).
	SaveRefreshToken(ctx context.Context, token *model.RefreshToken) error
	// RotateRefreshToken atomically revokes the old token and persists the
	// new one in a single database transaction.
	RotateRefreshToken(ctx context.Context, oldID uuid.UUID, newToken *model.RefreshToken) error
	// GetRefreshToken retrieves an active (non-revoked, non-expired) token by ID.
	GetRefreshToken(ctx context.Context, tokenID uuid.UUID) (*model.RefreshToken, error)

	// --- Users ---

	GetUser(ctx context.Context, userID uuid.UUID) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)

	// --- PQC Keys (Fallback) ---
	SavePQCKey(ctx context.Context, algorithm string, encryptedHex string) (uuid.UUID, error)
	GetPQCKey(ctx context.Context, keyID uuid.UUID) (algorithm string, encryptedHex string, err error)

	// --- Audit ---

	// AppendAuditLog writes an immutable audit record.
	// The DB role must have INSERT-only permission on the audit_log table.
	AppendAuditLog(ctx context.Context, event *model.AuditEvent) error

	// --- Admin ---
	ListClients(ctx context.Context) ([]model.Client, error)
	SaveClient(ctx context.Context, client *model.Client) error
	GetAuditLogs(ctx context.Context, limit int) ([]model.AuditEvent, error)
}
