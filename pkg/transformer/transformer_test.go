package transformer

import (
	"testing"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rbacv1alpha1 "github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
)

func TestSpiceDBTransformer_TransformPermissionBinding(t *testing.T) {
	transformer := NewSpiceDBTransformer()

	tests := []struct {
		name              string
		permissionBinding *rbacv1alpha1.PermissionBinding
		expectedCount     int
		expectError       bool
		validateFunc      func(t *testing.T, updates []*v1.RelationshipUpdate)
	}{
		{
			name: "simple user permission binding",
			permissionBinding: &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "User",
						Name: "alice",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Role:       "admin",
							Clusters:   []string{"cluster1"},
							Namespaces: []string{"default"},
							Resources:  []string{"pods"},
							Names:      []string{"pod1"},
						},
					},
				},
			},
			expectedCount: 1,
			expectError:   false,
			validateFunc: func(t *testing.T, updates []*v1.RelationshipUpdate) {
				require.Len(t, updates, 1)
				update := updates[0]

				assert.Equal(t, v1.RelationshipUpdate_OPERATION_CREATE, update.Operation)
				assert.Equal(t, "resource", update.Relationship.Resource.ObjectType)
				assert.Equal(t, "cluster/cluster1/namespace/default/pods/pod1", update.Relationship.Resource.ObjectId)
				assert.Equal(t, "admin", update.Relationship.Relation)
				assert.Equal(t, "user", update.Relationship.Subject.Object.ObjectType)
				assert.Equal(t, "alice", update.Relationship.Subject.Object.ObjectId)
			},
		},
		{
			name: "group permission binding with multiple resources",
			permissionBinding: &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "group-binding",
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "Group",
						Name: "developers",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Role:       "viewer",
							Clusters:   []string{"cluster1"},
							Namespaces: []string{"default", "kube-system"},
							Resources:  []string{"pods", "services"},
							Names:      []string{"*"},
						},
					},
				},
			},
			expectedCount: 4, // 1 cluster × 2 namespaces × 2 resources × 1 name
			expectError:   false,
			validateFunc: func(t *testing.T, updates []*v1.RelationshipUpdate) {
				require.Len(t, updates, 4)

				// Check that all relationships are for the group subject
				for _, update := range updates {
					assert.Equal(t, "group", update.Relationship.Subject.Object.ObjectType)
					assert.Equal(t, "developers", update.Relationship.Subject.Object.ObjectId)
					assert.Equal(t, "viewer", update.Relationship.Relation)
				}
			},
		},
		{
			name: "wildcard permissions",
			permissionBinding: &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wildcard-binding",
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "User",
						Name: "admin",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Role: "admin",
							// Empty arrays default to wildcard
						},
					},
				},
			},
			expectedCount: 1, // Should create one relationship with wildcard resource ID
			expectError:   false,
			validateFunc: func(t *testing.T, updates []*v1.RelationshipUpdate) {
				require.Len(t, updates, 1)
				update := updates[0]

				assert.Equal(t, "resource", update.Relationship.Resource.ObjectType)
				assert.Equal(t, "all", update.Relationship.Resource.ObjectId)
			},
		},
		{
			name:              "nil permission binding",
			permissionBinding: nil,
			expectedCount:     0,
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updates, err := transformer.TransformPermissionBinding(tt.permissionBinding)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, updates, tt.expectedCount)

			if tt.validateFunc != nil {
				tt.validateFunc(t, updates)
			}
		})
	}
}

func TestSpiceDBTransformer_TransformPermissionReview(t *testing.T) {
	transformer := NewSpiceDBTransformer()

	tests := []struct {
		name               string
		permissionReview   *rbacv1alpha1.PermissionReview
		expectError        bool
		expectedResource   string
		expectedPermission string
	}{
		{
			name: "simple permission review",
			permissionReview: &rbacv1alpha1.PermissionReview{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-request",
				},
				Spec: rbacv1alpha1.PermissionReviewSpec{
					Group:     "v1",
					Resource:  "pods",
					Verb:      "get",
					Cluster:   "cluster1",
					Namespace: "default",
					Name:      "pod1",
				},
			},
			expectError:        false,
			expectedResource:   "cluster/cluster1/namespace/default/pods/pod1",
			expectedPermission: "view",
		},
		{
			name: "edit permission review",
			permissionReview: &rbacv1alpha1.PermissionReview{
				ObjectMeta: metav1.ObjectMeta{
					Name: "edit-request",
				},
				Spec: rbacv1alpha1.PermissionReviewSpec{
					Resource: "services",
					Verb:     "create",
					Cluster:  "cluster2",
				},
			},
			expectError:        false,
			expectedResource:   "cluster/cluster2/services",
			expectedPermission: "edit",
		},
		{
			name:             "nil permission review",
			permissionReview: nil,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkReq, err := transformer.TransformPermissionReview(tt.permissionReview)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, checkReq)
			assert.Equal(t, "resource", checkReq.Resource.ObjectType)
			assert.Equal(t, tt.expectedResource, checkReq.Resource.ObjectId)
			assert.Equal(t, tt.expectedPermission, checkReq.Permission)
			// Subject should be nil as it needs to be filled by the caller
			assert.Nil(t, checkReq.Subject)
		})
	}
}

func TestSpiceDBTransformer_CheckPermissionFromReview(t *testing.T) {
	transformer := NewSpiceDBTransformer()

	permissionReview := &rbacv1alpha1.PermissionReview{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-request",
		},
		Spec: rbacv1alpha1.PermissionReviewSpec{
			Resource: "pods",
			Verb:     "get",
			Cluster:  "cluster1",
			Name:     "pod1",
		},
	}

	checkReq, err := transformer.CheckPermissionFromReview(permissionReview, "alice", "user")
	require.NoError(t, err)
	require.NotNil(t, checkReq)

	assert.Equal(t, "resource", checkReq.Resource.ObjectType)
	assert.Equal(t, "cluster/cluster1/pods/pod1", checkReq.Resource.ObjectId)
	assert.Equal(t, "view", checkReq.Permission)
	assert.Equal(t, "user", checkReq.Subject.Object.ObjectType)
	assert.Equal(t, "alice", checkReq.Subject.Object.ObjectId)
}

func TestSpiceDBTransformer_CreateRelationshipUpdatesForDeletion(t *testing.T) {
	transformer := NewSpiceDBTransformer()

	permissionBinding := &rbacv1alpha1.PermissionBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-binding",
		},
		Spec: rbacv1alpha1.PermissionBindingSpec{
			Subject: rbacv1.Subject{
				Kind: "User",
				Name: "alice",
			},
			Permissions: []rbacv1alpha1.Permission{
				{
					Role:       "admin",
					Clusters:   []string{"cluster1"},
					Namespaces: []string{"default"},
					Resources:  []string{"pods"},
					Names:      []string{"pod1"},
				},
			},
		},
	}

	deleteUpdates, err := transformer.CreateRelationshipUpdatesForDeletion(permissionBinding)
	require.NoError(t, err)
	require.Len(t, deleteUpdates, 1)

	update := deleteUpdates[0]
	assert.Equal(t, v1.RelationshipUpdate_OPERATION_DELETE, update.Operation)
	assert.Equal(t, "resource", update.Relationship.Resource.ObjectType)
	assert.Equal(t, "cluster/cluster1/namespace/default/pods/pod1", update.Relationship.Resource.ObjectId)
	assert.Equal(t, "admin", update.Relationship.Relation)
	assert.Equal(t, "user", update.Relationship.Subject.Object.ObjectType)
	assert.Equal(t, "alice", update.Relationship.Subject.Object.ObjectId)
}

func TestSpiceDBTransformer_MapFunctions(t *testing.T) {
	transformer := NewSpiceDBTransformer()

	t.Run("mapResourceType", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"pods", "resource"},
			{"services", "resource"},
			{"cluster", "cluster"},
			{"namespace", "namespace"},
			{"unknown", "resource"},
		}

		for _, tt := range tests {
			result := transformer.mapResourceType(tt.input)
			assert.Equal(t, tt.expected, result, "mapResourceType(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	})

	t.Run("mapRoleToRelation", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"admin", "admin"},
			{"viewer", "viewer"},
			{"editor", "editor"},
			{"unknown", "viewer"},
		}

		for _, tt := range tests {
			result := transformer.mapRoleToRelation(tt.input)
			assert.Equal(t, tt.expected, result, "mapRoleToRelation(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	})

	t.Run("mapVerbToPermission", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"get", "view"},
			{"list", "view"},
			{"create", "edit"},
			{"update", "edit"},
			{"delete", "edit"},
			{"*", "edit"},
			{"unknown", "view"},
		}

		for _, tt := range tests {
			result := transformer.mapVerbToPermission(tt.input)
			assert.Equal(t, tt.expected, result, "mapVerbToPermission(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	})

	t.Run("buildResourceID", func(t *testing.T) {
		tests := []struct {
			cluster   string
			namespace string
			resource  string
			name      string
			expected  string
		}{
			{"cluster1", "default", "pods", "pod1", "cluster/cluster1/namespace/default/pods/pod1"},
			{"cluster1", "default", "pods", "*", "cluster/cluster1/namespace/default/pods/_ALL_"},
			{"cluster1", "default", "pods", "", "cluster/cluster1/namespace/default/pods"},
			{"cluster1", "default", "", "pod1", "cluster/cluster1/namespace/default/pod1"},
			{"cluster1", "default", "", "", "cluster/cluster1/namespace/default"},
			{"cluster1", "", "", "", "cluster/cluster1"},
			{"", "", "", "", "all"},
			{"*", "*", "*", "*", "all"},
		}

		for _, tt := range tests {
			result := transformer.buildResourceID(tt.cluster, tt.namespace, tt.resource, tt.name)
			assert.Equal(t, tt.expected, result, "buildResourceID(%s, %s, %s, %s) = %s, want %s",
				tt.cluster, tt.namespace, tt.resource, tt.name, result, tt.expected)
		}
	})
}

func TestSpiceDBTransformer_ConvertSubject(t *testing.T) {
	transformer := NewSpiceDBTransformer()

	tests := []struct {
		name        string
		subject     *rbacv1.Subject
		expectError bool
		expectedRef *v1.SubjectReference
	}{
		{
			name: "user subject",
			subject: &rbacv1.Subject{
				Kind: "User",
				Name: "alice",
			},
			expectError: false,
			expectedRef: &v1.SubjectReference{
				Object: &v1.ObjectReference{
					ObjectType: "user",
					ObjectId:   "alice",
				},
			},
		},
		{
			name: "group subject",
			subject: &rbacv1.Subject{
				Kind: "Group",
				Name: "developers",
			},
			expectError: false,
			expectedRef: &v1.SubjectReference{
				Object: &v1.ObjectReference{
					ObjectType: "group",
					ObjectId:   "developers",
				},
			},
		},
		{
			name: "unsupported subject kind",
			subject: &rbacv1.Subject{
				Kind: "Robot",
				Name: "robot1",
			},
			expectError: true,
		},
		{
			name:        "nil subject",
			subject:     nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformer.convertSubject(tt.subject)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedRef.Object.ObjectType, result.Object.ObjectType)
			assert.Equal(t, tt.expectedRef.Object.ObjectId, result.Object.ObjectId)
		})
	}
}
