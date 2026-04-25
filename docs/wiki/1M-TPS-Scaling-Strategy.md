# 1M TPS Scaling Strategy

Achieving 1 Million Transactions Per Second (TPS) in an OIDC environment requires eliminating every possible source of lock contention and I/O wait.

## 🚀 The Three Pillars of Scale

### 1. Zero-Allocation Cryptography
PQC signing is computationally expensive. We minimize the impact by:
- **Reuse Buffer Pools**: Using `sync.Pool` for all cryptographic hashing contexts and signature buffers to zero out GC pauses.
- **Pre-hashing**: Utilizing SHA3-256 pre-computations for PKCE challenges to avoid redundant transforms.

### 2. Batch-Optimized Audit Persistence
Audit logging usually kills performance due to synchronous DB commits.
- **Protocol**: We bypass standard `INSERT` statements in favor of the PostgreSQL **COPY Protocol**.
- **Buffering**: Events are aggregated into a 100ms / 2,000-event buffer. Once a trigger is hit, a single `COPY` command flushes them to SSD storage in a single I/O burst.

### 3. Distributed Rate Limiting via Valkey
Rate limiting is the most common bottleneck in multi-pod OIDC deployments.
- **Atomic Lua**: We push the rate-limit logic into Valkey 8.x using Lua scripts. This ensures 100% consistency across 100+ pods without needing distributed locks (e.g., Redlock), which add significant latency.

---

## 📈 Cluster Geometry for 1M TPS

| Component | Instances | Configuration | Purpose |
| :--- | :--- | :--- | :--- |
| **OIDC Pods** | 100 | 2 Core / 4GB | Distributed signing & protocol logic |
| **Valkey** | 3-Node Cluster | 8 Core / 32GB | Global rate limiting & session caching |
| **Postgres** | 8 Shards | 16 Core / 64GB | Persistent audit & client metadata |
| **PgBouncer** | 10 Sidecars | Transaction Mode | Connection multiplexing |

## 🕹️ Load Balancing & Termination

- **PQC-TLS Termination**: Offloaded to specialized NGINX/Envoy ingress controllers supporting Hybrid Kyber-768.
- **Sticky Sessions**: Not required. The provider is 100% stateless (sessions are stored in Valkey).

## 🛡️ Forensic Observability at Scale

At 1M TPS, traditional logging generates terabytes of data. 
- **Sampling**: We log 100% of errors but only a configurable percentage of successful tokens (typically 1-5% for health baseline).
- **Request ID Propagation**: Every request is tagged with a UUID v7 (chronological) to allow forensic reconstruction without full-packet capture.
