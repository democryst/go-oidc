# Phase 2 Requirements: High-Load Performance & Audit Readiness

## 1. Objective
To evolve the Post-Quantum OIDC Provider into a global-scale Identity Platform capable of sustaining **1 Million Transactions Per Second (TPS)** while maintaining "Audit-Ready" security compliance for enterprise-grade investigations.

---

## 2. Scalability & Performance (1M TPS Target)

### 2.1. Distributed Architecture
- **Horizontal Scaling**: All components (APIs, Signers) must be stateless and deployable in a Geographically Distributed Kubernetes cluster.
- **L4/L7 Load Balancing**: Intelligent traffic routing with Global Server Load Balancing (GSLB) and low-latency termination of PQC-TLS.

### 2.2. Distributed Caching & Rate Limiting
- **Redis/Dragonfly Integration**: Replace MemoryStore with a high-performance distributed key-value store.
- **Adaptive Rate Limiting**: Implement sliding-window rate limiting capable of handling burst traffic across 100+ nodes without consistency bottlenecks.

### 2.3. Database Scaling
- **PostgreSQL Sharding/Citus**: Move from a single instance to a sharded database architecture to handle 1M+ write-intensive auth/refresh token records.
- **Read Replicas**: Direct all OIDC Discovery and Client Metadata lookups to read-optimized replicas.

### 2.4. Cryptographic Offloading
- **HSM Integration**: Migrate from OpenBao software Transit to Hardware Security Modules (HSMs) with dedicated PQC acceleration if available, or clustered OpenBao nodes with Nitro Enclaves.
- **Pre-Computation**: Implement pre-computation of cryptographic artifacts where possible to reduce per-request compute jitter.

---

## 3. Security Audit & Forensic Readiness

### 3.1. Non-Repudiable Audit Trails
- **Write-Ahead Logging (WAL)**: All audit events must be piped to a high-durability message bus (e.g., Kafka or Redpanda).
- **Audit Immutability**: Integration with a write-once-read-many (WORM) storage or blockchain-based timestamping for audit logs to prevent forensic tampering.

### 3.2. Forensic Observability
- **Distributed Tracing**: Full OpenTelemetry integration to trace a single `request_id` across 1M TPS traffic flows.
- **Secret Redaction (Phase 2)**: Implement automated deep-packet inspection (DPI) in the egress layer to ensure zero leak of PQC private key artifacts or PII.

### 3.3. Compliance & Certification
- **OIDC Conformance**: Successful pass of the [OpenID Foundation Conformance Suites](https://openid.net/certification/).
- **Automated Scanning**: Integration of Snyk/Trivy for container scanning and Gosec for SAST in the CI/CD pipeline.

---

## 4. Technical Constraints
- **Latency Budget**: 99.9th percentile (p99) latency for the `/token` endpoint must remain below **10ms** (excluding PQC signing time).
- **Availability**: 99.999% Service Level Objective (SLO).
- **Strict PQC**: No fallback to classical-only modes in Phase 2; hybrid or PQC-exclusive only.

---

## 5. Success Criteria
1.  **Stress Test**: Simulated load of 1.2M TPS with < 1% error rate.
2.  **Audit Simulation**: Successful recovery of an "attacker journey" log trace within 5 minutes of a simulated breach.
3.  **HA Failover**: Zero-downtime during a regional database failover.
