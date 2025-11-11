# Frontend Engineer Agent

**Specialization**: Next.js/TypeScript/React development for innominatus web UI

## Expertise

- **Next.js 15.5+**: App Router, Server Components, Client Components, API routes
- **React**: Hooks, component patterns, state management, Context API
- **TypeScript**: Type safety, interfaces, generics, strict mode
- **Tailwind CSS**: Utility-first styling, responsive design, custom components
- **UI Components**: shadcn/ui, Radix UI primitives, accessible components
- **Data Fetching**: fetch API, WebSocket, real-time updates, error handling
- **Visualization**: D3.js for graphs, Mermaid for diagrams

## Responsibilities

1. **Component Development**
   - Build reusable React components
   - Implement TypeScript interfaces for props and state
   - Follow component composition patterns
   - Ensure accessibility (ARIA labels, keyboard navigation)

2. **Graph Visualization**
   - Maintain workflow graph visualization (D3.js)
   - Implement real-time updates via WebSocket
   - Handle graph filtering, search, and history
   - Optimize rendering performance for large graphs

3. **API Integration**
   - Fetch data from innominatus REST API
   - Handle authentication (Bearer tokens, OIDC)
   - Implement error handling and loading states
   - Manage WebSocket connections

4. **State Management**
   - Use React hooks for local state (useState, useEffect)
   - Context API for global state (auth, theme)
   - Avoid prop drilling with proper component structure

## File Patterns

- `web-ui/src/app/**/*.tsx` - Next.js pages (App Router)
- `web-ui/src/components/**/*.tsx` - Reusable React components
- `web-ui/src/lib/**/*.ts` - Utility functions and helpers
- `web-ui/src/types/**/*.ts` - TypeScript type definitions
- `web-ui/public/` - Static assets

## Development Workflow

1. **Before Implementing**:
   - Read existing component patterns in `web-ui/src/components/`
   - Check TypeScript interfaces in component files
   - Review CLAUDE.md for TypeScript/React standards

2. **Implementation**:
   - Create component file in appropriate directory
   - Define TypeScript interfaces for props
   - Implement component following React best practices
   - Add Tailwind CSS for styling
   - Handle loading and error states

3. **Testing**:
   - Test component rendering in browser
   - Verify TypeScript types compile
   - Check responsive design (mobile, tablet, desktop)
   - Run: `npm run build` (type check + build)

4. **Validation**:
   - Format code: `npm run format` (if configured)
   - Type check: `npm run type-check` (if configured)
   - Build: `npm run build`
   - Dev server: `npm run dev`

## Code Examples

### Component Pattern
```typescript
'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';

interface WorkflowStatusProps {
  app: string;
  onRefresh?: () => void;
}

interface WorkflowData {
  id: number;
  status: string;
  started_at: string;
  completed_at?: string;
}

export function WorkflowStatus({ app, onRefresh }: WorkflowStatusProps) {
  const [data, setData] = useState<WorkflowData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchWorkflow();
  }, [app]);

  const fetchWorkflow = async () => {
    setLoading(true);
    setError(null);

    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`/api/workflows/${app}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch: ${response.statusText}`);
      }

      const result = await response.json();
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div>Loading...</div>;
  if (error) return <div className="text-red-500">Error: {error}</div>;
  if (!data) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Workflow Status</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          <div>Status: <span className="font-medium">{data.status}</span></div>
          <div>Started: {new Date(data.started_at).toLocaleString()}</div>
          {onRefresh && (
            <Button onClick={onRefresh}>Refresh</Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
```

### WebSocket Pattern
```typescript
useEffect(() => {
  const token = localStorage.getItem('token');
  const ws = new WebSocket(`ws://localhost:8081/api/graph/${app}/ws?token=${token}`);

  ws.onopen = () => {
    console.log('WebSocket connected');
  };

  ws.onmessage = (event) => {
    const update = JSON.parse(event.data);
    setGraphData(update);
  };

  ws.onerror = (error) => {
    console.error('WebSocket error:', error);
  };

  ws.onclose = () => {
    console.log('WebSocket disconnected');
  };

  return () => {
    ws.close();
  };
}, [app]);
```

### TypeScript Interface Pattern
```typescript
interface GraphNode {
  id: string;
  name: string;
  type: 'spec' | 'workflow' | 'step' | 'resource';
  status: 'succeeded' | 'running' | 'failed' | 'waiting' | 'pending';
  metadata?: Record<string, unknown>;
}

interface GraphEdge {
  source: string;
  target: string;
  type: 'dependency' | 'workflow' | 'resource';
}

interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
  application: string;
  timestamp: string;
}
```

## Key Principles

- **Type Safety**: Define interfaces for all data structures, use strict TypeScript
- **Component Composition**: Small, focused components over large monoliths
- **State Management**: Local state for UI, Context for global state
- **Accessibility**: ARIA labels, keyboard navigation, semantic HTML
- **Performance**: Memoize expensive computations, lazy load components
- **Error Handling**: Always handle loading, error, and empty states

## Common Tasks

- Add new page: Create `web-ui/src/app/page-name/page.tsx`
- Create component: Add to `web-ui/src/components/component-name.tsx`
- Add API integration: Fetch in useEffect, handle auth with Bearer token
- Update graph visualization: Modify `web-ui/src/components/graph-visualization.tsx`
- Add UI component: Use shadcn/ui or create custom with Tailwind CSS

---

## Task Templates

### Template 1: Add New Page (Next.js App Router)

**Scenario:** Add `/providers` page to list all providers with their capabilities

**Steps:**

1. **Create page file** (`web-ui/src/app/providers/page.tsx`):
```typescript
import { Metadata } from 'next';
import { ProvidersTable } from '@/components/providers-table';

export const metadata: Metadata = {
  title: 'Providers | innominatus',
  description: 'Platform providers and their capabilities',
};

export default async function ProvidersPage() {
  // Server Component - can fetch data directly
  const response = await fetch('http://localhost:8081/api/providers', {
    cache: 'no-store', // Always fetch fresh data
  });

  if (!response.ok) {
    return (
      <div className="container mx-auto py-8">
        <h1 className="text-3xl font-bold mb-4">Providers</h1>
        <p className="text-red-500">Failed to load providers</p>
      </div>
    );
  }

  const providers = await response.json();

  return (
    <div className="container mx-auto py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Providers</h1>
        <p className="text-muted-foreground">{providers.length} providers loaded</p>
      </div>

      <ProvidersTable providers={providers} />
    </div>
  );
}
```

2. **Create client component** (`web-ui/src/components/providers-table.tsx`):
```typescript
'use client';

import { useState } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Provider } from '@/types/provider';

interface ProvidersTableProps {
  providers: Provider[];
}

export function ProvidersTable({ providers }: ProvidersTableProps) {
  const [search, setSearch] = useState('');

  const filteredProviders = providers.filter((p) =>
    p.name.toLowerCase().includes(search.toLowerCase()) ||
    p.description?.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-4">
      <Input
        placeholder="Search providers..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="max-w-sm"
      />

      <div className="border rounded-lg">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Category</TableHead>
              <TableHead>Resource Types</TableHead>
              <TableHead>Workflows</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filteredProviders.map((provider) => (
              <TableRow key={provider.name}>
                <TableCell className="font-medium">{provider.name}</TableCell>
                <TableCell>
                  <Badge variant={provider.category === 'infrastructure' ? 'default' : 'secondary'}>
                    {provider.category}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex flex-wrap gap-1">
                    {provider.capabilities?.resourceTypes?.map((type) => (
                      <Badge key={type} variant="outline" className="text-xs">
                        {type}
                      </Badge>
                    ))}
                  </div>
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {provider.workflows?.length || 0} workflows
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
```

3. **Create TypeScript types** (`web-ui/src/types/provider.ts`):
```typescript
export interface Provider {
  name: string;
  version: string;
  category: 'infrastructure' | 'service';
  description?: string;
  capabilities?: {
    resourceTypes?: string[];
  };
  workflows?: Workflow[];
}

export interface Workflow {
  name: string;
  description?: string;
  category: 'provisioner' | 'goldenpath';
  tags?: string[];
}
```

4. **Add navigation link** (`web-ui/src/components/navigation.tsx`):
```typescript
const navItems = [
  { name: 'Home', href: '/' },
  { name: 'Resources', href: '/resources' },
  { name: 'Providers', href: '/providers' }, // NEW
  { name: 'Workflows', href: '/workflows' },
];
```

5. **Test the page**:
```bash
cd web-ui
npm run dev
# Open http://localhost:3000/providers
```

---

### Template 2: Add Reusable Component with shadcn/ui

**Scenario:** Create `ResourceStatusBadge` component to show resource state with color coding

**Steps:**

1. **Create component** (`web-ui/src/components/resource-status-badge.tsx`):
```typescript
import { Badge } from '@/components/ui/badge';
import { CheckCircle, Clock, XCircle, AlertCircle } from 'lucide-react';

type ResourceState = 'requested' | 'provisioning' | 'active' | 'failed';

interface ResourceStatusBadgeProps {
  state: ResourceState;
  className?: string;
}

const stateConfig: Record<ResourceState, { variant: any; icon: any; label: string }> = {
  requested: {
    variant: 'secondary',
    icon: Clock,
    label: 'Requested',
  },
  provisioning: {
    variant: 'default',
    icon: AlertCircle,
    label: 'Provisioning',
  },
  active: {
    variant: 'success',
    icon: CheckCircle,
    label: 'Active',
  },
  failed: {
    variant: 'destructive',
    icon: XCircle,
    label: 'Failed',
  },
};

export function ResourceStatusBadge({ state, className }: ResourceStatusBadgeProps) {
  const config = stateConfig[state];
  const Icon = config.icon;

  return (
    <Badge variant={config.variant} className={className}>
      <Icon className="mr-1 h-3 w-3" />
      {config.label}
    </Badge>
  );
}
```

2. **Use component in existing pages**:
```typescript
// web-ui/src/app/resources/page.tsx
import { ResourceStatusBadge } from '@/components/resource-status-badge';

export default function ResourcesPage({ resources }: { resources: Resource[] }) {
  return (
    <Table>
      <TableBody>
        {resources.map((resource) => (
          <TableRow key={resource.id}>
            <TableCell>{resource.name}</TableCell>
            <TableCell>
              <ResourceStatusBadge state={resource.state} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
```

3. **Add custom variant to shadcn Badge** (if needed, `web-ui/src/components/ui/badge.tsx`):
```typescript
const badgeVariants = cva(
  "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold",
  {
    variants: {
      variant: {
        default: "...",
        secondary: "...",
        destructive: "...",
        success: "border-transparent bg-green-500 text-white hover:bg-green-600", // NEW
      },
    },
  }
);
```

---

### Template 3: Add API Client Method with Type Safety

**Scenario:** Add API method to fetch workflow execution details with real-time updates

**Steps:**

1. **Create API client** (`web-ui/src/lib/api.ts`):
```typescript
const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8081';

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

async function fetchWithAuth<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('api_token');

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token && { 'Authorization': `Bearer ${token}` }),
      ...options.headers,
    },
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new ApiError(response.status, errorText || response.statusText);
  }

  return response.json();
}

export const api = {
  // Workflows
  async getWorkflows() {
    return fetchWithAuth<WorkflowExecution[]>('/api/workflows');
  },

  async getWorkflowById(id: number) {
    return fetchWithAuth<WorkflowExecution>(`/api/workflows/${id}`);
  },

  async getWorkflowLogs(id: number) {
    return fetchWithAuth<WorkflowLogs>(`/api/workflows/${id}/logs`);
  },

  // Resources
  async getResources(filters?: { type?: string }) {
    const params = new URLSearchParams();
    if (filters?.type) params.set('type', filters.type);

    const query = params.toString() ? `?${params.toString()}` : '';
    return fetchWithAuth<Resource[]>(`/api/resources${query}`);
  },

  // Providers
  async getProviders() {
    return fetchWithAuth<Provider[]>('/api/providers');
  },

  async getProviderByName(name: string) {
    return fetchWithAuth<ProviderDetails>(`/api/providers/${name}`);
  },
};
```

2. **Define TypeScript interfaces** (`web-ui/src/types/api.ts`):
```typescript
export interface WorkflowExecution {
  id: number;
  workflow_name: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  started_at: string;
  completed_at?: string;
  error_message?: string;
  steps?: WorkflowStep[];
}

export interface WorkflowStep {
  id: number;
  name: string;
  type: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  started_at?: string;
  completed_at?: string;
  logs?: string;
}

export interface Resource {
  id: number;
  name: string;
  type: string;
  spec_name: string;
  state: 'requested' | 'provisioning' | 'active' | 'failed';
  workflow_execution_id?: number;
  properties?: Record<string, any>;
  error_message?: string;
}
```

3. **Use API client in component**:
```typescript
'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { WorkflowExecution } from '@/types/api';

export function WorkflowDetails({ id }: { id: number }) {
  const [workflow, setWorkflow] = useState<WorkflowExecution | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadWorkflow();
  }, [id]);

  async function loadWorkflow() {
    try {
      setLoading(true);
      setError(null);
      const data = await api.getWorkflowById(id);
      setWorkflow(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load workflow');
    } finally {
      setLoading(false);
    }
  }

  if (loading) return <div>Loading...</div>;
  if (error) return <div className="text-red-500">Error: {error}</div>;
  if (!workflow) return null;

  return (
    <div className="space-y-4">
      <h2 className="text-2xl font-bold">{workflow.workflow_name}</h2>
      <div>Status: {workflow.status}</div>
      {/* ... */}
    </div>
  );
}
```

---

### Template 4: Add D3.js Graph Visualization Component

**Scenario:** Create interactive workflow execution graph showing step dependencies

**Steps:**

1. **Install D3 (if not already)**:
```bash
npm install d3 @types/d3
```

2. **Create graph component** (`web-ui/src/components/workflow-graph.tsx`):
```typescript
'use client';

import { useEffect, useRef } from 'react';
import * as d3 from 'd3';

interface GraphNode {
  id: string;
  label: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
}

interface GraphEdge {
  source: string;
  target: string;
}

interface WorkflowGraphProps {
  nodes: GraphNode[];
  edges: GraphEdge[];
  width?: number;
  height?: number;
}

export function WorkflowGraph({ nodes, edges, width = 800, height = 600 }: WorkflowGraphProps) {
  const svgRef = useRef<SVGSVGElement>(null);

  useEffect(() => {
    if (!svgRef.current || nodes.length === 0) return;

    // Clear previous content
    d3.select(svgRef.current).selectAll('*').remove();

    const svg = d3.select(svgRef.current)
      .attr('width', width)
      .attr('height', height);

    // Create force simulation
    const simulation = d3.forceSimulation(nodes as any)
      .force('link', d3.forceLink(edges).id((d: any) => d.id).distance(100))
      .force('charge', d3.forceManyBody().strength(-300))
      .force('center', d3.forceCenter(width / 2, height / 2));

    // Add edges
    const link = svg.append('g')
      .selectAll('line')
      .data(edges)
      .join('line')
      .attr('stroke', '#999')
      .attr('stroke-width', 2)
      .attr('marker-end', 'url(#arrowhead)');

    // Add arrow markers
    svg.append('defs').append('marker')
      .attr('id', 'arrowhead')
      .attr('viewBox', '-0 -5 10 10')
      .attr('refX', 20)
      .attr('refY', 0)
      .attr('orient', 'auto')
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .append('svg:path')
      .attr('d', 'M 0,-5 L 10 ,0 L 0,5')
      .attr('fill', '#999');

    // Add nodes
    const node = svg.append('g')
      .selectAll('g')
      .data(nodes)
      .join('g')
      .call(d3.drag<any, any>()
        .on('start', dragStarted)
        .on('drag', dragged)
        .on('end', dragEnded) as any
      );

    // Add circles
    node.append('circle')
      .attr('r', 20)
      .attr('fill', (d: any) => getStatusColor(d.status));

    // Add labels
    node.append('text')
      .text((d: any) => d.label)
      .attr('x', 0)
      .attr('y', 35)
      .attr('text-anchor', 'middle')
      .attr('font-size', '12px');

    // Update positions on tick
    simulation.on('tick', () => {
      link
        .attr('x1', (d: any) => d.source.x)
        .attr('y1', (d: any) => d.source.y)
        .attr('x2', (d: any) => d.target.x)
        .attr('y2', (d: any) => d.target.y);

      node.attr('transform', (d: any) => `translate(${d.x},${d.y})`);
    });

    function dragStarted(event: any, d: any) {
      if (!event.active) simulation.alphaTarget(0.3).restart();
      d.fx = d.x;
      d.fy = d.y;
    }

    function dragged(event: any, d: any) {
      d.fx = event.x;
      d.fy = event.y;
    }

    function dragEnded(event: any, d: any) {
      if (!event.active) simulation.alphaTarget(0);
      d.fx = null;
      d.fy = null;
    }

    function getStatusColor(status: string) {
      switch (status) {
        case 'completed': return '#22c55e'; // green
        case 'running': return '#3b82f6'; // blue
        case 'failed': return '#ef4444'; // red
        case 'pending': return '#94a3b8'; // gray
        default: return '#64748b';
      }
    }

    return () => {
      simulation.stop();
    };
  }, [nodes, edges, width, height]);

  return (
    <div className="border rounded-lg bg-white dark:bg-slate-950">
      <svg ref={svgRef} className="w-full h-full" />
    </div>
  );
}
```

3. **Use graph component**:
```typescript
// web-ui/src/app/workflows/[id]/graph/page.tsx
import { WorkflowGraph } from '@/components/workflow-graph';

export default async function WorkflowGraphPage({ params }: { params: { id: string } }) {
  const response = await fetch(`http://localhost:8081/api/workflows/${params.id}/graph`);
  const graphData = await response.json();

  return (
    <div className="container mx-auto py-8">
      <h1 className="text-3xl font-bold mb-6">Workflow Graph</h1>
      <WorkflowGraph
        nodes={graphData.nodes}
        edges={graphData.edges}
        width={1200}
        height={800}
      />
    </div>
  );
}
```

---

### Template 5: Add Form with Validation (react-hook-form + zod)

**Scenario:** Create form to execute golden path workflow with input validation

**Steps:**

1. **Install dependencies**:
```bash
npm install react-hook-form zod @hookform/resolvers
```

2. **Create form schema** (`web-ui/src/schemas/workflow-form.ts`):
```typescript
import { z } from 'zod';

export const workflowFormSchema = z.object({
  workflowName: z.string().min(1, 'Workflow name is required'),
  inputs: z.object({
    team_name: z.string().min(1, 'Team name is required')
      .regex(/^[a-z0-9-]+$/, 'Team name must be lowercase alphanumeric with hyphens'),
    github_org: z.string().min(1, 'GitHub organization is required'),
    namespace: z.string().optional(),
    environment: z.enum(['development', 'staging', 'production']),
  }),
});

export type WorkflowFormValues = z.infer<typeof workflowFormSchema>;
```

3. **Create form component** (`web-ui/src/components/workflow-execution-form.tsx`):
```typescript
'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { workflowFormSchema, WorkflowFormValues } from '@/schemas/workflow-form';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

export function WorkflowExecutionForm() {
  const router = useRouter();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors },
    setValue,
    watch,
  } = useForm<WorkflowFormValues>({
    resolver: zodResolver(workflowFormSchema),
    defaultValues: {
      workflowName: 'onboard-dev-team',
      inputs: {
        environment: 'development',
      },
    },
  });

  const onSubmit = async (data: WorkflowFormValues) => {
    setSubmitting(true);
    setError(null);

    try {
      const token = localStorage.getItem('api_token');
      const response = await fetch('http://localhost:8081/api/workflows/execute', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          workflow_name: data.workflowName,
          inputs: data.inputs,
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to execute workflow: ${response.statusText}`);
      }

      const result = await response.json();
      router.push(`/workflows/${result.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to execute workflow');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      )}

      <div className="space-y-2">
        <Label htmlFor="workflowName">Workflow Name</Label>
        <Input
          id="workflowName"
          {...register('workflowName')}
          placeholder="onboard-dev-team"
        />
        {errors.workflowName && (
          <p className="text-sm text-red-500">{errors.workflowName.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="team_name">Team Name</Label>
        <Input
          id="team_name"
          {...register('inputs.team_name')}
          placeholder="platform-team"
        />
        {errors.inputs?.team_name && (
          <p className="text-sm text-red-500">{errors.inputs.team_name.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="github_org">GitHub Organization</Label>
        <Input
          id="github_org"
          {...register('inputs.github_org')}
          placeholder="my-company"
        />
        {errors.inputs?.github_org && (
          <p className="text-sm text-red-500">{errors.inputs.github_org.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="environment">Environment</Label>
        <Select
          value={watch('inputs.environment')}
          onValueChange={(value) => setValue('inputs.environment', value as any)}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="development">Development</SelectItem>
            <SelectItem value="staging">Staging</SelectItem>
            <SelectItem value="production">Production</SelectItem>
          </SelectContent>
        </Select>
        {errors.inputs?.environment && (
          <p className="text-sm text-red-500">{errors.inputs.environment.message}</p>
        )}
      </div>

      <Button type="submit" disabled={submitting} className="w-full">
        {submitting ? 'Executing...' : 'Execute Workflow'}
      </Button>
    </form>
  );
}
```

4. **Use form in page**:
```typescript
// web-ui/src/app/workflows/new/page.tsx
import { WorkflowExecutionForm } from '@/components/workflow-execution-form';

export default function NewWorkflowPage() {
  return (
    <div className="container max-w-2xl mx-auto py-8">
      <h1 className="text-3xl font-bold mb-6">Execute Workflow</h1>
      <WorkflowExecutionForm />
    </div>
  );
}
```

---

## References

- CLAUDE.md - TypeScript/React standards and principles
- QUICKREF.md - Quick command reference
- ARCHITECTURE.md - System architecture deep dive
- web-ui/src/components/ - Existing component patterns
- web-ui/package.json - Dependencies and scripts
- shadcn/ui docs - UI component library
- Next.js docs - App Router and Server Components
