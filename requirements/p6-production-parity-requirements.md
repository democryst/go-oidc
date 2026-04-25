# Requirement P6: Production Parity & Forensic Observability

## 1. Overview
The OIDC provider's documentation and Wiki describe a production-hardened system using Valkey 8.x and live forensic observability. This phase aims to resolve gaps between the current implementation and those architectural claims, ensuring the codebase is 100% "Wiki-Accurate" and ready for 1M TPS operations.

## 2. Functional Requirements

### 2.1 Valkey Rebranding & Alignment
- **Code Refactoring:** Rename all internal references to `RedisStore`, `RedisConfig`, and `NewRedisClient` to their **Valkey** equivalents.
- **Protocol Verification:** Explicitly verify that the `github.com/redis/go-redis/v9` client is utilized in a way that leverages Valkey 8.x optimizations (multiprocessing, improved eviction).

### 2.2 JWKS Post-Quantum Extension
- **Dilithium3 JWK:** Update the `JWKS()` service method to include full Crystals-Dilithium3 public key parameters.
- **OIDF Compliance:** Ensure the JWKS adheres to emerging standards for Post-Quantum Key representation in OIDC discovery.

### 2.3 Live Admin Observability
- **Real-time Metrics:** Pivot `HandleStats` in the `AdminHandler` from mock data to live data sourced from:
  - **TPS:** Current request rates calculated via Valkey sliding windows or internal atomics.
  - **Latency:** Moving average of p99 signing latency.
  - **Sessions:** Actual count of active refresh tokens in the PostgreSQL/Valkey session store.

### 2.4 Forensic Traceability
- **Request ID Propagation:** Audit all logging points (SAST fix verification) to ensure the `X-Request-ID` is consistently propagated through the entire stack, including the `audit_log` table and middleware logging.

## 3. Technical Constraints
- No regression in p99 latency (< 1ms).
- Maintain 100% "Green" status on `make scan`.
- Zero change to the OIDC core protocol interfaces (backwards compatibility).

## 4. Success Criteria
- [ ] `make scan` pass with zero GoSecurity/StaticCheck issues.
- [ ] `/admin/api/stats` returns dynamic, real-time metrics.
- [ ] `/.well-known/jwks.json` includes both `EdDSA` and `Dilithium3` keys.
- [ ] No "Redis" string remains in the `internal/` codebase (replaced by `Valkey`).
- [ ] `audit_log` table successfully records the `request_id` for every OIDC flow event.
