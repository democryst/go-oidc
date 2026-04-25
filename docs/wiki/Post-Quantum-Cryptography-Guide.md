# Post-Quantum Cryptography (PQC) Guide

This guide explains the selection and implementation of quantum-resistant algorithms used in this OIDC provider.

## 🛡️ The Quantum Threat

Classical algorithms like RSA and Elliptic Curve Cryptography (ECC) are vulnerable to **Shor's Algorithm** once a sufficiently large cryptographically relevant quantum computer (CRQC) is built. This provider implements **Hybrid Cryptography** to protect identities today against "Harvest Now, Decrypt Later" attacks.

## 🧪 Algorithm Selection

We adhere to the NIST Post-Quantum Cryptography Standardization project (FIPS 203, 204).

### 1. ML-DSA-65 (Crystals-Dilithium3)
- **Use Case**: Primary quantum-resistant signature for JWTs and Admin operations.
- **Why**: Offers a balanced performance/security profile. While signatures find themselves larger than Ed25519, the verification speed is extremely fast, supporting our 1M TPS goal.
- **Implementation**: Handled via the `github.com/cloudflare/circl` library with multi-stage verification.

### 2. Ed25519
- **Use Case**: Classical "inner" signature in the Hybrid JWS.
- **Why**: Provides backward compatibility with existing OIDC-compliant applications while ensuring deterministic, fast performance.

### 3. ML-KEM-768 (Crystals-Kyber)
- **Use Case**: Key encapsulation for back-channel communication and TLS termination.
- **Why**: Provides NIST Level 3 security (equivalent to AES-192 bit-security).

---

## 🏗️ Hybrid Signature Structure (Nested JWS)

To ensure interoperability, we use a "Double Wrap" strategy:

```json
{
  "alg": "ML-DSA-65",
  "payload": {
    "protected": "base64(ed25519_jws_header)",
    "payload": "base64(claims)",
    "signature": "base64(ed25519_sig)"
  },
  "signature": "base64(dilithium_sig)"
}
```

- **Step 1**: Sign claims with Ed25519.
- **Step 2**: Wrap the entire Ed25519 JWS as a payload in a Dilithium3 JWS.
- **Result**: Legacy clients can unwrap and verify the inner JWS; high-security clients verify the outer PQC layer first.

## 🏛️ NIST Compliance & Standards

- **FIPS 203**: Module-Lattice-Based Key-Encapsulation Mechanism (ML-KEM).
- **FIPS 204**: Module-Lattice-Based Digital Signature Standard (ML-DSA).
- **JWK Extension**: Our `jwks` endpoint extends the standard to support `kty: "ML-DSA"` and associated public key parameters.
