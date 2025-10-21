package main

import (
	"context"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	basecompatibility "k8s.io/component-base/compatibility"
	"k8s.io/klog/v2"

	generatedopenapi "github.com/stolostron/rbac-apiserver/apis/generated/openapi"
	rbacv1alpha1 "github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
	"github.com/stolostron/rbac-apiserver/pkg/registry"
)

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	// Register Relationship API (rbac.open-cluster-management.io/v1alpha1)
	rbacGV := schema.GroupVersion{Group: rbacv1alpha1.GroupName, Version: rbacv1alpha1.APIVersion}
	Scheme.AddKnownTypes(rbacGV, &rbacv1alpha1.Relationship{}, &rbacv1alpha1.RelationshipList{})
	metav1.AddToGroupVersion(Scheme, rbacGV)

	// Register internal version types for PATCH operations
	rbacInternalGV := schema.GroupVersion{Group: rbacv1alpha1.GroupName, Version: runtime.APIVersionInternal}
	Scheme.AddKnownTypes(rbacInternalGV, &rbacv1alpha1.Relationship{}, &rbacv1alpha1.RelationshipList{})

	// Register meta types
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
}

func installAPI(s *genericapiserver.GenericAPIServer) error {
	// Install Relationship API (rbac.open-cluster-management.io/v1alpha1)
	relationshipREST := registry.NewRelationshipREST()

	rbacStorage := map[string]rest.Storage{
		"relationships": relationshipREST,
	}

	rbacAPIGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(rbacv1alpha1.GroupName, Scheme, metav1.ParameterCodec, Codecs)
	rbacAPIGroupInfo.VersionedResourcesStorageMap[rbacv1alpha1.APIVersion] = rbacStorage

	return s.InstallAPIGroup(&rbacAPIGroupInfo)
}

type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
}

type MyAPIServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

func (s *MyAPIServer) Run(ctx context.Context) error {
	return s.GenericAPIServer.PrepareRun().RunWithContext(ctx)
}

func NewConfig() *Config {
	return &Config{
		GenericConfig: genericapiserver.NewRecommendedConfig(Codecs),
	}
}

func (c *Config) Complete() *Config {
	c.GenericConfig.EffectiveVersion = basecompatibility.NewEffectiveVersionFromString("1.30.0", "", "")

	// Configure OpenAPI with generated definitions (includes standard types)
	defNamer := openapi.NewDefinitionNamer(Scheme)
	c.GenericConfig.OpenAPIConfig = genericapiserver.
		DefaultOpenAPIConfig(generatedopenapi.GetOpenAPIDefinitions, defNamer)
	c.GenericConfig.OpenAPIV3Config = genericapiserver.
		DefaultOpenAPIV3Config(generatedopenapi.GetOpenAPIDefinitions, defNamer)

	return c
}

func (c *Config) New() (*MyAPIServer, error) {
	genericServer, err := c.GenericConfig.Complete().New("my-apiserver", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	s := &MyAPIServer{
		GenericAPIServer: genericServer,
	}

	if err := installAPI(s.GenericAPIServer); err != nil {
		return nil, err
	}

	return s, nil
}

func main() {
	klog.InitFlags(nil)

	options := genericoptions.NewRecommendedOptions("", Codecs.LegacyCodec())

	// Now disable etcd for in-memory storage after validation passes
	options.Etcd = nil

	// Disable optional features not available in all clusters
	options.Admission = nil
	options.Features = nil

	options.AddFlags(pflag.CommandLine)

	pflag.Parse()

	if errs := options.Validate(); len(errs) != 0 {
		klog.Errorf("Error validating options: %v", errs)
	}

	config := NewConfig()
	if err := options.ApplyTo(config.GenericConfig); err != nil {
		klog.Fatalf("Error applying options: %v", err)
	}

	config = config.Complete()

	server, err := config.New()
	if err != nil {
		klog.Fatalf("Error creating server: %v", err)
	}

	ctx := context.Background()
	klog.Infof("Starting my-apiserver...")
	if err := server.Run(ctx); err != nil {
		klog.Fatalf("Error running server: %v", err)
	}
}
