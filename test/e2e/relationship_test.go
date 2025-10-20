package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	relationshipAPIGroup   = "multicluster-rbac.open-cluster-management.io"
	relationshipAPIVersion = "v1alpha1"
)

var _ = Describe("Relationship API", func() {
	var (
		gvr              schema.GroupVersionResource
		relationshipName string
	)

	BeforeEach(func() {
		gvr = schema.GroupVersionResource{
			Group:    relationshipAPIGroup,
			Version:  relationshipAPIVersion,
			Resource: "relationships",
		}
		// Use unique name with timestamp to avoid conflicts
		relationshipName = fmt.Sprintf("test-relationship-%d", time.Now().UnixNano())
	})

	AfterEach(func() {
		// Clean up: delete the relationship if it exists
		// Ignore errors as it may not exist
		_ = dynamicClient.Resource(gvr).Delete(ctx, relationshipName, metav1.DeleteOptions{})
	})

	Describe("CRUD Operations", func() {
		It("should create a Relationship successfully", func() {
			relationship := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": relationshipAPIGroup + "/" + relationshipAPIVersion,
					"kind":       "Relationship",
					"metadata": map[string]interface{}{
						"name": relationshipName,
					},
					"spec": map[string]interface{}{
						"tuples": []interface{}{
							map[string]interface{}{
								"resource": map[string]interface{}{
									"objectType": "resource",
									"objectId":   "cluster/cluster1/namespace/_wildcard_",
								},
								"relation": "admin",
								"subject": map[string]interface{}{
									"object": map[string]interface{}{
										"objectType": "user",
										"objectId":   "user1",
									},
								},
							},
						},
					},
				},
			}

			result, err := dynamicClient.Resource(gvr).Create(ctx, relationship, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create relationship")
			Expect(result.GetName()).To(Equal(relationshipName))

			// Verify the tuples were stored correctly
			spec := result.Object["spec"].(map[string]interface{})
			tuples := spec["tuples"].([]interface{})
			Expect(len(tuples)).To(Equal(1))
		})

		It("should list Relationships successfully", func() {
			// First create a relationship
			relationship := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": relationshipAPIGroup + "/" + relationshipAPIVersion,
					"kind":       "Relationship",
					"metadata": map[string]interface{}{
						"name": relationshipName,
					},
					"spec": map[string]interface{}{
						"tuples": []interface{}{
							map[string]interface{}{
								"resource": map[string]interface{}{
									"objectType": "resource",
									"objectId":   "cluster/cluster2/namespace/default",
								},
								"relation": "viewer",
								"subject": map[string]interface{}{
									"object": map[string]interface{}{
										"objectType": "user",
										"objectId":   "user2",
									},
								},
							},
						},
					},
				},
			}

			_, err := dynamicClient.Resource(gvr).Create(ctx, relationship, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// List relationships
			list, err := dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to list relationships")
			Expect(list.Items).NotTo(BeEmpty(), "Expected at least one relationship in the list")
		})

		It("should get a Relationship successfully", func() {
			// First create a relationship
			relationship := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": relationshipAPIGroup + "/" + relationshipAPIVersion,
					"kind":       "Relationship",
					"metadata": map[string]interface{}{
						"name": relationshipName,
					},
					"spec": map[string]interface{}{
						"tuples": []interface{}{
							map[string]interface{}{
								"resource": map[string]interface{}{
									"objectType": "resource",
									"objectId":   "cluster/cluster3/namespace/kube-system",
								},
								"relation": "editor",
								"subject": map[string]interface{}{
									"object": map[string]interface{}{
										"objectType": "user",
										"objectId":   "user3",
									},
								},
							},
						},
					},
				},
			}

			_, err := dynamicClient.Resource(gvr).Create(ctx, relationship, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Get the relationship
			result, err := dynamicClient.Resource(gvr).Get(ctx, relationshipName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to get relationship")
			Expect(result.GetName()).To(Equal(relationshipName))

			// Verify the spec
			spec := result.Object["spec"].(map[string]interface{})
			tuples := spec["tuples"].([]interface{})
			Expect(len(tuples)).To(Equal(1))

			tuple := tuples[0].(map[string]interface{})
			Expect(tuple["relation"]).To(Equal("editor"))
		})

		It("should delete a Relationship successfully", func() {
			// First create a relationship
			relationship := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": relationshipAPIGroup + "/" + relationshipAPIVersion,
					"kind":       "Relationship",
					"metadata": map[string]interface{}{
						"name": relationshipName,
					},
					"spec": map[string]interface{}{
						"tuples": []interface{}{
							map[string]interface{}{
								"resource": map[string]interface{}{
									"objectType": "resource",
									"objectId":   "cluster/cluster4/namespace/_wildcard_",
								},
								"relation": "admin",
								"subject": map[string]interface{}{
									"object": map[string]interface{}{
										"objectType": "group",
										"objectId":   "admin-group",
									},
								},
							},
						},
					},
				},
			}

			_, err := dynamicClient.Resource(gvr).Create(ctx, relationship, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Delete the relationship
			err = dynamicClient.Resource(gvr).Delete(ctx, relationshipName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to delete relationship")

			// Verify deletion
			_, err = dynamicClient.Resource(gvr).Get(ctx, relationshipName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred(), "Expected relationship to be deleted")
		})
	})

	Describe("Multi-tuple Relationships", func() {
		It("should create a relationship with multiple tuples", func() {
			relationship := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": relationshipAPIGroup + "/" + relationshipAPIVersion,
					"kind":       "Relationship",
					"metadata": map[string]interface{}{
						"name": relationshipName,
					},
					"spec": map[string]interface{}{
						"tuples": []interface{}{
							map[string]interface{}{
								"resource": map[string]interface{}{
									"objectType": "resource",
									"objectId":   "cluster/cluster1/namespace/default",
								},
								"relation": "admin",
								"subject": map[string]interface{}{
									"object": map[string]interface{}{
										"objectType": "user",
										"objectId":   "alice",
									},
								},
							},
							map[string]interface{}{
								"resource": map[string]interface{}{
									"objectType": "resource",
									"objectId":   "cluster/cluster2/namespace/production",
								},
								"relation": "viewer",
								"subject": map[string]interface{}{
									"object": map[string]interface{}{
										"objectType": "user",
										"objectId":   "bob",
									},
								},
							},
						},
					},
				},
			}

			result, err := dynamicClient.Resource(gvr).Create(ctx, relationship, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create relationship with multiple tuples")

			// Verify both tuples were stored
			spec := result.Object["spec"].(map[string]interface{})
			tuples := spec["tuples"].([]interface{})
			Expect(len(tuples)).To(Equal(2), "Expected 2 tuples in the relationship")
		})
	})

	Describe("Error Handling", func() {
		It("should fail when creating duplicate relationships", func() {
			relationship := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": relationshipAPIGroup + "/" + relationshipAPIVersion,
					"kind":       "Relationship",
					"metadata": map[string]interface{}{
						"name": relationshipName,
					},
					"spec": map[string]interface{}{
						"tuples": []interface{}{
							map[string]interface{}{
								"resource": map[string]interface{}{
									"objectType": "resource",
									"objectId":   "cluster/test",
								},
								"relation": "admin",
								"subject": map[string]interface{}{
									"object": map[string]interface{}{
										"objectType": "user",
										"objectId":   "test-user",
									},
								},
							},
						},
					},
				},
			}

			// Create first time - should succeed
			_, err := dynamicClient.Resource(gvr).Create(ctx, relationship, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Create second time - should fail
			_, err = dynamicClient.Resource(gvr).Create(ctx, relationship, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred(), "Expected error when creating duplicate relationship")
		})

		It("should fail when getting non-existent relationship", func() {
			_, err := dynamicClient.Resource(gvr).Get(ctx, "non-existent-relationship", metav1.GetOptions{})
			Expect(err).To(HaveOccurred(), "Expected error when getting non-existent relationship")
		})
	})
})
