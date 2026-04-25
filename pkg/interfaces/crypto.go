package interfaces
 
 import "context"

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

const (
	RequestIDKey ContextKey = "request_id"
)

// TokenClaims holds the standard OIDC/OAuth2 JWT claims.
type TokenClaims struct {
	Subject   string
	Issuer    string
	Audience  []string
	ExpiresAt int64
	IssuedAt  int64
	Nonce     string            // anti-replay for ID tokens
	Scope     string            // space-separated list
	ClientID  string
	Extra     map[string]any    // additional claims (profile, email, etc.)
}

// JSONWebKey is a representation of a public key for the JWKS endpoint.
// It includes standard fields (kid, kty, use) and PQC-specific extensions.
type JSONWebKey struct {
	KeyID     string         `json:"kid"`
	KeyType   string         `json:"kty"`      // e.g. "OKP" or "ML-DSA"
	Algorithm string         `json:"alg"`      // e.g. "EdDSA" or "ML-DSA-65"
	Use       string         `json:"use"`      // "sig"
	Crv       string         `json:"crv,omitempty"` // For OKP (Ed25519)
	X         string         `json:"x,omitempty"`   // Base64Url public key (for OKP/ML-DSA)
	Params    map[string]any `json:"params,omitempty"` // Extra parameters
}

// Signer defines the contract for issuing dual-signed (Option A Nested JWS) tokens.
// Implementations must use OpenBao Transit for all private key operations.
//
// Token format (Option A):
//   outer = Dilithium3-signed JWS whose payload = base64url(inner JWT)
//   inner = standard EdDSA-signed JWT
type Signer interface {
	// Sign issues a nested JWS token from the provided claims.
	// The inner JWT is signed with Ed25519; the outer JWS with Dilithium3.
	// Private keys are never held by the caller — all signing via OpenBao.
	Sign(ctx context.Context, claims TokenClaims) (token string, err error)

	// PublicKeys returns the JWKS entries for the /.well-known/jwks.json endpoint.
	// Returns entries for both the Ed25519 key and the Dilithium3 key.
	PublicKeys() []JSONWebKey
}

// Encryptor defines the contract for symmetric encryption of secrets at rest.
// Implementations delegate to OpenBao Transit (encrypt/decrypt endpoints).
// The application never handles raw key material.
type Encryptor interface {
	// Encrypt encrypts plaintext and returns opaque ciphertext.
	Encrypt(ctx context.Context, plaintext []byte) (ciphertext []byte, err error)
	// Decrypt decrypts ciphertext produced by Encrypt.
	Decrypt(ctx context.Context, ciphertext []byte) (plaintext []byte, err error)
}

// Hasher defines the contract for one-way hashing (PKCE challenges, code hashing).
type Hasher interface {
	// PKCEChallenge computes the S256 code challenge from a verifier using SHA3-256.
	// Challenge = BASE64URL(SHA3-256(ASCII(code_verifier)))
	PKCEChallenge(verifier string) string

	// VerifyPKCE checks whether a verifier matches its stored challenge.
	VerifyPKCE(verifier, challenge string) bool

	// HashCode computes a SHA3-256 hash of a raw authorization code for storage.
	HashCode(rawCode string) []byte

	// SecureNonce returns a cryptographically random nonce of at least 32 bytes
	// (256 bits), base64url-encoded.
	SecureNonce() (string, error)
}
