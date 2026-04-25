# Requirement P7: Migration to Native Valkey Client

## 1. Overview
The OIDC provider currently uses the `go-redis` client to communicate with the Valkey infrastructure. To achieve true production-grade alignment and leverage Valkey-specific optimizations, this requirement formalizes the migration to the native **valkey-go** client (`github.com/valkey-io/valkey-go`). This transition will improve performance for high-concurrency rate-limiting operations and remove legacy Redis branding from the dependency tree.

## 2. Functional Requirements

### 2.1 Native Valkey Protocol Support
- **Client Integration:** Replace the `go-redis` client with `valkey-go`.
- **Command Efficiency:** Utilize `valkey-go`'s high-performance command construction and pipelining (if needed).
- **Lua Scripting:** Migrate the sliding-window rate-limiting Lua script to the `valkey.NewLuaScript` pattern for optimal execution (automatic EVALSHA/EVAL fallback).

### 2.2 Configuration Alignment
- **Initialization:** Update the server configuration to support `InitAddress` (single or cluster) as required by the `valkey-go` client.
- **Failover:** Ensure the new client is configured for robust failover and connection pooling consistent with the 1M TPS performance goals.

## 3. Success Criteria
- [ ] `go-redis` dependency is removed from `go.mod`.
- [ ] `valkey-go` is successfully integrated into the middleware layer.
- [ ] Rate limiting remains 100% functional with zero code changes to the OIDC core logic.
- [ ] `make verify` passes with the new client.
- [ ] Local stack (`make up`) operates correctly with the native Valkey client.
