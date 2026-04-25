package signer

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/cloudflare/circl/sign/dilithium/mode3"
	"github.com/democryst/go-oidc/pkg/interfaces"
)

// MockClassical returns a signature instantly.
type MockClassical struct{}

func (m *MockClassical) Sign(ctx context.Context, claims interfaces.TokenClaims) (string, error) {
	return "eyJhbGciOiJFZERTQSJ9.eyJzdWIiOiIxMjMifQ.sig", nil
}
func (m *MockClassical) PublicKeys() []interfaces.JSONWebKey { return nil }

func BenchmarkDualSigner_Sign(b *testing.B) {
	ctx := context.Background()
	_, priv, _ := mode3.GenerateKey(rand.Reader)
	
	// Fast fetcher
	fetcher := func(ctx context.Context) (*mode3.PrivateKey, error) {
		return priv, nil
	}

	ds := NewDualSigner(&MockClassical{}, fetcher)
	claims := interfaces.TokenClaims{
		Subject: "user-123",
		Issuer:  "https://auth.example.com",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ds.Sign(ctx, claims)
		if err != nil {
			b.Fatal(err)
		}
	}
}
