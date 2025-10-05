// Shared workflow types (can be used in both server and client)
export interface WorkflowStep {
  name: string;
  type: string;
  condition?: string; // if field
  description?: string;
  config?: Record<string, any>;
  // Common fields from workflow YAML
  repoName?: string;
  owner?: string;
  appName?: string;
  namespace?: string;
  operation?: string;
  path?: string;
  [key: string]: any;
}

export interface WorkflowSpec {
  steps: WorkflowStep[];
}

export interface WorkflowMetadata {
  name: string;
  description?: string;
}

export interface WorkflowDefinition {
  apiVersion: string;
  kind: string;
  metadata: WorkflowMetadata;
  spec: WorkflowSpec;
}

/**
 * Get step icon and color based on type
 */
export function getStepStyle(stepType: string): {
  icon: string;
  color: string;
  bgColor: string;
  borderColor: string;
} {
  const styles: Record<
    string,
    { icon: string; color: string; bgColor: string; borderColor: string }
  > = {
    'gitea-repo': {
      icon: 'GitBranch',
      color: 'text-green-600',
      bgColor: 'bg-green-50',
      borderColor: 'border-green-300',
    },
    terraform: {
      icon: 'Cloud',
      color: 'text-blue-600',
      bgColor: 'bg-blue-50',
      borderColor: 'border-blue-300',
    },
    'terraform-generate': {
      icon: 'FileCode',
      color: 'text-indigo-600',
      bgColor: 'bg-indigo-50',
      borderColor: 'border-indigo-300',
    },
    kubernetes: {
      icon: 'Box',
      color: 'text-blue-500',
      bgColor: 'bg-blue-50',
      borderColor: 'border-blue-300',
    },
    'argocd-app': {
      icon: 'Workflow',
      color: 'text-orange-600',
      bgColor: 'bg-orange-50',
      borderColor: 'border-orange-300',
    },
    'git-commit-manifests': {
      icon: 'GitCommit',
      color: 'text-teal-600',
      bgColor: 'bg-teal-50',
      borderColor: 'border-teal-300',
    },
    policy: {
      icon: 'Shield',
      color: 'text-purple-600',
      bgColor: 'bg-purple-50',
      borderColor: 'border-purple-300',
    },
    ansible: {
      icon: 'Cog',
      color: 'text-red-600',
      bgColor: 'bg-red-50',
      borderColor: 'border-red-300',
    },
    validation: {
      icon: 'CheckCircle',
      color: 'text-emerald-600',
      bgColor: 'bg-emerald-50',
      borderColor: 'border-emerald-300',
    },
    default: {
      icon: 'Circle',
      color: 'text-gray-600',
      bgColor: 'bg-gray-50',
      borderColor: 'border-gray-300',
    },
  };

  return styles[stepType] || styles.default;
}

/**
 * Extract a brief description from step
 */
export function getStepDescription(step: WorkflowStep): string {
  if (step.description) {
    return step.description;
  }

  // Generate description from step config
  switch (step.type) {
    case 'gitea-repo':
      return `Create Git repository${step.repoName ? `: ${step.repoName}` : ''}`;
    case 'terraform':
      return `${step.operation || 'Apply'} Terraform configuration`;
    case 'terraform-generate':
      return `Generate Terraform for ${step.resource || 'resource'}`;
    case 'kubernetes':
      return `Deploy to Kubernetes${step.namespace ? ` (${step.namespace})` : ''}`;
    case 'argocd-app':
      return `Create ArgoCD Application${step.appName ? `: ${step.appName}` : ''}`;
    case 'git-commit-manifests':
      return 'Commit manifests to Git';
    case 'policy':
      return 'Run validation policy';
    case 'validation':
      return 'Validate deployment';
    default:
      return step.name;
  }
}
