'use client';

import { useCallback, useEffect, useState } from 'react';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  Edge,
  Node,
  MarkerType,
  Panel,
  BackgroundVariant,
} from 'reactflow';
import 'reactflow/dist/style.css';
import ELK from 'elkjs/lib/elk.bundled.js';
import { api } from '@/lib/api';
import { getNodeColor } from '@/lib/graph-colors';
import { exportSVG, exportPNG, exportGraphJSON } from '@/lib/graph-export';
import { Loader2, Download, Maximize2, RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

interface GraphViewReactFlowProps {
  applications: Array<{ name: string }>;
  onNodeSelect?: (node: any) => void;
}

// ELK layout configuration
const elk = new ELK();
const elkOptions = {
  'elk.algorithm': 'layered',
  'elk.direction': 'DOWN',
  'elk.spacing.nodeNode': '100',
  'elk.layered.spacing.nodeNodeBetweenLayers': '150',
  'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
};

async function getLayoutedElements(nodes: Node[], edges: Edge[]) {
  if (nodes.length === 0) return { nodes: [], edges: [] };

  const graph = {
    id: 'root',
    layoutOptions: elkOptions,
    children: nodes.map((node) => ({
      id: node.id,
      width: 180,
      height: 80,
    })),
    edges: edges.map((edge) => ({
      id: edge.id,
      sources: [edge.source],
      targets: [edge.target],
    })),
  };

  const layoutedGraph = await elk.layout(graph);

  const layoutedNodes = nodes.map((node) => {
    const layoutedNode = layoutedGraph.children?.find((n) => n.id === node.id);
    return {
      ...node,
      position: {
        x: layoutedNode?.x ?? 0,
        y: layoutedNode?.y ?? 0,
      },
    };
  });

  return { nodes: layoutedNodes, edges };
}

export function GraphViewReactFlow({ applications, onNodeSelect }: GraphViewReactFlowProps) {
  const [selectedApp, setSelectedApp] = useState<string>('');
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [loading, setLoading] = useState(false);
  const [autoLayout, setAutoLayout] = useState(true);

  const loadGraph = useCallback(
    async (appName: string) => {
      setLoading(true);
      try {
        const response = await api.getResourceGraph(appName);
        if (response.success && response.data) {
          // Convert to ReactFlow format
          const rfNodes: Node[] = response.data.nodes.map((node) => {
            const colors = getNodeColor(node.type, node.status);

            return {
              id: node.id,
              type: 'default',
              position: { x: 0, y: 0 },
              data: {
                label: node.name,
                type: node.type,
                status: node.status,
                description: node.description,
                metadata: node.metadata,
              },
              style: {
                background: colors.fill,
                color: colors.text,
                border: `2px solid ${colors.stroke}`,
                borderRadius: '8px',
                padding: '12px',
                minWidth: '180px',
                boxShadow: colors.glow ? `0 0 15px ${colors.fill}` : '0 2px 4px rgba(0,0,0,0.1)',
              },
            };
          });

          const rfEdges: Edge[] = response.data.edges.map((edge) => ({
            id: edge.id,
            source: edge.source_id,
            target: edge.target_id,
            type: 'smoothstep',
            animated: false,
            markerEnd: {
              type: MarkerType.ArrowClosed,
              width: 20,
              height: 20,
            },
            label: edge.relationship,
            style: {
              stroke: '#a1a1aa',
              strokeWidth: 2,
            },
            labelStyle: {
              fill: '#71717a',
              fontSize: 10,
            },
          }));

          // Apply automatic layout if enabled
          if (autoLayout) {
            const { nodes: layoutedNodes, edges: layoutedEdges } = await getLayoutedElements(
              rfNodes,
              rfEdges
            );
            setNodes(layoutedNodes);
            setEdges(layoutedEdges);
          } else {
            setNodes(rfNodes);
            setEdges(rfEdges);
          }
        }
      } catch (error) {
        console.error('Failed to load graph:', error);
      } finally {
        setLoading(false);
      }
    },
    [autoLayout, setNodes, setEdges]
  );

  useEffect(() => {
    if (selectedApp) {
      loadGraph(selectedApp);
    }
  }, [selectedApp, loadGraph]);

  const handleExport = (format: 'svg' | 'png' | 'json') => {
    const filename = `${selectedApp}-graph.${format}`;

    if (format === 'json') {
      // Export raw graph data
      api
        .getResourceGraph(selectedApp)
        .then((response) => {
          if (response.success && response.data) {
            exportGraphJSON(response.data.nodes, response.data.edges, filename);
          }
        })
        .catch(console.error);
    } else {
      // Export visual representation
      const rfPane = document.querySelector('.react-flow__viewport') as SVGSVGElement | null;
      if (rfPane) {
        const svgElement = rfPane.closest('svg');
        if (svgElement) {
          if (format === 'svg') {
            exportSVG(svgElement, filename);
          } else if (format === 'png') {
            exportPNG(svgElement, filename, 2).catch(console.error);
          }
        }
      }
    }
  };

  const handleReLayout = async () => {
    if (nodes.length > 0) {
      const { nodes: layoutedNodes, edges: layoutedEdges } = await getLayoutedElements(
        nodes,
        edges
      );
      setNodes(layoutedNodes);
      setEdges(layoutedEdges);
    }
  };

  const handleNodeClick = useCallback(
    (event: React.MouseEvent, node: Node) => {
      const graphNode = {
        id: node.id,
        name: node.data.label,
        type: node.data.type,
        status: node.data.status,
        metadata: node.data.metadata || {},
      };
      onNodeSelect?.(graphNode);
    },
    [onNodeSelect]
  );

  if (!selectedApp) {
    return (
      <div className="space-y-4">
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          Select an application to view its dependency graph with ReactFlow
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
          <Button variant="outline" size="sm" onClick={handleReLayout}>
            <RefreshCw size={14} className="mr-1" />
            Re-Layout
          </Button>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm">
                <Download size={14} className="mr-1" />
                Export
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem onClick={() => handleExport('png')}>Export as PNG</DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleExport('svg')}>Export as SVG</DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleExport('json')}>
                Export as JSON
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* Graph Canvas */}
      <div className="relative border border-zinc-200 dark:border-zinc-800 rounded-lg overflow-hidden bg-white dark:bg-zinc-950 h-[700px]">
        {loading ? (
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <Loader2 className="w-8 h-8 animate-spin text-zinc-400 mx-auto mb-2" />
              <div className="text-sm text-zinc-500">Loading graph...</div>
            </div>
          </div>
        ) : nodes.length === 0 ? (
          <div className="flex items-center justify-center h-full text-zinc-500">
            No graph data available for this application
          </div>
        ) : (
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={handleNodeClick}
            fitView
            attributionPosition="bottom-left"
            minZoom={0.1}
            maxZoom={4}
            defaultEdgeOptions={{
              type: 'smoothstep',
              animated: false,
            }}
          >
            <Background
              color="#71717a"
              variant={BackgroundVariant.Dots}
              gap={16}
              size={1}
              className="dark:opacity-30"
            />
            <Controls showInteractive={false} />
            <MiniMap
              nodeColor={(node) => {
                const colors = getNodeColor(node.data.type, node.data.status);
                return colors.fill;
              }}
              maskColor="rgba(0, 0, 0, 0.1)"
              className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800"
            />

            <Panel
              position="top-right"
              className="bg-white dark:bg-zinc-900 p-2 rounded-lg border border-zinc-200 dark:border-zinc-800 text-xs"
            >
              <div className="font-medium mb-1 text-zinc-900 dark:text-white">ReactFlow</div>
              <div className="text-zinc-500 space-y-1">
                <div>• Drag to pan</div>
                <div>• Scroll to zoom</div>
                <div>• Drag nodes to reposition</div>
              </div>
            </Panel>
          </ReactFlow>
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
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full" style={{ backgroundColor: '#3b82f6' }} />
          <span>Step</span>
        </div>
      </div>
    </div>
  );
}
