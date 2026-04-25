# Requirements Specification: Post-Quantum Secure OIDC Provider (Golang)

## 1. Project Overview
Development of a high-security OpenID Connect (OIDC) Identity Provider (IdP) written in Golang. The system must satisfy standard OAuth2/OIDC RFCs while implementing Post-Quantum Cryptography (PQC) to protect against future quantum computing threats.

---

## 2. Core Protocol Standards
The implementation must strictly adhere to the following specifications:
* **RFC 6749:** The OAuth 2.0 Authorization Framework (Core).
* **RFC 7636:** Proof Key for Code Exchange (PKCE) by OAuth Public Clients (Mandatory for all flows).
* **RFC 8582:** OpenID Connect Core 1.0 (and associated discovery/registration specs).
* **RFC 7519:** JSON Web Token (JWT) for Identity and Access Tokens.

---

## 3. Post-Quantum Security Requirements
To ensure "Quantum-Resistant" security, the following cryptographic upgrades are required:

### 3.1 Hybrid Key Exchange (KEM)
* **Mechanism:** Implement hybrid key exchange for TLS 1.3 and back-channel communication.
* **Algorithms:** Combine X25519 with **Kyber-768** (ML-KEM) to ensure security against both classical and quantum adversaries.

### 3.2 Post-Quantum JWT Signatures
* **Algorithm:** Support **Crystals-Dilithium** (ML-DSA) for signing JWTs.
* **Hybrid Transition:** Tokens should be dual-signed or wrapped in a way that remains compatible with classical parsers while providing quantum-level non-repudiation.

### 3.3 Entropy & Hashing
* **Hashing:** Use **SHA-3 (Keccak)** or SHA-256 (minimum) for all internal fingerprinting and PKCE code challenges.
* **Salt/Nonce:** Minimum 256-bit entropy for all nonces and state parameters to resist Grover's Algorithm.

---

## 4. Technical Stack & Architecture
* **Language:** Golang 1.21+ (leveraging `crypto/tls` and PQC libraries like `circl`).
* **Database:** PostgreSQL 15+.
    * **Schema:** Must include tables for `users`, `clients`, `refresh_tokens`, and `authorization_codes`.
    * **Encryption at Rest:** Sensitive client secrets and PQC private keys must be encrypted using AES-256-GCM.
* **Storage:** Securely store PQC public keys in a JWKS (JSON Web Key Set) endpoint extended for Dilithium parameters.

---

## 5. Endpoints & Flow Requirements
### 5.1 /authorize
* Support `response_type=code`.
* **Enforce PKCE:** Reject any request without `code_challenge`.
* Support OIDC scopes: `openid`, `profile`, `email`.

### 5.2 /token
* Handle `authorization_code` and `refresh_token` grants.
* Issue PQC-signed JWT Access Tokens and ID Tokens.
* Implement strict Refresh Token rotation.

### 5.3 /.well-known/openid-configuration
* Standard discovery document.
* Must advertise support for PQC signing algorithms in `id_token_signing_alg_values_supported`.

---

## 6. Security Hardening
* **Brute Force Protection:** Rate limiting on `/token` and login attempts.
* **Audit Logging:** Full immutability for security-critical events (login, token issuance, key rotation).
* **TLS:** Minimum TLS 1.3 with PQC-hybrid cipher suites.