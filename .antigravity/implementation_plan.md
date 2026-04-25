# Implementation Plan: Post-Quantum Secure OIDC Provider (Go)

## Problem Statement

Build a production-grade OpenID Connect Identity Provider in Golang that is compliant with RFC 6749, RFC 7636, OpenID Connect Core 1.0, and RFC 7519 — while layering Post-Quantum Cryptography (PQC) on top for long-term security against quantum computing threats.

> **Agent advisories synthesised from:** `gemma-arch`, `gemma-sec`, `gemma-dev`

---

## Decisions Locked ✅

> [!NOTE]
> **Dual-signature format: Option A — Nested JWS confirmed.**
> Classical JWT (Ed25519/`EdDSA`) is signed first, then the entire JWT string is signed again with Crystals-Dilithium to produce a nested JWS envelope. Validators must verify both signatures. The outer envelope carries a custom `alg: Dilithium3` header identifying the PQC layer.

> [!NOTE]
> **Master key storage: OpenBao (open-source Vault fork) confirmed.**
> All signing private keys and the AES-256-GCM master key are stored and retrieved exclusively via OpenBao's Transit secrets engine. The application never holds raw key material — it sends plaintext to OpenBao and receives ciphertext, or sends ciphertext and receives plaintext. Go client: `github.com/openbao/openbao/api/v2`.

---

## Proposed Solution

A clean-layered Go service: HTTP handlers → OIDC core service → repository + crypto — dependencies flow inward only, all major components depend on interfaces (not concrete types), making every layer independently testable and swappable.

---

## Package Structure

*Synthesised from `gemma-arch` + `gemma-dev` advisories.*

```
go-oidc/
├── cmd/
│   └── server/
│       └── main.go                  # Wire-up and server start
├── internal/
│   ├── config/                      # Env-var loading and validation
│   ├── api/
│   │   ├── handlers/                # HTTP handlers (thin — validate input, call service)
│   │   └── middleware/              # Rate limiting, logging, request ID
│   ├── core/
│   │   ├── oauth2/                  # Authorization code flow, PKCE validation
│   │   └── oidc/                    # ID token construction, discovery doc
│   ├── crypto/
│   │   ├── signer/                  # Dual-signer: Ed25519 + Dilithium
│   │   ├── kem/                     # X25519 + Kyber-768 hybrid TLS config
│   │   ├── cipher/                  # AES-256-GCM encrypt/decrypt
│   │   └── hashing/                 # SHA-3 PKCE challenge, nonce generation
│   └── repository/
│       └── postgres/                # pgx-based PostgreSQL implementation
├── pkg/
│   └── interfaces/
│       ├── crypto.go                # Signer, KeyExchange interfaces
│       ├── repository.go            # Repository interface
│       └── oidc.go                  # OIDCService interface
├── migrations/                      # SQL migration files (up/down)
├── requirements/
│   └── p1-oidc-feature-requirements.md
└── .antigravity/
    └── workflow.md
```

---

## Database Schema

*Tables and encryption approach advised by `gemma-arch`.*

All encrypted fields use AES-256-GCM. Master key is injected via `MASTER_KEY` environment variable.

```sql
-- users
CREATE TABLE users (
    user_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(255) UNIQUE NOT NULL,
    password_hash BYTEA NOT NULL,         -- Argon2id
    email         VARCHAR(255),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata      JSONB
);

-- clients
CREATE TABLE clients (
    client_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_secret_enc BYTEA NOT NULL,     -- AES-256-GCM encrypted
    redirect_uris     TEXT[] NOT NULL,
    scopes            TEXT[] NOT NULL DEFAULT '{openid}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- authorization_codes
CREATE TABLE authorization_codes (
    code_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id        UUID NOT NULL REFERENCES clients(client_id),
    user_id          UUID NOT NULL REFERENCES users(user_id),
    code_hash        BYTEA NOT NULL UNIQUE,   -- SHA-3 hash of the code
    code_challenge   TEXT NOT NULL,           -- S256 PKCE challenge
    redirect_uri     TEXT NOT NULL,
    scopes           TEXT[] NOT NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    used             BOOLEAN NOT NULL DEFAULT false
);

-- refresh_tokens
CREATE TABLE refresh_tokens (
    token_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id     UUID NOT NULL REFERENCES clients(client_id),
    user_id       UUID NOT NULL REFERENCES users(user_id),
    token_enc     BYTEA NOT NULL,            -- AES-256-GCM encrypted
    expires_at    TIMESTAMPTZ NOT NULL,
    issued_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked       BOOLEAN NOT NULL DEFAULT false
);

-- audit_log (append-only — no UPDATE/DELETE permissions)
CREATE TABLE audit_log (
    id         BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(64) NOT NULL,
    actor_id   UUID,
    client_id  UUID,
    metadata   JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- pqc_keys (Fallback for Dilithium3 keys until OpenBao native support)
CREATE TABLE pqc_keys (
    key_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    algorithm          VARCHAR(50) NOT NULL, -- e.g. 'Dilithium3'
    encrypted_key_blob BYTEA NOT NULL,       -- Private key encrypted via OpenBao Transit
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## Key Interfaces

*Design advised by `gemma-dev`.*

```go
// pkg/interfaces/crypto.go
type Signer interface {
    Sign(ctx context.Context, claims jwt.MapClaims) (string, error)
    PublicKeys() []jose.JSONWebKey   // for JWKS endpoint
}

type Encryptor interface {
    Encrypt(plaintext []byte) ([]byte, error)
    Decrypt(ciphertext []byte) ([]byte, error)
}

// pkg/interfaces/repository.go
type Repository interface {
    GetClient(ctx context.Context, clientID string) (*model.Client, error)
    SaveAuthCode(ctx context.Context, code *model.AuthCode) error
    GetAuthCode(ctx context.Context, codeHash []byte) (*model.AuthCode, error)
    MarkAuthCodeUsed(ctx context.Context, codeID uuid.UUID) error
    SaveRefreshToken(ctx context.Context, token *model.RefreshToken) error
    RotateRefreshToken(ctx context.Context, oldID uuid.UUID, newToken *model.RefreshToken) error
    GetUser(ctx context.Context, userID uuid.UUID) (*model.User, error)
    AppendAuditLog(ctx context.Context, event *model.AuditEvent) error
}
```

---

## Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/authorize` | GET | Authorization code flow entry. **PKCE mandatory** — reject if no `code_challenge`. |
| `/token` | POST | Exchange code → tokens. Handle `authorization_code` and `refresh_token` grants. Issue dual-signed JWTs. Strict refresh token rotation. |
| `/.well-known/openid-configuration` | GET | Discovery document. Advertises PQC signing algorithm. |
| `/.well-known/jwks.json` | GET | JSON Web Key Set including the Dilithium public key. |

---

## PQC Crypto Strategy

*Risks and controls advised by `gemma-sec`.*

### Dual-Signing — Option A (Nested JWS)
1. Build standard JWT claims (subject, issuer, audience, expiry, nonces).
2. Sign with **Ed25519** (key from OpenBao Transit) → produces classical JWT string (the inner token).
3. Treat the entire inner JWT string as the payload for the outer JWS.
4. Sign outer payload with **Crystals-Dilithium mode3** (key from OpenBao Transit) → outer JWS envelope.
5. Outer JWS header: `{"alg":"Dilithium3","cty":"JWT"}` — signals nested JWT to validators.
6. Final token format: `<dilithium-header>.<base64(inner-jwt)>.<dilithium-sig>`

### PQC Key Storage (Adapter Pattern)
- OpenBao Transit currently lacks native Dilithium3 support.
- **Strategy:** Dilithium private keys are generated via `circl`, encrypted with an AES-256-GCM key in OpenBao Transit, and stored in the `pqc_keys` table.
- **Signing Flow:** Fetch encrypted blob → Decrypt via OpenBao Transit → Sign via `circl` → Clear raw key from memory.
- This allows a seamless move to native OpenBao support once available.

### Hybrid TLS (X25519 + Kyber-768)
- Use `circl`'s `kem/hybrid` package to configure a custom `tls.Config`.
- Server **must reject** if either X25519 or Kyber share fails — no partial-security fallback.

### PKCE Code Challenge
- `code_challenge_method=S256` is the **only** accepted method.
- Challenge computed with **SHA-3 (SHA3-256)** over the verifier.

### Nonces and State
- All nonces and `state` values: `crypto/rand` with minimum 32 bytes (256 bits).

---

## Security Controls

*From `gemma-sec` threat model.*

| Control | Implementation |
|---------|---------------|
| PKCE mandatory | `400 Bad Request` if `code_challenge` absent on `/authorize` |
| Hybrid AND validation | Reject token if EITHER signature (classical or PQC) is invalid |
| Code replay prevention | `used` flag + `SELECT FOR UPDATE` on `authorization_codes` |
| Refresh token rotation | Old token revoked atomically with new token issued (single TX) |
| Rate limiting | Per-IP + per-`client_id` middleware on `/authorize` and `/token` |
| Secrets never logged | Structured logger with explicit redaction on all token/key fields |
| Audit log immutability | DB role with INSERT-only on `audit_log`; no UPDATE/DELETE granted |
| Input validation | Strict allowlist on `scope`, `redirect_uri`, `response_type` |
| TLS minimum | `tls.VersionTLS13` hardcoded — TLS 1.2 and below rejected |
| Master key storage | **OpenBao Transit engine** — app never holds raw keys; all encrypt/decrypt/sign via OpenBao API |

---

## Test Strategy

| Layer | Type | What to test |
|-------|------|-------------|
| `crypto/signer` | Unit | Dual-sign produces valid classical + PQC signatures; tampered payload rejected |
| `crypto/cipher` | Unit | Encrypt→decrypt roundtrip; wrong key fails |
| `crypto/hashing` | Unit | SHA3-256 PKCE challenge matches verifier; 256-bit nonce length |
| `core/oauth2` | Unit | PKCE validation logic; missing challenge → error; code replay → error |

## Phase 2: Performance & Audit Hardening (1M TPS)
- **Goal:** Transform the monolithic/single-node provider into a global-scale engine.
- **Key metrics:** < 10ms latency, 1M+ TPS capability.

### Phase 2a: Distributed Rate Limiting (Redis) [NEW]
- Transition from `MemoryStore` to `RedisStore`.
- Use Redis LUA scripts for atomic rate-limit checks (sliding window).
- Key Format: `ratelimit:{endpoint}:{ip}:{client_id}`.

### Phase 2b: Asynchronous Audit Pipeline [NEW]
- Moving audit log writes from the core transaction to an asynchronous buffer (worker pool pattern).
- Implement a `BatchingRepository` that flushes audit logs every 100ms or 1000 events to optimize DB I/O.
- Preparation for Kafka/Redpanda integration.

### Phase 2c: Performance Optimization [NEW]
- Use `sync.Pool` for JSON encoders and common crypto buffers to reduce GC pressure.
- Profile signing operations to identify Bottlenecks in the `circl` Dilithium implementation.
- Implement Connection Pooling optimizations for PostgreSQL.

### Phase 2d: Forensic Observability [NEW]
- Standardize all logs with OpenTelemetry Span IDs and Trace IDs.
- Implement an "Audit Integrity" check using SHA3-256 chaining (optional/deferred).

| `repository/postgres` | Integration | All CRUD + rotation + `used` flag (Testcontainers) |
| `api/handlers` | Integration | Full HTTP flows: authorize → token exchange; refresh token rotation |
| Security | Negative | No `code_challenge` → 400; expired code → 400; replayed code → 400 |

---

## Phased Delivery

| Phase | Deliverables |
|-------|-------------|
| **2a** | Project scaffold: `go mod init`, directory structure, all interfaces, config loading, DB migrations |
| **2b** | Crypto layer: AES-256-GCM encryptor, SHA-3 hashing, Ed25519 signer (classical baseline first) |
| **2c** | Repository layer: all PostgreSQL queries with Testcontainers integration tests |
| **2d** | Core OIDC flows: `/authorize` + `/token` (auth code + refresh), full PKCE enforcement |
| **2e** | PQC layer: Dilithium dual-signer + Kyber-768 hybrid TLS |
| **2f** | Discovery + JWKS endpoints, rate limiting middleware, audit logging |
| **2g** | Full test suite green → code review → walkthrough |

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/cloudflare/circl` | PQC primitives (Dilithium, Kyber) |
| `github.com/jackc/pgx/v5` | PostgreSQL driver |
| `github.com/golang-jwt/jwt/v5` | Classical JWT layer |
| `github.com/google/uuid` | UUID generation |
| `golang.org/x/crypto` | Argon2id, SHA-3 |
| `github.com/testcontainers/testcontainers-go` | PostgreSQL + OpenBao in integration tests |
| `github.com/openbao/openbao/api/v2` | OpenBao client for Transit secrets engine |

---

## Risks

| Risk | Severity | Mitigation |
|------|----------|-----------|
| PQC algorithm agility (NIST specs may shift) | Medium | `Signer` interface is swappable without touching core logic |
| Dilithium large signature size (~2.5 KB) adds latency | Medium | Benchmark in Phase 2e; cache JWKS endpoint |
| OpenBao availability (single point) | High | Dev: single node; prod: HA cluster with Raft storage documented |
| `circl` timing side-channels | Medium | Use only high-level API; flag for external crypto audit |
