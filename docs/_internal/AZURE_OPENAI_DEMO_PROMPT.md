# Azure OpenAI Demo Environment Configuration

## Objective

Configure the demo environment to use Azure OpenAI (Microsoft's managed cloud service) instead of the public OpenAI API for embeddings. This provides an example of enterprise integration with Azure's AI services.

## Background

Currently, the AI assistant uses:
- **OpenAI API** (`text-embedding-3-small`) for embeddings via `OPENAI_API_KEY`
- **Anthropic API** (`claude-sonnet-4-5`) for LLM chat via `ANTHROPIC_API_KEY`

In the demo environment, we want to demonstrate how to configure Azure OpenAI as an alternative embedding provider, showing enterprise customers how to integrate with Azure's managed services.

## Azure OpenAI Service Overview

**Azure OpenAI** is Microsoft's cloud service providing:
- Managed OpenAI models (GPT, Embeddings, DALL-E, etc.)
- Enterprise-grade security and compliance
- Regional deployment options
- Azure Active Directory authentication
- No self-hosted infrastructure required

**Key Differences from OpenAI API:**
- Different base URL format: `https://<resource-name>.openai.azure.com`
- Requires `api-version` parameter in requests
- Uses Azure API keys (not OpenAI keys)
- Models deployed to specific deployment names (not direct model names)

## Requirements

### 1. Azure OpenAI Prerequisites

**Azure Account Setup** (one-time, manual):
```bash
# 1. Create Azure account (if not exists)
# 2. Request access to Azure OpenAI service
#    https://aka.ms/oai/access
# 3. Wait for approval (can take several days)

# 4. Create Azure OpenAI resource via Azure Portal or CLI
az cognitiveservices account create \
  --name innominatus-demo-openai \
  --resource-group innominatus-demo \
  --location eastus \
  --kind OpenAI \
  --sku S0

# 5. Deploy embedding model
az cognitiveservices account deployment create \
  --resource-group innominatus-demo \
  --name innominatus-demo-openai \
  --deployment-name text-embedding-ada-002 \
  --model-name text-embedding-ada-002 \
  --model-version "2" \
  --model-format OpenAI \
  --scale-capacity 1

# 6. Get API key
az cognitiveservices account keys list \
  --resource-group innominatus-demo \
  --name innominatus-demo-openai
```

**Required Azure Information:**
- Azure OpenAI resource name: `innominatus-demo-openai`
- Endpoint: `https://innominatus-demo-openai.openai.azure.com`
- API Key: (from Azure Portal or CLI)
- Deployment name: `text-embedding-ada-002`
- API Version: `2024-02-15-preview` (or latest)

### 2. Integration with `demo-time` Command

**File**: `internal/cli/demo_time.go`

**Update `demo-time` logic**:

```go
func configureAzureOpenAI() error {
    log.Info().Msg("Configuring Azure OpenAI integration...")

    // Check if Azure credentials are provided
    azureEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
    azureKey := os.Getenv("AZURE_OPENAI_KEY")
    azureDeployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")

    if azureEndpoint == "" || azureKey == "" {
        log.Warn().Msg("⚠ Azure OpenAI not configured (optional)")
        log.Warn().Msg("  Set AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_KEY, and AZURE_OPENAI_DEPLOYMENT to enable")
        return nil
    }

    // Create Kubernetes ConfigMap with Azure OpenAI configuration
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
  AZURE_OPENAI_API_VERSION: "2024-02-15-preview"
`, azureEndpoint, azureDeployment)

    // Apply ConfigMap
    cmd := exec.Command("kubectl", "apply", "-f", "-")
    cmd.Stdin = strings.NewReader(configMap)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to create Azure OpenAI ConfigMap: %w", err)
    }

    // Create Secret with API key
    cmd = exec.Command("kubectl", "create", "secret", "generic", "azure-openai-secret",
        "--namespace", "innominatus-system",
        fmt.Sprintf("--from-literal=api-key=%s", azureKey),
        "--dry-run=client", "-o", "yaml")
    output, _ := cmd.Output()

    applyCmd := exec.Command("kubectl", "apply", "-f", "-")
    applyCmd.Stdin = strings.NewReader(string(output))
    if err := applyCmd.Run(); err != nil {
        return fmt.Errorf("failed to create Azure OpenAI Secret: %w", err)
    }

    // Update innominatus deployment to use Azure OpenAI config
    // (Patch deployment to add envFrom for ConfigMap and Secret)
    patchJSON := `{
        "spec": {
            "template": {
                "spec": {
                    "containers": [{
                        "name": "innominatus",
                        "envFrom": [
                            {"configMapRef": {"name": "azure-openai-config"}},
                            {"secretRef": {"name": "azure-openai-secret"}}
                        ]
                    }]
                }
            }
        }
    }`

    cmd = exec.Command("kubectl", "patch", "deployment", "innominatus",
        "-n", "innominatus-system",
        "--type", "strategic",
        "-p", patchJSON)
    if err := cmd.Run(); err != nil {
        log.Warn().Err(err).Msg("Failed to patch innominatus deployment")
    }

    log.Info().Msg("✓ Azure OpenAI integration configured")
    return nil
}
```

**Add to demo component list**:
```go
type DemoComponent struct {
    Name        string
    ConfigFunc  func() error  // Changed from InstallFunc
    CheckFunc   func() (bool, error)
    CleanupFunc func() error
}

var demoComponents = []DemoComponent{
    // ... existing components (Gitea, ArgoCD, Vault, etc.)
    {
        Name:        "Azure OpenAI",
        ConfigFunc:  configureAzureOpenAI,  // Configuration, not installation
        CheckFunc:   checkAzureOpenAI,
        CleanupFunc: cleanupAzureOpenAI,
    },
}
```

### 3. Environment Configuration

**Environment variables for demo-time**:

```bash
# Azure OpenAI configuration (optional)
export AZURE_OPENAI_ENDPOINT="https://innominatus-demo-openai.openai.azure.com"
export AZURE_OPENAI_KEY="your-azure-api-key"
export AZURE_OPENAI_DEPLOYMENT="text-embedding-ada-002"
export AZURE_OPENAI_API_VERSION="2024-02-15-preview"

# Run demo-time with Azure OpenAI
./innominatus-ctl demo-time
```

**Server environment detection** (File: `cmd/server/main.go`):

```go
func getEmbeddingConfig() (provider string, baseURL string, apiKey string, model string) {
    // Check if Azure OpenAI is configured
    if os.Getenv("AZURE_OPENAI_ENABLED") == "true" {
        endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
        deployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")
        apiVersion := os.Getenv("AZURE_OPENAI_API_VERSION")
        apiKey := os.Getenv("AZURE_OPENAI_KEY")

        if endpoint != "" && deployment != "" && apiKey != "" {
            // Azure OpenAI URL format: https://{resource}.openai.azure.com/openai/deployments/{deployment}/embeddings?api-version={version}
            baseURL := fmt.Sprintf("%s/openai/deployments/%s", endpoint, deployment)
            return "azure-openai", baseURL, apiKey, deployment
        }
    }

    // Fallback to standard OpenAI
    return "openai", "https://api.openai.com/v1", os.Getenv("OPENAI_API_KEY"), "text-embedding-3-small"
}
```

### 4. AI Service Integration

**File**: `internal/ai/service.go`

**Update SDK initialization** to support Azure OpenAI:

```go
func NewService(ctx context.Context, cfg Config) (*Service, error) {
    log.Debug().Msg("Initializing AI service")

    // Determine embedding provider (Azure OpenAI or standard OpenAI)
    embeddingProvider := "openai"
    embeddingBaseURL := "https://api.openai.com/v1"
    embeddingAPIKey := cfg.OpenAIKey
    embeddingModel := "text-embedding-3-small"

    // Check for Azure OpenAI configuration
    if os.Getenv("AZURE_OPENAI_ENABLED") == "true" {
        azureEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
        azureDeployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")
        azureAPIKey := os.Getenv("AZURE_OPENAI_KEY")
        azureAPIVersion := os.Getenv("AZURE_OPENAI_API_VERSION")

        if azureEndpoint != "" && azureDeployment != "" && azureAPIKey != "" {
            embeddingProvider = "azure-openai"
            embeddingBaseURL = fmt.Sprintf("%s/openai/deployments/%s?api-version=%s",
                azureEndpoint, azureDeployment, azureAPIVersion)
            embeddingAPIKey = azureAPIKey
            embeddingModel = azureDeployment

            log.Info().
                Str("provider", "azure-openai").
                Str("endpoint", azureEndpoint).
                Str("deployment", azureDeployment).
                Msg("Using Azure OpenAI for embeddings")
        }
    }

    if embeddingAPIKey == "" || cfg.AnthropicKey == "" {
        log.Warn().
            Bool("has_embedding_key", embeddingAPIKey != "").
            Bool("has_anthropic_key", cfg.AnthropicKey != "").
            Msg("AI service disabled: missing API keys")
        return &Service{enabled: false}, nil
    }

    log.Debug().
        Str("embedding_provider", embeddingProvider).
        Str("embedding_base_url", embeddingBaseURL).
        Str("llm_provider", "anthropic").
        Msg("Initializing AI SDK")

    sdk, err := platformai.New(ctx, &platformai.Config{
        LLM: platformai.LLMConfig{
            Provider:    "anthropic",
            APIKey:      cfg.AnthropicKey,
            Model:       "claude-sonnet-4-5-20250929",
            Temperature: 0.7,
            MaxTokens:   4096,
        },
        RAG: &rag.Config{
            EmbeddingProvider: embeddingProvider,
            APIKey:            embeddingAPIKey,
            BaseURL:           embeddingBaseURL,
            Model:             embeddingModel,
        },
    })
    if err != nil {
        return nil, fmt.Errorf("failed to initialize AI SDK: %w", err)
    }

    // ... rest of initialization
}
```

**Note**: The `innominatus-ai-sdk` must support:
- `BaseURL` configuration for custom endpoints
- Azure OpenAI authentication headers (`api-key` instead of `Authorization: Bearer`)
- Azure OpenAI URL format with `api-version` parameter

### 5. Health Checks and Status

**File**: `internal/cli/demo_status.go`

```go
func checkAzureOpenAI() (bool, error) {
    // Check if Azure OpenAI ConfigMap exists
    cmd := exec.Command("kubectl", "get", "configmap", "azure-openai-config",
        "-n", "innominatus-system",
        "-o", "jsonpath={.data.AZURE_OPENAI_ENABLED}")

    output, err := cmd.Output()
    if err != nil {
        return false, nil // Not configured (optional component)
    }

    enabled := strings.TrimSpace(string(output))
    if enabled != "true" {
        return false, nil
    }

    // Check if Secret exists
    cmd = exec.Command("kubectl", "get", "secret", "azure-openai-secret",
        "-n", "innominatus-system",
        "-o", "name")
    if err := cmd.Run(); err != nil {
        return false, fmt.Errorf("Azure OpenAI secret not found")
    }

    return true, nil
}

func getAzureOpenAIInfo() string {
    cmd := exec.Command("kubectl", "get", "configmap", "azure-openai-config",
        "-n", "innominatus-system",
        "-o", "jsonpath={.data.AZURE_OPENAI_ENDPOINT}")

    output, _ := cmd.Output()
    endpoint := strings.TrimSpace(string(output))

    if endpoint == "" {
        return "Not configured"
    }
    return endpoint
}
```

**Update `demo-status` output**:
```
Azure OpenAI:        ✓ Configured (https://innominatus-demo-openai.openai.azure.com)
  Deployment:        text-embedding-ada-002
  Provider:          Microsoft Azure
  Status:            Active (cloud service)
```

### 6. Cleanup with `demo-nuke`

**File**: `internal/cli/demo_nuke.go`

```go
func cleanupAzureOpenAI() error {
    log.Info().Msg("Cleaning up Azure OpenAI configuration...")

    // Delete ConfigMap
    cmd := exec.Command("kubectl", "delete", "configmap", "azure-openai-config",
        "-n", "innominatus-system",
        "--ignore-not-found")
    if err := cmd.Run(); err != nil {
        log.Warn().Err(err).Msg("Failed to delete Azure OpenAI ConfigMap")
    }

    // Delete Secret
    cmd = exec.Command("kubectl", "delete", "secret", "azure-openai-secret",
        "-n", "innominatus-system",
        "--ignore-not-found")
    if err := cmd.Run(); err != nil {
        log.Warn().Err(err).Msg("Failed to delete Azure OpenAI Secret")
    }

    log.Info().Msg("✓ Azure OpenAI configuration cleaned up")
    log.Info().Msg("  Note: Azure resources in cloud are not affected")
    return nil
}
```

**Note**: `demo-nuke` only removes Kubernetes configuration, not the Azure cloud resources.

### 7. Documentation Updates

**File**: `CLAUDE.md` (Demo Environment section)

Add Azure OpenAI to the list of demo components:

```markdown
**Optional Components:**
- **Azure OpenAI**: Enterprise embedding service integration (Microsoft Azure cloud)
  - Demonstrates Azure OpenAI integration for embeddings
  - Requires Azure account and Azure OpenAI resource
  - Set environment variables before running `demo-time`:
    ```bash
    export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
    export AZURE_OPENAI_KEY="your-azure-api-key"
    export AZURE_OPENAI_DEPLOYMENT="text-embedding-ada-002"
    ```
  - Falls back to standard OpenAI API if not configured
```

**File**: `docs/platform-team-guide/ai-assistant.md`

Document Azure OpenAI integration:

```markdown
### Azure OpenAI Integration (Enterprise)

For enterprise environments using Microsoft Azure, innominatus can use Azure OpenAI instead of the public OpenAI API:

#### Prerequisites

1. **Azure Account**: Active Azure subscription
2. **Azure OpenAI Access**: Request access at https://aka.ms/oai/access
3. **Resource Setup**:
   ```bash
   # Create Azure OpenAI resource
   az cognitiveservices account create \
     --name your-resource-name \
     --resource-group your-rg \
     --location eastus \
     --kind OpenAI \
     --sku S0

   # Deploy embedding model
   az cognitiveservices account deployment create \
     --resource-group your-rg \
     --name your-resource-name \
     --deployment-name text-embedding-ada-002 \
     --model-name text-embedding-ada-002 \
     --model-version "2" \
     --model-format OpenAI
   ```

#### Configuration

**For demo-time:**
```bash
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
export AZURE_OPENAI_KEY="your-azure-api-key"
export AZURE_OPENAI_DEPLOYMENT="text-embedding-ada-002"
./innominatus-ctl demo-time
```

**For production Kubernetes:**
```bash
kubectl create configmap azure-openai-config \
  -n innominatus-system \
  --from-literal=AZURE_OPENAI_ENABLED=true \
  --from-literal=AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com" \
  --from-literal=AZURE_OPENAI_DEPLOYMENT="text-embedding-ada-002" \
  --from-literal=AZURE_OPENAI_API_VERSION="2024-02-15-preview"

kubectl create secret generic azure-openai-secret \
  -n innominatus-system \
  --from-literal=AZURE_OPENAI_KEY="your-azure-api-key"
```

#### Benefits

- **Enterprise Security**: Azure AD integration, private endpoints
- **Compliance**: SOC 2, HIPAA, GDPR compliant
- **Regional Deployment**: Data residency requirements
- **Cost Management**: Azure billing and quotas
- **SLA**: Microsoft enterprise SLA guarantees
```

### 8. Testing

**Verify Azure OpenAI integration**:

```bash
# Set Azure credentials (if you have an Azure OpenAI resource)
export AZURE_OPENAI_ENDPOINT="https://innominatus-demo-openai.openai.azure.com"
export AZURE_OPENAI_KEY="your-azure-key"
export AZURE_OPENAI_DEPLOYMENT="text-embedding-ada-002"

# Install demo environment with Azure OpenAI
./innominatus-ctl demo-time

# Check Azure OpenAI status
./innominatus-ctl demo-status | grep "Azure OpenAI"

# Test embeddings via AI assistant
curl -X POST http://localhost:8081/api/ai/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_TOKEN" \
  -d '{"message":"What is innominatus?"}'

# Check logs for Azure OpenAI usage
kubectl logs -n innominatus-system deployment/innominatus | grep "azure-openai"

# Cleanup (removes K8s config only, not Azure resources)
./innominatus-ctl demo-nuke
```

**Without Azure credentials:**
```bash
# Demo-time will skip Azure OpenAI configuration
./innominatus-ctl demo-time

# Status will show "Not configured"
./innominatus-ctl demo-status | grep "Azure OpenAI"
# Output: Azure OpenAI: ⚠ Not configured (optional)
```

## Implementation Checklist

- [ ] Update `demo-time` to configure Azure OpenAI (optional component)
- [ ] Create Kubernetes ConfigMap/Secret for Azure credentials
- [ ] Add health check function for Azure OpenAI configuration
- [ ] Update `demo-status` to show Azure OpenAI integration status
- [ ] Update `demo-nuke` to clean up Azure OpenAI K8s resources
- [ ] Add environment variable detection in `internal/ai/service.go`
- [ ] Update `innominatus-ai-sdk` to support Azure OpenAI API format
  - [ ] Support `api-key` header (not `Authorization: Bearer`)
  - [ ] Support `api-version` query parameter
  - [ ] Handle Azure OpenAI URL format
- [ ] Update documentation (CLAUDE.md, docs/platform-team-guide/ai-assistant.md)
- [ ] Test with real Azure OpenAI resource
- [ ] Test fallback to standard OpenAI when Azure not configured
- [ ] Update logging to show which embedding provider is being used
- [ ] Document Azure resource setup process

## Azure OpenAI vs Standard OpenAI

| Feature | Standard OpenAI | Azure OpenAI |
|---------|----------------|--------------|
| **Infrastructure** | OpenAI managed cloud | Microsoft Azure cloud |
| **Authentication** | `Authorization: Bearer <key>` | `api-key: <key>` header |
| **Base URL** | `https://api.openai.com/v1` | `https://{resource}.openai.azure.com` |
| **Model Access** | Direct model names | Deployment names |
| **API Version** | Not required | Required (`?api-version=2024-02-15-preview`) |
| **Enterprise Features** | Limited | Azure AD, Private Endpoints, VNets |
| **Compliance** | Standard | HIPAA, SOC 2, GDPR, regional compliance |
| **Billing** | OpenAI billing | Azure billing (unified with other services) |

## Notes

- **Azure OpenAI is optional**: Demo environment works without it (falls back to standard OpenAI)
- **Cloud service**: No Kubernetes deployment required - uses Azure's managed service
- **Requires Azure account**: Users must have Azure subscription and OpenAI access
- **Approval process**: Azure OpenAI access requires Microsoft approval (can take days)
- **Production ready**: Azure OpenAI is designed for enterprise production use
- **Cost considerations**: Azure OpenAI pricing may differ from standard OpenAI
- **Regional availability**: Azure OpenAI is not available in all Azure regions

## Success Criteria

1. `demo-time` successfully configures Azure OpenAI when credentials provided
2. `demo-time` gracefully skips Azure OpenAI when credentials not provided
3. AI assistant uses Azure OpenAI for embeddings (verified in logs)
4. `demo-status` reports Azure OpenAI configuration status
5. `demo-nuke` cleanly removes K8s configuration without affecting Azure resources
6. Documentation clearly explains Azure OpenAI setup and benefits
7. System falls back to standard OpenAI when Azure OpenAI not configured

---

**Created**: 2025-10-08
**Updated**: 2025-10-08 - Corrected to use Azure's managed cloud service
**For**: innominatus demo environment enhancement
