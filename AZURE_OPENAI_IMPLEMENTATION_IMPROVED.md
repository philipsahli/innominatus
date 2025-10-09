# Azure OpenAI Integration - KISS & SOLID Implementation Guide

## Overview

This document provides a simplified, SOLID-compliant implementation for Azure OpenAI integration that follows best practices.

## Architecture Principles

### KISS (Keep It Simple, Stupid)
- **Single configuration source**: One function to determine embedding provider
- **No duplication**: Centralized provider logic
- **Clear separation**: Configuration vs. initialization vs. business logic

### SOLID Principles
- **SRP**: Each component has one responsibility
- **OCP**: Extensible without modifying existing code
- **LSP**: Provider implementations are interchangeable
- **ISP**: Minimal interfaces
- **DIP**: Depend on abstractions, not concrete implementations

---

## Implementation

### 1. Embedding Configuration (New File)

**File**: `internal/ai/embedding_config.go`

```go
package ai

import (
	"fmt"
	"os"
)

// EmbeddingProvider represents the embedding service configuration
type EmbeddingProvider struct {
	Type    string // "openai" or "azure-openai"
	BaseURL string
	APIKey  string
	Model   string
	Headers map[string]string // For Azure-specific headers
}

// Constants for default values
const (
	DefaultOpenAIBaseURL = "https://api.openai.com/v1"
	DefaultOpenAIModel   = "text-embedding-3-small"

	DefaultAzureAPIVersion = "2024-02-15-preview"
	DefaultAzureModel      = "text-embedding-ada-002"
)

// NewEmbeddingProvider creates embedding provider configuration from environment
// This is the SINGLE source of truth for embedding configuration
func NewEmbeddingProvider() (*EmbeddingProvider, error) {
	// Check Azure OpenAI first (higher priority for enterprise environments)
	if os.Getenv("AZURE_OPENAI_ENABLED") == "true" {
		return newAzureOpenAIProvider()
	}

	// Fallback to standard OpenAI
	return newOpenAIProvider()
}

// newAzureOpenAIProvider creates Azure OpenAI configuration
func newAzureOpenAIProvider() (*EmbeddingProvider, error) {
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	deployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	apiKey := os.Getenv("AZURE_OPENAI_KEY")
	apiVersion := getEnvOrDefault("AZURE_OPENAI_API_VERSION", DefaultAzureAPIVersion)

	if endpoint == "" || deployment == "" || apiKey == "" {
		return nil, fmt.Errorf("Azure OpenAI enabled but missing configuration: AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_DEPLOYMENT, or AZURE_OPENAI_KEY")
	}

	// Azure OpenAI URL format: {endpoint}/openai/deployments/{deployment}
	baseURL := fmt.Sprintf("%s/openai/deployments/%s", endpoint, deployment)

	return &EmbeddingProvider{
		Type:    "azure-openai",
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   deployment, // Azure uses deployment name as model
		Headers: map[string]string{
			"api-key": apiKey,
			"api-version": apiVersion,
		},
	}, nil
}

// newOpenAIProvider creates standard OpenAI configuration
func newOpenAIProvider() (*EmbeddingProvider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	baseURL := getEnvOrDefault("OPENAI_API_BASE", DefaultOpenAIBaseURL)
	model := getEnvOrDefault("OPENAI_EMBEDDING_MODEL", DefaultOpenAIModel)

	return &EmbeddingProvider{
		Type:    "openai",
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		},
	}, nil
}

// String returns a human-readable description
func (ep *EmbeddingProvider) String() string {
	return fmt.Sprintf("%s (%s, model: %s)", ep.Type, ep.BaseURL, ep.Model)
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
```

### 2. Simplified Service Initialization

**File**: `internal/ai/service.go` (Updated)

```go
package ai

import (
	"context"
	"fmt"

	"github.com/philipsahli/innominatus-ai-sdk/pkg/platformai"
	"github.com/philipsahli/innominatus-ai-sdk/pkg/platformai/rag"
	"github.com/rs/zerolog/log"
)

// Service provides AI assistance functionality
type Service struct {
	sdk     *platformai.SDK
	enabled bool
	embeddingProvider *EmbeddingProvider
}

// Config holds AI service configuration (simplified)
type Config struct {
	AnthropicKey  string
	DocsPath      string
	WorkflowsPath string
}

// NewService creates a new AI service with dependency injection
func NewService(ctx context.Context, cfg Config, embeddingProvider *EmbeddingProvider) (*Service, error) {
	log.Debug().Msg("Initializing AI service")

	// Validate required configuration
	if embeddingProvider == nil {
		return nil, fmt.Errorf("embedding provider is required")
	}
	if cfg.AnthropicKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	// Log configuration
	log.Info().
		Str("embedding_provider", embeddingProvider.Type).
		Str("embedding_model", embeddingProvider.Model).
		Str("embedding_base_url", embeddingProvider.BaseURL).
		Str("llm_provider", "anthropic").
		Msg("Initializing AI SDK")

	// Initialize Platform AI SDK
	sdk, err := platformai.New(ctx, &platformai.Config{
		LLM: platformai.LLMConfig{
			Provider:    "anthropic",
			APIKey:      cfg.AnthropicKey,
			Model:       "claude-sonnet-4-5-20250929",
			Temperature: 0.7,
			MaxTokens:   4096,
		},
		RAG: &rag.Config{
			EmbeddingProvider: embeddingProvider.Type,
			APIKey:            embeddingProvider.APIKey,
			BaseURL:           embeddingProvider.BaseURL,
			Model:             embeddingProvider.Model,
			CustomHeaders:     embeddingProvider.Headers, // Pass Azure-specific headers
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AI SDK: %w", err)
	}

	service := &Service{
		sdk:               sdk,
		enabled:           true,
		embeddingProvider: embeddingProvider,
	}

	// Load knowledge base
	if err := service.loadKnowledgeBase(ctx, cfg); err != nil {
		log.Warn().Err(err).Msg("Failed to load knowledge base, AI will work with limited context")
	}

	log.Info().
		Str("embedding_provider", embeddingProvider.Type).
		Msg("AI service initialized successfully")

	return service, nil
}

// NewServiceFromEnv creates AI service from environment variables
func NewServiceFromEnv(ctx context.Context) (*Service, error) {
	// Step 1: Determine embedding provider (SINGLE source of truth)
	embeddingProvider, err := NewEmbeddingProvider()
	if err != nil {
		log.Warn().Err(err).Msg("AI service disabled: embedding provider configuration error")
		return &Service{enabled: false}, nil
	}

	// Step 2: Get Anthropic key
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey == "" {
		log.Warn().Msg("AI service disabled: ANTHROPIC_API_KEY not set")
		return &Service{enabled: false}, nil
	}

	// Step 3: Create service with configuration
	config := Config{
		AnthropicKey:  anthropicKey,
		DocsPath:      "docs",
		WorkflowsPath: "workflows",
	}

	return NewService(ctx, config, embeddingProvider)
}

// GetStatus returns the current AI service status
func (s *Service) GetStatus(ctx context.Context) StatusResponse {
	if !s.enabled {
		return StatusResponse{
			Enabled: false,
			Status:  "not_configured",
			Message: "AI service is disabled. Check API key configuration.",
		}
	}

	docCount, err := s.sdk.RAG().Count(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get document count from RAG")
		docCount = 0
	}

	return StatusResponse{
		Enabled:           true,
		LLMProvider:       "anthropic",
		EmbeddingProvider: s.embeddingProvider.Type,
		EmbeddingModel:    s.embeddingProvider.Model,
		DocumentsLoaded:   docCount,
		Status:            "ready",
		Message:           fmt.Sprintf("AI assistant is ready with %d documents in knowledge base", docCount),
	}
}

// ... rest of service methods remain unchanged
```

### 3. Demo-Time Integration (Simplified)

**File**: `internal/cli/demo_components.go` (New or extracted)

```go
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

// AzureOpenAIComponent handles Azure OpenAI configuration for demo environment
type AzureOpenAIComponent struct{}

// Configure sets up Azure OpenAI integration (if credentials provided)
func (c *AzureOpenAIComponent) Configure() error {
	// Check if Azure credentials are provided
	if !isAzureOpenAIConfigured() {
		log.Info().Msg("⚠ Azure OpenAI not configured (optional)")
		log.Info().Msg("  Set AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_KEY, and AZURE_OPENAI_DEPLOYMENT to enable")
		return nil
	}

	log.Info().Msg("Configuring Azure OpenAI integration...")

	// Create ConfigMap and Secret
	if err := createAzureOpenAIConfigMap(); err != nil {
		return fmt.Errorf("failed to create Azure OpenAI ConfigMap: %w", err)
	}

	if err := createAzureOpenAISecret(); err != nil {
		return fmt.Errorf("failed to create Azure OpenAI Secret: %w", err)
	}

	log.Info().Msg("✓ Azure OpenAI integration configured")
	return nil
}

// Check returns whether Azure OpenAI is configured
func (c *AzureOpenAIComponent) Check() (bool, error) {
	// Check if ConfigMap exists
	cmd := exec.Command("kubectl", "get", "configmap", "azure-openai-config",
		"-n", "innominatus-system", "-o", "name")

	return cmd.Run() == nil, nil
}

// Cleanup removes Azure OpenAI Kubernetes resources
func (c *AzureOpenAIComponent) Cleanup() error {
	log.Info().Msg("Cleaning up Azure OpenAI configuration...")

	// Delete ConfigMap and Secret
	exec.Command("kubectl", "delete", "configmap", "azure-openai-config",
		"-n", "innominatus-system", "--ignore-not-found").Run()

	exec.Command("kubectl", "delete", "secret", "azure-openai-secret",
		"-n", "innominatus-system", "--ignore-not-found").Run()

	log.Info().Msg("✓ Azure OpenAI configuration cleaned up")
	return nil
}

// Helper functions

func isAzureOpenAIConfigured() bool {
	return os.Getenv("AZURE_OPENAI_ENDPOINT") != "" &&
		os.Getenv("AZURE_OPENAI_KEY") != "" &&
		os.Getenv("AZURE_OPENAI_DEPLOYMENT") != ""
}

func createAzureOpenAIConfigMap() error {
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	deployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	apiVersion := getEnvOrDefault("AZURE_OPENAI_API_VERSION", "2024-02-15-preview")

	configMap := fmt.Sprintf(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: azure-openai-config
  namespace: innominatus-system
data:
  AZURE_OPENAI_ENABLED: "true"
  AZURE_OPENAI_ENDPOINT: "%s"
  AZURE_OPENAI_DEPLOYMENT: "%s"
  AZURE_OPENAI_API_VERSION: "%s"
`, endpoint, deployment, apiVersion)

	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(configMap)
	return cmd.Run()
}

func createAzureOpenAISecret() error {
	apiKey := os.Getenv("AZURE_OPENAI_KEY")

	cmd := exec.Command("kubectl", "create", "secret", "generic", "azure-openai-secret",
		"--namespace", "innominatus-system",
		fmt.Sprintf("--from-literal=AZURE_OPENAI_KEY=%s", apiKey),
		"--dry-run=client", "-o", "yaml")

	output, err := cmd.Output()
	if err != nil {
		return err
	}

	applyCmd := exec.Command("kubectl", "apply", "-f", "-")
	applyCmd.Stdin = strings.NewReader(string(output))
	return applyCmd.Run()
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
```

### 4. Demo Components Registration (Simplified)

**File**: `internal/cli/demo_time.go` (Updated)

```go
package cli

type DemoComponent interface {
	Configure() error
	Check() (bool, error)
	Cleanup() error
}

var demoComponents = map[string]DemoComponent{
	"gitea":        &GiteaComponent{},
	"argocd":       &ArgoCDComponent{},
	"vault":        &VaultComponent{},
	"minio":        &MinioComponent{},
	"azure-openai": &AzureOpenAIComponent{}, // New optional component
	// ... other components
}

func runDemoTime() error {
	log.Info().Msg("🚀 Installing demo environment...")

	for name, component := range demoComponents {
		log.Info().Msgf("Configuring %s...", name)

		if err := component.Configure(); err != nil {
			// Azure OpenAI is optional - don't fail if it's not configured
			if name == "azure-openai" {
				log.Warn().Err(err).Msg("Azure OpenAI configuration skipped (optional)")
				continue
			}
			return fmt.Errorf("failed to configure %s: %w", name, err)
		}
	}

	log.Info().Msg("✓ Demo environment ready")
	return nil
}
```

---

## Benefits of This Approach

### KISS Compliance

1. **Single Configuration Source**: `NewEmbeddingProvider()` is the only place that determines which provider to use
2. **No Duplication**: Provider logic isn't scattered across multiple files
3. **Clear Flow**: `Environment → EmbeddingProvider → Service`

### SOLID Compliance

1. **SRP**: Each component has one job:
   - `embedding_config.go`: Determine embedding provider
   - `service.go`: Initialize AI service
   - `demo_components.go`: Manage K8s resources

2. **OCP**: Adding a new provider (e.g., Google Vertex AI) requires:
   - Add `newVertexAIProvider()` function
   - No changes to existing code

3. **DIP**: Service depends on `EmbeddingProvider` abstraction, not environment variables

4. **ISP**: `DemoComponent` interface is minimal (3 methods)

### Testing Benefits

```go
// Easy to test with mock providers
func TestServiceWithAzureOpenAI(t *testing.T) {
	azureProvider := &EmbeddingProvider{
		Type:    "azure-openai",
		BaseURL: "https://test.openai.azure.com/openai/deployments/test",
		APIKey:  "test-key",
		Model:   "test-deployment",
	}

	service, err := NewService(ctx, Config{
		AnthropicKey: "test-key",
	}, azureProvider)

	assert.NoError(t, err)
	assert.True(t, service.enabled)
}
```

---

## Configuration Examples

### Standard OpenAI

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
./innominatus
```

### Azure OpenAI

```bash
export AZURE_OPENAI_ENABLED="true"
export AZURE_OPENAI_ENDPOINT="https://my-resource.openai.azure.com"
export AZURE_OPENAI_KEY="abc123..."
export AZURE_OPENAI_DEPLOYMENT="text-embedding-ada-002"
export ANTHROPIC_API_KEY="sk-ant-..."
./innominatus
```

### Demo Environment with Azure OpenAI

```bash
export AZURE_OPENAI_ENDPOINT="https://my-resource.openai.azure.com"
export AZURE_OPENAI_KEY="abc123..."
export AZURE_OPENAI_DEPLOYMENT="text-embedding-ada-002"
./innominatus-ctl demo-time
```

---

## Implementation Checklist

- [ ] Create `internal/ai/embedding_config.go` with provider abstraction
- [ ] Update `internal/ai/service.go` to use dependency injection
- [ ] Create `internal/cli/demo_components.go` with component interface
- [ ] Update `internal/cli/demo_time.go` to use component abstraction
- [ ] Update `innominatus-ai-sdk` to support `CustomHeaders` in RAG config
- [ ] Add unit tests for `NewEmbeddingProvider()`
- [ ] Add integration tests for Azure OpenAI
- [ ] Update documentation with new configuration approach
- [ ] Add example `.env` files for different configurations

---

## Comparison: Before vs After

| Aspect | Before (Complex) | After (KISS + SOLID) |
|--------|-----------------|---------------------|
| **Configuration Logic** | Scattered across 5+ files | Single `embedding_config.go` file |
| **Provider Detection** | Duplicated in multiple places | One `NewEmbeddingProvider()` function |
| **Dependency** | Service reads `os.Getenv()` directly | Service receives injected config |
| **Testability** | Hard to mock environment | Easy to inject test providers |
| **Extensibility** | Requires modifying existing code | Add new provider function only |
| **Lines of Code** | ~200 lines across files | ~150 lines, centralized |

---

**Created**: 2025-10-08
**Purpose**: Simplified, SOLID-compliant Azure OpenAI integration
