#!/bin/bash

# Test script for authorization webhook
# Run this after setting up the POC

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
}

echo_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

echo_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
}

echo "=========================================="
echo "Authorization Webhook POC Test"
echo "=========================================="
echo ""

# Test 1: Create PermissionBindings on hub
echo_test "Creating PermissionBindings on hub cluster..."
kubectl config use-context kind-hub

# Delete existing PermissionBindings if they exist
kubectl delete -f examples/permissionbinding-alice.yaml --ignore-not-found=true
kubectl delete -f examples/permissionbinding-bob.yaml --ignore-not-found=true
kubectl delete -f examples/permissionbinding-charlie.yaml --ignore-not-found=true

# Create fresh PermissionBindings
kubectl create -f examples/permissionbinding-alice.yaml
kubectl create -f examples/permissionbinding-bob.yaml
kubectl create -f examples/permissionbinding-charlie.yaml

echo_pass "PermissionBindings created"
echo ""

# Give time for propagation
sleep 2

# Test 1b: Create native RBAC for charlie on managed cluster
echo_test "Creating native Kubernetes RBAC for charlie on managed cluster..."
kubectl config use-context kind-managed

kubectl apply -f examples/charlie-native-rbac.yaml

echo_pass "Native RBAC for charlie created"
echo ""

# Test 2: Check auth-server is running
echo_test "Checking auth-server status on managed cluster..."
kubectl config use-context kind-managed

if kubectl get pods -n auth-server-system -l app=auth-server | grep -q Running; then
    echo_pass "auth-server is running"
else
    echo_fail "auth-server is not running"
    exit 1
fi
echo ""

# Test 3: Test alice permissions
echo_test "Testing alice's permissions (should be allowed for pods in default)..."
kubectl config use-context kind-managed

# Note: Due to current limitation, this tests system:admin permissions
if kubectl auth can-i get pods --as=alice -n default &>/dev/null; then
    echo_pass "Alice can get pods in default namespace"
else
    echo_fail "Alice cannot get pods in default namespace"
fi

if kubectl auth can-i create deployments --as=alice -n default &>/dev/null; then
    echo_pass "Alice can create deployments in default namespace"
else
    echo_fail "Alice cannot create deployments in default namespace"
fi

# Test 4: Test bob permissions
echo_test "Testing bob's permissions (should be limited)..."

if kubectl auth can-i get pods --as=bob -n default &>/dev/null; then
    echo_pass "Bob can get pods in default namespace"
else
    echo_fail "Bob cannot get pods in default namespace"
fi

if kubectl auth can-i create pods --as=bob -n default &>/dev/null; then
    echo_fail "Bob can create pods (should not be allowed)"
else
    echo_pass "Bob cannot create pods (correctly denied)"
fi
echo ""

# Test 5: Test charlie permissions (NoOpinion case)
echo_test "Testing charlie's permissions (auth-server returns NoOpinion, native RBAC allows)..."

echo "  Charlie has PermissionBinding for 'other-cluster', not 'managed-cluster'"
echo "  Auth-server should return NoOpinion, falling back to native Kubernetes RBAC"
echo ""

if kubectl auth can-i get pods --as=charlie -n default &>/dev/null; then
    echo_pass "Charlie can get pods in default namespace (native RBAC allows)"
else
    echo_fail "Charlie cannot get pods in default namespace (should be allowed by native RBAC)"
fi

if kubectl auth can-i list pods --as=charlie -n default &>/dev/null; then
    echo_pass "Charlie can list pods in default namespace (native RBAC allows)"
else
    echo_fail "Charlie cannot list pods in default namespace (should be allowed by native RBAC)"
fi

if kubectl auth can-i create pods --as=charlie -n default &>/dev/null; then
    echo_fail "Charlie can create pods (native RBAC should not allow)"
else
    echo_pass "Charlie cannot create pods (correctly denied by native RBAC)"
fi

if kubectl auth can-i get pods --as=charlie -n kube-system &>/dev/null; then
    echo_fail "Charlie can get pods in kube-system (native RBAC should not allow)"
else
    echo_pass "Charlie cannot get pods in kube-system (correctly denied by native RBAC)"
fi
echo ""

# Test 6: Check auth-server logs
echo_test "Checking auth-server logs for recent authorization requests..."
kubectl logs -n auth-server-system deployment/auth-server --tail=30
echo ""

# Test 7: Verify PermissionReviews on hub
echo_test "Checking for PermissionReviews on hub (should be cleaned up)..."
kubectl config use-context kind-hub

PR_COUNT=$(kubectl get permissionreviews 2>/dev/null | grep -c authz- || true)
if [ "$PR_COUNT" -eq 0 ]; then
    echo_pass "PermissionReviews are properly cleaned up"
else
    echo_fail "Found $PR_COUNT PermissionReviews (may indicate cleanup issue)"
fi
echo ""

echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo ""
echo "Tests covered:"
echo "  1. Alice - Has hub PermissionBinding for managed-cluster (Allowed/Denied by auth-server)"
echo "  2. Bob - Has hub PermissionBinding for managed-cluster (Allowed/Denied by auth-server)"
echo "  3. Charlie - Has hub PermissionBinding for different cluster (NoOpinion from auth-server,"
echo "              falls back to native Kubernetes RBAC on managed cluster)"
echo ""
echo "Note: Current implementation hardcodes user to 'system:admin'"
echo "All checks currently evaluate against system:admin permissions."
echo "This will be fixed when rbac-apiserver accepts user context."
echo ""
echo "To see real-time authorization decisions, run:"
echo "  kubectl logs -n auth-server-system deployment/auth-server -f"
echo ""
