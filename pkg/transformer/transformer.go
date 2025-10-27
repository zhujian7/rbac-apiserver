package transformer

import (
	"fmt"
	"strings"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	rbacv1alpha1 "github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
)

// SpiceDBTransformer handles conversion between RBAC API resources and SpiceDB relationships
type SpiceDBTransformer struct{}

// NewSpiceDBTransformer creates a new transformer instance
func NewSpiceDBTransformer() *SpiceDBTransformer {
	return &SpiceDBTransformer{}
}

// TransformPermissionBinding converts a PermissionBinding to SpiceDB relationship updates
func (t *SpiceDBTransformer) TransformPermissionBinding(pb *rbacv1alpha1.PermissionBinding) ([]*v1.RelationshipUpdate, error) {
	if pb == nil {
		return nil, fmt.Errorf("PermissionBinding cannot be nil")
	}

	var updates []*v1.RelationshipUpdate

	// Convert subject to SpiceDB subject reference
	subjectRef, err := t.convertSubject(&pb.Spec.Subject)
	if err != nil {
		return nil, fmt.Errorf("failed to convert subject: %w", err)
	}

	// Process each permission in the binding
	for _, permission := range pb.Spec.Permissions {
		permissionUpdates, err := t.convertPermission(subjectRef, permission)
		if err != nil {
			return nil, fmt.Errorf("failed to convert permission: %w", err)
		}
		updates = append(updates, permissionUpdates...)
	}

	return updates, nil
}

// TransformPermissionRequest evaluates a PermissionRequest against SpiceDB
// This returns the resource patterns that should be checked
func (t *SpiceDBTransformer) TransformPermissionRequest(pr *rbacv1alpha1.PermissionRequest) (*v1.CheckPermissionRequest, error) {
	if pr == nil {
		return nil, fmt.Errorf("PermissionRequest cannot be nil")
	}

	// Build resource reference
	resourceType := t.mapResourceType(pr.Spec.Resource)
	resourceID := t.buildResourceID(pr.Spec.Cluster, pr.Spec.Namespace, pr.Spec.Name)

	// Map verb to permission
	permission := t.mapVerbToPermission(pr.Spec.Verb)

	// For PermissionRequest, we need to know who is making the request
	// This would typically come from the request context
	// For now, we'll create a placeholder that can be filled in by the caller
	return &v1.CheckPermissionRequest{
		Resource: &v1.ObjectReference{
			ObjectType: resourceType,
			ObjectId:   resourceID,
		},
		Permission: permission,
		// Subject will need to be filled in by the caller based on request context
		Subject: nil,
	}, nil
}

// convertSubject converts a Kubernetes RBAC subject to SpiceDB subject reference
func (t *SpiceDBTransformer) convertSubject(subject *rbacv1.Subject) (*v1.SubjectReference, error) {
	if subject == nil {
		return nil, fmt.Errorf("subject cannot be nil")
	}

	var objectType, objectID string

	switch subject.Kind {
	case "User":
		objectType = "user"
		objectID = subject.Name
	case "Group":
		objectType = "group"
		objectID = subject.Name
	default:
		return nil, fmt.Errorf("unsupported subject kind: %s", subject.Kind)
	}

	return &v1.SubjectReference{
		Object: &v1.ObjectReference{
			ObjectType: objectType,
			ObjectId:   objectID,
		},
	}, nil
}

// convertPermission converts a Permission to SpiceDB relationship updates
func (t *SpiceDBTransformer) convertPermission(subject *v1.SubjectReference, permission rbacv1alpha1.Permission) ([]*v1.RelationshipUpdate, error) {
	var updates []*v1.RelationshipUpdate

	// Generate relationships for each combination of clusters, namespaces, and resources
	clusters := permission.Clusters
	if len(clusters) == 0 {
		clusters = []string{"*"} // Default to all clusters if none specified
	}

	namespaces := permission.Namespaces
	if len(namespaces) == 0 {
		namespaces = []string{"*"} // Default to all namespaces if none specified
	}

	resources := permission.Resources
	if len(resources) == 0 {
		resources = []string{"*"} // Default to all resources if none specified
	}

	names := permission.Names
	if len(names) == 0 {
		names = []string{"*"} // Default to all names if none specified
	}

	// Create relationships for each combination
	for _, cluster := range clusters {
		for _, namespace := range namespaces {
			for _, resource := range resources {
				for _, name := range names {
					resourceType := t.mapResourceType(resource)
					resourceID := t.buildResourceID(cluster, namespace, name)
					relation := t.mapRoleToRelation(permission.Role)

					update := &v1.RelationshipUpdate{
						Operation: v1.RelationshipUpdate_OPERATION_CREATE,
						Relationship: &v1.Relationship{
							Resource: &v1.ObjectReference{
								ObjectType: resourceType,
								ObjectId:   resourceID,
							},
							Relation: relation,
							Subject:  subject,
						},
					}
					updates = append(updates, update)
				}
			}
		}
	}

	return updates, nil
}

// mapResourceType maps Kubernetes resource types to SpiceDB object types
func (t *SpiceDBTransformer) mapResourceType(resource string) string {
	// Map common Kubernetes resources to our SpiceDB schema
	switch strings.ToLower(resource) {
	case "clusters", "cluster":
		return "cluster"
	case "namespaces", "namespace":
		return "namespace"
	case "pods", "pod":
		return "resource"
	case "services", "service":
		return "resource"
	case "deployments", "deployment":
		return "resource"
	case "configmaps", "configmap":
		return "resource"
	case "secrets", "secret":
		return "resource"
	default:
		// Default to generic resource type
		return "resource"
	}
}

// mapRoleToRelation maps role names to SpiceDB relations
func (t *SpiceDBTransformer) mapRoleToRelation(role string) string {
	switch strings.ToLower(role) {
	case "admin", "administrator":
		return "admin"
	case "viewer", "view", "read":
		return "viewer"
	case "editor", "edit", "write":
		return "editor"
	default:
		// Default to viewer for unknown roles
		return "viewer"
	}
}

// mapVerbToPermission maps Kubernetes verbs to SpiceDB permissions
func (t *SpiceDBTransformer) mapVerbToPermission(verb string) string {
	switch strings.ToLower(verb) {
	case "get", "list", "watch":
		return "view"
	case "create", "update", "patch", "delete":
		return "edit"
	case "*":
		return "edit" // Full access maps to edit
	default:
		return "view" // Default to view permission
	}
}

// buildResourceID creates a hierarchical resource ID for SpiceDB
func (t *SpiceDBTransformer) buildResourceID(cluster, namespace, name string) string {
	var parts []string

	if cluster != "" && cluster != "*" {
		parts = append(parts, "cluster", cluster)
	}

	if namespace != "" && namespace != "*" {
		parts = append(parts, "namespace", namespace)
	}

	if name != "" && name != "*" {
		parts = append(parts, "name", name)
	}

	if len(parts) == 0 {
		return "*" // Wildcard for all resources
	}

	return strings.Join(parts, "/")
}

// CreateRelationshipUpdatesForDeletion creates relationship updates to delete all relationships for a PermissionBinding
func (t *SpiceDBTransformer) CreateRelationshipUpdatesForDeletion(pb *rbacv1alpha1.PermissionBinding) ([]*v1.RelationshipUpdate, error) {
	// Get the create updates first
	createUpdates, err := t.TransformPermissionBinding(pb)
	if err != nil {
		return nil, err
	}

	// Convert all CREATE operations to DELETE operations
	var deleteUpdates []*v1.RelationshipUpdate
	for _, update := range createUpdates {
		deleteUpdate := &v1.RelationshipUpdate{
			Operation:    v1.RelationshipUpdate_OPERATION_DELETE,
			Relationship: update.Relationship,
		}
		deleteUpdates = append(deleteUpdates, deleteUpdate)
	}

	return deleteUpdates, nil
}

// CheckPermissionFromRequest checks if a permission request should be allowed
// This is a helper method that creates the check request with a subject
func (t *SpiceDBTransformer) CheckPermissionFromRequest(pr *rbacv1alpha1.PermissionRequest, userID string, userType string) (*v1.CheckPermissionRequest, error) {
	checkReq, err := t.TransformPermissionRequest(pr)
	if err != nil {
		return nil, err
	}

	// Fill in the subject based on the user making the request
	subjectType := "user"
	if userType != "" {
		subjectType = userType
	}

	checkReq.Subject = &v1.SubjectReference{
		Object: &v1.ObjectReference{
			ObjectType: subjectType,
			ObjectId:   userID,
		},
	}

	return checkReq, nil
}
