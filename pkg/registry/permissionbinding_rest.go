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

	"github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
	"github.com/stolostron/rbac-apiserver/pkg/storage"
)

type PermissionBindingREST struct {
	storage *storage.PermissionBindingMemoryStorage
}

// Ensure PermissionBindingREST implements the required interfaces
var _ rest.Creater = &PermissionBindingREST{}
var _ rest.Lister = &PermissionBindingREST{}
var _ rest.Getter = &PermissionBindingREST{}
var _ rest.GracefulDeleter = &PermissionBindingREST{}
var _ rest.Scoper = &PermissionBindingREST{}
var _ rest.Storage = &PermissionBindingREST{}
var _ rest.GroupVersionKindProvider = &PermissionBindingREST{}

func NewPermissionBindingREST() *PermissionBindingREST {
	return &PermissionBindingREST{
		storage: storage.NewPermissionBindingMemoryStorage(),
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
	return r.storage.Create(permissionBinding)
}

func (r *PermissionBindingREST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	// Get the object before deleting for return value
	obj, err := r.storage.Get(name)
	if err != nil {
		return nil, false, err
	}

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
