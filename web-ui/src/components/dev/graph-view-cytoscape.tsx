'use client';

import { useState, useEffect, useRef } from 'react';
import CytoscapeComponent from 'react-cytoscapejs';
import Cytoscape from 'cytoscape';
import { api } from '@/lib/api';
import { getNodeColor } from '@/lib/graph-colors';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

// Lazy load and register dagre layout
let dagreRegistered = false;
async function initializeDagre() {
  if (!dagreRegistered && typeof window !== 'undefined') {
    try {
      const dagre = await import('cytoscape-dagre');
      // eslint-disable-next-line react-hooks/rules-of-hooks
      Cytoscape.use(dagre.default);
      dagreRegistered = true;
    } catch (error) {
      console.error('Failed to load cytoscape-dagre:', error);
    }
  }
}

interface GraphViewCytoscapeProps {
  applications: Array<{ name: string }>;
}

const LAYOUTS = [
  { value: 'dagre', label: 'Hierarchical (Dagre)' },
  { value: 'breadthfirst', label: 'Breadth First' },
  { value: 'circle', label: 'Circle' },
  { value: 'grid', label: 'Grid' },
  { value: 'cose', label: 'Force Directed (CoSE)' },
];

export function GraphViewCytoscape({ applications }: GraphViewCytoscapeProps) {
  const [selectedApp, setSelectedApp] = useState<string>('');
  const [elements, setElements] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [layout, setLayout] = useState('dagre');
  const cyRef = useRef<any>(null);

  // Register dagre layout on mount
  useEffect(() => {
    initializeDagre();
  }, []);

  async function loadGraph(appName: string) {
    setLoading(true);
    try {
      const response = await api.getResourceGraph(appName);
      if (response.success && response.data) {
        // Convert to Cytoscape format
        const cyElements = [
          ...response.data.nodes.map((node) => {
            const colors = getNodeColor(node.type, node.status);
            return {
              data: {
                id: node.id,
                label: node.name,
                type: node.type,
                status: node.status,
                color: colors.fill,
              },
            };
          }),
          ...response.data.edges.map((edge) => ({
            data: {
              id: edge.id,
              source: edge.source_id,
              target: edge.target_id,
              label: edge.relationship,
            },
          })),
        ];

        setElements(cyElements);
      }
    } catch (error) {
      console.error('Failed to load graph:', error);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (selectedApp) {
      loadGraph(selectedApp);
    }
  }, [selectedApp]);

  const handleLayoutChange = (newLayout: string) => {
    setLayout(newLayout);
    if (cyRef.current) {
      const cy = cyRef.current;
      cy.layout({ name: newLayout, animate: true }).run();
    }
  };

  const stylesheet = [
    {
      selector: 'node',
      style: {
        'background-color': 'data(color)',
        label: 'data(label)',
        color: '#fff',
        'text-valign': 'center',
        'text-halign': 'center',
        'font-size': '12px',
        width: 60,
        height: 60,
      },
    },
    {
      selector: 'edge',
      style: {
        width: 2,
        'line-color': '#a1a1aa',
        'target-arrow-color': '#a1a1aa',
        'target-arrow-shape': 'triangle',
        'curve-style': 'bezier',
        label: 'data(label)',
        'font-size': '10px',
        color: '#71717a',
      },
    },
  ];

  if (!selectedApp) {
    return (
      <div className="space-y-4">
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          Select an application to view its dependency graph with Cytoscape.js
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
          </div>
        </div>

        <Select value={layout} onValueChange={handleLayoutChange}>
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder="Select layout" />
          </SelectTrigger>
          <SelectContent>
            {LAYOUTS.map((l) => (
              <SelectItem key={l.value} value={l.value}>
                {l.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
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
        ) : elements.length === 0 ? (
          <div className="flex items-center justify-center h-full text-zinc-500">
            No graph data available for this application
          </div>
        ) : (
          <CytoscapeComponent
            elements={elements}
            style={{ width: '100%', height: '100%' }}
            layout={{ name: layout, animate: true }}
            stylesheet={stylesheet}
            cy={(cy) => {
              cyRef.current = cy;
            }}
          />
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
