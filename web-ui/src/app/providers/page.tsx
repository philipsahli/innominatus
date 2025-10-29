'use client';

import React, { useState, useEffect } from 'react';
import { ProtectedRoute } from '@/components/protected-route';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Package, RefreshCw, CheckCircle, XCircle } from 'lucide-react';
import { api, type ProviderSummary, type ProviderStats } from '@/lib/api';

export default function ProvidersPage() {
  const [providers, setProviders] = useState<ProviderSummary[]>([]);
  const [stats, setStats] = useState<ProviderStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchProviders = async () => {
    setLoading(true);
    setError(null);

    try {
      const [providersResponse, statsResponse] = await Promise.all([
        api.getProviders(),
        api.getProviderStats(),
      ]);

      if (providersResponse.success && providersResponse.data) {
        setProviders(providersResponse.data);
      } else {
        setError(providersResponse.error || 'Failed to fetch providers');
      }

      if (statsResponse.success && statsResponse.data) {
        setStats(statsResponse.data);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch providers');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProviders();
  }, []);

  return (
    <ProtectedRoute>
      <div className="container mx-auto py-8 px-4">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100 flex items-center gap-3">
              <Package className="w-8 h-8 text-purple-500" />
              Providers
            </h1>
            <p className="text-gray-600 dark:text-gray-400 mt-2">
              Extensible provisioners and golden paths from infrastructure and product teams
            </p>
          </div>
          <Button onClick={fetchProviders} variant="outline" disabled={loading}>
            <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
        </div>

        {/* Statistics Cards */}
        {stats && (
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium text-gray-600 dark:text-gray-400">
                  Total Providers
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                  {stats.providers}
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium text-gray-600 dark:text-gray-400">
                  Total Workflows
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-purple-600 dark:text-purple-400">
                  {providers.reduce((sum, p) => sum + p.provisioners + p.golden_paths, 0)}
                </div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  All workflows combined
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium text-gray-600 dark:text-gray-400">
                  Provisioners
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">
                  {providers.reduce((sum, p) => sum + p.provisioners, 0)}
                </div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  Single-resource workflows
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium text-gray-600 dark:text-gray-400">
                  Golden Paths
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-green-600 dark:text-green-400">
                  {providers.reduce((sum, p) => sum + p.golden_paths, 0)}
                </div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  Multi-resource orchestration
                </div>
              </CardContent>
            </Card>
          </div>
        )}

        {/* Providers Table */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Package className="w-5 h-5" />
              Loaded Providers
              <span className="text-sm font-normal text-gray-500">({providers.length})</span>
            </CardTitle>
            <CardDescription>Providers loaded from filesystem and Git repositories</CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="flex items-center justify-center py-8">
                <RefreshCw className="w-6 h-6 animate-spin text-gray-400" />
                <span className="ml-2 text-gray-600 dark:text-gray-400">Loading providers...</span>
              </div>
            ) : error ? (
              <div className="text-center py-8">
                <div className="flex items-center justify-center gap-2 text-red-600 dark:text-red-400 mb-4">
                  <XCircle className="w-6 h-6" />
                  <p>{error}</p>
                </div>
                <Button onClick={fetchProviders} variant="outline">
                  Try Again
                </Button>
              </div>
            ) : providers.length === 0 ? (
              <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                <Package className="w-12 h-12 mx-auto mb-4 text-gray-300 dark:text-gray-600" />
                <p className="text-lg font-medium">No providers loaded</p>
                <p className="text-sm mt-1">Configure providers in admin-config.yaml</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Version</TableHead>
                      <TableHead>Category</TableHead>
                      <TableHead>Workflows</TableHead>
                      <TableHead>Description</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {providers.map((provider) => {
                      const totalWorkflows = provider.provisioners + provider.golden_paths;
                      return (
                        <TableRow
                          key={provider.name}
                          className="hover:bg-gray-50 dark:hover:bg-gray-800"
                        >
                          <TableCell>
                            <div className="font-medium text-gray-900 dark:text-gray-100 flex items-center gap-2">
                              <Package className="w-4 h-4 text-purple-500" />
                              {provider.name}
                            </div>
                          </TableCell>
                          <TableCell>
                            <Badge variant="outline">v{provider.version}</Badge>
                          </TableCell>
                          <TableCell>
                            <Badge
                              variant="default"
                              className={
                                provider.category === 'infrastructure'
                                  ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-100'
                                  : 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-100'
                              }
                            >
                              {provider.category || 'unknown'}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <div className="flex flex-col gap-1">
                              <div className="text-sm font-semibold text-purple-600 dark:text-purple-400">
                                {totalWorkflows} total
                              </div>
                              <div className="text-xs text-gray-500 dark:text-gray-400 space-y-0.5">
                                <div>
                                  <Badge variant="secondary" className="text-xs mr-1">P</Badge>
                                  {provider.provisioners} provisioner{provider.provisioners !== 1 ? 's' : ''}
                                </div>
                                <div>
                                  <Badge variant="secondary" className="text-xs mr-1">GP</Badge>
                                  {provider.golden_paths} golden path{provider.golden_paths !== 1 ? 's' : ''}
                                </div>
                              </div>
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="text-sm text-gray-600 dark:text-gray-400">
                              {provider.description || 'No description'}
                            </div>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Information Card */}
        <Card className="mt-6">
          <CardHeader>
            <CardTitle className="text-base">About Providers & Workflows</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-gray-600 dark:text-gray-400 space-y-2">
            <p>
              <strong>Providers</strong> bundle workflows that teams can use to provision resources and orchestrate deployments.
              Each provider contains YAML-based workflows that define automated operations.
            </p>

            <p className="pt-2">
              <strong>Workflow Types:</strong>
            </p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>
                <strong>Provisioners (P):</strong> Single-resource workflows that create individual resources
                like databases, namespaces, or storage buckets
              </li>
              <li>
                <strong>Golden Paths (GP):</strong> Multi-resource orchestration workflows that combine
                multiple provisioners (e.g., onboard-dev-team, deploy-app)
              </li>
            </ul>

            <p className="pt-2">
              <strong>Provider Sources:</strong>
            </p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>
                <strong>Built-in Provider:</strong> Core workflows shipped with innominatus (filesystem)
              </li>
              <li>
                <strong>Extension Providers:</strong> Custom workflows from product/platform teams loaded from Git repositories
              </li>
            </ul>

            <p className="pt-2">
              <strong>Provider Categories:</strong>
            </p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>
                <strong>Infrastructure:</strong> Platform team providers (AWS, Azure, GCP, Kubernetes)
              </li>
              <li>
                <strong>Service:</strong> Product team providers (ecommerce, analytics, ML pipelines)
              </li>
            </ul>

            <p className="pt-2">
              Providers are configured in{' '}
              <code className="bg-gray-100 dark:bg-gray-800 px-1 rounded">admin-config.yaml</code>{' '}
              and loaded automatically on server startup.
            </p>
          </CardContent>
        </Card>
      </div>
    </ProtectedRoute>
  );
}
