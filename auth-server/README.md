# Auth-Server - Authorization Webhook for OCM Multi-Cluster RBAC

This is an authorization webhook server that integrates with the rbac-apiserver on the hub cluster to provide centralized authorization for managed clusters.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                            Hub Cluster                              │
│                                                                     │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │  rbac-apiserver                                            │    │
│  │  - PermissionBinding API                                   │    │
│  │  - PermissionRequest API                                   │    │
│  │  - SpiceDB backend                                         │    │
│  └────────────────────────────────────────────────────────────┘    │
│                            ▲                                        │
│                            │ HTTPS                                  │
└────────────────────────────┼───────────────────────────────────────┘
                             │
                             │
┌────────────────────────────┼───────────────────────────────────────┐
│                   Managed Cluster                                  │
│                            │                                        │
│  ┌──────────────────┐      │      ┌─────────────────────────────┐  │
│  │  kube-apiserver  │──────┼─────▶│  auth-server (this addon)   │  │
│  │                  │      │      │  - Receives webhook calls   │  │
│  │  --authorization-│      │      │  - Calls hub PermissionReq  │  │
│  │   webhook        │      │      │  - Returns allow/deny       │  │
│  └──────────────────┘      │      └─────────────────────────────┘  │
└────────────────────────────┼───────────────────────────────────────┘
```

## Prerequisites

- Go 1.24+
- Docker
- Kind
- kubectl
- Helm 3.0+

## Quick Start - POC Setup

### 1. Automated Setup

Run the automated POC setup script:

```bash
./poc-setup.sh
```

This script will:
1. Create two Kind clusters (hub and managed)
2. Deploy rbac-apiserver on the hub cluster
3. Deploy auth-server on the managed cluster
4. Configure the authorization webhook

### 2. Manual Setup

If you prefer manual setup, follow these steps:

#### Hub Cluster Setup

```bash
# Create hub cluster
kind create cluster --name hub

# Deploy rbac-apiserver
cd ..
make build-image
kind load docker-image quay.io/stolostron/rbac-apiserver:latest --name hub

kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml
kubectl wait --for=condition=Available --timeout=300s -n cert-manager deployment/cert-manager
kubectl wait --for=condition=Available --timeout=300s -n cert-manager deployment/cert-manager-webhook

helm install rbac-apiserver charts/rbac-apiserver/ \
  -n rbac-apiserver-system --create-namespace \
  --set image.tag=latest \
  --set tls.certManager.enabled=true \
  --wait
```

#### Managed Cluster Setup

```bash
# Create managed cluster
kind create cluster --name managed

# Build and load auth-server
cd auth-server
make build-image
kind load docker-image quay.io/stolostron/auth-server:latest --name managed

# Deploy auth-server
kubectl config use-context kind-managed
kubectl apply -f manifests/namespace.yaml
kubectl apply -f manifests/serviceaccount.yaml

# Create hub kubeconfig secret (get from hub cluster)
kubectl create secret generic hub-kubeconfig \
  --from-file=kubeconfig=/path/to/hub/kubeconfig \
  -n auth-server-system

# Generate TLS certs
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout /tmp/tls.key \
  -out /tmp/tls.crt \
  -days 365 \
  -subj "/CN=auth-server.auth-server-system.svc"

kubectl create secret tls auth-server-certs \
  --cert=/tmp/tls.crt \
  --key=/tmp/tls.key \
  -n auth-server-system

# Deploy
kubectl apply -f manifests/service.yaml
kubectl apply -f manifests/deployment.yaml
```

## Testing the POC

### 1. Create PermissionBindings on Hub

Switch to hub cluster and create a permission binding:

```bash
kubectl config use-context kind-hub

cat <<EOF | kubectl apply -f -
apiVersion: authorization.open-cluster-management.io/v1alpha1
kind: PermissionBinding
metadata:
  name: alice-admin
spec:
  subject:
    kind: User
    name: alice
  permissions:
  - resources: ["pods", "deployments", "services"]
    groups: ["", "apps"]
    namespaces: ["default"]
    role: "admin"
    clusters: ["managed-cluster"]
EOF
```

### 2. Test Authorization on Managed Cluster

**Important Note**: Due to the current rbac-apiserver implementation hardcoding the user to "system:admin", all authorization checks will currently pass for system:admin permissions. This will be fixed when the user context is properly extracted from the PermissionRequest.

```bash
kubectl config use-context kind-managed

# Test as alice (will check system:admin permissions for now)
kubectl auth can-i get pods --as=alice -n default

# Test as bob (should be denied if no permissions)
kubectl auth can-i get pods --as=bob -n default
```

### 3. Check Auth-Server Logs

```bash
kubectl logs -n auth-server-system deployment/auth-server -f
```

You should see logs like:
```
I1027 10:00:00.000000       1 server.go:XX] Authorization request: user=alice, resource=/pods, verb=get, namespace=default
I1027 10:00:00.000000       1 server.go:XX] Creating PermissionRequest on hub
I1027 10:00:00.000000       1 server.go:XX] Authorization result for user alice: allowed=true
```

### 4. Verify on Hub

Check PermissionRequests being created (they are automatically cleaned up):

```bash
kubectl config use-context kind-hub
kubectl get permissionrequests -A -w
```

## Development

### Build Locally

```bash
make build
```

### Run Locally (for testing)

```bash
make run-local
```

### Build Docker Image

```bash
make build-image
```

## Configuration

### Environment Variables

- `CLUSTER_NAME`: Name of the managed cluster (required)

### Command-Line Flags

- `--hub-kubeconfig`: Path to hub cluster kubeconfig (default: `/etc/hub/kubeconfig`)
- `--cluster-name`: Managed cluster name (can also be set via `CLUSTER_NAME` env var)
- `--tls-cert-file`: TLS certificate file (default: `/etc/certs/tls.crt`)
- `--tls-key-file`: TLS key file (default: `/etc/certs/tls.key`)
- `--addr`: Server address (default: `:8443`)
- `--v`: Log verbosity level

## Known Limitations

1. **User Context**: Currently, the rbac-apiserver hardcodes the user to "system:admin". This means all permission checks are evaluated for system:admin, not the actual requesting user. This will be fixed in a future update to the rbac-apiserver.

2. **Performance**: Each authorization check creates and deletes a PermissionRequest object on the hub. For high-frequency workloads, consider implementing caching or using SubjectAccessReview API in the future.

3. **HA**: While the deployment runs 2 replicas, proper HA testing has not been performed.

## Troubleshooting

### Auth-server not receiving webhook calls

Check kube-apiserver configuration:
```bash
docker exec managed-control-plane ps aux | grep kube-apiserver | grep authorization-webhook
```

### Connection to hub failing

Check the hub-kubeconfig secret:
```bash
kubectl get secret hub-kubeconfig -n auth-server-system -o yaml
```

Test connectivity:
```bash
kubectl exec -n auth-server-system deployment/auth-server -- wget -O- https://rbac-apiserver.rbac-apiserver-system.svc:443/apis/authorization.open-cluster-management.io/v1alpha1
```

### TLS errors

Check certificate validity:
```bash
kubectl get secret auth-server-certs -n auth-server-system -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout
```

## Cleanup

```bash
# Delete managed cluster
kind delete cluster --name managed

# Delete hub cluster
kind delete cluster --name hub
```

## Next Steps

1. Update rbac-apiserver to accept user context in PermissionRequest
2. Implement caching for authorization decisions
3. Add metrics and monitoring
4. Implement proper client certificate authentication
5. Add support for group-based permissions
6. Implement SubjectAccessReview API for better performance
