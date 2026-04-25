package oidc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/democryst/go-oidc/internal/config"
	"github.com/democryst/go-oidc/internal/model"
	"github.com/democryst/go-oidc/pkg/interfaces"
)

// --- Mocks ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetClient(ctx context.Context, id uuid.UUID) (*model.Client, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Client), args.Error(1)
}
func (m *MockRepository) SaveAuthCode(ctx context.Context, code *model.AuthCode) error {
	return m.Called(ctx, code).Error(0)
}
func (m *MockRepository) GetAuthCode(ctx context.Context, hash []byte) (*model.AuthCode, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AuthCode), args.Error(1)
}
func (m *MockRepository) MarkAuthCodeUsed(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockRepository) SaveRefreshToken(ctx context.Context, t *model.RefreshToken) error {
	return m.Called(ctx, t).Error(0)
}
func (m *MockRepository) RotateRefreshToken(ctx context.Context, id uuid.UUID, n *model.RefreshToken) error {
	return m.Called(ctx, id, n).Error(0)
}
func (m *MockRepository) GetRefreshToken(ctx context.Context, id uuid.UUID) (*model.RefreshToken, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RefreshToken), args.Error(1)
}
func (m *MockRepository) GetUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.User), args.Error(1)
}
func (m *MockRepository) GetUserByUsername(ctx context.Context, u string) (*model.User, error) {
	args := m.Called(ctx, u)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}
func (m *MockRepository) SavePQCKey(ctx context.Context, algorithm string, encryptedHex string) (uuid.UUID, error) {
	args := m.Called(ctx, algorithm, encryptedHex)
	return args.Get(0).(uuid.UUID), args.Error(1)
}
func (m *MockRepository) GetPQCKey(ctx context.Context, keyID uuid.UUID) (string, string, error) {
	args := m.Called(ctx, keyID)
	return args.String(0), args.String(1), args.Error(2)
}
func (m *MockRepository) AppendAuditLog(ctx context.Context, e *model.AuditEvent) error {
	return m.Called(ctx, e).Error(0)
}

type MockSigner struct {
	mock.Mock
}

func (m *MockSigner) Sign(ctx context.Context, claims interfaces.TokenClaims) (string, error) {
	args := m.Called(ctx, claims)
	return args.String(0), args.Error(1)
}
func (m *MockSigner) PublicKeys() []interfaces.JSONWebKey {
	return m.Called().Get(0).([]interfaces.JSONWebKey)
}

type MockHasher struct {
	mock.Mock
}

func (m *MockHasher) PKCEChallenge(v string) string {
	return m.Called(v).String(0)
}
func (m *MockHasher) VerifyPKCE(v, c string) bool {
	return m.Called(v, c).Bool(0)
}
func (m *MockHasher) HashCode(r string) []byte {
	return m.Called(r).Get(0).([]byte)
}
func (m *MockHasher) SecureNonce() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// --- Tests ---

func TestOIDCService_Authorize(t *testing.T) {
	ctx := context.Background()
	clientID := uuid.New()
	req := interfaces.AuthorizeRequest{
		ClientID:            clientID.String(),
		RedirectURI:         "http://localhost/cb",
		ResponseType:        "code",
		Scope:               "openid",
		State:               "xyz",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	}

	t.Run("Success", func(t *testing.T) {
		repo := new(MockRepository)
		signer := new(MockSigner)
		hasher := new(MockHasher)
		cfg := &config.Config{OIDC: config.OIDCConfig{AuthCodeTTL: 5 * time.Minute}}
		svc := NewOIDCService(repo, signer, hasher, cfg)

		repo.On("GetClient", ctx, clientID).Return(&model.Client{
			ID:           clientID,
			RedirectURIs: []string{"http://localhost/cb"},
		}, nil).Once()
		hasher.On("SecureNonce").Return("raw-code", nil).Once()
		hasher.On("HashCode", "raw-code").Return([]byte("hashed")).Once()
		repo.On("SaveAuthCode", ctx, mock.AnythingOfType("*model.AuthCode")).Return(nil).Once()

		resp, err := svc.Authorize(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, "raw-code", resp.Code)
		assert.Equal(t, "xyz", resp.State)
		repo.AssertExpectations(t)
	})

	t.Run("MissingPKCE", func(t *testing.T) {
		repo := new(MockRepository)
		signer := new(MockSigner)
		hasher := new(MockHasher)
		cfg := &config.Config{OIDC: config.OIDCConfig{AuthCodeTTL: 5 * time.Minute}}
		svc := NewOIDCService(repo, signer, hasher, cfg)

		repo.On("GetClient", ctx, clientID).Return(&model.Client{
			ID:           clientID,
			RedirectURIs: []string{"http://localhost/cb"},
		}, nil).Once()

		badReq := req
		badReq.CodeChallenge = ""
		_, err := svc.Authorize(ctx, badReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PKCE S256 is mandatory")
		repo.AssertExpectations(t)
	})
}

func TestOIDCService_Token_AuthCode(t *testing.T) {
	repo := new(MockRepository)
	signer := new(MockSigner)
	hasher := new(MockHasher)
	cfg := &config.Config{OIDC: config.OIDCConfig{
		Issuer: "http://test",
		AccessTokenTTL: 1 * time.Hour,
		IDTokenTTL: 1 * time.Hour,
	}}
	svc := NewOIDCService(repo, signer, hasher, cfg)

	ctx := context.Background()
	clientID := uuid.New()
	userID := uuid.New()
	codeStr := "raw-code"
	codeHash := []byte("hashed")
	req := interfaces.TokenRequest{
		GrantType:    "authorization_code",
		ClientID:     clientID.String(),
		Code:         codeStr,
		RedirectURI:  "http://localhost/cb",
		CodeVerifier: "verifier",
	}

	t.Run("Success", func(t *testing.T) {
		hasher.On("HashCode", codeStr).Return(codeHash).Once()
		repo.On("GetAuthCode", ctx, codeHash).Return(&model.AuthCode{
			ID:            uuid.New(),
			ClientID:      clientID,
			UserID:        userID,
			CodeChallenge: "challenge",
			RedirectURI:   "http://localhost/cb",
			Scopes:        []string{"openid"},
			ExpiresAt:     time.Now().Add(time.Hour),
		}, nil).Once()
		hasher.On("VerifyPKCE", "verifier", "challenge").Return(true).Once()
		repo.On("MarkAuthCodeUsed", ctx, mock.Anything).Return(nil).Once()
		signer.On("Sign", ctx, mock.Anything).Return("signed-at", nil).Twice()

		resp, err := svc.Token(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, "signed-at", resp.AccessToken)
		assert.Equal(t, "signed-at", resp.IDToken)
	})
}
