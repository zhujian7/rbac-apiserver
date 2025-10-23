package v1alpha1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PermissionBinding represents a set of permissions to bind to a subject.
type PermissionBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the permission bindings
	Spec PermissionBindingSpec `json:"spec,omitempty"`
}

// PermissionBindingSpec defines the desired permission bindings
type PermissionBindingSpec struct {
	// Subject identifies the entity that has the permissions
	Subject rbacv1.Subject `json:"subject"`

	// Permissions is a list of permissions the subject has
	// +listType=atomic
	Permissions []Permission `json:"permissions"`
}

// Permissions represents a single permission
type Permission struct {
	// Resource identifies the object being accessed
	// +listType=atomic
	Resources []string `json:"resources"`

	// Groups is the group of the resources
	// +listType=atomic
	Groups []string `json:"groups"`

	// Namespaces is the namespace of the resource
	// +listType=atomic
	Namespaces []string `json:"namespaces"`

	// Names of the resources
	// +listType=atomic
	Names []string `json:"names"`

	// Role describes the type of role on the resource (e.g., "admin", "viewer", "editor")
	Role string `json:"role"`

	// Cluster is the cluster of the resource
	// +listType=atomic
	Clusters []string `json:"clusters"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PermissionBinding contains a list of PermissionBinding objects
type PermissionBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of PermissionBinding objects
	Items []PermissionBinding `json:"items"`
}
