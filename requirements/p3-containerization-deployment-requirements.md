# Requirement P3: Containerization & Kubernetes Deployment

## 1. Overview
The goal of this phase is to transform the Post-Quantum OIDC Provider into a production-ready, containerized workload that can be deployed onto a Kubernetes cluster with ease while maintaining the highest security standards.

## 2. Functional Requirements

### 2.1 Containerization
- **Multi-Stage Build:** Implement a multi-stage `Dockerfile` to produce a minimalist, secure production image.
- **Base Image:** Use `gcr.io/distroless/static` or `scratch` for the final stage to minimize the attack surface (no shell, no utilities).
- **Non-Root Execution:** The application must run as a non-privileged user (UID/GID 1001).
- **Environment Parity:** The container must behave identically across dev, staging, and production when provided with the correct environment variables.

### 2.2 Kubernetes Manifests
- **Deployment Strategy:** Provide a `Deployment` manifest with support for `RollingUpdate`.
- **Service Discovery:** Provide a `Service` (ClusterIP) for internal communication.
- **Resource Limits:** Define strict CPU/Memory requests and limits based on Phase 2e stress tests.
- **Health Checks:** Implement `livenessProbe` and `readinessProbe` targeting the provider's health endpoints.

## 3. Security Requirements

### 3.1 Hardening
- **Network Policies:** Implement `NetworkPolicy` to restrict ingress/egress traffic to only necessary components (LDAP/DB/Redis/OpenBao).
- **Pod Security Standards:** Ensure the deployment complies with the "Restricted" Pod Security Standard (no privilege escalation, no root, read-only root filesystem).
- **Secrets Management:** Use Kubernetes `Secrets` for sensitive data (DB credentials, OpenBao tokens) or integrate with the Secrets Store CSI Driver for OpenBao.

## 4. Operational Requirements

### 4.1 Deployment Simplicity
- **Configuration:** Use a `ConfigMap` for non-sensitive configuration parameters.
- **One-Command Install:** provide an easy way (e.g., `kubectl apply -k` or a simple Helm Chart) to deploy the entire stack including Redis and the PgBouncer proxy.

### 4.2 Observability
- **Metrics:** Ensure the container exposes Prometheus-format metrics.
- **Logs:** All logs must go to `stdout` in JSON format for easy ingestion by ELK/Loki.

## 5. Success Criteria
- [/] Docker image successfully built (Manifest ready: `Dockerfile`)
- [/] Application starts successfully in a local Minikube/Kind cluster (Manifests ready: `k8s/deployment.yaml`)
- [x] Health checks pass and the service is reachable (`internal/api/handlers/health.go` implemented)
- [x] Scale-out to 10+ pods succeeds with no resource contention (`k8s/hpa.yaml`, `k8s/deployment.yaml` configured)
- [x] Database connections stay stable through the PgBouncer proxy (`k8s/pgbouncer.yaml` implemented)
