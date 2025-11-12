'use client';

import { useState, useEffect, useRef } from 'react';
import * as d3 from 'd3';
import { api, type GraphNode as ApiGraphNode, type GraphEdge } from '@/lib/api';
import { getNodeColor } from '@/lib/graph-colors';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface GraphNode extends d3.SimulationNodeDatum, ApiGraphNode {}

interface GraphViewD3Props {
  applications: Array<{ name: string }>;
}

export function GraphViewD3({ applications }: GraphViewD3Props) {
  const [selectedApp, setSelectedApp] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const svgRef = useRef<SVGSVGElement>(null);
  const [nodeCount, setNodeCount] = useState(0);
  const [edgeCount, setEdgeCount] = useState(0);

  async function loadGraph(appName: string) {
    setLoading(true);
    try {
      const response = await api.getResourceGraph(appName);
      if (response.success && response.data) {
        const nodes: GraphNode[] = response.data.nodes.map((node) => ({
          ...node,
        }));
        const edges = response.data.edges;

        setNodeCount(nodes.length);
        setEdgeCount(edges.length);

        // Wait for loading to complete and SVG to render
        setTimeout(() => {
          setLoading(false);
          // Then wait another tick to ensure SVG is visible and has dimensions
          setTimeout(() => {
            if (svgRef.current) {
              console.log(
                'D3: Rendering graph with',
                nodes.length,
                'nodes and',
                edges.length,
                'edges'
              );
              console.log(
                'D3: SVG dimensions:',
                svgRef.current.clientWidth,
                'x',
                svgRef.current.clientHeight
              );
              renderGraph(nodes, edges);
            }
          }, 100);
        }, 0);
      }
    } catch (error) {
      console.error('Failed to load graph:', error);
      setLoading(false);
    }
  }

  function renderGraph(nodes: GraphNode[], edges: GraphEdge[]) {
    if (!svgRef.current) {
      console.log('D3: SVG ref is null');
      return;
    }

    const svg = d3.select(svgRef.current);
    svg.selectAll('*').remove();

    const width = svgRef.current.clientWidth;
    const height = svgRef.current.clientHeight;

    console.log('D3: Starting render with dimensions', width, 'x', height);

    if (width === 0 || height === 0) {
      console.error('D3: SVG has zero dimensions, cannot render');
      return;
    }

    // Create zoom behavior
    const zoom = d3
      .zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.1, 4])
      .on('zoom', (event) => {
        g.attr('transform', event.transform);
      });

    svg.call(zoom);

    const g = svg.append('g');

    // Convert edges to D3 format (source/target instead of source_id/target_id)
    const d3Edges = edges.map((edge) => ({
      ...edge,
      source: edge.source_id,
      target: edge.target_id,
    }));

    // Create force simulation
    const simulation = d3
      .forceSimulation(nodes)
      .force(
        'link',
        d3
          .forceLink(d3Edges)
          .id((d: any) => d.id)
          .distance(150)
      )
      .force('charge', d3.forceManyBody().strength(-400))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide().radius(50));

    // Draw edges
    const link = g
      .append('g')
      .selectAll('line')
      .data(d3Edges)
      .enter()
      .append('line')
      .attr('stroke', '#a1a1aa')
      .attr('stroke-width', 2)
      .attr('marker-end', 'url(#arrowhead)');

    // Arrow marker
    svg
      .append('defs')
      .append('marker')
      .attr('id', 'arrowhead')
      .attr('markerWidth', 10)
      .attr('markerHeight', 10)
      .attr('refX', 35)
      .attr('refY', 3)
      .attr('orient', 'auto')
      .append('polygon')
      .attr('points', '0 0, 10 3, 0 6')
      .attr('fill', '#a1a1aa');

    // Draw nodes
    const node = g
      .append('g')
      .selectAll('g')
      .data(nodes)
      .enter()
      .append('g')
      .call(
        d3
          .drag<SVGGElement, GraphNode>()
          .on('start', dragstarted)
          .on('drag', dragged)
          .on('end', dragended)
      );

    node
      .append('circle')
      .attr('r', 30)
      .attr('fill', (d) => getNodeColor(d.type, d.status).fill)
      .attr('stroke', (d) => getNodeColor(d.type, d.status).stroke)
      .attr('stroke-width', 3)
      .style('cursor', 'move');

    node
      .append('text')
      .text((d) => (d.name.length > 15 ? d.name.substring(0, 15) + '...' : d.name))
      .attr('text-anchor', 'middle')
      .attr('dy', 45)
      .attr('class', 'text-xs')
      .attr('fill', 'currentColor')
      .style('pointer-events', 'none');

    // Update positions on simulation tick
    simulation.on('tick', () => {
      link
        .attr('x1', (d: any) => d.source.x)
        .attr('y1', (d: any) => d.source.y)
        .attr('x2', (d: any) => d.target.x)
        .attr('y2', (d: any) => d.target.y);

      node.attr('transform', (d) => `translate(${d.x},${d.y})`);
    });

    console.log('D3: Simulation started, nodes and edges rendered');

    // Drag functions
    function dragstarted(event: any, d: GraphNode) {
      if (!event.active) simulation.alphaTarget(0.3).restart();
      d.fx = d.x;
      d.fy = d.y;
    }

    function dragged(event: any, d: GraphNode) {
      d.fx = event.x;
      d.fy = event.y;
    }

    function dragended(event: any, d: GraphNode) {
      if (!event.active) simulation.alphaTarget(0);
      d.fx = null;
      d.fy = null;
    }

    // Initial zoom to fit (after simulation has run for a bit)
    setTimeout(() => {
      try {
        const bounds = g.node()?.getBBox();
        if (bounds && bounds.width > 0 && bounds.height > 0) {
          const fullWidth = bounds.width;
          const fullHeight = bounds.height;
          const midX = bounds.x + fullWidth / 2;
          const midY = bounds.y + fullHeight / 2;

          const scale = 0.8 / Math.max(fullWidth / width, fullHeight / height);
          const translate = [width / 2 - scale * midX, height / 2 - scale * midY];

          svg
            .transition()
            .duration(750)
            .call(
              zoom.transform as any,
              d3.zoomIdentity.translate(translate[0], translate[1]).scale(scale)
            );
        }
      } catch (error) {
        console.log('Could not zoom to fit:', error);
      }
    }, 500);
  }

  useEffect(() => {
    if (selectedApp) {
      loadGraph(selectedApp);
    }
  }, [selectedApp]);

  if (!selectedApp) {
    return (
      <div className="space-y-4">
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          Select an application to view its dependency graph with D3-Force
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
            {nodeCount > 0 && (
              <span className="ml-2 text-xs">
                ({nodeCount} nodes, {edgeCount} edges)
              </span>
            )}
          </div>
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
        ) : (
          <>
            <svg ref={svgRef} className="w-full h-full" />
            <div className="absolute top-4 right-4 bg-white dark:bg-zinc-900 p-3 rounded-lg border border-zinc-200 dark:border-zinc-800 text-xs">
              <div className="font-medium mb-1 text-zinc-900 dark:text-white">D3-Force</div>
              <div className="text-zinc-500 space-y-1">
                <div>• Scroll to zoom</div>
                <div>• Drag background to pan</div>
                <div>• Drag nodes to position</div>
              </div>
            </div>
          </>
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
