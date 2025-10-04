# Backstage Software Templates for innominatus

This directory contains Backstage Software Templates that provide wizard-based forms for creating Score specifications and deploying applications via the innominatus orchestrator.

## Quick Start (Demo Environment)

When you run `./innominatus-ctl demo-time`, the templates are automatically seeded to the Gitea `platform-config` repository. To use them in Backstage:

1. **Open Backstage**: http://backstage.localtest.me
2. **Register the template**:
   - Click "Create" in the sidebar
   - Click "Register Existing Component"
   - Enter URL: `http://gitea-http.gitea.svc.cluster.local:3000/giteaadmin/platform-config/raw/branch/main/backstage-templates/catalog-info.yaml`
   - Click "Analyze"
   - Click "Import"
3. **Use the template**:
   - Go back to "Create" (or refresh the page)
   - Find "Deploy Application with Score"
   - Fill out the 4-step wizard
   - Generate your Score specification!

## Overview

Instead of writing Score YAML files manually, developers can use these Backstage templates to fill out user-friendly forms that generate properly validated Score specifications. The templates integrate seamlessly with the innominatus orchestration platform.

## Templates

### Score Application Template

**File**: `score-app-template.yaml`

A comprehensive multi-step wizard that guides developers through creating a complete Score specification with:

- **Step 1: Application Details** - Name, description, owner, environment
- **Step 2: Container Configuration** - Image, port, replicas, resources, environment variables
- **Step 3: Resource Provisioning** - Optional S3 storage and database provisioning
- **Step 4: Workflow Configuration** - Select deployment workflow steps

## Installation in Backstage

### Method 1: Register Template via UI

1. Navigate to your Backstage instance
2. Go to **Create** → **Register Existing Component**
3. Enter the URL to the template:
   ```
   https://github.com/philipsahli/innominatus/blob/main/backstage-templates/score-app-template.yaml
   ```
4. Click **Analyze** and then **Import**

### Method 2: Add to Backstage app-config.yaml

```yaml
catalog:
  locations:
    - type: url
      target: https://github.com/philipsahli/innominatus/blob/main/backstage-templates/score-app-template.yaml
      rules:
        - allow: [Template]

    # Or for local development
    - type: file
      target: /path/to/innominatus/backstage-templates/score-app-template.yaml
      rules:
        - allow: [Template]
```

### Method 3: Bulk Import (for multiple templates)

```yaml
catalog:
  locations:
    - type: url
      target: https://github.com/philipsahli/innominatus/blob/main/backstage-templates/*/template.yaml
      rules:
        - allow: [Template]
```

## Template Features

### Form Validation

The template includes built-in validation:

- **Application Name**: Lowercase alphanumeric with dashes, max 50 characters
- **Container Port**: Valid port range (1-65535)
- **Replicas**: Reasonable replica count (1-10)
- **Resource Limits**: Pre-defined CPU and memory options

### Conditional Fields

Smart form behavior:

- **S3 Bucket Name** field appears only when "Provision S3 Storage" is checked
- **Database Type** and **Database Size** appear only when "Provision Database" is checked
- Form adapts based on user selections

### Resource Provisioning Options

#### S3 Storage
- Provisions Minio S3-compatible bucket
- Automatically injects S3 credentials into application environment
- Uses Terraform via innominatus workflow

#### Database Options
- **PostgreSQL**: Version 15, configurable storage
- **MySQL**: Version 8.0, configurable storage
- **Redis**: Version 7.0, in-memory cache
- Provisions via Helm charts through innominatus

### Workflow Steps

Developers can select which deployment steps to execute:

1. **create-namespace**: Create Kubernetes namespace for the environment
2. **deploy-app**: Deploy application to Kubernetes with specified replicas
3. **provision-storage**: Execute Terraform to provision S3 bucket (if enabled)
4. **setup-gitops**: Configure ArgoCD Application for GitOps deployment
5. **create-repository**: Create Git repository in Gitea for application code

## Generated Files

The template generates two key files:

### 1. `score.yaml`

Complete Score specification including:
- Metadata (name, description, labels, owner)
- Container configuration (image, ports, variables, resources)
- Resource definitions (S3 buckets, databases)
- Workflow steps for innominatus orchestrator

### 2. `catalog-info.yaml`

Backstage catalog entry including:
- Component metadata
- Annotations linking to innominatus, ArgoCD, Gitea
- Resource dependencies (S3, database)
- API definition
- External links to monitoring and deployment tools

## Usage Workflow

### For Developers

1. **Navigate to Backstage** → **Create** → **Choose a Template**
2. **Select** "Deploy Application with Score"
3. **Fill out the wizard** (4 steps, ~2-3 minutes):
   - Application details (name, owner, environment)
   - Container configuration (image, resources)
   - Resource provisioning (S3, database)
   - Workflow steps selection
4. **Review** the generated Score specification
5. **Click Deploy** (or download files for manual deployment)

### Deployment Options

After generating the Score specification:

#### Option 1: innominatus CLI

```bash
./innominatus-ctl run deploy-app score.yaml
```

#### Option 2: innominatus API

```bash
curl -X POST http://innominatus.localtest.me/api/specs \
  -H "Content-Type: application/yaml" \
  -H "Authorization: Bearer $API_TOKEN" \
  --data-binary @score.yaml
```

#### Option 3: Backstage Custom Action (Advanced)

Extend the template with a custom Backstage action to automatically deploy to innominatus. See the Custom Actions section below.

## Extending the Template

### Adding Custom Actions

To automatically deploy generated specs to innominatus, add a custom Backstage action:

**1. Create custom action** (`backstage-plugins/innominatus-backend/src/actions/deployToInnominatus.ts`):

```typescript
import { createTemplateAction } from '@backstage/plugin-scaffolder-node';

export const createDeployToInnominatusAction = () => {
  return createTemplateAction({
    id: 'innominatus:deploy',
    schema: {
      input: {
        required: ['scoreSpec', 'apiUrl'],
        properties: {
          scoreSpec: { type: 'string', description: 'Score YAML content' },
          apiUrl: { type: 'string', description: 'innominatus API URL' },
          apiToken: { type: 'string', description: 'API authentication token' },
        },
      },
      output: {
        properties: {
          workflowId: { type: 'string', description: 'Workflow execution ID' },
          deploymentUrl: { type: 'string', description: 'Link to deployment status' },
        },
      },
    },
    async handler(ctx) {
      const { scoreSpec, apiUrl, apiToken } = ctx.input;

      ctx.logger.info(`Deploying to innominatus: ${apiUrl}`);

      const response = await fetch(`${apiUrl}/api/specs`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/yaml',
          'Authorization': `Bearer ${apiToken}`,
        },
        body: scoreSpec,
      });

      if (!response.ok) {
        throw new Error(`Deployment failed: ${response.statusText}`);
      }

      const result = await response.json();

      ctx.output('workflowId', result.workflow_id);
      ctx.output('deploymentUrl', `${apiUrl}/workflows/${result.workflow_id}`);

      ctx.logger.info(`Deployment successful: Workflow ID ${result.workflow_id}`);
    },
  });
};
```

**2. Register the action** in your Backstage backend:

```typescript
// packages/backend/src/plugins/scaffolder.ts
import { createDeployToInnominatusAction } from '../actions/deployToInnominatus';

export default async function createPlugin(env: PluginEnvironment): Promise<Router> {
  const actions = [
    createDeployToInnominatusAction(),
    // ... other actions
  ];

  return await createRouter({
    actions,
    // ... other config
  });
}
```

**3. Add deployment step** to template:

```yaml
steps:
  - id: fetch-template
    name: Fetch Score Template
    action: fetch:template
    # ... existing config

  - id: deploy
    name: Deploy to innominatus
    action: innominatus:deploy
    input:
      scoreSpec: ${{ steps['fetch-template'].output.path }}/score.yaml
      apiUrl: ${{ parameters.innominatusUrl | default('http://innominatus.platform.svc.cluster.local:8081') }}
      apiToken: ${{ secrets.INNOMINATUS_API_TOKEN }}
```

**4. Configure Backstage** (`app-config.yaml`):

```yaml
innominatus:
  apiUrl: http://innominatus.platform.svc.cluster.local:8081
  # API token stored in Backstage secrets
```

### Adding Custom Fields

To add new input fields, extend the `parameters` section:

```yaml
parameters:
  - title: My Custom Step
    properties:
      customField:
        title: Custom Field
        type: string
        description: My custom configuration
        default: "default-value"
```

Then use it in the template:

```yaml
# In skeleton/score.yaml
metadata:
  custom_annotation: ${{ values.customField }}
```

## Configuration

### Environment Variables

When using the custom action, configure these in your Backstage deployment:

```bash
# innominatus API endpoint
INNOMINATUS_API_URL=http://innominatus.platform.svc.cluster.local:8081

# API token for authentication
INNOMINATUS_API_TOKEN=your-api-token-here
```

### Kubernetes ConfigMap

For Kubernetes deployments:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: backstage-config
data:
  app-config.yaml: |
    innominatus:
      apiUrl: http://innominatus.platform.svc.cluster.local:8081
```

## Examples

### Example 1: Simple Web Application

**User Input**:
- Name: `my-web-app`
- Container Image: `nginx:latest`
- Environment: `development`
- Replicas: `2`

**Generated Score**:
```yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-web-app
  environment: development
containers:
  main:
    image: nginx:latest
    ports:
      - port: 8080
```

### Example 2: Application with S3 Storage

**User Input**:
- Name: `data-processor`
- Container Image: `myapp:v1.0.0`
- Enable S3: ✓
- S3 Bucket Name: `data-processor-storage`

**Generated Resources**:
```yaml
resources:
  storage:
    type: minio-s3-bucket
    properties:
      bucket_name: data-processor-storage
      endpoint: http://minio.minio-system.svc.cluster.local:9000
```

### Example 3: Full Stack Application

**User Input**:
- Name: `fullstack-app`
- Container Image: `fullstack:latest`
- Enable S3: ✓
- Enable Database: ✓ (PostgreSQL, 10Gi)
- Workflow Steps: All selected

**Generated Workflow**:
```yaml
workflows:
  deploy:
    steps:
      - name: Create Kubernetes namespace
        type: kubernetes
      - name: Provision S3 bucket
        type: terraform
      - name: Provision postgresql database
        type: helm
      - name: Deploy application to Kubernetes
        type: kubernetes
      - name: Create Git repository
        type: gitea-repo
      - name: Setup ArgoCD application
        type: argocd
```

## Troubleshooting

### Template Not Appearing

- Check Backstage catalog import status
- Verify template YAML is valid
- Check Backstage logs for import errors

### Validation Errors

- Ensure application name follows pattern: `^[a-z][a-z0-9-]*$`
- Container port must be between 1-65535
- Replica count must be between 1-10

### Deployment Fails

- Verify innominatus API is accessible
- Check API token is valid
- Review innominatus logs for workflow execution errors

## Integration with innominatus Demo

When using the innominatus demo environment (`./innominatus-ctl demo-time`):

1. **Backstage** runs at http://backstage.localtest.me
2. **innominatus** runs at http://innominatus.localtest.me (port 8081)
3. Templates can deploy to the demo environment directly
4. All provisioned resources (Gitea, ArgoCD, Minio, etc.) are available

## Resources

- **Backstage Software Templates**: https://backstage.io/docs/features/software-templates/
- **Score Specification**: https://score.dev
- **innominatus Documentation**: ../README.md
- **Template Editor**: Use Backstage's built-in template editor to test templates

## Contributing

To contribute new templates or improve existing ones:

1. Fork the repository
2. Create templates in `backstage-templates/`
3. Test with Backstage Template Editor
4. Submit a pull request with examples and documentation

## License

These templates are licensed under Apache License 2.0, same as the innominatus project.
