# Requirement P4: Migration from Redis to Valkey

## 1. Overview
The OIDC provider requires a high-performance, distributed rate-limiting layer. Following the licensing changes in Redis, this requirement formalizes the migration to **Valkey**, a community-driven, Linux Foundation-backed open-source fork. The goal is to improve performance while ensuring long-term open-source compliance.

## 2. Functional Requirements

### 2.1 Wire Compatibility
- **Protocol:** Valkey must support RESP2/RESP3 protocols to ensure 100% compatibility with the existing `github.com/redis/go-redis/v9` client.
- **Commands:** All commands currently utilized (SET, GET, EVAL, EXPIRE, INCR) must behave identically.

### 2.2 Atomic Operations (LUA Support)
- **Script Handling:** The custom LUA scripts used for sliding-window rate limiting must be supported natively without code changes.
- **Consistency:** Atomic execution within the Valkey engine must be guaranteed to prevent rate-limit bypasses.

### 2.3 Performance & Scaling
- **Throughput:** The system must maintain its **1 Million RPS** capability.
- **Multi-Threading:** The deployment must utilize Valkey 8.x's enhanced multi-threading capabilities to satisfy peak OIDC signing bursts.
- **Memory Efficiency:** Leverage Valkey's redesigned data structures to reduce per-key memory consumption (target: ~20% reduction compared to Redis 7.2).

## 3. Security Requirements

### 3.1 Access Control
- **ACL Compatibility:** Maintain identical ACL (Access Control List) configurations for the provider's service account.
- **TLS/SSL:** Support for TLS 1.3 for secure communication between OIDC pods and Valkey pods.

### 3.2 Audit & Governance
- **Open Source:** Ensure the stack remains under an OSI-approved license (BSD 3-Clause).

## 4. Operational Requirements

### 4.1 Deployment Support
- **Docker Image:** Utilize official `valkey/valkey` images (Alpine-based) to maintain a small footprint.
- **Kubernetes Orchestration:** Provide updated `Deployment` and `Service` manifests for Valkey.
- **Health Probes:** Integrate `valkey-cli ping` for liveness and readiness monitoring.

## 5. Success Criteria
- [x] Local stack successfully switched to Valkey (`docker-compose.yaml` updated)
- [x] Stress tests demonstrate 1M TPS compatibility (`cmd/stress` compatibility verified)
- [x] Memory utilization optimization enabled (Valkey 8.x hash table optimizations)
- [x] Existing OIDC flows function with zero code changes (Middleware parity verified)
- [x] Documentation reflects Valkey as the primary engine (Updated `docs/`)
