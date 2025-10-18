'use client';

import React from 'react';
import Link from 'next/link';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { X, Database, Server, Clock, ExternalLink, FileCode, Settings } from 'lucide-react';
import { ResourceInstance } from '@/lib/api';

interface ResourceDetailsPaneProps {
  resource: ResourceInstance | null;
  onClose: () => void;
}

export function ResourceDetailsPane({ resource, onClose }: ResourceDetailsPaneProps) {
  if (!resource) {
    return null;
  }

  const getStateBadgeVariant = (
    state: string
  ): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch (state) {
      case 'active':
        return 'default';
      case 'provisioning':
      case 'scaling':
      case 'updating':
        return 'secondary';
      case 'failed':
      case 'degraded':
        return 'destructive';
      default:
        return 'outline';
    }
  };

  const getHealthColor = (health: string): string => {
    switch (health.toLowerCase()) {
      case 'healthy':
      case 'ok':
        return 'text-green-600 dark:text-green-400';
      case 'degraded':
      case 'warning':
        return 'text-yellow-600 dark:text-yellow-400';
      case 'unhealthy':
      case 'critical':
        return 'text-red-600 dark:text-red-400';
      default:
        return 'text-gray-600 dark:text-gray-400';
    }
  };

  return (
    <div className="w-full h-full flex flex-col bg-white dark:bg-gray-800 border-l">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <Database className="w-5 h-5 text-blue-500" />
            <h3 className="text-lg font-semibold">{resource.resource_name}</h3>
            <Badge variant={getStateBadgeVariant(resource.state)}>{resource.state}</Badge>
          </div>
          <p className="text-sm text-muted-foreground">Type: {resource.resource_type}</p>
          <Link
            href={`/graph/${resource.application_name}`}
            className="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 flex items-center gap-1 w-fit mt-1"
          >
            Application: {resource.application_name}
            <ExternalLink className="w-3 h-3" />
          </Link>
        </div>
        <Button variant="ghost" size="sm" onClick={onClose}>
          <X className="w-4 h-4" />
        </Button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-4">
        <Tabs defaultValue="overview" className="w-full">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="configuration">Config</TabsTrigger>
            <TabsTrigger value="provider">Provider</TabsTrigger>
            <TabsTrigger value="history">History</TabsTrigger>
          </TabsList>

          {/* Overview Tab */}
          <TabsContent value="overview" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Database className="w-4 h-4" />
                  Resource Information
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="grid grid-cols-2 gap-2 text-sm">
                  <div className="text-muted-foreground">ID:</div>
                  <div className="font-mono">{resource.id}</div>

                  <div className="text-muted-foreground">Name:</div>
                  <div>{resource.resource_name}</div>

                  <div className="text-muted-foreground">Type:</div>
                  <div>{resource.resource_type}</div>

                  <div className="text-muted-foreground">Application:</div>
                  <div>
                    <Link
                      href={`/graph/${resource.application_name}`}
                      className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 flex items-center gap-1"
                    >
                      {resource.application_name}
                      <ExternalLink className="w-3 h-3" />
                    </Link>
                  </div>

                  <div className="text-muted-foreground">State:</div>
                  <div>
                    <Badge variant={getStateBadgeVariant(resource.state)}>{resource.state}</Badge>
                  </div>

                  <div className="text-muted-foreground">Health Status:</div>
                  <div className={getHealthColor(resource.health_status)}>
                    {resource.health_status || 'Unknown'}
                  </div>

                  {resource.workflow_execution_id && (
                    <>
                      <div className="text-muted-foreground">Workflow ID:</div>
                      <div>
                        <Link
                          href={`/workflows?id=${resource.workflow_execution_id}`}
                          className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 font-mono"
                        >
                          #{resource.workflow_execution_id}
                        </Link>
                      </div>
                    </>
                  )}

                  {resource.provider_id && (
                    <>
                      <div className="text-muted-foreground">Provider ID:</div>
                      <div className="font-mono text-xs break-all">{resource.provider_id}</div>
                    </>
                  )}
                </div>

                {resource.error_message && (
                  <div className="mt-3 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded">
                    <p className="text-sm font-medium text-red-800 dark:text-red-200 mb-1">
                      Error Message
                    </p>
                    <p className="text-sm text-red-600 dark:text-red-300">
                      {resource.error_message}
                    </p>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          {/* Configuration Tab */}
          <TabsContent value="configuration" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Settings className="w-4 h-4" />
                  Resource Configuration
                </CardTitle>
              </CardHeader>
              <CardContent>
                {resource.configuration && Object.keys(resource.configuration).length > 0 ? (
                  <pre className="bg-gray-100 dark:bg-gray-900 p-3 rounded text-xs overflow-x-auto">
                    <code>{JSON.stringify(resource.configuration, null, 2)}</code>
                  </pre>
                ) : (
                  <p className="text-sm text-muted-foreground">No configuration data available</p>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          {/* Provider Tab */}
          <TabsContent value="provider" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Server className="w-4 h-4" />
                  Provider Information
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {resource.provider_id && (
                  <div>
                    <p className="text-sm font-medium mb-1">Provider ID</p>
                    <p className="text-xs font-mono bg-gray-100 dark:bg-gray-900 p-2 rounded break-all">
                      {resource.provider_id}
                    </p>
                  </div>
                )}

                {resource.provider_metadata &&
                Object.keys(resource.provider_metadata).length > 0 ? (
                  <div>
                    <p className="text-sm font-medium mb-2">Provider Metadata</p>
                    <pre className="bg-gray-100 dark:bg-gray-900 p-3 rounded text-xs overflow-x-auto">
                      <code>{JSON.stringify(resource.provider_metadata, null, 2)}</code>
                    </pre>
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No provider metadata available</p>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          {/* History Tab */}
          <TabsContent value="history" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Clock className="w-4 h-4" />
                  Timeline
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="space-y-3">
                  <div className="flex items-start gap-3 pb-3 border-b">
                    <div className="w-2 h-2 rounded-full bg-green-500 mt-1.5" />
                    <div className="flex-1">
                      <p className="text-sm font-medium">Created</p>
                      <p className="text-xs text-muted-foreground">
                        {new Date(resource.created_at).toLocaleString()}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-start gap-3 pb-3 border-b">
                    <div className="w-2 h-2 rounded-full bg-blue-500 mt-1.5" />
                    <div className="flex-1">
                      <p className="text-sm font-medium">Last Updated</p>
                      <p className="text-xs text-muted-foreground">
                        {new Date(resource.updated_at).toLocaleString()}
                      </p>
                    </div>
                  </div>

                  {resource.last_health_check && (
                    <div className="flex items-start gap-3 pb-3 border-b">
                      <div
                        className={`w-2 h-2 rounded-full mt-1.5 ${
                          resource.health_status.toLowerCase() === 'healthy' ||
                          resource.health_status.toLowerCase() === 'ok'
                            ? 'bg-green-500'
                            : 'bg-yellow-500'
                        }`}
                      />
                      <div className="flex-1">
                        <p className="text-sm font-medium">Last Health Check</p>
                        <p className="text-xs text-muted-foreground">
                          {new Date(resource.last_health_check).toLocaleString()}
                        </p>
                      </div>
                    </div>
                  )}

                  {resource.workflow_execution_id && (
                    <div className="flex items-start gap-3">
                      <div className="w-2 h-2 rounded-full bg-purple-500 mt-1.5" />
                      <div className="flex-1">
                        <p className="text-sm font-medium">Related Workflow</p>
                        <Link
                          href={`/workflows?id=${resource.workflow_execution_id}`}
                          className="text-xs text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 flex items-center gap-1"
                        >
                          Workflow #{resource.workflow_execution_id}
                          <ExternalLink className="w-3 h-3" />
                        </Link>
                      </div>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
