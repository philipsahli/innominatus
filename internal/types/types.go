package types

type ScoreSpec struct {
	APIVersion  string               `yaml:"apiVersion"`
	Metadata    Metadata             `yaml:"metadata"`
	Containers  map[string]Container `yaml:"containers"`
	Resources   map[string]Resource  `yaml:"resources"`
	Environment *Environment         `yaml:"environment,omitempty"`
	Workflows   map[string]Workflow  `yaml:"workflows,omitempty"`
}

type Metadata struct {
	Name string `yaml:"name"`
}

type Container struct {
	Image     string            `yaml:"image"`
	Variables map[string]string `yaml:"variables"`
}

type Resource struct {
	Type       string                 `yaml:"type"`
	Params     map[string]interface{} `yaml:"params,omitempty"`
	Properties map[string]interface{} `yaml:"properties,omitempty"`
}

type Environment struct {
	Type string `yaml:"type"`
	TTL  string `yaml:"ttl"`
}

type Workflow struct {
	Steps     []Step            `yaml:"steps"`
	Variables map[string]string `yaml:"variables,omitempty"` // Workflow-level variables
	Outputs   map[string]string `yaml:"outputs,omitempty"`   // Workflow outputs (bucket_name, endpoint, etc.)
}

// WorkflowSpec represents a complete workflow document with metadata
type WorkflowSpec struct {
	APIVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	Metadata   WorkflowMetadata `yaml:"metadata"`
	Spec       Workflow         `yaml:"spec"`
}

type WorkflowMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type Step struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	Path      string `yaml:"path"`
	Playbook  string `yaml:"playbook,omitempty"`
	Namespace string `yaml:"namespace,omitempty"`
	// New fields for TFE and Git workflows
	Resource      string `yaml:"resource,omitempty"`      // For terraform-generate
	OutputDir     string `yaml:"outputDir,omitempty"`     // For terraform-generate
	Repo          string `yaml:"repo,omitempty"`          // For git operations
	Branch        string `yaml:"branch,omitempty"`        // For git operations
	CommitMessage string `yaml:"commitMessage,omitempty"` // For git-pr
	Workspace     string `yaml:"workspace,omitempty"`     // For tfe-status
	// New fields for gitea-repo workflow
	RepoName    string `yaml:"repoName,omitempty"`    // For gitea-repo
	Description string `yaml:"description,omitempty"` // For gitea-repo
	Private     bool   `yaml:"private,omitempty"`     // For gitea-repo
	Owner       string `yaml:"owner,omitempty"`       // For gitea-repo
	// New fields for argocd-app workflow
	AppName    string `yaml:"appName,omitempty"`    // For argocd-app
	RepoURL    string `yaml:"repoURL,omitempty"`    // For argocd-app
	TargetPath string `yaml:"targetPath,omitempty"` // For argocd-app
	Project    string `yaml:"project,omitempty"`    // For argocd-app
	SyncPolicy string `yaml:"syncPolicy,omitempty"` // For argocd-app (manual/auto)
	// New fields for git-commit-manifests workflow
	ManifestPath string `yaml:"manifestPath,omitempty"` // For git-commit-manifests
	GitBranch    string `yaml:"gitBranch,omitempty"`    // For git-commit-manifests
	// New fields for sync waiting and timeout
	Timeout     int   `yaml:"timeout,omitempty"`     // Timeout in seconds for sync waiting
	WaitForSync *bool `yaml:"waitForSync,omitempty"` // Whether to wait for sync completion
	// New fields for parallel execution
	Parallel      bool     `yaml:"parallel,omitempty"`      // Whether this step can run in parallel
	DependsOn     []string `yaml:"dependsOn,omitempty"`     // Steps this step depends on
	ParallelGroup int      `yaml:"parallelGroup,omitempty"` // Group ID for parallel execution
	// New fields for conditional execution
	When   string            `yaml:"when,omitempty"`   // Condition expression (e.g., "always", "on_success", "on_failure")
	If     string            `yaml:"if,omitempty"`     // Condition that must be true to run
	Unless string            `yaml:"unless,omitempty"` // Condition that must be false to run
	Env    map[string]string `yaml:"env,omitempty"`    // Environment variables for condition evaluation
	// New fields for output capture and passing
	Outputs      []string          `yaml:"outputs,omitempty"`      // List of output variable names to capture
	OutputFile   string            `yaml:"outputFile,omitempty"`   // File to read outputs from (JSON or key=value format)
	SetVariables map[string]string `yaml:"setVariables,omitempty"` // Variables to set in workflow context
	// New fields for terraform step type
	Operation  string                 `yaml:"operation,omitempty"`  // Terraform operation (init, plan, apply, destroy, output)
	WorkingDir string                 `yaml:"workingDir,omitempty"` // Working directory for terraform
	Variables  map[string]interface{} `yaml:"variables,omitempty"`  // Terraform variables
	Config     map[string]interface{} `yaml:"config,omitempty"`     // Generic config map for flexible step configuration
}
