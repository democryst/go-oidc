package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/democryst/go-oidc/internal/model"
	"github.com/democryst/go-oidc/pkg/interfaces"
)

// PostgresRepository implements the Repository interface using pgx.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a newly instantiated PostgreSQL repository.
func NewPostgresRepository(pool *pgxpool.Pool) interfaces.Repository {
	return &PostgresRepository{
		pool: pool,
	}
}

// GetClient retrieves an OAuth2 client by its ID.
func (r *PostgresRepository) GetClient(ctx context.Context, clientID uuid.UUID) (*model.Client, error) {
	query := `SELECT client_id, client_secret_enc, redirect_uris, scopes, created_at FROM clients WHERE client_id = $1`
	client := &model.Client{}

	err := r.pool.QueryRow(ctx, query, clientID).Scan(
		&client.ID,
		&client.ClientSecretHash,
		&client.RedirectURIs,
		&client.Scopes,
		&client.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("client not found: %s", clientID)
		}
		return nil, fmt.Errorf("failed to query client: %w", err)
	}
	return client, nil
}

// SaveAuthCode persists a new authorization code into the database.
func (r *PostgresRepository) SaveAuthCode(ctx context.Context, code *model.AuthCode) error {
	query := `
		INSERT INTO authorization_codes 
		(client_id, user_id, code_hash, code_challenge, redirect_uri, scopes, expires_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) 
		RETURNING code_id`
	
	err := r.pool.QueryRow(ctx, query,
		code.ClientID,
		code.UserID,
		code.CodeHash,
		code.CodeChallenge,
		code.RedirectURI,
		code.Scopes,
		code.ExpiresAt,
	).Scan(&code.ID)
	
	if err != nil {
		return fmt.Errorf("failed to save auth code: %w", err)
	}
	return nil
}

// GetAuthCode retrieves a code by its SHA-3 hash.
func (r *PostgresRepository) GetAuthCode(ctx context.Context, codeHash []byte) (*model.AuthCode, error) {
	query := `
		SELECT code_id, client_id, user_id, code_hash, code_challenge, redirect_uri, scopes, expires_at, used 
		FROM authorization_codes 
		WHERE code_hash = $1`
	code := &model.AuthCode{}

	err := r.pool.QueryRow(ctx, query, codeHash).Scan(
		&code.ID,
		&code.ClientID,
		&code.UserID,
		&code.CodeHash,
		&code.CodeChallenge,
		&code.RedirectURI,
		&code.Scopes,
		&code.ExpiresAt,
		&code.Used,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("auth code not found")
		}
		return nil, fmt.Errorf("failed to query auth code: %w", err)
	}
	return code, nil
}

// MarkAuthCodeUsed atomically marks an auth code as used.
func (r *PostgresRepository) MarkAuthCodeUsed(ctx context.Context, codeID uuid.UUID) error {
	query := `UPDATE authorization_codes SET used = true WHERE code_id = $1 AND used = false`
	tag, err := r.pool.Exec(ctx, query, codeID)
	if err != nil {
		return fmt.Errorf("failed to mark auth code %s as used: %w", codeID, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("auth code %s was already used or doesn't exist", codeID)
	}
	return nil
}

// SaveRefreshToken persists a new refresh token.
func (r *PostgresRepository) SaveRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (client_id, user_id, token_enc, expires_at) 
		VALUES ($1, $2, $3, $4) 
		RETURNING token_id, issued_at, revoked`
	
	err := r.pool.QueryRow(ctx, query,
		token.ClientID,
		token.UserID,
		token.TokenEnc,
		token.ExpiresAt,
	).Scan(&token.ID, &token.IssuedAt, &token.Revoked)
	
	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}
	return nil
}

// GetRefreshToken retrieves an active token by ID.
func (r *PostgresRepository) GetRefreshToken(ctx context.Context, tokenID uuid.UUID) (*model.RefreshToken, error) {
	query := `
		SELECT token_id, client_id, user_id, token_enc, expires_at, issued_at, revoked 
		FROM refresh_tokens 
		WHERE token_id = $1`
	token := &model.RefreshToken{}

	err := r.pool.QueryRow(ctx, query, tokenID).Scan(
		&token.ID,
		&token.ClientID,
		&token.UserID,
		&token.TokenEnc,
		&token.ExpiresAt,
		&token.IssuedAt,
		&token.Revoked,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("refresh token not found: %s", tokenID)
		}
		return nil, fmt.Errorf("failed to query refresh token: %w", err)
	}
	return token, nil
}

// RotateRefreshToken atomically revokes the old token and inserts the new token.
func (r *PostgresRepository) RotateRefreshToken(ctx context.Context, oldID uuid.UUID, newToken *model.RefreshToken) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Revoke the old token
	revokeQuery := `UPDATE refresh_tokens SET revoked = true WHERE token_id = $1 AND revoked = false`
	tag, err := tx.Exec(ctx, revokeQuery, oldID)
	if err != nil {
		return fmt.Errorf("failed to revoke old refresh token %s: %w", oldID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("old refresh token %s was already revoked or doesn't exist", oldID)
	}

	// 2. Insert the new token
	insertQuery := `
		INSERT INTO refresh_tokens (client_id, user_id, token_enc, expires_at) 
		VALUES ($1, $2, $3, $4) 
		RETURNING token_id, issued_at, revoked`
	
	err = tx.QueryRow(ctx, insertQuery,
		newToken.ClientID,
		newToken.UserID,
		newToken.TokenEnc,
		newToken.ExpiresAt,
	).Scan(&newToken.ID, &newToken.IssuedAt, &newToken.Revoked)
	
	if err != nil {
		return fmt.Errorf("failed to insert new refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit token rotation tx: %w", err)
	}
	return nil
}

// GetUser retrieves a user by ID.
func (r *PostgresRepository) GetUser(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	query := `SELECT user_id, username, email, password_hash, created_at, metadata FROM users WHERE user_id = $1`
	user := &model.User{}

	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.Metadata,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %s", userID)
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	return user, nil
}

// GetUserByUsername retrieves a user by Username.
func (r *PostgresRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `SELECT user_id, username, email, password_hash, created_at, metadata FROM users WHERE username = $1`
	user := &model.User{}

	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.Metadata,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	return user, nil
}

// SavePQCKey persists an encrypted PQC private key.
func (r *PostgresRepository) SavePQCKey(ctx context.Context, algorithm string, encryptedHex string) (uuid.UUID, error) {
	var keyID uuid.UUID
	query := `INSERT INTO pqc_keys (algorithm, encrypted_key_blob) VALUES ($1, $2) RETURNING key_id`
	err := r.pool.QueryRow(ctx, query, algorithm, encryptedHex).Scan(&keyID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to save pqc key: %w", err)
	}
	return keyID, nil
}

// GetPQCKey retrieves an encrypted PQC private key.
func (r *PostgresRepository) GetPQCKey(ctx context.Context, keyID uuid.UUID) (string, string, error) {
	var algo, hex string
	query := `SELECT algorithm, encrypted_key_blob FROM pqc_keys WHERE key_id = $1`
	err := r.pool.QueryRow(ctx, query, keyID).Scan(&algo, &hex)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", fmt.Errorf("pqc key not found: %s", keyID)
		}
		return "", "", fmt.Errorf("failed to query pqc key: %w", err)
	}
	return algo, hex, nil
}

// AppendAuditLog securely appends an audit event to the append-only table.
func (r *PostgresRepository) AppendAuditLog(ctx context.Context, event *model.AuditEvent) error {
	query := `INSERT INTO audit_log (request_id, event_type, actor_id, client_id, metadata) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.pool.Exec(ctx, query,
		event.RequestID,
		event.EventType,
		event.ActorID,
		event.ClientID,
		event.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to append audit log: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListClients(ctx context.Context) ([]model.Client, error) {
	query := `SELECT client_id, name, redirect_uris, created_at FROM clients`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []model.Client
	for rows.Next() {
		var c model.Client
		if err := rows.Scan(&c.ID, &c.Name, &c.RedirectURIs, &c.CreatedAt); err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, nil
}

func (r *PostgresRepository) SaveClient(ctx context.Context, client *model.Client) error {
	query := `INSERT INTO clients (client_id, name, redirect_uris) VALUES ($1, $2, $3)`
	_, err := r.pool.Exec(ctx, query, client.ID, client.Name, client.RedirectURIs)
	return err
}

func (r *PostgresRepository) GetAuditLogs(ctx context.Context, limit int) ([]model.AuditEvent, error) {
	query := `SELECT log_id, request_id, event_type, actor_id, client_id, metadata, created_at FROM audit_log ORDER BY created_at DESC LIMIT $1`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.AuditEvent
	for rows.Next() {
		var e model.AuditEvent
		if err := rows.Scan(&e.ID, &e.RequestID, &e.EventType, &e.ActorID, &e.ClientID, &e.Metadata, &e.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, e)
	}
	return logs, nil
}
