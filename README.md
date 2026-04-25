# Post-Quantum Secure OIDC Provider

A production-grade OpenID Connect Identity Provider implemented in Go, featuring hybrid Post-Quantum Cryptography (PQC) for future-proof security.

## Features
- **Hybrid PQC:** Nested JWS (Ed25519 inner, Crystals-Dilithium3 outer).
- **KMS Integration:** Native support for **OpenBao Transit** (and HashiCorp Vault) for key management.
- **Strict Compliance:** OAuth2/OIDC with mandatory PKCE S256.
- **Persistence:** High-performance PostgreSQL repository with atomic token rotation.
- **Observability:** Structured JSON logging and per-IP/client rate limiting.

## Documentation
- [ARCHITECTURE.md](docs/ARCHITECTURE.md): High-level design and PQC logic.
- [DEVELOPER_GUIDE.md](docs/DEVELOPER_GUIDE.md): Local setup and contributing.
- [API_SPEC.md](docs/API_SPEC.md): Endpoint parameters and security requirements.

## Deployment & OpenBao HA
For production environments, OpenBao must be configured in High Availability (HA) mode.

### 1. Cluster Setup
- Deploy at least 3 OpenBao nodes across different availability zones.
- Use a resilient storage backend (e.g., Raft or Consul).
- Enable the **Transit Secrets Engine** on a specific mount (default: `transit`).

### 2. Key Provisioning
- **Classical:** Create an Ed25519 key in Transit: `openbao write -f transit/keys/oidc-ed25519 type=ed25519`.
- **Encryption:** Create an AES-256-GCM key for secret protection: `openbao write -f transit/keys/oidc-aes256 type=aes256-gcm96`.

### 3. HA Connectivity
- Use a load balancer in front of the OpenBao nodes.
- Set `OPENBAO_ADDR` to the load balancer URL.
- The Go client automatically handles connection pooling and retries.

## Environment Variables
- `DATABASE_DSN`: PostgreSQL connection string.
- `OPENBAO_ADDR`: OpenBao server URL.
- `OPENBAO_TOKEN`: AppRole or Service Account token.
- `TLS_CERT_FILE`: Path to the server TLS certificate.
- `TLS_KEY_FILE`: Path to the server TLS private key.
- `OIDC_ISSUER`: Canonical issuer URL.

## Development
```bash
# Run unit tests
go test ./...

# Run performance benchmark
cd internal/crypto/signer && go test -bench=.

# Build the server
go build -o oidc-server ./cmd/server/main.go
```
