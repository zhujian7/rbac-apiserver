package registry

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	sampleapi "github.com/stolostron/rbac-apiserver/apis/sample"
	"github.com/stolostron/rbac-apiserver/apis/sample/v1alpha1"
	"github.com/stolostron/rbac-apiserver/pkg/storage"
)

type WidgetREST struct {
	storage *storage.MemoryStorage
}

// Ensure WidgetREST implements the required interfaces
var _ rest.Creater = &WidgetREST{}
var _ rest.Lister = &WidgetREST{}
var _ rest.Getter = &WidgetREST{}
var _ rest.Updater = &WidgetREST{}
var _ rest.GracefulDeleter = &WidgetREST{}
var _ rest.Scoper = &WidgetREST{}
var _ rest.Storage = &WidgetREST{}
var _ rest.GroupVersionKindProvider = &WidgetREST{}

func NewWidgetREST() *WidgetREST {
	return &WidgetREST{
		storage: storage.NewMemoryStorage(),
	}
}

func (r *WidgetREST) New() runtime.Object {
	return &v1alpha1.Widget{}
}

func (r *WidgetREST) NewList() runtime.Object {
	return &v1alpha1.WidgetList{}
}

func (r *WidgetREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	var namespace string

	// Get namespace from request context
	requestInfo, ok := request.RequestInfoFrom(ctx)
	if ok && requestInfo.Namespace != "" {
		namespace = requestInfo.Namespace
	} else {
		// Check if the name contains namespace info (format: namespace/name)
		if strings.Contains(name, "/") {
			namespace, name = r.storage.ParseKey(name)
		} else {
			namespace = v1alpha1.DefaultNamespace // fallback
		}
	}

	return r.storage.Get(namespace, name)
}

func (r *WidgetREST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	var namespace string

	// Get namespace from request context
	requestInfo, ok := request.RequestInfoFrom(ctx)
	if ok && requestInfo.Namespace != "" {
		namespace = requestInfo.Namespace
	}
	// If no namespace specified, list all (namespace = "")

	return r.storage.List(namespace)
}

func (r *WidgetREST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {
	widget := obj.(*v1alpha1.Widget)
	widget.TypeMeta = metav1.TypeMeta{
		APIVersion: sampleapi.GroupName + "/" + v1alpha1.APIVersion,
		Kind:       "Widget",
	}
	return r.storage.Create(widget)
}

func (r *WidgetREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {

	var namespace string

	// Get namespace from request context (this is how Kubernetes passes the namespace)
	requestInfo, ok := request.RequestInfoFrom(ctx)
	if ok && requestInfo.Namespace != "" {
		namespace = requestInfo.Namespace
	} else {
		// Check if the name contains namespace info (format: namespace/name)
		if strings.Contains(name, "/") {
			namespace, name = r.storage.ParseKey(name)
		} else {
			namespace = v1alpha1.DefaultNamespace // fallback
		}
	}

	oldObj, err := r.storage.Get(namespace, name)
	if err != nil {
		return nil, false, err
	}

	updatedObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}

	widget := updatedObj.(*v1alpha1.Widget)
	widget.Name = name
	widget.Namespace = namespace
	updatedWidget, err := r.storage.Update(widget)
	return updatedWidget, false, err
}

func (r *WidgetREST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions) (runtime.Object, bool, error) {

	var namespace string

	// Get namespace from request context (this is how Kubernetes passes the namespace)
	requestInfo, ok := request.RequestInfoFrom(ctx)
	if ok && requestInfo.Namespace != "" {
		namespace = requestInfo.Namespace
	} else {
		// Check if the name contains namespace info (format: namespace/name)
		if strings.Contains(name, "/") {
			namespace, name = r.storage.ParseKey(name)
		} else {
			namespace = v1alpha1.DefaultNamespace // fallback
		}
	}

	obj, err := r.storage.Get(namespace, name)
	if err != nil {
		return nil, false, err
	}

	err = r.storage.Delete(namespace, name)
	return obj, true, err
}

func (r *WidgetREST) Watch(ctx context.Context, options *metav1.ListOptions) (watch.Interface, error) {
	return nil, fmt.Errorf("watch not implemented")
}

func (r *WidgetREST) ConvertToTable(ctx context.Context, object runtime.Object,
	tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(schema.GroupResource{Group: sampleapi.GroupName, Resource: "widgets"}).
		ConvertToTable(ctx, object, tableOptions)
}

func (r *WidgetREST) NamespaceScoped() bool {
	return true
}

func (r *WidgetREST) GetSingularName() string {
	return "widget"
}

func (r *WidgetREST) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   containingGV.Group,
		Version: containingGV.Version,
		Kind:    "Widget",
	}
}

func (r *WidgetREST) Destroy() {
	// Cleanup resources if needed
}
