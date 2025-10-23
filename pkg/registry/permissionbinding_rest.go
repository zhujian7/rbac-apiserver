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

type PermissionBindingREST struct {
	storage     *storage.PermissionBindingMemoryStorage
	integration *integration.SpiceDBIntegration
}

// Ensure PermissionBindingREST implements the required interfaces
var _ rest.Creater = &PermissionBindingREST{}
var _ rest.Lister = &PermissionBindingREST{}
var _ rest.Getter = &PermissionBindingREST{}
var _ rest.GracefulDeleter = &PermissionBindingREST{}
var _ rest.Scoper = &PermissionBindingREST{}
var _ rest.Storage = &PermissionBindingREST{}
var _ rest.GroupVersionKindProvider = &PermissionBindingREST{}

func NewPermissionBindingREST(integration *integration.SpiceDBIntegration) *PermissionBindingREST {
	return &PermissionBindingREST{
		storage:     storage.NewPermissionBindingMemoryStorage(),
		integration: integration,
	}
}

func (r *PermissionBindingREST) New() runtime.Object {
	return &v1alpha1.PermissionBinding{}
}

func (r *PermissionBindingREST) NewList() runtime.Object {
	return &v1alpha1.PermissionBindingList{}
}

func (r *PermissionBindingREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.storage.Get(name)
}

func (r *PermissionBindingREST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	return r.storage.List()
}

func (r *PermissionBindingREST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {
	permissionBinding := obj.(*v1alpha1.PermissionBinding)
	permissionBinding.TypeMeta = metav1.TypeMeta{
		APIVersion: v1alpha1.GroupName + "/" + v1alpha1.APIVersion,
		Kind:       "PermissionBinding",
	}
	
	// Create in storage first
	createdObj, err := r.storage.Create(permissionBinding)
	if err != nil {
		return nil, err
	}
	
	// Sync to SpiceDB
	if r.integration != nil {
		if err := r.integration.CreatePermissionBinding(ctx, permissionBinding); err != nil {
			klog.Errorf("Failed to sync PermissionBinding %s to SpiceDB: %v", permissionBinding.Name, err)
			// Note: We don't fail the creation if SpiceDB sync fails
			// The object exists in storage, but authorization might not work correctly
		}
	}
	
	return createdObj, nil
}

func (r *PermissionBindingREST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	// Get the object before deleting for return value
	obj, err := r.storage.Get(name)
	if err != nil {
		return nil, false, err
	}

	permissionBinding := obj.(*v1alpha1.PermissionBinding)
	
	// Delete from SpiceDB first
	if r.integration != nil {
		if err := r.integration.DeletePermissionBinding(ctx, permissionBinding); err != nil {
			klog.Errorf("Failed to delete PermissionBinding %s from SpiceDB: %v", name, err)
			// Continue with storage deletion even if SpiceDB deletion fails
		}
	}
	
	// Delete from storage
	err = r.storage.Delete(name)
	return obj, true, err
}

func (r *PermissionBindingREST) Watch(ctx context.Context, options *metav1.ListOptions) (watch.Interface, error) {
	return nil, fmt.Errorf("watch not implemented")
}

func (r *PermissionBindingREST) ConvertToTable(ctx context.Context, object runtime.Object,
	tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionbindings"}).
		ConvertToTable(ctx, object, tableOptions)
}

func (r *PermissionBindingREST) NamespaceScoped() bool {
	// PermissionBindings are cluster-scoped as they manage cross-cluster permissions
	return false
}

func (r *PermissionBindingREST) GetSingularName() string {
	return "permissionbinding"
}

func (r *PermissionBindingREST) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   containingGV.Group,
		Version: containingGV.Version,
		Kind:    "PermissionBinding",
	}
}

func (r *PermissionBindingREST) Destroy() {
	// Cleanup resources if needed
}
