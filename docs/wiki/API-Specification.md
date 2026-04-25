# API Specification

The OIDC Provider exposes standard OIDC endpoints, an Administrative API, and an Observability endpoint.

## 🔑 OIDC Endpoints

### 1. Authorization Endpoint
`GET /authorize`

| Parameter | Required | Description |
| :--- | :--- | :--- |
| `client_id` | Yes | Unique client identifier |
| `response_type` | Yes | Must be `code` |
| `redirect_uri` | Yes | Registered callback URL |
| `scope` | Yes | Must contain `openid` |
| `code_challenge` | **Yes** | **PKCE enforced** |
| `code_challenge_method` | Yes | Recommended `S256` |

### 2. Token Endpoint
`POST /token`

Supports `authorization_code` and `refresh_token` grants.
Returns a **Nested JWS** (PQC-Hybrid).

### 3. Discovery & Keys
- `GET /.well-known/openid-configuration`: Returns provider metadata.
- `GET /jwks`: Returns Dilithium3 and Ed25519 public keys.

---

## 🛠️ Administrative API
*Authentication: Requires `X-Admin-API-Key` header.*

### 1. Metrics & Stats
`GET /admin/api/stats`
Returns real-time performance data (TPS, Latency, Sessions).

### 2. Audit Logs
`GET /admin/api/audit?limit=50`
Fetches the latest forensic audit events from PostgreSQL.

### 3. Client Management
- `GET /admin/api/clients`: List all registered clients.
- `POST /admin/api/clients`: Register a new client.

### 4. Key Rotation
`POST /admin/api/rotate-keys`
Triggers an immediate rotation of the PQC (Dilithium3) and Classical (Ed25519) key pairs in the KMS.

---

## 📊 Observability
`GET /metrics`

Exposes standard Prometheus metrics. 

**Key Gauges:**
- `oidc_tps`: Current transactions per second.
- `oidc_latency_p99`: Current p99 latency in milliseconds.
- `oidc_active_refresh_tokens`: Number of valid refresh tokens in the database.
