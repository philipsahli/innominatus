'use client';

import React, { useEffect, useState, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  RefreshCw,
  Download,
  ArrowLeft,
  Search,
  Filter,
  X,
  Clock,
  MessageSquare,
  TrendingUp,
  Activity,
  List,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { GraphDiff } from '@/components/graph-diff';
import { GraphAnnotations } from '@/components/graph-annotations';
import PerformanceMetrics from '@/components/performance-metrics';
import { GraphTextView } from '@/components/graph-text-view';
import { WorkflowDetailsPane } from '@/components/workflow-details-pane';
import { api, WorkflowExecutionDetail } from '@/lib/api';
import { RotateCcw } from 'lucide-react';

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
  description?: string;
  metadata?: any;
}

interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

const getNodeColor = (type: string, status: string) => {
  if (status === 'failed') {
    return type === 'workflow' ? 'bg-red-500 border-red-600' : 'bg-red-500 border-red-600';
  }

  if (status === 'running') {
    switch (type) {
      case 'spec':
        return 'bg-blue-400 border-blue-500 animate-pulse';
      case 'workflow':
        return 'bg-yellow-400 border-yellow-500 animate-pulse';
      case 'step':
        return 'bg-orange-300 border-orange-400 animate-pulse';
      case 'resource':
        return 'bg-green-400 border-green-500 animate-pulse';
      default:
        return 'bg-gray-400 border-gray-500 animate-pulse';
    }
  }

  if (status === 'completed' || status === 'succeeded') {
    switch (type) {
      case 'spec':
        return 'bg-blue-600 border-blue-700';
      case 'workflow':
        return 'bg-yellow-600 border-yellow-700';
      case 'step':
        return 'bg-orange-500 border-orange-600';
      case 'resource':
        return 'bg-green-600 border-green-700';
      default:
        return 'bg-gray-600 border-gray-700';
    }
  }

  // Default/waiting/pending state
  switch (type) {
    case 'spec':
      return 'bg-blue-500 border-blue-600';
    case 'workflow':
      return 'bg-yellow-500 border-yellow-600';
    case 'step':
      return 'bg-orange-400 border-orange-500';
    case 'resource':
      return 'bg-green-500 border-green-600';
    default:
      return 'bg-gray-500 border-gray-600';
  }
};

export function GraphVisualization({ app }: { app: string }) {
  const router = useRouter();
  const [graph, setGraph] = useState<GraphData>({ nodes: [], edges: [] });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filter and search states
  const [showFilters, setShowFilters] = useState(false);
  const [showDiff, setShowDiff] = useState(false);
  const [showAnnotations, setShowAnnotations] = useState(false);
  const [showMetrics, setShowMetrics] = useState(false);
  const [selectedNode, setSelectedNode] = useState<{ id: string; name: string } | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [filters, setFilters] = useState({
    types: {
      spec: true,
      workflow: true,
      step: true,
      resource: true,
    },
    statuses: {
      succeeded: true,
      running: true,
      failed: true,
      waiting: true,
      pending: true,
    },
  });

  // Track node state changes for animations
  const [previousNodeStates, setPreviousNodeStates] = useState<Map<string, string>>(new Map());

  // Critical path state
  const [showCriticalPath, setShowCriticalPath] = useState(false);
  const [criticalPathNodes, setCriticalPathNodes] = useState<Set<string>>(new Set());

  // Details pane state
  const [showDetailsPane, setShowDetailsPane] = useState(false);
  const [selectedWorkflow, setSelectedWorkflow] = useState<WorkflowExecutionDetail | null>(null);
  const [loadingWorkflow, setLoadingWorkflow] = useState(false);

  const fetchGraph = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('auth-token');
      const response = await fetch(`http://localhost:8081/api/graph/${app}`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch graph: ${response.statusText}`);
      }

      const data = await response.json();
      setGraph(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load graph');
    } finally {
      setLoading(false);
    }
  }, [app]);

  const fetchCriticalPath = useCallback(async () => {
    try {
      const token = localStorage.getItem('auth-token');
      const response = await fetch(`http://localhost:8081/api/graph/${app}/critical-path`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch critical path: ${response.statusText}`);
      }

      const data = await response.json();
      const pathNodeIds = new Set<string>(data.path.map((n: any) => n.id));
      setCriticalPathNodes(pathNodeIds);
    } catch (err) {
      console.error('Failed to fetch critical path:', err);
      setCriticalPathNodes(new Set());
    }
  }, [app]);

  useEffect(() => {
    fetchGraph();
  }, [fetchGraph]);

  useEffect(() => {
    if (showCriticalPath) {
      fetchCriticalPath();
    } else {
      setCriticalPathNodes(new Set());
    }
  }, [showCriticalPath, fetchCriticalPath]);

  // Subscribe to WebSocket updates for real-time graph changes
  useEffect(() => {
    const token = localStorage.getItem('auth-token');
    const ws = new WebSocket(`ws://localhost:8081/api/graph/${app}/ws?token=${token}`);

    ws.onopen = () => {
      console.log('WebSocket connected for app:', app);
    };

    ws.onmessage = (event) => {
      try {
        const update = JSON.parse(event.data);
        // Full graph update from WebSocket
        if (update.nodes && update.edges) {
          setGraph(update);
        } else if (update.node) {
          // Partial node status update
          setGraph((prev) => ({
            ...prev,
            nodes: prev.nodes.map((n) =>
              n.id === update.node ? { ...n, status: update.status || update.state } : n
            ),
          }));
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
      console.log('WebSocket connection closed for app:', app);
    };

    return () => {
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
        ws.close();
      }
    };
  }, [app]);

  // Track node state changes for highlighting
  useEffect(() => {
    if (graph.nodes.length === 0) return;

    const newNodeStates = new Map<string, string>();
    graph.nodes.forEach((n) => {
      newNodeStates.set(n.id, n.status);
    });

    setPreviousNodeStates(newNodeStates);
  }, [graph.nodes]);

  const exportGraph = () => {
    const exportData = {
      app: app,
      graph: graph,
      timestamp: new Date().toISOString(),
    };

    const blob = new Blob([JSON.stringify(exportData, null, 2)], {
      type: 'application/json',
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${app}-graph-${Date.now()}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const clearFilters = () => {
    setFilters({
      types: {
        spec: true,
        workflow: true,
        step: true,
        resource: true,
      },
      statuses: {
        succeeded: true,
        running: true,
        failed: true,
        waiting: true,
        pending: true,
      },
    });
    setSearchTerm('');
  };

  const toggleFilter = (category: 'types' | 'statuses', key: string) => {
    setFilters((prev) => ({
      ...prev,
      [category]: {
        ...prev[category],
        [key]: !prev[category][key as keyof (typeof prev)[typeof category]],
      },
    }));
  };

  const handleNodeClick = async (node: GraphNode) => {
    // Only fetch workflow details for workflow nodes
    if (node.type !== 'workflow') {
      return;
    }

    // Extract workflow ID from node metadata if available
    const workflowId = node.metadata?.workflow_execution_id || node.id.split('-')[1];
    if (!workflowId) {
      console.error('No workflow ID found for node:', node);
      return;
    }

    setLoadingWorkflow(true);
    setShowDetailsPane(true);

    try {
      const response = await api.getWorkflowDetailsForGraph(app, workflowId);
      if (response.success && response.data) {
        setSelectedWorkflow(response.data);
      } else {
        console.error('Failed to fetch workflow details:', response.error);
        setSelectedWorkflow(null);
      }
    } catch (error) {
      console.error('Error fetching workflow details:', error);
      setSelectedWorkflow(null);
    } finally {
      setLoadingWorkflow(false);
    }
  };

  const handleRetry = async (workflowId: number) => {
    if (!selectedWorkflow) return;

    if (!confirm(`Retry workflow "${selectedWorkflow.workflow_name}" from the failed step?`)) {
      return;
    }

    setLoadingWorkflow(true);
    try {
      // Call retry API without workflow body - reconstructs from database
      const response = await api.retryWorkflow(workflowId.toString());

      if (response.success) {
        alert(
          'Workflow retry started successfully!\n\nThe workflow will retry from the failed step.'
        );

        // Wait a moment for the retry to start, then refresh workflow details
        setTimeout(async () => {
          const updated = await api.getWorkflowDetailsForGraph(app, workflowId.toString());
          if (updated.success && updated.data) {
            setSelectedWorkflow(updated.data);
          }
          setLoadingWorkflow(false);
        }, 1000);
      } else {
        alert(`Failed to retry workflow: ${response.error || 'Unknown error'}`);
        setLoadingWorkflow(false);
      }
    } catch (error) {
      alert(`Error retrying workflow: ${error}`);
      setLoadingWorkflow(false);
    }
  };

  const handleCloseDetailsPane = () => {
    setShowDetailsPane(false);
    setSelectedWorkflow(null);
  };

  return (
    <div className="flex flex-col h-screen">
      {/* Header */}
      <div className="p-4 border-b bg-white dark:bg-gray-800">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-4">
            <Button
              variant="outline"
              size="sm"
              onClick={() => router.push('/graph')}
              className="gap-2"
            >
              <ArrowLeft className="w-4 h-4" />
              Back
            </Button>
            <div>
              <h1 className="text-2xl font-bold">Workflow Graph: {app}</h1>
              <p className="text-sm text-muted-foreground">
                Real-time visualization of orchestration flow
              </p>
            </div>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => router.push(`/workflows?app=${app}`)}
              className="gap-2"
            >
              <List className="w-4 h-4" />
              Workflows
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowFilters(!showFilters)}
              className="gap-2"
            >
              <Filter className="w-4 h-4" />
              Filters
              {Object.values(filters.types).some((v) => !v) ||
              Object.values(filters.statuses).some((v) => !v) ? (
                <Badge variant="secondary" className="ml-1 px-1.5 py-0.5 text-xs">
                  Active
                </Badge>
              ) : null}
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowDiff(!showDiff)}
              className="gap-2"
            >
              <Clock className="w-4 h-4" />
              History
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowAnnotations(!showAnnotations)}
              className="gap-2"
            >
              <MessageSquare className="w-4 h-4" />
              Notes
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowMetrics(!showMetrics)}
              className="gap-2"
            >
              <Activity className="w-4 h-4" />
              Metrics
            </Button>
            <Button
              variant={showCriticalPath ? 'default' : 'outline'}
              size="sm"
              onClick={() => setShowCriticalPath(!showCriticalPath)}
              className="gap-2"
            >
              <TrendingUp className="w-4 h-4" />
              Critical Path
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={fetchGraph}
              disabled={loading}
              className="gap-2"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={exportGraph}
              disabled={graph.nodes.length === 0}
              className="gap-2"
            >
              <Download className="w-4 h-4" />
              Export
            </Button>
          </div>
        </div>

        {/* Search Bar */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input
            type="text"
            placeholder="Search nodes by name..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-10 pr-10"
          />
          {searchTerm && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setSearchTerm('')}
              className="absolute right-2 top-1/2 transform -translate-y-1/2 h-6 w-6 p-0"
            >
              <X className="w-4 h-4" />
            </Button>
          )}
        </div>
      </div>

      {/* Filter Panel */}
      {showFilters && (
        <div className="p-4 border-b bg-gray-50 dark:bg-gray-900">
          <div className="flex items-start gap-8">
            <div>
              <h3 className="font-semibold text-sm mb-3">Node Types</h3>
              <div className="space-y-2">
                {Object.keys(filters.types).map((type) => (
                  <div key={type} className="flex items-center gap-2">
                    <Checkbox
                      id={`type-${type}`}
                      checked={filters.types[type as keyof typeof filters.types]}
                      onCheckedChange={() => toggleFilter('types', type)}
                    />
                    <label htmlFor={`type-${type}`} className="text-sm capitalize cursor-pointer">
                      {type}
                    </label>
                  </div>
                ))}
              </div>
            </div>

            <div>
              <h3 className="font-semibold text-sm mb-3">Node Status</h3>
              <div className="space-y-2">
                {Object.keys(filters.statuses).map((status) => (
                  <div key={status} className="flex items-center gap-2">
                    <Checkbox
                      id={`status-${status}`}
                      checked={filters.statuses[status as keyof typeof filters.statuses]}
                      onCheckedChange={() => toggleFilter('statuses', status)}
                    />
                    <label
                      htmlFor={`status-${status}`}
                      className="text-sm capitalize cursor-pointer"
                    >
                      {status}
                    </label>
                  </div>
                ))}
              </div>
            </div>

            <div className="ml-auto">
              <Button variant="outline" size="sm" onClick={clearFilters} className="gap-2">
                <X className="w-4 h-4" />
                Clear All Filters
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* History/Diff Panel */}
      {showDiff && (
        <div className="p-4 border-b">
          <GraphDiff app={app} onClose={() => setShowDiff(false)} />
        </div>
      )}

      {/* Annotations Panel */}
      {showAnnotations && (
        <div className="p-4 border-b">
          <GraphAnnotations
            app={app}
            selectedNodeId={selectedNode?.id}
            selectedNodeName={selectedNode?.name}
            onClose={() => setShowAnnotations(false)}
          />
        </div>
      )}

      {/* Metrics Panel */}
      {showMetrics && (
        <div className="p-4 border-b">
          <PerformanceMetrics app={app} onClose={() => setShowMetrics(false)} />
        </div>
      )}

      {/* Graph Area with Split Pane */}
      <div className="flex-1 flex overflow-hidden">
        {/* Main Graph Area */}
        <div
          className={`flex-1 bg-gray-50 dark:bg-gray-900 ${showDetailsPane ? 'w-2/3' : 'w-full'} transition-all`}
        >
          {error ? (
            <div className="flex items-center justify-center h-full">
              <Card className="max-w-md">
                <CardHeader>
                  <CardTitle className="text-red-600">Error Loading Graph</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-muted-foreground">{error}</p>
                  <Button onClick={fetchGraph} className="mt-4" size="sm">
                    Retry
                  </Button>
                </CardContent>
              </Card>
            </div>
          ) : loading ? (
            <div className="flex items-center justify-center h-full">
              <div className="flex flex-col items-center gap-4">
                <RefreshCw className="w-8 h-8 animate-spin text-blue-500" />
                <p className="text-muted-foreground">Loading graph...</p>
              </div>
            </div>
          ) : graph.nodes.length === 0 ? (
            <div className="flex items-center justify-center h-full">
              <Card className="max-w-md">
                <CardHeader>
                  <CardTitle>No Graph Data</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-muted-foreground">
                    No workflow graph data available for this application yet.
                  </p>
                </CardContent>
              </Card>
            </div>
          ) : (
            <GraphTextView
              nodes={graph.nodes}
              edges={graph.edges}
              searchTerm={searchTerm}
              criticalPathNodes={criticalPathNodes}
              changedNodes={
                previousNodeStates.size > 0
                  ? new Set(
                      Array.from(previousNodeStates.entries())
                        .filter(([id, oldStatus]) => {
                          const node = graph.nodes.find((n) => n.id === id);
                          return node && node.status !== oldStatus;
                        })
                        .map(([id]) => id)
                    )
                  : new Set()
              }
              filters={filters}
              onNodeClick={handleNodeClick}
            />
          )}
        </div>

        {/* Details Pane */}
        {showDetailsPane && (
          <div className="w-1/3 h-full overflow-hidden">
            {loadingWorkflow ? (
              <div className="flex items-center justify-center h-full bg-white dark:bg-gray-800 border-l">
                <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
              </div>
            ) : (
              <WorkflowDetailsPane
                workflow={selectedWorkflow}
                onClose={handleCloseDetailsPane}
                onRetry={handleRetry}
              />
            )}
          </div>
        )}
      </div>

      {/* Legend */}
      {graph.nodes.length > 0 && (
        <div className="p-4 border-t bg-white dark:bg-gray-800">
          <div className="flex items-center gap-6 text-sm">
            <span className="font-semibold">Legend:</span>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 bg-blue-500 border-2 border-blue-600 rounded"></div>
              <span>Spec</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 bg-yellow-500 border-2 border-yellow-600 rounded"></div>
              <span>Workflow</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 bg-orange-400 border-2 border-orange-500 rounded"></div>
              <span>Step</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 bg-green-500 border-2 border-green-600 rounded"></div>
              <span>Resource</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 bg-red-500 border-2 border-red-600 rounded"></div>
              <span>Failed</span>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="outline" className="animate-pulse">
                Pulse
              </Badge>
              <span>Running</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
