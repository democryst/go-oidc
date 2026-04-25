# 🚀 Production Deployment Strategy: Post-Quantum OIDC Provider (1M TPS Target)

This plan outlines the high-level, resilient, and scalable architecture required to support 1 Million Transactions Per Second (TPS) for the Post-Quantum OIDC Provider. Given the extreme throughput requirement, the focus is on minimizing connection overhead, maximizing horizontal scaling, and ensuring zero single points of failure (SPOF).

---

## 🌐 1. Load Balancing Strategy: NLB vs. ALB
For a low-latency, high-throughput service, bypassing Layer 7 (L7) processing where possible is critical.

- **Recommendation:** **Use a Network Load Balancer (NLB)**. Raw TCP pass-through to the K8s ingress controller (Envoy or NGINX) minimizes latency.
- **TLS Termination:** Handled by the Ingress Controller within the cluster to allow for efficient L7 routing once inside the VPC.

## ⚙️ 2. Kubernetes Cluster Sizing
The cluster must be sized for peak capacity with a N+2 safety buffer.

- **Node Type:** High core count, high network bandwidth instances (e.g., AWS C6gn or equivalent).
- **Topology:** Baseline deployment of **10-15 worker nodes** across 3 Availability Zones (AZs).
- **Anti-Affinity:** Strict `podAntiAffinity` to ensure OIDC replicas, Redis nodes, and DB proxies are never co-located on the same physical host.

## 🛡️ 3. Component High Availability (HA)

### A. OIDC Provider Service
- **Statelessness:** The Golang service is entirely stateless, offloading sessions to Redis.
- **Replicas:** Baseline of 10 pods, scaling to 100 via HPA.

### B. Redis Cluster (Rate Limiting)
- **Deployment:** Clustered mode with **3 Primary + 3 Subordinate** nodes across AZs.
- **Tuning:** Utilize LUA scripts (as implemented) to ensure atomic checks with zero round-trip overhead.

### C. PostgreSQL & Connection Pooling
- **R/W Separation:** 1 Primary (Write) + 3 Read Replicas.
- **Connection Proxy:** **Mandatory PgBouncer** layer in `transaction` pooling mode. This prevents the "thundering herd" problem and connection exhaustion during 1M TPS bursts.

## 📈 4. Horizontal Pod Autoscaling (HPA)

Traditional CPU-only HPA is insufficient for this workload.

| Metric | Target Value | Rationale |
| :--- | :--- | :--- |
| **Custom: RPS** | 150k per Pod | **Primary Trigger.** Direct measure of load capacity. |
| **CPU Utilization** | 60% | Baseline safety net for compute-heavy signing logic. |
| **P95 Latency** | 1.0ms | Scaling trigger if I/O or crypto saturation occurs. |

## 📦 5. Infrastructure as Code (IaC)
- **Terraform/Pulumi:** Manage NLB, RDS (Postgres), and ElastiCache (Redis).
- **ArgoCD:** Maintain git-ops state for all Kubernetes manifests.

---

## 📝 Success Criteria Checklist
- [ ] NLB configured for L4 pass-through.
- [x] PgBouncer deployed in `transaction` mode (`k8s/pgbouncer.yaml`).
- [x] Prometheus metrics exported for `oidc_requests_total` and `signing_duration_seconds` (`internal/api/middleware/metrics.go`).
- [x] HPA configured with custom metrics API (`k8s/hpa.yaml`).
- [ ] OpenBao HA cluster initialized and unsealed.
