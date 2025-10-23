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

// PermissionBindingMemoryStorage provides in-memory storage for PermissionBinding resources
// This storage simulates what will eventually be stored in SpiceDB
type PermissionBindingMemoryStorage struct {
	mu                 sync.RWMutex
	permissionBindings map[string]*v1alpha1.PermissionBinding
	versionCounter     int64
}

// NewPermissionBindingMemoryStorage creates a new in-memory storage for permission bindings
func NewPermissionBindingMemoryStorage() *PermissionBindingMemoryStorage {
	return &PermissionBindingMemoryStorage{
		permissionBindings: make(map[string]*v1alpha1.PermissionBinding),
		versionCounter:     1,
	}
}

// Get retrieves a permission binding by name
func (s *PermissionBindingMemoryStorage) Get(name string) (*v1alpha1.PermissionBinding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	permissionBinding, exists := s.permissionBindings[name]
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionbindings"},
			name,
		)
	}
	return permissionBinding.DeepCopyObject().(*v1alpha1.PermissionBinding), nil
}

// List retrieves all permission bindings
func (s *PermissionBindingMemoryStorage) List() (*v1alpha1.PermissionBindingList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := &v1alpha1.PermissionBindingList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupName + "/" + v1alpha1.APIVersion,
			Kind:       "PermissionBindingList",
		},
		Items: make([]v1alpha1.PermissionBinding, 0, len(s.permissionBindings)),
	}

	for _, permissionBinding := range s.permissionBindings {
		list.Items = append(list.Items, *permissionBinding.DeepCopyObject().(*v1alpha1.PermissionBinding))
	}

	return list, nil
}

// Create creates a new permission binding
func (s *PermissionBindingMemoryStorage) Create(permissionBinding *v1alpha1.PermissionBinding) (*v1alpha1.PermissionBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate name if not provided
	if permissionBinding.Name == "" {
		permissionBinding.Name = string(uuid.NewUUID())
	}

	// Check if already exists
	if _, exists := s.permissionBindings[permissionBinding.Name]; exists {
		return nil, errors.NewAlreadyExists(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionbindings"},
			permissionBinding.Name,
		)
	}

	// Set metadata
	now := metav1.NewTime(time.Now())
	permissionBinding.CreationTimestamp = now
	permissionBinding.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++
	permissionBinding.UID = uuid.NewUUID()

	// Store the permission binding
	s.permissionBindings[permissionBinding.Name] = permissionBinding.DeepCopyObject().(*v1alpha1.PermissionBinding)
	return permissionBinding, nil
}

// Update updates an existing permission binding
func (s *PermissionBindingMemoryStorage) Update(permissionBinding *v1alpha1.PermissionBinding) (*v1alpha1.PermissionBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.permissionBindings[permissionBinding.Name]
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionbindings"},
			permissionBinding.Name,
		)
	}

	// Preserve immutable fields
	permissionBinding.CreationTimestamp = existing.CreationTimestamp
	permissionBinding.UID = existing.UID
	permissionBinding.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++

	// Update the permission binding
	s.permissionBindings[permissionBinding.Name] = permissionBinding.DeepCopyObject().(*v1alpha1.PermissionBinding)
	return permissionBinding, nil
}

// Delete removes a permission binding
func (s *PermissionBindingMemoryStorage) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.permissionBindings[name]; !exists {
		return errors.NewNotFound(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionbindings"},
			name,
		)
	}

	delete(s.permissionBindings, name)
	return nil
}
