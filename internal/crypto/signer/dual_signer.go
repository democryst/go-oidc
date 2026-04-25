package signer

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/cloudflare/circl/sign/dilithium/mode3"

	"github.com/democryst/go-oidc/pkg/interfaces"
)

// DualSigner implements the Nested JWS (Option A) format.
// It wraps a classical Ed25519 JWT in a Dilithium3 JWS envelope.
type DualSigner struct {
	classicalSigner interfaces.Signer
	pqcKeyFetcher   func(context.Context) (*mode3.PrivateKey, error)
}

func NewDualSigner(classical interfaces.Signer, pqcKeyFetcher func(context.Context) (*mode3.PrivateKey, error)) interfaces.Signer {
	return &DualSigner{
		classicalSigner: classical,
		pqcKeyFetcher:   pqcKeyFetcher,
	}
}

func (s *DualSigner) PublicKeys() []interfaces.JSONWebKey {
	// Combines Ed25519 and Dilithium public keys
	// In a real implementation, it would fetch both from KMS/DB
	return s.classicalSigner.PublicKeys() // Placeholder: need to add Dilithium3 JWK
}

func (s *DualSigner) Sign(ctx context.Context, claims interfaces.TokenClaims) (string, error) {
	// 1. Generate Inner JWT (Ed25519)
	innerJWT, err := s.classicalSigner.Sign(ctx, claims)
	if err != nil {
		return "", fmt.Errorf("inner sign failed: %w", err)
	}

	// 2. Fetch Dilithium3 Private Key
	privKey, err := s.pqcKeyFetcher(ctx)
	if err != nil {
		return "", fmt.Errorf("pqc key fetch failed: %w", err)
	}

	// 3. Construct Outer JWS (Option A)
	// Outer Header: {"alg":"Dilithium3","cty":"JWT"}
	header := `{"alg":"Dilithium3","cty":"JWT"}`
	headerB64 := base64.RawURLEncoding.EncodeToString([]byte(header))
	
	// Payload is the already signed inner JWT string
	payloadB64 := base64.RawURLEncoding.EncodeToString([]byte(innerJWT))

	signingString := headerB64 + "." + payloadB64
	
	// 4. Sign Outer JWS using Dilithium3 (mode3)
	// sk.Sign(rand, message, opts) - returns ([]byte, error)
	sig, err := privKey.Sign(nil, []byte(signingString), nil)
	if err != nil {
		return "", fmt.Errorf("pqc outer sign failed: %w", err)
	}
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	// Combine: Header.Payload.Signature
	return signingString + "." + sigB64, nil
}
