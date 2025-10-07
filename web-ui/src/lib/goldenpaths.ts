// Golden Path metadata structure matching the backend
export interface ParameterSchema {
  type: string;
  default?: string;
  description?: string;
  required?: boolean;
  allowed_values?: string[];
  min?: number;
  max?: number;
  pattern?: string;
}

export interface GoldenPath {
  name: string;
  description: string;
  category: string;
  tags: string[];
  estimated_duration: string;
  workflow: string;
  parameters?: {
    [key: string]: ParameterSchema;
  };
}

// GitHub repository configuration
export const GITHUB_REPO = 'https://github.com/philipsahli/idp-o';
export const DEFAULT_BRANCH = 'v01';

// Static golden paths data (loaded from goldenpaths.yaml)
export const GOLDEN_PATHS: GoldenPath[] = [
  {
    name: 'deploy-app',
    description:
      'Deploy application with full GitOps pipeline including Git repository and ArgoCD onboarding',
    category: 'deployment',
    tags: ['deployment', 'gitops', 'argocd', 'production'],
    estimated_duration: '5-10 minutes',
    workflow: './workflows/deploy-app.yaml',
    parameters: {
      sync_policy: {
        type: 'enum',
        default: 'auto',
        description: 'ArgoCD sync policy for automatic or manual deployment',
        allowed_values: ['auto', 'manual'],
      },
      namespace_prefix: {
        type: 'string',
        default: '',
        description: 'Optional prefix for Kubernetes namespace',
        pattern: '^[a-z0-9\\-]*$',
      },
    },
  },
  {
    name: 'undeploy-app',
    description: 'Remove deployed application and clean up all associated resources',
    category: 'cleanup',
    tags: ['cleanup', 'teardown', 'removal'],
    estimated_duration: '2-5 minutes',
    workflow: './workflows/undeploy-app.yaml',
    parameters: {},
  },
  {
    name: 'ephemeral-env',
    description: 'Create temporary environment for testing with automatic TTL-based cleanup',
    category: 'environment',
    tags: ['testing', 'ephemeral', 'temporary', 'preview'],
    estimated_duration: '3-7 minutes',
    workflow: './workflows/ephemeral-env.yaml',
    parameters: {
      ttl: {
        type: 'duration',
        default: '2h',
        description: 'Time-to-live for ephemeral environment (hours, minutes, or days)',
        pattern: '^\\d+[hmd]$',
      },
      environment_type: {
        type: 'enum',
        default: 'preview',
        description: 'Type of ephemeral environment to create',
        allowed_values: ['preview', 'staging', 'development', 'testing'],
      },
      replicas: {
        type: 'int',
        default: '1',
        description: 'Number of application replicas',
        min: 1,
        max: 10,
      },
    },
  },
  {
    name: 'db-lifecycle',
    description: 'Manage database operations including backup, migration, and health checks',
    category: 'database',
    tags: ['database', 'backup', 'migration', 'maintenance'],
    estimated_duration: '5-15 minutes',
    workflow: './workflows/db-lifecycle.yaml',
    parameters: {
      operation: {
        type: 'enum',
        default: 'health-check',
        description: 'Database operation to perform',
        allowed_values: ['health-check', 'backup', 'migration', 'restore'],
      },
      backup_retention: {
        type: 'duration',
        default: '7d',
        description: 'Retention period for database backups',
        pattern: '^\\d+[dw]$',
      },
      enable_compression: {
        type: 'bool',
        default: 'true',
        description: 'Enable backup compression to save storage space',
      },
    },
  },
  {
    name: 'observability-setup',
    description: 'Setup monitoring and observability stack with metrics, logs, and tracing',
    category: 'observability',
    tags: ['monitoring', 'observability', 'metrics', 'logging', 'tracing'],
    estimated_duration: '10-20 minutes',
    workflow: './workflows/observability-setup.yaml',
    parameters: {
      enable_metrics: {
        type: 'bool',
        default: 'true',
        description: 'Enable Prometheus metrics collection',
      },
      enable_logs: {
        type: 'bool',
        default: 'true',
        description: 'Enable centralized logging with Loki',
      },
      enable_tracing: {
        type: 'bool',
        default: 'false',
        description: 'Enable distributed tracing with Tempo',
      },
      retention_days: {
        type: 'int',
        default: '30',
        description: 'Metrics and logs retention period in days',
        min: 7,
        max: 365,
      },
    },
  },
];

// Line number mappings for goldenpaths.yaml (for GitHub anchor links)
const GOLDENPATH_LINE_RANGES: { [key: string]: { start: number; end: number } } = {
  'deploy-app': { start: 2, end: 18 },
  'undeploy-app': { start: 20, end: 26 },
  'ephemeral-env': { start: 28, end: 50 },
  'db-lifecycle': { start: 52, end: 72 },
  'observability-setup': { start: 74, end: 98 },
};

/**
 * Get all golden paths
 */
export function getAllGoldenPaths(): GoldenPath[] {
  return GOLDEN_PATHS;
}

/**
 * Get a specific golden path by name
 */
export function getGoldenPathByName(name: string): GoldenPath | null {
  return GOLDEN_PATHS.find((path) => path.name === name) || null;
}

/**
 * Get category color for badges
 */
export function getCategoryColor(category: string): string {
  const colors: { [key: string]: string } = {
    deployment: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300',
    cleanup: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300',
    environment: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300',
    database: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300',
    observability: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300',
  };
  return colors[category] || 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300';
}

/**
 * Get icon color based on category
 */
export function getIconColor(category: string): string {
  const colors: { [key: string]: string } = {
    deployment: 'text-blue-500',
    cleanup: 'text-red-500',
    environment: 'text-green-500',
    database: 'text-purple-500',
    observability: 'text-orange-500',
  };
  return colors[category] || 'text-gray-500';
}

/**
 * Generate GitHub URL for a workflow file
 */
export function getGitHubWorkflowUrl(
  workflowPath: string,
  branch: string = DEFAULT_BRANCH
): string {
  // Remove leading ./ from path
  const cleanPath = workflowPath.replace(/^\.\//, '');
  return `${GITHUB_REPO}/blob/${branch}/${cleanPath}`;
}

/**
 * Generate GitHub URL for goldenpaths.yaml with line anchor
 */
export function getGitHubGoldenPathConfigUrl(
  pathName: string,
  branch: string = DEFAULT_BRANCH
): string {
  const lineRange = GOLDENPATH_LINE_RANGES[pathName];
  if (lineRange) {
    return `${GITHUB_REPO}/blob/${branch}/goldenpaths.yaml#L${lineRange.start}-L${lineRange.end}`;
  }
  return `${GITHUB_REPO}/blob/${branch}/goldenpaths.yaml`;
}

/**
 * Get parameter type badge color
 */
export function getParameterTypeBadgeColor(type: string): string {
  const colors: { [key: string]: string } = {
    string: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300',
    int: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300',
    bool: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300',
    enum: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300',
    duration: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300',
  };
  return colors[type] || 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300';
}
