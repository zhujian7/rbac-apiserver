# RBAC API Server Helm Chart

This Helm chart deploys the RBAC API Server as a Kubernetes aggregated API server.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.0+
- (Optional) cert-manager for automatic TLS certificate generation

## Installation

### Option 1: With cert-manager (recommended for development/e2e)

Install cert-manager first:
```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.2/cert-manager.yaml
```

Then install the RBAC API Server:
```bash
helm install rbac-apiserver charts/rbac-apiserver/ \
  -n rbac-apiserver-system --create-namespace \
  --set tls.certManager.enabled=true \
  --wait
```

### Option 2: With your own TLS certificate (for production)

Create a TLS secret with your certificate:
```bash
kubectl create secret tls my-tls-cert \
  --cert=path/to/tls.crt \
  --key=path/to/tls.key \
  -n rbac-apiserver-system
```

Install the RBAC API Server:
```bash
helm install rbac-apiserver charts/rbac-apiserver/ \
  -n rbac-apiserver-system --create-namespace \
  --set tls.secretName=my-tls-cert \
  --wait
```

## Configuration

The following table lists the configurable parameters and their default values:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `quay.io/stolostron/rbac-apiserver` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `service.port` | Service port | `443` |
| `service.targetPort` | Container port | `6443` |
| `apiserver.verbosity` | Log verbosity level | `2` |
| `tls.certManager.enabled` | Use cert-manager for TLS | `false` |
| `tls.secretName` | TLS secret name (if not using cert-manager) | `""` |
| `resources.limits.cpu` | CPU limit | `100m` |
| `resources.limits.memory` | Memory limit | `128Mi` |

## Testing the Installation

After installation, verify the API server is running:

```bash
# Check if the APIService is available
kubectl get apiservices v1alpha1.widget.stolostron.io

# Create a test widget
kubectl apply -f - <<EOF
apiVersion: widget.stolostron.io/v1alpha1
kind: Widget
metadata:
  name: test-widget
  namespace: default
spec:
  size: 10
EOF

# List widgets
kubectl get widgets -n default
```

## Uninstallation

```bash
helm uninstall rbac-apiserver -n rbac-apiserver-system
```

## TLS Certificate Details

The RBAC API Server requires TLS certificates to serve HTTPS traffic. You have two options:

### cert-manager (Development/E2E)
- Automatically generates self-signed certificates
- CA bundle is automatically injected into the APIService
- No manual certificate management needed
- Perfect for development and testing

### Manual Certificates (Production)
- Bring your own TLS certificate
- Create a Kubernetes secret with `tls.crt` and `tls.key`
- Specify the secret name via `tls.secretName`
- You may need to configure the CA bundle manually in the APIService

## Notes

- The API server uses in-cluster authentication and authorization by default
- RBAC permissions are automatically configured via ClusterRole and ClusterRoleBinding
- The service account has delegated authentication and authorization permissions
