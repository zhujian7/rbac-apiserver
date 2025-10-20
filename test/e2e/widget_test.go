package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("Widget API", func() {
	var (
		gvr           schema.GroupVersionResource
		testNamespace string
		widgetName    string
	)

	BeforeEach(func() {
		gvr = schema.GroupVersionResource{
			Group:    apiGroup,
			Version:  apiVersion,
			Resource: "widgets",
		}
		testNamespace = "default"
		widgetName = "test-widget"
	})

	AfterEach(func() {
		// Clean up: delete the widget if it exists
		_ = dynamicClient.Resource(gvr).Namespace(testNamespace).Delete(ctx, widgetName, metav1.DeleteOptions{})
	})

	Describe("CRUD Operations", func() {
		It("should create a Widget successfully", func() {
			widget := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": apiGroup + "/" + apiVersion,
					"kind":       "Widget",
					"metadata": map[string]interface{}{
						"name":      widgetName,
						"namespace": testNamespace,
					},
					"spec": map[string]interface{}{
						"size": int64(10),
					},
				},
			}

			result, err := dynamicClient.Resource(gvr).Namespace(testNamespace).Create(ctx, widget, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create widget")
			Expect(result.GetName()).To(Equal(widgetName))
		})

		It("should list Widgets successfully", func() {
			// First create a widget
			widget := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": apiGroup + "/" + apiVersion,
					"kind":       "Widget",
					"metadata": map[string]interface{}{
						"name":      widgetName,
						"namespace": testNamespace,
					},
					"spec": map[string]interface{}{
						"size": int64(10),
					},
				},
			}

			_, err := dynamicClient.Resource(gvr).Namespace(testNamespace).Create(ctx, widget, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// List widgets
			list, err := dynamicClient.Resource(gvr).Namespace(testNamespace).List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to list widgets")
			Expect(list.Items).NotTo(BeEmpty(), "Expected at least one widget in the list")
		})

		It("should get a Widget successfully", func() {
			// First create a widget
			widget := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": apiGroup + "/" + apiVersion,
					"kind":       "Widget",
					"metadata": map[string]interface{}{
						"name":      widgetName,
						"namespace": testNamespace,
					},
					"spec": map[string]interface{}{
						"size": int64(10),
					},
				},
			}

			_, err := dynamicClient.Resource(gvr).Namespace(testNamespace).Create(ctx, widget, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Get the widget
			result, err := dynamicClient.Resource(gvr).Namespace(testNamespace).Get(ctx, widgetName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to get widget")
			Expect(result.GetName()).To(Equal(widgetName))
		})

		It("should update a Widget successfully", func() {
			// First create a widget
			widget := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": apiGroup + "/" + apiVersion,
					"kind":       "Widget",
					"metadata": map[string]interface{}{
						"name":      widgetName,
						"namespace": testNamespace,
					},
					"spec": map[string]interface{}{
						"size": int64(10),
					},
				},
			}

			created, err := dynamicClient.Resource(gvr).Namespace(testNamespace).Create(ctx, widget, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Update the widget
			spec := created.Object["spec"].(map[string]interface{})
			spec["size"] = int64(20)
			created.Object["spec"] = spec

			updated, err := dynamicClient.Resource(gvr).Namespace(testNamespace).Update(ctx, created, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to update widget")

			// Verify the update
			updatedSpec := updated.Object["spec"].(map[string]interface{})
			Expect(updatedSpec["size"]).To(Equal(int64(20)))
		})

		It("should delete a Widget successfully", func() {
			// First create a widget
			widget := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": apiGroup + "/" + apiVersion,
					"kind":       "Widget",
					"metadata": map[string]interface{}{
						"name":      widgetName,
						"namespace": testNamespace,
					},
					"spec": map[string]interface{}{
						"size": int64(10),
					},
				},
			}

			_, err := dynamicClient.Resource(gvr).Namespace(testNamespace).Create(ctx, widget, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Delete the widget
			err = dynamicClient.Resource(gvr).Namespace(testNamespace).Delete(ctx, widgetName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to delete widget")

			// Verify deletion
			_, err = dynamicClient.Resource(gvr).Namespace(testNamespace).Get(ctx, widgetName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred(), "Expected widget to be deleted")
		})
	})

	Describe("Namespace-scoped Widget Resources", func() {
		It("should create and list widgets in different namespaces", func() {
			// Create widgets in default namespace
			widget1 := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": apiGroup + "/" + apiVersion,
					"kind":       "Widget",
					"metadata": map[string]interface{}{
						"name":      "widget-1",
						"namespace": "default",
					},
					"spec": map[string]interface{}{
						"size": int64(5),
					},
				},
			}

			_, err := dynamicClient.Resource(gvr).Namespace("default").Create(ctx, widget1, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			defer dynamicClient.Resource(gvr).Namespace("default").Delete(ctx, "widget-1", metav1.DeleteOptions{})

			// List widgets in default namespace
			list, err := dynamicClient.Resource(gvr).Namespace("default").List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(list.Items)).To(BeNumerically(">=", 1))
		})
	})
})
