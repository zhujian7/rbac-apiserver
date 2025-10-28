package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	rbacv1alpha1 "github.com/stolostron/rbac-apiserver/apis/rbac/v1alpha1"
	rbacclient "github.com/stolostron/rbac-apiserver/apis/generated/clientset/versioned"
)

type AuthServer struct {
	hubConfig   *rest.Config
	clusterName string
}

func NewAuthServer(hubKubeconfig, clusterName string) (*AuthServer, error) {
	config, err := clientcmd.BuildConfigFromFlags("", hubKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build hub config: %w", err)
	}

	// Set reasonable timeouts for authz checks
	config.Timeout = 5 * time.Second

	// Test connection
	_, err = rbacclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create hub client: %w", err)
	}

	klog.Infof("Successfully connected to hub rbac-apiserver")
	return &AuthServer{
		hubConfig:   config,
		clusterName: clusterName,
	}, nil
}

func (s *AuthServer) Start(addr, certFile, keyFile string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/authorize", s.HandleAuthorization)
	mux.HandleFunc("/healthz", s.HandleHealth)

	klog.Infof("Server listening on %s", addr)
	return http.ListenAndServeTLS(addr, certFile, keyFile, mux)
}

func (s *AuthServer) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *AuthServer) HandleAuthorization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Decode SubjectAccessReview from managed cluster kube-apiserver
	var sar authzv1.SubjectAccessReview
	if err := json.NewDecoder(r.Body).Decode(&sar); err != nil {
		klog.Errorf("Failed to decode SubjectAccessReview: %v", err)
		s.respondError(w, &sar, "Invalid request format")
		return
	}

	klog.V(4).Infof("Authorization request: user=%s, resource=%s/%s, verb=%s, namespace=%s, name=%s",
		sar.Spec.User,
		sar.Spec.ResourceAttributes.Group,
		sar.Spec.ResourceAttributes.Resource,
		sar.Spec.ResourceAttributes.Verb,
		sar.Spec.ResourceAttributes.Namespace,
		sar.Spec.ResourceAttributes.Name)

	// Check permission via hub rbac-apiserver
	allowed, reason, err := s.checkPermission(ctx, &sar)
	if err != nil {
		klog.Errorf("Permission check failed: %v", err)
		// On error, deny access by default (fail-closed)
		s.respondDenied(w, &sar, fmt.Sprintf("Authorization error: %v", err))
		return
	}

	// Respond with result
	if allowed {
		s.respondAllowed(w, &sar, reason)
	} else {
		// Return NoOpinion instead of Denied to allow RBAC chain to continue
		s.respondNoOpinion(w, &sar, reason)
	}
}

func (s *AuthServer) checkPermission(ctx context.Context, sar *authzv1.SubjectAccessReview) (bool, string, error) {
	// Create PermissionRequest for hub API
	pr := &rbacv1alpha1.PermissionRequest{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("authz-%s-", s.clusterName),
		},
		Spec: rbacv1alpha1.PermissionRequestSpec{
			Group:       sar.Spec.ResourceAttributes.Group,
			Resource:    sar.Spec.ResourceAttributes.Resource,
			SubResource: sar.Spec.ResourceAttributes.Subresource,
			Verb:        sar.Spec.ResourceAttributes.Verb,
			Cluster:     s.clusterName,
			Namespace:   sar.Spec.ResourceAttributes.Namespace,
			Name:        sar.Spec.ResourceAttributes.Name,
		},
	}

	klog.V(5).Infof("Creating PermissionRequest on hub for user %s: %+v", sar.Spec.User, pr.Spec)

	// Create a client with user impersonation
	// This allows the rbac-apiserver to evaluate permissions for the actual user
	impersonatedConfig := rest.CopyConfig(s.hubConfig)

	// Convert Extra from authzv1.ExtraValue to []string
	extra := make(map[string][]string)
	for k, v := range sar.Spec.Extra {
		extra[k] = []string(v)
	}

	impersonatedConfig.Impersonate = rest.ImpersonationConfig{
		UserName: sar.Spec.User,
		Groups:   sar.Spec.Groups,
		Extra:    extra,
	}

	impersonatedClient, err := rbacclient.NewForConfig(impersonatedConfig)
	if err != nil {
		return false, "", fmt.Errorf("failed to create impersonated client: %w", err)
	}

	// Call hub rbac-apiserver with impersonation
	// Note: PermissionRequest is ephemeral (like CSR) - it's evaluated immediately and not persisted
	result, err := impersonatedClient.AuthorizationV1alpha1().PermissionRequests().Create(ctx, pr, metav1.CreateOptions{})
	if err != nil {
		return false, "", fmt.Errorf("failed to create PermissionRequest: %w", err)
	}

	// Parse status to determine if access is allowed
	allowed := s.isAllowed(result.Status.AllowedList, pr.Spec)

	reason := s.buildReason(allowed)

	klog.V(4).Infof("Authorization result for user %s: allowed=%v", sar.Spec.User, allowed)
	return allowed, reason, nil
}

func (s *AuthServer) isAllowed(allowedList []rbacv1alpha1.AllowedItem, spec rbacv1alpha1.PermissionRequestSpec) bool {
	if len(allowedList) == 0 {
		return false
	}

	for _, item := range allowedList {
		// Check cluster match
		if spec.Cluster != "" && item.Cluster != spec.Cluster && item.Cluster != "*" {
			continue
		}

		for _, nsName := range item.NamespacedNames {
			// Check namespace match
			if spec.Namespace != "" && nsName.Namespace != spec.Namespace && nsName.Namespace != "*" {
				continue
			}

			// Check name match
			for _, name := range nsName.Names {
				if name == "*" || spec.Name == "" || spec.Name == name {
					return true
				}
			}
		}
	}

	return false
}

func (s *AuthServer) buildReason(allowed bool) string {
	if allowed {
		return fmt.Sprintf("Allowed by hub RBAC policy in cluster %s", s.clusterName)
	}
	return fmt.Sprintf("No hub RBAC policy for this resource in cluster %s, deferring to local RBAC", s.clusterName)
}

func (s *AuthServer) respondAllowed(w http.ResponseWriter, sar *authzv1.SubjectAccessReview, reason string) {
	sar.Status = authzv1.SubjectAccessReviewStatus{
		Allowed: true,
		Reason:  reason,
	}
	s.respond(w, sar)
}

func (s *AuthServer) respondDenied(w http.ResponseWriter, sar *authzv1.SubjectAccessReview, reason string) {
	sar.Status = authzv1.SubjectAccessReviewStatus{
		Allowed: false,
		Denied:  true,
		Reason:  reason,
	}
	s.respond(w, sar)
}

func (s *AuthServer) respondNoOpinion(w http.ResponseWriter, sar *authzv1.SubjectAccessReview, reason string) {
	// NoOpinion: neither Allowed nor Denied
	// This allows the authorization chain to continue to the next authorizer (e.g., RBAC)
	sar.Status = authzv1.SubjectAccessReviewStatus{
		Allowed: false,
		Denied:  false,
		Reason:  reason,
	}
	s.respond(w, sar)
}

func (s *AuthServer) respondError(w http.ResponseWriter, sar *authzv1.SubjectAccessReview, reason string) {
	// On error, deny by default (fail-closed security policy)
	s.respondDenied(w, sar, reason)
}

func (s *AuthServer) respond(w http.ResponseWriter, sar *authzv1.SubjectAccessReview) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sar); err != nil {
		klog.Errorf("Failed to encode response: %v", err)
	}
}
