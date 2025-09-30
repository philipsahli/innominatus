package graph

import (
	"regexp"
	"innominatus/internal/types"
	"time"
)

// NodeType represents the type of node in the resource graph
type NodeType string

const (
	NodeTypeSpec     NodeType = "spec"
	NodeTypeWorkflow NodeType = "workflow"
	NodeTypeResource NodeType = "resource"
)

// NodeStatus represents the status of a node
type NodeStatus string

const (
	NodeStatusPending    NodeStatus = "pending"
	NodeStatusRunning    NodeStatus = "running"
	NodeStatusCompleted  NodeStatus = "completed"
	NodeStatusFailed     NodeStatus = "failed"
	NodeStatusUnknown    NodeStatus = "unknown"
)

// GraphNode represents a node in the resource graph
type GraphNode struct {
	ID          string                 `json:"id"`
	Type        NodeType               `json:"type"`
	Name        string                 `json:"name"`
	Status      NodeStatus             `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
	Description string                 `json:"description,omitempty"`
}

// GraphEdge represents an edge/connection between nodes
type GraphEdge struct {
	ID          string                 `json:"id"`
	SourceID    string                 `json:"source_id"`
	TargetID    string                 `json:"target_id"`
	Type        string                 `json:"type"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// Graph represents the complete resource graph
type Graph struct {
	AppName   string       `json:"app_name"`
	Nodes     []*GraphNode `json:"nodes"`
	Edges     []*GraphEdge `json:"edges"`
	Timestamp time.Time    `json:"timestamp"`
}

// BuildResourceGraph builds a comprehensive resource graph for an application
func BuildResourceGraph(appName string, scoreSpec *types.ScoreSpec) *Graph {
	graph := &Graph{
		AppName:   appName,
		Nodes:     []*GraphNode{},
		Edges:     []*GraphEdge{},
		Timestamp: time.Now(),
	}

	// Add Score specification node
	specNode := &GraphNode{
		ID:          "spec:" + appName,
		Type:        NodeTypeSpec,
		Name:        appName + " Score Spec",
		Status:      NodeStatusCompleted,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Description: "Score specification defining application requirements",
		Metadata: map[string]interface{}{
			"spec_name": appName,
			"version":   "1.0",
		},
	}
	graph.Nodes = append(graph.Nodes, specNode)

	// Add resource nodes from Score spec
	for resourceName, resource := range scoreSpec.Resources {
		resourceNode := &GraphNode{
			ID:          "resource:" + resourceName,
			Type:        NodeTypeResource,
			Name:        resourceName,
			Status:      NodeStatusUnknown, // Will be updated with actual resource state
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Description: "Infrastructure resource defined in Score spec",
			Metadata: map[string]interface{}{
				"resource_name": resourceName,
				"resource_type": resource.Type,
			},
		}
		graph.Nodes = append(graph.Nodes, resourceNode)

		// Create edge from spec to resource
		edge := &GraphEdge{
			ID:          "spec-resource:" + resourceName,
			SourceID:    specNode.ID,
			TargetID:    resourceNode.ID,
			Type:        "defines",
			Description: "Score spec defines this resource",
		}
		graph.Edges = append(graph.Edges, edge)
	}

	return graph
}

// AddWorkflowNodes adds workflow-related nodes to the graph
func (g *Graph) AddWorkflowNodes(workflowName string, status string, steps []string) {
	workflowNode := &GraphNode{
		ID:          "workflow:" + workflowName,
		Type:        NodeTypeWorkflow,
		Name:        workflowName,
		Status:      mapWorkflowStatus(status),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Description: "Workflow execution for application deployment",
		Metadata: map[string]interface{}{
			"workflow_name": workflowName,
			"steps":         steps,
		},
	}
	g.Nodes = append(g.Nodes, workflowNode)

	// Connect spec to workflow
	if specNode := g.findNodeByType(NodeTypeSpec); specNode != nil {
		edge := &GraphEdge{
			ID:          "spec-workflow:" + workflowName,
			SourceID:    specNode.ID,
			TargetID:    workflowNode.ID,
			Type:        "triggers",
			Description: "Score spec triggers workflow execution",
		}
		g.Edges = append(g.Edges, edge)
	}

	// Connect workflow to resources (workflow provisions/manages resources)
	for _, node := range g.Nodes {
		if node.Type == NodeTypeResource {
			edge := &GraphEdge{
				ID:          "workflow-resource:" + workflowName + "-" + node.Name,
				SourceID:    workflowNode.ID,
				TargetID:    node.ID,
				Type:        "provisions",
				Description: "Workflow provisions this resource",
			}
			g.Edges = append(g.Edges, edge)
		}
	}
}

// UpdateResourceStatus updates the status of resource nodes based on actual infrastructure state
func (g *Graph) UpdateResourceStatus(resourceName string, status NodeStatus, metadata map[string]interface{}) {
	for _, node := range g.Nodes {
		if node.Type == NodeTypeResource && node.Name == resourceName {
			node.Status = status
			node.UpdatedAt = time.Now()
			if metadata != nil {
				for k, v := range metadata {
					node.Metadata[k] = v
				}
			}
			break
		}
	}
}

// DetectPostgresResources finds PostgreSQL resources in the Score spec
func (g *Graph) DetectPostgresResources() []*GraphNode {
	var postgresNodes []*GraphNode
	for _, node := range g.Nodes {
		if node.Type == NodeTypeResource {
			if resourceType, ok := node.Metadata["resource_type"].(string); ok {
				if resourceType == "postgres" || resourceType == "postgresql" {
					postgresNodes = append(postgresNodes, node)
				}
			}
		}
	}
	return postgresNodes
}

// Helper functions
func (g *Graph) findNodeByType(nodeType NodeType) *GraphNode {
	for _, node := range g.Nodes {
		if node.Type == nodeType {
			return node
		}
	}
	return nil
}

func mapWorkflowStatus(status string) NodeStatus {
	switch status {
	case "completed":
		return NodeStatusCompleted
	case "failed":
		return NodeStatusFailed
	case "running":
		return NodeStatusRunning
	default:
		return NodeStatusPending
	}
}

func BuildGraph(scoreSpec *types.ScoreSpec) map[string][]string {
	graph := make(map[string][]string)
	
	// Parse container variables to find resource references
	for containerName, container := range scoreSpec.Containers {
		containerKey := "container:" + containerName
		var dependencies []string
		
		// Look for ${resources.resourceName.*} patterns in variables
		resourcePattern := regexp.MustCompile(`\$\{resources\.([^.}]+)`)
		
		for _, value := range container.Variables {
			matches := resourcePattern.FindAllStringSubmatch(value, -1)
			for _, match := range matches {
				if len(match) > 1 {
					resourceName := match[1]
					// Check if this resource actually exists in the spec
					if _, exists := scoreSpec.Resources[resourceName]; exists {
						// Avoid duplicates
						if !contains(dependencies, resourceName) {
							dependencies = append(dependencies, resourceName)
						}
					}
				}
			}
		}
		
		if len(dependencies) > 0 {
			graph[containerKey] = dependencies
		}
	}
	
	// Add environment node and connect it to all resources
	if scoreSpec.Environment != nil {
		var resourceList []string
		for resourceName := range scoreSpec.Resources {
			resourceList = append(resourceList, resourceName)
		}
		if len(resourceList) > 0 {
			graph["environment"] = resourceList
		}
	}
	
	return graph
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}