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

// PermissionRequestMemoryStorage provides in-memory storage for PermissionRequest resources
type PermissionRequestMemoryStorage struct {
	mu                 sync.RWMutex
	permissionRequests map[string]*v1alpha1.PermissionRequest
	versionCounter     int64
}

// NewPermissionRequestMemoryStorage creates a new in-memory storage for permission requests
func NewPermissionRequestMemoryStorage() *PermissionRequestMemoryStorage {
	return &PermissionRequestMemoryStorage{
		permissionRequests: make(map[string]*v1alpha1.PermissionRequest),
		versionCounter:     1,
	}
}

// Get retrieves a permission request by name
func (s *PermissionRequestMemoryStorage) Get(name string) (*v1alpha1.PermissionRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	permissionRequest, exists := s.permissionRequests[name]
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionrequests"},
			name,
		)
	}
	return permissionRequest.DeepCopyObject().(*v1alpha1.PermissionRequest), nil
}

// List retrieves all permission requests
func (s *PermissionRequestMemoryStorage) List() (*v1alpha1.PermissionRequestList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := &v1alpha1.PermissionRequestList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupName + "/" + v1alpha1.APIVersion,
			Kind:       "PermissionRequestList",
		},
		Items: make([]v1alpha1.PermissionRequest, 0, len(s.permissionRequests)),
	}

	for _, permissionRequest := range s.permissionRequests {
		list.Items = append(list.Items, *permissionRequest.DeepCopyObject().(*v1alpha1.PermissionRequest))
	}

	return list, nil
}

// Create creates a new permission request
func (s *PermissionRequestMemoryStorage) Create(permissionRequest *v1alpha1.PermissionRequest) (*v1alpha1.PermissionRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate name if not provided
	if permissionRequest.Name == "" {
		permissionRequest.Name = string(uuid.NewUUID())
	}

	// Check if already exists
	if _, exists := s.permissionRequests[permissionRequest.Name]; exists {
		return nil, errors.NewAlreadyExists(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionrequests"},
			permissionRequest.Name,
		)
	}

	// Set metadata
	now := metav1.NewTime(time.Now())
	permissionRequest.CreationTimestamp = now
	permissionRequest.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++
	permissionRequest.UID = uuid.NewUUID()

	// Store the permission request
	s.permissionRequests[permissionRequest.Name] = permissionRequest.DeepCopyObject().(*v1alpha1.PermissionRequest)
	return permissionRequest, nil
}

// Update updates an existing permission request
func (s *PermissionRequestMemoryStorage) Update(permissionRequest *v1alpha1.PermissionRequest) (*v1alpha1.PermissionRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.permissionRequests[permissionRequest.Name]
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionrequests"},
			permissionRequest.Name,
		)
	}

	// Preserve immutable fields
	permissionRequest.CreationTimestamp = existing.CreationTimestamp
	permissionRequest.UID = existing.UID
	permissionRequest.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++

	// Update the permission request
	s.permissionRequests[permissionRequest.Name] = permissionRequest.DeepCopyObject().(*v1alpha1.PermissionRequest)
	return permissionRequest, nil
}

// Delete removes a permission request
func (s *PermissionRequestMemoryStorage) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.permissionRequests[name]; !exists {
		return errors.NewNotFound(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionrequests"},
			name,
		)
	}

	delete(s.permissionRequests, name)
	return nil
}