# Authorization API Implementation

## Overview

This document describes the Authorization API that has been added to the rbac-apiserver. This API is designed to manage multi-cluster RBAC authorization using embedded SpiceDB for real-time permission evaluation.

## API Details

### API Group and Version

- **API Group**: `authorization.open-cluster-management.io`
- **Version**: `v1alpha1`
- **Resources**: `permissionbindings`, `permissionreviews`
- **Full API Path**: `/apis/authorization.open-cluster-management.io/v1alpha1/`

### Resource Types

#### PermissionBinding

Represents permission bindings that grant users or groups access to resources across clusters. These bindings are automatically synced to SpiceDB.

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
```

#### PermissionReview (CSR-like)

Represents a permission check request. Like Kubernetes CertificateSigningRequests, PermissionReviews are evaluated immediately by SpiceDB and not persisted.

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

#### Key Components

**PermissionBinding Spec**:

- `subject`: The entity receiving permissions
  - `kind`: "User" or "Group"
  - `name`: Subject identifier (e.g., "alice", "developers")
- `permissions`: List of permission grants
  - `role`: Permission level ("admin", "editor", "viewer")
  - `resources`: Resource types (e.g., "pods", "deployments")
  - `groups`: API groups (e.g., "", "apps", "\*")
  - `namespaces`: Target namespaces or ["\*"]
  - `names`: Specific resource names or ["\*"]
  - `clusters`: Target cluster names

**PermissionReview Spec**:

- `group`: API group ("" for core)
- `resource`: Resource type (e.g., "pods")
- `verb`: Action to check (e.g., "get", "create", "delete")
- `cluster`: Target cluster
- `namespace`: Target namespace (optional)
- `name`: Specific resource name (optional)

**PermissionReview Status**:

- `allowedList`: List of allowed resources
  - `cluster`: Cluster name
  - `namespacedNames`: List of namespace/name combinations allowed

## Implementation Status

### ✅ Fully Implemented

1. **API Types Defined**
   - [apis/rbac/v1alpha1/permissionbinding_type.go](../apis/rbac/v1alpha1/permissionbinding_type.go)
   - [apis/rbac/v1alpha1/permissionreview_type.go](../apis/rbac/v1alpha1/permissionreview_type.go)
   - DeepCopy methods for runtime.Object interface
   - OpenAPI schema markers for validation

2. **Embedded SpiceDB Backend**
   - [pkg/spicedb/](../pkg/spicedb/) - Embedded SpiceDB server
   - Real-time authorization evaluation
   - Automatic tuple management from PermissionBindings
   - Permission check API for PermissionReviews

3. **REST Storage Registry**
   - [pkg/registry/permissionbinding_rest.go](../pkg/registry/permissionbinding_rest.go)
   - [pkg/registry/permissionreview_rest.go](../pkg/registry/permissionreview_rest.go)
   - Implements all required Kubernetes storage interfaces
   - Integrated with SpiceDB for authorization

4. **API Registration**
   - Added in [cmd/main.go](../cmd/main.go)
   - Registered PermissionBinding and PermissionReview types
   - Installed API group with both OpenAPI v2 and v3

5. **OpenAPI Generation**
   - Updated [hack/update-codegen.sh](../hack/update-codegen.sh)
   - Generated OpenAPI specs in [apis/generated/openapi/](../apis/generated/openapi/)

6. **E2E Tests**
   - [test/e2e/permissionbinding_test.go](../test/e2e/permissionbinding_test.go)
   - [test/e2e/permissionreview_test.go](../test/e2e/permissionreview_test.go)
   - Full CRUD operation tests
   - Authorization evaluation tests
   - Uses Ginkgo BDD framework

### 🎯 Working Operations

#### PermissionBinding Operations

1. **Create Operation**:
   - Endpoint: `POST /apis/authorization.open-cluster-management.io/v1alpha1/permissionbindings`
   - ✅ Creates permission bindings
   - ✅ Automatically syncs to SpiceDB as tuples
   - ✅ Sets metadata (timestamps, UID, resource version)

2. **List Operation**:
   - Endpoint: `GET /apis/authorization.open-cluster-management.io/v1alpha1/permissionbindings`
   - ✅ Returns all stored bindings

3. **Get Operation**:
   - Endpoint: `GET /apis/authorization.open-cluster-management.io/v1alpha1/permissionbindings/{name}`
   - ✅ Retrieves specific binding by name

4. **Update Operation**:
   - Endpoint: `PUT /apis/authorization.open-cluster-management.io/v1alpha1/permissionbindings/{name}`
   - ✅ Updates binding and re-syncs to SpiceDB

5. **Delete Operation**:
   - Endpoint: `DELETE /apis/authorization.open-cluster-management.io/v1alpha1/permissionbindings/{name}`
   - ✅ Removes binding and cleans up SpiceDB tuples

#### PermissionReview Operations

1. **Create Operation** (CSR-like):
   - Endpoint: `POST /apis/authorization.open-cluster-management.io/v1alpha1/permissionreviews`
   - ✅ Evaluates permissions immediately against SpiceDB
   - ✅ Returns result in status field
   - ✅ NOT persisted (evaluated and returned immediately)
   - ✅ User context extracted from request (via `--as` flag)

## Files Created/Modified

### New Files

- `apis/rbac/doc.go` - Package constants
- `apis/rbac/v1alpha1/doc.go` - API version constants
- `apis/rbac/v1alpha1/permissionbinding_type.go` - PermissionBinding type definitions
- `apis/rbac/v1alpha1/permissionreview_type.go` - PermissionReview type definitions
- `pkg/registry/permissionbinding_rest.go` - PermissionBinding REST storage
- `pkg/registry/permissionreview_rest.go` - PermissionReview REST storage
- `pkg/spicedb/` - Embedded SpiceDB integration
- `test/e2e/permissionbinding_test.go` - PermissionBinding E2E tests
- `test/e2e/permissionreview_test.go` - PermissionReview E2E tests

### Modified Files

- `cmd/main.go` - Added Authorization API registration
- `hack/update-codegen.sh` - Added rbac/v1alpha1 to code generation
- `apis/generated/openapi/zz_generated.openapi.go` - Auto-generated OpenAPI specs

## SpiceDB Integration

### Schema Design

The API uses a SpiceDB schema that supports multi-cluster RBAC:

```zed
definition user {}

definition group {
  relation member: user
}

definition resource {
  relation admin: user | group#member
  relation edit: user | group#member
  relation view: user | group#member

  permission can_admin = admin
  permission can_edit = edit + admin
  permission can_view = view + edit + admin
}
```

### Tuple Format

PermissionBindings are converted to SpiceDB tuples:

```
resource:<cluster>/<namespace>/<resource>/<name>#<permission>@user:<username>
resource:<cluster>/<namespace>/<resource>/<name>#<permission>@group:<groupname>#member
```

### Permission Evaluation

PermissionReviews query SpiceDB to check permissions:

- **View verbs** (get, list, watch) → check `can_view` permission
- **Edit verbs** (create, update, patch, delete) → check `can_edit` permission
- **Admin role** → check `can_admin` permission

## Example Usage

### Create PermissionBinding

```bash
kubectl apply -f - <<EOF
apiVersion: authorization.open-cluster-management.io/v1alpha1
kind: PermissionBinding
metadata:
  name: bob-multi-cluster-admin
spec:
  subject:
    kind: User
    name: bob
  permissions:
    - role: admin
      resources:
        - pods
        - deployments
        - services
      groups:
        - ""
        - apps
      namespaces:
        - production
        - staging
      names:
        - "*"
      clusters:
        - cluster1
        - cluster2
EOF
```

### Check Permissions with PermissionReview

```bash
kubectl create -f - <<EOF --as=bob -o yaml
apiVersion: authorization.open-cluster-management.io/v1alpha1
kind: PermissionReview
metadata:
  name: check-bob-access
spec:
  group: apps
  resource: deployments
  verb: create
  cluster: cluster1
  namespace: production
  name: web-app
EOF
```

Expected output:

```yaml
status:
  allowedList:
    - cluster: cluster1
      namespacedNames:
        - namespace: production
          names:
            - web-app
```

### List PermissionBindings

```bash
kubectl get permissionbindings
```

### Delete PermissionBinding

```bash
kubectl delete permissionbinding bob-multi-cluster-admin
```

## Design Principles

### PermissionReview CSR-like Pattern

PermissionReviews follow the Kubernetes CertificateSigningRequest pattern:

1. **Immediate Evaluation**: Evaluated immediately upon creation
2. **Not Persisted**: Results returned directly, not stored in etcd
3. **User Context**: User extracted from Kubernetes request context
4. **Use `kubectl create`**: Not `kubectl apply` (since they're not stored)

### Benefits

- Real-time authorization decisions using SpiceDB
- Multi-cluster RBAC with consistent policy enforcement
- Declarative permission management via Kubernetes CRDs
- Fine-grained access control at resource level
- Support for wildcards and group-based permissions

## Migration Notes

This API replaced the previous Relationship API with:

- More Kubernetes-native resource types
- Embedded SpiceDB instead of external dependency
- CSR-like pattern for permission checks
- Better alignment with Kubernetes RBAC concepts

## See Also

- [EXAMPLES.md](../EXAMPLES.md) - Detailed usage examples
- [README.md](../README.md) - Project overview and setup
- [SpiceDB Documentation](https://authzed.com/docs)
