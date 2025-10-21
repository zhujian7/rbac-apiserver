package e2e

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stolostron/rbac-apiserver/apis/generated/clientset/versioned"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeConfig *rest.Config
	rbacClient versioned.Interface
	ctx        context.Context
	cancelFunc context.CancelFunc
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RBAC API Server E2E Suite")
}

var _ = BeforeSuite(func() {
	var err error

	// Create context for tests
	ctx, cancelFunc = context.WithCancel(context.Background())

	// Get kubeconfig
	kubeConfig, err = getKubeConfig()
	Expect(err).NotTo(HaveOccurred(), "Failed to get kubeconfig")

	// Create dynamic client
	rbacClient, err = versioned.NewForConfig(kubeConfig)
	Expect(err).NotTo(HaveOccurred(), "Failed to create dynamic client")
})

var _ = AfterSuite(func() {
	if cancelFunc != nil {
		cancelFunc()
	}
})

// getKubeConfig returns a rest.Config for connecting to the cluster
func getKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
	}
	return config, nil
}
