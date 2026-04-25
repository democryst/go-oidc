# Deployment Hardening & Orchestration

The provider is containerized and orchestrated for high-availability and zero-trust security.

## 📦 Hardened Containers

Unlike standard containers, we use a **Multi-Stage Distroless** build:
- **Build Stage**: Compiles the static Go binary using `golang:1.26.2-alpine`.
- **Final Stage**: `gcr.io/distroless/static:nonroot`.
- **Result**: No shell, no package manager, no SSH, and no root user. An attacker who gains a foothold has no tools to escalate or move laterally.

## ☸️ Kubernetes Resilience

Our standard deployment in `k8s/` includes:

### Horizontal Pod Autoscaling (HPA)
Targets:
- **CPU**: 60%
- **RPS**: 10,000 per pod
The system can scale from 10 to 100+ pods in seconds to handle traffic surges.

### Network Isolation
We enforce strict `NetworkPolicy` rules:
- **Ingress**: Only from the Ingress Controller on port 8080.
- **Egress**: Restricted to PostgreSQL, Valkey, and OpenBao IPs. External internet access is blocked.

### Pod Security standards
The deployment complies with the **Kubernetes Restricted Profile**:
- `allowPrivilegeEscalation: false`
- `readOnlyRootFilesystem: true`
- `runAsNonRoot: true`

## 🌉 Connection Pooling (PgBouncer)
At 1M TPS, managing 100,000+ persistent database connections is impossible. We use **PgBouncer in Transaction Mode** to multiplex thousands of pod connections onto a small footprint of database backends.
