'use client';

import React, { useEffect, useState, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  RefreshCw,
  Download,
  ArrowLeft,
  Filter,
  X,
  Clock,
  MessageSquare,
  TrendingUp,
  Activity,
  List,
  ChevronDown,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import { Checkbox } from '@/components/ui/checkbox';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { GraphDiff } from '@/components/graph-diff';
import { GraphAnnotations } from '@/components/graph-annotations';
import PerformanceMetrics from '@/components/performance-metrics';
import { GraphTextView } from '@/components/graph-text-view';
import { WorkflowDetailsPane } from '@/components/workflow-details-pane';
import { ResourceDetailsPane } from '@/components/resource-details-pane';
import { SpecDetailsPane } from '@/components/spec-details-pane';
import { StepDetailsPane } from '@/components/step-details-pane';
import { SearchAutocomplete } from '@/components/search-autocomplete';
import { api, WorkflowExecutionDetail, ResourceInstance, GraphNode, GraphEdge } from '@/lib/api';
import { RotateCcw } from 'lucide-react';

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
      active: true,
      provisioning: true,
      terminated: true,
    },
    resourceTypes: {} as Record<string, boolean>,
    providers: {} as Record<string, boolean>,
    resourceStates: {} as Record<string, boolean>,
    healthStatuses: {} as Record<string, boolean>,
  });

  // Track node state changes for animations
  const [previousNodeStates, setPreviousNodeStates] = useState<Map<string, string>>(new Map());

  // Critical path state
  const [showCriticalPath, setShowCriticalPath] = useState(false);
  const [criticalPathNodes, setCriticalPathNodes] = useState<Set<string>>(new Set());

  // Details pane state
  const [showDetailsPane, setShowDetailsPane] = useState(false);
  const [selectedDetailType, setSelectedDetailType] = useState<
    'workflow' | 'resource' | 'spec' | 'step' | null
  >(null);
  const [selectedWorkflow, setSelectedWorkflow] = useState<WorkflowExecutionDetail | null>(null);
  const [selectedResource, setSelectedResource] = useState<ResourceInstance | null>(null);
  const [selectedSpec, setSelectedSpec] = useState<GraphNode | null>(null);
  const [selectedStep, setSelectedStep] = useState<GraphNode | null>(null);
  const [loadingDetails, setLoadingDetails] = useState(false);

  // Real-time update indicators
  const [wsConnected, setWsConnected] = useState(false);
  const [lastUpdateTime, setLastUpdateTime] = useState<Date | null>(null);
  const [showUpdateBadge, setShowUpdateBadge] = useState(false);

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

  // Extract unique resource types, providers, states, and health statuses from graph data
  useEffect(() => {
    if (graph.nodes.length === 0) return;

    const resourceTypes = new Set<string>();
    const providers = new Set<string>();
    const resourceStates = new Set<string>();
    const healthStatuses = new Set<string>();

    graph.nodes.forEach((node) => {
      // Extract resource type for resource nodes
      if (node.type === 'resource' && node.metadata?.resource_type) {
        resourceTypes.add(node.metadata.resource_type);
      }

      // Extract provider
      if (node.metadata?.provider_id) {
        providers.add(node.metadata.provider_id);
      }

      // Extract resource state (different from status)
      if (node.type === 'resource' && node.metadata?.state) {
        resourceStates.add(node.metadata.state);
      }

      // Extract health status
      if (node.metadata?.health_status) {
        healthStatuses.add(node.metadata.health_status);
      }
    });

    // Initialize filters with all options enabled
    const newResourceTypeFilters: Record<string, boolean> = {};
    resourceTypes.forEach((type) => {
      newResourceTypeFilters[type] = true;
    });

    const newProviderFilters: Record<string, boolean> = {};
    providers.forEach((provider) => {
      newProviderFilters[provider] = true;
    });

    const newResourceStateFilters: Record<string, boolean> = {};
    resourceStates.forEach((state) => {
      newResourceStateFilters[state] = true;
    });

    const newHealthStatusFilters: Record<string, boolean> = {};
    healthStatuses.forEach((status) => {
      newHealthStatusFilters[status] = true;
    });

    setFilters((prev) => ({
      ...prev,
      resourceTypes: newResourceTypeFilters,
      providers: newProviderFilters,
      resourceStates: newResourceStateFilters,
      healthStatuses: newHealthStatusFilters,
    }));
  }, [graph.nodes]);

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
      setWsConnected(true);
    };

    ws.onmessage = (event) => {
      try {
        const update = JSON.parse(event.data);

        // Show update indicator
        setLastUpdateTime(new Date());
        setShowUpdateBadge(true);

        // Hide update badge after 3 seconds
        setTimeout(() => setShowUpdateBadge(false), 3000);

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
      setWsConnected(false);
    };

    ws.onclose = () => {
      console.log('WebSocket connection closed for app:', app);
      setWsConnected(false);
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

  const exportGraph = async (format: 'json' | 'svg' | 'png' | 'dot' | 'mermaid') => {
    if (format === 'json') {
      // Client-side JSON export
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
    } else {
      // Server-side export for SVG, PNG, DOT, Mermaid
      try {
        const response = await fetch(`/api/graph/${app}/export?format=${format}`, {
          credentials: 'include',
        });

        if (!response.ok) {
          throw new Error(`Export failed: ${response.statusText}`);
        }

        const blob = await response.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;

        // Set appropriate file extension
        const extension = format === 'mermaid' ? 'mmd' : format;
        a.download = `${app}-graph-${Date.now()}.${extension}`;

        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
      } catch (error) {
        console.error('Export failed:', error);
        alert(`Failed to export graph as ${format.toUpperCase()}: ${error}`);
      }
    }
  };

  const clearFilters = () => {
    // Re-enable all filter options
    const allResourceTypes: Record<string, boolean> = {};
    Object.keys(filters.resourceTypes).forEach((type) => {
      allResourceTypes[type] = true;
    });

    const allProviders: Record<string, boolean> = {};
    Object.keys(filters.providers).forEach((provider) => {
      allProviders[provider] = true;
    });

    const allResourceStates: Record<string, boolean> = {};
    Object.keys(filters.resourceStates).forEach((state) => {
      allResourceStates[state] = true;
    });

    const allHealthStatuses: Record<string, boolean> = {};
    Object.keys(filters.healthStatuses).forEach((status) => {
      allHealthStatuses[status] = true;
    });

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
        active: true,
        provisioning: true,
        terminated: true,
      },
      resourceTypes: allResourceTypes,
      providers: allProviders,
      resourceStates: allResourceStates,
      healthStatuses: allHealthStatuses,
    });
    setSearchTerm('');
  };

  const toggleFilter = (
    category:
      | 'types'
      | 'statuses'
      | 'resourceTypes'
      | 'providers'
      | 'resourceStates'
      | 'healthStatuses',
    key: string
  ) => {
    setFilters((prev) => ({
      ...prev,
      [category]: {
        ...prev[category],
        [key]: !prev[category][key as keyof (typeof prev)[typeof category]],
      },
    }));
  };

  const handleNodeClick = async (node: GraphNode) => {
    setLoadingDetails(true);
    setShowDetailsPane(true);
    setSelectedDetailType(node.type as 'workflow' | 'resource' | 'spec' | 'step');

    try {
      switch (node.type) {
        case 'workflow': {
          // Extract workflow ID from node metadata if available
          const workflowId = node.metadata?.workflow_execution_id || node.id.split('-')[1];
          if (!workflowId) {
            console.error('No workflow ID found for node:', node);
            setSelectedWorkflow(null);
            break;
          }

          const response = await api.getWorkflowDetailsForGraph(app, workflowId);
          if (response.success && response.data) {
            setSelectedWorkflow(response.data);
          } else {
            console.error('Failed to fetch workflow details:', response.error);
            setSelectedWorkflow(null);
          }
          break;
        }

        case 'resource': {
          // Extract resource ID from node metadata or ID
          const resourceId = node.metadata?.resource_id || node.id.split('-')[1];
          if (!resourceId) {
            console.error('No resource ID found for node:', node);
            setSelectedResource(null);
            break;
          }

          const response = await api.getResource(resourceId);
          if (response.success && response.data) {
            setSelectedResource(response.data);
          } else {
            console.error('Failed to fetch resource details:', response.error);
            setSelectedResource(null);
          }
          break;
        }

        case 'spec': {
          // For spec nodes, we can display directly from the node data
          setSelectedSpec(node);
          break;
        }

        case 'step': {
          // For step nodes, we can display directly from the node data
          setSelectedStep(node);
          break;
        }

        default:
          console.warn('Unknown node type:', node.type);
      }
    } catch (error) {
      console.error('Error fetching node details:', error);
    } finally {
      setLoadingDetails(false);
    }
  };

  const handleRetry = async (workflowId: number) => {
    if (!selectedWorkflow) return;

    if (!confirm(`Retry workflow "${selectedWorkflow.workflow_name}" from the failed step?`)) {
      return;
    }

    setLoadingDetails(true);
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
          setLoadingDetails(false);
        }, 1000);
      } else {
        alert(`Failed to retry workflow: ${response.error || 'Unknown error'}`);
        setLoadingDetails(false);
      }
    } catch (error) {
      alert(`Error retrying workflow: ${error}`);
      setLoadingDetails(false);
    }
  };

  const handleCloseDetailsPane = () => {
    setShowDetailsPane(false);
    setSelectedDetailType(null);
    setSelectedWorkflow(null);
    setSelectedResource(null);
    setSelectedSpec(null);
    setSelectedStep(null);
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
              <div className="flex items-center gap-3">
                <h1 className="text-2xl font-bold">Workflow Graph: {app}</h1>
                {/* WebSocket Connection Status */}
                <div className="flex items-center gap-2">
                  <div
                    className={`w-2 h-2 rounded-full ${
                      wsConnected ? 'bg-green-500 animate-pulse' : 'bg-gray-400'
                    }`}
                    title={wsConnected ? 'Connected' : 'Disconnected'}
                  />
                  <span className="text-xs text-muted-foreground">
                    {wsConnected ? 'Live' : 'Offline'}
                  </span>
                </div>
                {/* Update Indicator */}
                {showUpdateBadge && (
                  <Badge variant="secondary" className="animate-bounce">
                    New updates
                  </Badge>
                )}
              </div>
              <div className="flex items-center gap-2">
                <p className="text-sm text-muted-foreground">
                  Real-time visualization of orchestration flow
                </p>
                {lastUpdateTime && (
                  <span className="text-xs text-muted-foreground">
                    • Updated {lastUpdateTime.toLocaleTimeString()}
                  </span>
                )}
              </div>
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
              Object.values(filters.statuses).some((v) => !v) ||
              Object.values(filters.resourceTypes).some((v) => !v) ||
              Object.values(filters.providers).some((v) => !v) ||
              Object.values(filters.resourceStates).some((v) => !v) ||
              Object.values(filters.healthStatuses).some((v) => !v) ? (
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
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={graph.nodes.length === 0}
                  className="gap-2"
                >
                  <Download className="w-4 h-4" />
                  Export
                  <ChevronDown className="w-3 h-3 ml-1" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => exportGraph('json')}>
                  Export as JSON
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => exportGraph('svg')}>
                  Export as SVG
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => exportGraph('png')}>
                  Export as PNG
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => exportGraph('dot')}>
                  Export as DOT (Graphviz)
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => exportGraph('mermaid')}>
                  Export as Mermaid
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        {/* Search Bar */}
        <SearchAutocomplete
          nodes={graph.nodes}
          value={searchTerm}
          onChange={setSearchTerm}
          onSelectNode={handleNodeClick}
          placeholder="Search nodes by name, type, or provider..."
        />
      </div>

      {/* Filter Panel */}
      {showFilters && (
        <div className="p-4 border-b bg-gray-50 dark:bg-gray-900 max-h-96 overflow-y-auto">
          <div className="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-6">
            {/* Node Types */}
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

            {/* Node Status */}
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

            {/* Resource Types */}
            {Object.keys(filters.resourceTypes).length > 0 && (
              <div>
                <h3 className="font-semibold text-sm mb-3">Resource Types</h3>
                <div className="space-y-2">
                  {Object.keys(filters.resourceTypes).map((type) => (
                    <div key={type} className="flex items-center gap-2">
                      <Checkbox
                        id={`resourceType-${type}`}
                        checked={filters.resourceTypes[type]}
                        onCheckedChange={() => toggleFilter('resourceTypes', type)}
                      />
                      <label htmlFor={`resourceType-${type}`} className="text-sm cursor-pointer">
                        {type}
                      </label>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Providers */}
            {Object.keys(filters.providers).length > 0 && (
              <div>
                <h3 className="font-semibold text-sm mb-3">Providers</h3>
                <div className="space-y-2">
                  {Object.keys(filters.providers).map((provider) => (
                    <div key={provider} className="flex items-center gap-2">
                      <Checkbox
                        id={`provider-${provider}`}
                        checked={filters.providers[provider]}
                        onCheckedChange={() => toggleFilter('providers', provider)}
                      />
                      <label htmlFor={`provider-${provider}`} className="text-sm cursor-pointer">
                        {provider}
                      </label>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Resource States */}
            {Object.keys(filters.resourceStates).length > 0 && (
              <div>
                <h3 className="font-semibold text-sm mb-3">Resource States</h3>
                <div className="space-y-2">
                  {Object.keys(filters.resourceStates).map((state) => (
                    <div key={state} className="flex items-center gap-2">
                      <Checkbox
                        id={`resourceState-${state}`}
                        checked={filters.resourceStates[state]}
                        onCheckedChange={() => toggleFilter('resourceStates', state)}
                      />
                      <label
                        htmlFor={`resourceState-${state}`}
                        className="text-sm capitalize cursor-pointer"
                      >
                        {state}
                      </label>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Health Statuses */}
            {Object.keys(filters.healthStatuses).length > 0 && (
              <div>
                <h3 className="font-semibold text-sm mb-3">Health Status</h3>
                <div className="space-y-2">
                  {Object.keys(filters.healthStatuses).map((status) => (
                    <div key={status} className="flex items-center gap-2">
                      <Checkbox
                        id={`healthStatus-${status}`}
                        checked={filters.healthStatuses[status]}
                        onCheckedChange={() => toggleFilter('healthStatuses', status)}
                      />
                      <label
                        htmlFor={`healthStatus-${status}`}
                        className="text-sm capitalize cursor-pointer"
                      >
                        {status}
                      </label>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Clear Filters Button */}
          <div className="mt-4 pt-4 border-t flex justify-end">
            <Button variant="outline" size="sm" onClick={clearFilters} className="gap-2">
              <X className="w-4 h-4" />
              Clear All Filters
            </Button>
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
            {loadingDetails ? (
              <div className="flex items-center justify-center h-full bg-white dark:bg-gray-800 border-l">
                <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
              </div>
            ) : (
              <>
                {selectedDetailType === 'workflow' && (
                  <WorkflowDetailsPane
                    workflow={selectedWorkflow}
                    onClose={handleCloseDetailsPane}
                    onRetry={handleRetry}
                  />
                )}
                {selectedDetailType === 'resource' && (
                  <ResourceDetailsPane
                    resource={selectedResource}
                    onClose={handleCloseDetailsPane}
                  />
                )}
                {selectedDetailType === 'spec' && (
                  <SpecDetailsPane
                    spec={selectedSpec}
                    edges={graph.edges}
                    allNodes={graph.nodes}
                    onClose={handleCloseDetailsPane}
                    onNavigateToNode={(nodeId) => {
                      const node = graph.nodes.find((n) => n.id === nodeId);
                      if (node) handleNodeClick(node);
                    }}
                  />
                )}
                {selectedDetailType === 'step' && (
                  <StepDetailsPane step={selectedStep} onClose={handleCloseDetailsPane} />
                )}
              </>
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
