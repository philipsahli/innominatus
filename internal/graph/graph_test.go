package graph

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"innominatus/internal/types"
)

func TestBuildResourceGraph(t *testing.T) {
	scoreSpec := &types.ScoreSpec{
		APIVersion: "score.dev/v1b1",
		Metadata: types.Metadata{
			Name: "test-app",
		},
		Containers: map[string]types.Container{
			"web": {
				Image: "nginx:latest",
			},
		},
		Resources: map[string]types.Resource{
			"db": {
				Type: "postgres",
				Params: map[string]interface{}{
					"version": "13",
				},
			},
			"cache": {
				Type: "redis",
				Params: map[string]interface{}{
					"version": "6",
				},
			},
		},
	}

	graph := BuildResourceGraph("test-app", scoreSpec)

	require.NotNil(t, graph)
	assert.Equal(t, "test-app", graph.AppName)
	assert.NotEmpty(t, graph.Timestamp)

	// Should have 1 spec node + 2 resource nodes = 3 nodes
	assert.Len(t, graph.Nodes, 3)

	// Should have 2 edges (spec -> db, spec -> cache)
	assert.Len(t, graph.Edges, 2)

	// Check spec node
	specNode := findNodeByID(graph.Nodes, "spec:test-app")
	require.NotNil(t, specNode)
	assert.Equal(t, NodeTypeSpec, specNode.Type)
	assert.Equal(t, "test-app Score Spec", specNode.Name)
	assert.Equal(t, NodeStatusCompleted, specNode.Status)

	// Check resource nodes
	dbNode := findNodeByID(graph.Nodes, "resource:db")
	require.NotNil(t, dbNode)
	assert.Equal(t, NodeTypeResource, dbNode.Type)
	assert.Equal(t, "db", dbNode.Name)
	assert.Equal(t, NodeStatusUnknown, dbNode.Status)
	assert.Equal(t, "postgres", dbNode.Metadata["resource_type"])

	cacheNode := findNodeByID(graph.Nodes, "resource:cache")
	require.NotNil(t, cacheNode)
	assert.Equal(t, NodeTypeResource, cacheNode.Type)
	assert.Equal(t, "cache", cacheNode.Name)
	assert.Equal(t, "redis", cacheNode.Metadata["resource_type"])

	// Check edges
	dbEdge := findEdgeByID(graph.Edges, "spec-resource:db")
	require.NotNil(t, dbEdge)
	assert.Equal(t, specNode.ID, dbEdge.SourceID)
	assert.Equal(t, dbNode.ID, dbEdge.TargetID)
	assert.Equal(t, "defines", dbEdge.Type)

	cacheEdge := findEdgeByID(graph.Edges, "spec-resource:cache")
	require.NotNil(t, cacheEdge)
	assert.Equal(t, specNode.ID, cacheEdge.SourceID)
	assert.Equal(t, cacheNode.ID, cacheEdge.TargetID)
}

func TestBuildResourceGraphEmptySpec(t *testing.T) {
	scoreSpec := &types.ScoreSpec{
		Metadata: types.Metadata{
			Name: "empty-app",
		},
		Containers: map[string]types.Container{},
		Resources:  map[string]types.Resource{},
	}

	graph := BuildResourceGraph("empty-app", scoreSpec)

	require.NotNil(t, graph)
	assert.Equal(t, "empty-app", graph.AppName)

	// Should have only 1 spec node
	assert.Len(t, graph.Nodes, 1)
	assert.Len(t, graph.Edges, 0)

	specNode := graph.Nodes[0]
	assert.Equal(t, NodeTypeSpec, specNode.Type)
	assert.Equal(t, "empty-app Score Spec", specNode.Name)
}

func TestAddWorkflowNodes(t *testing.T) {
	scoreSpec := &types.ScoreSpec{
		Metadata: types.Metadata{Name: "test-app"},
		Resources: map[string]types.Resource{
			"db": {Type: "postgres"},
		},
	}

	graph := BuildResourceGraph("test-app", scoreSpec)
	initialNodeCount := len(graph.Nodes)
	initialEdgeCount := len(graph.Edges)

	steps := []string{"terraform", "kubernetes", "validation"}
	graph.AddWorkflowNodes("deploy", "running", steps)

	// Should add 1 workflow node
	assert.Len(t, graph.Nodes, initialNodeCount+1)
	// Should add 2 edges (spec -> workflow, workflow -> resource)
	assert.Len(t, graph.Edges, initialEdgeCount+2)

	// Check workflow node
	workflowNode := findNodeByID(graph.Nodes, "workflow:deploy")
	require.NotNil(t, workflowNode)
	assert.Equal(t, NodeTypeWorkflow, workflowNode.Type)
	assert.Equal(t, "deploy", workflowNode.Name)
	assert.Equal(t, NodeStatusRunning, workflowNode.Status)
	assert.Equal(t, steps, workflowNode.Metadata["steps"])

	// Check spec -> workflow edge
	specWorkflowEdge := findEdgeByID(graph.Edges, "spec-workflow:deploy")
	require.NotNil(t, specWorkflowEdge)
	assert.Equal(t, "spec:test-app", specWorkflowEdge.SourceID)
	assert.Equal(t, "workflow:deploy", specWorkflowEdge.TargetID)
	assert.Equal(t, "triggers", specWorkflowEdge.Type)

	// Check workflow -> resource edge
	workflowResourceEdge := findEdgeByID(graph.Edges, "workflow-resource:deploy-db")
	require.NotNil(t, workflowResourceEdge)
	assert.Equal(t, "workflow:deploy", workflowResourceEdge.SourceID)
	assert.Equal(t, "resource:db", workflowResourceEdge.TargetID)
	assert.Equal(t, "provisions", workflowResourceEdge.Type)
}

func TestUpdateResourceStatus(t *testing.T) {
	scoreSpec := &types.ScoreSpec{
		Metadata: types.Metadata{Name: "test-app"},
		Resources: map[string]types.Resource{
			"db": {Type: "postgres"},
		},
	}

	graph := BuildResourceGraph("test-app", scoreSpec)

	metadata := map[string]interface{}{
		"endpoint": "db.example.com:5432",
		"version":  "13.4",
	}

	// Update resource status
	graph.UpdateResourceStatus("db", NodeStatusRunning, metadata)

	// Check that status was updated
	dbNode := findNodeByID(graph.Nodes, "resource:db")
	require.NotNil(t, dbNode)
	assert.Equal(t, NodeStatusRunning, dbNode.Status)
	assert.Equal(t, "db.example.com:5432", dbNode.Metadata["endpoint"])
	assert.Equal(t, "13.4", dbNode.Metadata["version"])

	// Test updating non-existent resource
	graph.UpdateResourceStatus("nonexistent", NodeStatusFailed, nil)
	// Should not panic and should not affect existing nodes
	assert.Equal(t, NodeStatusRunning, dbNode.Status)
}

func TestDetectPostgresResources(t *testing.T) {
	scoreSpec := &types.ScoreSpec{
		Metadata: types.Metadata{Name: "test-app"},
		Resources: map[string]types.Resource{
			"db":       {Type: "postgres"},
			"cache":    {Type: "redis"},
			"database": {Type: "postgresql"},
			"storage":  {Type: "volume"},
		},
	}

	graph := BuildResourceGraph("test-app", scoreSpec)
	postgresNodes := graph.DetectPostgresResources()

	// Should find 2 postgres resources
	assert.Len(t, postgresNodes, 2)

	nodeNames := make([]string, len(postgresNodes))
	for i, node := range postgresNodes {
		nodeNames[i] = node.Name
	}

	assert.Contains(t, nodeNames, "db")
	assert.Contains(t, nodeNames, "database")
	assert.NotContains(t, nodeNames, "cache")
	assert.NotContains(t, nodeNames, "storage")
}

func TestMapWorkflowStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected NodeStatus
	}{
		{"completed", NodeStatusCompleted},
		{"failed", NodeStatusFailed},
		{"running", NodeStatusRunning},
		{"unknown", NodeStatusPending},
		{"", NodeStatusPending},
		{"invalid", NodeStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapWorkflowStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildGraph(t *testing.T) {
	scoreSpec := &types.ScoreSpec{
		Containers: map[string]types.Container{
			"web": {
				Variables: map[string]string{
					"DB_HOST": "${resources.db.host}",
					"DB_PORT": "5432",
					"CACHE_URL": "${resources.cache.url}",
				},
			},
			"worker": {
				Variables: map[string]string{
					"REDIS_URL": "${resources.cache.url}",
					"STATIC_VAR": "static_value",
				},
			},
		},
		Resources: map[string]types.Resource{
			"db":    {Type: "postgres"},
			"cache": {Type: "redis"},
		},
		Environment: &types.Environment{
			Type: "kubernetes",
		},
	}

	graph := BuildGraph(scoreSpec)

	// Check web container dependencies
	webDeps := graph["container:web"]
	assert.Len(t, webDeps, 2)
	assert.Contains(t, webDeps, "db")
	assert.Contains(t, webDeps, "cache")

	// Check worker container dependencies
	workerDeps := graph["container:worker"]
	assert.Len(t, workerDeps, 1)
	assert.Contains(t, workerDeps, "cache")

	// Check environment dependencies
	envDeps := graph["environment"]
	assert.Len(t, envDeps, 2)
	assert.Contains(t, envDeps, "db")
	assert.Contains(t, envDeps, "cache")
}

func TestBuildGraphNoResourceReferences(t *testing.T) {
	scoreSpec := &types.ScoreSpec{
		Containers: map[string]types.Container{
			"web": {
				Variables: map[string]string{
					"PORT": "8080",
					"ENV":  "production",
				},
			},
		},
		Resources: map[string]types.Resource{
			"db": {Type: "postgres"},
		},
	}

	graph := BuildGraph(scoreSpec)

	// Should not have container dependencies since no ${resources.*} references
	assert.Empty(t, graph["container:web"])

	// Should not have environment node if no environment specified
	assert.Empty(t, graph["environment"])
}

func TestBuildGraphInvalidResourceReferences(t *testing.T) {
	scoreSpec := &types.ScoreSpec{
		Containers: map[string]types.Container{
			"web": {
				Variables: map[string]string{
					"DB_HOST": "${resources.nonexistent.host}",
					"VALID_REF": "${resources.db.host}",
				},
			},
		},
		Resources: map[string]types.Resource{
			"db": {Type: "postgres"},
		},
	}

	graph := BuildGraph(scoreSpec)

	// Should only include valid resource references
	webDeps := graph["container:web"]
	assert.Len(t, webDeps, 1)
	assert.Contains(t, webDeps, "db")
	assert.NotContains(t, webDeps, "nonexistent")
}

func TestGraphNodeStructure(t *testing.T) {
	node := &GraphNode{
		ID:          "test-node-1",
		Type:        NodeTypeResource,
		Name:        "test-resource",
		Status:      NodeStatusRunning,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Description: "Test resource node",
		Metadata: map[string]interface{}{
			"type": "postgres",
			"version": "13",
		},
	}

	assert.Equal(t, "test-node-1", node.ID)
	assert.Equal(t, NodeTypeResource, node.Type)
	assert.Equal(t, "test-resource", node.Name)
	assert.Equal(t, NodeStatusRunning, node.Status)
	assert.Equal(t, "Test resource node", node.Description)
	assert.Equal(t, "postgres", node.Metadata["type"])
	assert.Equal(t, "13", node.Metadata["version"])
}

func TestGraphEdgeStructure(t *testing.T) {
	edge := &GraphEdge{
		ID:          "test-edge-1",
		SourceID:    "node-1",
		TargetID:    "node-2",
		Type:        "depends_on",
		Description: "Test dependency edge",
		Metadata: map[string]interface{}{
			"strength": "strong",
		},
	}

	assert.Equal(t, "test-edge-1", edge.ID)
	assert.Equal(t, "node-1", edge.SourceID)
	assert.Equal(t, "node-2", edge.TargetID)
	assert.Equal(t, "depends_on", edge.Type)
	assert.Equal(t, "Test dependency edge", edge.Description)
	assert.Equal(t, "strong", edge.Metadata["strength"])
}

func TestContainsHelper(t *testing.T) {
	slice := []string{"a", "b", "c"}

	assert.True(t, contains(slice, "a"))
	assert.True(t, contains(slice, "b"))
	assert.True(t, contains(slice, "c"))
	assert.False(t, contains(slice, "d"))
	assert.False(t, contains(slice, ""))

	// Test empty slice
	emptySlice := []string{}
	assert.False(t, contains(emptySlice, "a"))
}

func TestFindNodeByType(t *testing.T) {
	graph := &Graph{
		Nodes: []*GraphNode{
			{ID: "spec-1", Type: NodeTypeSpec, Name: "spec"},
			{ID: "workflow-1", Type: NodeTypeWorkflow, Name: "workflow"},
			{ID: "resource-1", Type: NodeTypeResource, Name: "resource"},
		},
	}

	specNode := graph.findNodeByType(NodeTypeSpec)
	require.NotNil(t, specNode)
	assert.Equal(t, "spec-1", specNode.ID)

	workflowNode := graph.findNodeByType(NodeTypeWorkflow)
	require.NotNil(t, workflowNode)
	assert.Equal(t, "workflow-1", workflowNode.ID)

	resourceNode := graph.findNodeByType(NodeTypeResource)
	require.NotNil(t, resourceNode)
	assert.Equal(t, "resource-1", resourceNode.ID)

	// Test non-existent type
	nonExistent := graph.findNodeByType("nonexistent")
	assert.Nil(t, nonExistent)
}

// Helper functions for tests

func findNodeByID(nodes []*GraphNode, id string) *GraphNode {
	for _, node := range nodes {
		if node.ID == id {
			return node
		}
	}
	return nil
}

func findEdgeByID(edges []*GraphEdge, id string) *GraphEdge {
	for _, edge := range edges {
		if edge.ID == id {
			return edge
		}
	}
	return nil
}