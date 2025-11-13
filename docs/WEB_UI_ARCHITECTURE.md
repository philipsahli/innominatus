# Web UI Architecture

## Overview

The Innominatus Web UI is a modern Next.js application providing a rich interface for platform orchestration and monitoring.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      Browser (Client)                        │
├─────────────────────────────────────────────────────────────┤
│  Next.js App (SSR + CSR)                                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Pages/Routes (App Router)                          │   │
│  │  - /dev/graph    (Graph Explorer)                   │   │
│  │  - /workflows    (Workflow Monitor)                 │   │
│  │  - /resources    (Resource Browser)                 │   │
│  │  - /goldenpaths  (Golden Path Templates)            │   │
│  │  - /ai-assistant (AI Interface)                     │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Components (React)                                  │   │
│  │  - Detail Panes (Workflow, Resource, Spec, Step)   │   │
│  │  - Graph Visualizations (SVG, Cytoscape, D3, etc.) │   │
│  │  - UI Components (shadcn/ui)                        │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  API Client (src/lib/api.ts)                        │   │
│  │  - REST calls                                        │   │
│  │  - WebSocket connections                             │   │
│  │  - Type-safe responses                               │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP/WebSocket
                              ▼
┌─────────────────────────────────────────────────────────────┐
│            Innominatus Backend (Go Server)                  │
│            - REST API (port 8081)                           │
│            - WebSocket streaming                            │
│            - Graph database                                 │
└─────────────────────────────────────────────────────────────┘
```

## Technology Stack

### Core Framework
- **Next.js 15.5.6**: React framework with App Router
- **React 18**: UI library with concurrent features
- **TypeScript 5.x**: Type safety and developer experience

### Styling & UI
- **Tailwind CSS**: Utility-first CSS framework
- **shadcn/ui**: High-quality React components
- **Lucide React**: Icon library

### Data Visualization
- **Cytoscape.js**: Graph visualization with physics
- **ReactFlow**: Interactive node-based graphs
- **D3.js**: Force-directed layouts

### Testing
- **Playwright**: E2E testing framework

## Component Architecture

### Detail Panes Pattern

All detail pane components follow a consistent three-section architecture:

```typescript
// ============================================================================
// Types & Utilities
// ============================================================================
interface ComponentProps { /* ... */ }
type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

const STATUS_CONFIG = {
  active: { badgeVariant: 'default', icon: <Icon />, color: '...' },
  // ... other states
} as const;

const getStatusConfig = (status: string) => { /* ... */ };
const formatDuration = (ms: number) => { /* ... */ };

// ============================================================================
// Sub-Components
// ============================================================================
function HeaderComponent(props) { /* ... */ }
function CardComponent(props) { /* ... */ }

// ============================================================================
// Main Component
// ============================================================================
export function MainComponent(props) {
  // State management
  // Event handlers
  // Render sub-components
}
```

**Benefits:**
- Consistent structure across all components
- Easy to locate types, utilities, and sub-components
- Better code organization and maintainability
- Encourages component composition over monolithic components

### Detail Panes Inventory

| Component | Sub-Components | Purpose |
|-----------|----------------|---------|
| **WorkflowDetailsPane** | WorkflowHeader, StepCard, TimelineBar, RetryDialog | Workflow execution details with retry functionality |
| **ResourceDetailsPane** | ResourceHeader, QuickAccessCard, ResourceInfoCard, ConfigurationCard, TimelineCard | Resource instance details with health monitoring |
| **SpecDetailsPane** | SpecHeader, SpecOverviewCard, WorkflowsCard, ResourceCard, SpecContentCard | Score specification viewer with related resources |
| **StepDetailsPane** | (Pending refactoring) | Individual workflow step details |

### Graph Visualization Modes

#### 1. SVG Mode (Default)
- **Use case**: Simple graphs, quick rendering
- **Pros**: Lightweight, fast, no dependencies
- **Cons**: Limited interactivity, manual layout

#### 2. Text Mode
- **Use case**: Terminal-friendly, accessibility
- **Pros**: Works without graphics, copy-paste friendly
- **Cons**: Limited visual appeal

#### 3. Cytoscape Mode
- **Use case**: Complex graphs, automatic layouts
- **Pros**: Physics simulation, many layout algorithms
- **Cons**: Heavier bundle size

#### 4. ReactFlow Mode
- **Use case**: Interactive editing, drag-and-drop
- **Pros**: Built for React, excellent UX
- **Cons**: Larger bundle, learning curve

#### 5. D3-Force Mode
- **Use case**: Custom styling, force-directed layouts
- **Pros**: Highly customizable, beautiful animations
- **Cons**: Complex to maintain, performance considerations

## State Management

### Component-Level State
```typescript
// Local UI state
const [expanded, setExpanded] = useState(false);
const [copied, setCopied] = useState(false);
```

### Server State
```typescript
// Fetch on mount, cache results
useEffect(() => {
  async function loadData() {
    const response = await api.getWorkflows();
    if (response.success) {
      setWorkflows(response.data);
    }
  }
  loadData();
}, []);
```

### Real-time State (WebSocket)
```typescript
useEffect(() => {
  const ws = new WebSocket(wsUrl);
  ws.onmessage = (event) => {
    const update = JSON.parse(event.data);
    setNodes(update.nodes);
    setEdges(update.edges);
  };
  return () => ws.close();
}, [wsUrl]);
```

## API Integration

### Type-Safe Client

```typescript
// src/lib/api.ts
export const api = {
  async getGraphForApp(appName: string): Promise<ApiResponse<GraphData>> {
    // Type-safe REST call
  },

  async executeWorkflow(id: number, inputs: Record<string, any>): Promise<ApiResponse<WorkflowExecution>> {
    // Type-safe POST request
  },

  // ... 50+ endpoints
};
```

### Error Handling

```typescript
const response = await api.getResource(id);
if (!response.success) {
  // Handle error
  console.error(response.error);
  return;
}

// TypeScript knows response.data is ResourceInstance
const resource = response.data;
```

## Performance Optimization

### Code Splitting
- Dynamic imports for large components
- Route-based code splitting (automatic with Next.js)
- Lazy loading for visualization libraries

```typescript
const CytoscapeGraph = dynamic(() => import('./graph-cytoscape'), {
  loading: () => <div>Loading...</div>,
  ssr: false
});
```

### Memoization
```typescript
const StatusBadge = React.memo(({ status }: { status: string }) => {
  const config = getStatusConfig(status);
  return <Badge variant={config.badgeVariant}>{status}</Badge>;
});
```

### WebSocket Optimization
- Debounced updates (max 100ms)
- Automatic reconnection with exponential backoff
- Connection state management

## Accessibility

### Keyboard Navigation
- Tab/Shift+Tab: Navigate between interactive elements
- Enter/Space: Activate buttons and links
- Arrow keys: Navigate in autocomplete, lists
- Escape: Close modals, cancel actions

### Screen Reader Support
- Semantic HTML (`<main>`, `<nav>`, `<article>`)
- ARIA labels on icon-only buttons
- Focus management in modals
- Status announcements for async operations

### Color Contrast
- All text meets WCAG AA standards (4.5:1 contrast)
- Interactive elements have clear hover/focus states
- Status colors don't rely solely on color (icons too)

## Testing Strategy

### E2E Tests (Playwright)
```typescript
test('should display workflow details when clicking node', async ({ page }) => {
  await page.goto('/dev/graph?app=test-app');
  await page.click('[data-node-id="workflow-123"]');
  await expect(page.locator('.workflow-details-pane')).toBeVisible();
});
```

### Component Tests (Future)
- Unit tests for utility functions
- Component tests for detail panes
- Integration tests for API client

## Build & Deploy

### Development
```bash
npm run dev          # Local dev server (http://localhost:3000)
```

### Production
```bash
npm run build        # Create optimized build
npm run start        # Start production server
```

### Docker
```bash
docker build -t innominatus-ui .
docker run -p 3000:3000 innominatus-ui
```

## Security

### Authentication
- Session-based auth with httpOnly cookies
- OIDC integration for SSO
- User impersonation for admin debugging

### API Security
- All API calls include authentication tokens
- CORS configured for allowed origins
- No sensitive data in client-side code

### Content Security Policy
```typescript
// next.config.ts
const securityHeaders = [
  {
    key: 'X-Content-Type-Options',
    value: 'nosniff'
  },
  {
    key: 'X-Frame-Options',
    value: 'DENY'
  },
  // ... more headers
];
```

## Monitoring & Debugging

### Development Tools
- React DevTools for component inspection
- Next.js DevTools for route debugging
- Network tab for API calls
- WebSocket frames in browser inspector

### Error Tracking
```typescript
try {
  await riskyOperation();
} catch (err) {
  console.error('Operation failed:', err);
  // Could integrate with Sentry, Datadog, etc.
}
```

## Future Enhancements

### Planned Features
- [ ] Real-time collaboration (multi-user graph viewing)
- [ ] Graph diff viewer (compare versions)
- [ ] Workflow designer (visual workflow builder)
- [ ] Advanced analytics dashboard
- [ ] Mobile-responsive layouts
- [ ] Offline mode with service workers

### Technical Debt
- [ ] Migrate remaining class components to hooks
- [ ] Add unit tests for utility functions
- [ ] Implement React Query for server state
- [ ] Add Storybook for component documentation
- [ ] Improve bundle size with code splitting

## Contributing

### Adding a New Page
1. Create page in `src/app/[route]/page.tsx`
2. Add navigation link in `src/components/navigation.tsx`
3. Update documentation
4. Add E2E test in `tests/e2e/`

### Adding a New Component
1. Create component in `src/components/[name].tsx`
2. Follow the three-section pattern (Types, Sub-components, Main)
3. Add TypeScript interfaces for all props
4. Use existing UI components from `src/components/ui/`
5. Add to exports if reusable

### Coding Standards
- Use functional components with hooks
- TypeScript strict mode (no `any`)
- Prettier formatting (2-space indent)
- ESLint for code quality
- Meaningful variable names
- Comments for complex logic only

## Resources

- [Next.js Documentation](https://nextjs.org/docs)
- [React Documentation](https://react.dev)
- [shadcn/ui Components](https://ui.shadcn.com)
- [Tailwind CSS](https://tailwindcss.com)
- [TypeScript Handbook](https://www.typescriptlang.org/docs)

---

**Last Updated**: 2025-01-13
**Maintained By**: Platform Engineering Team
