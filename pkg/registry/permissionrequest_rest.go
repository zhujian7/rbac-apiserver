package registry

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"

	"github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
	"github.com/stolostron/rbac-apiserver/pkg/integration"
)

type PermissionRequestREST struct {
	integration *integration.SpiceDBIntegration
}

// Ensure PermissionRequestREST implements the required interfaces
// PermissionRequest works like CSR - Create only, evaluate immediately and return result
var _ rest.Creater = &PermissionRequestREST{}
var _ rest.Scoper = &PermissionRequestREST{}
var _ rest.Storage = &PermissionRequestREST{}
var _ rest.GroupVersionKindProvider = &PermissionRequestREST{}

func NewPermissionRequestREST(integration *integration.SpiceDBIntegration) *PermissionRequestREST {
	return &PermissionRequestREST{
		integration: integration,
	}
}

func (r *PermissionRequestREST) New() runtime.Object {
	return &v1alpha1.PermissionRequest{}
}

// Create evaluates the PermissionRequest immediately and returns the result with status
// Similar to CertificateSigningRequest, this doesn't persist the object
func (r *PermissionRequestREST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {
	permissionRequest := obj.(*v1alpha1.PermissionRequest)
	permissionRequest.TypeMeta = metav1.TypeMeta{
		APIVersion: v1alpha1.GroupName + "/" + v1alpha1.APIVersion,
		Kind:       "PermissionRequest",
	}

	// Set creation timestamp
	now := metav1.Now()
	permissionRequest.CreationTimestamp = now

	// Extract user information from request context
	userInfo, ok := request.UserFrom(ctx)
	if !ok {
		klog.Warningf("No user information found in context for PermissionRequest %s", permissionRequest.Name)
		return permissionRequest, nil
	}

	userID := userInfo.GetName()
	userType := "user"

	klog.Infof("Evaluating PermissionRequest %s for user %s (from context)", permissionRequest.Name, userID)

	// Evaluate against SpiceDB immediately
	if r.integration != nil {
		if err := r.integration.ProcessPermissionRequestStatus(ctx, permissionRequest, userID, userType); err != nil {
			klog.Errorf("Failed to evaluate PermissionRequest %s with SpiceDB: %v", permissionRequest.Name, err)
			// Return the request with error information in status
			// Could add a condition or error field to status if needed
		} else {
			klog.Infof("Successfully evaluated PermissionRequest %s with status: %+v", permissionRequest.Name, permissionRequest.Status)
		}
	} else {
		klog.Warningf("SpiceDB integration is nil, cannot evaluate PermissionRequest %s", permissionRequest.Name)
	}

	// Return the evaluated result immediately (not persisted)
	return permissionRequest, nil
}

func (r *PermissionRequestREST) NamespaceScoped() bool {
	// PermissionRequests are cluster-scoped
	return false
}

func (r *PermissionRequestREST) GetSingularName() string {
	return "permissionrequest"
}

func (r *PermissionRequestREST) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   containingGV.Group,
		Version: containingGV.Version,
		Kind:    "PermissionRequest",
	}
}

func (r *PermissionRequestREST) Destroy() {
	// Cleanup resources if needed
}
