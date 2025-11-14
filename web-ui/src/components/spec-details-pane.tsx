'use client';

import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  X,
  FileCode,
  Package,
  Users,
  Clock,
  Copy,
  Check,
  Database,
  ChevronDown,
  ChevronRight,
  CheckCircle2,
  AlertCircle,
  Activity,
  GitBranch,
} from 'lucide-react';
import { GraphNode, GraphEdge } from '@/lib/api';
import { formatAsYAML } from '@/lib/formatters';

// ============================================================================
// Types & Utilities
// ============================================================================

interface SpecDetailsPaneProps {
  spec: GraphNode | null;
  edges: GraphEdge[];
  allNodes: GraphNode[];
  onClose: () => void;
  onNavigateToNode?: (nodeId: string) => void;
}

type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

const STATUS_CONFIG = {
  active: {
    badgeVariant: 'default' as BadgeVariant,
    icon: <CheckCircle2 className="w-4 h-4 text-green-500" />,
    color: 'text-green-600 dark:text-green-400',
  },
  succeeded: {
    badgeVariant: 'default' as BadgeVariant,
    icon: <CheckCircle2 className="w-4 h-4 text-green-500" />,
    color: 'text-green-600 dark:text-green-400',
  },
  completed: {
    badgeVariant: 'default' as BadgeVariant,
    icon: <CheckCircle2 className="w-4 h-4 text-green-500" />,
    color: 'text-green-600 dark:text-green-400',
  },
  pending: {
    badgeVariant: 'secondary' as BadgeVariant,
    icon: <Clock className="w-4 h-4 text-gray-400" />,
    color: 'text-gray-600 dark:text-gray-400',
  },
  provisioning: {
    badgeVariant: 'secondary' as BadgeVariant,
    icon: <Activity className="w-4 h-4 text-blue-500" />,
    color: 'text-blue-600 dark:text-blue-400',
  },
  failed: {
    badgeVariant: 'destructive' as BadgeVariant,
    icon: <AlertCircle className="w-4 h-4 text-red-500" />,
    color: 'text-red-600 dark:text-red-400',
  },
  error: {
    badgeVariant: 'destructive' as BadgeVariant,
    icon: <AlertCircle className="w-4 h-4 text-red-500" />,
    color: 'text-red-600 dark:text-red-400',
  },
} as const;

const getStatusConfig = (status: string) => {
  return (
    STATUS_CONFIG[status?.toLowerCase() as keyof typeof STATUS_CONFIG] || STATUS_CONFIG.pending
  );
};

// ============================================================================
// Sub-Components
// ============================================================================

interface SpecHeaderProps {
  spec: GraphNode;
  team: string;
  onClose: () => void;
}

function SpecHeader({ spec, team, onClose }: SpecHeaderProps) {
  const statusConfig = getStatusConfig(spec.status);

  return (
    <div className="flex items-center justify-between p-4 border-b">
      <div className="flex-1">
        <div className="flex items-center gap-2 mb-2">
          <Package className="w-5 h-5 text-purple-500" />
          <h3 className="text-lg font-semibold">{spec.name}</h3>
          <Badge variant={statusConfig.badgeVariant}>{spec.status}</Badge>
        </div>
        <div className="text-sm text-muted-foreground space-y-1">
          <div>Application Specification</div>
          {team !== 'Unknown' && (
            <div className="flex items-center gap-1">
              <Users className="w-3 h-3" />
              Team: {team}
            </div>
          )}
        </div>
      </div>
      <Button variant="ghost" size="sm" onClick={onClose}>
        <X className="w-4 h-4" />
      </Button>
    </div>
  );
}

interface SpecOverviewCardProps {
  spec: GraphNode;
  team: string;
  createdBy: string;
  environment: string;
  labels: Record<string, any>;
  resourceCount: number;
  workflowCount: number;
}

function SpecOverviewCard({
  spec,
  team,
  createdBy,
  environment,
  labels,
  resourceCount,
  workflowCount,
}: SpecOverviewCardProps) {
  const statusConfig = getStatusConfig(spec.status);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm flex items-center gap-2">
          <FileCode className="w-4 h-4" />
          Application Information
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div className="text-muted-foreground">Name</div>
          <div className="font-medium">{spec.name}</div>

          <div className="text-muted-foreground">Status</div>
          <div className="flex items-center gap-2">
            {statusConfig.icon}
            <Badge variant={statusConfig.badgeVariant}>{spec.status}</Badge>
          </div>

          {team !== 'Unknown' && (
            <>
              <div className="text-muted-foreground">Team</div>
              <div className="flex items-center gap-1">
                <Users className="w-3 h-3" />
                {team}
              </div>
            </>
          )}

          {environment !== 'Unknown' && (
            <>
              <div className="text-muted-foreground">Environment</div>
              <div>{environment}</div>
            </>
          )}

          {createdBy !== 'Unknown' && (
            <>
              <div className="text-muted-foreground">Created By</div>
              <div>{createdBy}</div>
            </>
          )}

          {spec.created_at && (
            <>
              <div className="text-muted-foreground">Created</div>
              <div className="flex items-center gap-1">
                <Clock className="w-3 h-3" />
                {new Date(spec.created_at).toLocaleString()}
              </div>
            </>
          )}

          {spec.updated_at && (
            <>
              <div className="text-muted-foreground">Updated</div>
              <div className="flex items-center gap-1">
                <Clock className="w-3 h-3" />
                {new Date(spec.updated_at).toLocaleString()}
              </div>
            </>
          )}

          <div className="text-muted-foreground">Resources</div>
          <div className="flex items-center gap-1">
            <Database className="w-3 h-3" />
            {resourceCount} provisioned
          </div>

          <div className="text-muted-foreground">Workflows</div>
          <div className="flex items-center gap-1">
            <GitBranch className="w-3 h-3" />
            {workflowCount} executions
          </div>
        </div>

        {Object.keys(labels).length > 0 && (
          <div className="mt-4 pt-3 border-t">
            <div className="text-sm text-muted-foreground mb-2">Labels</div>
            <div className="flex flex-wrap gap-2">
              {Object.entries(labels).map(([key, value]) => (
                <Badge key={key} variant="outline" className="text-xs">
                  {key}: {String(value)}
                </Badge>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

interface WorkflowsCardProps {
  workflows: GraphNode[];
  isExpanded: boolean;
  onToggle: () => void;
  onNavigateToNode?: (nodeId: string) => void;
}

function WorkflowsCard({ workflows, isExpanded, onToggle, onNavigateToNode }: WorkflowsCardProps) {
  if (workflows.length === 0) return null;

  return (
    <Card>
      <CardHeader
        className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700"
        onClick={onToggle}
      >
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm flex items-center gap-2">
            <GitBranch className="w-4 h-4" />
            Associated Workflows ({workflows.length})
          </CardTitle>
          {isExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
        </div>
      </CardHeader>
      {isExpanded && (
        <CardContent className="space-y-2">
          {workflows.map((workflow) => {
            const statusConfig = getStatusConfig(workflow.status);
            return (
              <div
                key={workflow.id}
                className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-900 rounded hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer transition-colors"
                onClick={() => onNavigateToNode?.(workflow.id)}
              >
                <div className="flex items-center gap-2">
                  {statusConfig.icon}
                  <span className="text-sm font-medium">{workflow.name}</span>
                </div>
                <Badge variant={statusConfig.badgeVariant} className="text-xs">
                  {workflow.status}
                </Badge>
              </div>
            );
          })}
        </CardContent>
      )}
    </Card>
  );
}

interface ResourceCardProps {
  resource: GraphNode;
  onNavigateToNode?: (nodeId: string) => void;
}

function ResourceCard({ resource, onNavigateToNode }: ResourceCardProps) {
  const statusConfig = getStatusConfig(resource.status);

  return (
    <Card
      className="cursor-pointer hover:shadow-md transition-shadow"
      onClick={() => onNavigateToNode?.(resource.id)}
    >
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Database className="w-4 h-4 text-blue-500" />
            <CardTitle className="text-sm font-medium">{resource.name}</CardTitle>
          </div>
          <Badge variant={statusConfig.badgeVariant}>{resource.status}</Badge>
        </div>
      </CardHeader>
      <CardContent>
        <div className="text-xs text-muted-foreground space-y-1">
          {resource.metadata?.resource_type && (
            <div className="flex items-center gap-1">
              <span>Type:</span>
              <span className="font-medium">{resource.metadata.resource_type}</span>
            </div>
          )}
          {resource.metadata?.provider_id && (
            <div className="flex items-center gap-1">
              <span>Provider:</span>
              <span className="font-medium">{resource.metadata.provider_id}</span>
            </div>
          )}
          {resource.created_at && (
            <div className="flex items-center gap-1">
              <Clock className="w-3 h-3" />
              {new Date(resource.created_at).toLocaleString()}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

interface SpecContentCardProps {
  specContent: string | null;
  onCopy: () => void;
  isCopied: boolean;
}

function SpecContentCard({ specContent, onCopy, isCopied }: SpecContentCardProps) {
  if (!specContent) {
    return (
      <p className="text-sm text-muted-foreground text-center py-8">
        No specification content available
      </p>
    );
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm flex items-center gap-2">
            <FileCode className="w-4 h-4" />
            Score Specification
          </CardTitle>
          <Button variant="outline" size="sm" onClick={onCopy}>
            {isCopied ? (
              <>
                <Check className="w-4 h-4 mr-1 text-green-500" />
                Copied!
              </>
            ) : (
              <>
                <Copy className="w-4 h-4 mr-1" />
                Copy
              </>
            )}
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <pre className="bg-gray-900 text-green-400 p-3 rounded text-xs overflow-x-auto max-h-96 overflow-y-auto">
          <code>{specContent}</code>
        </pre>
      </CardContent>
    </Card>
  );
}

// ============================================================================
// Main Component
// ============================================================================

export function SpecDetailsPane({
  spec,
  edges,
  allNodes,
  onClose,
  onNavigateToNode,
}: SpecDetailsPaneProps) {
  const [copiedContent, setCopiedContent] = useState(false);
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['workflows']));

  if (!spec) {
    return null;
  }

  const toggleSection = (section: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(section)) {
        next.delete(section);
      } else {
        next.add(section);
      }
      return next;
    });
  };

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedContent(true);
      setTimeout(() => setCopiedContent(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  // Find associated resources and workflows
  const associatedResources = edges
    .filter((edge) => edge.source_id === spec.id && edge.relationship === 'contains')
    .map((edge) => allNodes.find((node) => node.id === edge.target_id))
    .filter((node): node is GraphNode => node?.type === 'resource');

  const associatedWorkflows = edges
    .filter((edge) => edge.source_id === spec.id || edge.target_id === spec.id)
    .map((edge) =>
      edge.source_id === spec.id
        ? allNodes.find((node) => node.id === edge.target_id)
        : allNodes.find((node) => node.id === edge.source_id)
    )
    .filter((node): node is GraphNode => node?.type === 'workflow');

  // Extract metadata
  const specContent = spec.metadata?.spec_content || spec.metadata?.score_spec || null;
  const labels = spec.metadata?.labels || {};
  const team = spec.metadata?.team || labels.team || 'Unknown';
  const createdBy = spec.metadata?.created_by || 'Unknown';
  const environment = spec.metadata?.environment || labels.environment || 'Unknown';

  return (
    <div className="w-full h-full flex flex-col bg-white dark:bg-gray-800 border-l">
      <SpecHeader spec={spec} team={team} onClose={onClose} />

      <div className="flex-1 overflow-auto p-4">
        <Tabs defaultValue="overview" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="spec">Spec</TabsTrigger>
            <TabsTrigger value="resources">Resources</TabsTrigger>
          </TabsList>

          {/* Overview Tab */}
          <TabsContent value="overview" className="space-y-4 mt-4">
            <SpecOverviewCard
              spec={spec}
              team={team}
              createdBy={createdBy}
              environment={environment}
              labels={labels}
              resourceCount={associatedResources.length}
              workflowCount={associatedWorkflows.length}
            />

            <WorkflowsCard
              workflows={associatedWorkflows}
              isExpanded={expandedSections.has('workflows')}
              onToggle={() => toggleSection('workflows')}
              onNavigateToNode={onNavigateToNode}
            />
          </TabsContent>

          {/* Spec Tab */}
          <TabsContent value="spec" className="space-y-2 mt-4">
            {specContent ? (
              <SpecContentCard
                specContent={specContent}
                onCopy={() => copyToClipboard(specContent)}
                isCopied={copiedContent}
              />
            ) : spec.metadata && Object.keys(spec.metadata).length > 0 ? (
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm flex items-center gap-2">
                    <FileCode className="w-4 h-4" />
                    Specification Metadata
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <pre className="bg-gray-100 dark:bg-gray-900 p-3 rounded text-xs overflow-x-auto max-h-96 overflow-y-auto">
                    <code>{formatAsYAML(spec.metadata)}</code>
                  </pre>
                </CardContent>
              </Card>
            ) : (
              <p className="text-sm text-muted-foreground text-center py-8">
                No specification content available
              </p>
            )}
          </TabsContent>

          {/* Resources Tab */}
          <TabsContent value="resources" className="space-y-2 mt-4">
            {associatedResources.length > 0 ? (
              associatedResources.map((resource) => (
                <ResourceCard
                  key={resource.id}
                  resource={resource}
                  onNavigateToNode={onNavigateToNode}
                />
              ))
            ) : (
              <p className="text-sm text-muted-foreground text-center py-8">
                No resources provisioned for this spec
              </p>
            )}
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
