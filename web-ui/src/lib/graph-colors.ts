/**
 * Consistent color scheme for graph visualizations
 * Used across all graph implementations (SVG, ReactFlow, Cytoscape, D3)
 */

export const NODE_COLORS = {
  // Node types
  spec: {
    fill: '#8b5cf6',
    stroke: '#7c3aed',
    text: '#ffffff',
  },
  resource: {
    fill: '#10b981',
    stroke: '#059669',
    text: '#ffffff',
  },
  provider: {
    fill: '#f59e0b',
    stroke: '#d97706',
    text: '#ffffff',
  },
  workflow: {
    fill: '#06b6d4',
    stroke: '#0891b2',
    text: '#ffffff',
  },
  step: {
    fill: '#3b82f6',
    stroke: '#2563eb',
    text: '#ffffff',
  },
  default: {
    fill: '#6b7280',
    stroke: '#4b5563',
    text: '#ffffff',
  },
} as const;

export const STATUS_COLORS = {
  // Status states
  running: {
    fill: '#3b82f6',
    stroke: '#2563eb',
    text: '#ffffff',
    glow: true,
  },
  succeeded: {
    fill: '#10b981',
    stroke: '#059669',
    text: '#ffffff',
    glow: false,
  },
  completed: {
    fill: '#10b981',
    stroke: '#059669',
    text: '#ffffff',
    glow: false,
  },
  active: {
    fill: '#10b981',
    stroke: '#059669',
    text: '#ffffff',
    glow: false,
  },
  failed: {
    fill: '#ef4444',
    stroke: '#dc2626',
    text: '#ffffff',
    glow: false,
  },
  waiting: {
    fill: '#f59e0b',
    stroke: '#d97706',
    text: '#ffffff',
    glow: false,
  },
  pending: {
    fill: '#f59e0b',
    stroke: '#d97706',
    text: '#ffffff',
    glow: false,
  },
  requested: {
    fill: '#eab308',
    stroke: '#ca8a04',
    text: '#ffffff',
    glow: false,
  },
  provisioning: {
    fill: '#3b82f6',
    stroke: '#2563eb',
    text: '#ffffff',
    glow: true,
  },
  terminating: {
    fill: '#f97316',
    stroke: '#ea580c',
    text: '#ffffff',
    glow: false,
  },
  terminated: {
    fill: '#71717a',
    stroke: '#52525b',
    text: '#ffffff',
    glow: false,
  },
} as const;

export const EDGE_COLORS = {
  default: '#a1a1aa',
  highlight: '#3b82f6',
  error: '#ef4444',
  success: '#10b981',
} as const;

/**
 * Get node color based on type and optional status
 * Status takes precedence over type
 */
export function getNodeColor(
  type: string,
  status?: string
): { fill: string; stroke: string; text: string; glow?: boolean } {
  // Status colors override type colors
  if (status && status in STATUS_COLORS) {
    return STATUS_COLORS[status as keyof typeof STATUS_COLORS];
  }

  // Fall back to type colors
  if (type in NODE_COLORS) {
    return NODE_COLORS[type as keyof typeof NODE_COLORS];
  }

  return NODE_COLORS.default;
}

/**
 * Get edge color based on relationship type
 */
export function getEdgeColor(relationship?: string): string {
  switch (relationship) {
    case 'error':
      return EDGE_COLORS.error;
    case 'success':
      return EDGE_COLORS.success;
    default:
      return EDGE_COLORS.default;
  }
}

/**
 * Get Tailwind CSS classes for badges/labels
 */
export function getStatusBadgeClass(status: string): string {
  switch (status) {
    case 'running':
    case 'provisioning':
      return 'bg-blue-500/10 text-blue-500 border-blue-500/20 animate-pulse';
    case 'succeeded':
    case 'completed':
    case 'active':
      return 'bg-lime-500/10 text-lime-500 border-lime-500/20';
    case 'failed':
      return 'bg-red-500/10 text-red-500 border-red-500/20';
    case 'waiting':
    case 'pending':
    case 'requested':
      return 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20';
    case 'terminating':
      return 'bg-orange-500/10 text-orange-500 border-orange-500/20';
    case 'terminated':
      return 'bg-zinc-500/10 text-zinc-500 border-zinc-500/20';
    default:
      return 'bg-zinc-500/10 text-zinc-500 border-zinc-500/20';
  }
}

/**
 * Get type badge class for node types
 */
export function getTypeBadgeClass(type: string): string {
  switch (type) {
    case 'spec':
      return 'bg-purple-500/10 text-purple-500 border-purple-500/20';
    case 'resource':
      return 'bg-green-500/10 text-green-500 border-green-500/20';
    case 'provider':
      return 'bg-amber-500/10 text-amber-500 border-amber-500/20';
    case 'workflow':
      return 'bg-cyan-500/10 text-cyan-500 border-cyan-500/20';
    case 'step':
      return 'bg-blue-500/10 text-blue-500 border-blue-500/20';
    default:
      return 'bg-zinc-500/10 text-zinc-500 border-zinc-500/20';
  }
}
