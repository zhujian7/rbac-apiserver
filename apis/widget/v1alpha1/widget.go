package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// DefaultNamespace is the fallback namespace when none is specified
	DefaultNamespace = "default"
)

// Widget represents a sample widget resource
type Widget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of Widget
	Spec WidgetSpec `json:"spec,omitempty"`

	// Status defines the observed state of Widget
	Status WidgetStatus `json:"status,omitempty"`
}

// WidgetSpec defines the desired state of Widget
type WidgetSpec struct {
	// Name is the name of the widget
	Name string `json:"name"`

	// Description describes what the widget does
	Description string `json:"description"`

	// Size indicates the size of the widget
	Size int32 `json:"size"`
}

// WidgetStatus defines the observed state of Widget
type WidgetStatus struct {
	// Phase indicates the current phase of the widget
	Phase string `json:"phase,omitempty"`
}

// WidgetList contains a list of Widget
type WidgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of Widget objects
	Items []Widget `json:"items"`
}

func (w *Widget) DeepCopyObject() runtime.Object {
	return &Widget{
		TypeMeta:   w.TypeMeta,
		ObjectMeta: *w.ObjectMeta.DeepCopy(),
		Spec:       w.Spec,
		Status:     w.Status,
	}
}

func (wl *WidgetList) DeepCopyObject() runtime.Object {
	out := &WidgetList{
		TypeMeta: wl.TypeMeta,
		ListMeta: wl.ListMeta,
		Items:    make([]Widget, len(wl.Items)),
	}
	for i := range wl.Items {
		out.Items[i] = *wl.Items[i].DeepCopyObject().(*Widget)
	}
	return out
}
