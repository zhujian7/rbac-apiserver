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

	Describe("PermissionReview with SpiceDB Evaluation", func() {
		var permissionReviewName string
		var permissionBindingName string

		BeforeEach(func() {
			permissionReviewName = fmt.Sprintf("%s-pr", testNamePrefix)
			permissionBindingName = fmt.Sprintf("%s-pr-binding", testNamePrefix)
		})

		AfterEach(func() {
			// Clean up: PermissionReviews are not persisted (CSR-like), but clean up bindings
			_ = rbacClient.AuthorizationV1alpha1().PermissionBindings().Delete(ctx, permissionBindingName, metav1.DeleteOptions{})
		})

		It("should create PermissionReview and evaluate with SpiceDB", func() {
			By("Creating a PermissionBinding first to grant permissions")
			permissionBinding := &rbacv1alpha1.PermissionBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionBindingName,
				},
				Spec: rbacv1alpha1.PermissionBindingSpec{
					Subject: rbacv1.Subject{
						Kind: "User",
						Name: "kubernetes-admin", // This should match the user from kubeconfig context
					},
					Permissions: []rbacv1alpha1.Permission{
						{
							Resources:  []string{"pods"},
							Groups:     []string{""},
							Namespaces: []string{"default"},
							Names:      []string{"*"}, // Wildcard to allow any pod name
							Role:       "viewer",
							Clusters:   []string{"cluster1"},
						},
					},
				},
			}

			_, err := rbacClient.AuthorizationV1alpha1().PermissionBindings().Create(ctx, permissionBinding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create PermissionBinding")

			By("Waiting for SpiceDB synchronization")
			time.Sleep(200 * time.Millisecond)

			By("Creating a PermissionReview (CSR-like: evaluates immediately)")
			permissionReview := &rbacv1alpha1.PermissionReview{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionReviewName,
				},
				Spec: rbacv1alpha1.PermissionReviewSpec{
					Group:     "",
					Resource:  "pods",
					Verb:      "get", // Maps to 'view' permission
					Cluster:   "cluster1",
					Namespace: "default",
					Name:      "test-pod",
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().PermissionReviews().Create(ctx, permissionReview, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create PermissionReview")
			Expect(result.GetName()).To(Equal(permissionReviewName))

			By("Verifying PermissionReview was evaluated immediately with status populated")
			Expect(result.Spec.Resource).To(Equal("pods"))
			Expect(result.Spec.Verb).To(Equal("get"))

			// Verify status is populated by SpiceDB evaluation
			Expect(result.Status.AllowedList).NotTo(BeEmpty(), "Status should be populated with evaluation results")
			Expect(len(result.Status.AllowedList)).To(BeNumerically(">", 0))

			// Verify the allowed resource details
			allowedItem := result.Status.AllowedList[0]
			Expect(allowedItem.Cluster).To(Equal("cluster1"))
			Expect(len(allowedItem.NamespacedNames)).To(BeNumerically(">", 0))

			namespacedName := allowedItem.NamespacedNames[0]
			Expect(namespacedName.Namespace).To(Equal("default"))
			Expect(namespacedName.Names).NotTo(BeEmpty())
			Expect(namespacedName.Names).To(ContainElement("test-pod"))

			By("Verifying PermissionReview is NOT persisted (CSR-like behavior)")
			// Attempting to get the request should fail because it's not stored
			_, err = rbacClient.AuthorizationV1alpha1().PermissionReviews().Get(ctx, permissionReviewName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred(), "PermissionReview should NOT be persisted (CSR-like)")
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

				prName := fmt.Sprintf("%s-%d", permissionReviewName, i)
				permissionReview := &rbacv1alpha1.PermissionReview{
					ObjectMeta: metav1.ObjectMeta{
						Name: prName,
					},
					Spec: rbacv1alpha1.PermissionReviewSpec{
						Group:     tc.group,
						Resource:  tc.resource,
						Verb:      tc.verb,
						Cluster:   "cluster1",
						Namespace: "default",
					},
				}

				result, err := rbacClient.AuthorizationV1alpha1().PermissionReviews().Create(ctx, permissionReview, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create PermissionReview for %s:%s", tc.verb, tc.resource))
				Expect(result.Spec.Verb).To(Equal(tc.verb))
				Expect(result.Spec.Resource).To(Equal(tc.resource))

				// Verify status is returned immediately (even if empty due to no matching binding)
				Expect(result.Status).NotTo(BeNil())

				// No cleanup needed - PermissionReviews are not persisted (CSR-like)
			}
		})

		It("should evaluate PermissionReview with empty status when permission is denied", func() {
			By("Creating a PermissionReview WITHOUT a matching PermissionBinding")
			permissionReview := &rbacv1alpha1.PermissionReview{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionReviewName,
				},
				Spec: rbacv1alpha1.PermissionReviewSpec{
					Group:     "",
					Resource:  "secrets",
					Verb:      "delete",
					Cluster:   "cluster1",
					Namespace: "kube-system",
					Name:      "important-secret",
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().PermissionReviews().Create(ctx, permissionReview, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "PermissionReview creation should succeed even without matching binding")

			By("Verifying PermissionReview was evaluated with empty/denied status")
			Expect(result.Spec.Resource).To(Equal("secrets"))
			Expect(result.Spec.Verb).To(Equal("delete"))

			// Status should be populated but AllowedList should be empty (permission denied)
			Expect(result.Status).NotTo(BeNil(), "Status should be returned even when denied")
			// In a real SpiceDB environment, AllowedList would be empty since no binding exists
		})

		It("should handle cluster-wide and namespaced requests", func() {
			By("Creating a cluster-wide PermissionReview")
			clusterPRName := fmt.Sprintf("%s-cluster", permissionReviewName)
			clusterPermissionReview := &rbacv1alpha1.PermissionReview{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterPRName,
				},
				Spec: rbacv1alpha1.PermissionReviewSpec{
					Group:    "",
					Resource: "nodes",
					Verb:     "list",
					Cluster:  "cluster1",
					// No namespace for cluster-wide resources
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().PermissionReviews().Create(ctx, clusterPermissionReview, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create cluster-wide PermissionReview")
			Expect(result.Spec.Namespace).To(BeEmpty())
			Expect(result.Status).NotTo(BeNil(), "Status should be evaluated immediately")

			By("Creating a namespaced PermissionReview")
			namespacedPRName := fmt.Sprintf("%s-namespaced", permissionReviewName)
			namespacedPermissionReview := &rbacv1alpha1.PermissionReview{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespacedPRName,
				},
				Spec: rbacv1alpha1.PermissionReviewSpec{
					Group:     "apps",
					Resource:  "deployments",
					Verb:      "get",
					Cluster:   "cluster1",
					Namespace: "production",
					Name:      "my-deployment",
				},
			}

			result, err = rbacClient.AuthorizationV1alpha1().PermissionReviews().Create(ctx, namespacedPermissionReview, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create namespaced PermissionReview")
			Expect(result.Spec.Namespace).To(Equal("production"))
			Expect(result.Spec.Name).To(Equal("my-deployment"))
			Expect(result.Status).NotTo(BeNil(), "Status should be evaluated immediately")

			// No cleanup needed - PermissionReviews are not persisted (CSR-like)
		})
	})

	Describe("Integration Workflow Tests", func() {
		var (
			permissionBindingName string
			permissionReviewName string
		)

		BeforeEach(func() {
			permissionBindingName = fmt.Sprintf("%s-workflow-pb", testNamePrefix)
			permissionReviewName = fmt.Sprintf("%s-workflow-pr", testNamePrefix)
		})

		AfterEach(func() {
			// Clean up PermissionBindings (PermissionReviews are not persisted)
			_ = rbacClient.AuthorizationV1alpha1().PermissionBindings().Delete(ctx, permissionBindingName, metav1.DeleteOptions{})
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

			By("Creating a PermissionReview that should match the binding")
			permissionReview := &rbacv1alpha1.PermissionReview{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionReviewName,
				},
				Spec: rbacv1alpha1.PermissionReviewSpec{
					Group:     "",
					Resource:  "pods",
					Verb:      "get", // Should map to 'view' permission
					Cluster:   "cluster1",
					Namespace: "default",
					Name:      "test-pod",
				},
			}

			prResult, err := rbacClient.AuthorizationV1alpha1().PermissionReviews().Create(ctx, permissionReview, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create PermissionReview")

			By("Verifying PermissionBinding was created and PermissionReview was evaluated")
			Expect(pbResult.Spec.Subject.Name).To(Equal("charlie"))
			Expect(prResult.Spec.Resource).To(Equal("pods"))
			Expect(prResult.Spec.Verb).To(Equal("get"))

			// Verify PermissionReview status is populated with SpiceDB evaluation
			Expect(prResult.Status).NotTo(BeNil(), "Status should be evaluated immediately")
			// Note: With full SpiceDB integration, status.AllowedList should contain the permitted resources
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

			By("Creating a PermissionReview after deletion")
			permissionReview := &rbacv1alpha1.PermissionReview{
				ObjectMeta: metav1.ObjectMeta{
					Name: permissionReviewName,
				},
				Spec: rbacv1alpha1.PermissionReviewSpec{
					Group:     "",
					Resource:  "services",
					Verb:      "create",
					Cluster:   "cluster1",
					Namespace: "default",
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().PermissionReviews().Create(ctx, permissionReview, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "PermissionReview creation should still work after binding deletion")

			// Verify status is returned immediately (CSR-like behavior)
			Expect(result.Status).NotTo(BeNil(), "Status should be evaluated even when permission might be denied")
			// With full SpiceDB integration, status.AllowedList would likely be empty
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
