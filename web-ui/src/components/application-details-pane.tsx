'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  X,
  Package,
  Clock,
  ExternalLink,
  Database,
  Activity,
  FileCode,
  Trash2,
  Archive
} from 'lucide-react';
import type { Application } from '@/lib/api';
import { api } from '@/lib/api';
import { useDeprovisionApplication, useDeleteApplication } from '@/hooks/use-api';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { useToast } from '@/hooks/use-toast';

interface ApplicationDetailsPaneProps {
  application: Application | null;
  onClose: () => void;
}

export function ApplicationDetailsPane({ application, onClose }: ApplicationDetailsPaneProps) {
  const router = useRouter();
  const { toast } = useToast();
  const [spec, setSpec] = useState<any>(null);
  const [workflows, setWorkflows] = useState<any[]>([]);
  const [resources, setResources] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [showDeprovisionDialog, setShowDeprovisionDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const { mutate: deprovision, loading: deprovisionLoading } = useDeprovisionApplication();
  const { mutate: deleteApp, loading: deleteLoading } = useDeleteApplication();

  useEffect(() => {
    if (application) {
      loadDetails();
    }
  }, [application]);

  const loadDetails = async () => {
    if (!application) return;

    setLoading(true);
    try {
      // Load spec data
      const specsResponse = await api.getSpecs();
      if (specsResponse.success && specsResponse.data) {
        setSpec(specsResponse.data[application.name]);
      }

      // Load workflows for this app
      const workflowsResponse = await api.getWorkflows(application.name);
      if (workflowsResponse.success && workflowsResponse.data) {
        setWorkflows(workflowsResponse.data.data.slice(0, 10));
      }

      // Load resources for this app
      const resourcesResponse = await api.getResources(application.name);
      if (resourcesResponse.success && resourcesResponse.data) {
        setResources(Object.values(resourcesResponse.data).flat());
      }
    } catch (error) {
      console.error('Failed to load application details:', error);
    } finally {
      setLoading(false);
    }
  };

  if (!application) {
    return null;
  }

  const getStatusBadgeVariant = (
    status: string
  ): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch (status) {
      case 'running':
        return 'default';
      case 'pending':
        return 'secondary';
      case 'failed':
        return 'destructive';
      default:
        return 'outline';
    }
  };

  const handleDeprovision = async () => {
    if (!application) return;

    const result = await deprovision(application.name);
    if (result.success) {
      toast({
        title: 'Application Deprovisioned',
        description: `Infrastructure for ${application.name} has been deprovisioned. Audit trail preserved.`,
      });
      onClose();
      router.refresh();
    } else {
      toast({
        title: 'Deprovision Failed',
        description: result.error || 'Failed to deprovision application',
        variant: 'destructive',
      });
    }
    setShowDeprovisionDialog(false);
  };

  const handleDelete = async () => {
    if (!application) return;

    const result = await deleteApp(application.name);
    if (result.success) {
      toast({
        title: 'Application Deleted',
        description: `${application.name} has been completely removed.`,
      });
      onClose();
      router.refresh();
    } else {
      toast({
        title: 'Delete Failed',
        description: result.error || 'Failed to delete application',
        variant: 'destructive',
      });
    }
    setShowDeleteDialog(false);
  };

  return (
    <div className="w-full h-full flex flex-col bg-white dark:bg-gray-800 border-l">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <Package className="w-5 h-5 text-blue-500" />
            <h3 className="text-lg font-semibold">{application.name}</h3>
            <Badge variant={getStatusBadgeVariant(application.status)}>{application.status}</Badge>
          </div>
          <p className="text-sm text-muted-foreground">Environment: {application.environment}</p>
          <Link
            href={`/graph/${application.name}`}
            className="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 flex items-center gap-1 w-fit mt-1"
          >
            View Dependency Graph
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
            <TabsTrigger value="spec">Score Spec</TabsTrigger>
            <TabsTrigger value="workflows">Workflows</TabsTrigger>
            <TabsTrigger value="resources">Resources</TabsTrigger>
          </TabsList>

          {/* Overview Tab */}
          <TabsContent value="overview" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Package className="w-4 h-4" />
                  Application Information
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="grid grid-cols-2 gap-2 text-sm">
                  <div className="text-muted-foreground">Name:</div>
                  <div className="font-medium">{application.name}</div>

                  <div className="text-muted-foreground">Status:</div>
                  <div>
                    <Badge variant={getStatusBadgeVariant(application.status)}>
                      {application.status}
                    </Badge>
                  </div>

                  <div className="text-muted-foreground">Environment:</div>
                  <div>{application.environment}</div>

                  <div className="text-muted-foreground">Resources:</div>
                  <div>{application.resources} configured</div>

                  <div className="text-muted-foreground">Last Updated:</div>
                  <div className="text-xs">
                    {new Date(application.lastUpdated).toLocaleString()}
                  </div>
                </div>

                <div className="pt-3 border-t space-y-2">
                  <div className="flex gap-2">
                    <Link href={`/graph/${application.name}`} className="flex-1">
                      <Button variant="outline" className="w-full">
                        <ExternalLink className="w-4 h-4 mr-2" />
                        View Graph
                      </Button>
                    </Link>
                    <Link href={`/workflows?app=${application.name}`} className="flex-1">
                      <Button variant="outline" className="w-full">
                        <Activity className="w-4 h-4 mr-2" />
                        View Workflows
                      </Button>
                    </Link>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      className="flex-1"
                      onClick={() => setShowDeprovisionDialog(true)}
                      disabled={deprovisionLoading}
                    >
                      <Archive className="w-4 h-4 mr-2" />
                      Deprovision
                    </Button>
                    <Button
                      variant="destructive"
                      className="flex-1"
                      onClick={() => setShowDeleteDialog(true)}
                      disabled={deleteLoading}
                    >
                      <Trash2 className="w-4 h-4 mr-2" />
                      Delete
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* Score Spec Tab */}
          <TabsContent value="spec" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <FileCode className="w-4 h-4" />
                  Score Specification (YAML)
                </CardTitle>
              </CardHeader>
              <CardContent>
                {loading ? (
                  <p className="text-sm text-muted-foreground">Loading spec...</p>
                ) : spec ? (
                  <pre className="bg-gray-100 dark:bg-gray-900 p-3 rounded text-xs overflow-x-auto max-h-96 overflow-y-auto">
                    <code>{JSON.stringify(spec, null, 2)}</code>
                  </pre>
                ) : (
                  <p className="text-sm text-muted-foreground">No spec data available</p>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          {/* Workflows Tab */}
          <TabsContent value="workflows" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Activity className="w-4 h-4" />
                  Recent Workflows (Last 10)
                </CardTitle>
              </CardHeader>
              <CardContent>
                {loading ? (
                  <p className="text-sm text-muted-foreground">Loading workflows...</p>
                ) : workflows.length > 0 ? (
                  <div className="space-y-2">
                    {workflows.map((workflow) => (
                      <Link
                        key={workflow.id}
                        href={`/workflows?id=${workflow.id}`}
                        className="flex items-center justify-between p-3 rounded-lg border hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                      >
                        <div className="flex-1">
                          <p className="text-sm font-medium">{workflow.name}</p>
                          <p className="text-xs text-muted-foreground">{workflow.timestamp}</p>
                        </div>
                        <Badge
                          variant={
                            workflow.status === 'completed'
                              ? 'default'
                              : workflow.status === 'failed'
                                ? 'destructive'
                                : 'secondary'
                          }
                          className="text-xs"
                        >
                          {workflow.status}
                        </Badge>
                      </Link>
                    ))}
                    <Link href={`/workflows?app=${application.name}`} className="block pt-2">
                      <Button variant="outline" className="w-full">
                        View All Workflows
                      </Button>
                    </Link>
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No workflows found</p>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          {/* Resources Tab */}
          <TabsContent value="resources" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Database className="w-4 h-4" />
                  Associated Resources ({resources.length})
                </CardTitle>
              </CardHeader>
              <CardContent>
                {loading ? (
                  <p className="text-sm text-muted-foreground">Loading resources...</p>
                ) : resources.length > 0 ? (
                  <div className="space-y-2">
                    {resources.map((resource) => (
                      <Link
                        key={resource.id}
                        href={`/resources?resourceId=${resource.id}`}
                        className="flex items-center justify-between p-3 rounded-lg border hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                      >
                        <div className="flex items-center gap-3 flex-1">
                          <Database className="w-4 h-4 text-gray-500" />
                          <div className="flex-1">
                            <p className="text-sm font-medium">{resource.resource_name}</p>
                            <p className="text-xs text-muted-foreground">
                              {resource.resource_type}
                            </p>
                          </div>
                        </div>
                        <Badge variant="outline" className="text-xs">
                          {resource.state}
                        </Badge>
                      </Link>
                    ))}
                    <Link href={`/resources?app=${application.name}`} className="block pt-2">
                      <Button variant="outline" className="w-full">
                        View All Resources
                      </Button>
                    </Link>
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No resources found</p>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>

      {/* Deprovision Confirmation Dialog */}
      <AlertDialog open={showDeprovisionDialog} onOpenChange={setShowDeprovisionDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Deprovision Application</AlertDialogTitle>
            <AlertDialogDescription className="space-y-2">
              <p>
                Are you sure you want to deprovision <strong>{application.name}</strong>?
              </p>
              <p>
                This will remove all provisioned infrastructure (databases, storage, etc.) but will
                preserve the application record and audit trail for compliance.
              </p>
              <p className="text-amber-600 dark:text-amber-400 font-medium">
                ⚠️ Infrastructure resources will be deleted. Data may be lost unless backed up.
              </p>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleDeprovision} disabled={deprovisionLoading}>
              {deprovisionLoading ? 'Deprovisioning...' : 'Deprovision'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Application</AlertDialogTitle>
            <AlertDialogDescription className="space-y-2">
              <p>
                Are you sure you want to <strong>permanently delete</strong>{' '}
                <strong>{application.name}</strong>?
              </p>
              <p>
                This will remove all infrastructure, database records, and audit trail. This action
                cannot be undone.
              </p>
              <p className="text-red-600 dark:text-red-400 font-medium">
                ⚠️ DANGER: This is a permanent, destructive operation.
              </p>
              <p className="text-sm text-muted-foreground">
                Tip: Use &ldquo;Deprovision&rdquo; instead if you want to keep audit records for compliance.
              </p>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={deleteLoading}
              className="bg-red-600 hover:bg-red-700"
            >
              {deleteLoading ? 'Deleting...' : 'Delete Permanently'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
