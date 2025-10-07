package demo

import (
	"fmt"
	"os"
)

// IsRunningInKubernetes detects if the application is running inside a Kubernetes cluster
// It uses multiple detection methods for robustness:
// 1. Explicit environment variable RUNNING_IN_KUBERNETES=true
// 2. Kubernetes service account token file existence
// 3. KUBERNETES_SERVICE_HOST environment variable
func IsRunningInKubernetes() bool {
	// Method 1: Explicit environment variable (set by Helm chart)
	if os.Getenv("RUNNING_IN_KUBERNETES") == "true" {
		fmt.Println("🔍 K8s mode detected: RUNNING_IN_KUBERNETES=true")
		return true
	}

	// Method 2: Check for Kubernetes service account token (most reliable)
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		fmt.Println("🔍 K8s mode detected: service account token found")
		return true
	}

	// Method 3: Check for KUBERNETES_SERVICE_HOST environment variable
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		fmt.Println("🔍 K8s mode detected: KUBERNETES_SERVICE_HOST set")
		return true
	}

	fmt.Println("🔍 Local mode detected: no Kubernetes environment")
	return false
}

// GetInnominatusURL returns the appropriate innominatus service URL based on deployment mode
func GetInnominatusURL(namespace string) string {
	if IsRunningInKubernetes() {
		// In K8s mode: use service DNS name
		// Format: <service-name>.<namespace>.svc.cluster.local:<port>
		serviceName := os.Getenv("INNOMINATUS_SERVICE_NAME")
		if serviceName == "" {
			serviceName = "innominatus" // Default from Helm chart
		}

		if namespace == "" {
			namespace = "innominatus-system" // Default namespace
		}

		return fmt.Sprintf("http://%s.%s.svc.cluster.local:8081", serviceName, namespace)
	}

	// Local mode: use localhost
	return "http://localhost:8081"
}

// GetComponentIngressConfig returns ingress configuration based on deployment mode
func GetComponentIngressConfig() IngressConfig {
	if IsRunningInKubernetes() {
		// In K8s mode: use actual ingress controller
		return IngressConfig{
			Enabled:     true,
			ClassName:   "nginx",
			BaseHost:    "localtest.me", // Can be overridden via env var
			UsesIngress: true,
		}
	}

	// Local mode: use localtest.me which resolves to 127.0.0.1
	return IngressConfig{
		Enabled:     true,
		ClassName:   "nginx",
		BaseHost:    "localtest.me",
		UsesIngress: true,
	}
}

// IngressConfig holds ingress-related configuration
type IngressConfig struct {
	Enabled     bool
	ClassName   string
	BaseHost    string
	UsesIngress bool
}

// GetDatabaseHost returns the appropriate database host based on deployment mode
func GetDatabaseHost() string {
	// Check for explicit database host override (from Helm values)
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		return dbHost
	}

	if IsRunningInKubernetes() {
		// In K8s mode: use PostgreSQL service from Helm subchart
		// Format: <release-name>-postgresql.<namespace>.svc.cluster.local
		namespace := os.Getenv("POD_NAMESPACE")
		if namespace == "" {
			namespace = "innominatus-system"
		}
		return fmt.Sprintf("innominatus-postgresql.%s.svc.cluster.local", namespace)
	}

	// Local mode: use localhost
	return "localhost"
}

// GetDeploymentMode returns a human-readable deployment mode string
func GetDeploymentMode() string {
	if IsRunningInKubernetes() {
		return "Kubernetes"
	}
	return "Local (Docker Desktop)"
}

// PrintDeploymentInfo prints deployment mode information
func PrintDeploymentInfo() {
	mode := GetDeploymentMode()
	fmt.Printf("\n")
	fmt.Printf("===========================================\n")
	fmt.Printf("  Deployment Mode: %s\n", mode)
	fmt.Printf("===========================================\n")

	if IsRunningInKubernetes() {
		namespace := os.Getenv("POD_NAMESPACE")
		if namespace == "" {
			namespace = "innominatus-system"
		}
		fmt.Printf("  Namespace: %s\n", namespace)
		fmt.Printf("  Service URL: %s\n", GetInnominatusURL(namespace))
		fmt.Printf("  Database: %s\n", GetDatabaseHost())
	} else {
		fmt.Printf("  Database: %s\n", GetDatabaseHost())
		fmt.Printf("  Kube Context: docker-desktop\n")
	}

	fmt.Printf("===========================================\n\n")
}
