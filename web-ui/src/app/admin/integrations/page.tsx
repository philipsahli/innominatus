'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useEffect, useState } from 'react';
import { api, AdminConfig } from '@/lib/api';
import {
  Loader2,
  Lock,
  GitBranch,
  Workflow,
  Shield,
  Key,
  Database,
  Activity,
  BarChart3,
  Server,
  ExternalLink,
  CheckCircle2,
  XCircle,
} from 'lucide-react';
import { AdminRouteProtection } from '@/components/admin-route-protection';

interface Integration {
  name: string;
  category: string;
  description: string;
  icon: React.ComponentType<{ className?: string }>;
  configured: boolean;
  url?: string;
  details?: { label: string; value: string; masked?: boolean }[];
}

export default function IntegrationsPage() {
  return (
    <AdminRouteProtection>
      <IntegrationsPageContent />
    </AdminRouteProtection>
  );
}

function IntegrationsPageContent() {
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
          <p className="text-sm text-muted-foreground">Loading integrations...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="rounded-md bg-destructive/15 p-4 text-destructive">
          <h3 className="font-semibold">Error Loading Integrations</h3>
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

  // Build integrations list from config
  const integrations: Integration[] = [
    // Version Control
    {
      name: 'Gitea',
      category: 'Version Control',
      description: 'Git repository hosting and code management',
      icon: GitBranch,
      configured: !!config.gitea.url,
      url: config.gitea.url,
      details: config.gitea.url
        ? [
            { label: 'URL', value: config.gitea.url },
            { label: 'Internal URL', value: config.gitea.internalURL },
            { label: 'Username', value: config.gitea.username },
            { label: 'Password', value: config.gitea.password, masked: true },
            { label: 'Organization', value: config.gitea.orgName },
          ]
        : undefined,
    },
    // GitOps
    {
      name: 'ArgoCD',
      category: 'GitOps',
      description: 'Declarative continuous deployment for Kubernetes',
      icon: Workflow,
      configured: !!config.argocd.url,
      url: config.argocd.url,
      details: config.argocd.url
        ? [
            { label: 'URL', value: config.argocd.url },
            { label: 'Username', value: config.argocd.username },
            { label: 'Password', value: config.argocd.password, masked: true },
          ]
        : undefined,
    },
    // Security
    {
      name: 'Vault',
      category: 'Security',
      description: 'Secrets and sensitive data management',
      icon: Shield,
      configured: !!config.vault.url,
      url: config.vault.url,
      details: config.vault.url
        ? [
            { label: 'URL', value: config.vault.url },
            { label: 'Token', value: config.vault.token, masked: true },
            { label: 'Namespace', value: config.vault.namespace || 'default' },
          ]
        : undefined,
    },
    {
      name: 'Keycloak',
      category: 'Security',
      description: 'Identity and access management (OIDC/SSO)',
      icon: Key,
      configured: !!config.keycloak.url,
      url: config.keycloak.url,
      details: config.keycloak.url
        ? [
            { label: 'URL', value: config.keycloak.url },
            { label: 'Realm', value: config.keycloak.realm },
            { label: 'Admin User', value: config.keycloak.adminUser },
            { label: 'Admin Password', value: config.keycloak.adminPassword, masked: true },
          ]
        : undefined,
    },
    // Storage
    {
      name: 'Minio',
      category: 'Storage',
      description: 'S3-compatible object storage',
      icon: Database,
      configured: !!config.minio.url,
      url: config.minio.consoleURL || config.minio.url,
      details: config.minio.url
        ? [
            { label: 'API URL', value: config.minio.url },
            { label: 'Console URL', value: config.minio.consoleURL },
            { label: 'Access Key', value: config.minio.accessKey },
            { label: 'Secret Key', value: config.minio.secretKey, masked: true },
          ]
        : undefined,
    },
    // Observability
    {
      name: 'Prometheus',
      category: 'Observability',
      description: 'Metrics collection and monitoring',
      icon: Activity,
      configured: !!config.prometheus.url,
      url: config.prometheus.url,
      details: config.prometheus.url ? [{ label: 'URL', value: config.prometheus.url }] : undefined,
    },
    {
      name: 'Grafana',
      category: 'Observability',
      description: 'Metrics visualization and dashboards',
      icon: BarChart3,
      configured: !!config.grafana.url,
      url: config.grafana.url,
      details: config.grafana.url
        ? [
            { label: 'URL', value: config.grafana.url },
            { label: 'Username', value: config.grafana.username },
            { label: 'Password', value: config.grafana.password, masked: true },
          ]
        : undefined,
    },
    // Platform
    {
      name: 'Kubernetes Dashboard',
      category: 'Platform',
      description: 'Kubernetes cluster management UI',
      icon: Server,
      configured: !!config.kubernetesDashboard.url,
      url: config.kubernetesDashboard.url,
      details: config.kubernetesDashboard.url
        ? [{ label: 'URL', value: config.kubernetesDashboard.url }]
        : undefined,
    },
  ];

  // Group by category
  const categories = Array.from(new Set(integrations.map((i) => i.category)));

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Integrations</h1>
        <p className="text-muted-foreground">Platform service integrations and external systems</p>
      </div>

      {/* Summary Stats */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Total Integrations</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{integrations.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Configured</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600 dark:text-green-400">
              {integrations.filter((i) => i.configured).length}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Not Configured</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-muted-foreground">
              {integrations.filter((i) => !i.configured).length}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Integration Cards by Category */}
      {categories.map((category) => {
        const categoryIntegrations = integrations.filter((i) => i.category === category);

        return (
          <div key={category} className="space-y-4">
            <h2 className="text-xl font-semibold">{category}</h2>
            <div className="grid gap-4 md:grid-cols-2">
              {categoryIntegrations.map((integration) => {
                const Icon = integration.icon;

                return (
                  <Card key={integration.name} className="relative">
                    <CardHeader>
                      <div className="flex items-start justify-between">
                        <div className="flex items-center gap-3">
                          <div className="p-2 bg-muted rounded-lg">
                            <Icon className="h-5 w-5" />
                          </div>
                          <div>
                            <CardTitle className="text-base">{integration.name}</CardTitle>
                            <CardDescription className="text-sm mt-1">
                              {integration.description}
                            </CardDescription>
                          </div>
                        </div>
                        <Badge variant={integration.configured ? 'default' : 'secondary'}>
                          {integration.configured ? (
                            <div className="flex items-center gap-1">
                              <CheckCircle2 className="h-3 w-3" />
                              <span>Configured</span>
                            </div>
                          ) : (
                            <div className="flex items-center gap-1">
                              <XCircle className="h-3 w-3" />
                              <span>Not Configured</span>
                            </div>
                          )}
                        </Badge>
                      </div>
                    </CardHeader>

                    {integration.configured && integration.details && (
                      <CardContent className="space-y-3">
                        <div className="space-y-2">
                          {integration.details.map((detail, idx) => (
                            <div
                              key={idx}
                              className="grid grid-cols-[120px_1fr] gap-2 text-sm items-center"
                            >
                              <span className="text-muted-foreground font-medium">
                                {detail.label}:
                              </span>
                              {detail.masked ? (
                                <div className="flex items-center gap-2">
                                  <Lock className="h-3 w-3 text-muted-foreground" />
                                  <span className="font-mono text-muted-foreground">
                                    {detail.value}
                                  </span>
                                </div>
                              ) : (
                                <span className="font-mono text-sm truncate">{detail.value}</span>
                              )}
                            </div>
                          ))}
                        </div>

                        {integration.url && (
                          <div className="pt-3 border-t">
                            <Button
                              variant="outline"
                              size="sm"
                              className="w-full"
                              onClick={() => window.open(integration.url, '_blank')}
                            >
                              <ExternalLink className="h-4 w-4 mr-2" />
                              Open {integration.name}
                            </Button>
                          </div>
                        )}
                      </CardContent>
                    )}

                    {!integration.configured && (
                      <CardContent>
                        <p className="text-sm text-muted-foreground">
                          Configure this integration in{' '}
                          <code className="text-xs">admin-config.yaml</code>
                        </p>
                      </CardContent>
                    )}
                  </Card>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}
