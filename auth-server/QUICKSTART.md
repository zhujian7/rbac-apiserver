# Quick Start Guide - Auth-Server POC

This guide will help you quickly set up and test the authorization webhook POC with two Kind clusters.

## Prerequisites

Ensure you have the following installed:
- Docker
- Kind
- kubectl
- Helm 3
- Go 1.24+

**Note**: The POC uses Kind with Kubernetes 1.30.0 (`docker.io/kindest/node:v1.30.0`)

## Step 1: Run the POC Setup

From the `auth-server` directory, run:

```bash
./poc-setup.sh
```

This will:
1. Create two Kind clusters (`hub` and `managed`)
2. Deploy rbac-apiserver on the hub cluster
3. Deploy auth-server on the managed cluster
4. Configure the authorization webhook

**Expected time**: 5-10 minutes

## Step 2: Restart the Managed Cluster API Server

The authorization webhook requires the kube-apiserver to restart:

```bash
# Stop the kube-apiserver container
docker exec managed-control-plane crictl stopp $(docker exec managed-control-plane crictl pods --name kube-apiserver -q)

# Wait for it to restart (should happen automatically)
sleep 30

# Verify it's running with webhook config
docker exec managed-control-plane ps aux | grep kube-apiserver | grep authorization-webhook
```

## Step 3: Create PermissionBindings on Hub

Switch to the hub cluster and create some test permissions:

```bash
kubectl config use-context kind-hub

# Apply example PermissionBindings
kubectl apply -f examples/permissionbinding-alice.yaml
kubectl apply -f examples/permissionbinding-bob.yaml

# Verify they were created
kubectl get permissionbindings
```

Expected output:
```
NAME                       AGE
alice-admin-permissions    10s
bob-viewer-permissions     10s
```

## Step 4: Test Authorization on Managed Cluster

Run the test script:

```bash
./examples/test-authorization.sh
```

Or test manually:

```bash
kubectl config use-context kind-managed

# Test as alice (should have admin permissions)
kubectl auth can-i get pods --as=alice -n default
kubectl auth can-i create deployments --as=alice -n default

# Test as bob (should have limited permissions)
kubectl auth can-i get pods --as=bob -n default
kubectl auth can-i create pods --as=bob -n default  # Should be denied
```

## Step 5: Monitor Authorization Decisions

Watch the auth-server logs in real-time:

```bash
kubectl config use-context kind-managed
kubectl logs -n auth-server-system deployment/auth-server -f
```

You should see logs like:
```
I1027 10:00:00.000000       1 server.go:XX] Starting authorization webhook server for cluster managed-cluster on :8443
I1027 10:00:01.000000       1 server.go:XX] Authorization request: user=alice, resource=/pods, verb=get, namespace=default
I1027 10:00:01.000000       1 server.go:XX] Creating PermissionRequest on hub
I1027 10:00:01.000000       1 server.go:XX] Authorization result for user alice: allowed=true
```

## Verification Checklist

- [ ] Hub cluster is running with rbac-apiserver
- [ ] Managed cluster is running with auth-server
- [ ] Authorization webhook is configured
- [ ] PermissionBindings exist on hub
- [ ] Authorization requests are being processed
- [ ] Auth-server logs show authorization decisions

## Common Issues

### Issue: Webhook not being called

**Solution**: Ensure the kube-apiserver was restarted with webhook configuration:

```bash
docker exec managed-control-plane cat /etc/kubernetes/authz-webhook/config.yaml
docker exec managed-control-plane ps aux | grep kube-apiserver | grep webhook
```

### Issue: Connection refused from auth-server to hub

**Solution**: Check the hub-kubeconfig secret:

```bash
kubectl config use-context kind-managed
kubectl get secret hub-kubeconfig -n auth-server-system -o yaml
```

### Issue: Auth-server pod not running

**Solution**: Check pod logs:

```bash
kubectl logs -n auth-server-system deployment/auth-server
kubectl describe pod -n auth-server-system -l app=auth-server
```

## Architecture Flow

When you run `kubectl auth can-i get pods --as=alice`:

1. **Managed cluster kube-apiserver** receives the authorization check
2. **Managed cluster kube-apiserver** calls the authorization webhook (auth-server)
3. **Auth-server** receives the SubjectAccessReview request
4. **Auth-server** creates a PermissionRequest on the hub rbac-apiserver
5. **Hub rbac-apiserver** evaluates against SpiceDB
6. **Hub rbac-apiserver** returns the status (allowed list)
7. **Auth-server** parses the response and determines allow/deny
8. **Auth-server** responds to the webhook with allow/deny
9. **Managed cluster kube-apiserver** enforces the decision
10. **Auth-server** deletes the PermissionRequest (cleanup)

## Next Steps

1. Experiment with different PermissionBindings
2. Test with actual pod creation/deletion
3. Monitor SpiceDB relationships on the hub
4. Try multi-namespace scenarios

## Cleanup

When done testing:

```bash
kind delete cluster --name hub
kind delete cluster --name managed
```

## Authorization Behavior

The auth-server uses a **NoOpinion** pattern:
- If a PermissionBinding exists on the hub for the user → **Allowed** (authorized via hub policy)
- If no PermissionBinding exists on the hub → **NoOpinion** (defers to local RBAC)
- If error contacting hub → **Denied** (fail-closed)

This means users can be authorized by **either** hub policies **or** local RBAC, providing flexibility and backwards compatibility.

## Support

For issues or questions:
1. Check auth-server logs: `kubectl logs -n auth-server-system deployment/auth-server`
2. Check rbac-apiserver logs: `kubectl config use-context kind-hub && kubectl logs -n rbac-apiserver-system deployment/rbac-apiserver`
3. Verify PermissionBindings: `kubectl get permissionbindings -o yaml`
