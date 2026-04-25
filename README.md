# 🛡️ Post-Quantum Secure OIDC Provider (1M TPS)

A high-performance, production-grade OpenID Connect Identity Provider implemented in Go, featuring hybrid Post-Quantum Cryptography (PQC) and zero-trust orchestration.

## 🚀 Key Features
- **Hybrid PQC:** Nested JWS (Ed25519 inner, Crystals-Dilithium3 outer) for backward compatibility and future-proof security.
- **Extreme Scale:** Architected for **1 Million Transactions Per Second (TPS)** using:
  - Asynchronous Audit Batching (PostgreSQL `COPY` protocol).
  - Distributed Rate Limiting (Valkey + Lua).
  - High-Efficiency Memory Management (`sync.Pool`).
- **KMS Integration:** Native support for **OpenBao Transit** (and HashiCorp Vault).
- **Zero-Trust Deployment:** OCI-compliant distroless images with Kubernetes NetworkPolicies and HPA.
- **Observability:** Built-in **Administrative Dashboard** + Prometheus metrics exporter.

## 📚 Documentation
- [**ARCHITECTURE.md**](docs/ARCHITECTURE.md): Performance engineering & PQC design.
- [**DEVELOPER_GUIDE.md**](docs/DEVELOPER_GUIDE.md): Setup, build, and contribution.
- [**API_SPEC.md**](docs/API_SPEC.md): OIDC, Admin, and Metrics endpoint specifications.
- [**DEPLOYMENT_PLAN.md**](.antigravity/deployment_plan.md): 1M TPS scale-out strategy.

## 🛠️ Getting Started (Local)

### Prerequisites
- **Docker** and **Docker Compose**
- **Go 1.26.2+** (for local development)

### One-Command Start
Run the full stack (Provider, DB, Valkey, OpenBao) locally:
```bash
make up
```
*The provider will be available at `http://localhost:8080`.*

### Admin Dashboard
Access the secure monitoring suite at `http://localhost:8080/admin` using the default key: `dev-root-token`.

## 📜 Makefile Commands
| Command | Description |
| :--- | :--- |
| `make up` | Start the local stack via Docker Compose |
| `make down` | Stop the local stack |
| `make build` | Build the OIDC provider binary |
| `make test` | Run all unit tests |
| `make docker-build` | Build the hardened production image |
| `make verify` | Run build and tests to verify project state |

## 📊 Performance Benchmarks
Validated at **1M TPS** scale with **0.74ms/op** dual-signature overhead. For detailed stress-test metrics, refer to the [**walkthrough.md**](.antigravity/walkthrough.md).

## 🛡️ License
BSD-3-Clause
