export const theme = {
  colors: {
    // Primary lemon green palette
    primary: '#A8E10C',
    primaryDark: '#8BC700',
    primaryLight: '#B8E834',

    // Status colors with semantic meaning
    status: {
      success: '#16a34a', // Green - operations completed successfully
      warning: '#f59e0b', // Amber - attention needed
      error: '#ef4444', // Red - failures or critical issues
      info: '#3b82f6', // Blue - informational states
      pending: '#6b7280', // Gray - waiting or queued states
    },

    // IDP-specific workflow states
    workflow: {
      running: '#f59e0b', // Amber - actively executing
      completed: '#16a34a', // Green - finished successfully
      failed: '#ef4444', // Red - execution failed
      pending: '#6b7280', // Gray - queued for execution
    },

    // IDP-specific resource states
    resource: {
      active: '#16a34a', // Green - healthy and operational
      provisioning: '#3b82f6', // Blue - being created/configured
      degraded: '#f59e0b', // Amber - operational but impaired
      terminated: '#6b7280', // Gray - decommissioned
    },

    // Neutral palette
    background: {
      from: 'slate-50',
      via: 'gray-50',
      to: 'zinc-50',
    },

    // Card gradients for different sections
    cards: {
      applications: {
        from: 'blue-500/10',
        to: 'blue-600/5',
      },
      workflows: {
        from: 'emerald-500/10',
        to: 'emerald-600/5',
      },
      resources: {
        from: 'purple-500/10',
        to: 'purple-600/5',
      },
      users: {
        from: 'orange-500/10',
        to: 'orange-600/5',
      },
    },
  },

  // Spacing and sizing
  spacing: {
    containerPadding: '1.5rem',
    sectionGap: '2rem',
    cardPadding: '1.5rem',
  },

  // Border radius
  borderRadius: {
    card: '0.75rem',
    button: '0.5rem',
    input: '0.5rem',
  },

  // Animation durations
  animation: {
    fast: '150ms',
    normal: '200ms',
    slow: '300ms',
  },
} as const;

// Type definitions for better TypeScript support
export type ThemeColors = typeof theme.colors;
export type StatusColors = typeof theme.colors.status;
export type WorkflowColors = typeof theme.colors.workflow;
export type ResourceColors = typeof theme.colors.resource;

// Utility function to get status colors
export function getStatusColor(status: keyof StatusColors): string {
  return theme.colors.status[status];
}

// Utility function to get workflow status colors
export function getWorkflowStatusColor(status: keyof WorkflowColors): string {
  return theme.colors.workflow[status];
}

// Utility function to get resource status colors
export function getResourceStatusColor(status: keyof ResourceColors): string {
  return theme.colors.resource[status];
}

// Utility function to get status badge class names
export function getStatusBadgeClass(status: string): string {
  switch (status.toLowerCase()) {
    case 'running':
      return 'workflow-running';
    case 'completed':
    case 'success':
      return 'workflow-completed';
    case 'failed':
    case 'error':
      return 'workflow-failed';
    case 'pending':
      return 'workflow-pending';
    case 'active':
      return 'status-success';
    case 'provisioning':
      return 'status-info';
    case 'degraded':
    case 'warning':
      return 'status-warning';
    case 'terminated':
      return 'workflow-pending';
    default:
      return 'workflow-pending';
  }
}

// Utility function to get appropriate icon for status
export function getStatusIconName(status: string): string {
  switch (status.toLowerCase()) {
    case 'running':
      return 'Zap';
    case 'completed':
    case 'success':
    case 'active':
      return 'CheckCircle';
    case 'failed':
    case 'error':
      return 'XCircle';
    case 'pending':
    case 'terminated':
      return 'Clock';
    case 'provisioning':
      return 'Loader';
    case 'degraded':
    case 'warning':
      return 'AlertTriangle';
    default:
      return 'Clock';
  }
}

// CSS-in-JS style generators for dynamic theming
export function generateCardGradient(type: keyof typeof theme.colors.cards): string {
  const cardColors = theme.colors.cards[type];
  return `bg-gradient-to-br from-${cardColors.from} to-${cardColors.to}`;
}

export function generateBackgroundGradient(): string {
  const bg = theme.colors.background;
  return `bg-gradient-to-br from-${bg.from} via-${bg.via} to-${bg.to}`;
}

// Accessibility helpers
export function getContrastTextColor(bgColor: string): 'text-white' | 'text-black' {
  // Simple heuristic for contrast - in a real app you'd use a proper contrast calculation
  const lightColors = ['yellow', 'lime', 'amber', 'orange', 'cyan'];
  const isLight = lightColors.some((color) => bgColor.includes(color));
  return isLight ? 'text-black' : 'text-white';
}

// Theme validation
export function validateThemeColors(colors: Record<string, string>): boolean {
  const requiredColors = ['primary', 'success', 'warning', 'error', 'info'];
  return requiredColors.every((color) => color in colors);
}
