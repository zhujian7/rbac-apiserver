package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PermissionBinding represents a set of permissions to bind to a subject.
type PermissionRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the relationship tuples to create or delete
	Spec PermissionRequestSpec `json:"spec,omitempty"`

	Status PermissionRequestStatus `json:"status,omitempty"`
}

type PermissionRequestSpec struct {
	// grou is the group of the resource
	Group string `json:"group"`

	// resource is the resource type
	Resource string `json:"resource"`

	// subResource is the sub resource of the resource type
	SubResource string `json:"subResource,omitempty"`

	// verb is the action to the resource
	Verb string `json:"verb"`

	// cluster name
	Cluster string `json:"cluster,omitempty"`

	// namespace
	Namespace string `json:"namespace,omitempty"`

	// name
	Name string `json:"name,omitempty"`
}

type PermissionRequestStatus struct {
	// allowedList is a list of
	// +listType=map
	// +listMapKey=cluster
	AllowedList []AllowedItem `json:"allowedList,omitempty"`
}

type AllowedItem struct {
	Cluster string `json:"cluster,omitempty"`

	// +listType=map
	// +listMapKey=namespace
	NamespacedNames []NamespacedName `json:"namespacedNames"`
}

type NamespacedName struct {
	// +listType=atomic
	Names []string `json:"names,omitempty"`

	Namespace string `json:"namespace"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PermissionRequest contains a list of PermissionRequest objects
type PermissionRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of Relationship objects
	Items []PermissionRequest `json:"items"`
}
