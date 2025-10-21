package graph

import (
	"strings"
	"testing"

	sdk "github.com/philipsahli/innominatus-graph/pkg/graph"
)

// Test helper to create a sample graph
func createTestGraph() *sdk.Graph {
	g := sdk.NewGraph("test-app")

	// Add nodes
	specNode := &sdk.Node{
		ID:          "spec-1",
		Type:        sdk.NodeTypeSpec,
		Name:        "Test Spec",
		Description: "Test spec node",
		State:       sdk.NodeStateSucceeded,
	}
	_ = g.AddNode(specNode)

	workflowNode := &sdk.Node{
		ID:          "workflow-1",
		Type:        sdk.NodeTypeWorkflow,
		Name:        "Deploy Workflow",
		Description: "Main deployment workflow",
		State:       sdk.NodeStateRunning,
	}
	_ = g.AddNode(workflowNode)

	stepNode := &sdk.Node{
		ID:          "step-1",
		Type:        sdk.NodeTypeStep,
		Name:        "Build Step",
		Description: "Build application",
		State:       sdk.NodeStatePending,
	}
	_ = g.AddNode(stepNode)

	resourceNode := &sdk.Node{
		ID:          "resource-1",
		Type:        sdk.NodeTypeResource,
		Name:        "Database",
		Description: "PostgreSQL database",
		State:       sdk.NodeStateWaiting,
	}
	_ = g.AddNode(resourceNode)

	// Add edges
	_ = g.AddEdge(&sdk.Edge{
		ID:          "edge-1",
		FromNodeID:  "workflow-1",
		ToNodeID:    "step-1",
		Type:        sdk.EdgeTypeContains,
		Description: "Workflow contains step",
	})

	_ = g.AddEdge(&sdk.Edge{
		ID:          "edge-2",
		FromNodeID:  "step-1",
		ToNodeID:    "resource-1",
		Type:        sdk.EdgeTypeConfigures,
		Description: "Step configures resource",
	})

	return g
}

func TestNewMermaidExporter(t *testing.T) {
	exporter := NewMermaidExporter()
	if exporter == nil {
		t.Fatal("NewMermaidExporter() returned nil")
	}
}

func TestExportGraph(t *testing.T) {
	tests := []struct {
		name      string
		graph     *sdk.Graph
		wantErr   bool
		wantParts []string // Strings that should be present in output
	}{
		{
			name:    "nil graph returns error",
			graph:   nil,
			wantErr: true,
		},
		{
			name:    "valid graph exports successfully",
			graph:   createTestGraph(),
			wantErr: false,
			wantParts: []string{
				"flowchart TD",
				"%% Workflow Execution Graph",
				"%% Nodes",
				"%% Edges",
				"%% Styling",
				"spec_1",
				"workflow_1",
				"step_1",
				"resource_1",
			},
		},
		{
			name: "empty graph exports successfully",
			graph: func() *sdk.Graph {
				return sdk.NewGraph("empty-app")
			}(),
			wantErr: false,
			wantParts: []string{
				"flowchart TD",
				"%% Nodes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMermaidExporter()
			got, err := m.ExportGraph(tt.graph)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExportGraph() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Check for expected parts in output
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("ExportGraph() output missing expected part: %q", part)
				}
			}
		})
	}
}

func TestExportGraphSimple(t *testing.T) {
	tests := []struct {
		name      string
		graph     *sdk.Graph
		wantErr   bool
		wantParts []string
	}{
		{
			name:    "nil graph returns error",
			graph:   nil,
			wantErr: true,
		},
		{
			name:    "valid graph exports successfully",
			graph:   createTestGraph(),
			wantErr: false,
			wantParts: []string{
				"flowchart LR",
				"%% Simplified Workflow Graph",
				"spec_1[\"Test Spec\"]",
				"workflow_1[\"Deploy Workflow\"]",
				"step_1[\"Build Step\"]",
				"resource_1[\"Database\"]",
				"-->",
			},
		},
		{
			name: "empty graph exports successfully",
			graph: func() *sdk.Graph {
				return sdk.NewGraph("empty-app")
			}(),
			wantErr: false,
			wantParts: []string{
				"flowchart LR",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMermaidExporter()
			got, err := m.ExportGraphSimple(tt.graph)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExportGraphSimple() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("ExportGraphSimple() output missing expected part: %q", part)
				}
			}
		})
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "replace colon", input: "node:1", want: "node_1"},
		{name: "replace hyphen", input: "node-1", want: "node_1"},
		{name: "replace dot", input: "node.1", want: "node_1"},
		{name: "replace space", input: "node 1", want: "node_1"},
		{name: "multiple replacements", input: "spec:app-1.test", want: "spec_app_1_test"},
		{name: "no replacements needed", input: "node_1", want: "node_1"},
		{name: "empty string", input: "", want: ""},
	}

	m := NewMermaidExporter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.sanitizeID(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatNodeLabel(t *testing.T) {
	tests := []struct {
		name      string
		nodeName  string
		nodeType  sdk.NodeType
		state     sdk.NodeState
		wantParts []string // Parts that should be present in the label
	}{
		{
			name:      "spec node succeeded",
			nodeName:  "My Spec",
			nodeType:  sdk.NodeTypeSpec,
			state:     sdk.NodeStateSucceeded,
			wantParts: []string{"✓", "My Spec", "spec"},
		},
		{
			name:      "workflow failed",
			nodeName:  "Deploy",
			nodeType:  sdk.NodeTypeWorkflow,
			state:     sdk.NodeStateFailed,
			wantParts: []string{"✗", "Deploy", "workflow"},
		},
		{
			name:      "step running",
			nodeName:  "Build",
			nodeType:  sdk.NodeTypeStep,
			state:     sdk.NodeStateRunning,
			wantParts: []string{"▶", "Build", "step"},
		},
		{
			name:      "resource waiting",
			nodeName:  "Database",
			nodeType:  sdk.NodeTypeResource,
			state:     sdk.NodeStateWaiting,
			wantParts: []string{"⏸", "Database", "resource"},
		},
		{
			name:      "quotes in name are escaped",
			nodeName:  "Node \"quoted\"",
			nodeType:  sdk.NodeTypeSpec,
			state:     sdk.NodeStatePending,
			wantParts: []string{"Node 'quoted'"},
		},
	}

	m := NewMermaidExporter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.formatNodeLabel(tt.nodeName, tt.nodeType, tt.state)
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("formatNodeLabel() = %q, missing expected part: %q", got, part)
				}
			}
		})
	}
}

func TestGetNodeStyle(t *testing.T) {
	tests := []struct {
		name     string
		nodeType sdk.NodeType
		state    sdk.NodeState
		want     string
	}{
		{name: "spec node", nodeType: sdk.NodeTypeSpec, state: sdk.NodeStateSucceeded, want: "["},
		{name: "workflow node", nodeType: sdk.NodeTypeWorkflow, state: sdk.NodeStateRunning, want: "{"},
		{name: "step node", nodeType: sdk.NodeTypeStep, state: sdk.NodeStatePending, want: "["},
		{name: "resource node", nodeType: sdk.NodeTypeResource, state: sdk.NodeStateWaiting, want: "("},
		{name: "unknown type defaults to rectangle", nodeType: sdk.NodeType("unknown"), state: sdk.NodeStateSucceeded, want: "["},
	}

	m := NewMermaidExporter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.getNodeStyle(tt.nodeType, tt.state)
			if got != tt.want {
				t.Errorf("getNodeStyle(%q, %q) = %q, want %q", tt.nodeType, tt.state, got, tt.want)
			}
		})
	}
}

func TestFormatEdgeLabel(t *testing.T) {
	tests := []struct {
		name        string
		edgeType    sdk.EdgeType
		description string
		want        string
	}{
		{name: "contains edge", edgeType: sdk.EdgeTypeContains, description: "", want: "contains"},
		{name: "depends on edge", edgeType: sdk.EdgeTypeDependsOn, description: "", want: "depends on"},
		{name: "provisions edge", edgeType: sdk.EdgeTypeProvisions, description: "", want: "provisions"},
		{name: "creates edge", edgeType: sdk.EdgeTypeCreates, description: "", want: "creates"},
		{name: "binds to edge", edgeType: sdk.EdgeTypeBindsTo, description: "", want: "binds to"},
		{name: "configures edge", edgeType: sdk.EdgeTypeConfigures, description: "", want: "configures"},
		{name: "unknown edge type", edgeType: sdk.EdgeType("unknown"), description: "", want: "unknown"},
	}

	m := NewMermaidExporter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.formatEdgeLabel(tt.edgeType, tt.description)
			if got != tt.want {
				t.Errorf("formatEdgeLabel(%q, %q) = %q, want %q", tt.edgeType, tt.description, got, tt.want)
			}
		})
	}
}

func TestGetStateIcon(t *testing.T) {
	tests := []struct {
		state sdk.NodeState
		want  string
	}{
		{state: sdk.NodeStateSucceeded, want: "✓"},
		{state: sdk.NodeStateFailed, want: "✗"},
		{state: sdk.NodeStateRunning, want: "▶"},
		{state: sdk.NodeStateWaiting, want: "⏸"},
		{state: sdk.NodeStatePending, want: "○"},
		{state: sdk.NodeState("unknown"), want: "•"},
	}

	m := NewMermaidExporter()
	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := m.getStateIcon(tt.state)
			if got != tt.want {
				t.Errorf("getStateIcon(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestGetNodeClass(t *testing.T) {
	tests := []struct {
		name     string
		nodeType sdk.NodeType
		state    sdk.NodeState
		want     string
	}{
		// State takes precedence
		{name: "failed state takes precedence", nodeType: sdk.NodeTypeSpec, state: sdk.NodeStateFailed, want: "failed"},
		{name: "running state takes precedence", nodeType: sdk.NodeTypeWorkflow, state: sdk.NodeStateRunning, want: "running"},
		{name: "waiting state takes precedence", nodeType: sdk.NodeTypeStep, state: sdk.NodeStateWaiting, want: "waiting"},

		// Type-based classes for non-failed/running/waiting states
		{name: "spec type class", nodeType: sdk.NodeTypeSpec, state: sdk.NodeStateSucceeded, want: "spec"},
		{name: "workflow type class", nodeType: sdk.NodeTypeWorkflow, state: sdk.NodeStatePending, want: "workflow"},
		{name: "step type class", nodeType: sdk.NodeTypeStep, state: sdk.NodeStateSucceeded, want: "step"},
		{name: "resource type class", nodeType: sdk.NodeTypeResource, state: sdk.NodeStatePending, want: "resource"},
		{name: "unknown type defaults to waiting", nodeType: sdk.NodeType("unknown"), state: sdk.NodeStateSucceeded, want: "waiting"},
	}

	m := NewMermaidExporter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.getNodeClass(tt.nodeType, tt.state)
			if got != tt.want {
				t.Errorf("getNodeClass(%q, %q) = %q, want %q", tt.nodeType, tt.state, got, tt.want)
			}
		})
	}
}

func TestExportGraphWithAllStates(t *testing.T) {
	// Create a graph with all possible states
	g := sdk.NewGraph("all-states-app")

	states := []sdk.NodeState{
		sdk.NodeStateSucceeded,
		sdk.NodeStateFailed,
		sdk.NodeStateRunning,
		sdk.NodeStateWaiting,
		sdk.NodeStatePending,
	}

	for _, state := range states {
		node := &sdk.Node{
			ID:    string(state) + "-node",
			Type:  sdk.NodeTypeStep,
			Name:  string(state),
			State: state,
		}
		_ = g.AddNode(node)
	}

	m := NewMermaidExporter()
	output, err := m.ExportGraph(g)
	if err != nil {
		t.Fatalf("ExportGraph() error = %v", err)
	}

	// Verify all state icons are present
	expectedIcons := []string{"✓", "✗", "▶", "⏸", "○"}
	for _, icon := range expectedIcons {
		if !strings.Contains(output, icon) {
			t.Errorf("ExportGraph() missing state icon: %q", icon)
		}
	}

	// Verify all CSS classes are present
	expectedClasses := []string{"failed", "running", "waiting", "step"}
	for _, class := range expectedClasses {
		if !strings.Contains(output, class) {
			t.Errorf("ExportGraph() missing CSS class: %q", class)
		}
	}
}

func TestExportGraphWithAllNodeTypes(t *testing.T) {
	// Create a graph with all node types
	g := sdk.NewGraph("all-types-app")

	nodeTypes := []sdk.NodeType{
		sdk.NodeTypeSpec,
		sdk.NodeTypeWorkflow,
		sdk.NodeTypeStep,
		sdk.NodeTypeResource,
	}

	for _, nodeType := range nodeTypes {
		node := &sdk.Node{
			ID:    string(nodeType) + "-node",
			Type:  nodeType,
			Name:  string(nodeType),
			State: sdk.NodeStateSucceeded,
		}
		_ = g.AddNode(node)
	}

	m := NewMermaidExporter()
	output, err := m.ExportGraph(g)
	if err != nil {
		t.Fatalf("ExportGraph() error = %v", err)
	}

	// Verify all node types are present
	for _, nodeType := range nodeTypes {
		if !strings.Contains(output, string(nodeType)) {
			t.Errorf("ExportGraph() missing node type: %q", nodeType)
		}
	}

	// Verify all CSS classes are present
	expectedClasses := []string{"spec", "workflow", "step", "resource"}
	for _, class := range expectedClasses {
		if !strings.Contains(output, class) {
			t.Errorf("ExportGraph() missing CSS class for node type: %q", class)
		}
	}
}

func TestExportGraphWithAllEdgeTypes(t *testing.T) {
	// Create a graph with all edge types
	g := sdk.NewGraph("all-edges-app")

	// Create nodes
	workflow := &sdk.Node{ID: "workflow-1", Type: sdk.NodeTypeWorkflow, Name: "Workflow", State: sdk.NodeStateRunning}
	step := &sdk.Node{ID: "step-1", Type: sdk.NodeTypeStep, Name: "Step", State: sdk.NodeStatePending}
	resource := &sdk.Node{ID: "resource-1", Type: sdk.NodeTypeResource, Name: "Resource", State: sdk.NodeStateWaiting}
	spec := &sdk.Node{ID: "spec-1", Type: sdk.NodeTypeSpec, Name: "Spec", State: sdk.NodeStateSucceeded}

	_ = g.AddNode(workflow)
	_ = g.AddNode(step)
	_ = g.AddNode(resource)
	_ = g.AddNode(spec)

	// Add edges with all types
	edges := []struct {
		id       string
		from     string
		to       string
		edgeType sdk.EdgeType
	}{
		{"edge-1", "workflow-1", "step-1", sdk.EdgeTypeContains},
		{"edge-2", "step-1", "resource-1", sdk.EdgeTypeConfigures},
		{"edge-3", "workflow-1", "resource-1", sdk.EdgeTypeProvisions},
	}

	for _, e := range edges {
		_ = g.AddEdge(&sdk.Edge{
			ID:         e.id,
			FromNodeID: e.from,
			ToNodeID:   e.to,
			Type:       e.edgeType,
		})
	}

	m := NewMermaidExporter()
	output, err := m.ExportGraph(g)
	if err != nil {
		t.Fatalf("ExportGraph() error = %v", err)
	}

	// Verify edge labels are present
	expectedLabels := []string{"contains", "configures", "provisions"}
	for _, label := range expectedLabels {
		if !strings.Contains(output, label) {
			t.Errorf("ExportGraph() missing edge label: %q", label)
		}
	}
}
