# Developer Guide: Contributing to the PQC OIDC Provider

This guide provides developers with the knowledge needed to set up, modify, and extend the Post-Quantum Secure OIDC Provider. For design details, see [ARCHITECTURE.md](ARCHITECTURE.md).

## 1. Local Development Setup

To run the full stack locally, you need:
- **Go 1.21+**
- **Docker** (for PostgreSQL and OpenBao)
- **OpenBao** (or Vault) running the Transit engine.

### Quick Start with Docker
```bash
# Start a development database
docker run --name oidc-db -e POSTGRES_PASSWORD=password -p 5432:5432 -d postgres:15-alpine

# Start OpenBao in dev mode (for testing only)
docker run --name openbao -p 8200:8200 -e "BAO_DEV_ROOT_TOKEN_ID=root" -d openbao/openbao
```

### Configuration
Initialize your environment variables (or a `.env` file):
```bash
export DATABASE_DSN="postgres://postgres:password@localhost:5432/postgres?sslmode=disable"
export OPENBAO_ADDR="http://localhost:8200"
export OPENBAO_TOKEN="root"
export TLS_CERT_FILE="certs/server.crt"
export TLS_KEY_FILE="certs/server.key"
export OIDC_ISSUER="http://localhost:8080"
```

## 2. Infrastructure Patterns

### Interface-First Development
This project follows a strict "Ports & Adapters" pattern. **Never** depend on a concrete implementation (e.g., `*PostgresRepository`) in your core services.

1.  **Define the Interface**: Add methods to `pkg/interfaces/`.
2.  **Implement the Adapter**: Create the logic in `internal/repository/` or `internal/crypto/`.
3.  **Wire in Main**: Add the initialization to `cmd/server/main.go`.

### Error Handling
Use `fmt.Errorf("context: %w", err)` to wrap errors. This preserves the error chain for debugging while allowing handlers to check for specific error types using `errors.Is` or `errors.As`.

## 3. Cryptographic Extensions

If you need to add a new PQC algorithm:
1.  Add the algorithm string to the `model` or use it in the `pqc_keys` table.
2.  Implement a new `Signer` or `KEM` in `internal/crypto/`.
3.  Update the `DualSigner` if you want to support multiple outer JWS algorithms.

## 4. Testing Strategy

### Unit Tests
- Prefer unit tests with mocks for business logic (`internal/core/oidc`).
- Use `testify/mock` to simulate repository and crypto behavior.
- Run: `go test ./internal/core/oidc/...`

### Integration Tests
- Repository tests in `internal/repository/postgres` use **Testcontainers**.
- These tests automatically spin up a real PostgreSQL container.
- Run: `go test ./internal/repository/postgres/...` (Requires Docker).

## 5. Security Checklist for New Code
- **No Secrets in Logs**: Use the `middleware.Logger` and ensure any new request fields carrying secrets are redacted.
- **Constant Time**: Any part of the code that compares hashes, verifiers, or signatures **must** use `subtle.ConstantTimeCompare`.
- **Atomic Rotation**: Any state change involving token validity must be atomic (Database transactions).
- **Graceful Failures**: If a PQC signature fails, the entire request must fail. Never fall back to classical-only "silently".
