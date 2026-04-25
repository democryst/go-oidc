# OIDC API Specification

This document details the OIDC/OAuth2 and Administrative endpoints implemented by the provider, including Post-Quantum security requirements and performance monitoring.

## 1. OIDC Endpoints

### 1.1 Authorization Endpoint
**URL:** `/authorize` | **Method:** `GET`
- **PKCE Mandatory**: Use of `code_challenge` (S256) and `code_challenge_method` (S256) is mandatory. Requests without PKCE are rejected.

### 1.2 Token Endpoint
**URL:** `/token` | **Method:** `POST`
- **Grants**: `authorization_code`, `refresh_token`.
- **Response**: Returns a **Nested JWS** (Dilithium3 outer, Ed25519 inner).

### 1.3 Discovery & JWKS
- **Discovery**: `/.well-known/openid-configuration`
- **JWKS**: `/.well-known/jwks.json` (Includes both Ed25519 and Dilithium3 public keys).

---

## 2. Administrative API (RBAC Protected)
All admin endpoints require an `Authorization: Bearer <ADMIN_API_KEY>` header.

### 2.1 System Statistics
**URL:** `/admin/stats` | **Method:** `GET`
Returns real-time performance metrics (TPS, Latency, Success Rate, Active Sessions).

### 2.2 Client Management
- **List Clients**: `GET /admin/clients`
- **Register Client**: `POST /admin/clients/create`
  - Body: `{"name": "App Name", "redirect_uris": ["https://..."]}`

### 2.3 Forensic Audit Logs
**URL:** `/admin/audit` | **Method:** `GET`
Returns the recent audit events from the asynchronous PostgreSQL repository.

### 2.4 PQC Governance
**URL:** `/admin/rotate-keys` | **Method:** `POST`
Triggers the rotation of the Dilithium3 signing keypair. Existing sessions remain valid for their inner classical layer, but new tokens will use the rotated PQC key.

---

## 3. Operational Endpoints

### 3.1 Health Probes (Kubernetes Native)
- **Liveness**: `/live` (200 OK if process is running).
- **Readiness**: `/ready` (200 OK only if DB, Redis, and OpenBao connections are healthy).

### 3.2 Observability Exporter
**URL:** `/metrics` | **Method:** `GET`
Exposes **Prometheus-formatted** metrics:
- `oidc_requests_total`: Throughput counter by path/method/status.
- `signing_duration_seconds`: Histogram of cryptographic operations (Ed25519 vs Dilithium3).
- `http_request_duration_seconds`: Request latency distribution.

---

## 4. Common Headers

### X-Request-ID
Clients or Load Balancers are encouraged to provide an `X-Request-ID`. If missing, the provider generates a UUID. This ID is logged across the system to enable forensic tracing of high-load transactions.

### Rate Limiting
Requests are subject to global rate limiting. If exceeded, the server returns `429 Too Many Requests` with a `Retry-After` header.
