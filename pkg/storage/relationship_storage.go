package storage

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"

	rbacapi "github.com/stolostron/rbac-apiserver/apis/rbac"
	"github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
)

// RelationshipMemoryStorage provides in-memory storage for Relationship resources
// This storage simulates what will eventually be stored in SpiceDB
type RelationshipMemoryStorage struct {
	mu             sync.RWMutex
	relationships  map[string]*v1alpha1.Relationship
	versionCounter int64
}

// NewRelationshipMemoryStorage creates a new in-memory storage for relationships
func NewRelationshipMemoryStorage() *RelationshipMemoryStorage {
	return &RelationshipMemoryStorage{
		relationships:  make(map[string]*v1alpha1.Relationship),
		versionCounter: 1,
	}
}

// Get retrieves a relationship by name
func (s *RelationshipMemoryStorage) Get(name string) (*v1alpha1.Relationship, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	relationship, exists := s.relationships[name]
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: rbacapi.GroupName, Resource: "relationships"},
			name,
		)
	}
	return relationship.DeepCopyObject().(*v1alpha1.Relationship), nil
}

// List retrieves all relationships
func (s *RelationshipMemoryStorage) List() (*v1alpha1.RelationshipList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := &v1alpha1.RelationshipList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacapi.GroupName + "/" + v1alpha1.APIVersion,
			Kind:       "RelationshipList",
		},
		Items: make([]v1alpha1.Relationship, 0, len(s.relationships)),
	}

	for _, relationship := range s.relationships {
		list.Items = append(list.Items, *relationship.DeepCopyObject().(*v1alpha1.Relationship))
	}

	return list, nil
}

// Create creates a new relationship
func (s *RelationshipMemoryStorage) Create(relationship *v1alpha1.Relationship) (*v1alpha1.Relationship, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate name if not provided
	if relationship.Name == "" {
		relationship.Name = string(uuid.NewUUID())
	}

	// Check if already exists
	if _, exists := s.relationships[relationship.Name]; exists {
		return nil, errors.NewAlreadyExists(
			schema.GroupResource{Group: rbacapi.GroupName, Resource: "relationships"},
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
	s.relationships[relationship.Name] = relationship.DeepCopyObject().(*v1alpha1.Relationship)
	return relationship, nil
}

// Update updates an existing relationship
func (s *RelationshipMemoryStorage) Update(relationship *v1alpha1.Relationship) (*v1alpha1.Relationship, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.relationships[relationship.Name]
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: rbacapi.GroupName, Resource: "relationships"},
			relationship.Name,
		)
	}

	// Preserve immutable fields
	relationship.CreationTimestamp = existing.CreationTimestamp
	relationship.UID = existing.UID
	relationship.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++

	// Update the relationship
	s.relationships[relationship.Name] = relationship.DeepCopyObject().(*v1alpha1.Relationship)
	return relationship, nil
}

// Delete removes a relationship
func (s *RelationshipMemoryStorage) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.relationships[name]; !exists {
		return errors.NewNotFound(
			schema.GroupResource{Group: rbacapi.GroupName, Resource: "relationships"},
			name,
		)
	}

	delete(s.relationships, name)
	return nil
}
