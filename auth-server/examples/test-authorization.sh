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

kubectl apply -f examples/permissionbinding-alice.yaml
kubectl apply -f examples/permissionbinding-bob.yaml

echo_pass "PermissionBindings created"
echo ""

# Give time for propagation
sleep 2

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

if kubectl auth can-i get pods --as=alice -n kube-system &>/dev/null; then
    echo_pass "Alice can get pods in kube-system namespace (viewer)"
else
    echo_fail "Alice cannot get pods in kube-system namespace"
fi
echo ""

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

# Test 5: Check auth-server logs
echo_test "Checking auth-server logs for recent authorization requests..."
kubectl logs -n auth-server-system deployment/auth-server --tail=20
echo ""

# Test 6: Verify PermissionRequests on hub
echo_test "Checking for PermissionRequests on hub (should be cleaned up)..."
kubectl config use-context kind-hub

PR_COUNT=$(kubectl get permissionrequests 2>/dev/null | grep -c authz- || true)
if [ "$PR_COUNT" -eq 0 ]; then
    echo_pass "PermissionRequests are properly cleaned up"
else
    echo_fail "Found $PR_COUNT PermissionRequests (may indicate cleanup issue)"
fi
echo ""

echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo ""
echo "Note: Current implementation hardcodes user to 'system:admin'"
echo "All checks currently evaluate against system:admin permissions."
echo "This will be fixed when rbac-apiserver accepts user context."
echo ""
echo "To see real-time authorization decisions, run:"
echo "  kubectl logs -n auth-server-system deployment/auth-server -f"
echo ""
