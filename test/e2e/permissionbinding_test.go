package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1alpha1 "github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PermissionBinding API", func() {
	var (
		permissionBindingName string
	)

	BeforeEach(func() {
		// Use unique name with timestamp to avoid conflicts
		permissionBindingName = fmt.Sprintf("test-permissionbinding-%d", time.Now().UnixNano())
	})

	AfterEach(func() {
		// Clean up: delete the permission binding if it exists
		// Ignore errors as it may not exist
		_ = rbacClient.AuthorizationV1alpha1().PermissionBindings().Delete(ctx, permissionBindingName, metav1.DeleteOptions{})
	})

	Describe("CRUD Operations", func() {
		It("should create a PermissionBinding successfully", func() {
			permissionBinding := &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionBindingName,
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "User",
						Name: "user1",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Resources:  []string{"deployments"},
							Groups:     []string{"apps"},
							Namespaces: []string{"default"},
							Role:       "admin",
							Clusters:   []string{"cluster1"},
						},
					},
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create permission binding")
			Expect(result.GetName()).To(Equal(permissionBindingName))

			// Verify the permissions were stored correctly
			permissions := result.Spec.Permissions
			Expect(len(permissions)).To(Equal(1))
		})

		It("should list PermissionBindings successfully", func() {
			// First create a permission binding
			permissionBinding := &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionBindingName,
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "User",
						Name: "user2",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Resources:  []string{"pods"},
							Groups:     []string{""},
							Namespaces: []string{"default"},
							Role:       "viewer",
							Clusters:   []string{"cluster2"},
						},
					},
				},
			}

			_, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// List permission bindings
			list, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to list permission bindings")
			Expect(list.Items).NotTo(BeEmpty(), "Expected at least one permission binding in the list")
		})

		It("should get a PermissionBinding successfully", func() {
			// First create a permission binding
			permissionBinding := &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionBindingName,
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "User",
						Name: "user3",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Resources:  []string{"services"},
							Groups:     []string{""},
							Namespaces: []string{"kube-system"},
							Role:       "editor",
							Clusters:   []string{"cluster3"},
						},
					},
				},
			}

			_, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Get the permission binding
			result, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Get(ctx, permissionBindingName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to get permission binding")
			Expect(result.GetName()).To(Equal(permissionBindingName))

			// Verify the spec
			permissions := result.Spec.Permissions
			Expect(len(permissions)).To(Equal(1))

			permission := permissions[0]
			Expect(permission.Role).To(Equal("editor"))
		})

		It("should delete a PermissionBinding successfully", func() {
			// First create a permission binding
			permissionBinding := &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionBindingName,
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "Group",
						Name: "admin-group",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Resources:  []string{"*"},
							Groups:     []string{"*"},
							Namespaces: []string{"*"},
							Role:       "editor",
							Clusters:   []string{"cluster4"},
						},
					},
				},
			}

			_, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Delete the permission binding
			err = rbacClient.AuthorizationV1alpha1().PermissionBindings().Delete(ctx, permissionBindingName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to delete permission binding")

			// Verify deletion
			_, err = rbacClient.AuthorizationV1alpha1().PermissionBindings().Get(ctx, permissionBindingName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred(), "Expected permission binding to be deleted")
		})
	})

	Describe("Multi-permission Bindings", func() {
		It("should create a permission binding with multiple permissions", func() {
			permissionBinding := &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionBindingName,
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "User",
						Name: "alice",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Resources:  []string{"deployments", "replicasets"},
							Groups:     []string{"apps"},
							Namespaces: []string{"default"},
							Role:       "editor",
							Clusters:   []string{"cluster1"},
						},
						{
							Resources:  []string{"pods", "services"},
							Groups:     []string{""},
							Namespaces: []string{"production"},
							Role:       "viewer",
							Clusters:   []string{"cluster2"},
						},
					},
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create permission binding with multiple permissions")

			// Verify both permissions were stored
			permissions := result.Spec.Permissions
			Expect(len(permissions)).To(Equal(2), "Expected 2 permissions in the permission binding")
		})
	})
})
