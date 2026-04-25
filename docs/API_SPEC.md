# OIDC API Specification

This document details the OIDC/OAuth2 endpoints implemented by the provider and their specific requirements, particularly around Post-Quantum security.

## 1. Authorization Endpoint
**URL:** `/authorize`  
**Method:** `GET`

### Parameters
- `client_id` (Required): The UUID of the client.
- `response_type` (Required): Must be `code`.
- `redirect_uri` (Required): Must match one of the registered URIs for the client.
- `scope` (Required): Must contain `openid`.
- `state` (Recommended): Opaque value for CSRF protection.
- `code_challenge` (**Mandatory**): The SHA3-256 hash of the code verifier, Base64URL encoded.
- `code_challenge_method` (**Mandatory**): Must be `S256`.

### Security Nuance
Unlike standard OIDC which might allow `plain` PKCE or no PKCE for certain clients, **this provider rejects any request without `S256` PKCE**.

---

## 2. Token Endpoint
**URL:** `/token`  
**Method:** `POST`  
**Content-Type:** `application/x-www-form-urlencoded`

### Grant Type: `authorization_code`
- `grant_type`: `authorization_code`
- `code`: The raw authorization code.
- `redirect_uri`: Must match the URI used in the authorize request.
- `code_verifier`: The raw PKCE verifier string.

### Grant Type: `refresh_token`
- `grant_type`: `refresh_token`
- `refresh_token`: The opaque rotating refresh token.

### Authentication
The endpoint supports:
1.  **Client Secret Post**: `client_id` and `client_secret` in the body.
2.  **Basic Auth**: `Authorization: Basic <base64(id:secret)>`.

---

## 3. Discovery Endpoint
**URL:** `/.well-known/openid-configuration`  
**Method:** `GET`

Returns the standard OIDC discovery document. Note the `id_token_signing_alg_values_supported` field which includes `Dilithium3` in the outer JWS layer.

---

## 4. JWKS Endpoint
**URL:** `/.well-known/jwks.json`  
**Method:** `GET`

Returns the public keys for both the classical (Ed25519) and Post-Quantum (Dilithium3) signers.

### JWK Entry for Dilithium3 (Experimental)
Dilithium3 keys are exported using a custom `kty` or extended `alg` identifier depending on the validator's capabilities.
- `kty`: `PQC` (Proposed)
- `alg`: `Dilithium3`
- `use`: `sig`
- `pub`: Standard binary representation in Base64URL.

---

## 5. Token Formats

### ID Token / Access Token
Tokens are returned in a **Nested JWS** string format:
`BASE64(Dilithium-Header) . BASE64(Ed25519-Signed-JWT) . BASE64(Dilithium-Signature)`

Validators should:
1.  Verify the outer Dilithium signature using the `JWKS` Dilithium public key.
2.  Unwrap the payload (which is the inner JWT).
3.  Verify the inner Ed25519 signature using the `JWKS` Ed25519 public key.
4.  Process standard OIDC claims (`iss`, `sub`, `aud`, etc).
