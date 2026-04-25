package postgres

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/democryst/go-oidc/internal/model"
	"github.com/democryst/go-oidc/pkg/interfaces"
)

func setupTestDB(t *testing.T) (interfaces.Repository, *pgxpool.Pool, context.Context) {
	// Skip if Docker is not available in the environment
	if os.Getenv("DOCKER_HOST") == "" && os.Getenv("CI") != "" {
		t.Skip("Skipping integration test: Docker not found")
	}

	ctx := context.Background()

	// Locate migrations directory (tests run in their own folder, so we map upwards)
	wd, err := os.Getwd()
	require.NoError(t, err)
	migrationFile := filepath.Join(wd, "../../../migrations/001_initial_schema.up.sql")

	postgresContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithInitScripts(migrationFile),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		if strings.Contains(err.Error(), "failed to create Docker provider") || strings.Contains(err.Error(), "Docker not found") {
			t.Skipf("Skipping integration test: %v", err)
		}
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate postgres container: %s", err)
		}
	})

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	repo := NewPostgresRepository(pool)
	return repo, pool, ctx
}

func setupFixture(t *testing.T, pool *pgxpool.Pool, ctx context.Context) (uuid.UUID, uuid.UUID) {
	userID := uuid.New()
	clientID := uuid.New()

	// Insert mock user
	_, err := pool.Exec(ctx, `
		INSERT INTO users (user_id, username, email, password_hash) 
		VALUES ($1, 'testuser', 'test@example.com', 'fakehash')`, userID)
	require.NoError(t, err)

	// Insert mock client
	_, err = pool.Exec(ctx, `
		INSERT INTO clients (client_id, client_secret_enc, redirect_uris, scopes) 
		VALUES ($1, 'fakeenc', '{"http://localhost/cb"}', '{"openid"}')`, clientID)
	require.NoError(t, err)

	return userID, clientID
}

func TestPostgresRepository_AuthCode(t *testing.T) {
	repo, pool, ctx := setupTestDB(t)
	userID, clientID := setupFixture(t, pool, ctx)

	codeHash := []byte("sha3-256-hash123")
	code := &model.AuthCode{
		ClientID:      clientID,
		UserID:        userID,
		CodeHash:      codeHash,
		CodeChallenge: "challenge-123",
		RedirectURI:   "http://localhost/cb",
		Scopes:        []string{"openid"},
		ExpiresAt:     time.Now().Add(5 * time.Minute).Round(time.Microsecond),
	}

	// 1. Save
	err := repo.SaveAuthCode(ctx, code)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, code.ID)

	// 2. Get (Success)
	fetched, err := repo.GetAuthCode(ctx, codeHash)
	assert.NoError(t, err)
	assert.Equal(t, code.ID, fetched.ID)
	assert.False(t, fetched.Used)

	// 3. Mark Used (Success)
	err = repo.MarkAuthCodeUsed(ctx, code.ID)
	assert.NoError(t, err)

	// 4. Mark Used Again (Should Fail - already used)
	err = repo.MarkAuthCodeUsed(ctx, code.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already used or doesn't exist")

	// 5. Verify Used flag via Get
	fetchedAgain, err := repo.GetAuthCode(ctx, codeHash)
	assert.NoError(t, err)
	assert.True(t, fetchedAgain.Used)
}

func TestPostgresRepository_RefreshToken(t *testing.T) {
	repo, pool, ctx := setupTestDB(t)
	userID, clientID := setupFixture(t, pool, ctx)

	token := &model.RefreshToken{
		ClientID:  clientID,
		UserID:    userID,
		TokenEnc:  []byte("enc_token_1"),
		ExpiresAt: time.Now().Add(24 * time.Hour).Round(time.Microsecond),
	}

	// 1. Save
	err := repo.SaveRefreshToken(ctx, token)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, token.ID)

	// 2. Get
	fetched, err := repo.GetRefreshToken(ctx, token.ID)
	assert.NoError(t, err)
	assert.False(t, fetched.Revoked)
	assert.Equal(t, token.TokenEnc, fetched.TokenEnc)

	// 3. Rotate
	newToken := &model.RefreshToken{
		ClientID:  clientID,
		UserID:    userID,
		TokenEnc:  []byte("enc_token_2"),
		ExpiresAt: time.Now().Add(24 * time.Hour).Round(time.Microsecond),
	}
	err = repo.RotateRefreshToken(ctx, token.ID, newToken)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, newToken.ID)

	// 4. Verify old is revoked
	oldFetched, err := repo.GetRefreshToken(ctx, token.ID)
	assert.NoError(t, err)
	assert.True(t, oldFetched.Revoked)

	// 5. Verify new is active
	newFetched, err := repo.GetRefreshToken(ctx, newToken.ID)
	assert.NoError(t, err)
	assert.False(t, newFetched.Revoked)
	assert.Equal(t, []byte("enc_token_2"), newFetched.TokenEnc)

	// 6. Rotate an already revoked token (Should Fail)
	attemptNew := &model.RefreshToken{
		ClientID:  clientID,
		UserID:    userID,
		TokenEnc:  []byte("enc_token_3"),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err = repo.RotateRefreshToken(ctx, token.ID, attemptNew)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already revoked or doesn't exist")
}
