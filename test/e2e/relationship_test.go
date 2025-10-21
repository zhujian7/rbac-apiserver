package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1alpha1 "github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Relationship API", func() {
	var (
		relationshipName string
	)

	BeforeEach(func() {
		// Use unique name with timestamp to avoid conflicts
		relationshipName = fmt.Sprintf("test-relationship-%d", time.Now().UnixNano())
	})

	AfterEach(func() {
		// Clean up: delete the relationship if it exists
		// Ignore errors as it may not exist
		_ = rbacClient.AuthorizationV1alpha1().Relationships().Delete(ctx, relationshipName, metav1.DeleteOptions{})
	})

	Describe("CRUD Operations", func() {
		It("should create a Relationship successfully", func() {
			relationship := &rbacv1alpha1.Relationship{
				ObjectMeta: metav1.ObjectMeta{
					Name: relationshipName,
				},
				Spec: rbacv1alpha1.RelationshipSpec{
					Tuples: []rbacv1alpha1.Tuple{
						{
							Resource: rbacv1alpha1.ObjectReference{
								ObjectType: "resource",
								ObjectId:   "cluster/cluster1/namespace/_wildcard_",
							},
							Relation: "admin",
							Subject: rbacv1alpha1.SubjectReference{
								Object: rbacv1alpha1.ObjectReference{
									ObjectType: "user",
									ObjectId:   "user1",
								},
							},
						},
					},
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().Relationships().Create(ctx, relationship, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create relationship")
			Expect(result.GetName()).To(Equal(relationshipName))

			// Verify the tuples were stored correctly
			tuples := result.Spec.Tuples
			Expect(len(tuples)).To(Equal(1))
		})

		It("should list Relationships successfully", func() {
			// First create a relationship
			relationship := &rbacv1alpha1.Relationship{
				ObjectMeta: metav1.ObjectMeta{
					Name: relationshipName,
				},
				Spec: rbacv1alpha1.RelationshipSpec{
					Tuples: []rbacv1alpha1.Tuple{
						{
							Resource: rbacv1alpha1.ObjectReference{
								ObjectType: "resource",
								ObjectId:   "cluster/cluster2/namespace/default",
							},
							Relation: "viewer",
							Subject: rbacv1alpha1.SubjectReference{
								Object: rbacv1alpha1.ObjectReference{
									ObjectType: "user",
									ObjectId:   "user2",
								},
							},
						},
					},
				},
			}

			_, err := rbacClient.AuthorizationV1alpha1().Relationships().Create(ctx, relationship, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// List relationships
			list, err := rbacClient.AuthorizationV1alpha1().Relationships().List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to list relationships")
			Expect(list.Items).NotTo(BeEmpty(), "Expected at least one relationship in the list")
		})

		It("should get a Relationship successfully", func() {
			// First create a relationship
			relationship := &rbacv1alpha1.Relationship{
				ObjectMeta: metav1.ObjectMeta{
					Name: relationshipName,
				},
				Spec: rbacv1alpha1.RelationshipSpec{
					Tuples: []rbacv1alpha1.Tuple{
						{
							Resource: rbacv1alpha1.ObjectReference{
								ObjectType: "resource",
								ObjectId:   "cluster/cluster3/namespace/kube-system",
							},
							Relation: "editor",
							Subject: rbacv1alpha1.SubjectReference{
								Object: rbacv1alpha1.ObjectReference{
									ObjectType: "user",
									ObjectId:   "user3",
								},
							},
						},
					},
				},
			}

			_, err := rbacClient.AuthorizationV1alpha1().Relationships().Create(ctx, relationship, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Get the relationship
			result, err := rbacClient.AuthorizationV1alpha1().Relationships().Get(ctx, relationshipName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to get relationship")
			Expect(result.GetName()).To(Equal(relationshipName))

			// Verify the spec
			tuples := result.Spec.Tuples
			Expect(len(tuples)).To(Equal(1))

			tuple := tuples[0]
			Expect(tuple.Relation).To(Equal("editor"))
		})

		It("should delete a Relationship successfully", func() {
			// First create a relationship
			relationship := &rbacv1alpha1.Relationship{
				ObjectMeta: metav1.ObjectMeta{
					Name: relationshipName,
				},
				Spec: rbacv1alpha1.RelationshipSpec{
					Tuples: []rbacv1alpha1.Tuple{
						{
							Resource: rbacv1alpha1.ObjectReference{
								ObjectType: "resource",
								ObjectId:   "cluster/cluster4/namespace/_wildcard_",
							},
							Relation: "editor",
							Subject: rbacv1alpha1.SubjectReference{
								Object: rbacv1alpha1.ObjectReference{
									ObjectType: "group",
									ObjectId:   "admin-group",
								},
							},
						},
					},
				},
			}

			_, err := rbacClient.AuthorizationV1alpha1().Relationships().Create(ctx, relationship, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Delete the relationship
			err = rbacClient.AuthorizationV1alpha1().Relationships().Delete(ctx, relationshipName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to delete relationship")

			// Verify deletion
			_, err = rbacClient.AuthorizationV1alpha1().Relationships().Get(ctx, relationshipName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred(), "Expected relationship to be deleted")
		})
	})

	Describe("Multi-tuple Relationships", func() {
		It("should create a relationship with multiple tuples", func() {
			relationship := &rbacv1alpha1.Relationship{
				ObjectMeta: metav1.ObjectMeta{
					Name: relationshipName,
				},
				Spec: rbacv1alpha1.RelationshipSpec{
					Tuples: []rbacv1alpha1.Tuple{
						{
							Resource: rbacv1alpha1.ObjectReference{
								ObjectType: "resource",
								ObjectId:   "cluster/cluster1/namespace/default",
							},
							Relation: "editor",
							Subject: rbacv1alpha1.SubjectReference{
								Object: rbacv1alpha1.ObjectReference{
									ObjectType: "user",
									ObjectId:   "alice",
								},
							},
						},
						{
							Resource: rbacv1alpha1.ObjectReference{
								ObjectType: "resource",
								ObjectId:   "cluster/cluster2/namespace/production",
							},
							Relation: "editor",
							Subject: rbacv1alpha1.SubjectReference{
								Object: rbacv1alpha1.ObjectReference{
									ObjectType: "user",
									ObjectId:   "bob",
								},
							},
						},
					},
				},
			}

			result, err := rbacClient.AuthorizationV1alpha1().Relationships().Create(ctx, relationship, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create relationship with multiple tuples")

			// Verify both tuples were stored
			tuples := result.Spec.Tuples
			Expect(len(tuples)).To(Equal(2), "Expected 2 tuples in the relationship")
		})
	})
})
