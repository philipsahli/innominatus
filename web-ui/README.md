# Innominatus Web UI

Modern Next.js-based web interface for the Innominatus platform orchestration system.

## Tech Stack

- **Framework**: Next.js 15.5.6 with App Router
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **UI Components**: shadcn/ui
- **Icons**: Lucide React
- **Testing**: Playwright (E2E)

## Getting Started

### Development

```bash
npm install          # Install dependencies
npm run dev          # Start dev server (http://localhost:3000)
npm run build        # Build for production
npm run start        # Start production server
npm run lint         # Run ESLint
```

### Testing

```bash
npx playwright test              # Run all E2E tests
npx playwright test --ui         # Run tests in UI mode
npx playwright test --debug      # Run tests in debug mode
npx playwright show-report       # Show test report
```

## Key Features

### 🔍 Advanced Graph Explorer (`/dev/graph`)

Interactive visualization and exploration of application dependency graphs with real-time updates.

**Features:**

- Multiple visualization modes (SVG, Text, Cytoscape, ReactFlow, D3-Force)
- Real-time WebSocket updates for live workflow tracking
- Advanced filtering by type, status, resource type, provider, and health
- Search autocomplete with keyboard navigation
- Click-to-navigate between related nodes
- Detailed side panels for all node types

**Detail Panes:**

- **Workflow Details**: Progress tracking, step-by-step execution, timeline view, retry functionality
- **Resource Details**: Quick access hints, health status, configuration, provider info
- **Spec Details**: Score specification viewer, associated resources and workflows
- **Step Details**: Execution logs, configuration, error messages

### 📊 AI Assistant (`/ai-assistant`)

Conversational AI interface for platform operations powered by Claude.

**Capabilities:**

- Natural language queries about platform state
- Automatic Score specification generation
- Resource provisioning recommendations
- Workflow execution guidance

### 🎯 Golden Paths (`/goldenpaths`)

Pre-built workflow templates for common platform tasks:

- Application deployment
- Team onboarding
- Environment provisioning
- Database lifecycle management

### 🔐 Admin Interface (`/admin`)

Platform administration features:

- User management
- OIDC integration configuration
- User impersonation for debugging
- System settings

### 📦 Resource Management (`/resources`, `/dev/resources`)

Browse and manage provisioned platform resources:

- Real-time status monitoring
- Health check information
- Configuration viewing
- Provider metadata

### 🔄 Workflow Tracking (`/workflows`, `/dev/workflows`)

Monitor and control workflow executions:

- Execution history
- Step-by-step progress
- Log streaming
- Retry failed workflows
- Workflow analytics

## Project Structure

```
web-ui/
├── src/
│   ├── app/                    # Next.js App Router pages
│   │   ├── dev/               # Developer tools (/dev/*)
│   │   │   ├── graph/         # Advanced graph explorer
│   │   │   ├── applications/  # Application management
│   │   │   ├── resources/     # Resource browser
│   │   │   ├── workflows/     # Workflow monitor
│   │   │   └── assistant/     # AI assistant
│   │   ├── admin/             # Admin pages
│   │   ├── ai-assistant/      # Public AI assistant
│   │   ├── goldenpaths/       # Golden path templates
│   │   ├── providers/         # Provider registry
│   │   ├── resources/         # Resource listing
│   │   └── workflows/         # Workflow listing
│   ├── components/            # React components
│   │   ├── ui/               # shadcn/ui components
│   │   ├── dev/              # Dev tools components
│   │   ├── deploy-wizard/    # Deployment wizard
│   │   ├── *-details-pane.tsx # Detail panel components
│   │   ├── graph-*.tsx       # Graph visualization components
│   │   └── workflow-*.tsx    # Workflow components
│   ├── lib/                   # Utilities and helpers
│   │   ├── api.ts            # API client
│   │   └── utils.ts          # Utility functions
│   └── styles/               # Global styles
├── public/                    # Static assets
├── tests/                     # E2E tests
│   └── e2e/                  # Playwright tests
├── playwright.config.ts       # Playwright configuration
├── next.config.ts            # Next.js configuration
└── tailwind.config.ts        # Tailwind configuration
```

## Component Architecture

### Detail Panes (Refactored Architecture)

All detail pane components follow a consistent pattern:

```typescript
// ============================================================================
// Types & Utilities
// ============================================================================
// - TypeScript interfaces
// - Configuration objects (STATUS_CONFIG, STATE_CONFIG)
// - Utility functions (getStatusConfig, formatDuration)

// ============================================================================
// Sub-Components
// ============================================================================
// - Header component
// - Card components (info, timeline, etc.)
// - Specialized UI elements

// ============================================================================
// Main Component
// ============================================================================
// - Main pane component orchestrating sub-components
```

**Benefits:**

- **Consistent**: All panes follow same structure
- **Maintainable**: Easy to understand and modify
- **Reusable**: Sub-components can be extracted
- **Type-safe**: Comprehensive TypeScript coverage
- **Testable**: Components isolated for unit testing

### Graph Visualization System

Multiple rendering engines for different use cases:

1. **SVG** - Simple, lightweight, good for small graphs
2. **Text View** - Tree-based ASCII art representation
3. **Cytoscape** - Advanced graph layouts, physics simulation
4. **ReactFlow** - Interactive node graph with drag-and-drop
5. **D3-Force** - Force-directed graph with custom styling

## API Integration

### REST API Client (`src/lib/api.ts`)

Type-safe API client with comprehensive error handling:

```typescript
import { api } from '@/lib/api';

// Fetch graph data
const response = await api.getGraphForApp(appName);

// Execute workflow
await api.executeWorkflow(workflowId, inputs);

// Get resource details
const resource = await api.getResource(resourceId);
```

### WebSocket Integration

Real-time updates for live workflow tracking:

```typescript
const ws = new WebSocket(`ws://localhost:8081/api/graph/${app}/ws`);
ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  // Handle graph updates
};
```

## Environment Variables

```bash
# API Configuration
NEXT_PUBLIC_API_URL=http://localhost:8081

# Authentication
NEXT_PUBLIC_OIDC_ENABLED=false
NEXT_PUBLIC_OIDC_ISSUER=
NEXT_PUBLIC_OIDC_CLIENT_ID=

# Feature Flags
NEXT_PUBLIC_ENABLE_DEV_TOOLS=true
```

## Development Guidelines

### Code Style

- **TypeScript**: Strict mode enabled, no `any` types
- **Formatting**: Prettier with 2-space indentation
- **Naming**: PascalCase (components), camelCase (functions), SCREAMING_SNAKE_CASE (constants)
- **Imports**: Organized (React → External → Internal → Styles)

### Component Patterns

```typescript
// Prefer functional components with TypeScript
interface MyComponentProps {
  title: string;
  onAction: () => void;
}

export function MyComponent({ title, onAction }: MyComponentProps) {
  const [state, setState] = useState<string>('');

  // Component logic

  return (
    <div>
      {/* JSX */}
    </div>
  );
}
```

### State Management

- **Local state**: `useState` for component-level state
- **Server state**: React Query patterns (fetch on mount)
- **Global state**: Context API sparingly
- **Form state**: Controlled components with validation

## Performance Optimizations

- **Code splitting**: Dynamic imports for large components
- **Lazy loading**: React.lazy() for non-critical routes
- **Memoization**: React.memo() for expensive re-renders
- **Virtual scrolling**: For large lists (workflows, resources)
- **WebSocket optimization**: Debounced updates, reconnection logic

## Accessibility

- Semantic HTML elements
- ARIA labels where needed
- Keyboard navigation support
- Focus management in modals/dialogs
- Color contrast compliance

## Browser Support

- Chrome/Edge: Latest 2 versions
- Firefox: Latest 2 versions
- Safari: Latest 2 versions

## Contributing

When adding new features:

1. Follow the existing component architecture
2. Add TypeScript types for all props and state
3. Include E2E tests for critical paths
4. Update documentation for user-facing features
5. Ensure accessibility standards are met

## Recent Improvements (2025-01-13)

### Graph Explorer Enhancement

**Detail Pane Refactoring:**

- Extracted 17 reusable sub-components across 3 detail panes
- Centralized status configuration (STATUS_CONFIG, STATE_CONFIG, HEALTH_CONFIG)
- Improved type safety with explicit BadgeVariant types
- Better code organization with clear section separators

**WorkflowDetailsPane:**

- ✅ Retry confirmation dialog with detailed warnings
- ✅ Visual progress bar showing workflow completion
- ✅ Enhanced step status icons (animated spinner for running steps)
- ✅ Timeline view with horizontal execution bars
- ✅ Per-step retry functionality
- ✅ Copy configuration buttons

**ResourceDetailsPane:**

- ✅ Enhanced state indicators with icons
- ✅ Health status dots with color coding
- ✅ Copy configuration button
- ✅ Improved timeline with health coloring

**SpecDetailsPane:**

- ✅ Enhanced status icons throughout
- ✅ Collapsible workflows section with counts
- ✅ Better resource cards with status indicators
- ✅ Visual transition effects

**Code Quality:**

- 30% increase in lines (better spacing, features, types)
- Consistent patterns across all components
- Improved maintainability and reusability
- Zero TypeScript errors, clean build

## Learn More

- [Next.js Documentation](https://nextjs.org/docs)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
- [shadcn/ui](https://ui.shadcn.com/)
- [Playwright Testing](https://playwright.dev/)

## License

See parent project LICENSE file.
