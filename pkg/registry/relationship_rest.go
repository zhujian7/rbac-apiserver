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

	rbacapi "github.com/stolostron/rbac-apiserver/apis/rbac"
	"github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
	"github.com/stolostron/rbac-apiserver/pkg/storage"
)

type RelationshipREST struct {
	storage *storage.RelationshipMemoryStorage
}

// Ensure RelationshipREST implements the required interfaces
var _ rest.Creater = &RelationshipREST{}
var _ rest.Lister = &RelationshipREST{}
var _ rest.Getter = &RelationshipREST{}
var _ rest.GracefulDeleter = &RelationshipREST{}
var _ rest.Scoper = &RelationshipREST{}
var _ rest.Storage = &RelationshipREST{}
var _ rest.GroupVersionKindProvider = &RelationshipREST{}

func NewRelationshipREST() *RelationshipREST {
	return &RelationshipREST{
		storage: storage.NewRelationshipMemoryStorage(),
	}
}

func (r *RelationshipREST) New() runtime.Object {
	return &v1alpha1.Relationship{}
}

func (r *RelationshipREST) NewList() runtime.Object {
	return &v1alpha1.RelationshipList{}
}

func (r *RelationshipREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.storage.Get(name)
}

func (r *RelationshipREST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	return r.storage.List()
}

func (r *RelationshipREST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {
	relationship := obj.(*v1alpha1.Relationship)
	relationship.TypeMeta = metav1.TypeMeta{
		APIVersion: rbacapi.GroupName + "/" + v1alpha1.APIVersion,
		Kind:       "Relationship",
	}
	return r.storage.Create(relationship)
}

func (r *RelationshipREST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	// Get the object before deleting for return value
	obj, err := r.storage.Get(name)
	if err != nil {
		return nil, false, err
	}

	err = r.storage.Delete(name)
	return obj, true, err
}

func (r *RelationshipREST) Watch(ctx context.Context, options *metav1.ListOptions) (watch.Interface, error) {
	return nil, fmt.Errorf("watch not implemented")
}

func (r *RelationshipREST) ConvertToTable(ctx context.Context, object runtime.Object,
	tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(schema.GroupResource{Group: rbacapi.GroupName, Resource: "relationships"}).
		ConvertToTable(ctx, object, tableOptions)
}

func (r *RelationshipREST) NamespaceScoped() bool {
	// Relationships are cluster-scoped as they manage cross-cluster permissions
	return false
}

func (r *RelationshipREST) GetSingularName() string {
	return "relationship"
}

func (r *RelationshipREST) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   containingGV.Group,
		Version: containingGV.Version,
		Kind:    "Relationship",
	}
}

func (r *RelationshipREST) Destroy() {
	// Cleanup resources if needed
}
