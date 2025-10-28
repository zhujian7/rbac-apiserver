package transformer

import (
	"fmt"
	"regexp"
	"strings"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	rbacv1alpha1 "github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
)

// validObjectIDRegex matches SpiceDB's validation pattern for ObjectID
// Pattern: ^(([a-zA-Z0-9/_|\-=+]{1,})|\\*)$
var invalidObjectIDChars = regexp.MustCompile(`[^a-zA-Z0-9/_|\-=+*]`)

// sanitizeObjectID removes or replaces characters that are invalid for SpiceDB ObjectIDs
// SpiceDB only allows: a-zA-Z0-9/_|\-=+ and *
func sanitizeObjectID(id string) string {
	if id == "" {
		return ""
	}
	// Replace invalid characters with underscore
	sanitized := invalidObjectIDChars.ReplaceAllString(id, "_")
	// Ensure it's not empty after sanitization
	if sanitized == "" {
		return "invalid"
	}
	return sanitized
}

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
	resourceID := t.buildResourceID(pr.Spec.Cluster, pr.Spec.Namespace, pr.Spec.Resource, pr.Spec.Name)

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

	var objectType, objectID, optionalRelation string

	switch subject.Kind {
	case "User":
		objectType = "user"
		objectID = sanitizeObjectID(subject.Name)
		// Users don't need an optional relation
	case "Group":
		objectType = "group"
		objectID = sanitizeObjectID(subject.Name)
		// Groups require the "member" relation to reference group members
		optionalRelation = "member"
	default:
		return nil, fmt.Errorf("unsupported subject kind: %s", subject.Kind)
	}

	subjectRef := &v1.SubjectReference{
		Object: &v1.ObjectReference{
			ObjectType: objectType,
			ObjectId:   objectID,
		},
	}

	// Set optional relation if specified (e.g., for groups)
	if optionalRelation != "" {
		subjectRef.OptionalRelation = optionalRelation
	}

	return subjectRef, nil
}

// convertPermission converts a Permission to SpiceDB relationship updates
func (t *SpiceDBTransformer) convertPermission(subject *v1.SubjectReference, permission rbacv1alpha1.Permission) ([]*v1.RelationshipUpdate, error) {
	// Use a map to deduplicate relationships based on their string representation
	// Key format: "resourceType:resourceID#relation@subjectType:subjectID"
	uniqueRelationships := make(map[string]*v1.RelationshipUpdate)

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
					resourceID := t.buildResourceID(cluster, namespace, resource, name)
					relation := t.mapRoleToRelation(permission.Role)

					// Create a unique key for this relationship
					subjectRelation := ""
					if subject.OptionalRelation != "" {
						subjectRelation = "#" + subject.OptionalRelation
					}
					relationshipKey := fmt.Sprintf("%s:%s#%s@%s:%s%s",
						resourceType, resourceID, relation,
						subject.Object.ObjectType, subject.Object.ObjectId, subjectRelation)

					// Only add if we haven't seen this exact relationship before
					if _, exists := uniqueRelationships[relationshipKey]; !exists {
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
						uniqueRelationships[relationshipKey] = update
					}
				}
			}
		}
	}

	// Convert map to slice
	var updates []*v1.RelationshipUpdate
	for _, update := range uniqueRelationships {
		updates = append(updates, update)
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
func (t *SpiceDBTransformer) buildResourceID(cluster, namespace, resource, name string) string {
	var parts []string

	// Handle complete wildcard case - SpiceDB requires alphanumeric ObjectId
	if cluster == "*" && namespace == "*" && resource == "*" && name == "*" {
		return "all"
	}

	if cluster != "" && cluster != "*" {
		sanitized := sanitizeObjectID(cluster)
		if sanitized != "" {
			parts = append(parts, "cluster", sanitized)
		}
	}

	if namespace != "" && namespace != "*" {
		sanitized := sanitizeObjectID(namespace)
		if sanitized != "" {
			parts = append(parts, "namespace", sanitized)
		}
	}

	if resource != "" && resource != "*" {
		sanitized := sanitizeObjectID(resource)
		if sanitized != "" {
			parts = append(parts, sanitized)
		}
	}

	if name != "" && name != "*" {
		sanitized := sanitizeObjectID(name)
		if sanitized != "" {
			parts = append(parts, sanitized)
		}
	} else if name == "*" && resource != "" && resource != "*" {
		// When name is wildcard but resource is specified, use _ALL_
		parts = append(parts, "_ALL_")
	}

	if len(parts) == 0 {
		return "all" // Wildcard for all resources (SpiceDB requires alphanumeric)
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

	// Sanitize userID to ensure it's valid for SpiceDB
	sanitizedUserID := sanitizeObjectID(userID)
	if sanitizedUserID == "" {
		return nil, fmt.Errorf("invalid userID: cannot be empty after sanitization")
	}

	checkReq.Subject = &v1.SubjectReference{
		Object: &v1.ObjectReference{
			ObjectType: subjectType,
			ObjectId:   sanitizedUserID,
		},
	}

	return checkReq, nil
}

// BuildResourceIDWithWildcard builds a resource ID with _ALL_ for wildcard name matching
func (t *SpiceDBTransformer) BuildResourceIDWithWildcard(cluster, namespace, resource string) string {
	return t.buildResourceID(cluster, namespace, resource, "*")
}
