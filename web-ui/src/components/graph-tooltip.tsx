'use client';

import React from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Package,
  Database,
  GitBranch,
  PlayCircle,
  Users,
  Clock,
  CheckCircle2,
  XCircle,
  AlertCircle,
} from 'lucide-react';
import { GraphNode } from '@/lib/api';

interface GraphTooltipProps {
  node: GraphNode | null;
  x: number;
  y: number;
  visible: boolean;
}

export function GraphTooltip({ node, x, y, visible }: GraphTooltipProps) {
  if (!visible || !node) {
    return null;
  }

  const getNodeIcon = (type: string) => {
    switch (type.toLowerCase()) {
      case 'spec':
        return <Package className="w-4 h-4 text-purple-500" />;
      case 'resource':
        return <Database className="w-4 h-4 text-blue-500" />;
      case 'workflow':
        return <GitBranch className="w-4 h-4 text-orange-500" />;
      case 'step':
        return <PlayCircle className="w-4 h-4 text-green-500" />;
      case 'provider':
        return <Users className="w-4 h-4 text-indigo-500" />;
      default:
        return <AlertCircle className="w-4 h-4 text-gray-500" />;
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status.toLowerCase()) {
      case 'active':
      case 'completed':
      case 'succeeded':
        return <CheckCircle2 className="w-3 h-3 text-green-500" />;
      case 'failed':
      case 'error':
      case 'terminated':
        return <XCircle className="w-3 h-3 text-red-500" />;
      case 'running':
      case 'provisioning':
      case 'in_progress':
        return <PlayCircle className="w-3 h-3 text-blue-500" />;
      default:
        return <Clock className="w-3 h-3 text-gray-400" />;
    }
  };

  const getStatusBadgeVariant = (status: string) => {
    switch (status.toLowerCase()) {
      case 'active':
      case 'completed':
      case 'succeeded':
        return 'default';
      case 'running':
      case 'provisioning':
      case 'in_progress':
        return 'secondary';
      case 'failed':
      case 'error':
      case 'terminated':
        return 'destructive';
      default:
        return 'outline';
    }
  };

  const formatDuration = (ms: number | undefined) => {
    if (!ms) return null;
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  // Extract common metadata
  const resourceType = node.metadata?.resource_type;
  const providerId = node.metadata?.provider_id;
  const stepNumber = node.metadata?.step_number || node.step_number;
  const stepType = node.metadata?.step_type;
  const duration = formatDuration(node.duration_ms);
  const team = node.metadata?.team || node.metadata?.labels?.team;
  const healthStatus = node.metadata?.health_status;

  return (
    <div
      className="fixed z-50 pointer-events-none"
      style={{
        left: `${x + 15}px`,
        top: `${y + 15}px`,
      }}
    >
      <Card className="shadow-lg border-2 bg-white dark:bg-gray-800 min-w-64 max-w-80">
        <CardContent className="p-3 space-y-2">
          {/* Header */}
          <div className="flex items-start gap-2">
            <div className="mt-0.5">{getNodeIcon(node.type)}</div>
            <div className="flex-1 min-w-0">
              <div className="font-semibold text-sm truncate">{node.name}</div>
              <div className="text-xs text-muted-foreground capitalize">{node.type}</div>
            </div>
            <div className="flex items-center gap-1">{getStatusIcon(node.status)}</div>
          </div>

          {/* Status Badge */}
          <div className="flex items-center gap-2">
            <Badge variant={getStatusBadgeVariant(node.status)} className="text-xs">
              {node.status}
            </Badge>
            {healthStatus && (
              <Badge variant="outline" className="text-xs">
                {healthStatus}
              </Badge>
            )}
          </div>

          {/* Description */}
          {node.description && (
            <div className="text-xs text-muted-foreground line-clamp-2">{node.description}</div>
          )}

          {/* Type-specific details */}
          <div className="space-y-1 text-xs">
            {/* Resource details */}
            {node.type === 'resource' && resourceType && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">Type:</span>
                <span className="font-medium">{resourceType}</span>
              </div>
            )}

            {providerId && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">Provider:</span>
                <span className="font-medium">{providerId}</span>
              </div>
            )}

            {/* Step details */}
            {node.type === 'step' && (
              <>
                {stepNumber && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Step:</span>
                    <span className="font-medium">#{stepNumber}</span>
                  </div>
                )}
                {stepType && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Type:</span>
                    <span className="font-medium">{stepType}</span>
                  </div>
                )}
              </>
            )}

            {/* Workflow details */}
            {node.type === 'workflow' && node.step_number !== undefined && node.total_steps && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">Progress:</span>
                <span className="font-medium">
                  {node.step_number}/{node.total_steps} steps
                </span>
              </div>
            )}

            {/* Duration */}
            {duration && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">Duration:</span>
                <span className="font-medium">{duration}</span>
              </div>
            )}

            {/* Team */}
            {team && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">Team:</span>
                <span className="font-medium">{team}</span>
              </div>
            )}

            {/* Timestamps */}
            {node.created_at && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">Created:</span>
                <span className="font-medium">
                  {new Date(node.created_at).toLocaleDateString()}
                </span>
              </div>
            )}
          </div>

          {/* Click hint */}
          <div className="text-xs text-muted-foreground italic border-t pt-2">
            Click for details
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
