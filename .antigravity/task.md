# Task: Post-Quantum Secure OIDC Provider

# Phase 1: Foundational PQC Hardening [COMPLETED]
- [x] Dilithium3 + Ed25519 Dual-Signing (Option A)
- [x] OpenBao Transit Integration (Master Key Protection)
- [x] Atomic Token Rotation & Replay Protection
- [x] PKCE S256 Enforcement (Mandatory)
- [x] PostgreSQL Repository (pgx/v5)

# Phase 2: Performance & Audit Hardening (1M TPS) [COMPLETED]

## Phase 2a — Distributed Rate Limiting
- [x] `internal/api/middleware/redis_store.go` — Implement `RateLimitStore` with LUA script
- [x] Add `github.com/redis/go-redis/v9` to `go.mod`
- [x] Update `main.go` to support Redis configuration

## Phase 2b — Asynchronous Audit Pipeline
- [x] `internal/repository/postgres/batch_logger.go` — Implement asynchronous audit queue
- [x] Update `OIDCService` to use non-blocking audit logging
- [x] Verify audit log durability under high load

## Phase 2c — Performance Tuning
- [x] Profile PQC signers for overhead reduction
- [x] Implement `sync.Pool` for byte buffers and encoders
- [x] Optimize server-side JSON handling

## Phase 2e — Stress Testing [x]
- [x] `cmd/stress/main.go` — High-concurrency load generator
- [x] Measure p99 latency for Token endpoint with dual-signing (0.74ms/op overhead)
- [x] Report throughput bottleneck (CPU signing is the primary compute cost)

# Phase 3: Administrative UI [COMPLETED]

## Phase 3a — Foundation [x]
- [x] Minimalist Premium Design System (CSS)
- [x] Admin API endpoints with RBAC
- [x] Real-time audit log websocket or polling

## Phase 3b — Dashboard Implementation [x]
- [x] Analytics visualization (TPS/Latency)
- [x] Client CRUD
- [x] Key rotation trigger

# Phase 4: Containerization & Kubernetes [x]

## Phase 4a — Containerization [x]
- [x] `Dockerfile` — Multi-stage distroless build
- [x] Build & scan automation (local)
- [x] Health check endpoint implementation

## Phase 4b — Kubernetes Orchestration [x]
- [x] `k8s/deployment.yaml` — Hardened deployment with probes
- [x] `k8s/network-policy.yaml` — Egress isolation
- [x] `k8s/hpa.yaml` — RPS-based autoscaling manifest
- [x] `k8s/service.yaml` — ClusterIP and Ingress
