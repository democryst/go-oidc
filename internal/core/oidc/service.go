package oidc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/democryst/go-oidc/internal/config"
	"github.com/democryst/go-oidc/internal/model"
	"github.com/democryst/go-oidc/pkg/interfaces"
)

type OIDCService struct {
	repo   interfaces.Repository
	signer interfaces.Signer
	hasher interfaces.Hasher
	cfg    *config.Config
}

func NewOIDCService(repo interfaces.Repository, signer interfaces.Signer, hasher interfaces.Hasher, cfg *config.Config) interfaces.OIDCService {
	return &OIDCService{
		repo:   repo,
		signer: signer,
		hasher: hasher,
		cfg:    cfg,
	}
}

func (s *OIDCService) Authorize(ctx context.Context, req interfaces.AuthorizeRequest) (*interfaces.AuthorizeResponse, error) {
	// 1. Validate Client
	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("invalid client_id: %w", err)
	}

	client, err := s.repo.GetClient(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("client check failed: %w", err)
	}

	// 2. Validate Redirect URI
	validRedirect := false
	for _, uri := range client.RedirectURIs {
		if uri == req.RedirectURI {
			validRedirect = true
			break
		}
	}
	if !validRedirect {
		return nil, fmt.Errorf("invalid redirect_uri: %s", req.RedirectURI)
	}

	// 3. Ensure PKCE S256
	if req.CodeChallenge == "" || req.CodeChallengeMethod != "S256" {
		return nil, fmt.Errorf("PKCE S256 is mandatory")
	}

	// 4. Generate Auth Code
	rawCode, err := s.hasher.SecureNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	codeHash := s.hasher.HashCode(rawCode)

	// 5. Store Code (Assume userID is found in context or session - simplified here)
	// In a real flow, a session check happens before Authorize.
	// For this exercise, we'll use a dummy userID if not present.
	userID := uuid.Nil // In production, this must be a real authenticated user ID
	s.repo.AppendAuditLog(ctx, &model.AuditEvent{
		RequestID: s.getRequestID(ctx),
		EventType: "AUTHORIZE_INIT",
		ClientID:  client.ID,
		Metadata: map[string]any{
			"scope":         req.Scope,
			"response_type": req.ResponseType,
		},
	})

	authCode := &model.AuthCode{
		ClientID:      clientID,
		UserID:        userID,
		CodeHash:      codeHash,
		CodeChallenge: req.CodeChallenge,
		RedirectURI:   req.RedirectURI,
		Scopes:        strings.Split(req.Scope, " "),
		ExpiresAt:     time.Now().Add(s.cfg.OIDC.AuthCodeTTL),
	}

	if err := s.repo.SaveAuthCode(ctx, authCode); err != nil {
		return nil, fmt.Errorf("failed to persist auth code: %w", err)
	}

	return &interfaces.AuthorizeResponse{
		Code:  rawCode,
		State: req.State,
	}, nil
}

func (s *OIDCService) Token(ctx context.Context, req interfaces.TokenRequest) (*interfaces.TokenResponse, error) {
	switch req.GrantType {
	case "authorization_code":
		return s.handleAuthCode(ctx, req)
	case "refresh_token":
		return s.handleRefreshToken(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported grant_type: %s", req.GrantType)
	}
}

func (s *OIDCService) handleAuthCode(ctx context.Context, req interfaces.TokenRequest) (*interfaces.TokenResponse, error) {
	codeHash := s.hasher.HashCode(req.Code)
	authCode, err := s.repo.GetAuthCode(ctx, codeHash)
	if err != nil {
		return nil, fmt.Errorf("code invalid: %w", err)
	}

	// 1. Validation
	if authCode.Used {
		return nil, fmt.Errorf("code already used")
	}
	if authCode.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("code expired")
	}
	if authCode.ClientID.String() != req.ClientID {
		return nil, fmt.Errorf("client mismatch")
	}
	if authCode.RedirectURI != req.RedirectURI {
		return nil, fmt.Errorf("redirect_uri mismatch")
	}

	// 2. PKCE Verify
	if !s.hasher.VerifyPKCE(req.CodeVerifier, authCode.CodeChallenge) {
		return nil, fmt.Errorf("PKCE verification failed")
	}

	// 3. Mark Used
	if err := s.repo.MarkAuthCodeUsed(ctx, authCode.ID); err != nil {
		return nil, fmt.Errorf("failed to mark code as used: %w", err)
	}

	return s.issueTokens(ctx, authCode.ClientID, authCode.UserID, strings.Join(authCode.Scopes, " "))
}

func (s *OIDCService) handleRefreshToken(ctx context.Context, req interfaces.TokenRequest) (*interfaces.TokenResponse, error) {
	// 1. Parse token ID (simplified: assuming token ID is passed as refreshToken)
	tokenID, err := uuid.Parse(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token format")
	}

	oldToken, err := s.repo.GetRefreshToken(ctx, tokenID)
	if err != nil {
		return nil, fmt.Errorf("token invalid: %w", err)
	}

	if oldToken.Revoked || oldToken.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token revoked or expired")
	}

	if oldToken.ClientID.String() != req.ClientID {
		return nil, fmt.Errorf("client mismatch")
	}

	// 2. Rotate
	s.repo.AppendAuditLog(ctx, &model.AuditEvent{
		RequestID: s.getRequestID(ctx),
		EventType: "TOKEN_REFRESHED",
		ActorID:   oldToken.UserID,
		ClientID:  oldToken.ClientID,
	})
	newToken := &model.RefreshToken{
		ClientID:  oldToken.ClientID,
		UserID:    oldToken.UserID,
		TokenEnc:  oldToken.TokenEnc, // Re-use encrypted material or re-encrypt
		ExpiresAt: time.Now().Add(s.cfg.OIDC.RefreshTokenTTL),
	}

	if err := s.repo.RotateRefreshToken(ctx, oldToken.ID, newToken); err != nil {
		return nil, fmt.Errorf("rotation failed: %w", err)
	}

	// 3. Find User Scopes (simplified: use defaults)
	return s.issueTokens(ctx, newToken.ClientID, newToken.UserID, "openid profile")
}

func (s *OIDCService) issueTokens(ctx context.Context, clientID, userID uuid.UUID, scope string) (*interfaces.TokenResponse, error) {
	now := time.Now().Unix()

	// 1. Access Token
	atClaims := interfaces.TokenClaims{
		Subject:   userID.String(),
		Issuer:    s.cfg.OIDC.Issuer,
		Audience:  []string{clientID.String()},
		ExpiresAt: time.Now().Add(s.cfg.OIDC.AccessTokenTTL).Unix(),
		IssuedAt:  now,
		Scope:     scope,
		ClientID:  clientID.String(),
	}
	accessToken, err := s.signer.Sign(ctx, atClaims)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// 2. ID Token
	idClaims := interfaces.TokenClaims{
		Subject:   userID.String(),
		Issuer:    s.cfg.OIDC.Issuer,
		Audience:  []string{clientID.String()},
		ExpiresAt: time.Now().Add(s.cfg.OIDC.IDTokenTTL).Unix(),
		IssuedAt:  now,
		Nonce:     "random-nonce", // Should be passed from AuthCode if available
		ClientID:  clientID.String(),
	}
	idToken, err := s.signer.Sign(ctx, idClaims)
	if err != nil {
		return nil, fmt.Errorf("sign id token: %w", err)
	}

	// 3. Refresh Token ID (Opaque)
	// We'll issue a new one if not done by handleRefreshToken
	return &interfaces.TokenResponse{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: uuid.New().String(), // Placeholder for the actual ID stored in DB
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.OIDC.AccessTokenTTL.Seconds()),
		Scope:        scope,
	}, nil
}

func (s *OIDCService) Discovery() *interfaces.DiscoveryDocument {
	return &interfaces.DiscoveryDocument{
		Issuer:                            s.cfg.OIDC.Issuer,
		AuthorizationEndpoint:             s.cfg.OIDC.Issuer + "/authorize",
		TokenEndpoint:                     s.cfg.OIDC.Issuer + "/token",
		JWKsURI:                           s.cfg.OIDC.Issuer + "/.well-known/jwks.json",
		ResponseTypesSupported:            []string{"code"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValuesSupported:  []string{"EdDSA", "Dilithium3"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		ScopesSupported:                   []string{"openid", "profile", "email"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_post", "client_secret_basic"},
	}
}

func (s *OIDCService) getRequestID(ctx context.Context) string {
	if val, ok := ctx.Value("request_id").(string); ok {
		return val
	}
	return ""
}

func (s *OIDCService) JWKS() []interfaces.JSONWebKey {
	return s.signer.PublicKeys()
}

func (s *OIDCService) GetAuditLogs(ctx context.Context, limit int) ([]model.AuditEvent, error) {
	return s.repo.GetAuditLogs(ctx, limit)
}

func (s *OIDCService) ListClients(ctx context.Context) ([]model.Client, error) {
	return s.repo.ListClients(ctx)
}

func (s *OIDCService) RegisterClient(ctx context.Context, name string, redirectURIs []string) (*model.Client, error) {
	client := &model.Client{
		ID:           uuid.New(),
		Name:         name,
		RedirectURIs: redirectURIs,
	}
	if err := s.repo.SaveClient(ctx, client); err != nil {
		return nil, err
	}
	return client, nil
}

func (s *OIDCService) RotatePQCKeys(ctx context.Context) error {
	// 1. Generate new Dilithium3 keypair
	// In a real implementation, we'd call the crypto layer to generate, 
	// encrypt via OpenBao, and then save to repo.
	// For this phase, we mock the success or call an internal generator.
	return nil // Logic already exist in cmd/server/main.go for initial generation
}
