'use client';

import React from 'react';
import { Badge } from '@/components/ui/badge';

interface GraphNode {
  id: string;
  name: string;
  type: string;
  status: string;
  description?: string;
  metadata?: any;
  step_number?: number;
  total_steps?: number;
  workflow_id?: number;
  duration_ms?: number;
  execution_order?: number;
  created_at?: string;
  updated_at?: string;
}

interface GraphEdge {
  id: string;
  source_id: string;
  target_id: string;
  type: string;
}

interface GraphTextViewProps {
  nodes: GraphNode[];
  edges: GraphEdge[];
  searchTerm?: string;
  criticalPathNodes?: Set<string>;
  changedNodes?: Set<string>;
  filters?: {
    types: Record<string, boolean>;
    statuses: Record<string, boolean>;
  };
  onNodeClick?: (node: GraphNode) => void;
}

interface TreeNode {
  node: GraphNode;
  children: TreeNode[];
  level: number;
}

const getNodeIcon = (type: string): string => {
  switch (type) {
    case 'spec':
      return '📦';
    case 'workflow':
      return '📋';
    case 'step':
      return '🔧';
    case 'resource':
      return '💾';
    default:
      return '📄';
  }
};

const getStatusBadgeVariant = (
  status: string
): 'default' | 'secondary' | 'destructive' | 'outline' => {
  switch (status.toLowerCase()) {
    case 'completed':
    case 'succeeded':
      return 'default';
    case 'running':
      return 'secondary';
    case 'failed':
      return 'destructive';
    default:
      return 'outline';
  }
};

const getStatusColor = (status: string): string => {
  switch (status.toLowerCase()) {
    case 'completed':
    case 'succeeded':
      return 'text-green-600 dark:text-green-400';
    case 'running':
      return 'text-yellow-600 dark:text-yellow-400 animate-pulse';
    case 'failed':
      return 'text-red-600 dark:text-red-400';
    default:
      return 'text-gray-500 dark:text-gray-400';
  }
};

export function GraphTextView({
  nodes,
  edges,
  searchTerm = '',
  criticalPathNodes = new Set(),
  changedNodes = new Set(),
  filters,
  onNodeClick,
}: GraphTextViewProps) {
  // Build tree structure from graph data
  const buildTree = (): TreeNode[] => {
    // Filter nodes based on filters
    let filteredNodes = nodes;
    if (filters) {
      filteredNodes = nodes.filter((n) => {
        const typeMatch = filters.types[n.type] !== false;
        const statusMatch = filters.statuses[n.status] !== false;
        return typeMatch && statusMatch;
      });
    }

    // Create node lookup map
    const nodeMap = new Map<string, TreeNode>();
    filteredNodes.forEach((node) => {
      nodeMap.set(node.id, { node, children: [], level: 0 });
    });

    // Build parent-child relationships from edges
    const roots: TreeNode[] = [];
    const childrenSet = new Set<string>();

    edges.forEach((edge) => {
      const parent = nodeMap.get(edge.source_id);
      const child = nodeMap.get(edge.target_id);

      if (parent && child && edge.type === 'contains') {
        parent.children.push(child);
        child.level = parent.level + 1;
        childrenSet.add(child.node.id);
      }
    });

    // Find root nodes (nodes with no parents)
    nodeMap.forEach((treeNode, id) => {
      if (!childrenSet.has(id)) {
        roots.push(treeNode);
      }
    });

    // Sort roots by type (spec first, then workflows)
    return roots.sort((a, b) => {
      const typeOrder = { spec: 0, workflow: 1, step: 2, resource: 3 };
      const aOrder = typeOrder[a.node.type as keyof typeof typeOrder] ?? 99;
      const bOrder = typeOrder[b.node.type as keyof typeof typeOrder] ?? 99;
      return aOrder - bOrder;
    });
  };

  const renderTreeNode = (
    treeNode: TreeNode,
    isLast: boolean,
    prefix: string = '',
    isRoot: boolean = false
  ): React.ReactNode => {
    const { node, children } = treeNode;
    const icon = getNodeIcon(node.type);
    const statusColor = getStatusColor(node.status);
    const isOnCriticalPath = criticalPathNodes.has(node.id);
    const hasChanged = changedNodes.has(node.id);
    const isSearchMatch = searchTerm && node.name.toLowerCase().includes(searchTerm.toLowerCase());

    // Build tree connector
    const connector = isRoot ? '' : isLast ? '└── ' : '├── ';
    const childPrefix = isRoot ? '' : isLast ? '    ' : '│   ';

    const isClickable = node.type === 'workflow' && onNodeClick;
    const nodeClasses = `
      ${isOnCriticalPath ? 'bg-purple-100 dark:bg-purple-900/30 font-bold' : ''}
      ${isSearchMatch ? 'bg-yellow-100 dark:bg-yellow-900/30' : ''}
      ${hasChanged ? 'bg-blue-100 dark:bg-blue-900/30 animate-pulse' : ''}
      ${isOnCriticalPath || isSearchMatch || hasChanged ? 'px-2 py-1 rounded' : ''}
      ${isClickable ? 'cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700' : ''}
    `.trim();

    const handleClick = () => {
      if (isClickable && onNodeClick) {
        onNodeClick(node);
      }
    };

    return (
      <div key={node.id} className="font-mono text-sm">
        <div className={`flex items-center gap-2 py-1 ${nodeClasses}`} onClick={handleClick}>
          <span className="text-gray-400 dark:text-gray-600 whitespace-pre">
            {prefix}
            {connector}
          </span>
          <span className="text-lg">{icon}</span>
          <span className={`font-medium ${statusColor}`}>{node.name}</span>
          <span className="text-xs text-gray-500">({node.type})</span>
          {node.step_number && node.total_steps && (
            <span className="text-xs text-gray-500">
              [Step {node.step_number}/{node.total_steps}]
            </span>
          )}
          {node.duration_ms && (
            <span className="text-xs text-gray-600 dark:text-gray-400">
              ⏱{' '}
              {node.duration_ms < 1000
                ? `${node.duration_ms}ms`
                : `${(node.duration_ms / 1000).toFixed(1)}s`}
            </span>
          )}
          {node.execution_order && (
            <span className="text-xs text-blue-600 dark:text-blue-400">
              #{node.execution_order}
            </span>
          )}
          <Badge variant={getStatusBadgeVariant(node.status)} className="text-xs">
            {node.status}
          </Badge>
          {isOnCriticalPath && <span className="text-purple-500 text-xs">⚡ Critical Path</span>}
          {hasChanged && <span className="text-blue-500 text-xs">🔄 Updated</span>}
          {node.status === 'failed' && <span className="text-red-500">⚠️</span>}
        </div>

        {children.length > 0 && (
          <div>
            {children.map((child, index) =>
              renderTreeNode(child, index === children.length - 1, prefix + childPrefix, false)
            )}
          </div>
        )}
      </div>
    );
  };

  const tree = buildTree();

  if (tree.length === 0) {
    return (
      <div className="text-center py-12 text-gray-500">
        <p>No nodes to display</p>
        {filters && <p className="text-sm mt-2">Try adjusting your filters</p>}
      </div>
    );
  }

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 overflow-auto">
      <div className="space-y-2">
        {tree.map((root, index) => renderTreeNode(root, index === tree.length - 1, '', true))}
      </div>

      {searchTerm && (
        <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Search: &quot;{searchTerm}&quot; -{' '}
            {nodes.filter((n) => n.name.toLowerCase().includes(searchTerm.toLowerCase())).length}{' '}
            match(es)
          </p>
        </div>
      )}
    </div>
  );
}
