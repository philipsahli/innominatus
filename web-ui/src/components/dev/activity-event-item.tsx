'use client';

import { formatDistanceToNow } from 'date-fns';
import type { GraphEvent } from '@/lib/graph-websocket';

interface ActivityEventItemProps {
  event: GraphEvent;
}

/**
 * Displays a single activity event in a user-friendly format
 * Translates technical graph events into readable messages
 */
export function ActivityEventItem({ event }: ActivityEventItemProps) {
  const getEventMessage = (): { icon: string; message: string; color: string } => {
    switch (event.type) {
      case 'node_state_changed':
        return {
          icon: '🔄',
          message: `${event.node_name || event.node_id} changed from ${event.old_state} to ${event.new_state}`,
          color: getStateColor(event.new_state),
        };

      case 'node_updated':
        return {
          icon: '📝',
          message: `${event.node_name || event.node_id} updated`,
          color: 'text-zinc-600 dark:text-zinc-400',
        };

      case 'edge_added':
        return {
          icon: '🔗',
          message: `Connected ${event.from_node} to ${event.to_node}`,
          color: 'text-zinc-600 dark:text-zinc-400',
        };

      case 'graph_updated':
        return {
          icon: '📊',
          message: `Graph updated (${event.node_count} nodes, ${event.edge_count} edges)`,
          color: 'text-zinc-600 dark:text-zinc-400',
        };

      default:
        return {
          icon: '•',
          message: 'Unknown event',
          color: 'text-zinc-400',
        };
    }
  };

  const getStateColor = (state?: string): string => {
    switch (state) {
      case 'running':
      case 'provisioning':
        return 'text-blue-600 dark:text-blue-400';
      case 'succeeded':
      case 'completed':
      case 'active':
        return 'text-green-600 dark:text-green-400';
      case 'failed':
      case 'error':
        return 'text-red-600 dark:text-red-400';
      case 'pending':
      case 'waiting':
        return 'text-yellow-600 dark:text-yellow-400';
      default:
        return 'text-zinc-600 dark:text-zinc-400';
    }
  };

  const getNodeTypeLabel = (type?: string): string => {
    switch (type) {
      case 'workflow':
        return 'Workflow';
      case 'step':
        return 'Step';
      case 'resource':
        return 'Resource';
      case 'provider':
        return 'Provider';
      default:
        return type || '';
    }
  };

  const { icon, message, color } = getEventMessage();

  const timeAgo = event.timestamp
    ? formatDistanceToNow(new Date(event.timestamp), { addSuffix: true })
    : 'just now';

  return (
    <div className="flex items-start gap-3 border-l-2 border-zinc-200 py-2 pl-4 dark:border-zinc-700">
      <span className="text-lg leading-none">{icon}</span>
      <div className="flex-1 min-w-0">
        <p className={`text-sm ${color}`}>{message}</p>
        <div className="mt-1 flex items-center gap-2 text-xs text-zinc-500 dark:text-zinc-500">
          {event.node_type && (
            <span className="rounded bg-zinc-100 px-1.5 py-0.5 font-mono dark:bg-zinc-800">
              {getNodeTypeLabel(event.node_type)}
            </span>
          )}
          <span>{timeAgo}</span>
        </div>
      </div>
    </div>
  );
}
