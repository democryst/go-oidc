# Stage 1: Build
FROM golang:1.26.2-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
# -ldflags="-s -w" to reduce binary size
# CGO_ENABLED=0 for static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o oidc-server ./cmd/server/main.go

# Stage 2: Production
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/oidc-server .

# Copy UI assets for the dashboard
COPY --from=builder /app/web ./web

# Use non-root user (provided by distroless)
USER 65532:65532

# Expose the OIDC port (default 8080)
EXPOSE 8080

# Run the server
ENTRYPOINT ["/app/oidc-server"]
