# Relationship API Implementation

## Overview

This document describes the Relationship API framework that has been added to the rbac-apiserver. This API is designed to manage SpiceDB-based authorization relationships for multi-cluster RBAC scenarios.

## API Details

### API Group and Version
- **API Group**: `multicluster-rbac.open-cluster-management.io`
- **Version**: `v1alpha1`
- **Resource**: `relationships`
- **Full API Path**: `/apis/multicluster-rbac.open-cluster-management.io/v1alpha1/relationships`

### Resource Types

#### Relationship
Represents a relationship tuple for SpiceDB-based authorization.

```yaml
apiVersion: multicluster-rbac.open-cluster-management.io/v1alpha1
kind: Relationship
metadata:
  name: example-relationship
spec:
  tuples:
  - resource:
      objectType: "resource"
      objectId: "cluster/cluster1/namespace/_wildcard_"
    relation: "admin"
    subject:
      object:
        objectType: "user"
        objectId: "user2"
```

#### Key Components

**Tuple**: Defines a single relationship between a subject and a resource
- `resource`: The object being accessed (ObjectReference)
  - `objectType`: Type of object (e.g., "resource", "namespace", "cluster")
  - `objectId`: Identifier for the object (e.g., "cluster/cluster1/namespace/_wildcard_")
- `relation`: Type of relationship (e.g., "admin", "viewer", "editor")
- `subject`: The entity with the relationship (SubjectReference)
  - `object`: ObjectReference identifying the subject
  - `relation`: Optional relation for group membership

## Implementation Status

### ✅ Fully Implemented (In-Memory Storage)

1. **API Types Defined** ([apis/rbac/v1alpha1/relationship.go](apis/rbac/v1alpha1/relationship.go))
   - Relationship and RelationshipList types
   - Tuple, ObjectReference, SubjectReference types
   - DeepCopy methods for runtime.Object interface

2. **In-Memory Storage Backend** ([pkg/storage/relationship_storage.go](pkg/storage/relationship_storage.go))
   - Thread-safe concurrent storage using sync.RWMutex
   - Full CRUD operations implemented
   - Resource versioning with counter
   - Cluster-scoped storage (no namespace handling needed)

3. **REST Storage Registry** ([pkg/registry/relationship_rest.go](pkg/registry/relationship_rest.go))
   - Implements all required Kubernetes storage interfaces:
     - rest.Creater - ✅ Create relationships
     - rest.Lister - ✅ List all relationships
     - rest.Getter - ✅ Get relationship by name
     - rest.GracefulDeleter - ✅ Delete relationships
     - rest.Scoper - Cluster-scoped resource
     - rest.Storage - Storage interface
     - rest.GroupVersionKindProvider - GVK provider
   - Connected to in-memory storage backend

4. **API Registration** ([cmd/main.go](cmd/main.go))
   - Added rbac API group imports
   - Registered Relationship types in scheme
   - Installed API group in installAPI function
   - Supports both OpenAPI v2 and v3

5. **OpenAPI Generation** ([hack/update-codegen.sh](hack/update-codegen.sh))
   - Updated to include rbac/v1alpha1 package
   - Generated OpenAPI specs include Relationship types
   - Verified in [apis/generated/openapi/zz_generated.openapi.go](apis/generated/openapi/zz_generated.openapi.go)

6. **E2E Tests** ([test/e2e/relationship_test.go](test/e2e/relationship_test.go))
   - Full CRUD operation tests
   - Multi-tuple relationship tests
   - Error handling tests (duplicates, not found)
   - Uses Ginkgo BDD framework

7. **Build Verification**
   - ✅ Project builds successfully
   - ✅ E2E test binary builds
   - ✅ No compilation errors

### 🎯 Working Operations

All CRUD operations are fully functional with in-memory storage:

1. **Create Operation**:
   - Endpoint: `POST /apis/multicluster-rbac.open-cluster-management.io/v1alpha1/relationships`
   - ✅ Creates relationship tuples in memory
   - ✅ Generates UUID if name not provided
   - ✅ Sets metadata (timestamps, UID, resource version)

2. **List Operation**:
   - Endpoint: `GET /apis/multicluster-rbac.open-cluster-management.io/v1alpha1/relationships`
   - ✅ Returns all stored relationships

3. **Get Operation**:
   - Endpoint: `GET /apis/multicluster-rbac.open-cluster-management.io/v1alpha1/relationships/{name}`
   - ✅ Retrieves specific relationship by name

4. **Delete Operation**:
   - Endpoint: `DELETE /apis/multicluster-rbac.open-cluster-management.io/v1alpha1/relationships/{name}`
   - ✅ Removes relationship from storage

## Files Created/Modified

### New Files

- `apis/rbac/doc.go` - Package constants
- `apis/rbac/v1alpha1/doc.go` - API version constants
- `apis/rbac/v1alpha1/relationship.go` - Type definitions
- `pkg/storage/relationship_storage.go` - In-memory storage backend
- `pkg/registry/relationship_rest.go` - REST storage implementation
- `test/e2e/relationship_test.go` - E2E tests

### Modified Files

- `cmd/main.go` - Added Relationship API registration
- `hack/update-codegen.sh` - Added rbac/v1alpha1 to code generation (including OpenAPI)
- `apis/generated/openapi/zz_generated.openapi.go` - Auto-generated OpenAPI specs

## Current Storage: In-Memory

The API currently uses in-memory storage, similar to the Widget API. This allows:

- Full testing and development without external dependencies
- Easy e2e test execution
- Simple demonstration of the API
- Clear separation between API layer and storage backend

## Next Steps for SpiceDB Integration

To migrate from in-memory storage to SpiceDB, you'll need to:

1. **Add SpiceDB Client Integration**
   - Add SpiceDB client library to go.mod
   - Create SpiceDB storage backend in pkg/storage/
   - Configure SpiceDB connection parameters

2. **Implement Storage Operations**
   - Update `relationship_rest.go` to use SpiceDB storage
   - Implement Create: Write tuples to SpiceDB
   - Implement Delete: Remove tuples from SpiceDB
   - Implement List: Query relationships from SpiceDB
   - Implement Get: Retrieve specific relationship

3. **Add Configuration**
   - Add SpiceDB connection flags to main.go
   - Update Helm chart with SpiceDB configuration
   - Add environment variables for SpiceDB endpoint

4. **Testing**
   - Create e2e tests for Relationship CRUD operations
   - Test integration with SpiceDB
   - Verify authorization flows

## Design Alignment

This implementation follows the design documented in [bin/spicedb-investigation.md](bin/spicedb-investigation.md):

- Supports tuple-based relationship model
- Compatible with SpiceDB schema design
- Enables multi-cluster authorization
- Ready for integration with permission check APIs

## Example Usage (Once Implemented)

### Create Relationship
```bash
kubectl apply -f - <<EOF
apiVersion: multicluster-rbac.open-cluster-management.io/v1alpha1
kind: Relationship
metadata:
  name: user2-cluster1-admin
spec:
  tuples:
  - resource:
      objectType: "resource"
      objectId: "cluster/cluster1/namespace/_wildcard_"
    relation: "admin"
    subject:
      object:
        objectType: "user"
        objectId: "user2"
EOF
```

### List Relationships
```bash
kubectl get relationships
```

### Delete Relationship
```bash
kubectl delete relationship user2-cluster1-admin
```
