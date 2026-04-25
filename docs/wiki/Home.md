# Welcome to the Post-Quantum Secure OIDC Wiki

Welcome to the official documentation for the **PQC OIDC Provider**, a high-performance Identity Platform architected for the next generation of security threats.

## 🚀 Overview

This project provides a production-ready OpenID Connect Identity Provider (IdP) built from the ground up for:
- **Quantum Resistance**: Protecting identity tokens against future Shor's algorithm threats using Hybrid PQC signatures.
- **Extreme Scale**: Sustaining 1 Million Transactions Per Second (TPS) with sub-millisecond overhead.
- **Zero-Trust**: Hardened containerization and observability for mission-critical deployments.

---

## 📖 Navigation

### Core Concepts
- [[Architecture Deep Dive]]: System design, Hybrid PQC, and component boundaries.
- [[Post-Quantum Cryptography Guide]]: Detailed look at Dilithium3, Ed25519, and NIST compliance.

### Engineering for Scale
- [[1M TPS Scaling Strategy]]: How we achieve extreme throughput with PostgreSQL and Valkey.
- [[Performance Benchmarks]]: Real-world stress test results and p99 latency data.

### Operations & Deployment
- [[Deployment Hardening]]: Kubernetes orchestration, Distroless images, and NetworkPolicies.
- [[Administrative Dashboard]]: User guide for the live monitoring suite.

### Development
- [[API Specification]]: Complete OIDC, Admin, and Metrics endpoint reference.
- [[Security Governance]]: SAST, SCA, and quality gates for developers.

---

## 🛡️ Quick Start

To run the full stack locally:
```bash
make up
```
The provider will be available at `http://localhost:8080`.
The Admin Dashboard is at `http://localhost:8080/admin`.
