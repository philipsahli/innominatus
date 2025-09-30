package demo

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

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
	fmt.Println("🚀 OpenAlps Demo Environment")
	fmt.Println("===============================")
	fmt.Println()
}

// PrintInstallationComplete prints the installation complete message
func (c *CheatSheet) PrintInstallationComplete() {
	fmt.Println()
	fmt.Println("🎉 Demo Environment Installation Complete!")
	fmt.Println()
	fmt.Println("Your OpenAlps demo environment is now ready with:")
	fmt.Println("• GitOps workflow with ArgoCD")
	fmt.Println("• Git repository with Gitea")
	fmt.Println("• Secret management with Vault")
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
}

// PrintQuickStart prints quick start instructions
func (c *CheatSheet) PrintQuickStart() {
	fmt.Println("🚀 Quick Start Guide")
	fmt.Println("====================")
	fmt.Println()
	fmt.Println("1. Access Gitea (Git Repository):")
	fmt.Println("   • URL: http://gitea.localtest.me")
	fmt.Println("   • Login: giteaadmin / admin")
	fmt.Println("   • Check the 'platform-config' repository")
	fmt.Println()
	fmt.Println("2. Access ArgoCD (GitOps):")
	fmt.Println("   • URL: http://argocd.localtest.me")
	fmt.Println("   • Login: admin / admin123")
	fmt.Println("   • View deployed applications")
	fmt.Println()
	fmt.Println("3. Access Grafana (Monitoring):")
	fmt.Println("   • URL: http://grafana.localtest.me")
	fmt.Println("   • Login: admin / admin")
	fmt.Println("   • Explore pre-configured dashboards")
	fmt.Println()
	fmt.Println("4. Access Kubernetes Dashboard:")
	fmt.Println("   • URL: http://k8s.localtest.me")
	fmt.Println("   • Login token: kubectl -n kubernetes-dashboard create token admin-user")
	fmt.Println("   • View cluster resources and workloads")
	fmt.Println()
	fmt.Println("5. Access Demo Application:")
	fmt.Println("   • URL: http://demo.localtest.me")
	fmt.Println("   • Sample app deployed via Score spec")
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
	case "demo-app":
		return "Demo App"
	case "nginx-ingress":
		return "Ingress"
	case "kubernetes-dashboard":
		return "K8s Dashboard"
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
