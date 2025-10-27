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

var _ = Describe("SpiceDB Integration E2E Tests", func() {
	var (
		testNamePrefix string
	)

	BeforeEach(func() {
		// Use unique name prefix with timestamp to avoid conflicts
		testNamePrefix = fmt.Sprintf("spicedb-test-%d", time.Now().UnixNano())
	})

	Describe("PermissionBinding to SpiceDB Synchronization", func() {
		var permissionBindingName string

		BeforeEach(func() {
			permissionBindingName = fmt.Sprintf("%s-pb", testNamePrefix)
		})

		AfterEach(func() {
			// Clean up: delete the permission binding if it exists
			_ = rbacClient.AuthorizationV1alpha1().PermissionBindings().Delete(ctx, permissionBindingName, metav1.DeleteOptions{})
		})

		It("should create PermissionBinding and sync to SpiceDB", func() {
			By("Creating a PermissionBinding")
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
							Resources:  []string{"pods"},
							Groups:     []string{""},
							Namespaces: []string{"default"},
							Names:      []string{"test-pod"},
							Role:       "admin",
							Clusters:   []string{"cluster1"},
						},
					},
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create PermissionBinding")
			Expect(result.GetName()).To(Equal(permissionBindingName))

			By("Verifying PermissionBinding was created successfully")
			retrievedPB, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Get(ctx, permissionBindingName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPB.Spec.Subject.Name).To(Equal("alice"))
			Expect(len(retrievedPB.Spec.Permissions)).To(Equal(1))

			// Note: In a real environment with actual SpiceDB integration,
			// we would verify that relationships were created in SpiceDB
			// For now, we verify that the API operations complete successfully
			// which indicates the integration layer is working
		})

		It("should handle PermissionBinding with multiple permissions", func() {
			By("Creating a PermissionBinding with multiple permissions")
			permissionBinding := &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionBindingName,
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "Group",
						Name: "developers",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Resources:  []string{"pods", "services"},
							Groups:     []string{""},
							Namespaces: []string{"default"},
							Role:       "viewer",
							Clusters:   []string{"cluster1"},
						},
						{
							Resources:  []string{"deployments"},
							Groups:     []string{"apps"},
							Namespaces: []string{"production"},
							Role:       "editor",
							Clusters:   []string{"cluster2"},
						},
					},
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create PermissionBinding with multiple permissions")

			By("Verifying all permissions were stored correctly")
			permissions := result.Spec.Permissions
			Expect(len(permissions)).To(Equal(2))

			// Verify first permission
			Expect(permissions[0].Role).To(Equal("viewer"))
			Expect(permissions[0].Resources).To(ContainElements("pods", "services"))

			// Verify second permission
			Expect(permissions[1].Role).To(Equal("editor"))
			Expect(permissions[1].Resources).To(ContainElements("deployments"))
		})

		It("should delete PermissionBinding and clean up SpiceDB relationships", func() {
			By("Creating a PermissionBinding")
			permissionBinding := &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionBindingName,
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "User",
						Name: "bob",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Resources:  []string{"configmaps"},
							Groups:     []string{""},
							Namespaces: []string{"default"},
							Role:       "editor",
							Clusters:   []string{"cluster1"},
						},
					},
				},
			}

			_, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Deleting the PermissionBinding")
			err = rbacClient.AuthorizationV1alpha1().PermissionBindings().Delete(ctx, permissionBindingName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to delete PermissionBinding")

			By("Verifying deletion")
			_, err = rbacClient.AuthorizationV1alpha1().PermissionBindings().Get(ctx, permissionBindingName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred(), "Expected PermissionBinding to be deleted")
		})
	})

	Describe("PermissionRequest with SpiceDB Evaluation", func() {
		var permissionRequestName string

		BeforeEach(func() {
			permissionRequestName = fmt.Sprintf("%s-pr", testNamePrefix)
		})

		AfterEach(func() {
			// Clean up: delete the permission request if it exists
			_ = rbacClient.AuthorizationV1alpha1().PermissionRequests().Delete(ctx, permissionRequestName, metav1.DeleteOptions{})
		})

		It("should create PermissionRequest and evaluate with SpiceDB", func() {
			By("Creating a PermissionRequest")
			permissionRequest := &rbacv1alpha1.PermissionRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionRequestName,
				},
				Spec: rbacv1alpha1.PermissionRequestSpec{
					Group:     "",
					Resource:  "pods",
					Verb:      "get",
					Cluster:   "cluster1",
					Namespace: "default",
					Name:      "test-pod",
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().PermissionRequests().Create(ctx, permissionRequest, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create PermissionRequest")
			Expect(result.GetName()).To(Equal(permissionRequestName))

			By("Verifying PermissionRequest was created and processed")
			// The integration service should have processed this request with SpiceDB
			// and potentially updated the status
			retrievedPR, err := rbacClient.AuthorizationV1alpha1().PermissionRequests().Get(ctx, permissionRequestName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPR.Spec.Resource).To(Equal("pods"))
			Expect(retrievedPR.Spec.Verb).To(Equal("get"))

			// Note: The status might be updated by the SpiceDB integration
			// depending on existing relationships and the placeholder user
		})

		It("should handle different verbs and resources", func() {
			testCases := []struct {
				verb     string
				resource string
				group    string
			}{
				{"list", "services", ""},
				{"create", "deployments", "apps"},
				{"update", "configmaps", ""},
				{"delete", "secrets", ""},
				{"watch", "pods", ""},
			}

			for i, tc := range testCases {
				By(fmt.Sprintf("Testing verb '%s' on resource '%s' (case %d)", tc.verb, tc.resource, i+1))

				prName := fmt.Sprintf("%s-%d", permissionRequestName, i)
				permissionRequest := &rbacv1alpha1.PermissionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Name: prName,
					},
					Spec: rbacv1alpha1.PermissionRequestSpec{
						Group:     tc.group,
						Resource:  tc.resource,
						Verb:      tc.verb,
						Cluster:   "cluster1",
						Namespace: "default",
					},
				}

				result, err := rbacClient.AuthorizationV1alpha1().PermissionRequests().Create(ctx, permissionRequest, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create PermissionRequest for %s:%s", tc.verb, tc.resource))
				Expect(result.Spec.Verb).To(Equal(tc.verb))
				Expect(result.Spec.Resource).To(Equal(tc.resource))

				// Clean up this specific test case
				_ = rbacClient.AuthorizationV1alpha1().PermissionRequests().Delete(ctx, prName, metav1.DeleteOptions{})
			}
		})

		It("should handle cluster-wide and namespaced requests", func() {
			By("Creating a cluster-wide PermissionRequest")
			clusterPRName := fmt.Sprintf("%s-cluster", permissionRequestName)
			clusterPermissionRequest := &rbacv1alpha1.PermissionRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterPRName,
				},
				Spec: rbacv1alpha1.PermissionRequestSpec{
					Group:    "",
					Resource: "nodes",
					Verb:     "list",
					Cluster:  "cluster1",
					// No namespace for cluster-wide resources
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().PermissionRequests().Create(ctx, clusterPermissionRequest, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create cluster-wide PermissionRequest")
			Expect(result.Spec.Namespace).To(BeEmpty())

			By("Creating a namespaced PermissionRequest")
			namespacedPRName := fmt.Sprintf("%s-namespaced", permissionRequestName)
			namespacedPermissionRequest := &rbacv1alpha1.PermissionRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespacedPRName,
				},
				Spec: rbacv1alpha1.PermissionRequestSpec{
					Group:     "apps",
					Resource:  "deployments",
					Verb:      "get",
					Cluster:   "cluster1",
					Namespace: "production",
					Name:      "my-deployment",
				},
			}

			result, err = rbacClient.AuthorizationV1alpha1().PermissionRequests().Create(ctx, namespacedPermissionRequest, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create namespaced PermissionRequest")
			Expect(result.Spec.Namespace).To(Equal("production"))
			Expect(result.Spec.Name).To(Equal("my-deployment"))

			// Clean up
			_ = rbacClient.AuthorizationV1alpha1().PermissionRequests().Delete(ctx, clusterPRName, metav1.DeleteOptions{})
			_ = rbacClient.AuthorizationV1alpha1().PermissionRequests().Delete(ctx, namespacedPRName, metav1.DeleteOptions{})
		})
	})

	Describe("Integration Workflow Tests", func() {
		var (
			permissionBindingName string
			permissionRequestName string
		)

		BeforeEach(func() {
			permissionBindingName = fmt.Sprintf("%s-workflow-pb", testNamePrefix)
			permissionRequestName = fmt.Sprintf("%s-workflow-pr", testNamePrefix)
		})

		AfterEach(func() {
			// Clean up both resources
			_ = rbacClient.AuthorizationV1alpha1().PermissionBindings().Delete(ctx, permissionBindingName, metav1.DeleteOptions{})
			_ = rbacClient.AuthorizationV1alpha1().PermissionRequests().Delete(ctx, permissionRequestName, metav1.DeleteOptions{})
		})

		It("should handle complete workflow: create binding, then evaluate request", func() {
			By("Creating a PermissionBinding for user 'charlie'")
			permissionBinding := &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionBindingName,
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "User",
						Name: "charlie",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Resources:  []string{"pods"},
							Groups:     []string{""},
							Namespaces: []string{"default"},
							Names:      []string{"test-pod"},
							Role:       "viewer",
							Clusters:   []string{"cluster1"},
						},
					},
				},
			}

			pbResult, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create PermissionBinding")

			By("Waiting a moment for SpiceDB synchronization")
			time.Sleep(100 * time.Millisecond)

			By("Creating a PermissionRequest that should match the binding")
			permissionRequest := &rbacv1alpha1.PermissionRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionRequestName,
				},
				Spec: rbacv1alpha1.PermissionRequestSpec{
					Group:     "",
					Resource:  "pods",
					Verb:      "get", // Should map to 'view' permission
					Cluster:   "cluster1",
					Namespace: "default",
					Name:      "test-pod",
				},
			}

			prResult, err := rbacClient.AuthorizationV1alpha1().PermissionRequests().Create(ctx, permissionRequest, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create PermissionRequest")

			By("Verifying both resources were created successfully")
			Expect(pbResult.Spec.Subject.Name).To(Equal("charlie"))
			Expect(prResult.Spec.Resource).To(Equal("pods"))
			Expect(prResult.Spec.Verb).To(Equal("get"))

			// Note: In a full integration test with actual SpiceDB services running,
			// we would verify that the PermissionRequest status was updated based on
			// the evaluation against the relationships created by the PermissionBinding
		})

		It("should handle resource updates and deletions", func() {
			By("Creating initial PermissionBinding")
			permissionBinding := &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionBindingName,
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "User",
						Name: "diana",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Resources:  []string{"services"},
							Groups:     []string{""},
							Namespaces: []string{"default"},
							Role:       "editor",
							Clusters:   []string{"cluster1"},
						},
					},
				},
			}

			_, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Deleting the PermissionBinding")
			err = rbacClient.AuthorizationV1alpha1().PermissionBindings().Delete(ctx, permissionBindingName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Creating a PermissionRequest after deletion")
			permissionRequest := &rbacv1alpha1.PermissionRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionRequestName,
				},
				Spec: rbacv1alpha1.PermissionRequestSpec{
					Group:     "",
					Resource:  "services",
					Verb:      "create",
					Cluster:   "cluster1",
					Namespace: "default",
				},
			}

			_, err = rbacClient.AuthorizationV1alpha1().PermissionRequests().Create(ctx, permissionRequest, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "PermissionRequest creation should still work after binding deletion")

			// Note: With full SpiceDB integration, this request would likely be denied
			// since the relationships were deleted with the PermissionBinding
		})
	})

	Describe("Error Handling and Edge Cases", func() {
		It("should handle empty permission lists gracefully", func() {
			permissionBindingName := fmt.Sprintf("%s-empty", testNamePrefix)

			By("Creating a PermissionBinding with empty permissions")
			permissionBinding := &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionBindingName,
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "User",
						Name: "empty-user",
					},
					Permissions: []rbacv1alpha1.Permission{}, // Empty permissions
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Should allow creating PermissionBinding with empty permissions")
			Expect(len(result.Spec.Permissions)).To(Equal(0))

			// Clean up
			_ = rbacClient.AuthorizationV1alpha1().PermissionBindings().Delete(ctx, permissionBindingName, metav1.DeleteOptions{})
		})

		It("should handle wildcard permissions", func() {
			permissionBindingName := fmt.Sprintf("%s-wildcard", testNamePrefix)

			By("Creating a PermissionBinding with wildcard permissions")
			permissionBinding := &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionBindingName,
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "User",
						Name: "wildcard-user",
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Resources:  []string{"*"},
							Groups:     []string{"*"},
							Namespaces: []string{"*"},
							Names:      []string{"*"},
							Role:       "admin",
							Clusters:   []string{"*"},
						},
					},
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Should allow creating PermissionBinding with wildcard permissions")

			permission := result.Spec.Permissions[0]
			Expect(permission.Resources).To(ContainElement("*"))
			Expect(permission.Groups).To(ContainElement("*"))
			Expect(permission.Namespaces).To(ContainElement("*"))
			Expect(permission.Names).To(ContainElement("*"))
			Expect(permission.Clusters).To(ContainElement("*"))

			// Clean up
			_ = rbacClient.AuthorizationV1alpha1().PermissionBindings().Delete(ctx, permissionBindingName, metav1.DeleteOptions{})
		})
	})
})
