package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Relationship represents a relationship tuple for SpiceDB-based authorization
// This API is designed to manage relationships between subjects and resources
// following the SpiceDB relationship model.
type Relationship struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the relationship tuples to create or delete
	Spec RelationshipSpec `json:"spec,omitempty"`
}

// RelationshipSpec defines the desired relationship tuples
type RelationshipSpec struct {
	// Tuples is a list of relationship tuples to be created or deleted
	// +listType=atomic
	Tuples []Tuple `json:"tuples"`
}

// Tuple represents a single relationship tuple in SpiceDB format
// It defines a relationship between a subject and a resource
type Tuple struct {
	// Resource identifies the object being accessed
	Resource ObjectReference `json:"resource"`

	// Relation describes the type of relationship (e.g., "admin", "viewer", "editor")
	Relation string `json:"relation"`

	// Subject identifies the entity that has the relationship
	Subject SubjectReference `json:"subject"`
}

// ObjectReference identifies a resource in the system
type ObjectReference struct {
	// ObjectType is the type of the object (e.g., "resource", "namespace", "cluster")
	ObjectType string `json:"objectType"`

	// ObjectId is the identifier for the specific object
	// Format examples:
	//   - "cluster/cluster1/namespace/_wildcard_"
	//   - "cluster/cluster1/namespace/namespace1/core/pods/_wildcard_"
	ObjectId string `json:"objectId"`
}

// SubjectReference identifies the subject of a relationship
type SubjectReference struct {
	// Object identifies the subject
	Object ObjectReference `json:"object"`

	// Relation is an optional relation for the subject (used for group membership, etc.)
	// +optional
	Relation string `json:"relation,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// RelationshipList contains a list of Relationship objects
type RelationshipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of Relationship objects
	Items []Relationship `json:"items"`
}
