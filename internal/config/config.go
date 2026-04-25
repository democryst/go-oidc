// Package config loads and validates all server configuration from environment
// variables. No configuration is hardcoded. Secrets are never stored here —
// only addresses and non-sensitive identifiers.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the OIDC server.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	OpenBao  OpenBaoConfig
	OIDC     OIDCConfig
}

// ServerConfig controls the HTTP server behaviour.
type ServerConfig struct {
	// Addr is the listen address, e.g. ":8443".
	Addr string
	// TLSCertFile and TLSKeyFile are paths to the TLS certificate and private key.
	// These are the classical TLS credentials; the hybrid KEM is layered on top.
	TLSCertFile string
	TLSKeyFile  string
	// ReadTimeout / WriteTimeout guard against slow-loris attacks.
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	// RateLimit is the maximum requests per second per client_id+IP pair.
	RateLimit int
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	// DSN is the full PostgreSQL connection string.
	// Example: postgres://user:pass@host:5432/dbname?sslmode=require
	DSN string
	// MaxConns is the pgx connection pool size.
	MaxConns int
}

// OpenBaoConfig holds connection settings for the OpenBao (Vault-compatible) KMS.
// All private key operations and secret encryption happen inside OpenBao.
// The application never receives raw key material.
type OpenBaoConfig struct {
	// Address is the OpenBao server URL, e.g. https://openbao:8200
	Address string
	// Token is the AppRole or service-account token used to authenticate.
	// In production this should be injected via a secrets operator, not env.
	Token string
	// TransitMount is the path of the Transit secrets engine, default "transit".
	TransitMount string
	// Ed25519KeyName is the named key in Transit used for classical JWT signing.
	Ed25519KeyName string
	// DilithiumKeyName is the named key used for PQC (outer JWS) signing.
	// OpenBao's Transit BYOK import is used to register the circl-generated key.
	DilithiumKeyName string
	// EncryptionKeyName is the Transit key used for AES-256-GCM secrets-at-rest.
	EncryptionKeyName string
}

// OIDCConfig holds values that appear in the discovery document and tokens.
type OIDCConfig struct {
	// Issuer is the canonical URL of this IdP, e.g. https://auth.example.com
	Issuer string
	// AccessTokenTTL is how long access tokens remain valid.
	AccessTokenTTL time.Duration
	// IDTokenTTL is how long ID tokens remain valid.
	IDTokenTTL time.Duration
	// RefreshTokenTTL is how long refresh tokens remain valid before rotation.
	RefreshTokenTTL time.Duration
	// AuthCodeTTL is how long authorization codes remain valid (RFC recommends ≤10m).
	AuthCodeTTL time.Duration
}

// Load reads all configuration from environment variables and returns a validated
// Config. Returns an error listing every missing/invalid variable so operators
// do not have to fix them one at a time.
func Load() (*Config, error) {
	var errs []error

	cfg := &Config{
		Server: ServerConfig{
			Addr:         getEnvDefault("SERVER_ADDR", ":8443"),
			TLSCertFile:  requireEnv("TLS_CERT_FILE", &errs),
			TLSKeyFile:   requireEnv("TLS_KEY_FILE", &errs),
			ReadTimeout:  parseDuration("SERVER_READ_TIMEOUT", 5*time.Second, &errs),
			WriteTimeout: parseDuration("SERVER_WRITE_TIMEOUT", 10*time.Second, &errs),
			RateLimit:    parseInt("RATE_LIMIT_RPS", 100, &errs),
		},
		Database: DatabaseConfig{
			DSN:      requireEnv("DATABASE_DSN", &errs),
			MaxConns: parseInt("DATABASE_MAX_CONNS", 20, &errs),
		},
		OpenBao: OpenBaoConfig{
			Address:           requireEnv("OPENBAO_ADDR", &errs),
			Token:             requireEnv("OPENBAO_TOKEN", &errs),
			TransitMount:      getEnvDefault("OPENBAO_TRANSIT_MOUNT", "transit"),
			Ed25519KeyName:    getEnvDefault("OPENBAO_ED25519_KEY", "oidc-ed25519"),
			DilithiumKeyName:  getEnvDefault("OPENBAO_DILITHIUM_KEY", "oidc-dilithium3"),
			EncryptionKeyName: getEnvDefault("OPENBAO_ENCRYPT_KEY", "oidc-aes256"),
		},
		OIDC: OIDCConfig{
			Issuer:          requireEnv("OIDC_ISSUER", &errs),
			AccessTokenTTL:  parseDuration("OIDC_ACCESS_TOKEN_TTL", 15*time.Minute, &errs),
			IDTokenTTL:      parseDuration("OIDC_ID_TOKEN_TTL", 15*time.Minute, &errs),
			RefreshTokenTTL: parseDuration("OIDC_REFRESH_TOKEN_TTL", 24*time.Hour, &errs),
			AuthCodeTTL:     parseDuration("OIDC_AUTH_CODE_TTL", 5*time.Minute, &errs),
		},
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("configuration errors:\n%w", errors.Join(errs...))
	}
	return cfg, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func requireEnv(key string, errs *[]error) string {
	v := os.Getenv(key)
	if v == "" {
		*errs = append(*errs, fmt.Errorf("required environment variable %q is not set", key))
	}
	return v
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseDuration(key string, def time.Duration, errs *[]error) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("invalid duration for %q: %w", key, err))
		return def
	}
	return d
}

func parseInt(key string, def int, errs *[]error) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("invalid integer for %q: %w", key, err))
		return def
	}
	return n
}
