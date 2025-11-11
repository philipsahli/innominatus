import { cn } from '@/lib/utils';
import { Circle, Loader2, CheckCircle2, XCircle, Clock, AlertTriangle } from 'lucide-react';

type Status =
  | 'running'
  | 'completed'
  | 'failed'
  | 'pending'
  | 'active'
  | 'provisioning'
  | 'degraded'
  | 'terminated'
  | 'requested'
  | 'scaling'
  | 'updating'
  | 'terminating';

const statusConfig: Record<
  Status,
  {
    label: string;
    icon: typeof Circle;
    className: string;
  }
> = {
  // Workflow statuses
  running: {
    label: 'Running',
    icon: Loader2,
    className: 'text-blue-600 dark:text-blue-400',
  },
  completed: {
    label: 'Completed',
    icon: CheckCircle2,
    className: 'text-green-600 dark:text-green-400',
  },
  failed: {
    label: 'Failed',
    icon: XCircle,
    className: 'text-red-600 dark:text-red-400',
  },
  pending: {
    label: 'Pending',
    icon: Clock,
    className: 'text-zinc-500 dark:text-zinc-400',
  },
  // Resource statuses
  active: {
    label: 'Active',
    icon: CheckCircle2,
    className: 'text-green-600 dark:text-green-400',
  },
  provisioning: {
    label: 'Provisioning',
    icon: Loader2,
    className: 'text-blue-600 dark:text-blue-400',
  },
  degraded: {
    label: 'Degraded',
    icon: AlertTriangle,
    className: 'text-amber-600 dark:text-amber-400',
  },
  terminated: {
    label: 'Terminated',
    icon: XCircle,
    className: 'text-zinc-500 dark:text-zinc-400',
  },
  requested: {
    label: 'Requested',
    icon: Clock,
    className: 'text-zinc-500 dark:text-zinc-400',
  },
  scaling: {
    label: 'Scaling',
    icon: Loader2,
    className: 'text-blue-600 dark:text-blue-400',
  },
  updating: {
    label: 'Updating',
    icon: Loader2,
    className: 'text-blue-600 dark:text-blue-400',
  },
  terminating: {
    label: 'Terminating',
    icon: Loader2,
    className: 'text-amber-600 dark:text-amber-400',
  },
};

export function StatusBadge({ status, className }: { status: Status; className?: string }) {
  const config = statusConfig[status];
  if (!config) {
    return <span className="text-xs text-zinc-500">{status}</span>;
  }

  const Icon = config.icon;
  const isAnimating =
    status === 'running' ||
    status === 'provisioning' ||
    status === 'scaling' ||
    status === 'updating' ||
    status === 'terminating';

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 text-xs font-medium',
        config.className,
        className
      )}
    >
      <Icon size={12} className={isAnimating ? 'animate-spin' : ''} />
      {config.label}
    </span>
  );
}

// Minimal dot-only version for compact displays
export function StatusDot({ status }: { status: Status }) {
  const config = statusConfig[status];
  if (!config) {
    return <Circle size={8} className="text-zinc-400" fill="currentColor" />;
  }

  return (
    <Circle size={8} className={config.className} fill="currentColor" aria-label={config.label} />
  );
}
