# Security Governance & Compliance

Security is baked into every layer of the provider, from development to deployment.

## 🛡️ Automated Governance (The verification Gate)

Our SDLC enforces a "Never-Green-Without-Security" policy via the `make verify` and `make scan` targets.

### 1. Static Analysis (SAST)
We use `gosec` to scan for:
- Weak cryptography usage.
- Hardcoded secrets and credentials.
- Input validation failures.
- Improper SQL handling.

### 2. Dependency Analysis (SCA)
We use `govulncheck` to identify known vulnerabilities in the Go standard library and third-party modules. **All builds are currently on Go 1.26.2** to ensure zero known vulnerabilities in `crypto/tls` and `crypto/x509`.

### 3. Linting & Quality
We use `staticcheck` to find potential logic errors and performance anti-patterns that could lead to DoS or side-channel attacks.

---

## 🔒 Runtime Security

### Zero-Trust Orchestration
- **Distroless**: Final production images contain zero binaries other than the compiled Go server.
- **Network Policies**: Kubernetes egress is strictly limited to the DB, Valkey, and KMS.
- **mTLS**: Internal communication between pods is encrypted using mutual TLS.

### KMS Policy (OpenBao)
Private keys never touch the OIDC provider's application memory.
- **Signing**: Done via the `/sign` endpoint of the KMS.
- **Access Control**: The provider uses a scoped AppRole with `sign`-only permissions (no `export` or `read` on private keys).

---

## 🏛️ Security Compliance Standards

- **OAuth 2.1 Compatibility**: Enforcing PKCE and rejecting insecure redirect URIs.
- **NIST Post-Quantum**: Implementation of FIPS 203 (ML-KEM) and FIPS 204 (ML-DSA).
- **Zero-Trust (NIST SP 800-207)**: Per-request authentication and fine-grained network segmentation.
