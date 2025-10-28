# RBAC API Server - Usage Examples

This guide provides runnable examples for using the RBAC API Server with embedded SpiceDB for authorization.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Example 1: Basic User Permission](#example-1-basic-user-permission)
- [Example 2: Group Permissions](#example-2-group-permissions)
- [Example 3: Multi-Cluster Permissions](#example-3-multi-cluster-permissions)
- [Example 4: Wildcard Permissions](#example-4-wildcard-permissions)
- [Understanding PermissionBindings](#understanding-permissionbindings)
- [Understanding PermissionRequests](#understanding-permissionrequests)
- [Troubleshooting](#troubleshooting)

## Overview

The RBAC API Server provides two main custom resources:

1. **PermissionBinding**: Defines what permissions a user or group has on resources across clusters
2. **PermissionRequest**: Queries whether a user has permission to perform an action on a resource

The API server uses embedded SpiceDB to store and evaluate these permissions in real-time.

## Prerequisites

- Kubernetes cluster (or kind/minikube)
- kubectl configured to access the cluster
- RBAC API Server deployed (see main README.md)

## Quick Start

### Verify the API Server is Running

```bash
# Check if the API server pod is running
kubectl get pods -n rbac-apiserver-system

# Check if the API is registered
kubectl api-resources | grep authorization.open-cluster-management.io
```

Expected output:
```
permissionbindings          authorization.open-cluster-management.io/v1alpha1   false   PermissionBinding
permissionrequests          authorization.open-cluster-management.io/v1alpha1   false   PermissionRequest
```

## Example 1: Basic User Permission

This example shows how to grant a user permission to view pods in a specific namespace.

### Step 1: Create a PermissionBinding

Apply the example PermissionBinding for alice with viewer role:

```bash
kubectl apply -f examples/01-user-viewer-binding.yaml
```

This binding grants user "alice" viewer access to all pods in the default namespace of cluster1.

<details>
<summary>View the YAML definition</summary>

See [examples/01-user-viewer-binding.yaml](examples/01-user-viewer-binding.yaml)
</details>

### Step 2: Verify the PermissionBinding

```bash
kubectl get permissionbindings
kubectl get permissionbinding alice-pod-viewer -o yaml
```

### Step 3: Create a PermissionRequest to Check Access

Apply the example PermissionRequest to check if alice can view a pod:

```bash
kubectl apply -f examples/02-check-user-view-access.yaml
```

This request checks if alice has permission to get (view) a pod named "my-pod" in the default namespace of cluster1.

<details>
<summary>View the YAML definition</summary>

See [examples/02-check-user-view-access.yaml](examples/02-check-user-view-access.yaml)
</details>

### Step 4: View the Result

```bash
kubectl get permissionrequest check-alice-pod-view -o yaml
```

The PermissionRequest object will be created and processed by the API server. The status field will contain the evaluation results from SpiceDB:

```yaml
status:
  allowedList:
    - cluster: cluster1
      namespacedNames:
        - namespace: default
          names:
            - my-pod
```

Since alice has viewer role and the wildcard matching is enabled, the status shows alice has permission to view my-pod.

### Cleanup

```bash
kubectl delete permissionbinding alice-pod-viewer
kubectl delete permissionrequest check-alice-pod-view
```

## Example 2: Group Permissions

This example demonstrates how to grant permissions to a group of users.

### Step 1: Create a PermissionBinding for a Group

Apply the example PermissionBinding for the developers group:

```bash
kubectl apply -f examples/02-group-editor-binding.yaml
```

This binding grants the "developers" group editor access to deployments and services in the development namespace of cluster1.

<details>
<summary>View the YAML definition</summary>

See [examples/02-group-editor-binding.yaml](examples/02-group-editor-binding.yaml)
</details>

### Step 2: Verify Group Permissions

```bash
kubectl get permissionbinding developers-group-editor -o yaml
```

The developers group can now create, update, and delete deployments and services in the development namespace.

### Cleanup

```bash
kubectl delete permissionbinding developers-group-editor
```

## Example 3: Multi-Cluster Permissions

This example shows how to grant permissions across multiple clusters and namespaces.

### Step 1: Create Multi-Cluster PermissionBinding

Apply the example PermissionBinding with multi-cluster admin access:

```bash
kubectl apply -f examples/03-multi-cluster-binding.yaml
```

This binding grants user "bob" admin access to pods, deployments, and services across cluster1 and cluster2 in both production and staging namespaces.

<details>
<summary>View the YAML definition</summary>

See [examples/03-multi-cluster-binding.yaml](examples/03-multi-cluster-binding.yaml)
</details>

### Step 2: Verify Multi-Cluster Access

```bash
kubectl get permissionbinding bob-multi-cluster-admin -o yaml
```

Bob can now perform all operations (create, read, update, delete) on the specified resources in both clusters and namespaces.

### Cleanup

```bash
kubectl delete permissionbinding bob-multi-cluster-admin
```

## Example 4: Wildcard Permissions

This example demonstrates granting broad permissions using wildcards.

### Step 1: Create Wildcard PermissionBinding

Apply the example PermissionBinding with wildcard admin access:

```bash
kubectl apply -f examples/04-wildcard-admin-binding.yaml
```

This binding grants user "admin-user" full admin access to ALL resources in ALL namespaces of cluster1 using wildcard matching.

<details>
<summary>View the YAML definition</summary>

See [examples/04-wildcard-admin-binding.yaml](examples/04-wildcard-admin-binding.yaml)
</details>

### Step 2: Verify Wildcard Access

```bash
kubectl get permissionbinding cluster-admin-all-access -o yaml
```

The admin-user can now perform all operations on any resource type in any namespace within cluster1.

**⚠️ Warning**: Wildcard permissions grant very broad access. Use with caution in production environments.

### Cleanup

```bash
kubectl delete permissionbinding cluster-admin-all-access
```

## Understanding PermissionBindings

### PermissionBinding Spec

```yaml
spec:
  subject:
    kind: User | Group        # Type of subject
    name: string              # Subject name (e.g., "alice", "developers")
  permissions:
    - role: admin | editor | viewer  # Permission level
      resources:              # List of Kubernetes resources
        - pods
        - deployments
      groups:                 # API groups (e.g., "", "apps", "*")
        - ""
        - apps
      namespaces:             # List of namespaces (or ["*"])
        - default
      clusters:               # List of clusters (or ["*"])
        - cluster1
      names:                  # Specific resource names (or ["*"] for all)
        - my-pod
```

### Available Roles

- **admin**: Full access (create, read, update, delete)
- **editor**: Read and write access (create, read, update, delete)
- **viewer**: Read-only access (get, list, watch)

### Subject Types

- **User**: Individual user (referenced as `user:username` in SpiceDB)
- **Group**: Group of users (referenced as `group:groupname#member` in SpiceDB)

## Understanding PermissionRequests

### How PermissionRequests Work (CSR-like Pattern)

PermissionRequests work similarly to Kubernetes CertificateSigningRequests (CSR):

- **Immediate Evaluation**: When you create a PermissionRequest, it is evaluated immediately against SpiceDB
- **Not Persisted**: The request is NOT stored in etcd - it's evaluated and returned immediately
- **No Get/List Operations**: You cannot use `kubectl get permissionrequest` to retrieve past requests
- **User Context**: The requesting user is extracted from the Kubernetes request context using the `--as` flag

### Usage Pattern

Use `kubectl create` (not `kubectl apply`) with the `--as` flag to specify which user's permissions to check:

```bash
kubectl create -f examples/01-check-user-view-access.yaml --as=alice -o yaml
```

The command returns immediately with the evaluation result in the status field.

### Required RBAC Permissions

Before users can create PermissionRequests, they need Kubernetes RBAC permissions:

```bash
kubectl apply -f examples/01-permissionrequests-assign.yaml
```

This grants the user permission to create PermissionRequest resources in the API server.

### PermissionRequest Spec

```yaml
spec:
  group: string         # API group ("" for core, "apps", etc.)
  resource: string      # Kubernetes resource type (pods, deployments, etc.)
  verb: string          # Action to perform (get, list, create, update, delete, watch)
  cluster: string       # Target cluster name
  namespace: string     # Target namespace (optional for cluster-scoped resources)
  name: string          # Specific resource name (optional)
```

### Verb Mapping

- **get, list, watch** → maps to **view** permission
- **create, update, patch, delete** → maps to **edit** permission

### PermissionRequest Status

After processing, the PermissionRequest status will contain:

```yaml
status:
  allowedList:
    - cluster: cluster1
      namespacedNames:
        - namespace: default
          names:
            - my-pod
```

### Example Usage

```bash
# Check if alice can view a pod
kubectl create -f examples/01-check-user-view-access.yaml --as=alice -o yaml

# Check if bob can create a deployment
kubectl create -f examples/03-check-multi-cluster-access.yaml --as=bob -o yaml

# Check if admin-user can delete a configmap
kubectl create -f examples/04-check-wildcard-admin-access.yaml --as=admin-user -o yaml
```

## Troubleshooting

### Check API Server Logs

```bash
# Get the pod name
POD_NAME=$(kubectl get pods -n rbac-apiserver-system -l app=rbac-apiserver -o jsonpath='{.items[0].metadata.name}')

# View logs
kubectl logs -n rbac-apiserver-system $POD_NAME

# Follow logs
kubectl logs -n rbac-apiserver-system $POD_NAME -f
```

### Common Issues

1. **PermissionBinding not syncing to SpiceDB**
   - Check API server logs for SpiceDB errors
   - Verify SpiceDB is running in the pod
   - Check for validation errors

2. **PermissionRequest returns empty status**
   - Ensure the user has Kubernetes RBAC permissions to create PermissionRequests (see examples/01-permissionrequests-assign.yaml)
   - Verify you're using `kubectl create` with the `--as` flag to specify the user
   - Check that the resource names, cluster, and namespace match the PermissionBinding
   - Ensure PermissionBindings have been created and synced to SpiceDB first

3. **Cannot retrieve PermissionRequest with kubectl get**
   - This is expected behavior - PermissionRequests are NOT persisted
   - They work like CSR (CertificateSigningRequest) and are evaluated immediately
   - Use `kubectl create -o yaml` to see the immediate evaluation result

4. **Invalid ObjectId errors**
   - The system automatically sanitizes IDs, but check logs if issues persist
   - Avoid special characters in subject names

### Verify SpiceDB Integration

Check that SpiceDB relationships are created:

```bash
# Port forward to SpiceDB (if exposed)
kubectl port-forward -n rbac-apiserver-system svc/rbac-apiserver 50051:50051

# Use SpiceDB CLI or grpcurl to query relationships
# (requires SpiceDB CLI tools)
```

## Next Steps

- Read the [Architecture Documentation](ARCHITECTURE.md) to understand how SpiceDB integration works
- Review the [API Reference](API.md) for complete API specifications
- Check out the [E2E Tests](test/e2e/) for more complex examples
- Learn about [SpiceDB Schema](pkg/spicedb/bootstrap.yaml) used by the API server

## Contributing

Found an issue or have a suggestion? Please open an issue or submit a pull request!
