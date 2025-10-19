'use client';

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { useEffect, useState } from 'react';
import { api, AdminConfig } from '@/lib/api';
import { Loader2, Lock, Settings2, GitBranch, Workflow } from 'lucide-react';

export default function SettingsPage() {
  const [config, setConfig] = useState<AdminConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchConfig() {
      try {
        setLoading(true);
        const response = await api.getAdminConfig();
        if (response.success && response.data) {
          setConfig(response.data);
          setError(null);
        } else {
          setError(response.error || 'Failed to load configuration');
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load configuration');
      } finally {
        setLoading(false);
      }
    }

    fetchConfig();
  }, []);

  if (loading) {
    return (
      <div className="p-6 flex items-center justify-center min-h-[400px]">
        <div className="flex flex-col items-center gap-2">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          <p className="text-sm text-muted-foreground">Loading configuration...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="rounded-md bg-destructive/15 p-4 text-destructive">
          <h3 className="font-semibold">Error Loading Configuration</h3>
          <p className="text-sm mt-1">{error}</p>
        </div>
      </div>
    );
  }

  if (!config) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">No configuration available</p>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground">Platform configuration and administrative settings</p>
      </div>

      {/* Admin Settings */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Settings2 className="h-5 w-5" />
            <CardTitle>Admin Settings</CardTitle>
          </div>
          <CardDescription>General administrative configuration</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="grid grid-cols-2 gap-2">
            <div className="text-sm font-medium text-muted-foreground">Cost Center</div>
            <div className="text-sm font-mono">{config.admin.defaultCostCenter}</div>

            <div className="text-sm font-medium text-muted-foreground">Default Runtime</div>
            <div className="text-sm font-mono">{config.admin.defaultRuntime}</div>

            <div className="text-sm font-medium text-muted-foreground">Splunk Index</div>
            <div className="text-sm font-mono">{config.admin.splunkIndex}</div>
          </div>
        </CardContent>
      </Card>

      {/* Resource Definitions */}
      <Card>
        <CardHeader>
          <CardTitle>Resource Definitions</CardTitle>
          <CardDescription>Mapping of resource types to platform implementations</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-2">
            {Object.entries(config.resourceDefinitions).map(([type, definition]) => (
              <div key={type} className="flex justify-between items-center py-1">
                <span className="text-sm font-medium text-muted-foreground">{type}</span>
                <span className="text-sm font-mono">{definition}</span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Policies */}
      <Card>
        <CardHeader>
          <CardTitle>Policies</CardTitle>
          <CardDescription>Platform-wide governance policies</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex justify-between items-center">
            <span className="text-sm font-medium">Enforce Backups</span>
            <Badge variant={config.policies.enforceBackups ? 'default' : 'secondary'}>
              {config.policies.enforceBackups ? 'Enabled' : 'Disabled'}
            </Badge>
          </div>

          <div>
            <div className="text-sm font-medium mb-2">Allowed Environments</div>
            <div className="flex flex-wrap gap-2">
              {config.policies.allowedEnvironments.map((env) => (
                <Badge key={env} variant="outline">
                  {env}
                </Badge>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Gitea Configuration */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <GitBranch className="h-5 w-5" />
            <CardTitle>Gitea Configuration</CardTitle>
          </div>
          <CardDescription>Git repository hosting configuration</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3">
          <div className="grid grid-cols-2 gap-2">
            <div className="text-sm font-medium text-muted-foreground">URL</div>
            <a
              href={config.gitea.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-sm font-mono text-primary hover:underline"
            >
              {config.gitea.url}
            </a>

            <div className="text-sm font-medium text-muted-foreground">Internal URL</div>
            <div className="text-sm font-mono">{config.gitea.internalURL}</div>

            <div className="text-sm font-medium text-muted-foreground">Username</div>
            <div className="text-sm font-mono">{config.gitea.username}</div>

            <div className="text-sm font-medium text-muted-foreground">Password</div>
            <div className="flex items-center gap-2">
              <Lock className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm font-mono text-muted-foreground">{config.gitea.password}</span>
            </div>

            <div className="text-sm font-medium text-muted-foreground">Organization</div>
            <div className="text-sm font-mono">{config.gitea.orgName}</div>
          </div>
        </CardContent>
      </Card>

      {/* ArgoCD Configuration */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Workflow className="h-5 w-5" />
            <CardTitle>ArgoCD Configuration</CardTitle>
          </div>
          <CardDescription>GitOps continuous deployment configuration</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3">
          <div className="grid grid-cols-2 gap-2">
            <div className="text-sm font-medium text-muted-foreground">URL</div>
            <a
              href={config.argocd.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-sm font-mono text-primary hover:underline"
            >
              {config.argocd.url}
            </a>

            <div className="text-sm font-medium text-muted-foreground">Username</div>
            <div className="text-sm font-mono">{config.argocd.username}</div>

            <div className="text-sm font-medium text-muted-foreground">Password</div>
            <div className="flex items-center gap-2">
              <Lock className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm font-mono text-muted-foreground">{config.argocd.password}</span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Workflow Policies */}
      <Card>
        <CardHeader>
          <CardTitle>Workflow Policies</CardTitle>
          <CardDescription>Workflow execution constraints and security policies</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-2">
            <div className="text-sm font-medium text-muted-foreground">Workflows Root</div>
            <div className="text-sm font-mono">{config.workflowPolicies.workflowsRoot}</div>

            <div className="text-sm font-medium text-muted-foreground">Max Duration</div>
            <div className="text-sm font-mono">{config.workflowPolicies.maxWorkflowDuration}</div>

            <div className="text-sm font-medium text-muted-foreground">Max Concurrent</div>
            <div className="text-sm font-mono">{config.workflowPolicies.maxConcurrentWorkflows}</div>

            <div className="text-sm font-medium text-muted-foreground">Max Steps</div>
            <div className="text-sm font-mono">{config.workflowPolicies.maxStepsPerWorkflow}</div>
          </div>

          <div>
            <div className="text-sm font-medium mb-2">Required Platform Workflows</div>
            <div className="flex flex-wrap gap-2">
              {config.workflowPolicies.requiredPlatformWorkflows.map((workflow) => (
                <Badge key={workflow} variant="secondary">
                  {workflow}
                </Badge>
              ))}
            </div>
          </div>

          <div>
            <div className="text-sm font-medium mb-2">Allowed Product Workflows</div>
            <div className="flex flex-wrap gap-2">
              {config.workflowPolicies.allowedProductWorkflows.map((workflow) => (
                <Badge key={workflow} variant="outline">
                  {workflow}
                </Badge>
              ))}
            </div>
          </div>

          <div>
            <div className="text-sm font-medium mb-2">Allowed Step Types</div>
            <div className="flex flex-wrap gap-2">
              {config.workflowPolicies.allowedStepTypes.map((type) => (
                <Badge key={type} variant="outline" className="text-xs">
                  {type}
                </Badge>
              ))}
            </div>
          </div>

          <div>
            <div className="text-sm font-medium mb-2">Security</div>
            <div className="grid gap-2 pl-4">
              <div className="text-sm">
                <span className="font-medium text-muted-foreground">Require Approval:</span>{' '}
                {config.workflowPolicies.security.requireApproval.join(', ')}
              </div>
              <div className="text-sm">
                <span className="font-medium text-muted-foreground">Allowed Executors:</span>{' '}
                {config.workflowPolicies.security.allowedExecutors.join(', ')}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
