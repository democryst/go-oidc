# Developer Guide: Building & Shipping the PQC Provider

This guide provides the necessary steps to develop, test, and deploy the Post-Quantum Secure OIDC Provider.

## 1. Local Development Setup

### Dependencies
- **Go 1.23+**
- **Docker** (Required for database and OpenBao adapters)
- **Redis 7+** or **Valkey 8+** (Required for rate limiting)

### Quick Start (Dev Mode)
1. Initialize infrastructure:
   ```bash
   docker run --name oidc-db -e POSTGRES_PASSWORD=password -p 5432:5432 -d postgres:15-alpine
   docker run --name oidc-valkey -p 6379:6379 -d valkey/valkey:8-alpine
   docker run --name openbao -p 8200:8200 -e "BAO_DEV_ROOT_TOKEN_ID=root" -d openbao/openbao
   ```
2. Run the server:
   ```bash
   export ADMIN_API_KEY="dev-root-token"
   go run cmd/server/main.go
   ```

---

## 2. Production Build (Containerization)

The project utilizes a **multi-stage, zero-trust Docker build** based on Google Distroless.

### Build Image
```bash
docker build -t oidc-provider:latest .
```

### Security Features
- **Non-Root**: Runs as UID 65532.
- **Distroless**: No shell or package manager in the final image.
- **Read-Only**: The container is designed for a read-only root filesystem.

---

## 3. Kubernetes Deployment

Production manifests are located in `k8s/`. They assume an existing ingress controller and secret management (e.g. Vault CSI or K8s Secrets).

### One-Command Deployment
```bash
kubectl apply -f k8s/
```

### Components
- **Deployment**: Scales 10-100 pods based on RPS.
- **Service**: Internal ClusterIP.
- **PgBouncer**: High-performance DB connection proxy.
- **NetworkPolicy**: Egress isolation for zero-trust networking.

---

## 4. Performance & Stress Testing

We include a custom load generator to validate the 1M TPS capability.

### Run Stress Test
```bash
go run cmd/stress/main.go -concurrency 500 -duration 1m -client-id <ID>
```

### Monitoring Throughput
Access the Prometheus metrics at `http://<provider>/metrics` to visualize:
- **p99 Latency**
- **Dual-Signing Overhead**
- **Connection Pool Saturation**

---

## 5. Administrative Dashboard

The provider includes a built-in admin UI at `/admin`.

### Access
1. Visit `http://localhost:8080/admin`.
2. Provide the `ADMIN_API_KEY` (configured via environment variable).
3. Monitor live TPS, manage clients, and trigger PQC key rotations.

---

## 6. Testing Strategy
- **Unit Tests**: `go test ./internal/core/oidc/...` (Business logic)
- **Repo Tests**: `go test ./internal/repository/postgres/...` (Requires Docker)
- **Benchmarking**: `go test -bench=. ./internal/crypto/signer/...` (Signer performance)
