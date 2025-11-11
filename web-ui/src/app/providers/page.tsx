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
import {
  Package,
  RefreshCw,
  CheckCircle,
  XCircle,
  ChevronDown,
  ChevronRight,
  Tag,
} from 'lucide-react';
import { api, type ProviderSummary, type ProviderStats } from '@/lib/api';

export default function ProvidersPage() {
  const [providers, setProviders] = useState<ProviderSummary[]>([]);
  const [stats, setStats] = useState<ProviderStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedProvider, setSelectedProvider] = useState<ProviderSummary | null>(null);

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

        {/* Providers Table and Detail Pane */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Providers Table */}
          <Card className={selectedProvider ? 'lg:col-span-2' : 'lg:col-span-3'}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Package className="w-5 h-5" />
                Loaded Providers
                <span className="text-sm font-normal text-gray-500">({providers.length})</span>
              </CardTitle>
              <CardDescription>
                Providers loaded from filesystem and Git repositories
              </CardDescription>
            </CardHeader>
            <CardContent>
              {loading ? (
                <div className="flex items-center justify-center py-8">
                  <RefreshCw className="w-6 h-6 animate-spin text-gray-400" />
                  <span className="ml-2 text-gray-600 dark:text-gray-400">
                    Loading providers...
                  </span>
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
                            onClick={() => setSelectedProvider(provider)}
                            className="hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer"
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
                                    <Badge variant="secondary" className="text-xs mr-1">
                                      P
                                    </Badge>
                                    {provider.provisioners} provisioner
                                    {provider.provisioners !== 1 ? 's' : ''}
                                  </div>
                                  <div>
                                    <Badge variant="secondary" className="text-xs mr-1">
                                      GP
                                    </Badge>
                                    {provider.golden_paths} golden path
                                    {provider.golden_paths !== 1 ? 's' : ''}
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

          {/* Provider Detail Pane */}
          {selectedProvider && (
            <Card className="lg:col-span-1">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="flex items-center gap-2">
                    <Package className="w-5 h-5 text-purple-500" />
                    {selectedProvider.name}
                  </CardTitle>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setSelectedProvider(null)}
                    className="h-8 w-8 p-0"
                  >
                    <XCircle className="w-4 h-4" />
                  </Button>
                </div>
                <CardDescription>Provider Details</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Version */}
                <div>
                  <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">
                    Version
                  </h3>
                  <Badge variant="outline">v{selectedProvider.version}</Badge>
                </div>

                {/* Category */}
                <div>
                  <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">
                    Category
                  </h3>
                  <Badge
                    variant="default"
                    className={
                      selectedProvider.category === 'infrastructure'
                        ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-100'
                        : 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-100'
                    }
                  >
                    {selectedProvider.category || 'unknown'}
                  </Badge>
                </div>

                {/* Description */}
                <div>
                  <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">
                    Description
                  </h3>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    {selectedProvider.description || 'No description available'}
                  </p>
                </div>

                {/* Workflow Statistics */}
                <div>
                  <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                    Workflows
                  </h3>
                  <div className="space-y-2">
                    <div className="flex items-center justify-between p-2 bg-purple-50 dark:bg-purple-900/20 rounded">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        Total Workflows
                      </span>
                      <span className="text-sm font-bold text-purple-600 dark:text-purple-400">
                        {selectedProvider.workflows?.length || 0}
                      </span>
                    </div>
                    <div className="flex items-center justify-between p-2 bg-blue-50 dark:bg-blue-900/20 rounded">
                      <span className="text-sm text-gray-700 dark:text-gray-300">Provisioners</span>
                      <span className="text-sm font-bold text-blue-600 dark:text-blue-400">
                        {selectedProvider.workflows?.filter((w) => w.category === 'provisioner')
                          .length || 0}
                      </span>
                    </div>
                    <div className="flex items-center justify-between p-2 bg-green-50 dark:bg-green-900/20 rounded">
                      <span className="text-sm text-gray-700 dark:text-gray-300">Golden Paths</span>
                      <span className="text-sm font-bold text-green-600 dark:text-green-400">
                        {selectedProvider.workflows?.filter((w) => w.category === 'goldenpath')
                          .length || 0}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Workflow Details */}
                {selectedProvider.workflows && selectedProvider.workflows.length > 0 && (
                  <div className="border-t pt-4">
                    <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">
                      Available Workflows
                    </h3>

                    {/* Provisioners */}
                    {selectedProvider.workflows.filter((w) => w.category === 'provisioner').length >
                      0 && (
                      <div className="mb-4">
                        <h4 className="text-xs font-semibold text-blue-600 dark:text-blue-400 mb-2 flex items-center gap-1">
                          <Package className="w-3 h-3" />
                          Provisioners (
                          {
                            selectedProvider.workflows.filter((w) => w.category === 'provisioner')
                              .length
                          }
                          )
                        </h4>
                        <div className="space-y-2">
                          {selectedProvider.workflows
                            .filter((w) => w.category === 'provisioner')
                            .map((workflow, idx) => (
                              <div
                                key={idx}
                                className="p-2 bg-gray-50 dark:bg-gray-800/50 rounded text-xs"
                              >
                                <div className="font-medium text-gray-900 dark:text-gray-100">
                                  {workflow.name}
                                </div>
                                {workflow.description && (
                                  <div className="text-gray-600 dark:text-gray-400 mt-1">
                                    {workflow.description}
                                  </div>
                                )}
                                {workflow.tags && workflow.tags.length > 0 && (
                                  <div className="flex flex-wrap gap-1 mt-2">
                                    {workflow.tags.map((tag, tagIdx) => (
                                      <span
                                        key={tagIdx}
                                        className="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded text-xs"
                                      >
                                        <Tag className="w-2.5 h-2.5" />
                                        {tag}
                                      </span>
                                    ))}
                                  </div>
                                )}
                              </div>
                            ))}
                        </div>
                      </div>
                    )}

                    {/* Golden Paths */}
                    {selectedProvider.workflows.filter((w) => w.category === 'goldenpath').length >
                      0 && (
                      <div>
                        <h4 className="text-xs font-semibold text-green-600 dark:text-green-400 mb-2 flex items-center gap-1">
                          <Package className="w-3 h-3" />
                          Golden Paths (
                          {
                            selectedProvider.workflows.filter((w) => w.category === 'goldenpath')
                              .length
                          }
                          )
                        </h4>
                        <div className="space-y-2">
                          {selectedProvider.workflows
                            .filter((w) => w.category === 'goldenpath')
                            .map((workflow, idx) => (
                              <div
                                key={idx}
                                className="p-2 bg-gray-50 dark:bg-gray-800/50 rounded text-xs"
                              >
                                <div className="font-medium text-gray-900 dark:text-gray-100">
                                  {workflow.name}
                                </div>
                                {workflow.description && (
                                  <div className="text-gray-600 dark:text-gray-400 mt-1">
                                    {workflow.description}
                                  </div>
                                )}
                                {workflow.tags && workflow.tags.length > 0 && (
                                  <div className="flex flex-wrap gap-1 mt-2">
                                    {workflow.tags.map((tag, tagIdx) => (
                                      <span
                                        key={tagIdx}
                                        className="inline-flex items-center gap-1 px-2 py-0.5 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 rounded text-xs"
                                      >
                                        <Tag className="w-2.5 h-2.5" />
                                        {tag}
                                      </span>
                                    ))}
                                  </div>
                                )}
                              </div>
                            ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>

        {/* Information Card */}
        <Card className="mt-6">
          <CardHeader>
            <CardTitle className="text-base">About Providers & Workflows</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-gray-600 dark:text-gray-400 space-y-2">
            <p>
              <strong>Providers</strong> bundle workflows that teams can use to provision resources
              and orchestrate deployments. Each provider contains YAML-based workflows that define
              automated operations.
            </p>

            <p className="pt-2">
              <strong>Workflow Types:</strong>
            </p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>
                <strong>Provisioners (P):</strong> Single-resource workflows that create individual
                resources like databases, namespaces, or storage buckets
              </li>
              <li>
                <strong>Golden Paths (GP):</strong> Multi-resource orchestration workflows that
                combine multiple provisioners (e.g., onboard-dev-team, deploy-app)
              </li>
            </ul>

            <p className="pt-2">
              <strong>Provider Sources:</strong>
            </p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>
                <strong>Built-in Provider:</strong> Core workflows shipped with innominatus
                (filesystem)
              </li>
              <li>
                <strong>Extension Providers:</strong> Custom workflows from product/platform teams
                loaded from Git repositories
              </li>
            </ul>

            <p className="pt-2">
              <strong>Provider Categories:</strong>
            </p>
            <ul className="list-disc list-inside space-y-1 ml-2">
              <li>
                <strong>Infrastructure:</strong> Platform team providers (AWS, Azure, GCP,
                Kubernetes)
              </li>
              <li>
                <strong>Service:</strong> Product team providers (ecommerce, analytics, ML
                pipelines)
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
