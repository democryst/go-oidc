# Technical Architecture: Post-Quantum Secure OIDC Provider

This document provides a comprehensive overview of the internal workings, cryptographic choices, and security architecture of the Post-Quantum Secure OIDC Identity Provider, designed for 1 Million TPS scale.

## 1. System Overview
The provider is a standard-compliant OIDC Identity Provider resilient against quantum computer attacks. It utilizes **Hybrid Cryptography**, combining NIST-approved classical algorithms (Ed25519) with Post-Quantum primitives (Dilithium3).

## 2. Layered Architecture
The project follows a **Hexagonal Architecture** (Ports & Adapters) to decouple business logic from infrastructure.

- **Domain Model (`internal/model`)**: Pure data structures (Client, User, Token).
- **Core Logic (`internal/core/oidc`)**: Orchestrates the OIDC flows (Authorize, Token, Admin).
- **Ports (`pkg/interfaces`)**: Define contracts for Repository, Signer, Hasher, and Encryptor.
- **Adapters**:
    - **Performance Repository (`internal/repository/postgres`)**: Multi-modal DB adapter supporting synchronous queries and asynchronous **Batch Logging** for high-throughput audit trails.
    - **OpenBao (`internal/crypto/signer`)**: KMS-backed signing and encryption.
    - **Global Cache (`internal/api/middleware`)**: Redis-backed distributed rate limiting.
    - **API (`internal/api/handlers`)**: HTTP entry points including OIDC, Admin UI, and Health Probes.

## 3. Cryptographic Design

### 3.1. Nested Dual-Signing (Option A)
All tokens are dual-signed using a **Nested JWS** approach for backward compatibility and future security.

- **Inner Layer**: A standard Ed25519 JWT signed via OpenBao Transit. Ensures current OIDC clients can verify tokens classically.
- **Outer Layer**: The entire Ed25519 JWT string is the payload for a **Crystals-Dilithium3** signature (Headers: `{"alg":"Dilithium3","cty":"JWT"}`).

### 3.2. PQC Key Storage (Adapter Pattern)
As many KMS systems do not yet natively support Dilithium3:
1.  **Protection**: Private keys are encrypted with **AES-256-GCM** via OpenBao Transit.
2.  **Storage**: Encrypted blobs reside in the `pqc_keys` table.
3.  **Wiping**: Raw key material is decrypted in-memory only for signing and then immediately ready for GC.

## 4. High-Scale Engineering (1M TPS Strategy)

### 4.1. Distributed Rate Limiting
Utilizes a **Redis-backed middleware** with atomic Lua scripts. This allows the provider to throttle clients globally across multiple Kubernetes pods with sub-millisecond overhead.

### 4.2. Asynchronous Audit Pipeline
Under high load (1M TPS), synchronous I/O to PostgreSQL is a bottleneck. We implement an **Asynchronous Batch Repository**:
- Events are queued in a high-concurrency buffer.
- A background worker pool flushes events to the `audit_log` table using the **PostgreSQL COPY protocol** for maximum ingestion speed.

### 4.3. Resource Efficiency
- **`sync.Pool`**: Used for JSON encoders and byte buffers to drastically reduce GC pressure during high concurrent signing operations.
- **PgBouncer**: Mandatory connection proxy in `transaction` mode to handle the "thundering herd" of thousands of incoming OIDC requests.

## 5. Security & Governance

### 5.1. Forensic Observability
Every request is injected with a `RequestID` at the middleware layer. This ID propagates through the OIDC flow and is recorded in the asynchronous audit log, enabling full forensic correlation from initial authorize to final token issue.

### 5.2. Zero-Trust Kubernetes
The deployment follows the **"Restricted" Pod Security Standard**:
- **Distroless Runtime**: Minimal attack surface (no shell).
- **Egress Isolation**: `NetworkPolicy` allows traffic only to authorized Redis, Postgres, and OpenBao endpoints.

## 6. Project Structure
```text
.
├── cmd/             # Binaries (Server, Stress Test)
├── internal/
│   ├── api/         # Handlers, Middleware, Health, Admin
│   ├── core/oidc/   # Service Orchestration & Admin Logic
│   ├── crypto/      # Dual-Signer, PQC Adapters, Metrics
│   ├── model/       # Domain Objects & Audit Events
│   └── repository/  # PostgreSQL (Batch), Redis Adapters
├── pkg/interfaces/  # Port Definitions
├── k8s/             # Production Orchestration (HPA, NP, PgBouncer)
└── migrations/      # SQL Schema (Audit-ready)
```
