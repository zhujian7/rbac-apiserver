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

type PermissionReviewREST struct {
	integration *integration.SpiceDBIntegration
}

// Ensure PermissionReviewREST implements the required interfaces
// PermissionReview works like SubjectAccessReview - Create only, evaluate immediately and return result
var _ rest.Creater = &PermissionReviewREST{}
var _ rest.Scoper = &PermissionReviewREST{}
var _ rest.Storage = &PermissionReviewREST{}
var _ rest.GroupVersionKindProvider = &PermissionReviewREST{}

func NewPermissionReviewREST(integration *integration.SpiceDBIntegration) *PermissionReviewREST {
	return &PermissionReviewREST{
		integration: integration,
	}
}

func (r *PermissionReviewREST) New() runtime.Object {
	return &v1alpha1.PermissionReview{}
}

// Create evaluates the PermissionReview immediately and returns the result with status
// Similar to SubjectAccessReview, this doesn't persist the object
func (r *PermissionReviewREST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {
	permissionReview := obj.(*v1alpha1.PermissionReview)
	permissionReview.TypeMeta = metav1.TypeMeta{
		APIVersion: v1alpha1.GroupName + "/" + v1alpha1.APIVersion,
		Kind:       "PermissionReview",
	}

	// Set creation timestamp
	now := metav1.Now()
	permissionReview.CreationTimestamp = now

	// Extract user information from request context
	userInfo, ok := request.UserFrom(ctx)
	if !ok {
		klog.Warningf("No user information found in context for PermissionReview %s", permissionReview.Name)
		return permissionReview, nil
	}

	userID := userInfo.GetName()
	userType := "user"

	klog.Infof("Evaluating PermissionReview %s for user %s (from context)", permissionReview.Name, userID)

	// Evaluate against SpiceDB immediately
	if r.integration != nil {
		if err := r.integration.ProcessPermissionReviewStatus(ctx, permissionReview, userID, userType); err != nil {
			klog.Errorf("Failed to evaluate PermissionReview %s with SpiceDB: %v", permissionReview.Name, err)
			// Return the request with error information in status
			// Could add a condition or error field to status if needed
		} else {
			klog.Infof("Successfully evaluated PermissionReview %s with status: %+v", permissionReview.Name, permissionReview.Status)
		}
	} else {
		klog.Warningf("SpiceDB integration is nil, cannot evaluate PermissionReview %s", permissionReview.Name)
	}

	// Return the evaluated result immediately (not persisted)
	return permissionReview, nil
}

func (r *PermissionReviewREST) NamespaceScoped() bool {
	// PermissionReviews are cluster-scoped
	return false
}

func (r *PermissionReviewREST) GetSingularName() string {
	return "permissionreview"
}

func (r *PermissionReviewREST) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   containingGV.Group,
		Version: containingGV.Version,
		Kind:    "PermissionReview",
	}
}

func (r *PermissionReviewREST) Destroy() {
	// Cleanup resources if needed
}
