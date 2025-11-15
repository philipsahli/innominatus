'use client';

import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  X,
  PlayCircle,
  CheckCircle2,
  XCircle,
  Clock,
  Terminal,
  FileCode,
  Copy,
  Check,
  AlertCircle,
} from 'lucide-react';
import { GraphNode } from '@/lib/api';
import { formatAsYAML } from '@/lib/formatters';

interface StepDetailsPaneProps {
  step: GraphNode | null;
  onClose: () => void;
}

export function StepDetailsPane({ step, onClose }: StepDetailsPaneProps) {
  const [copiedConfig, setCopiedConfig] = useState(false);
  const [copiedLogs, setCopiedLogs] = useState(false);

  if (!step) {
    return null;
  }

  const copyToClipboard = async (text: string, setter: (val: boolean) => void) => {
    try {
      await navigator.clipboard.writeText(text);
      setter(true);
      setTimeout(() => setter(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  // Extract step metadata
  const stepNumber = step.metadata?.step_number || step.step_number;
  const stepType = step.metadata?.step_type || 'unknown';
  const startedAt = step.metadata?.started_at || step.created_at;
  const completedAt = step.metadata?.completed_at || step.updated_at;
  const durationMs = step.metadata?.duration_ms || step.duration_ms;
  const errorMessage = step.metadata?.error_message;
  const outputLogs = step.metadata?.output_logs;
  const stepConfig = step.metadata?.step_config || step.metadata?.config;
  const workflowId = step.metadata?.workflow_execution_id || step.workflow_id;

  const getStatusIcon = (status: string) => {
    switch (status.toLowerCase()) {
      case 'completed':
      case 'succeeded':
        return <CheckCircle2 className="w-5 h-5 text-green-500" />;
      case 'running':
      case 'in_progress':
        return <PlayCircle className="w-5 h-5 text-blue-500 animate-pulse" />;
      case 'failed':
      case 'error':
        return <XCircle className="w-5 h-5 text-red-500" />;
      case 'pending':
        return <Clock className="w-5 h-5 text-gray-400" />;
      default:
        return <AlertCircle className="w-5 h-5 text-yellow-500" />;
    }
  };

  const getStatusBadgeVariant = (status: string) => {
    switch (status.toLowerCase()) {
      case 'completed':
      case 'succeeded':
        return 'default';
      case 'running':
      case 'in_progress':
        return 'secondary';
      case 'failed':
      case 'error':
        return 'destructive';
      default:
        return 'outline';
    }
  };

  const formatDuration = (ms: number | undefined) => {
    if (!ms) return 'N/A';
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  return (
    <div className="w-full h-full flex flex-col bg-white dark:bg-gray-800 border-l">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            {getStatusIcon(step.status)}
            <h3 className="text-lg font-semibold">{step.name}</h3>
            <Badge variant={getStatusBadgeVariant(step.status)}>{step.status}</Badge>
          </div>
          <div className="text-sm text-muted-foreground space-y-1">
            <div>
              Step {stepNumber} • Type: {stepType}
            </div>
            {workflowId && <div>Workflow: {workflowId}</div>}
          </div>
        </div>
        <Button variant="ghost" size="sm" onClick={onClose}>
          <X className="w-4 h-4" />
        </Button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-4">
        <Tabs defaultValue="overview" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="configuration">Configuration</TabsTrigger>
            <TabsTrigger value="logs">Logs</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-4 mt-4">
            {/* Execution Information */}
            <Card>
              <CardHeader>
                <CardTitle className="text-sm flex items-center gap-2">
                  <PlayCircle className="w-4 h-4" />
                  Execution Details
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 text-sm">
                <div className="grid grid-cols-2 gap-2">
                  <div className="text-muted-foreground">Step Number</div>
                  <div className="font-medium">{stepNumber || 'N/A'}</div>

                  <div className="text-muted-foreground">Step Type</div>
                  <div className="font-medium">{stepType}</div>

                  <div className="text-muted-foreground">Status</div>
                  <div>
                    <Badge variant={getStatusBadgeVariant(step.status)}>{step.status}</Badge>
                  </div>

                  <div className="text-muted-foreground">Duration</div>
                  <div className="font-medium">{formatDuration(durationMs)}</div>

                  {startedAt && (
                    <>
                      <div className="text-muted-foreground">Started</div>
                      <div className="flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {new Date(startedAt).toLocaleString()}
                      </div>
                    </>
                  )}

                  {completedAt && (
                    <>
                      <div className="text-muted-foreground">Completed</div>
                      <div className="flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {new Date(completedAt).toLocaleString()}
                      </div>
                    </>
                  )}
                </div>

                {step.description && (
                  <div className="mt-4">
                    <div className="text-muted-foreground mb-1">Description</div>
                    <div className="text-sm">{step.description}</div>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Error Message */}
            {errorMessage && (
              <Card className="border-red-200 dark:border-red-900">
                <CardHeader>
                  <CardTitle className="text-sm flex items-center gap-2 text-red-600 dark:text-red-400">
                    <XCircle className="w-4 h-4" />
                    Error Details
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <pre className="bg-red-50 dark:bg-red-950 text-red-800 dark:text-red-200 p-3 rounded text-xs overflow-x-auto whitespace-pre-wrap">
                    {errorMessage}
                  </pre>
                </CardContent>
              </Card>
            )}
          </TabsContent>

          <TabsContent value="configuration" className="space-y-2 mt-4">
            {stepConfig && Object.keys(stepConfig).length > 0 ? (
              <Card>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-sm flex items-center gap-2">
                      <FileCode className="w-4 h-4" />
                      Step Configuration
                    </CardTitle>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => copyToClipboard(formatAsYAML(stepConfig), setCopiedConfig)}
                    >
                      {copiedConfig ? (
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
                    <code>{formatAsYAML(stepConfig)}</code>
                  </pre>
                </CardContent>
              </Card>
            ) : (
              <p className="text-sm text-muted-foreground text-center py-8">
                No configuration data available
              </p>
            )}
          </TabsContent>

          <TabsContent value="logs" className="space-y-2 mt-4">
            {outputLogs ? (
              <Card>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-sm flex items-center gap-2">
                      <Terminal className="w-4 h-4" />
                      Execution Logs
                    </CardTitle>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => copyToClipboard(outputLogs, setCopiedLogs)}
                    >
                      {copiedLogs ? (
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
                  <pre className="bg-gray-900 text-gray-100 p-3 rounded text-xs overflow-x-auto max-h-96 overflow-y-auto font-mono">
                    {outputLogs}
                  </pre>
                </CardContent>
              </Card>
            ) : (
              <p className="text-sm text-muted-foreground text-center py-8">
                No logs available for this step
              </p>
            )}
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
