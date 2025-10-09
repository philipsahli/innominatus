package graph

import (
	"fmt"
	"strings"

	sdk "github.com/philipsahli/innominatus-graph/pkg/graph"
)

// MermaidExporter generates Mermaid diagrams from graph data
type MermaidExporter struct{}

// NewMermaidExporter creates a new Mermaid exporter
func NewMermaidExporter() *MermaidExporter {
	return &MermaidExporter{}
}

// ExportGraph exports a graph to Mermaid flowchart format
func (m *MermaidExporter) ExportGraph(graph *sdk.Graph) (string, error) {
	if graph == nil {
		return "", fmt.Errorf("graph cannot be nil")
	}

	var sb strings.Builder

	// Start Mermaid flowchart
	sb.WriteString("flowchart TD\n")
	sb.WriteString("    %% Workflow Execution Graph\n\n")

	// Add nodes with styling
	sb.WriteString("    %% Nodes\n")
	for _, node := range graph.Nodes {
		nodeStyle := m.getNodeStyle(node.Type, node.State)
		nodeLabel := m.formatNodeLabel(node.Name, node.Type, node.State)
		sb.WriteString(fmt.Sprintf("    %s%s%s\n", m.sanitizeID(node.ID), nodeStyle, nodeLabel))
	}

	sb.WriteString("\n    %% Edges\n")
	// Add edges with labels
	for _, edge := range graph.Edges {
		edgeLabel := m.formatEdgeLabel(edge.Type, edge.Description)
		sb.WriteString(fmt.Sprintf("    %s -->|%s| %s\n",
			m.sanitizeID(edge.FromNodeID),
			edgeLabel,
			m.sanitizeID(edge.ToNodeID)))
	}

	// Add styling classes
	sb.WriteString("\n    %% Styling\n")
	sb.WriteString("    classDef spec fill:#3b82f6,stroke:#2563eb,stroke-width:2px,color:#fff\n")
	sb.WriteString("    classDef workflow fill:#eab308,stroke:#ca8a04,stroke-width:2px,color:#fff\n")
	sb.WriteString("    classDef step fill:#fb923c,stroke:#ea580c,stroke-width:2px,color:#fff\n")
	sb.WriteString("    classDef resource fill:#22c55e,stroke:#16a34a,stroke-width:2px,color:#fff\n")
	sb.WriteString("    classDef failed fill:#ef4444,stroke:#dc2626,stroke-width:3px,color:#fff\n")
	sb.WriteString("    classDef running fill:#06b6d4,stroke:#0891b2,stroke-width:2px,color:#fff,stroke-dasharray: 5 5\n")
	sb.WriteString("    classDef waiting fill:#9ca3af,stroke:#6b7280,stroke-width:2px,color:#fff\n")

	// Apply classes to nodes
	sb.WriteString("\n    %% Apply styles\n")
	for _, node := range graph.Nodes {
		className := m.getNodeClass(node.Type, node.State)
		sb.WriteString(fmt.Sprintf("    class %s %s\n", m.sanitizeID(node.ID), className))
	}

	return sb.String(), nil
}

// ExportGraphSimple exports a simplified Mermaid graph (LR layout, minimal labels)
func (m *MermaidExporter) ExportGraphSimple(graph *sdk.Graph) (string, error) {
	if graph == nil {
		return "", fmt.Errorf("graph cannot be nil")
	}

	var sb strings.Builder

	// Horizontal layout for simple view
	sb.WriteString("flowchart LR\n")
	sb.WriteString("    %% Simplified Workflow Graph\n\n")

	// Add nodes with minimal styling
	for _, node := range graph.Nodes {
		sb.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", m.sanitizeID(node.ID), node.Name))
	}

	sb.WriteString("\n")
	// Add edges
	for _, edge := range graph.Edges {
		sb.WriteString(fmt.Sprintf("    %s --> %s\n",
			m.sanitizeID(edge.FromNodeID),
			m.sanitizeID(edge.ToNodeID)))
	}

	return sb.String(), nil
}

// Helper functions

// sanitizeID ensures node IDs are valid Mermaid identifiers
func (m *MermaidExporter) sanitizeID(id string) string {
	// Replace invalid characters with underscores
	sanitized := strings.ReplaceAll(id, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	return sanitized
}

// formatNodeLabel creates a formatted label for a node
func (m *MermaidExporter) formatNodeLabel(name string, nodeType sdk.NodeType, state sdk.NodeState) string {
	// Escape quotes in name
	escapedName := strings.ReplaceAll(name, "\"", "'")

	// Add state indicator
	stateIcon := m.getStateIcon(state)

	return fmt.Sprintf("[\"%s %s<br/>%s\"]", stateIcon, escapedName, string(nodeType))
}

// getNodeStyle returns the Mermaid node shape based on type
func (m *MermaidExporter) getNodeStyle(nodeType sdk.NodeType, state sdk.NodeState) string {
	// Different shapes for different node types
	switch nodeType {
	case sdk.NodeTypeSpec:
		return "[" // Rectangle
	case sdk.NodeTypeWorkflow:
		return "{" // Hexagon (requires closing })
	case sdk.NodeTypeStep:
		return "[" // Rectangle
	case sdk.NodeTypeResource:
		return "(" // Rounded rectangle
	default:
		return "["
	}
}

// formatEdgeLabel creates a label for an edge
func (m *MermaidExporter) formatEdgeLabel(edgeType sdk.EdgeType, description string) string {
	// Use edge type as primary label
	label := string(edgeType)

	// Map edge types to friendly labels
	switch edgeType {
	case sdk.EdgeTypeContains:
		label = "contains"
	case sdk.EdgeTypeDependsOn:
		label = "depends on"
	case sdk.EdgeTypeProvisions:
		label = "provisions"
	case sdk.EdgeTypeCreates:
		label = "creates"
	case sdk.EdgeTypeBindsTo:
		label = "binds to"
	case sdk.EdgeTypeConfigures:
		label = "configures"
	}

	return label
}

// getStateIcon returns an emoji/icon for node state
func (m *MermaidExporter) getStateIcon(state sdk.NodeState) string {
	switch state {
	case sdk.NodeStateSucceeded:
		return "✓"
	case sdk.NodeStateFailed:
		return "✗"
	case sdk.NodeStateRunning:
		return "▶"
	case sdk.NodeStateWaiting:
		return "⏸"
	case sdk.NodeStatePending:
		return "○"
	default:
		return "•"
	}
}

// getNodeClass returns the CSS class name for a node
func (m *MermaidExporter) getNodeClass(nodeType sdk.NodeType, state sdk.NodeState) string {
	// State takes precedence over type for styling
	switch state {
	case sdk.NodeStateFailed:
		return "failed"
	case sdk.NodeStateRunning:
		return "running"
	case sdk.NodeStateWaiting:
		return "waiting"
	}

	// Type-based styling for completed/pending nodes
	switch nodeType {
	case sdk.NodeTypeSpec:
		return "spec"
	case sdk.NodeTypeWorkflow:
		return "workflow"
	case sdk.NodeTypeStep:
		return "step"
	case sdk.NodeTypeResource:
		return "resource"
	default:
		return "waiting"
	}
}
