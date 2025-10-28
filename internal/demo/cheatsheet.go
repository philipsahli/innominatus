package demo

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"
)

// NOTE: This file contains numerous fmt.Println/Printf statements that are INTENTIONAL
// user-facing output for displaying demo environment credentials and status. These are
// NOT debug logging and should NOT be converted to structured logging. They provide
// formatted terminal output for the demo cheat sheet display.

// CheatSheet handles the display of demo environment status and credentials
type CheatSheet struct {
	environment *DemoEnvironment
}

// NewCheatSheet creates a new cheat sheet instance
func NewCheatSheet(env *DemoEnvironment) *CheatSheet {
	return &CheatSheet{
		environment: env,
	}
}

// PrintWelcome prints the welcome message
func (c *CheatSheet) PrintWelcome() {
	fmt.Println()
	fmt.Println("Innominatus Demo Application")
	fmt.Println("===============================")
	fmt.Println()
}

// PrintInstallationComplete prints the installation complete message
func (c *CheatSheet) PrintInstallationComplete() {
	fmt.Println()
	fmt.Println("🎉 Demo Environment Installation Complete!")
	fmt.Println()
	fmt.Println("Your Innominatus demo environment is now ready with:")
	fmt.Println("• GitOps workflow with ArgoCD")
	fmt.Println("• Git repository with Gitea")
	fmt.Println("• Secret management with Vault")
	fmt.Println("• Object storage with Minio (S3-compatible)")
	fmt.Println("• Monitoring with Prometheus & Grafana")
	fmt.Println("• Sample application deployment")
	fmt.Println()
}

// PrintStatus prints the current status of all services
func (c *CheatSheet) PrintStatus(healthResults []HealthStatus) {
	fmt.Println("📊 Service Status")
	fmt.Println("==================")
	fmt.Println()

	// Calculate overall health
	healthy := 0
	total := len(healthResults)
	for _, result := range healthResults {
		if result.Healthy {
			healthy++
		}
	}

	// Print each service status
	for _, result := range healthResults {
		status := "❌"
		if result.Healthy {
			status = "✅"
		}

		latency := ""
		if result.Latency > 0 {
			latency = fmt.Sprintf(" (%dms)", result.Latency.Milliseconds())
		}

		// Get component for additional info
		component, _ := c.environment.GetComponent(result.Name)
		if component != nil && component.IngressHost != "" {
			fmt.Printf("%s %-20s http://%-25s %s%s\n",
				status,
				c.formatServiceName(result.Name),
				result.Host,
				result.Status,
				latency)
		} else {
			fmt.Printf("%s %-20s %s%s\n",
				status,
				c.formatServiceName(result.Name),
				result.Status,
				latency)
		}
	}

	fmt.Println()

	// Print overall status
	if healthy == total {
		fmt.Printf("🎉 All services healthy (%d/%d)\n", healthy, total)
	} else {
		fmt.Printf("⚠️  %d/%d services healthy\n", healthy, total)
	}
	fmt.Println()
}

// PrintCredentials prints the login credentials for all services
func (c *CheatSheet) PrintCredentials() {
	fmt.Println("🔐 Service Credentials")
	fmt.Println("======================")
	fmt.Println()

	credentialsFound := false

	for _, component := range c.environment.Components {
		if len(component.Credentials) > 0 && component.IngressHost != "" {
			credentialsFound = true
			fmt.Printf("🌐 %-15s http://%-25s\n", c.formatServiceName(component.Name)+":", component.IngressHost)

			for key, value := range component.Credentials {
				fmt.Printf("   %-12s %s\n", key+":", value)
			}
			fmt.Println()
		}
	}

	if !credentialsFound {
		fmt.Println("No services with credentials configured.")
		fmt.Println()
	}

	// Print Keycloak SSO information
	fmt.Println("🔑 Single Sign-On (Keycloak)")
	fmt.Println("=============================")
	fmt.Println()
	fmt.Printf("🌐 %-15s http://%-25s\n", "Keycloak:", "keycloak.localtest.me")
	fmt.Printf("   %-12s %s\n", "admin:", "admin")
	fmt.Printf("   %-12s %s\n", "password:", "adminpassword")
	fmt.Println()
	fmt.Println("📝 Demo Users (for SSO login):")
	fmt.Printf("   %-12s %s / %s\n", "demo-user:", "demo-user", "password123")
	fmt.Printf("   %-12s %s / %s\n", "test-user:", "test-user", "test123")
	fmt.Println()
	// Get appropriate innominatus URL based on deployment mode
	innominatusURL := "http://localhost:8081"
	if IsRunningInKubernetes() {
		namespace := os.Getenv("POD_NAMESPACE")
		if namespace == "" {
			namespace = "innominatus-system"
		}
		innominatusURL = GetInnominatusURL(namespace)
	}

	fmt.Println("🔗 OIDC-Enabled Services:")
	fmt.Println("   • ArgoCD      - Login with Keycloak button")
	fmt.Println("   • Grafana     - Login with Keycloak button")
	fmt.Println("   • Backstage   - OIDC option on login page")
	fmt.Println("   • Gitea       - Sign in with Keycloak option")
	fmt.Println("   • Vault       - OIDC auth method enabled")
	fmt.Printf("   • Innominatus - Visit %s/auth/oidc/login\n", innominatusURL)
	fmt.Println()
	fmt.Println("💡 Tip: Local admin login still works for all services")
	fmt.Println()
}

// PrintQuickStart prints quick start instructions
func (c *CheatSheet) PrintQuickStart() {
	fmt.Println("🚀 Quick Start Guide")
	fmt.Println("====================")
	fmt.Println()
	fmt.Println("1. Access Gitea (Git Repository):")
	fmt.Println("   • URL: http://gitea.localtest.me")
	fmt.Println("   • Login: giteaadmin / admin123")
	fmt.Println("   • Check the 'platform-config' repository")
	fmt.Println()
	fmt.Println("2. Access ArgoCD (GitOps):")
	fmt.Println("   • URL: http://argocd.localtest.me")
	fmt.Println("   • Login Methods:")
	fmt.Println("     - Admin: admin / admin123")
	fmt.Println("     - OIDC: Click 'LOG IN VIA KEYCLOAK' (demo-user / password123)")
	fmt.Println("   • View deployed applications")
	fmt.Println()
	fmt.Println("3. Access Keycloak (Identity Provider):")
	fmt.Println("   • URL: http://keycloak.localtest.me")
	fmt.Println("   • Admin: admin / adminpassword")
	fmt.Println("   • Realm: demo-realm")
	fmt.Println("   • Demo Users:")
	fmt.Println("     - demo-user / password123")
	fmt.Println("     - test-user / test123")
	fmt.Println()
	fmt.Println("4. Access Grafana (Monitoring):")
	fmt.Println("   • URL: http://grafana.localtest.me")
	fmt.Println("   • Login: admin / admin")
	fmt.Println("   • Explore pre-configured dashboards")
	fmt.Println()
	fmt.Println("5. Access Minio (Object Storage):")
	fmt.Println("   • API: http://minio.localtest.me")
	fmt.Println("   • Console: http://minio-console.localtest.me")
	fmt.Println("   • Login: minioadmin / minioadmin")
	fmt.Println("   • S3-compatible object storage for applications")
	fmt.Println()
	fmt.Println("6. Access Backstage (Developer Portal):")
	fmt.Println("   • URL: http://backstage.localtest.me")
	fmt.Println("   • Demo mode - no authentication required")
	fmt.Println("   • Software catalog and developer portal")
	fmt.Println()
	fmt.Println("7. Access Kubernetes Dashboard:")
	fmt.Println("   • URL: http://k8s.localtest.me")
	fmt.Println("   • Login token: kubectl -n kubernetes-dashboard create token admin-user")
	fmt.Println("   • View cluster resources and workloads")
	fmt.Println()
	fmt.Println("8. Access Demo Application:")
	fmt.Println("   • URL: http://demo.localtest.me")
	fmt.Println("   • Sample app deployed via Score spec")
	fmt.Println()
	// Get innominatus URL dynamically
	innominatusURL2 := "http://localhost:8081"
	if IsRunningInKubernetes() {
		namespace := os.Getenv("POD_NAMESPACE")
		if namespace == "" {
			namespace = "innominatus-system"
		}
		innominatusURL2 = GetInnominatusURL(namespace)
	}

	fmt.Println("9. Access Innominatus Server (with SSO):")
	fmt.Printf("   • URL: %s\n", innominatusURL2)
	fmt.Println("   • Login Methods:")
	fmt.Printf("     - SSO: %s/auth/oidc/login (demo-user / password123)\n", innominatusURL2)
	fmt.Println("     - Local: users.yaml configuration")
	fmt.Println("   • OIDC: Auto-enabled when demo-time is running!")
	fmt.Println("     Innominatus automatically detects Keycloak and enables SSO login")
	fmt.Println()
}

// PrintProviderInfo prints information about installed providers
func (c *CheatSheet) PrintProviderInfo() {
	fmt.Println("📦 Provider Configuration")
	fmt.Println("=========================")
	fmt.Println()
	fmt.Println("Admin configuration file created: admin-config.yaml")
	fmt.Println()
	fmt.Println("Available Providers:")
	fmt.Println("  • builtin (filesystem) - Active for local development")
	fmt.Println("    Path: ./providers/builtin")
	fmt.Println("    Resources: PostgreSQL, Redis, volumes, routes, vault-space")
	fmt.Println()
	fmt.Println("  • storage-team (git) - Available as git-based example")
	fmt.Println("    Repository: http://gitea.localtest.me/giteaadmin/platform-config")
	fmt.Println("    Path: providers/builtin (copy to providers/storage-team to customize)")
	fmt.Println()
	fmt.Println("Provider Loading:")
	fmt.Println("  1. Filesystem providers (development):")
	fmt.Println("     - Load from local ./providers/ directory")
	fmt.Println("     - Configured in admin-config.yaml with type: filesystem")
	fmt.Println()
	fmt.Println("  2. Git-based providers (production):")
	fmt.Println("     - Load from Git repository")
	fmt.Println("     - Configured in admin-config.yaml with type: git")
	fmt.Println("     - Example in platform-config repo: providers/builtin/")
	fmt.Println()
	fmt.Println("To use git-based provider:")
	fmt.Println("  1. Customize provider in Gitea: http://gitea.localtest.me/giteaadmin/platform-config")
	fmt.Println("  2. Edit admin-config.yaml to enable git provider")
	fmt.Println("  3. Restart Innominatus server to load changes")
	fmt.Println()
}

// PrintCommands prints useful commands
func (c *CheatSheet) PrintCommands() {
	fmt.Println("💡 Useful Commands")
	fmt.Println("==================")
	fmt.Println()
	fmt.Println("Check demo status:")
	fmt.Printf("  %s demo-status\n", c.getCliName())
	fmt.Println()
	fmt.Println("Remove demo environment:")
	fmt.Printf("  %s demo-nuke\n", c.getCliName())
	fmt.Println()
	fmt.Println("Reinstall demo environment:")
	fmt.Printf("  %s demo-time\n", c.getCliName())
	fmt.Println()
	fmt.Println("View Kubernetes resources:")
	fmt.Println("  kubectl get pods -A")
	fmt.Println("  kubectl get ingress -A")
	fmt.Println()
	fmt.Println("View Helm releases:")
	fmt.Println("  helm list -A")
	fmt.Println()
}

// PrintNukeComplete prints the nuke completion message
func (c *CheatSheet) PrintNukeComplete() {
	fmt.Println()
	fmt.Println("💥 Demo Environment Cleanup Complete!")
	fmt.Println()
	fmt.Println("All demo components have been removed:")
	fmt.Println("• Helm releases uninstalled")
	fmt.Println("• Namespaces deleted")
	fmt.Println("• PVCs and secrets cleaned up")
	fmt.Println()
	fmt.Println("The platform-config repository in Gitea has been preserved.")
	fmt.Println()
	fmt.Printf("To reinstall: %s demo-time\n", c.getCliName())
	fmt.Println()
}

// PrintError prints an error message
func (c *CheatSheet) PrintError(operation string, err error) {
	fmt.Println()
	fmt.Printf("❌ %s Failed\n", operation)
	fmt.Println(strings.Repeat("=", len(operation)+9))
	fmt.Println()
	fmt.Printf("Error: %v\n", err)
	fmt.Println()
	fmt.Println("💡 Troubleshooting:")
	fmt.Println("• Ensure Docker Desktop is running")
	fmt.Println("• Check Kubernetes context: kubectl config current-context")
	fmt.Println("• Verify Helm is installed: helm version")
	fmt.Println("• Check network connectivity")
	fmt.Println()
}

// PrintProgress prints a progress message
func (c *CheatSheet) PrintProgress(message string) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] %s\n", timestamp, message)
}

// PrintSeparator prints a visual separator
func (c *CheatSheet) PrintSeparator() {
	fmt.Println(strings.Repeat("─", 60))
}

// PrintFooter prints the footer with links and support info
func (c *CheatSheet) PrintFooter() {
	fmt.Println("📚 Resources")
	fmt.Println("============")
	fmt.Println()
	fmt.Println("Documentation: https://github.com/your-org/openalps")
	fmt.Println("Issues: https://github.com/your-org/openalps/issues")
	fmt.Println("Score Spec: https://score.dev")
	fmt.Println()
	fmt.Println("Happy coding! 🎉")
	fmt.Println()
}

// PrintCompactStatus prints a compact one-line status
func (c *CheatSheet) PrintCompactStatus(healthResults []HealthStatus) {
	healthy := 0
	total := len(healthResults)

	for _, result := range healthResults {
		if result.Healthy {
			healthy++
		}
	}

	status := "🟢"
	if healthy == 0 {
		status = "🔴"
	} else if healthy < total {
		status = "🟡"
	}

	fmt.Printf("%s OpenAlps Demo: %d/%d services healthy", status, healthy, total)

	if healthy < total {
		unhealthy := []string{}
		for _, result := range healthResults {
			if !result.Healthy {
				unhealthy = append(unhealthy, result.Name)
			}
		}
		fmt.Printf(" (issues: %s)", strings.Join(unhealthy, ", "))
	}

	fmt.Println()
}

// formatServiceName formats service names for display
func (c *CheatSheet) formatServiceName(name string) string {
	switch name {
	case "gitea":
		return "Gitea"
	case "argocd":
		return "ArgoCD"
	case "vault":
		return "Vault"
	case "vault-secrets-operator":
		return "Vault Secrets Operator"
	case "grafana":
		return "Grafana"
	case "prometheus":
		return "Prometheus"
	case "pushgateway":
		return "Pushgateway"
	case "demo-app":
		return "Demo App"
	case "nginx-ingress":
		return "Ingress"
	case "kubernetes-dashboard":
		return "K8s Dashboard"
	case "backstage":
		return "Backstage"
	case "minio":
		return "Minio"
	default:
		return toTitle(name)
	}
}

// toTitle converts a string to title case (replacement for deprecated strings.Title)
func toTitle(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// getCliName returns the CLI executable name
func (c *CheatSheet) getCliName() string {
	return "innominatus-ctl"
}
