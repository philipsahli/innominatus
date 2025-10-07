'use client';

import React, { useMemo } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Background,
  Controls,
  BackgroundVariant,
  Position,
} from 'reactflow';
import 'reactflow/dist/style.css';
import {
  GitBranch,
  Cloud,
  FileCode,
  Box,
  Workflow,
  GitCommit,
  Shield,
  Cog,
  CheckCircle,
  Circle,
} from 'lucide-react';
import { WorkflowStep, getStepStyle, getStepDescription } from '@/lib/workflow-types';

// Icon mapping
const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  GitBranch,
  Cloud,
  FileCode,
  Box,
  Workflow,
  GitCommit,
  Shield,
  Cog,
  CheckCircle,
  Circle,
};

interface WorkflowDiagramProps {
  steps: WorkflowStep[];
}

// Custom node component
function CustomNode({ data }: { data: any }) {
  const style = getStepStyle(data.type);
  const IconComponent = iconMap[style.icon] || Circle;
  const isConditional = !!data.condition;

  return (
    <div
      className={`px-4 py-3 rounded-lg border-2 ${style.bgColor} ${style.borderColor} ${
        isConditional ? 'border-dashed' : ''
      } min-w-[200px] shadow-sm hover:shadow-md transition-shadow`}
    >
      <div className="flex items-start gap-3">
        <IconComponent className={`w-5 h-5 ${style.color} mt-0.5 flex-shrink-0`} />
        <div className="flex-1 min-w-0">
          <div className="font-semibold text-sm text-gray-900 dark:text-gray-100 mb-1">
            {data.label}
          </div>
          <div className="text-xs text-gray-600 dark:text-gray-400">{data.description}</div>
          {isConditional && (
            <div className="mt-2 text-xs text-gray-500 dark:text-gray-500 font-mono">
              if: {data.condition}
            </div>
          )}
        </div>
      </div>
      <div className={`mt-2 text-xs font-mono px-2 py-1 rounded ${style.bgColor} ${style.color}`}>
        {data.type}
      </div>
    </div>
  );
}

const nodeTypes = {
  custom: CustomNode,
};

export function WorkflowDiagram({ steps }: WorkflowDiagramProps) {
  const { nodes, edges } = useMemo(() => {
    const nodes: Node[] = [];
    const edges: Edge[] = [];

    const VERTICAL_SPACING = 120;
    const HORIZONTAL_OFFSET = 50;

    steps.forEach((step, index) => {
      const style = getStepStyle(step.type);

      nodes.push({
        id: `step-${index}`,
        type: 'custom',
        position: {
          x: HORIZONTAL_OFFSET,
          y: index * VERTICAL_SPACING,
        },
        data: {
          label: step.name,
          type: step.type,
          description: getStepDescription(step),
          condition: step.condition || step.if,
        },
        sourcePosition: Position.Bottom,
        targetPosition: Position.Top,
      });

      // Create edge to next step
      if (index < steps.length - 1) {
        edges.push({
          id: `edge-${index}`,
          source: `step-${index}`,
          target: `step-${index + 1}`,
          type: 'smoothstep',
          animated: false,
          style: {
            stroke: '#6b7280',
            strokeWidth: 2,
          },
        });
      }
    });

    return { nodes, edges };
  }, [steps]);

  if (steps.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
        <p className="text-gray-500 dark:text-gray-400">No workflow steps defined</p>
      </div>
    );
  }

  return (
    <div className="h-[600px] bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        fitView
        attributionPosition="bottom-left"
        proOptions={{ hideAttribution: true }}
      >
        <Background variant={BackgroundVariant.Dots} gap={16} size={1} />
        <Controls />
      </ReactFlow>
    </div>
  );
}
