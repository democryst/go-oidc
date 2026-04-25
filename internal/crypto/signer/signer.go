package signer

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/openbao/openbao/api/v2"

	"github.com/democryst/go-oidc/pkg/interfaces"
)

// OpenBaoSigner implements interfaces.Signer using OpenBao Transit.
// In Phase 2b, this only does Ed25519 classical signing.
type OpenBaoSigner struct {
	client         *api.Client
	transitMount   string
	ed25519KeyName string
}

// NewOpenBaoSigner creates a classical Ed25519 signer via OpenBao.
func NewOpenBaoSigner(client *api.Client, mount, ed25519KeyName string) interfaces.Signer {
	if mount == "" {
		mount = "transit"
	}
	return &OpenBaoSigner{
		client:         client,
		transitMount:   mount,
		ed25519KeyName: ed25519KeyName,
	}
}

func (s *OpenBaoSigner) Sign(ctx context.Context, claims interfaces.TokenClaims) (string, error) {
	// 1. Build standard jwt Claims map
	mapClaims := jwt.MapClaims{
		"sub":       claims.Subject,
		"iss":       claims.Issuer,
		"exp":       claims.ExpiresAt,
		"iat":       claims.IssuedAt,
		"client_id": claims.ClientID,
	}
	if len(claims.Audience) > 0 {
		mapClaims["aud"] = claims.Audience
	}
	if claims.Scope != "" {
		mapClaims["scope"] = claims.Scope
	}
	if claims.Nonce != "" {
		mapClaims["nonce"] = claims.Nonce
	}
	for k, v := range claims.Extra {
		mapClaims[k] = v
	}

	token := jwt.NewWithClaims(&openBaoMethod{
		client: s.client,
		path:   fmt.Sprintf("%s/sign/%s", s.transitMount, s.ed25519KeyName),
		ctx:    ctx,
	}, mapClaims)

	// Since the signing method holds the client/path locally, the 'key' param is ignored
	signedString, err := token.SignedString(nil)
	if err != nil {
		return "", fmt.Errorf("failed to sign token via OpenBao: %w", err)
	}

	return signedString, nil
}

func (s *OpenBaoSigner) PublicKeys() []interfaces.JSONWebKey {
	path := fmt.Sprintf("%s/keys/%s", s.transitMount, s.ed25519KeyName)
	secret, err := s.client.Logical().Read(path)
	if err != nil {
		return nil
	}
	if secret == nil || secret.Data == nil {
		return nil
	}

	keys, ok := secret.Data["keys"].(map[string]any)
	if !ok {
		return nil
	}

	// Fetch the latest version's public key
	var latestKey string
	latestVersion := 0
	for k, v := range keys {
		vers, _ := strconv.Atoi(k)
		if vers > latestVersion {
			latestVersion = vers
			kMap, _ := v.(map[string]any)
			latestKey, _ = kMap["public_key"].(string)
		}
	}

	if latestKey == "" {
		return nil
	}

	return []interfaces.JSONWebKey{
		{
			KeyID:     fmt.Sprintf("%s-v%d", s.ed25519KeyName, latestVersion),
			KeyType:   "OKP",
			Algorithm: "EdDSA",
			Use:       "sig",
			Crv:       "Ed25519",
			X:         latestKey,
		},
	}
}

// ─── Custom jwt.SigningMethod for OpenBao Transit ────────────────────────────

type openBaoMethod struct {
	client *api.Client
	path   string
	ctx    context.Context
}

func (m *openBaoMethod) Alg() string {
	return "EdDSA"
}

func (m *openBaoMethod) Verify(signingString string, signature []byte, key any) error {
	return fmt.Errorf("Verify not implemented for OpenBao signer")
}

func (m *openBaoMethod) Sign(signingString string, key any) ([]byte, error) {
	input := base64.StdEncoding.EncodeToString([]byte(signingString))

	reqData := map[string]interface{}{
		"input":          input,
		"marshaling_algorithm": "jws", // Sometimes required by Vault/Bao for Ed25519
	}

	secret, err := m.client.Logical().WriteWithContext(m.ctx, m.path, reqData)
	if err != nil {
		return nil, fmt.Errorf("openbao sign failed: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("openbao returned empty response")
	}

	sigVal, ok := secret.Data["signature"].(string)
	if !ok {
		return nil, fmt.Errorf("openbao response missing signature field")
	}

	// OpenBao returns signatures like "vault:v1:base64sig"
	parts := strings.Split(sigVal, ":")
	var rawBase64 string
	if len(parts) >= 3 && parts[0] == "vault" && parts[1] == "v1" {
		rawBase64 = parts[2]
	} else {
		return nil, fmt.Errorf("unexpected openbao signature format: %s", sigVal)
	}

	// Vault/OpenBao returns standard base64 encoding (oftentimes padded).
	// JWT requires URL-safe base64 without padding.
	sigBytes, err := base64.StdEncoding.DecodeString(rawBase64)
	if err != nil {
		// Try base64 URL encoding just in case OpenBao switched formats
		sigBytes, err = base64.URLEncoding.DecodeString(rawBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode openbao signature base64: %w", err)
		}
	}

	return sigBytes, nil
}
