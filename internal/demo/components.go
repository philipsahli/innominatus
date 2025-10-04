package demo

import (
	"fmt"
)

// DemoComponent represents a component in the demo environment
type DemoComponent struct {
	Name        string
	Namespace   string
	Chart       string
	Repo        string
	Version     string
	IngressHost string
	Credentials map[string]string
	Values      map[string]interface{}
	HealthPath  string
	Port        int
}

// DemoEnvironment holds all components and configuration
type DemoEnvironment struct {
	Components      []DemoComponent
	IngressClass    string
	KubeContext     string
	BaseLocalDomain string
}

// NewDemoEnvironment creates a new demo environment configuration
func NewDemoEnvironment() *DemoEnvironment {
	return &DemoEnvironment{
		IngressClass:    "nginx",
		KubeContext:     "docker-desktop",
		BaseLocalDomain: "localtest.me",
		Components: []DemoComponent{
			{
				Name:        "nginx-ingress",
				Namespace:   "ingress-nginx",
				Chart:       "ingress-nginx",
				Repo:        "https://kubernetes.github.io/ingress-nginx",
				Version:     "4.8.3",
				IngressHost: "",
				Credentials: map[string]string{},
				Values: map[string]interface{}{
					"controller": map[string]interface{}{
						"service": map[string]interface{}{
							"type": "LoadBalancer",
						},
						"admissionWebhooks": map[string]interface{}{
							"enabled": false,
						},
					},
				},
				HealthPath: "/healthz",
				Port:       80,
			},
			{
				Name:        "gitea",
				Namespace:   "gitea",
				Chart:       "gitea",
				Repo:        "https://dl.gitea.com/charts/",
				Version:     "9.6.1",
				IngressHost: "gitea.localtest.me",
				Credentials: map[string]string{
					"username": "giteaadmin",
					"password": "admin",
				},
				Values: map[string]interface{}{
					"gitea": map[string]interface{}{
						"admin": map[string]interface{}{
							"username": "giteaadmin",
							"password": "admin",
							"email":    "giteaadmin@localtest.me",
						},
						"config": map[string]interface{}{
							"server": map[string]interface{}{
								"DOMAIN":      "gitea.localtest.me",
								"ROOT_URL":    "http://gitea.localtest.me",
								"HTTP_PORT":   3000,
								"DISABLE_SSH": true,
							},
							"service": map[string]interface{}{
								"DISABLE_REGISTRATION": true,
							},
						},
					},
					"ingress": map[string]interface{}{
						"enabled":   true,
						"className": "nginx",
						"hosts": []map[string]interface{}{
							{
								"host": "gitea.localtest.me",
								"paths": []map[string]interface{}{
									{
										"path":     "/",
										"pathType": "Prefix",
									},
								},
							},
						},
					},
					"postgresql": map[string]interface{}{
						"enabled": true,
						"auth": map[string]interface{}{
							"database":         "gitea",
							"username":         "gitea",
							"password":         "gitea",
							"postgresPassword": "postgres",
						},
					},
					"postgresql-ha": map[string]interface{}{
						"enabled": false,
					},
				},
				HealthPath: "/api/healthz",
				Port:       3000,
			},
			{
				Name:        "keycloak",
				Namespace:   "keycloak",
				Chart:       "oci://registry-1.docker.io/cloudpirates/keycloak",
				Repo:        "",
				Version:     "0.1.10",
				IngressHost: "keycloak.localtest.me",
				Credentials: map[string]string{
					"username": "admin",
					"password": "adminpassword",
					"realm":    "demo-realm",
					"users":    "demo-user/password123, test-user/test123",
				},
				Values: map[string]interface{}{
					"keycloak": map[string]interface{}{
						"adminUser":     "admin",
						"adminPassword": "adminpassword",
						"httpEnabled":   true,
						"hostname":      "keycloak.localtest.me",
					},
					"database": map[string]interface{}{
						"type":     "postgres",
						"host":     "keycloak-postgresql",
						"port":     "5432",
						"name":     "keycloak",
						"username": "keycloak",
						"password": "keycloak123",
					},
					"ingress": map[string]interface{}{
						"enabled":   true,
						"className": "nginx",
						"hosts": []map[string]interface{}{
							{
								"host": "keycloak.localtest.me",
								"paths": []map[string]interface{}{
									{
										"path":     "/",
										"pathType": "Prefix",
									},
								},
							},
						},
					},
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"memory": "512Mi",
							"cpu":    "500m",
						},
						"limits": map[string]interface{}{
							"memory": "1Gi",
							"cpu":    "1000m",
						},
					},
				},
				HealthPath: "/",
				Port:       8080,
			},
			{
				Name:        "argocd",
				Namespace:   "argocd",
				Chart:       "argo-cd",
				Repo:        "https://argoproj.github.io/argo-helm",
				Version:     "5.51.6",
				IngressHost: "argocd.localtest.me",
				Credentials: map[string]string{
					"username": "admin",
					"password": "admin123",
				},
				Values: map[string]interface{}{
					"server": map[string]interface{}{
						"ingress": map[string]interface{}{
							"enabled":          true,
							"ingressClassName": "nginx",
							"hosts":            []string{"argocd.localtest.me"},
							"annotations": map[string]string{
								"nginx.ingress.kubernetes.io/ssl-redirect":     "false",
								"nginx.ingress.kubernetes.io/backend-protocol": "HTTP",
							},
						},
						"extraArgs": []string{
							"--insecure",
						},
					},
					"configs": map[string]interface{}{
						"secret": map[string]interface{}{
							"argocdServerAdminPassword": "$2a$12$CPhilZvs2GgHLYyXet.oMOOLDswMubNr7jtvWzxGpMR6YO6cpA3De", // admin123
						},
					},
				},
				HealthPath: "/healthz",
				Port:       8080,
			},
			{
				Name:        "vault",
				Namespace:   "vault",
				Chart:       "vault",
				Repo:        "https://helm.releases.hashicorp.com",
				Version:     "0.27.0",
				IngressHost: "vault.localtest.me",
				Credentials: map[string]string{
					"root_token": "root",
				},
				Values: map[string]interface{}{
					"server": map[string]interface{}{
						"dev": map[string]interface{}{
							"enabled":      true,
							"devRootToken": "root",
						},
						"ingress": map[string]interface{}{
							"enabled":          true,
							"ingressClassName": "nginx",
							"hosts": []map[string]interface{}{
								{
									"host":  "vault.localtest.me",
									"paths": []string{"/"},
								},
							},
						},
					},
				},
				HealthPath: "/v1/sys/health",
				Port:       8200,
			},
			{
				Name:        "vault-secrets-operator",
				Namespace:   "vault-secrets-operator-system",
				Chart:       "vault-secrets-operator",
				Repo:        "https://helm.releases.hashicorp.com",
				Version:     "0.4.3",
				IngressHost: "",
				Credentials: map[string]string{},
				Values: map[string]interface{}{
					"defaultVaultConnection": map[string]interface{}{
						"enabled": true,
						"address": "http://vault.vault.svc.cluster.local:8200",
						"headers": map[string]interface{}{},
					},
					"defaultAuthMethod": map[string]interface{}{
						"enabled":   true,
						"namespace": "vault-secrets-operator-system",
						"method":    "kubernetes",
						"mount":     "kubernetes",
						"kubernetes": map[string]interface{}{
							"role":                   "vault-secrets-operator",
							"serviceAccount":         "vault-secrets-operator",
							"audiences":              []string{"vault"},
							"tokenExpirationSeconds": 600,
						},
					},
				},
				HealthPath: "",
				Port:       8080,
			},
			{
				Name:        "prometheus",
				Namespace:   "monitoring",
				Chart:       "prometheus",
				Repo:        "https://prometheus-community.github.io/helm-charts",
				Version:     "25.8.0",
				IngressHost: "prometheus.localtest.me",
				Credentials: map[string]string{},
				Values: map[string]interface{}{
					"server": map[string]interface{}{
						"ingress": map[string]interface{}{
							"enabled":          true,
							"ingressClassName": "nginx",
							"hosts":            []string{"prometheus.localtest.me"},
						},
					},
					"alertmanager": map[string]interface{}{
						"enabled": false,
					},
				},
				HealthPath: "/-/healthy",
				Port:       9090,
			},
			{
				Name:        "grafana",
				Namespace:   "monitoring",
				Chart:       "grafana",
				Repo:        "https://grafana.github.io/helm-charts",
				Version:     "7.0.19",
				IngressHost: "grafana.localtest.me",
				Credentials: map[string]string{
					"username": "admin",
					"password": "admin",
				},
				Values: map[string]interface{}{
					"adminUser":     "admin",
					"adminPassword": "admin",
					"ingress": map[string]interface{}{
						"enabled":          true,
						"ingressClassName": "nginx",
						"hosts":            []string{"grafana.localtest.me"},
					},
					"datasources": map[string]interface{}{
						"datasources.yaml": map[string]interface{}{
							"apiVersion": 1,
							"datasources": []map[string]interface{}{
								{
									"name":      "Prometheus",
									"type":      "prometheus",
									"url":       "http://prometheus-server.monitoring.svc.cluster.local",
									"access":    "proxy",
									"isDefault": true,
								},
							},
						},
					},
				},
				HealthPath: "/api/health",
				Port:       3000,
			},
			{
				Name:        "minio",
				Namespace:   "minio-system",
				Chart:       "minio",
				Repo:        "https://charts.min.io/",
				Version:     "5.0.14",
				IngressHost: "minio.localtest.me",
				Credentials: map[string]string{
					"username": "minioadmin",
					"password": "minioadmin",
					"console":  "http://minio-console.localtest.me",
				},
				Values: map[string]interface{}{
					"mode":         "standalone",
					"rootUser":     "minioadmin",
					"rootPassword": "minioadmin",
					"replicas":     1,
					"persistence": map[string]interface{}{
						"enabled": true,
						"size":    "10Gi",
					},
					"ingress": map[string]interface{}{
						"enabled":          true,
						"ingressClassName": "nginx",
						"hosts":            []string{"minio.localtest.me"},
						"path":             "/",
					},
					"consoleIngress": map[string]interface{}{
						"enabled":          true,
						"ingressClassName": "nginx",
						"hosts":            []string{"minio-console.localtest.me"},
						"path":             "/",
					},
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"memory": "512Mi",
						},
					},
				},
				HealthPath: "/minio/health/live",
				Port:       9000,
			},
			{
				Name:        "backstage",
				Namespace:   "backstage",
				Chart:       "backstage",
				Repo:        "https://backstage.github.io/charts",
				Version:     "2.6.1",
				IngressHost: "backstage.localtest.me",
				Credentials: map[string]string{
					"info": "Demo mode - no authentication required",
				},
				Values: map[string]interface{}{
					"backstage": map[string]interface{}{
						"image": map[string]interface{}{
							"pullPolicy": "IfNotPresent",
						},
						"appConfig": map[string]interface{}{
							"app": map[string]interface{}{
								"baseUrl": "http://backstage.localtest.me",
								"title":   "OpenAlps Developer Portal",
							},
							"backend": map[string]interface{}{
								"baseUrl": "http://backstage.localtest.me",
								"listen": map[string]interface{}{
									"port": 7007,
								},
								"cors": map[string]interface{}{
									"origin":      "http://backstage.localtest.me",
									"methods":     []string{"GET", "HEAD", "PATCH", "POST", "PUT", "DELETE"},
									"credentials": true,
								},
								"database": map[string]interface{}{
									"client": "better-sqlite3",
									"connection": map[string]interface{}{
										"filename": ":memory:",
									},
								},
								"reading": map[string]interface{}{
									"allow": []map[string]string{
										{"host": "gitea-http.gitea.svc.cluster.local:3000"},
										{"host": "gitea-http.gitea.svc.cluster.local"},
										{"host": "gitea.localtest.me"},
									},
								},
							},
							"organization": map[string]interface{}{
								"name": "OpenAlps Demo",
							},
							"catalog": map[string]interface{}{
								"import": map[string]interface{}{
									"entityFilename":        "catalog-info.yaml",
									"pullRequestBranchName": "backstage-integration",
								},
								"rules": []map[string]interface{}{
									{"allow": []string{"Component", "System", "API", "Resource", "Location", "Template"}},
								},
							},
							"techdocs": map[string]interface{}{
								"builder": "local",
								"generator": map[string]interface{}{
									"runIn": "local",
								},
								"publisher": map[string]interface{}{
									"type": "local",
								},
							},
							"auth": map[string]interface{}{
								"environment": "development",
								"providers": map[string]interface{}{
									"guest": map[string]interface{}{
										"dangerouslyAllowOutsideDevelopment": true,
									},
								},
							},
							"permission": map[string]interface{}{
								"enabled": false,
							},
							"integrations": map[string]interface{}{
								"gitea": []map[string]interface{}{
									{
										"host":     "gitea-http.gitea.svc.cluster.local:3000",
										"username": "giteaadmin",
										"password": "admin",
									},
								},
							},
						},
					},
					"ingress": map[string]interface{}{
						"enabled":   true,
						"className": "nginx",
						"host":      "backstage.localtest.me",
						"annotations": map[string]string{
							"nginx.ingress.kubernetes.io/ssl-redirect": "false",
						},
					},
					"postgresql": map[string]interface{}{
						"enabled": false,
					},
					"service": map[string]interface{}{
						"type": "ClusterIP",
						"ports": map[string]interface{}{
							"backend": 7007,
						},
					},
				},
				HealthPath: "/healthcheck",
				Port:       7007,
			},
			{
				Name:        "demo-app",
				Namespace:   "demo",
				Chart:       "", // Will be deployed via kubectl
				Repo:        "",
				Version:     "",
				IngressHost: "demo.localtest.me",
				Credentials: map[string]string{},
				Values:      map[string]interface{}{},
				HealthPath:  "/",
				Port:        80,
			},
			{
				Name:        "kubernetes-dashboard",
				Namespace:   "kubernetes-dashboard",
				Chart:       "", // Will be deployed via kubectl manifest
				Repo:        "",
				Version:     "v2.7.0",
				IngressHost: "k8s.localtest.me",
				Credentials: map[string]string{
					"instructions": "kubectl -n kubernetes-dashboard create token admin-user",
				},
				Values:     map[string]interface{}{},
				HealthPath: "/",
				Port:       443,
			},
		},
	}
}

// GetComponent returns a component by name
func (d *DemoEnvironment) GetComponent(name string) (*DemoComponent, error) {
	for _, component := range d.Components {
		if component.Name == name {
			return &component, nil
		}
	}
	return nil, fmt.Errorf("component %s not found", name)
}

// GetHelmComponents returns only components that use Helm
func (d *DemoEnvironment) GetHelmComponents() []DemoComponent {
	var helmComponents []DemoComponent
	for _, component := range d.Components {
		if component.Chart != "" {
			helmComponents = append(helmComponents, component)
		}
	}
	return helmComponents
}

// GetIngressComponents returns components that have ingress hosts
func (d *DemoEnvironment) GetIngressComponents() []DemoComponent {
	var ingressComponents []DemoComponent
	for _, component := range d.Components {
		if component.IngressHost != "" {
			ingressComponents = append(ingressComponents, component)
		}
	}
	return ingressComponents
}

// GetSystemComponents returns system components (both ingress and non-ingress)
func (d *DemoEnvironment) GetSystemComponents() []DemoComponent {
	var systemComponents []DemoComponent
	for _, component := range d.Components {
		// Include ingress components and specific system components like VSO
		if component.IngressHost != "" || component.Name == "vault-secrets-operator" {
			systemComponents = append(systemComponents, component)
		}
	}
	return systemComponents
}
