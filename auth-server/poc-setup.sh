#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    echo_info "Checking prerequisites..."

    if ! command -v kind &> /dev/null; then
        echo_error "kind is not installed. Please install it from: https://kind.sigs.k8s.io/"
        exit 1
    fi

    if ! command -v kubectl &> /dev/null; then
        echo_error "kubectl is not installed."
        exit 1
    fi

    if ! command -v docker &> /dev/null; then
        echo_error "docker is not installed."
        exit 1
    fi

    if ! command -v helm &> /dev/null; then
        echo_error "helm is not installed."
        exit 1
    fi

    echo_info "All prerequisites are met!"
}

# Create Kind clusters
create_clusters() {
    echo_info "Creating Kind clusters..."

    # Create hub cluster
    if kind get clusters | grep -q "^hub$"; then
        echo_warn "Hub cluster already exists, skipping..."
    else
        echo_info "Creating hub cluster..."
        kind create cluster --name hub --image docker.io/kindest/node:v1.30.0
    fi

    # Create managed cluster WITHOUT webhook configuration initially
    # We'll configure webhook after auth-server is deployed
    if kind get clusters | grep -q "^managed$"; then
        echo_warn "Managed cluster already exists, skipping..."
    else
        echo_info "Creating managed cluster (webhook will be configured later)..."
        kind create cluster --name managed --image docker.io/kindest/node:v1.30.0
    fi

    echo_info "Clusters created successfully!"
}

# Deploy rbac-apiserver on hub
deploy_hub_rbac_apiserver() {
    echo_info "Deploying rbac-apiserver on hub cluster..."

    kubectl config use-context kind-hub

    # Wait for hub cluster to be ready
    echo_info "Waiting for hub cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=300s

    # Install cert-manager
    echo_info "Installing cert-manager..."
    kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml
    kubectl wait --for=condition=Available --timeout=300s -n cert-manager deployment/cert-manager
    kubectl wait --for=condition=Available --timeout=300s -n cert-manager deployment/cert-manager-webhook
    kubectl wait --for=condition=Available --timeout=300s -n cert-manager deployment/cert-manager-cainjector

    # Build and load rbac-apiserver image
    echo_info "Building rbac-apiserver image..."
    cd ../
    make build-image
    kind load docker-image quay.io/stolostron/rbac-apiserver:latest --name hub

    # Deploy rbac-apiserver
    echo_info "Installing rbac-apiserver with Helm..."
    helm install rbac-apiserver charts/rbac-apiserver/ \
        -n rbac-apiserver-system --create-namespace \
        --set image.tag=latest \
        --set tls.certManager.enabled=true \
        --wait

    # Verify deployment
    kubectl wait --for=condition=Available --timeout=300s -n rbac-apiserver-system deployment/rbac-apiserver

    echo_info "rbac-apiserver deployed successfully!"
}

# Build and deploy auth-server on managed cluster
deploy_managed_auth_server() {
    echo_info "Deploying auth-server on managed cluster..."

    # Wait for managed cluster to be ready
    kubectl config use-context kind-managed
    echo_info "Waiting for managed cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=300s

    # Build auth-server image
    echo_info "Building auth-server image..."
    cd auth-server
    docker build -t quay.io/stolostron/auth-server:latest .
    cd ..

    # Load image to managed cluster
    kind load docker-image quay.io/stolostron/auth-server:latest --name managed

    # Switch to managed cluster
    kubectl config use-context kind-managed

    # Create namespace
    kubectl apply -f auth-server/manifests/namespace.yaml
    kubectl apply -f auth-server/manifests/serviceaccount.yaml

    # Create hub kubeconfig secret
    echo_info "Creating hub kubeconfig secret..."
    kubectl config use-context kind-hub

    # Get hub cluster internal IP (accessible from managed cluster pods)
    HUB_IP=$(podman inspect hub-control-plane | grep -A5 Networks | grep IPAddress | head -1 | awk -F'"' '{print $4}')
    HUB_CA=$(kubectl config view --minify --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')
    HUB_TOKEN=$(kubectl create token default -n default --duration=87600h)

    # Create RBAC permissions for auth-server to call PermissionRequest API
    echo_info "Creating RBAC for PermissionRequest API access..."
    cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: permissionrequest-creator
rules:
- apiGroups: ["authorization.open-cluster-management.io"]
  resources: ["permissionrequests"]
  verbs: ["create", "get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: default-permissionrequest-creator
subjects:
- kind: ServiceAccount
  name: default
  namespace: default
roleRef:
  kind: ClusterRole
  name: permissionrequest-creator
  apiGroup: rbac.authorization.k8s.io
EOF

    kubectl config use-context kind-managed

    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: hub-kubeconfig
  namespace: auth-server-system
type: Opaque
stringData:
  kubeconfig: |
    apiVersion: v1
    kind: Config
    clusters:
    - name: hub
      cluster:
        server: https://${HUB_IP}:6443
        certificate-authority-data: ${HUB_CA}
    contexts:
    - name: hub
      context:
        cluster: hub
        user: hub-user
    current-context: hub
    users:
    - name: hub-user
      user:
        token: ${HUB_TOKEN}
EOF

    # Generate self-signed certs for auth-server
    echo_info "Generating TLS certificates for auth-server..."
    openssl req -x509 -newkey rsa:2048 -nodes \
        -keyout /tmp/tls.key \
        -out /tmp/tls.crt \
        -days 365 \
        -subj "/CN=auth-server.auth-server-system.svc" \
        -addext "subjectAltName=DNS:auth-server.auth-server-system.svc,DNS:auth-server.auth-server-system.svc.cluster.local"

    kubectl create secret tls auth-server-certs \
        --cert=/tmp/tls.crt \
        --key=/tmp/tls.key \
        -n auth-server-system

    # Deploy auth-server
    kubectl apply -f auth-server/manifests/service.yaml
    kubectl apply -f auth-server/manifests/deployment.yaml

    # Wait for deployment
    kubectl wait --for=condition=Available --timeout=300s -n auth-server-system deployment/auth-server

    echo_info "auth-server deployed successfully!"
}

# Configure webhook on managed cluster
configure_webhook() {
    echo_info "Configuring authorization webhook on managed cluster..."

    kubectl config use-context kind-managed

    # Get auth-server service ClusterIP
    AUTH_SERVER_IP=$(kubectl get svc auth-server -n auth-server-system -o jsonpath='{.spec.clusterIP}')

    # Create webhook config file
    cat > /tmp/webhook-config.yaml <<EOF
apiVersion: v1
kind: Config
clusters:
- name: auth-server
  cluster:
    insecure-skip-tls-verify: true
    server: https://${AUTH_SERVER_IP}:443/authorize
users:
- name: auth-server
contexts:
- context:
    cluster: auth-server
    user: auth-server
  name: webhook
current-context: webhook
EOF

    # Copy webhook config and TLS cert to control plane
    echo_info "Copying webhook config to managed cluster control plane..."
    podman cp /tmp/webhook-config.yaml managed-control-plane:/etc/kubernetes/pki/webhook-config.yaml
    podman cp /tmp/tls.crt managed-control-plane:/etc/kubernetes/pki/auth-server-ca.crt

    # Update kube-apiserver to use webhook authorization
    echo_info "Updating kube-apiserver to enable webhook authorization..."
    podman exec managed-control-plane bash -c "sed -i '/--authorization-mode=/c\    - --authorization-mode=Node,RBAC,Webhook' /etc/kubernetes/manifests/kube-apiserver.yaml"
    podman exec managed-control-plane bash -c "sed -i '/- kube-apiserver/a\    - --authorization-webhook-config-file=/etc/kubernetes/pki/webhook-config.yaml' /etc/kubernetes/manifests/kube-apiserver.yaml"

    # Wait for apiserver to restart
    echo_info "Waiting for kube-apiserver to restart..."
    sleep 15

    # Verify apiserver is running
    kubectl get pods -n kube-system | grep apiserver || echo_warn "Apiserver may still be restarting..."

    echo_info "Webhook configuration complete!"
    echo_info "The managed cluster kube-apiserver will now call auth-server for authorization decisions."
}

# Main execution
main() {
    echo_info "Starting POC setup..."

    check_prerequisites
    create_clusters
    deploy_hub_rbac_apiserver
    deploy_managed_auth_server
    configure_webhook

    echo_info "POC setup complete!"
    echo_info ""
    echo_info "==================================================================="
    echo_info "Setup Summary:"
    echo_info "  - Hub cluster: kind-hub (rbac-apiserver deployed)"
    echo_info "  - Managed cluster: kind-managed (auth-server addon + webhook configured)"
    echo_info "==================================================================="
    echo_info ""
    echo_info "Next steps to test:"
    echo_info "1. Create PermissionBindings on hub cluster:"
    echo_info "   kubectl config use-context kind-hub"
    echo_info "   kubectl apply -f auth-server/examples/permissionbinding-alice.yaml"
    echo_info "   kubectl apply -f auth-server/examples/permissionbinding-bob.yaml"
    echo_info ""
    echo_info "2. Test authorization webhook end-to-end:"
    echo_info "   kubectl config use-context kind-managed"
    echo_info "   kubectl run test-webhook --image=curlimages/curl --rm -i --restart=Never -- sh -c \\"
    echo_info "     'curl -k -X POST https://auth-server.auth-server-system.svc:443/authorize \\"
    echo_info "     -H \"Content-Type: application/json\" \\"
    echo_info "     -d '{\"apiVersion\":\"authorization.k8s.io/v1\",\"kind\":\"SubjectAccessReview\",\"spec\":{\"user\":\"alice\",\"resourceAttributes\":{\"namespace\":\"default\",\"verb\":\"get\",\"group\":\"\",\"resource\":\"pods\"}}}'"
    echo_info ""
    echo_info "Note: Fix the SpiceDB API integration on hub for full authorization functionality."
}

# Run main
main "$@"
