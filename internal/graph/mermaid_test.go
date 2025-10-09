package graph

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/philipsahli/innominatus-graph/pkg/graph"
)

func TestMermaidExporter_ExportGraph(t *testing.T) {
	// Create a sample graph
	graph := sdk.NewGraph("test-app")

	// Add nodes
	specNode := &sdk.Node{
		ID:    "spec-1",
		Type:  sdk.NodeTypeSpec,
		Name:  "my-app",
		State: sdk.NodeStateSucceeded,
		Properties: map[string]interface{}{
			"version": "1.0",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := graph.AddNode(specNode); err != nil {
		t.Fatalf("Failed to add spec node: %v", err)
	}

	workflowNode := &sdk.Node{
		ID:    "workflow-1",
		Type:  sdk.NodeTypeWorkflow,
		Name:  "deploy-app",
		State: sdk.NodeStateRunning,
		Properties: map[string]interface{}{
			"execution_id": 123,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := graph.AddNode(workflowNode); err != nil {
		t.Fatalf("Failed to add workflow node: %v", err)
	}

	stepNode := &sdk.Node{
		ID:    "step-1",
		Type:  sdk.NodeTypeStep,
		Name:  "provision-database",
		State: sdk.NodeStateSucceeded,
		Properties: map[string]interface{}{
			"step_number": 1,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := graph.AddNode(stepNode); err != nil {
		t.Fatalf("Failed to add step node: %v", err)
	}

	resourceNode := &sdk.Node{
		ID:    "resource-1",
		Type:  sdk.NodeTypeResource,
		Name:  "postgres-db",
		State: sdk.NodeStateWaiting,
		Properties: map[string]interface{}{
			"resource_type": "postgres",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := graph.AddNode(resourceNode); err != nil {
		t.Fatalf("Failed to add resource node: %v", err)
	}

	// Add edges (workflow contains step is the primary relationship in SDK)
	edge1 := &sdk.Edge{
		ID:         "edge-1",
		FromNodeID: "workflow-1",
		ToNodeID:   "step-1",
		Type:       sdk.EdgeTypeContains,
	}
	if err := graph.AddEdge(edge1); err != nil {
		t.Logf("Warning: failed to add edge 1: %v", err)
	}

	edge2 := &sdk.Edge{
		ID:          "edge-2",
		FromNodeID:  "step-1",
		ToNodeID:    "resource-1",
		Type:        sdk.EdgeTypeConfigures,
		Description: "Step configures resource",
	}
	if err := graph.AddEdge(edge2); err != nil {
		t.Logf("Warning: failed to add edge 2: %v", err)
	}

	// Debug: Check graph edges
	t.Logf("Graph has %d edges", len(graph.Edges))
	for id, edge := range graph.Edges {
		t.Logf("Edge %s: %s -> %s (type: %s)", id, edge.FromNodeID, edge.ToNodeID, edge.Type)
	}

	// Export to Mermaid
	exporter := NewMermaidExporter()
	mermaidDiagram, err := exporter.ExportGraph(graph)

	// Verify export succeeded
	if err != nil {
		t.Fatalf("ExportGraph() failed: %v", err)
	}

	// Verify Mermaid diagram structure
	if !strings.Contains(mermaidDiagram, "flowchart TD") {
		t.Error("Expected Mermaid diagram to start with 'flowchart TD'")
	}

	// Verify all nodes are present
	expectedNodes := []string{"spec_1", "workflow_1", "step_1", "resource_1"}
	for _, nodeID := range expectedNodes {
		if !strings.Contains(mermaidDiagram, nodeID) {
			t.Errorf("Expected diagram to contain node '%s'", nodeID)
		}
	}

	// Verify node names are present
	expectedNames := []string{"my-app", "deploy-app", "provision-database", "postgres-db"}
	for _, name := range expectedNames {
		if !strings.Contains(mermaidDiagram, name) {
			t.Errorf("Expected diagram to contain node name '%s'", name)
		}
	}

	// Verify edges are present
	if !strings.Contains(mermaidDiagram, "-->") {
		t.Error("Expected diagram to contain edges (-->)")
	}

	// Verify styling is present
	if !strings.Contains(mermaidDiagram, "classDef") {
		t.Error("Expected diagram to contain styling definitions")
	}

	// Verify state indicators
	if !strings.Contains(mermaidDiagram, "✓") || !strings.Contains(mermaidDiagram, "▶") {
		t.Error("Expected diagram to contain state icons")
	}

	t.Log("Generated Mermaid diagram:")
	t.Log(mermaidDiagram)
}

func TestMermaidExporter_ExportGraphSimple(t *testing.T) {
	// Create a sample graph
	graph := sdk.NewGraph("test-app")

	// Add minimal nodes (workflow and step)
	if err := graph.AddNode(&sdk.Node{
		ID:    "workflow-1",
		Type:  sdk.NodeTypeWorkflow,
		Name:  "Deploy Workflow",
		State: sdk.NodeStateRunning,
	}); err != nil {
		t.Fatalf("Failed to add workflow node: %v", err)
	}

	if err := graph.AddNode(&sdk.Node{
		ID:    "step-1",
		Type:  sdk.NodeTypeStep,
		Name:  "Provision Step",
		State: sdk.NodeStateSucceeded,
	}); err != nil {
		t.Fatalf("Failed to add step node: %v", err)
	}

	// Add edge (workflow contains step)
	if err := graph.AddEdge(&sdk.Edge{
		ID:         "edge-1",
		FromNodeID: "workflow-1",
		ToNodeID:   "step-1",
		Type:       sdk.EdgeTypeContains,
	}); err != nil {
		t.Logf("Warning: failed to add edge: %v", err)
	}

	// Export to simple Mermaid format
	exporter := NewMermaidExporter()
	mermaidDiagram, err := exporter.ExportGraphSimple(graph)

	// Verify export succeeded
	if err != nil {
		t.Fatalf("ExportGraphSimple() failed: %v", err)
	}

	// Verify it's a horizontal layout
	if !strings.Contains(mermaidDiagram, "flowchart LR") {
		t.Error("Expected simple diagram to use horizontal layout (LR)")
	}

	// Verify nodes are present
	if !strings.Contains(mermaidDiagram, "workflow_1") || !strings.Contains(mermaidDiagram, "step_1") {
		t.Error("Expected diagram to contain sanitized node IDs")
	}

	// Verify edge is present
	if !strings.Contains(mermaidDiagram, "-->") {
		t.Error("Expected diagram to contain edge")
	}

	// Verify it's simpler (no styling)
	if strings.Contains(mermaidDiagram, "classDef") {
		t.Error("Simple diagram should not contain styling definitions")
	}

	t.Log("Generated simple Mermaid diagram:")
	t.Log(mermaidDiagram)
}

func TestMermaidExporter_SanitizeID(t *testing.T) {
	exporter := NewMermaidExporter()

	tests := []struct {
		input    string
		expected string
	}{
		{"spec:my-app", "spec_my_app"},
		{"workflow-123", "workflow_123"},
		{"step.1", "step_1"},
		{"resource:postgres:db", "resource_postgres_db"},
		{"node with spaces", "node_with_spaces"},
	}

	for _, test := range tests {
		result := exporter.sanitizeID(test.input)
		if result != test.expected {
			t.Errorf("sanitizeID(%s) = %s, want %s", test.input, result, test.expected)
		}
	}
}

func TestMermaidExporter_GetStateIcon(t *testing.T) {
	exporter := NewMermaidExporter()

	tests := []struct {
		state        sdk.NodeState
		expectedIcon string
	}{
		{sdk.NodeStateSucceeded, "✓"},
		{sdk.NodeStateFailed, "✗"},
		{sdk.NodeStateRunning, "▶"},
		{sdk.NodeStateWaiting, "⏸"},
		{sdk.NodeStatePending, "○"},
	}

	for _, test := range tests {
		icon := exporter.getStateIcon(test.state)
		if icon != test.expectedIcon {
			t.Errorf("getStateIcon(%s) = %s, want %s", test.state, icon, test.expectedIcon)
		}
	}
}

func TestMermaidExporter_NilGraph(t *testing.T) {
	exporter := NewMermaidExporter()

	_, err := exporter.ExportGraph(nil)
	if err == nil {
		t.Error("Expected error when exporting nil graph")
	}

	_, err = exporter.ExportGraphSimple(nil)
	if err == nil {
		t.Error("Expected error when exporting nil graph (simple)")
	}
}
