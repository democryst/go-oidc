// Package interfaces defines the service contract for the OIDC core flows.
package interfaces

import (
	"context"

	"github.com/democryst/go-oidc/internal/model"
)

// AuthorizeRequest holds validated parameters from the /authorize endpoint.
type AuthorizeRequest struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string // must be "code"
	Scope               string // space-separated: "openid profile email"
	State               string // caller-supplied CSRF token (256-bit min)
	Nonce               string // replay protection for ID tokens
	CodeChallenge       string // BASE64URL(SHA3-256(verifier))
	CodeChallengeMethod string // must be "S256"
}

// AuthorizeResponse carries the authorization code back to the redirect URI.
type AuthorizeResponse struct {
	Code  string // raw, single-use authorization code
	State string // echoed from the request
}

// TokenRequest holds validated parameters from the /token endpoint.
type TokenRequest struct {
	GrantType    string // "authorization_code" | "refresh_token"
	ClientID     string
	ClientSecret string
	// For authorization_code grant:
	Code         string
	RedirectURI  string
	CodeVerifier string // raw PKCE verifier
	// For refresh_token grant:
	RefreshToken string
}

// TokenResponse carries the issued tokens.
type TokenResponse struct {
	// AccessToken is a nested JWS: outer=Dilithium3, inner=EdDSA JWT.
	AccessToken  string
	// IDToken is a nested JWS, same format, with OIDC claims.
	IDToken      string
	// RefreshToken is an opaque rotating token.
	RefreshToken string
	TokenType    string // always "Bearer"
	ExpiresIn    int64  // access token lifetime in seconds
	Scope        string
}

// DiscoveryDocument is the /.well-known/openid-configuration response body.
type DiscoveryDocument struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	JWKsURI                           string   `json:"jwks_uri"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// OIDCService is the top-level service interface for all OIDC flows.
// Handlers call this; it orchestrates the repository and crypto layers.
type OIDCService interface {
	// Authorize validates the request and issues an authorization code.
	// Returns an error if PKCE is missing, the client is unknown, or the
	// redirect URI is not registered.
	Authorize(ctx context.Context, req AuthorizeRequest) (*AuthorizeResponse, error)

	// Token exchanges a code or refresh token for access + ID tokens.
	// Enforces PKCE, single-use codes, and refresh token rotation.
	Token(ctx context.Context, req TokenRequest) (*TokenResponse, error)

	// Discovery returns the OpenID Connect discovery document.
	Discovery() *DiscoveryDocument

	// JWKS returns public keys for both the Ed25519 and Dilithium3 signers.
	JWKS() []JSONWebKey

	// Admin Actions
	GetAuditLogs(ctx context.Context, limit int) ([]model.AuditEvent, error)
	ListClients(ctx context.Context) ([]model.Client, error)
	RegisterClient(ctx context.Context, name string, redirectURIs []string) (*model.Client, error)
	RotatePQCKeys(ctx context.Context) error
}
