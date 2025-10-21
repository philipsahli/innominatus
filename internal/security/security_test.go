package security

import (
	"strings"
	"testing"
)

// ===== Path Validation Tests =====

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		allowedBases []string
		expectError  bool
	}{
		{
			name:        "clean relative path",
			path:        "data/file.txt",
			expectError: false,
		},
		{
			name:        "path with dot-dot traversal",
			path:        "../../../etc/passwd",
			expectError: true,
		},
		{
			name:        "path with encoded dot-dot",
			path:        "data/../../../etc/passwd",
			expectError: true,
		},
		{
			name:         "absolute path without allowed base",
			path:         "/etc/passwd",
			allowedBases: []string{"/tmp"},
			expectError:  true,
		},
		{
			name:         "absolute path with matching base",
			path:         "/tmp/data/file.txt",
			allowedBases: []string{"/tmp"},
			expectError:  false,
		},
		{
			name:        "clean path with no traversal",
			path:        "workflows/deploy.yaml",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.path, tt.allowedBases...)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateFilePath() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestSafeFilePath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		allowedBases []string
		expectError  bool
		expectedPath string
	}{
		{
			name:         "clean relative path",
			path:         "data/file.txt",
			expectError:  false,
			expectedPath: "data/file.txt",
		},
		{
			name:        "path with traversal",
			path:        "../../../etc/passwd",
			expectError: true,
		},
		{
			name:         "path with redundant separators",
			path:         "data//file.txt",
			expectError:  false,
			expectedPath: "data/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SafeFilePath(tt.path, tt.allowedBases...)
			if (err != nil) != tt.expectError {
				t.Errorf("SafeFilePath() error = %v, expectError %v", err, tt.expectError)
			}
			if !tt.expectError && result != tt.expectedPath {
				t.Errorf("SafeFilePath() = %v, expected %v", result, tt.expectedPath)
			}
		})
	}
}

func TestValidateWorkflowPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "valid workflows path",
			path:        "workflows/deploy.yaml",
			expectError: false,
		},
		{
			name:        "valid workspaces path",
			path:        "workspaces/app1",
			expectError: false,
		},
		{
			name:        "valid data path",
			path:        "data/config.json",
			expectError: false,
		},
		{
			name:        "valid terraform path",
			path:        "terraform/main.tf",
			expectError: false,
		},
		{
			name:        "path with traversal",
			path:        "workflows/../../../etc/passwd",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkflowPath(tt.path)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateWorkflowPath() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateConfigPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "admin config file",
			path:        "admin-config.yaml",
			expectError: false,
		},
		{
			name:        "golden paths config",
			path:        "goldenpaths.yaml",
			expectError: false,
		},
		{
			name:        "test config file",
			path:        "test-config.yaml",
			expectError: false,
		},
		{
			name:        "path with traversal",
			path:        "../../etc/passwd",
			expectError: true,
		},
		{
			name:        "file in config directory",
			path:        "config/settings.yaml",
			expectError: false,
		},
		{
			name:        "any yaml file",
			path:        "myapp.yaml",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfigPath(tt.path)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateConfigPath() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// ===== URL Validation Tests =====

func TestValidateArgoCDURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "valid HTTPS URL",
			url:         "https://argocd.example.com",
			expectError: false,
		},
		{
			name:        "valid HTTP URL",
			url:         "http://argocd.example.com",
			expectError: false,
		},
		{
			name:        "allowed local test domain",
			url:         "http://argocd.localtest.me",
			expectError: false,
		},
		{
			name:        "allowed cluster local",
			url:         "http://argocd-server.argocd.svc.cluster.local",
			expectError: false,
		},
		{
			name:        "blocked localhost",
			url:         "http://localhost:8080",
			expectError: true,
		},
		{
			name:        "blocked 127.0.0.1",
			url:         "http://127.0.0.1:8080",
			expectError: true,
		},
		{
			name:        "blocked private IP 192.168",
			url:         "http://192.168.1.1",
			expectError: true,
		},
		{
			name:        "blocked private IP 10.x",
			url:         "http://10.0.0.1",
			expectError: true,
		},
		{
			name:        "blocked private IP 172.16",
			url:         "http://172.16.0.1",
			expectError: true,
		},
		{
			name:        "invalid scheme file",
			url:         "file:///etc/passwd",
			expectError: true,
		},
		{
			name:        "invalid scheme ftp",
			url:         "ftp://example.com",
			expectError: true,
		},
		{
			name:        "malformed URL",
			url:         "://invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateArgoCDURL(tt.url)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateArgoCDURL() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateExternalURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "valid HTTPS URL",
			url:         "https://api.example.com",
			expectError: false,
		},
		{
			name:        "valid HTTP URL",
			url:         "http://api.example.com",
			expectError: false,
		},
		{
			name:        "invalid scheme",
			url:         "ftp://example.com",
			expectError: true,
		},
		{
			name:        "invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExternalURL(tt.url)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateExternalURL() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// ===== Command Validation Tests =====

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		expectError bool
	}{
		{"allowed terraform", "terraform", false},
		{"allowed kubectl", "kubectl", false},
		{"allowed ansible-playbook", "ansible-playbook", false},
		{"allowed git", "git", false},
		{"allowed docker", "docker", false},
		{"allowed helm", "helm", false},
		{"not allowed rm", "rm", true},
		{"not allowed bash", "bash", true},
		{"not allowed sh", "sh", true},
		{"not allowed python", "python", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommand(tt.command)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateCommand() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateCommandArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "safe args",
			args:        []string{"apply", "-auto-approve", "-var=foo=bar"},
			expectError: false,
		},
		{
			name:        "kubectl pipe notation allowed",
			args:        []string{"get", "pods", "-|", "grep", "Running"},
			expectError: false,
		},
		{
			name:        "command chaining semicolon",
			args:        []string{"apply", ";", "rm", "-rf"},
			expectError: true,
		},
		{
			name:        "command chaining &&",
			args:        []string{"apply", "&&", "rm"},
			expectError: true,
		},
		{
			name:        "command chaining ||",
			args:        []string{"apply", "||", "rm"},
			expectError: true,
		},
		{
			name:        "piping",
			args:        []string{"get", "|", "grep"},
			expectError: true,
		},
		{
			name:        "command substitution backtick",
			args:        []string{"apply", "`rm -rf`"},
			expectError: true,
		},
		{
			name:        "command substitution $()",
			args:        []string{"apply", "$(rm -rf /)"},
			expectError: true,
		},
		{
			name:        "path traversal",
			args:        []string{"../../../etc/passwd"},
			expectError: true,
		},
		{
			name:        "home directory expansion",
			args:        []string{"~/malicious.sh"},
			expectError: true,
		},
		{
			name:        "null byte injection",
			args:        []string{"file\x00.txt"},
			expectError: true,
		},
		{
			name:        "empty args",
			args:        []string{"apply", "", "-auto-approve"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommandArgs(tt.args)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateCommandArgs() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestSafeCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
	}{
		{
			name:        "safe terraform command",
			command:     "terraform",
			args:        []string{"apply", "-auto-approve"},
			expectError: false,
		},
		{
			name:        "safe kubectl command",
			command:     "kubectl",
			args:        []string{"get", "pods"},
			expectError: false,
		},
		{
			name:        "disallowed command",
			command:     "rm",
			args:        []string{"-rf", "/"},
			expectError: true,
		},
		{
			name:        "allowed command with dangerous args",
			command:     "terraform",
			args:        []string{"apply", ";", "rm", "-rf"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := SafeCommand(tt.command, tt.args...)
			if (err != nil) != tt.expectError {
				t.Errorf("SafeCommand() error = %v, expectError %v", err, tt.expectError)
			}
			if !tt.expectError && cmd == nil {
				t.Error("SafeCommand() returned nil command when no error expected")
			}
			if !tt.expectError && cmd != nil {
				// Verify command was created correctly
				if cmd.Path == "" {
					t.Error("SafeCommand() created command with empty path")
				}
			}
		})
	}
}

func TestValidateResourceName(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		expectError  bool
	}{
		{"valid lowercase", "my-app", false},
		{"valid with numbers", "app123", false},
		{"valid with hyphens", "my-app-v2", false},
		{"invalid uppercase", "MyApp", true},
		{"invalid underscore", "my_app", true},
		{"invalid spaces", "my app", true},
		{"invalid starts with hyphen", "-myapp", true},
		{"invalid ends with hyphen", "myapp-", true},
		{"invalid dot", "my.app", true},
		{"valid single char", "a", false},
		{"too long name", strings.Repeat("a", 254), true},
		{"max length valid", strings.Repeat("a", 253), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResourceName(tt.resourceName)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateResourceName() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateNamespace(t *testing.T) {
	// Namespace validation uses same rules as resource name
	tests := []struct {
		name        string
		namespace   string
		expectError bool
	}{
		{"valid namespace", "my-namespace", false},
		{"invalid uppercase", "MyNamespace", true},
		{"valid with numbers", "namespace123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNamespace(tt.namespace)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateNamespace() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// ===== Test AllowedCommands Map =====

func TestAllowedCommands(t *testing.T) {
	expectedCommands := []string{
		"terraform",
		"kubectl",
		"ansible-playbook",
		"git",
		"docker",
		"helm",
		"curl",
		"lsof",
		"pkill",
	}

	for _, cmd := range expectedCommands {
		if !AllowedCommands[cmd] {
			t.Errorf("Expected command %s to be in AllowedCommands", cmd)
		}
	}

	// Verify some commands are NOT allowed
	disallowedCommands := []string{
		"rm",
		"bash",
		"sh",
		"python",
		"perl",
	}

	for _, cmd := range disallowedCommands {
		if AllowedCommands[cmd] {
			t.Errorf("Command %s should NOT be in AllowedCommands", cmd)
		}
	}
}
