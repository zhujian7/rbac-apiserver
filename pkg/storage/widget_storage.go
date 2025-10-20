package storage

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"time"

	widgetapi "github.com/stolostron/rbac-apiserver/apis/widget"
	"github.com/stolostron/rbac-apiserver/apis/widget/v1alpha1"
)

type MemoryStorage struct {
	mu             sync.RWMutex
	widgets        map[string]*v1alpha1.Widget
	versionCounter int64
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		widgets:        make(map[string]*v1alpha1.Widget),
		versionCounter: 1,
	}
}

func (s *MemoryStorage) getKey(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}

func (s *MemoryStorage) ParseKey(key string) (namespace, name string) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}

func (s *MemoryStorage) Get(namespace, name string) (*v1alpha1.Widget, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.getKey(namespace, name)
	widget, exists := s.widgets[key]
	if !exists {
		return nil, errors.NewNotFound(schema.GroupResource{Group: widgetapi.GroupName, Resource: "widgets"}, name)
	}
	return widget.DeepCopyObject().(*v1alpha1.Widget), nil
}

func (s *MemoryStorage) List(ns string) (*v1alpha1.WidgetList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := &v1alpha1.WidgetList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: widgetapi.GroupName + "/" + v1alpha1.APIVersion,
			Kind:       "WidgetList",
		},
		Items: make([]v1alpha1.Widget, 0, len(s.widgets)),
	}

	for _, widget := range s.widgets {
		if len(ns) == 0 {
			list.Items = append(list.Items, *widget.DeepCopyObject().(*v1alpha1.Widget))
		} else if widget.Namespace == ns {
			list.Items = append(list.Items, *widget.DeepCopyObject().(*v1alpha1.Widget))
		}

	}

	return list, nil
}

func (s *MemoryStorage) Create(widget *v1alpha1.Widget) (*v1alpha1.Widget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if widget.Name == "" {
		widget.Name = string(uuid.NewUUID())
	}

	key := s.getKey(widget.Namespace, widget.Name)
	if _, exists := s.widgets[key]; exists {
		return nil, fmt.Errorf("widget %s already exists in namespace %s", widget.Name, widget.Namespace)
	}

	now := metav1.NewTime(time.Now())
	widget.CreationTimestamp = now
	widget.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++
	widget.UID = uuid.NewUUID()
	widget.Status.Phase = "Active"

	s.widgets[key] = widget.DeepCopyObject().(*v1alpha1.Widget)
	return widget, nil
}

func (s *MemoryStorage) Update(widget *v1alpha1.Widget) (*v1alpha1.Widget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.getKey(widget.Namespace, widget.Name)
	existing, exists := s.widgets[key]
	if !exists {
		return nil, errors.NewNotFound(schema.GroupResource{Group: widgetapi.GroupName, Resource: "widgets"}, widget.Name)
	}

	widget.CreationTimestamp = existing.CreationTimestamp
	widget.UID = existing.UID
	widget.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++

	s.widgets[key] = widget.DeepCopyObject().(*v1alpha1.Widget)
	return widget, nil
}

func (s *MemoryStorage) Delete(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.getKey(namespace, name)
	if _, exists := s.widgets[key]; !exists {
		return errors.NewNotFound(schema.GroupResource{Group: widgetapi.GroupName, Resource: "widgets"}, name)
	}

	delete(s.widgets, key)
	return nil
}
