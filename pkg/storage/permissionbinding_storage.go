package storage

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
)

// RelationshipMemoryStorage provides in-memory storage for Relationship resources
// This storage simulates what will eventually be stored in SpiceDB
type PermissionBindingMemoryStorage struct {
	mu                 sync.RWMutex
	permissionBindings map[string]*v1alpha1.PermissionBinding
	versionCounter     int64
}

// NewRelationshipMemoryStorage creates a new in-memory storage for relationships
func NewPermissionBindingMemoryStorage() *PermissionBindingMemoryStorage {
	return &PermissionBindingMemoryStorage{
		permissionBindings: make(map[string]*v1alpha1.PermissionBinding),
		versionCounter:     1,
	}
}

// Get retrieves a relationship by name
func (s *PermissionBindingMemoryStorage) Get(name string) (*v1alpha1.PermissionBinding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	relationship, exists := s.permissionBindings[name]
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "relationships"},
			name,
		)
	}
	return relationship.DeepCopyObject().(*v1alpha1.PermissionBinding), nil
}

// List retrieves all relationships
func (s *PermissionBindingMemoryStorage) List() (*v1alpha1.PermissionBindingList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := &v1alpha1.PermissionBindingList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupName + "/" + v1alpha1.APIVersion,
			Kind:       "RelationshipList",
		},
		Items: make([]v1alpha1.PermissionBinding, 0, len(s.permissionBindings)),
	}

	for _, relationship := range s.permissionBindings {
		list.Items = append(list.Items, *relationship.DeepCopyObject().(*v1alpha1.PermissionBinding))
	}

	return list, nil
}

// Create creates a new relationship
func (s *PermissionBindingMemoryStorage) Create(relationship *v1alpha1.PermissionBinding) (*v1alpha1.PermissionBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate name if not provided
	if relationship.Name == "" {
		relationship.Name = string(uuid.NewUUID())
	}

	// Check if already exists
	if _, exists := s.permissionBindings[relationship.Name]; exists {
		return nil, errors.NewAlreadyExists(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "relationships"},
			relationship.Name,
		)
	}

	// Set metadata
	now := metav1.NewTime(time.Now())
	relationship.CreationTimestamp = now
	relationship.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++
	relationship.UID = uuid.NewUUID()

	// Store the relationship
	s.permissionBindings[relationship.Name] = relationship.DeepCopyObject().(*v1alpha1.PermissionBinding)
	return relationship, nil
}

// Update updates an existing relationship
func (s *PermissionBindingMemoryStorage) Update(relationship *v1alpha1.PermissionBinding) (*v1alpha1.PermissionBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.permissionBindings[relationship.Name]
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "relationships"},
			relationship.Name,
		)
	}

	// Preserve immutable fields
	relationship.CreationTimestamp = existing.CreationTimestamp
	relationship.UID = existing.UID
	relationship.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++

	// Update the relationship
	s.permissionBindings[relationship.Name] = relationship.DeepCopyObject().(*v1alpha1.PermissionBinding)
	return relationship, nil
}

// Delete removes a relationship
func (s *PermissionBindingMemoryStorage) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.permissionBindings[name]; !exists {
		return errors.NewNotFound(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "relationships"},
			name,
		)
	}

	delete(s.permissionBindings, name)
	return nil
}
