FROM golang:1.24-bullseye AS builder

WORKDIR /workspace

# Copy auth-server code
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o auth-server .

# Final image
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

WORKDIR /

COPY --from=builder /workspace/auth-server .

USER 65532:65532

ENTRYPOINT ["/auth-server"]
