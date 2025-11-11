'use client';

import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Activity, Clock } from 'lucide-react';
import type { Workflow, WorkflowStep } from './workflow-table';

interface LiveStepsMonitorProps {
  workflows: Workflow[];
}

function LiveStepCard({ workflow, step }: { workflow: Workflow; step: WorkflowStep }) {
  const getElapsedTime = (startedAt?: string) => {
    if (!startedAt) return '0s';
    const start = new Date(startedAt);
    const now = new Date();
    const elapsedMs = now.getTime() - start.getTime();
    const seconds = Math.floor(elapsedMs / 1000);
    const minutes = Math.floor(seconds / 60);

    if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    }
    return `${seconds}s`;
  };

  const getStatusBadgeStyle = (status: string) => {
    if (status === 'running') {
      return 'bg-blue-500/10 text-blue-500 border-blue-500/20 animate-pulse';
    }
    if (status === 'completed') {
      return 'bg-lime-500/10 text-lime-500 border-lime-500/20';
    }
    if (status === 'failed') {
      return 'bg-red-500/10 text-red-500 border-red-500/20';
    }
    return 'bg-zinc-500/10 text-zinc-500 border-zinc-500/20';
  };

  const getIconStyle = (status: string) => {
    if (status === 'running') return 'text-blue-500 animate-pulse';
    if (status === 'completed') return 'text-lime-500';
    if (status === 'failed') return 'text-red-500';
    return 'text-zinc-500';
  };

  return (
    <Card className="p-4 bg-white dark:bg-zinc-900/50 border-zinc-200 dark:border-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-900/70 transition-colors">
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <Activity size={14} className={getIconStyle(step.status)} />
            <div className="text-sm font-medium text-zinc-900 dark:text-white truncate">
              {step.step_name}
            </div>
          </div>
          <div className="text-xs text-zinc-500 truncate">
            {workflow.app_name} / {workflow.name}
          </div>
        </div>

        <Badge variant="outline" className={`ml-2 text-xs ${getStatusBadgeStyle(step.status)}`}>
          {step.status}
        </Badge>
      </div>

      <div className="flex items-center justify-between text-xs">
        <div className="flex items-center gap-4">
          <div>
            <span className="text-zinc-500">Step:</span>{' '}
            <span className="text-zinc-900 dark:text-white font-mono">#{step.step_number}</span>
          </div>
          <div>
            <span className="text-zinc-500">Type:</span>{' '}
            <span className="text-zinc-900 dark:text-white">{step.step_type}</span>
          </div>
        </div>

        <div className="flex items-center gap-1 text-zinc-500">
          <Clock size={12} />
          <span>{getElapsedTime(step.started_at)}</span>
        </div>
      </div>

      <div className="mt-3 pt-3 border-t border-zinc-200 dark:border-zinc-800 text-xs text-zinc-500">
        Workflow #{workflow.id} • Started{' '}
        {step.started_at ? new Date(step.started_at).toLocaleTimeString() : 'N/A'}
      </div>
    </Card>
  );
}

export function LiveStepsMonitor({ workflows }: LiveStepsMonitorProps) {
  // Find all running steps and recently completed steps (last 10 minutes)
  const runningSteps: Array<{ workflow: Workflow; step: WorkflowStep }> = [];
  const RECENT_THRESHOLD_MS = 10 * 60 * 1000; // 10 minutes

  workflows.forEach((workflow) => {
    if (workflow.steps) {
      workflow.steps
        .filter((s) => {
          // Include running steps
          if (s.status === 'running') return true;

          // Include recently completed steps (last 10 minutes)
          if (s.status === 'completed' && s.completed_at) {
            const completedTime = new Date(s.completed_at);
            const now = new Date();
            const timeSinceCompletion = now.getTime() - completedTime.getTime();
            return timeSinceCompletion <= RECENT_THRESHOLD_MS;
          }

          // Include recently failed steps (last 10 minutes) for debugging
          if (s.status === 'failed' && s.completed_at) {
            const failedTime = new Date(s.completed_at);
            const now = new Date();
            const timeSinceFailure = now.getTime() - failedTime.getTime();
            return timeSinceFailure <= RECENT_THRESHOLD_MS;
          }

          return false;
        })
        .forEach((step) => {
          runningSteps.push({ workflow, step });
        });
    }
  });

  if (runningSteps.length === 0) {
    return (
      <Card className="p-8 bg-white dark:bg-zinc-900/50 border-zinc-200 dark:border-zinc-800">
        <div className="text-center">
          <Activity size={32} className="mx-auto mb-3 text-zinc-400 dark:text-zinc-600" />
          <div className="text-zinc-600 dark:text-zinc-400">
            No workflow steps currently running
          </div>
          <div className="text-xs text-zinc-500 dark:text-zinc-500 mt-1">
            Running steps will appear here automatically
          </div>
        </div>
      </Card>
    );
  }

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <div className="text-sm text-zinc-600 dark:text-zinc-400">
          Monitoring {runningSteps.length} active {runningSteps.length === 1 ? 'step' : 'steps'}
        </div>
        <div className="flex items-center gap-2 text-xs text-zinc-500 dark:text-zinc-400">
          <div className="w-2 h-2 rounded-full bg-blue-500 animate-pulse" />
          Live updates every 3s
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {runningSteps.map(({ workflow, step }) => (
          <LiveStepCard key={`${workflow.id}-${step.id}`} workflow={workflow} step={step} />
        ))}
      </div>
    </div>
  );
}
