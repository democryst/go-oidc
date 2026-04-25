# Administrative Dashboard & Monitoring

The provider includes a built-in, secure administrative suite for real-time observability.

## 📊 Live Metrics

Accessing the dashboard at `/admin` (secured by `ADMIN_API_KEY`) allows you to monitor:
- **Throughput (TPS)**: Live calculation of current request volume.
- **p99 Latency**: Critical for monitoring the impact of PQC signature overhead.
- **Success Rate**: Real-time error monitoring.
- **Active Sessions**: Total count of active refresh tokens in the system.

## 🛠️ Management Operations

### 1. Client Lifecycle
- Register new OIDC clients with specific `redirect_uris`.
- List active clients and their associated metadata.

### 2. PQC Key Rotation
To maintain "Forward Secrecy" in a post-quantum world, we provide a one-click **PQC Key Rotation**:
- Generates new Dilithium3/Ed25519 pairs in OpenBao.
- Transparently updates the JWKS endpoint.
- Old keys are kept for the duration of the token TTL to prevent user logout.

### 3. Forensic Audit Search
Search the PostgreSQL audit trail for specific:
- `request_id`
- `client_id`
- Event types (e.g., `TOKEN_REFRESHED`, `AUTHORIZE_INIT`).

## 📈 Prometheus Exporter
Standard metrics are exposed at `/metrics` for ingestion by Prometheus/Grafana:
- `http_requests_total`
- `signing_latency_seconds`
- `db_connection_pool_active`
