package integration

import (
	"context"
	"fmt"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"k8s.io/klog/v2"

	rbacv1alpha1 "github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
	"github.com/stolostron/rbac-apiserver/pkg/transformer"
)

// SpiceDBIntegration handles the integration between RBAC API resources and SpiceDB
type SpiceDBIntegration struct {
	transformer       *transformer.SpiceDBTransformer
	permissionsClient v1.PermissionsServiceClient
}

// NewSpiceDBIntegration creates a new SpiceDB integration service
func NewSpiceDBIntegration(permissionsClient v1.PermissionsServiceClient) *SpiceDBIntegration {
	return &SpiceDBIntegration{
		transformer:       transformer.NewSpiceDBTransformer(),
		permissionsClient: permissionsClient,
	}
}

// CreatePermissionBinding handles the creation of a PermissionBinding and syncs it to SpiceDB
func (s *SpiceDBIntegration) CreatePermissionBinding(ctx context.Context, pb *rbacv1alpha1.PermissionBinding) error {
	klog.Infof("Creating PermissionBinding relationships in SpiceDB for %s", pb.Name)

	// Transform PermissionBinding to SpiceDB relationships
	updates, err := s.transformer.TransformPermissionBinding(pb)
	if err != nil {
		return fmt.Errorf("failed to transform PermissionBinding: %w", err)
	}

	// Write relationships to SpiceDB
	err = s.writeRelationships(ctx, updates)
	if err != nil {
		return fmt.Errorf("failed to write relationships to SpiceDB: %w", err)
	}

	klog.Infof("Successfully created %d relationships in SpiceDB for PermissionBinding %s", len(updates), pb.Name)
	return nil
}

// UpdatePermissionBinding handles the update of a PermissionBinding and syncs changes to SpiceDB
func (s *SpiceDBIntegration) UpdatePermissionBinding(ctx context.Context, oldPB, newPB *rbacv1alpha1.PermissionBinding) error {
	klog.Infof("Updating PermissionBinding relationships in SpiceDB for %s", newPB.Name)

	// Delete old relationships
	if oldPB != nil {
		deleteUpdates, err := s.transformer.CreateRelationshipUpdatesForDeletion(oldPB)
		if err != nil {
			return fmt.Errorf("failed to create deletion updates: %w", err)
		}

		err = s.writeRelationships(ctx, deleteUpdates)
		if err != nil {
			klog.Warningf("Failed to delete old relationships for PermissionBinding %s: %v", oldPB.Name, err)
			// Continue with creating new relationships even if deletion fails
		}
	}

	// Create new relationships
	createUpdates, err := s.transformer.TransformPermissionBinding(newPB)
	if err != nil {
		return fmt.Errorf("failed to transform new PermissionBinding: %w", err)
	}

	err = s.writeRelationships(ctx, createUpdates)
	if err != nil {
		return fmt.Errorf("failed to write new relationships to SpiceDB: %w", err)
	}

	klog.Infof("Successfully updated relationships in SpiceDB for PermissionBinding %s", newPB.Name)
	return nil
}

// DeletePermissionBinding handles the deletion of a PermissionBinding and removes relationships from SpiceDB
func (s *SpiceDBIntegration) DeletePermissionBinding(ctx context.Context, pb *rbacv1alpha1.PermissionBinding) error {
	klog.Infof("Deleting PermissionBinding relationships from SpiceDB for %s", pb.Name)

	// Transform to deletion updates
	deleteUpdates, err := s.transformer.CreateRelationshipUpdatesForDeletion(pb)
	if err != nil {
		return fmt.Errorf("failed to create deletion updates: %w", err)
	}

	// Delete relationships from SpiceDB
	err = s.writeRelationships(ctx, deleteUpdates)
	if err != nil {
		return fmt.Errorf("failed to delete relationships from SpiceDB: %w", err)
	}

	klog.Infof("Successfully deleted %d relationships from SpiceDB for PermissionBinding %s", len(deleteUpdates), pb.Name)
	return nil
}

// EvaluatePermissionRequest evaluates a PermissionRequest against SpiceDB
func (s *SpiceDBIntegration) EvaluatePermissionRequest(ctx context.Context, pr *rbacv1alpha1.PermissionRequest, userID, userType string) (*v1.CheckPermissionResponse, error) {
	klog.Infof("Evaluating PermissionRequest %s for user %s", pr.Name, userID)

	// Transform PermissionRequest to SpiceDB check request
	checkReq, err := s.transformer.CheckPermissionFromRequest(pr, userID, userType)
	if err != nil {
		return nil, fmt.Errorf("failed to transform PermissionRequest: %w", err)
	}

	klog.Infof("Checking permission: resourceType=%s, resourceID=%s, permission=%s, subject=%s:%s",
		checkReq.Resource.ObjectType, checkReq.Resource.ObjectId, checkReq.Permission,
		checkReq.Subject.Object.ObjectType, checkReq.Subject.Object.ObjectId)

	// Check permission in SpiceDB for the specific resource
	response, err := s.checkPermission(ctx, checkReq)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission in SpiceDB: %w", err)
	}

	klog.Infof("Permission check result for user %s on resource %s: %v", userID, checkReq.Resource.ObjectId, response.Permissionship)

	// If permission denied and a specific name was requested, also check against wildcard (_ALL_)
	if response.Permissionship != v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION && pr.Spec.Name != "" && pr.Spec.Name != "*" {
		// Build wildcard resource ID by replacing the specific name with _ALL_
		wildcardResourceID := s.transformer.BuildResourceIDWithWildcard(pr.Spec.Cluster, pr.Spec.Namespace, pr.Spec.Resource)

		wildcardCheckReq := &v1.CheckPermissionRequest{
			Resource: &v1.ObjectReference{
				ObjectType: checkReq.Resource.ObjectType,
				ObjectId:   wildcardResourceID,
			},
			Permission: checkReq.Permission,
			Subject:    checkReq.Subject,
		}

		klog.Infof("Checking wildcard permission: resourceType=%s, resourceID=%s", wildcardCheckReq.Resource.ObjectType, wildcardResourceID)
		wildcardResponse, wildcardErr := s.checkPermission(ctx, wildcardCheckReq)
		if wildcardErr != nil {
			klog.Infof("Wildcard check failed: %v", wildcardErr)
			// Return original response if wildcard check fails
			return response, nil
		}

		klog.Infof("Wildcard permission check result: %v", wildcardResponse.Permissionship)
		// Use wildcard response if it grants permission
		if wildcardResponse.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
			return wildcardResponse, nil
		}
	}

	return response, nil
}

// ProcessPermissionRequestStatus updates the status of a PermissionRequest based on SpiceDB evaluation
func (s *SpiceDBIntegration) ProcessPermissionRequestStatus(ctx context.Context, pr *rbacv1alpha1.PermissionRequest, userID, userType string) error {
	klog.Infof("Processing PermissionRequest status for %s, userID=%s, spec=%+v", pr.Name, userID, pr.Spec)

	// Evaluate the permission request
	response, err := s.EvaluatePermissionRequest(ctx, pr, userID, userType)
	if err != nil {
		return fmt.Errorf("failed to evaluate permission request: %w", err)
	}

	klog.Infof("Permission evaluation result for %s: permissionship=%v", pr.Name, response.Permissionship)

	// Update the status based on the response
	// For now, we'll create a simple status based on the permission check
	// In a real implementation, you might want to query multiple permissions
	// and build a more comprehensive allowed list

	if response.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
		// User has permission - add to allowed list
		allowedItem := rbacv1alpha1.AllowedItem{
			Cluster: pr.Spec.Cluster,
			NamespacedNames: []rbacv1alpha1.NamespacedName{
				{
					Namespace: pr.Spec.Namespace,
					Names:     []string{pr.Spec.Name},
				},
			},
		}

		// Update or create the allowed list
		found := false
		for i, item := range pr.Status.AllowedList {
			if item.Cluster == pr.Spec.Cluster {
				// Update existing cluster entry
				namespaceFound := false
				for j, nsName := range item.NamespacedNames {
					if nsName.Namespace == pr.Spec.Namespace {
						// Add to existing namespace entry
						pr.Status.AllowedList[i].NamespacedNames[j].Names = append(pr.Status.AllowedList[i].NamespacedNames[j].Names, pr.Spec.Name)
						namespaceFound = true
						break
					}
				}
				if !namespaceFound {
					pr.Status.AllowedList[i].NamespacedNames = append(pr.Status.AllowedList[i].NamespacedNames, rbacv1alpha1.NamespacedName{
						Namespace: pr.Spec.Namespace,
						Names:     []string{pr.Spec.Name},
					})
				}
				found = true
				break
			}
		}
		if !found {
			pr.Status.AllowedList = append(pr.Status.AllowedList, allowedItem)
		}
	}

	klog.Infof("Successfully processed PermissionRequest status for %s, final status=%+v", pr.Name, pr.Status)
	return nil
}

// writeRelationships writes relationship updates to SpiceDB
func (s *SpiceDBIntegration) writeRelationships(ctx context.Context, updates []*v1.RelationshipUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	// Write relationships using WriteRelationships
	req := &v1.WriteRelationshipsRequest{
		Updates: updates,
	}

	_, err := s.permissionsClient.WriteRelationships(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to write relationships: %w", err)
	}

	return nil
}

// checkPermission checks a permission in SpiceDB
func (s *SpiceDBIntegration) checkPermission(ctx context.Context, checkReq *v1.CheckPermissionRequest) (*v1.CheckPermissionResponse, error) {

	response, err := s.permissionsClient.CheckPermission(ctx, checkReq)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}

	return response, nil
}

// HealthCheck verifies that SpiceDB integration is healthy
func (s *SpiceDBIntegration) HealthCheck(ctx context.Context) error {

	// Check if clients are available
	if s.permissionsClient == nil {
		return fmt.Errorf("SpiceDB permissions client not available")
	}

	// You could add a simple permission check here to verify connectivity
	// For now, just return success if clients are available
	return nil
}
