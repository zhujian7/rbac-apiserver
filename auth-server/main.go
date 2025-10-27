package main

import (
	"flag"
	"os"

	"k8s.io/klog/v2"
)

func main() {
	klog.InitFlags(nil)

	var (
		hubKubeconfig string
		clusterName   string
		certFile      string
		keyFile       string
		addr          string
	)

	flag.StringVar(&hubKubeconfig, "hub-kubeconfig", "/etc/hub/kubeconfig", "Path to hub cluster kubeconfig")
	flag.StringVar(&clusterName, "cluster-name", os.Getenv("CLUSTER_NAME"), "Managed cluster name")
	flag.StringVar(&certFile, "tls-cert-file", "/etc/certs/tls.crt", "TLS certificate file")
	flag.StringVar(&keyFile, "tls-key-file", "/etc/certs/tls.key", "TLS key file")
	flag.StringVar(&addr, "addr", ":8443", "Server address")
	flag.Parse()

	if clusterName == "" {
		klog.Fatal("cluster-name is required")
	}

	server, err := NewAuthServer(hubKubeconfig, clusterName)
	if err != nil {
		klog.Fatalf("Failed to create auth server: %v", err)
	}

	klog.Infof("Starting authorization webhook server for cluster %s on %s", clusterName, addr)
	if err := server.Start(addr, certFile, keyFile); err != nil {
		klog.Fatalf("Failed to start server: %v", err)
	}
}
