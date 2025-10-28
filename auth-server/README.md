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

# Generate TLS certs signed by cluster CA
# Deploy service first to get ClusterIP
kubectl apply -f manifests/service.yaml
AUTH_SERVER_IP=$(kubectl get svc auth-server -n auth-server-system -o jsonpath='{.spec.clusterIP}')

# Generate certificate inside control plane using cluster CA
# (Adjust for your cluster type - this example is for Kind)
kubectl exec -n kube-system <control-plane-pod> -- bash -c "
cd /etc/kubernetes/pki
openssl genrsa -out auth-server.key 2048
openssl req -new -key auth-server.key -out auth-server.csr \
  -subj '/CN=auth-server.auth-server-system.svc' \
  -addext 'subjectAltName=DNS:auth-server.auth-server-system.svc,DNS:auth-server.auth-server-system.svc.cluster.local,IP:${AUTH_SERVER_IP}'
openssl x509 -req -in auth-server.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out auth-server.crt -days 365 \
  -extensions v3_req -extfile <(printf '[v3_req]\nsubjectAltName=DNS:auth-server.auth-server-system.svc,DNS:auth-server.auth-server-system.svc.cluster.local,IP:${AUTH_SERVER_IP}')
"

# Extract certs and create secret
kubectl cp kube-system/<control-plane-pod>:/etc/kubernetes/pki/auth-server.crt /tmp/auth-server.crt
kubectl cp kube-system/<control-plane-pod>:/etc/kubernetes/pki/auth-server.key /tmp/auth-server.key

kubectl create secret tls auth-server-certs \
  --cert=/tmp/auth-server.crt \
  --key=/tmp/auth-server.key \
  -n auth-server-system

# Deploy
kubectl apply -f manifests/service.yaml
kubectl apply -f manifests/deployment.yaml
```

## Testing the POC

### 1. Create PermissionBindings on Hub

Switch to hub cluster and create permission bindings:

```bash
kubectl config use-context kind-hub

# Apply example PermissionBindings
kubectl apply -f auth-server/examples/permissionbinding-alice.yaml
kubectl apply -f auth-server/examples/permissionbinding-bob.yaml
```

### 2. Test Authorization on Managed Cluster

The auth-server implements NoOpinion authorization, allowing both hub and local RBAC:

```bash
kubectl config use-context kind-managed

# Test as alice (has PermissionBinding on hub)
kubectl auth can-i get pods --as=alice -n default
# Expected: yes (authorized via hub policy)

# Test as bob (no hub policy, but may have local RBAC)
kubectl auth can-i get pods --as=bob -n default
# Expected: depends on local RoleBindings (NoOpinion defers to RBAC)

# Test as charlie (no hub policy, no local RBAC)
kubectl auth can-i get pods --as=charlie -n default
# Expected: no (NoOpinion → RBAC checks → denied)
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

## Authorization Behavior

The auth-server implements a **NoOpinion** authorization pattern that allows falling back to local RBAC:

1. **Hub policy allows**: Returns `Allowed=true` (authorization chain stops, request allowed)
2. **No hub policy found**: Returns `NoOpinion` (`Allowed=false, Denied=false`) - authorization chain continues to local RBAC
3. **Error contacting hub**: Returns `Denied=true` (fail-closed for security)

This means users can be authorized by either:
- Hub RBAC policies (PermissionBindings on hub cluster)
- Local RBAC policies (RoleBindings on managed cluster)

Example: If alice has a PermissionBinding on the hub, she's authorized via hub policy. If bob has no hub policy but has a local RoleBinding, he's authorized via local RBAC.

## Technical Details

### Why ClusterIP Instead of DNS Names?

The webhook configuration uses ClusterIP addresses instead of DNS names (e.g., `https://10.96.11.8:443` instead of `https://auth-server.auth-server-system.svc:443`).

**Reason**: kube-apiserver runs with `hostNetwork: true` and cannot resolve in-cluster DNS names. This is a Kubernetes design limitation that affects authorization webhooks (which use kubeconfig-based configuration).

This limitation applies to all Kubernetes distributions (Kind, OpenShift, EKS, AKS, GKE, etc.), not just Kind clusters.

**Note**: ClusterIP addresses are stable and don't change unless the Service is deleted and recreated.

References:
- [Kind Issue #2467](https://github.com/kubernetes-sigs/kind/issues/2467)
- Kubernetes source: `staging/src/k8s.io/apiserver/plugin/pkg/authorizer/webhook/webhook.go`

### TLS Certificate Requirements

The auth-server certificate must be:
- Signed by the cluster's CA (same CA that signs apiserver certificates)
- Include proper SANs (Subject Alternative Names):
  - DNS: `auth-server.auth-server-system.svc`
  - DNS: `auth-server.auth-server-system.svc.cluster.local`
  - IP: Service ClusterIP address

This allows the kube-apiserver to verify the certificate using its trusted CA bundle.

## Known Limitations

1. **Performance**: Each authorization check creates and deletes a PermissionRequest object on the hub. For high-frequency workloads, consider implementing caching.

2. **HA**: While the deployment runs 2 replicas, proper HA testing has not been performed.

3. **ClusterIP Dependency**: The webhook configuration uses ClusterIP which requires the Service to remain stable. Deleting and recreating the Service will require webhook reconfiguration.

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
