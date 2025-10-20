# ============================
# 1️⃣ Build stage
# ============================
FROM golang:1.24-bullseye AS builder

WORKDIR /go/src/github.com/stolostron/rbac-apiserver

# Copy source
COPY . .

RUN make build-bin

# ============================
# 2️⃣ Runtime stage
# ============================
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

# # Install minimal CA certificates for TLS verification
# RUN microdnf install -y ca-certificates && microdnf clean all

# Copy binary from builder
COPY --from=builder /go/src/github.com/stolostron/rbac-apiserver/bin/rbac-apiserver /usr/local/bin/rbac-apiserver

# # Create nonroot user
# RUN microdnf install -y shadow-utils && \
#     useradd -u 65532 nonroot && \
#     microdnf clean all

USER 65532:65532

# # Health check for API server
# HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
#     CMD ["/usr/local/bin/rbac-apiserver", "--health-check"] || exit 1

ENTRYPOINT ["/usr/local/bin/rbac-apiserver"]
