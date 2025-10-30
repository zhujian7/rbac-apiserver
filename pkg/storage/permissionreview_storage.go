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

// PermissionReviewMemoryStorage provides in-memory storage for PermissionReview resources
type PermissionReviewMemoryStorage struct {
	mu                 sync.RWMutex
	permissionReviews  map[string]*v1alpha1.PermissionReview
	versionCounter     int64
}

// NewPermissionReviewMemoryStorage creates a new in-memory storage for permission reviews
func NewPermissionReviewMemoryStorage() *PermissionReviewMemoryStorage {
	return &PermissionReviewMemoryStorage{
		permissionReviews: make(map[string]*v1alpha1.PermissionReview),
		versionCounter:    1,
	}
}

// Get retrieves a permission review by name
func (s *PermissionReviewMemoryStorage) Get(name string) (*v1alpha1.PermissionReview, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	permissionReview, exists := s.permissionReviews[name]
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionreviews"},
			name,
		)
	}
	return permissionReview.DeepCopyObject().(*v1alpha1.PermissionReview), nil
}

// List retrieves all permission reviews
func (s *PermissionReviewMemoryStorage) List() (*v1alpha1.PermissionReviewList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := &v1alpha1.PermissionReviewList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupName + "/" + v1alpha1.APIVersion,
			Kind:       "PermissionReviewList",
		},
		Items: make([]v1alpha1.PermissionReview, 0, len(s.permissionReviews)),
	}

	for _, permissionReview := range s.permissionReviews {
		list.Items = append(list.Items, *permissionReview.DeepCopyObject().(*v1alpha1.PermissionReview))
	}

	return list, nil
}

// Create creates a new permission review
func (s *PermissionReviewMemoryStorage) Create(permissionReview *v1alpha1.PermissionReview) (*v1alpha1.PermissionReview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate name if not provided
	if permissionReview.Name == "" {
		permissionReview.Name = string(uuid.NewUUID())
	}

	// Check if already exists
	if _, exists := s.permissionReviews[permissionReview.Name]; exists {
		return nil, errors.NewAlreadyExists(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionreviews"},
			permissionReview.Name,
		)
	}

	// Set metadata
	now := metav1.NewTime(time.Now())
	permissionReview.CreationTimestamp = now
	permissionReview.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++
	permissionReview.UID = uuid.NewUUID()

	// Store the permission review
	s.permissionReviews[permissionReview.Name] = permissionReview.DeepCopyObject().(*v1alpha1.PermissionReview)
	return permissionReview, nil
}

// Update updates an existing permission review
func (s *PermissionReviewMemoryStorage) Update(permissionReview *v1alpha1.PermissionReview) (*v1alpha1.PermissionReview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.permissionReviews[permissionReview.Name]
	if !exists {
		return nil, errors.NewNotFound(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionreviews"},
			permissionReview.Name,
		)
	}

	// Preserve immutable fields
	permissionReview.CreationTimestamp = existing.CreationTimestamp
	permissionReview.UID = existing.UID
	permissionReview.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++

	// Update the permission review
	s.permissionReviews[permissionReview.Name] = permissionReview.DeepCopyObject().(*v1alpha1.PermissionReview)
	return permissionReview, nil
}

// Delete removes a permission review
func (s *PermissionReviewMemoryStorage) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.permissionReviews[name]; !exists {
		return errors.NewNotFound(
			schema.GroupResource{Group: v1alpha1.GroupName, Resource: "permissionreviews"},
			name,
		)
	}

	delete(s.permissionReviews, name)
	return nil
}
