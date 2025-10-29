package demo

// #nosec G204 - Demo installer executes helm and kubectl with controlled parameters for local demo setup only

import (
	"fmt"
	"os/exec"
	"strings"
)

// InstallPostgresOperator installs Zalando PostgreSQL Operator via Helm
func InstallPostgresOperator(kubeContext string) error {
	fmt.Println("📦 Installing PostgreSQL Operator (Zalando)...")

	// Add Helm repo
	addRepoCmd := exec.Command("helm", "repo", "add", "postgres-operator-charts",
		"https://opensource.zalando.com/postgres-operator/charts/postgres-operator")
	if err := addRepoCmd.Run(); err != nil {
		return fmt.Errorf("failed to add postgres operator helm repo: %w", err)
	}

	// Update repos
	updateCmd := exec.Command("helm", "repo", "update")
	if err := updateCmd.Run(); err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	// Install operator
	installCmd := exec.Command("helm", "upgrade", "--install", "postgres-operator", // #nosec G204 - helm with controlled params
		"postgres-operator-charts/postgres-operator",
		"--namespace", "postgres-operator",
		"--create-namespace",
		"--kube-context", kubeContext,
		"--set", "configKubernetes.enable_pod_antiaffinity=false",
		"--set", "configKubernetes.enable_pod_disruption_budget=false",
		"--wait",
		"--timeout", "5m")

	output, err := installCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install postgres operator: %w\nOutput: %s", err, string(output))
	}

	// Install UI (optional)
	fmt.Println("📦 Installing PostgreSQL Operator UI...")
	uiCmd := exec.Command("helm", "upgrade", "--install", "postgres-operator-ui", // #nosec G204 - helm with controlled params
		"postgres-operator-charts/postgres-operator-ui",
		"--namespace", "postgres-operator",
		"--kube-context", kubeContext,
		"--wait",
		"--timeout", "5m")

	uiOutput, err := uiCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to install postgres operator UI (continuing anyway): %v\nOutput: %s\n", err, string(uiOutput))
	}

	fmt.Println("✅ PostgreSQL Operator installed successfully")
	return nil
}

// UninstallPostgresOperator removes PostgreSQL Operator
func UninstallPostgresOperator(kubeContext string) error {
	fmt.Println("🗑️  Uninstalling PostgreSQL Operator...")

	// Uninstall UI
	uiCmd := exec.Command("helm", "uninstall", "postgres-operator-ui", // #nosec G204 - helm uninstall
		"-n", "postgres-operator",
		"--kube-context", kubeContext)
	_ = uiCmd.Run() // Ignore errors

	// Uninstall operator
	operatorCmd := exec.Command("helm", "uninstall", "postgres-operator", // #nosec G204 - helm uninstall
		"-n", "postgres-operator",
		"--kube-context", kubeContext)
	_ = operatorCmd.Run() // Ignore errors

	// Delete namespace
	nsCmd := exec.Command("kubectl", "--context", kubeContext, "delete", "namespace", "postgres-operator", // #nosec G204 - kubectl delete
		"--ignore-not-found=true")
	_ = nsCmd.Run() // Ignore errors

	fmt.Println("✅ PostgreSQL Operator uninstalled")
	return nil
}

// CheckPostgresOperatorStatus checks if operator is running
func CheckPostgresOperatorStatus(kubeContext string) error {
	cmd := exec.Command("kubectl", "--context", kubeContext, "get", "pods", // #nosec G204 - kubectl get
		"-n", "postgres-operator",
		"-l", "app.kubernetes.io/name=postgres-operator",
		"--no-headers")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postgres operator not found: %w", err)
	}

	// Check if any pods are running
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return fmt.Errorf("no postgres operator pods found")
	}

	// Check if at least one pod is Running
	hasRunning := false
	for _, line := range lines {
		if strings.Contains(line, "Running") {
			hasRunning = true
			break
		}
	}

	if !hasRunning {
		return fmt.Errorf("postgres operator pods are not running")
	}

	return nil
}
