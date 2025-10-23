package registry

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"

	"github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
	"github.com/stolostron/rbac-apiserver/pkg/integration"
	"github.com/stolostron/rbac-apiserver/pkg/storage"
)

type PermissionRequestREST struct {
	storage     *storage.PermissionRequestMemoryStorage
	integration *integration.SpiceDBIntegration
}

// Ensure PermissionRequestREST implements the required interfaces
var _ rest.Creater = &PermissionRequestREST{}
var _ rest.Lister = &PermissionRequestREST{}
var _ rest.Getter = &PermissionRequestREST{}
var _ rest.GracefulDeleter = &PermissionRequestREST{}
var _ rest.Scoper = &PermissionRequestREST{}
var _ rest.Storage = &PermissionRequestREST{}
var _ rest.GroupVersionKindProvider = &PermissionRequestREST{}

func NewPermissionRequestREST(integration *integration.SpiceDBIntegration) *PermissionRequestREST {
	return &PermissionRequestREST{
		storage:     storage.NewPermissionRequestMemoryStorage(),
		integration: integration,
	}
}

func (r *PermissionRequestREST) New() runtime.Object {
	return &v1alpha1.PermissionRequest{}
}

func (r *PermissionRequestREST) NewList() runtime.Object {
	return &v1alpha1.PermissionRequestList{}
}

func (r *PermissionRequestREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.storage.Get(name)
}

func (r *PermissionRequestREST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	return r.storage.List()
}

func (r *PermissionRequestREST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {
	permissionRequest := obj.(*v1alpha1.PermissionRequest)
	permissionRequest.TypeMeta = metav1.TypeMeta{
		APIVersion: v1alpha1.GroupName + "/" + v1alpha1.APIVersion,
		Kind:       "PermissionRequest",
	}
	
	// Create in storage first
	createdObj, err := r.storage.Create(permissionRequest)
	if err != nil {
		return nil, err
	}
	
	// Evaluate against SpiceDB and update status
	if r.integration != nil {
		// For now, we'll use a placeholder user ID and type
		// In a real implementation, this would come from the request context
		userID := "system:admin" // TODO: Extract from request context
		userType := "user"
		
		if err := r.integration.ProcessPermissionRequestStatus(ctx, permissionRequest, userID, userType); err != nil {
			klog.Errorf("Failed to process PermissionRequest %s status with SpiceDB: %v", permissionRequest.Name, err)
			// Note: We don't fail the creation if SpiceDB evaluation fails
			// The object exists but status might not be accurate
		} else {
			// Update the object in storage with the new status
			if updatedObj, updateErr := r.storage.Update(permissionRequest); updateErr != nil {
				klog.Errorf("Failed to update PermissionRequest %s status in storage: %v", permissionRequest.Name, updateErr)
			} else {
				createdObj = updatedObj
			}
		}
	}
	
	return createdObj, nil
}

func (r *PermissionRequestREST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	// Get the object before deleting for return value
	obj, err := r.storage.Get(name)
	if err != nil {
		return nil, false, err
	}

	err = r.storage.Delete(name)
	return obj, true, err
}

func (r *PermissionRequestREST) Watch(ctx context.Context, options *metav1.ListOptions) (watch.Interface, error) {
	return nil, fmt.Errorf("watch not implemented")
}

func (r *PermissionRequestREST) ConvertToTable(ctx context.Context, object runtime.Object,
	tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionrequests"}).
		ConvertToTable(ctx, object, tableOptions)
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