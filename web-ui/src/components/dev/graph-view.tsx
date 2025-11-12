'use client';

import { useState, useEffect } from 'react';
import { api, type GraphNode as ApiGraphNode, type GraphEdge as ApiGraphEdge } from '@/lib/api';
import { Loader2, Maximize2, ZoomIn, ZoomOut } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface GraphNode extends ApiGraphNode {
  x?: number;
  y?: number;
}

type GraphEdge = ApiGraphEdge;

interface GraphViewProps {
  applications: Array<{ name: string }>;
}

export function GraphView({ applications }: GraphViewProps) {
  const [selectedApp, setSelectedApp] = useState<string>('');
  const [nodes, setNodes] = useState<GraphNode[]>([]);
  const [edges, setEdges] = useState<GraphEdge[]>([]);
  const [loading, setLoading] = useState(false);
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  useEffect(() => {
    if (selectedApp && selectedApp !== 'all') {
      loadGraph(selectedApp);
    }
  }, [selectedApp]);

  async function loadGraph(appName: string) {
    setLoading(true);
    try {
      const response = await api.getResourceGraph(appName);
      if (response.success && response.data) {
        const graphNodes = response.data.nodes || [];
        const graphEdges = response.data.edges || [];

        // Simple force-directed layout
        const positionedNodes = layoutNodes(graphNodes, graphEdges);
        setNodes(positionedNodes);
        setEdges(graphEdges);
      }
    } catch (error) {
      console.error('Failed to load graph:', error);
    } finally {
      setLoading(false);
    }
  }

  function layoutNodes(nodes: GraphNode[], edges: GraphEdge[]): GraphNode[] {
    const width = 900;
    const height = 700;
    const nodeSpacing = 150;
    const layerSpacing = 120;

    // Build adjacency map for topology-based layout
    const nodeMap = new Map(nodes.map((n) => [n.id, n]));
    const outgoing = new Map<string, string[]>();
    const incoming = new Map<string, number>();

    // Initialize maps
    nodes.forEach((node) => {
      outgoing.set(node.id, []);
      incoming.set(node.id, 0);
    });

    // Build graph structure
    edges.forEach((edge) => {
      const sources = outgoing.get(edge.source_id) || [];
      sources.push(edge.target_id);
      outgoing.set(edge.source_id, sources);
      incoming.set(edge.target_id, (incoming.get(edge.target_id) || 0) + 1);
    });

    // Topological sort to assign layers (BFS-based)
    const layers: string[][] = [];
    const nodeLayer = new Map<string, number>();
    const queue: Array<{ id: string; layer: number }> = [];

    // Start with nodes that have no incoming edges
    nodes.forEach((node) => {
      if ((incoming.get(node.id) || 0) === 0) {
        queue.push({ id: node.id, layer: 0 });
        nodeLayer.set(node.id, 0);
      }
    });

    // If no root nodes, start with all nodes at layer 0
    if (queue.length === 0) {
      nodes.forEach((node) => {
        queue.push({ id: node.id, layer: 0 });
        nodeLayer.set(node.id, 0);
      });
    }

    // Process queue
    const visited = new Set<string>();
    while (queue.length > 0) {
      const { id, layer } = queue.shift()!;
      if (visited.has(id)) continue;
      visited.add(id);

      // Add to layer
      if (!layers[layer]) layers[layer] = [];
      layers[layer].push(id);

      // Add children to next layer
      const children = outgoing.get(id) || [];
      children.forEach((childId) => {
        if (!visited.has(childId)) {
          const childLayer = Math.max(layer + 1, nodeLayer.get(childId) || 0);
          nodeLayer.set(childId, childLayer);
          queue.push({ id: childId, layer: childLayer });
        }
      });
    }

    // Add any unvisited nodes to final layer
    nodes.forEach((node) => {
      if (!visited.has(node.id)) {
        const lastLayer = layers.length;
        if (!layers[lastLayer]) layers[lastLayer] = [];
        layers[lastLayer].push(node.id);
      }
    });

    // Position nodes based on layers
    const positioned: GraphNode[] = [];
    layers.forEach((layerNodes, layerIndex) => {
      const y = layerIndex * layerSpacing + 80;
      const layerWidth = layerNodes.length * nodeSpacing;
      const startX = (width - layerWidth) / 2;

      layerNodes.forEach((nodeId, nodeIndex) => {
        const node = nodeMap.get(nodeId);
        if (node) {
          positioned.push({
            ...node,
            x: startX + nodeIndex * nodeSpacing + nodeSpacing / 2,
            y: y,
          });
        }
      });
    });

    return positioned;
  }

  function getNodeColor(type: string, status?: string) {
    if (status === 'failed') return '#ef4444';
    if (status === 'running') return '#3b82f6';

    switch (type) {
      case 'spec':
        return '#8b5cf6';
      case 'resource':
        return '#10b981';
      case 'provider':
        return '#f59e0b';
      case 'workflow':
        return '#06b6d4';
      default:
        return '#6b7280';
    }
  }

  function handleWheel(e: React.WheelEvent) {
    e.preventDefault();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    setZoom((z) => Math.max(0.1, Math.min(3, z * delta)));
  }

  function handleMouseDown(e: React.MouseEvent) {
    setIsDragging(true);
    setDragStart({ x: e.clientX - pan.x, y: e.clientY - pan.y });
  }

  function handleMouseMove(e: React.MouseEvent) {
    if (isDragging) {
      setPan({
        x: e.clientX - dragStart.x,
        y: e.clientY - dragStart.y,
      });
    }
  }

  function handleMouseUp() {
    setIsDragging(false);
  }

  if (!selectedApp || selectedApp === 'all') {
    return (
      <div className="space-y-4">
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          Select an application to view its dependency graph
        </p>

        <div className="space-y-2">
          {applications
            .filter((app) => app.name && app.name.trim() !== '')
            .map((app) => (
              <button
                key={app.name}
                onClick={() => setSelectedApp(app.name)}
                className="block w-full rounded-lg border border-zinc-200 dark:border-zinc-800 px-4 py-3 text-left text-sm hover:bg-zinc-50 dark:hover:bg-zinc-900 transition-colors"
              >
                <div className="font-medium text-zinc-900 dark:text-white">{app.name}</div>
              </button>
            ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Controls */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setSelectedApp('')}
            className="text-xs"
          >
            ← Back
          </Button>
          <div className="text-sm text-zinc-600 dark:text-zinc-400">
            Viewing:{' '}
            <span className="font-medium text-zinc-900 dark:text-white">{selectedApp}</span>
            {nodes.length > 0 && (
              <span className="ml-2 text-xs">
                ({nodes.length} nodes, {edges.length} edges)
              </span>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => setZoom((z) => Math.min(3, z * 1.2))}>
            <ZoomIn size={14} />
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setZoom((z) => Math.max(0.1, z * 0.8))}
          >
            <ZoomOut size={14} />
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setZoom(1);
              setPan({ x: 0, y: 0 });
            }}
          >
            <Maximize2 size={14} />
          </Button>
        </div>
      </div>

      {/* Graph Canvas */}
      <div className="relative border border-zinc-200 dark:border-zinc-800 rounded-lg overflow-hidden bg-white dark:bg-zinc-950">
        {loading ? (
          <div className="flex items-center justify-center h-[600px]">
            <Loader2 className="w-8 h-8 animate-spin text-zinc-400" />
          </div>
        ) : nodes.length === 0 ? (
          <div className="flex items-center justify-center h-[600px] text-zinc-500">
            No graph data available for this application
          </div>
        ) : (
          <svg
            width="100%"
            height="600"
            onWheel={handleWheel}
            onMouseDown={handleMouseDown}
            onMouseMove={handleMouseMove}
            onMouseUp={handleMouseUp}
            onMouseLeave={handleMouseUp}
            className="cursor-move"
          >
            <g transform={`translate(${pan.x}, ${pan.y}) scale(${zoom})`}>
              {/* Draw edges */}
              {edges.map((edge) => {
                const source = nodes.find((n) => n.id === edge.source_id);
                const target = nodes.find((n) => n.id === edge.target_id);
                if (!source || !target || !source.x || !source.y || !target.x || !target.y)
                  return null;

                const midX = (source.x + target.x) / 2;
                const midY = (source.y + target.y) / 2;

                return (
                  <g key={edge.id}>
                    <line
                      x1={source.x}
                      y1={source.y}
                      x2={target.x}
                      y2={target.y}
                      stroke="#a1a1aa"
                      strokeWidth="2"
                      markerEnd="url(#arrowhead)"
                      className="transition-all"
                    />
                    {/* Edge label */}
                    {edge.relationship && (
                      <text
                        x={midX}
                        y={midY - 5}
                        textAnchor="middle"
                        className="text-[10px] fill-zinc-600 dark:fill-zinc-400"
                        style={{ pointerEvents: 'none' }}
                      >
                        {edge.relationship}
                      </text>
                    )}
                  </g>
                );
              })}

              {/* Arrow marker */}
              <defs>
                <marker
                  id="arrowhead"
                  markerWidth="10"
                  markerHeight="10"
                  refX="9"
                  refY="3"
                  orient="auto"
                >
                  <polygon points="0 0, 10 3, 0 6" fill="#a1a1aa" />
                </marker>
              </defs>

              {/* Draw nodes */}
              {nodes.map((node) => {
                if (!node.x || !node.y) return null;
                const color = getNodeColor(node.type, node.status);

                return (
                  <g key={node.id}>
                    {/* Node circle */}
                    <circle
                      cx={node.x}
                      cy={node.y}
                      r="30"
                      fill={color}
                      stroke="#fff"
                      strokeWidth="3"
                      className="cursor-pointer hover:opacity-80 transition-opacity"
                    />

                    {/* Node label */}
                    <text
                      x={node.x}
                      y={node.y + 50}
                      textAnchor="middle"
                      className="text-xs fill-zinc-900 dark:fill-white font-medium"
                      style={{ pointerEvents: 'none' }}
                    >
                      {node.name.length > 20 ? `${node.name.substring(0, 20)}...` : node.name}
                    </text>

                    {/* Node type */}
                    <text
                      x={node.x}
                      y={node.y + 65}
                      textAnchor="middle"
                      className="text-[10px] fill-zinc-500 dark:fill-zinc-400"
                      style={{ pointerEvents: 'none' }}
                    >
                      {node.type}
                    </text>
                  </g>
                );
              })}
            </g>
          </svg>
        )}
      </div>

      {/* Legend */}
      <div className="flex items-center gap-6 text-xs text-zinc-600 dark:text-zinc-400">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full" style={{ backgroundColor: '#8b5cf6' }} />
          <span>Spec</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full" style={{ backgroundColor: '#10b981' }} />
          <span>Resource</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full" style={{ backgroundColor: '#f59e0b' }} />
          <span>Provider</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full" style={{ backgroundColor: '#06b6d4' }} />
          <span>Workflow</span>
        </div>
      </div>
    </div>
  );
}
