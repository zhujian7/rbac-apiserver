# Auth-Server Project Summary

## What Was Created

A complete authorization webhook addon for OCM (Open Cluster Management) multi-cluster RBAC integration with the rbac-apiserver on the hub cluster.

### Project Structure

```
auth-server/
├── main.go                      # Entry point
├── server.go                    # Authorization webhook server implementation
├── go.mod                       # Go module with dependencies
├── Dockerfile                   # Container image build
├── Makefile                     # Build automation
├── README.md                    # Comprehensive documentation
├── QUICKSTART.md               # Quick start guide
├── PROJECT_SUMMARY.md          # This file
├── .gitignore                  # Git ignore rules
├── poc-setup.sh                # Automated POC setup script
├── manifests/                  # Kubernetes manifests
│   ├── namespace.yaml
│   ├── serviceaccount.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   └── webhook-config.yaml
└── examples/                   # Example PermissionBindings and test script
    ├── permissionbinding-alice.yaml
    ├── permissionbinding-bob.yaml
    └── test-authorization.sh
```

## Key Components

### 1. Authorization Webhook Server (`server.go`)

- Receives `SubjectAccessReview` requests from managed cluster kube-apiserver
- Transforms them to `PermissionRequest` for hub rbac-apiserver
- Parses the response to determine allow/deny
- Returns authorization decision to managed cluster

**Key Features:**
- HTTP/HTTPS server on port 8443
- Health check endpoint at `/healthz`
- Authorization endpoint at `/authorize`
- Automatic cleanup of PermissionRequest objects
- Configurable timeouts and logging

### 2. POC Setup Script (`poc-setup.sh`)

Automated script that:
- Creates two Kind clusters (hub and managed)
- Deploys rbac-apiserver on hub with cert-manager
- Builds and deploys auth-server on managed cluster
- Configures authorization webhook
- Sets up all necessary secrets and certificates

### 3. Kubernetes Manifests

Production-ready manifests including:
- Deployment with 2 replicas for HA
- Service for cluster-local access
- ServiceAccount for pod identity
- Webhook configuration for kube-apiserver

### 4. Examples and Testing

- Sample PermissionBindings for users alice and bob
- Automated test script for verification
- Quick start guide for immediate testing

## How It Works

```
┌─────────────────────────────────────────────────────────────────────┐
│                            Hub Cluster                              │
│                                                                     │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │  rbac-apiserver                                            │    │
│  │  - PermissionBinding API (stores permissions)              │    │
│  │  - PermissionRequest API (evaluates permissions)           │    │
│  │  - SpiceDB backend (relationship store)                    │    │
│  └────────────────────────────────────────────────────────────┘    │
│                            ▲                                        │
│                            │ HTTPS (PermissionRequest API)          │
└────────────────────────────┼───────────────────────────────────────┘
                             │
                             │
┌────────────────────────────┼───────────────────────────────────────┐
│                   Managed Cluster                                  │
│                            │                                        │
│  ┌──────────────────┐      │      ┌─────────────────────────────┐  │
│  │  kube-apiserver  │      │      │  auth-server (this addon)   │  │
│  │                  │──────┼─────▶│                             │  │
│  │  Authorization   │      │      │  1. Receive webhook         │  │
│  │  Webhook         │      │      │  2. Call hub API            │  │
│  │                  │◀─────┼──────│  3. Return decision         │  │
│  └──────────────────┘      │      └─────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────┘
```

### Authorization Flow

1. User makes request to managed cluster (e.g., `kubectl get pods --as=alice`)
2. Managed cluster kube-apiserver calls authorization webhook
3. Auth-server receives `SubjectAccessReview`
4. Auth-server creates `PermissionRequest` on hub rbac-apiserver
5. Hub rbac-apiserver evaluates against SpiceDB
6. Hub returns status with allowed resources
7. Auth-server parses response and determines allow/deny
8. Auth-server responds to webhook
9. Managed cluster enforces the decision
10. Auth-server cleans up PermissionRequest object

## Quick Start

```bash
# 1. Run POC setup
cd auth-server
./poc-setup.sh

# 2. Restart managed cluster apiserver (required for webhook)
docker exec managed-control-plane crictl stopp \
  $(docker exec managed-control-plane crictl pods --name kube-apiserver -q)
sleep 30

# 3. Create PermissionBindings on hub
kubectl config use-context kind-hub
kubectl apply -f examples/permissionbinding-alice.yaml

# 4. Test authorization on managed cluster
kubectl config use-context kind-managed
kubectl auth can-i get pods --as=alice -n default

# 5. Watch logs
kubectl logs -n auth-server-system deployment/auth-server -f
```

## Current Limitations

### 1. User Context Hardcoded in Hub API

**Issue**: The rbac-apiserver currently hardcodes the user to `system:admin` when processing PermissionRequests (see `pkg/registry/permissionrequest_rest.go:75`).

**Impact**: All authorization checks evaluate against `system:admin` permissions, not the actual requesting user.

**Workaround**: None currently. This needs to be fixed in rbac-apiserver by:
- Adding `user` field to `PermissionRequestSpec`
- Extracting user from request context
- Passing user to SpiceDB integration

**Future Fix**: Update rbac-apiserver to accept user context (documented in parent README).

### 2. Performance

**Issue**: Each authorization check creates and deletes a PermissionRequest object.

**Impact**: Higher latency and API server load for high-frequency authorization checks.

**Future Optimization**:
- Implement caching in auth-server
- Add SubjectAccessReview API to rbac-apiserver (no object creation)
- Use long-lived connections/streaming

### 3. HA and Reliability

**Current State**: Deployment runs 2 replicas but full HA testing not performed.

**Considerations**:
- Network partitions between managed cluster and hub
- Hub API server downtime
- Certificate rotation
- Graceful degradation strategies

## Testing

### Automated Testing

```bash
./examples/test-authorization.sh
```

### Manual Testing

```bash
# Create PermissionBinding on hub
kubectl config use-context kind-hub
kubectl apply -f examples/permissionbinding-alice.yaml

# Test on managed cluster
kubectl config use-context kind-managed
kubectl auth can-i get pods --as=alice
kubectl auth can-i create pods --as=alice
```

### Observability

```bash
# Auth-server logs
kubectl logs -n auth-server-system deployment/auth-server -f

# Hub rbac-apiserver logs
kubectl config use-context kind-hub
kubectl logs -n rbac-apiserver-system deployment/rbac-apiserver

# Check PermissionRequests (should be cleaned up)
kubectl get permissionrequests -w
```

## Configuration

### Environment Variables

- `CLUSTER_NAME`: Managed cluster name (required)

### Command Flags

- `--hub-kubeconfig`: Path to hub kubeconfig
- `--cluster-name`: Managed cluster name
- `--tls-cert-file`: TLS certificate
- `--tls-key-file`: TLS key
- `--addr`: Server address (default: `:8443`)
- `--v`: Log verbosity (0-5)

### Secrets Required

1. **hub-kubeconfig**: Kubeconfig for accessing hub rbac-apiserver
2. **auth-server-certs**: TLS certificates for HTTPS server

## Next Steps

### Phase 1: Fix User Context (rbac-apiserver)

1. Add `user` field to `PermissionRequestSpec`
2. Extract user from request context
3. Pass to SpiceDB integration
4. Update auth-server to include user in requests

### Phase 2: Performance Optimization

1. Implement auth-server caching with TTL
2. Add metrics and monitoring
3. Implement SubjectAccessReview API in rbac-apiserver
4. Load testing and optimization

### Phase 3: Production Readiness

1. Proper certificate management (cert-manager integration)
2. HA testing and failure scenarios
3. Graceful degradation when hub unavailable
4. Security hardening and audit logging
5. Group-based permissions support

### Phase 4: OCM Integration

1. Package as OCM ManagedClusterAddon
2. Automated deployment via addon manager
3. Hub-managed configuration
4. Integration with OCM RBAC

## Cleanup

```bash
kind delete cluster --name hub
kind delete cluster --name managed
```

## Dependencies

- Go 1.24+
- Docker
- Kind
- kubectl
- Helm 3
- rbac-apiserver (parent project)

## License

Follows the parent project's licensing.

## Support

For issues:
1. Check logs: `kubectl logs -n auth-server-system deployment/auth-server`
2. Verify connectivity to hub
3. Check PermissionBindings exist on hub
4. Verify webhook configuration

## Additional Resources

- [README.md](README.md) - Full documentation
- [QUICKSTART.md](QUICKSTART.md) - Quick start guide
- [Parent rbac-apiserver](../) - Hub API server
- [OCM Project](https://open-cluster-management.io/)
