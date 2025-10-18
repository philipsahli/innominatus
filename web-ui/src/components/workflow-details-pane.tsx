'use client';

import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { X, FileCode, Terminal, ChevronDown, ChevronRight } from 'lucide-react';
import { WorkflowExecutionDetail, WorkflowStepExecution } from '@/lib/api';

interface WorkflowDetailsPaneProps {
  workflow: WorkflowExecutionDetail | null;
  onClose: () => void;
  onRetry?: (workflowId: number) => void;
}

export function WorkflowDetailsPane({ workflow, onClose, onRetry }: WorkflowDetailsPaneProps) {
  const [expandedSteps, setExpandedSteps] = useState<Set<number>>(new Set());

  if (!workflow) {
    return null;
  }

  const toggleStep = (stepId: number) => {
    setExpandedSteps((prev) => {
      const next = new Set(prev);
      if (next.has(stepId)) {
        next.delete(stepId);
      } else {
        next.add(stepId);
      }
      return next;
    });
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'bg-green-500';
      case 'running':
        return 'bg-yellow-500 animate-pulse';
      case 'failed':
        return 'bg-red-500';
      case 'pending':
        return 'bg-gray-400';
      default:
        return 'bg-gray-300';
    }
  };

  const getStatusBadgeVariant = (status: string) => {
    switch (status) {
      case 'completed':
        return 'default';
      case 'running':
        return 'secondary';
      case 'failed':
        return 'destructive';
      default:
        return 'outline';
    }
  };

  return (
    <div className="w-full h-full flex flex-col bg-white dark:bg-gray-800 border-l">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <h3 className="text-lg font-semibold">{workflow.workflow_name}</h3>
            <Badge variant={getStatusBadgeVariant(workflow.status)}>{workflow.status}</Badge>
          </div>
          <p className="text-sm text-muted-foreground">Application: {workflow.application_name}</p>
        </div>
        <div className="flex items-center gap-2">
          {workflow.status === 'failed' && onRetry && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => onRetry(workflow.id)}
              className="text-orange-600 hover:text-orange-700 border-orange-200 hover:bg-orange-50"
            >
              Retry
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={onClose}>
            <X className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-4">
        <Tabs defaultValue="steps" className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="steps">Steps</TabsTrigger>
            <TabsTrigger value="overview">Overview</TabsTrigger>
          </TabsList>

          <TabsContent value="steps" className="space-y-2 mt-4">
            {workflow.steps && workflow.steps.length > 0 ? (
              workflow.steps.map((step) => (
                <Card key={step.id} className="overflow-hidden">
                  <CardHeader
                    className="p-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700"
                    onClick={() => toggleStep(step.id)}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        {expandedSteps.has(step.id) ? (
                          <ChevronDown className="w-4 h-4" />
                        ) : (
                          <ChevronRight className="w-4 h-4" />
                        )}
                        <div className={`w-2 h-2 rounded-full ${getStatusColor(step.status)}`} />
                        <span className="font-medium">
                          {step.step_number}. {step.step_name}
                        </span>
                      </div>
                      <Badge variant="outline" className="text-xs">
                        {step.step_type}
                      </Badge>
                    </div>
                  </CardHeader>

                  {expandedSteps.has(step.id) && (
                    <CardContent className="p-3 pt-0 space-y-3">
                      {/* Step Configuration */}
                      {step.step_config && Object.keys(step.step_config).length > 0 && (
                        <div>
                          <div className="flex items-center gap-2 mb-2">
                            <FileCode className="w-4 h-4 text-blue-500" />
                            <span className="text-sm font-medium">Configuration</span>
                          </div>
                          <pre className="bg-gray-100 dark:bg-gray-900 p-2 rounded text-xs overflow-x-auto">
                            <code>{JSON.stringify(step.step_config, null, 2)}</code>
                          </pre>
                        </div>
                      )}

                      {/* Step Logs */}
                      {step.output_logs && (
                        <div>
                          <div className="flex items-center gap-2 mb-2">
                            <Terminal className="w-4 h-4 text-green-500" />
                            <span className="text-sm font-medium">Logs</span>
                          </div>
                          <pre className="bg-gray-900 text-green-400 p-2 rounded text-xs overflow-x-auto max-h-64 overflow-y-auto">
                            <code>{step.output_logs}</code>
                          </pre>
                        </div>
                      )}

                      {/* Error Message */}
                      {step.error_message && (
                        <div>
                          <div className="flex items-center gap-2 mb-2">
                            <Terminal className="w-4 h-4 text-red-500" />
                            <span className="text-sm font-medium text-red-600">Error</span>
                          </div>
                          <pre className="bg-red-50 dark:bg-red-950 text-red-800 dark:text-red-200 p-2 rounded text-xs overflow-x-auto">
                            <code>{step.error_message}</code>
                          </pre>
                        </div>
                      )}

                      {/* Step Metadata */}
                      <div className="text-xs text-muted-foreground space-y-1">
                        {step.started_at && (
                          <div>Started: {new Date(step.started_at).toLocaleString()}</div>
                        )}
                        {step.completed_at && (
                          <div>Completed: {new Date(step.completed_at).toLocaleString()}</div>
                        )}
                        {step.duration_ms && <div>Duration: {step.duration_ms}ms</div>}
                      </div>
                    </CardContent>
                  )}
                </Card>
              ))
            ) : (
              <p className="text-sm text-muted-foreground text-center py-8">No steps available</p>
            )}
          </TabsContent>

          <TabsContent value="overview" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Workflow Information</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 text-sm">
                <div className="grid grid-cols-2 gap-2">
                  <div className="text-muted-foreground">Execution ID</div>
                  <div className="font-mono">{workflow.id}</div>

                  <div className="text-muted-foreground">Total Steps</div>
                  <div>{workflow.total_steps}</div>

                  <div className="text-muted-foreground">Started</div>
                  <div>{new Date(workflow.started_at).toLocaleString()}</div>

                  {workflow.completed_at && (
                    <>
                      <div className="text-muted-foreground">Completed</div>
                      <div>{new Date(workflow.completed_at).toLocaleString()}</div>
                    </>
                  )}
                </div>

                {workflow.error_message && (
                  <div className="mt-4">
                    <div className="text-muted-foreground mb-1">Error Message</div>
                    <pre className="bg-red-50 dark:bg-red-950 text-red-800 dark:text-red-200 p-2 rounded text-xs overflow-x-auto">
                      <code>{workflow.error_message}</code>
                    </pre>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
