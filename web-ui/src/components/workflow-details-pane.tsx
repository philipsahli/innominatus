'use client';

import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
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
import {
  X,
  FileCode,
  Terminal,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
  RotateCcw,
  CheckCircle2,
  XCircle,
  Clock,
  Loader2,
  Copy,
  Check,
  Info,
} from 'lucide-react';
import { WorkflowExecutionDetail, WorkflowStepExecution } from '@/lib/api';

// ============================================================================
// Types & Utilities
// ============================================================================

interface WorkflowDetailsPaneProps {
  workflow: WorkflowExecutionDetail | null;
  onClose: () => void;
  onRetry?: (workflowId: number) => void;
}

type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

const STATUS_CONFIG = {
  completed: {
    badgeVariant: 'default' as BadgeVariant,
    color: 'bg-green-500',
    icon: <CheckCircle2 className="w-5 h-5 text-green-500" />,
    timelineColor: 'bg-green-500',
  },
  running: {
    badgeVariant: 'secondary' as BadgeVariant,
    color: 'bg-yellow-500 animate-pulse',
    icon: <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />,
    timelineColor: 'bg-blue-500 animate-pulse',
  },
  failed: {
    badgeVariant: 'destructive' as BadgeVariant,
    color: 'bg-red-500',
    icon: <XCircle className="w-5 h-5 text-red-500" />,
    timelineColor: 'bg-red-500',
  },
  pending: {
    badgeVariant: 'outline' as BadgeVariant,
    color: 'bg-gray-400',
    icon: <Clock className="w-5 h-5 text-gray-400" />,
    timelineColor: 'bg-gray-400',
  },
} as const;

const getStatusConfig = (status: string) => {
  return STATUS_CONFIG[status as keyof typeof STATUS_CONFIG] || STATUS_CONFIG.pending;
};

const formatDuration = (ms: number | undefined): string => {
  if (!ms) return 'N/A';
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`;
};

const calculateProgress = (steps: WorkflowStepExecution[], totalSteps: number) => {
  const completedSteps = steps?.filter((s) => s.status === 'completed').length || 0;
  const percentage = totalSteps > 0 ? (completedSteps / totalSteps) * 100 : 0;
  return { completedSteps, totalSteps, percentage };
};

// ============================================================================
// Sub-Components
// ============================================================================

interface WorkflowHeaderProps {
  workflow: WorkflowExecutionDetail;
  completedSteps: number;
  totalSteps: number;
  percentage: number;
  onRetry?: () => void;
  onClose: () => void;
}

function WorkflowHeader({
  workflow,
  completedSteps,
  totalSteps,
  percentage,
  onRetry,
  onClose,
}: WorkflowHeaderProps) {
  const statusConfig = getStatusConfig(workflow.status);
  const progressColor =
    workflow.status === 'failed'
      ? 'bg-red-500'
      : workflow.status === 'completed'
        ? 'bg-green-500'
        : 'bg-blue-500';

  return (
    <div className="flex items-center justify-between p-4 border-b">
      <div className="flex-1">
        <div className="flex items-center gap-2 mb-2">
          <h3 className="text-lg font-semibold">{workflow.workflow_name}</h3>
          <Badge variant={statusConfig.badgeVariant}>{workflow.status}</Badge>
        </div>
        <p className="text-sm text-muted-foreground mb-2">
          Application: {workflow.application_name}
        </p>
        <div className="space-y-1">
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>
              Progress: {completedSteps} of {totalSteps} steps completed
            </span>
            <span>{Math.round(percentage)}%</span>
          </div>
          <div className="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
            <div
              className={`h-full transition-all duration-300 ${progressColor}`}
              style={{ width: `${percentage}%` }}
            />
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2">
        {workflow.status === 'failed' && onRetry && (
          <Button
            variant="outline"
            size="sm"
            onClick={onRetry}
            className="text-orange-600 hover:text-orange-700 border-orange-200 hover:bg-orange-50"
          >
            <RotateCcw className="w-4 h-4 mr-1" />
            Retry
          </Button>
        )}
        <Button variant="ghost" size="sm" onClick={onClose}>
          <X className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
}

interface StepCardProps {
  step: WorkflowStepExecution;
  isExpanded: boolean;
  onToggle: () => void;
  onRetry?: () => void;
  onCopyConfig: () => void;
  isCopied: boolean;
}

function StepCard({ step, isExpanded, onToggle, onRetry, onCopyConfig, isCopied }: StepCardProps) {
  const statusConfig = getStatusConfig(step.status);
  const cardBgClass =
    step.status === 'failed'
      ? 'border-red-200 dark:border-red-900 bg-red-50/50 dark:bg-red-950/20'
      : step.status === 'running'
        ? 'border-blue-200 dark:border-blue-900 bg-blue-50/50 dark:bg-blue-950/20'
        : '';

  return (
    <Card className={`overflow-hidden transition-all ${cardBgClass}`}>
      <CardHeader
        className="p-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700"
        onClick={onToggle}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {isExpanded ? (
              <ChevronDown className="w-4 h-4 flex-shrink-0" />
            ) : (
              <ChevronRight className="w-4 h-4 flex-shrink-0" />
            )}
            <div className="flex-shrink-0">{statusConfig.icon}</div>
            <div className="flex items-center gap-2">
              <span className="font-medium">
                {step.step_number}. {step.step_name}
              </span>
              {step.duration_ms && (
                <span className="text-xs text-muted-foreground">
                  ({formatDuration(step.duration_ms)})
                </span>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant="outline" className="text-xs">
              {step.step_type}
            </Badge>
            <Badge variant={statusConfig.badgeVariant} className="text-xs">
              {step.status}
            </Badge>
          </div>
        </div>
      </CardHeader>

      {isExpanded && (
        <CardContent className="p-3 pt-0 space-y-3">
          {/* Failed Step Action Banner */}
          {step.status === 'failed' && onRetry && (
            <div className="bg-orange-50 dark:bg-orange-950/30 border border-orange-200 dark:border-orange-900 rounded p-3 space-y-2">
              <div className="flex items-start gap-2">
                <Info className="w-4 h-4 text-orange-600 dark:text-orange-400 mt-0.5 flex-shrink-0" />
                <div className="text-sm text-orange-800 dark:text-orange-200">
                  <p className="font-medium mb-1">This step failed during execution</p>
                  <p className="text-xs">
                    Retrying the workflow will restart from this step. Previous successful steps
                    will not be re-executed.
                  </p>
                </div>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={onRetry}
                className="w-full text-orange-600 hover:text-orange-700 border-orange-200 hover:bg-orange-50 dark:hover:bg-orange-900/20"
              >
                <RotateCcw className="w-4 h-4 mr-2" />
                Retry from this step
              </Button>
            </div>
          )}

          {/* Step Configuration */}
          {step.step_config && Object.keys(step.step_config).length > 0 && (
            <div>
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <FileCode className="w-4 h-4 text-blue-500" />
                  <span className="text-sm font-medium">Configuration</span>
                </div>
                <Button variant="ghost" size="sm" onClick={onCopyConfig} className="h-7 text-xs">
                  {isCopied ? (
                    <>
                      <Check className="w-3 h-3 mr-1 text-green-500" />
                      Copied
                    </>
                  ) : (
                    <>
                      <Copy className="w-3 h-3 mr-1" />
                      Copy
                    </>
                  )}
                </Button>
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
            {step.started_at && <div>Started: {new Date(step.started_at).toLocaleString()}</div>}
            {step.completed_at && (
              <div>Completed: {new Date(step.completed_at).toLocaleString()}</div>
            )}
            {step.duration_ms && <div>Duration: {formatDuration(step.duration_ms)}</div>}
          </div>
        </CardContent>
      )}
    </Card>
  );
}

interface TimelineBarProps {
  step: WorkflowStepExecution;
  workflowStart: number;
  totalDuration: number;
}

function TimelineBar({ step, workflowStart, totalDuration }: TimelineBarProps) {
  const statusConfig = getStatusConfig(step.status);
  const startTime = step.started_at ? new Date(step.started_at).getTime() : null;
  const endTime = step.completed_at ? new Date(step.completed_at).getTime() : null;
  const duration = step.duration_ms || 0;

  const startPercent = startTime ? ((startTime - workflowStart) / totalDuration) * 100 : 0;
  const widthPercent = (duration / totalDuration) * 100;

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-sm">
        <div className="flex items-center gap-2">
          <div className="flex-shrink-0">{statusConfig.icon}</div>
          <span className="font-medium">
            {step.step_number}. {step.step_name}
          </span>
        </div>
        <span className="text-xs text-muted-foreground">{formatDuration(duration)}</span>
      </div>

      {/* Timeline Bar */}
      <div className="relative h-8 bg-gray-100 dark:bg-gray-900 rounded overflow-hidden">
        {startTime && (
          <div
            className={`absolute h-full transition-all ${statusConfig.timelineColor}`}
            style={{
              left: `${startPercent}%`,
              width: `${Math.max(widthPercent, 1)}%`,
            }}
            title={`Started: ${new Date(startTime).toLocaleTimeString()}${
              endTime ? `\nEnded: ${new Date(endTime).toLocaleTimeString()}` : ''
            }`}
          />
        )}

        {/* Time labels */}
        {startTime && (
          <div
            className="absolute top-0 bottom-0 flex items-center px-2 text-xs font-medium text-white"
            style={{ left: `${startPercent}%` }}
          >
            {new Date(startTime).toLocaleTimeString([], {
              hour: '2-digit',
              minute: '2-digit',
              second: '2-digit',
            })}
          </div>
        )}
      </div>

      {/* Step metadata */}
      <div className="flex items-center gap-4 text-xs text-muted-foreground pl-7">
        {step.step_type && <span>Type: {step.step_type}</span>}
        {startTime && <span>Started: {new Date(startTime).toLocaleTimeString()}</span>}
        {endTime && <span>Ended: {new Date(endTime).toLocaleTimeString()}</span>}
      </div>
    </div>
  );
}

interface RetryDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  workflow: WorkflowExecutionDetail;
  failedStep: WorkflowStepExecution | undefined;
}

function RetryDialog({ isOpen, onClose, onConfirm, workflow, failedStep }: RetryDialogProps) {
  return (
    <AlertDialog open={isOpen} onOpenChange={onClose}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle className="flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-orange-500" />
            Retry Workflow Execution?
          </AlertDialogTitle>
          <AlertDialogDescription className="space-y-3">
            <div>You are about to retry the workflow execution:</div>

            <div className="bg-gray-100 dark:bg-gray-900 p-3 rounded space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-medium">{workflow.workflow_name}</span>
                <Badge variant="destructive">Failed</Badge>
              </div>
              <div className="text-sm text-muted-foreground">
                Application: {workflow.application_name}
              </div>
              {failedStep && (
                <div className="text-sm">
                  <span className="text-muted-foreground">Failed at step:</span>{' '}
                  <span className="font-medium text-red-600 dark:text-red-400">
                    {failedStep.step_number}. {failedStep.step_name}
                  </span>
                </div>
              )}
            </div>

            <div className="text-sm">
              <strong>What will happen:</strong>
              <ul className="list-disc list-inside mt-1 space-y-1">
                <li>The workflow will restart from the failed step</li>
                <li>Previous successful steps will not be re-executed</li>
                <li>Any resources created by failed steps may need manual cleanup</li>
              </ul>
            </div>

            <div className="text-sm text-orange-600 dark:text-orange-400 flex items-start gap-2">
              <AlertTriangle className="w-4 h-4 mt-0.5 flex-shrink-0" />
              <span>
                Make sure the underlying issue causing the failure has been resolved before
                retrying.
              </span>
            </div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm} className="bg-orange-600 hover:bg-orange-700">
            <RotateCcw className="w-4 h-4 mr-1" />
            Retry Workflow
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

// ============================================================================
// Main Component
// ============================================================================

export function WorkflowDetailsPane({ workflow, onClose, onRetry }: WorkflowDetailsPaneProps) {
  const [expandedSteps, setExpandedSteps] = useState<Set<number>>(new Set());
  const [showRetryDialog, setShowRetryDialog] = useState(false);
  const [copiedStepId, setCopiedStepId] = useState<number | null>(null);

  if (!workflow) {
    return null;
  }

  const failedStep = workflow.steps?.find((step) => step.status === 'failed');
  const progress = calculateProgress(
    workflow.steps || [],
    workflow.total_steps || workflow.steps?.length || 0
  );

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

  const handleRetryClick = () => setShowRetryDialog(true);

  const handleRetryConfirm = () => {
    if (onRetry) {
      onRetry(workflow.id);
    }
    setShowRetryDialog(false);
  };

  const copyStepConfig = async (step: WorkflowStepExecution) => {
    try {
      const configText = step.step_config
        ? JSON.stringify(step.step_config, null, 2)
        : 'No configuration available';
      await navigator.clipboard.writeText(configText);
      setCopiedStepId(step.id);
      setTimeout(() => setCopiedStepId(null), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  // Timeline calculations
  const workflowStart = new Date(workflow.started_at).getTime();
  const workflowEnd = workflow.completed_at
    ? new Date(workflow.completed_at).getTime()
    : Date.now();
  const totalDuration = workflowEnd - workflowStart;

  return (
    <div className="w-full h-full flex flex-col bg-white dark:bg-gray-800 border-l">
      <WorkflowHeader
        workflow={workflow}
        completedSteps={progress.completedSteps}
        totalSteps={progress.totalSteps}
        percentage={progress.percentage}
        onRetry={onRetry && handleRetryClick}
        onClose={onClose}
      />

      <div className="flex-1 overflow-auto p-4">
        <Tabs defaultValue="steps" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="steps">Steps</TabsTrigger>
            <TabsTrigger value="timeline">Timeline</TabsTrigger>
            <TabsTrigger value="overview">Overview</TabsTrigger>
          </TabsList>

          {/* Steps Tab */}
          <TabsContent value="steps" className="space-y-2 mt-4">
            {workflow.steps && workflow.steps.length > 0 ? (
              workflow.steps.map((step) => (
                <StepCard
                  key={step.id}
                  step={step}
                  isExpanded={expandedSteps.has(step.id)}
                  onToggle={() => toggleStep(step.id)}
                  onRetry={onRetry && handleRetryClick}
                  onCopyConfig={() => copyStepConfig(step)}
                  isCopied={copiedStepId === step.id}
                />
              ))
            ) : (
              <p className="text-sm text-muted-foreground text-center py-8">No steps available</p>
            )}
          </TabsContent>

          {/* Timeline Tab */}
          <TabsContent value="timeline" className="space-y-4 mt-4">
            {workflow.steps && workflow.steps.length > 0 ? (
              <>
                <div className="text-sm text-muted-foreground mb-4">
                  Visual timeline showing when each step started and completed
                </div>
                <div className="space-y-3">
                  {workflow.steps.map((step) => (
                    <TimelineBar
                      key={step.id}
                      step={step}
                      workflowStart={workflowStart}
                      totalDuration={totalDuration}
                    />
                  ))}
                </div>

                {/* Timeline Summary */}
                <Card className="mt-6">
                  <CardHeader>
                    <CardTitle className="text-sm">Execution Summary</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2 text-sm">
                    <div className="grid grid-cols-2 gap-2">
                      <div className="text-muted-foreground">Total Duration</div>
                      <div className="font-medium">
                        {workflow.completed_at ? formatDuration(totalDuration) : 'In progress...'}
                      </div>

                      <div className="text-muted-foreground">Workflow Started</div>
                      <div>{new Date(workflow.started_at).toLocaleString()}</div>

                      {workflow.completed_at && (
                        <>
                          <div className="text-muted-foreground">Workflow Ended</div>
                          <div>{new Date(workflow.completed_at).toLocaleString()}</div>
                        </>
                      )}
                    </div>
                  </CardContent>
                </Card>
              </>
            ) : (
              <p className="text-sm text-muted-foreground text-center py-8">
                No timeline data available
              </p>
            )}
          </TabsContent>

          {/* Overview Tab */}
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

      <RetryDialog
        isOpen={showRetryDialog}
        onClose={() => setShowRetryDialog(false)}
        onConfirm={handleRetryConfirm}
        workflow={workflow}
        failedStep={failedStep}
      />
    </div>
  );
}
