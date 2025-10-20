# Relationship API Implementation

## Overview

This document describes the Relationship API framework that has been added to the rbac-apiserver. This API is designed to manage SpiceDB-based authorization relationships for multi-cluster RBAC scenarios.

## API Details

### API Group and Version
- **API Group**: `rbac.open-cluster-management.io`
- **Version**: `v1alpha1`
- **Resource**: `relationships`
- **Full API Path**: `/apis/rbac.open-cluster-management.io/v1alpha1/relationships`

### Resource Types

#### Relationship
Represents a relationship tuple for SpiceDB-based authorization.

```yaml
apiVersion: rbac.open-cluster-management.io/v1alpha1
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

### ✅ Completed
1. **API Types Defined** ([apis/rbac/v1alpha1/relationship.go](apis/rbac/v1alpha1/relationship.go))
   - Relationship and RelationshipList types
   - Tuple, ObjectReference, SubjectReference types
   - DeepCopy methods for runtime.Object interface

2. **REST Storage Registry** ([pkg/registry/relationship_rest.go](pkg/registry/relationship_rest.go))
   - Implements required Kubernetes storage interfaces:
     - rest.Creater
     - rest.Lister
     - rest.Getter
     - rest.GracefulDeleter
     - rest.Scoper
     - rest.Storage
     - rest.GroupVersionKindProvider
   - Resource is **cluster-scoped** (not namespaced)

3. **API Registration** ([cmd/main.go](cmd/main.go))
   - Added rbac API group imports
   - Registered Relationship types in scheme
   - Installed API group in installAPI function
   - Supports both OpenAPI v2 and v3

4. **OpenAPI Generation** ([hack/update-openapi.sh](hack/update-openapi.sh))
   - Updated to include rbac/v1alpha1 package
   - Generated OpenAPI specs include Relationship types
   - Verified in [apis/generated/openapi/zz_generated.openapi.go](apis/generated/openapi/zz_generated.openapi.go)

5. **Build Verification**
   - Project builds successfully
   - Binary size: ~67MB
   - No compilation errors

### 🚧 Not Yet Implemented (Framework Only)

The following operations return "not implemented" errors and are ready for SpiceDB integration:

1. **Create Operation**:
   - Endpoint: `POST /apis/rbac.open-cluster-management.io/v1alpha1/relationships`
   - TODO: Integrate with SpiceDB to create relationship tuples

2. **Delete Operation**:
   - Endpoint: `DELETE /apis/rbac.open-cluster-management.io/v1alpha1/relationships/{name}`
   - TODO: Integrate with SpiceDB to delete relationship tuples

3. **List Operation**:
   - Endpoint: `GET /apis/rbac.open-cluster-management.io/v1alpha1/relationships`
   - Currently returns empty list
   - TODO: Query SpiceDB for existing relationships

4. **Get Operation**:
   - Endpoint: `GET /apis/rbac.open-cluster-management.io/v1alpha1/relationships/{name}`
   - TODO: Retrieve specific relationship from SpiceDB

## Files Created/Modified

### New Files
- `apis/rbac/doc.go` - Package constants
- `apis/rbac/v1alpha1/doc.go` - API version constants
- `apis/rbac/v1alpha1/relationship.go` - Type definitions
- `pkg/registry/relationship_rest.go` - REST storage implementation

### Modified Files
- `cmd/main.go` - Added Relationship API registration
- `hack/update-openapi.sh` - Added rbac/v1alpha1 to OpenAPI generation
- `apis/generated/openapi/zz_generated.openapi.go` - Auto-generated OpenAPI specs

## Next Steps for Implementation

To complete the Relationship API implementation, you'll need to:

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
apiVersion: rbac.open-cluster-management.io/v1alpha1
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
