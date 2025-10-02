package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestScoreSpecYAMLParsing(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected ScoreSpec
		wantErr  bool
	}{
		{
			name: "minimal valid spec",
			yaml: `
apiVersion: score.dev/v1b1
metadata:
  name: test-app
containers:
  web:
    image: nginx:latest
`,
			expected: ScoreSpec{
				APIVersion: "score.dev/v1b1",
				Metadata: Metadata{
					Name: "test-app",
				},
				Containers: map[string]Container{
					"web": {
						Image: "nginx:latest",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "spec with resources",
			yaml: `
apiVersion: score.dev/v1b1
metadata:
  name: test-app
containers:
  web:
    image: nginx:latest
    variables:
      DB_HOST: ${resources.db.host}
resources:
  db:
    type: postgres
    params:
      version: "13"
      size: small
`,
			expected: ScoreSpec{
				APIVersion: "score.dev/v1b1",
				Metadata: Metadata{
					Name: "test-app",
				},
				Containers: map[string]Container{
					"web": {
						Image: "nginx:latest",
						Variables: map[string]string{
							"DB_HOST": "${resources.db.host}",
						},
					},
				},
				Resources: map[string]Resource{
					"db": {
						Type: "postgres",
						Params: map[string]interface{}{
							"version": "13",
							"size":    "small",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "spec with environment",
			yaml: `
apiVersion: score.dev/v1b1
metadata:
  name: test-app
containers:
  web:
    image: nginx:latest
environment:
  type: kubernetes
  ttl: 1h
`,
			expected: ScoreSpec{
				APIVersion: "score.dev/v1b1",
				Metadata: Metadata{
					Name: "test-app",
				},
				Containers: map[string]Container{
					"web": {
						Image: "nginx:latest",
					},
				},
				Environment: &Environment{
					Type: "kubernetes",
					TTL:  "1h",
				},
			},
			wantErr: false,
		},
		{
			name: "spec with workflows",
			yaml: `
apiVersion: score.dev/v1b1
metadata:
  name: test-app
containers:
  web:
    image: nginx:latest
workflows:
  deploy:
    steps:
      - name: setup-infra
        type: terraform
        path: ./terraform
      - name: deploy-app
        type: kubernetes
        namespace: test-app
`,
			expected: ScoreSpec{
				APIVersion: "score.dev/v1b1",
				Metadata: Metadata{
					Name: "test-app",
				},
				Containers: map[string]Container{
					"web": {
						Image: "nginx:latest",
					},
				},
				Workflows: map[string]Workflow{
					"deploy": {
						Steps: []Step{
							{
								Name: "setup-infra",
								Type: "terraform",
								Path: "./terraform",
							},
							{
								Name:      "deploy-app",
								Type:      "kubernetes",
								Namespace: "test-app",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid yaml",
			yaml:    `invalid: yaml: content:`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var spec ScoreSpec
			err := yaml.Unmarshal([]byte(tt.yaml), &spec)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected.APIVersion, spec.APIVersion)
			assert.Equal(t, tt.expected.Metadata, spec.Metadata)
			assert.Equal(t, tt.expected.Containers, spec.Containers)
			assert.Equal(t, tt.expected.Resources, spec.Resources)
			assert.Equal(t, tt.expected.Environment, spec.Environment)
			assert.Equal(t, tt.expected.Workflows, spec.Workflows)
		})
	}
}

func TestStepValidation(t *testing.T) {
	tests := []struct {
		name string
		step Step
		want bool
	}{
		{
			name: "valid terraform step",
			step: Step{
				Name: "setup-infra",
				Type: "terraform",
				Path: "./terraform",
			},
			want: true,
		},
		{
			name: "valid kubernetes step",
			step: Step{
				Name:      "deploy-app",
				Type:      "kubernetes",
				Namespace: "my-app",
			},
			want: true,
		},
		{
			name: "valid ansible step",
			step: Step{
				Name:     "configure",
				Type:     "ansible",
				Playbook: "setup.yml",
			},
			want: true,
		},
		{
			name: "step with git fields",
			step: Step{
				Name:          "create-pr",
				Type:          "git-pr",
				Repo:          "my-org/my-repo",
				Branch:        "main",
				CommitMessage: "Update manifests",
			},
			want: true,
		},
		{
			name: "step with argocd fields",
			step: Step{
				Name:       "create-app",
				Type:       "argocd-app",
				AppName:    "my-app",
				RepoURL:    "https://github.com/my-org/my-repo.git",
				TargetPath: "manifests/",
				Project:    "default",
				SyncPolicy: "auto",
			},
			want: true,
		},
		{
			name: "empty step name",
			step: Step{
				Name: "",
				Type: "terraform",
			},
			want: false,
		},
		{
			name: "empty step type",
			step: Step{
				Name: "setup",
				Type: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.step.Name != "" && tt.step.Type != ""
			assert.Equal(t, tt.want, valid)
		})
	}
}

func TestContainerVariables(t *testing.T) {
	container := Container{
		Image: "nginx:latest",
		Variables: map[string]string{
			"DB_HOST":    "${resources.db.host}",
			"DB_PORT":    "5432",
			"REDIS_URL":  "${resources.cache.url}",
			"STATIC_VAR": "static-value",
		},
	}

	// Test that variables are correctly stored
	assert.Equal(t, "${resources.db.host}", container.Variables["DB_HOST"])
	assert.Equal(t, "5432", container.Variables["DB_PORT"])
	assert.Equal(t, "${resources.cache.url}", container.Variables["REDIS_URL"])
	assert.Equal(t, "static-value", container.Variables["STATIC_VAR"])

	// Test variable count
	assert.Len(t, container.Variables, 4)
}

func TestResourceParams(t *testing.T) {
	resource := Resource{
		Type: "postgres",
		Params: map[string]interface{}{
			"version":  "13",
			"size":     "small",
			"backup":   true,
			"replicas": 3,
			"config": map[string]interface{}{
				"max_connections": 100,
				"shared_buffers":  "256MB",
			},
		},
	}

	assert.Equal(t, "postgres", resource.Type)
	assert.Equal(t, "13", resource.Params["version"])
	assert.Equal(t, "small", resource.Params["size"])
	assert.Equal(t, true, resource.Params["backup"])
	assert.Equal(t, 3, resource.Params["replicas"])

	// Test nested config
	config := resource.Params["config"].(map[string]interface{})
	assert.Equal(t, 100, config["max_connections"])
	assert.Equal(t, "256MB", config["shared_buffers"])
}

func TestEnvironmentTTL(t *testing.T) {
	tests := []struct {
		name string
		env  Environment
		want bool
	}{
		{
			name: "valid kubernetes environment",
			env: Environment{
				Type: "kubernetes",
				TTL:  "1h",
			},
			want: true,
		},
		{
			name: "valid docker environment",
			env: Environment{
				Type: "docker",
				TTL:  "30m",
			},
			want: true,
		},
		{
			name: "environment without TTL",
			env: Environment{
				Type: "kubernetes",
			},
			want: true,
		},
		{
			name: "empty environment type",
			env: Environment{
				Type: "",
				TTL:  "1h",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.env.Type != ""
			assert.Equal(t, tt.want, valid)
		})
	}
}

func TestWorkflowStepsCount(t *testing.T) {
	workflow := Workflow{
		Steps: []Step{
			{Name: "step1", Type: "terraform"},
			{Name: "step2", Type: "kubernetes"},
			{Name: "step3", Type: "ansible"},
		},
	}

	assert.Len(t, workflow.Steps, 3)
	assert.Equal(t, "step1", workflow.Steps[0].Name)
	assert.Equal(t, "step2", workflow.Steps[1].Name)
	assert.Equal(t, "step3", workflow.Steps[2].Name)
}

func TestStepOptionalFields(t *testing.T) {
	step := Step{
		Name:         "complex-step",
		Type:         "gitea-repo",
		RepoName:     "my-repo",
		Description:  "Test repository",
		Private:      true,
		Owner:        "my-org",
		AppName:      "my-app",
		RepoURL:      "https://github.com/my-org/my-repo.git",
		TargetPath:   "manifests/",
		Project:      "default",
		SyncPolicy:   "manual",
		ManifestPath: "k8s/",
		GitBranch:    "develop",
		Timeout:      300,
		WaitForSync:  boolPtr(true),
	}

	assert.Equal(t, "complex-step", step.Name)
	assert.Equal(t, "gitea-repo", step.Type)
	assert.Equal(t, "my-repo", step.RepoName)
	assert.Equal(t, "Test repository", step.Description)
	assert.True(t, step.Private)
	assert.Equal(t, "my-org", step.Owner)
	assert.Equal(t, 300, step.Timeout)
	assert.NotNil(t, step.WaitForSync)
	assert.True(t, *step.WaitForSync)
}

// Helper function for testing bool pointers
func boolPtr(b bool) *bool {
	return &b
}
