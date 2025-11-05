# RBAC API Server

A Kubernetes aggregated API server for custom RBAC resources, built on the Kubernetes API aggregation layer.

## Overview

This project implements a standalone API server that extends the Kubernetes API with custom RBAC resources. It uses Kubernetes API aggregation to seamlessly integrate with existing Kubernetes clusters, providing a foundation for building custom RBAC solutions.

## Features

- **PermissionBinding API** (`authorization.open-cluster-management.io/v1alpha1`) - Multi-cluster RBAC permission bindings backed by embedded SpiceDB
- **PermissionReview API** (`authorization.open-cluster-management.io/v1alpha1`) - Permission check requests evaluated immediately (CSR-like pattern)
- **Embedded SpiceDB** for real-time authorization evaluation
- **OpenAPI schema generation** for automatic API documentation
- **Helm chart deployment** with flexible TLS configuration
- **Optional cert-manager integration** for automatic certificate management
- **Comprehensive e2e testing** using Ginkgo/Gomega BDD framework
- **CI/CD pipeline** with GitHub Actions

## Architecture

```text
┌──────────────────────────────────────────────────────────────────────────────────┐
│                         Kubernetes Cluster                                       │
│                                                                                  │
│   ┌──────────────────┐                  ┌────────────────────────────────────┐   │
│   │  kube-apiserver  │─────────────────▶│  rbac-apiserver (aggregated)       │   │
│   │                  │                  │                                    │   │
│   │  APIServices:    │                  │  Authorization API:                │   │
│   │  - v1alpha1.     │                  │    authorization.open-cluster-     │   │
│   │    authorization │                  │    management.io/v1alpha1          │   │
│   │    .open-cluster │                  │                                    │   │
│   │    -management   │                  │  Resources:                        │   │
│   │    .io           │                  │  - PermissionBindings              │   │
│   │                  │                  │  - PermissionReviews (CSR-like)    │   │
│   │                  │                  │                                    │   │
│   │                  │                  │  Backend: Embedded SpiceDB         │   │
│   │                  │                  │                                    │   │
│   └──────────────────┘                  └────────────────────────────────────┘   │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Go 1.24+
- Kubernetes 1.24+
- Docker or Podman
- Helm 3.0+
- (Optional) cert-manager for TLS certificates

## Quick Start

### 1. Build the Binary

```bash
make build
```

### 2. Build the Docker Image

```bash
make build-image
```

### 3. Deploy with Helm

#### Option A: With cert-manager (Development/E2E)

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml

# Install rbac-apiserver
helm install rbac-apiserver charts/rbac-apiserver/ \
  -n rbac-apiserver-system --create-namespace \
  --set tls.certManager.enabled=true \
  --wait
```

#### Option B: With Custom Certificates (Production)

```bash
# Create TLS secret
kubectl create secret tls rbac-apiserver-cert \
  --cert=path/to/tls.crt \
  --key=path/to/tls.key \
  -n rbac-apiserver-system

# Install rbac-apiserver
helm install rbac-apiserver charts/rbac-apiserver/ \
  -n rbac-apiserver-system --create-namespace \
  --set tls.secretName=rbac-apiserver-cert \
  --wait
```

### 4. Verify Installation

```bash
# Check APIService status
kubectl get apiservices v1alpha1.authorization.open-cluster-management.io

# Create a test PermissionBinding
kubectl apply -f - <<EOF
apiVersion: authorization.open-cluster-management.io/v1alpha1
kind: PermissionBinding
metadata:
  name: test-binding
spec:
  subject:
    kind: User
    name: alice
  permissions:
    - role: viewer
      resources:
        - pods
      groups:
        - ""
      namespaces:
        - default
      names:
        - "*"
      clusters:
        - cluster1
EOF

# List PermissionBindings
kubectl get permissionbindings

# Check permissions with PermissionReview
kubectl create -f - <<EOF
apiVersion: authorization.open-cluster-management.io/v1alpha1
kind: PermissionReview
metadata:
  name: check-alice-access
spec:
  group: ""
  resource: pods
  verb: get
  cluster: cluster1
  namespace: default
  name: my-pod
EOF
```

## Development

### Project Structure

```text
.
├── apis/                       # API definitions
│   ├── generated/             # Generated client, informers, listers, and OpenAPI specs
│   └── rbac/                  # Authorization API (authorization.open-cluster-management.io)
│       └── v1alpha1/         # v1alpha1 API version (PermissionBinding, PermissionReview)
├── charts/                    # Helm charts
│   └── rbac-apiserver/       # Main Helm chart
├── cmd/                       # Main application entry point
├── pkg/                       # Core packages
│   ├── registry/             # REST storage implementation
│   ├── spicedb/              # Embedded SpiceDB integration
│   └── storage/              # Storage backend
├── test/                      # Test suites
│   └── e2e/                  # End-to-end tests
├── docs/                      # Documentation
├── examples/                  # Example YAML files
└── hack/                      # Development scripts
```

### Available Make Targets

```bash
make help                   # Show all available targets
make build                  # Build the binary
make build-image           # Build Docker image
make test-unit             # Run unit tests
make build-e2e             # Build e2e test binary
make run-e2e               # Run e2e tests
make test-e2e              # Build and run e2e tests (build-e2e + run-e2e)
make lint                  # Run linter
make fmt                   # Format code
make generate              # Generate OpenAPI specs
```

### Running Tests

#### Unit Tests

```bash
make test-unit
```

#### E2E Tests

The e2e tests use the Ginkgo BDD framework and require a running Kubernetes cluster.

```bash
# Build the e2e test binary
make build-e2e

# Run the tests
make run-e2e

# Or combine both steps
make test-e2e
```

**Note**: The e2e tests expect the rbac-apiserver to be deployed and accessible via kubeconfig.

### Local Development with Kind

```bash
# Create a Kind cluster
kind create cluster --name rbac-dev

# Build and load the image
make build-image
kind load docker-image quay.io/stolostron/rbac-apiserver:latest --name rbac-dev

# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml
kubectl wait --for=condition=Available --timeout=300s -n cert-manager deployment/cert-manager
kubectl wait --for=condition=Available --timeout=300s -n cert-manager deployment/cert-manager-webhook
kubectl wait --for=condition=Available --timeout=300s -n cert-manager deployment/cert-manager-cainjector

# Deploy rbac-apiserver
helm install rbac-apiserver charts/rbac-apiserver/ \
  -n rbac-apiserver-system --create-namespace \
  --set image.tag=latest \
  --set tls.certManager.enabled=true \
  --wait

# Run e2e tests
make test-e2e
```

## Configuration

### Helm Chart Values

Key configuration options in `charts/rbac-apiserver/values.yaml`:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `quay.io/stolostron/rbac-apiserver` |
| `image.tag` | Image tag | `latest` |
| `service.port` | Service port | `443` |
| `service.targetPort` | Container port | `6443` |
| `tls.certManager.enabled` | Use cert-manager for TLS | `false` |
| `tls.secretName` | TLS secret name | `""` |
| `resources.limits.cpu` | CPU limit | `100m` |
| `resources.limits.memory` | Memory limit | `128Mi` |

See [charts/rbac-apiserver/README.md](charts/rbac-apiserver/README.md) for complete documentation.

## API Reference

### PermissionBinding Resource

The PermissionBinding API defines what permissions a user or group has on resources across clusters. These bindings are automatically synced to the embedded SpiceDB for authorization evaluation.

```yaml
apiVersion: authorization.open-cluster-management.io/v1alpha1
kind: PermissionBinding
metadata:
  name: alice-pod-viewer
spec:
  subject:
    kind: User
    name: alice
  permissions:
    - role: viewer          # admin, editor, or viewer
      resources:
        - pods
      groups:
        - ""              # "" for core API group
      namespaces:
        - default
      names:
        - "*"             # wildcard for all resources
      clusters:
        - cluster1
```

### PermissionReview Resource (CSR-like)

The PermissionReview API checks whether a user has permission to perform an action on a resource. Like Kubernetes CertificateSigningRequests, PermissionReviews are evaluated immediately and not persisted.

```yaml
apiVersion: authorization.open-cluster-management.io/v1alpha1
kind: PermissionReview
metadata:
  name: check-alice-access
spec:
  group: ""
  resource: pods
  verb: get
  cluster: cluster1
  namespace: default
  name: my-pod
```

For detailed API documentation, see [docs/authorization-api.md](docs/authorization-api.md).

For detailed usage examples, see [EXAMPLES.md](EXAMPLES.md).

### Supported Operations

**PermissionBinding**:

- **Create**: `kubectl apply -f binding.yaml`
- **Get**: `kubectl get permissionbinding alice-pod-viewer`
- **List**: `kubectl get permissionbindings`
- **Update**: `kubectl edit permissionbinding alice-pod-viewer`
- **Delete**: `kubectl delete permissionbinding alice-pod-viewer`

**PermissionReview** (CSR-like pattern):

- **Create**: `kubectl create -f review.yaml --as=alice -o yaml` (evaluated immediately, not persisted)
- **Note**: Get and List operations are not supported for PermissionReviews

## CI/CD

The project uses GitHub Actions for continuous integration:

- **Pre-submit checks**: Build, lint, unit tests, e2e tests
- **E2E testing**: Automated testing in Kind cluster with cert-manager
- **Image building**: Multi-stage Docker builds

See [.github/workflows/go-presubmit.yml](.github/workflows/go-presubmit.yml) for pipeline configuration.

## Security

### TLS Configuration

The API server requires TLS certificates for secure communication:

- **Development/Testing**: Use cert-manager with self-signed certificates
- **Production**: Provide your own CA-signed certificates via Kubernetes secrets

### RBAC Permissions

The API server requires specific RBAC permissions:

- Authentication delegation (`system:auth-delegator`)
- Authorization delegation (SubjectAccessReview creation)
- Extension API server authentication reader

These are automatically configured by the Helm chart.

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass (`make test-unit test-e2e`)
2. Code is properly formatted (`make fmt`)
3. Linter passes (`make lint`)
4. Commits are signed off (`git commit --signoff`)

## Troubleshooting

### APIService shows "FailedDiscoveryCheck"

```bash
# Check APIService status
kubectl get apiservices v1alpha1.authorization.open-cluster-management.io -o yaml

# Check API server pods
kubectl get pods -n rbac-apiserver-system

# Check API server logs
kubectl logs -n rbac-apiserver-system deployment/rbac-apiserver
```

### TLS Certificate Issues

```bash
# Check certificate status (if using cert-manager)
kubectl get certificate -n rbac-apiserver-system

# Check certificate secret
kubectl get secret -n rbac-apiserver-system rbac-apiserver-serving-cert -o yaml
```

### E2E Tests Failing

```bash
# Ensure cluster is accessible
kubectl cluster-info

# Verify APIServices are available
kubectl get apiservices | grep open-cluster-management.io

# Test the API with a simple PermissionBinding
kubectl apply -f examples/01-user-viewer-binding.yaml

# Check e2e test logs with verbose output
./_output/test/e2e.test -ginkgo.v -ginkgo.fail-fast
```

### PermissionReviews Not Working

```bash
# Ensure you have RBAC permissions to create PermissionReviews
kubectl apply -f examples/01-permissionreviews-assign.yaml

# Use kubectl create (not apply) with --as flag
kubectl create -f examples/01-check-user-view-access.yaml --as=alice -o yaml

# Check SpiceDB logs in the API server
kubectl logs -n rbac-apiserver-system deployment/rbac-apiserver | grep -i spicedb
```

## License

This project follows the licensing of the parent organization.

## References

- [Kubernetes API Aggregation](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/)
- [Ginkgo BDD Testing Framework](https://onsi.github.io/ginkgo/)
- [cert-manager Documentation](https://cert-manager.io/docs/)
- [OCM Project](https://open-cluster-management.io/)
