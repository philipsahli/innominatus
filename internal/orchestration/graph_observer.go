package orchestration

import (
	"innominatus/internal/graph"
	"innominatus/internal/logging"

	graphSDK "github.com/philipsahli/innominatus-graph/pkg/graph"
)

// GraphObserver implements the SDK's GraphObserver interface
// and forwards state changes to the WebSocket hub for real-time updates
type GraphObserver struct {
	graphAdapter *graph.Adapter
	wsHub        WebSocketHub
	logger       *logging.ZerologAdapter
}

// WebSocketHub defines the interface for broadcasting graph updates
// This allows us to avoid circular dependencies with the server package
type WebSocketHub interface {
	BroadcastGraphUpdate(appName string, graphData interface{})
}

// NewGraphObserver creates a new observer that bridges graph events to WebSocket broadcasts
func NewGraphObserver(graphAdapter *graph.Adapter, wsHub WebSocketHub) *GraphObserver {
	return &GraphObserver{
		graphAdapter: graphAdapter,
		wsHub:        wsHub,
		logger:       logging.NewStructuredLogger("graph-observer"),
	}
}

// OnNodeStateChanged is called when a node's state changes
func (o *GraphObserver) OnNodeStateChanged(g *graphSDK.Graph, nodeID string, oldState, newState graphSDK.NodeState) {
	o.logger.InfoWithFields("Node state changed", map[string]interface{}{
		"app_name":  g.AppName,
		"node_id":   nodeID,
		"old_state": oldState,
		"new_state": newState,
	})

	// Broadcast the full graph with updated state
	o.broadcastGraph(g)
}

// OnNodeUpdated is called when a node is updated (including timing changes)
func (o *GraphObserver) OnNodeUpdated(g *graphSDK.Graph, nodeID string) {
	o.logger.InfoWithFields("Node updated", map[string]interface{}{
		"app_name": g.AppName,
		"node_id":  nodeID,
	})

	// Broadcast the full graph with updated node
	o.broadcastGraph(g)
}

// OnEdgeAdded is called when an edge is added to the graph
func (o *GraphObserver) OnEdgeAdded(g *graphSDK.Graph, edge *graphSDK.Edge) {
	o.logger.InfoWithFields("Edge added", map[string]interface{}{
		"app_name":     g.AppName,
		"edge_id":      edge.ID,
		"from_node_id": edge.FromNodeID,
		"to_node_id":   edge.ToNodeID,
		"edge_type":    edge.Type,
	})

	// Broadcast the full graph with new edge
	o.broadcastGraph(g)
}

// OnGraphUpdated is called when the graph structure changes
func (o *GraphObserver) OnGraphUpdated(g *graphSDK.Graph) {
	o.logger.InfoWithFields("Graph updated", map[string]interface{}{
		"app_name":   g.AppName,
		"node_count": len(g.Nodes),
		"edge_count": len(g.Edges),
	})

	// Broadcast the updated graph
	o.broadcastGraph(g)
}

// broadcastGraph sends the graph to all connected WebSocket clients
func (o *GraphObserver) broadcastGraph(g *graphSDK.Graph) {
	// Convert SDK graph to frontend format
	graphData := convertSDKGraphToFrontend(g)

	// Broadcast to WebSocket clients
	if o.wsHub != nil {
		o.wsHub.BroadcastGraphUpdate(g.AppName, graphData)
	}
}

// convertSDKGraphToFrontend converts the SDK graph model to the format expected by the frontend
func convertSDKGraphToFrontend(g *graphSDK.Graph) map[string]interface{} {
	nodes := make([]map[string]interface{}, 0, len(g.Nodes))
	for _, node := range g.Nodes {
		nodeData := map[string]interface{}{
			"id":          node.ID,
			"type":        string(node.Type),
			"name":        node.Name,
			"description": node.Description,
			"state":       string(node.State),
			"properties":  node.Properties,
			"created_at":  node.CreatedAt,
			"updated_at":  node.UpdatedAt,
		}

		// Include timing information if available
		if node.StartedAt != nil {
			nodeData["started_at"] = node.StartedAt
		}
		if node.CompletedAt != nil {
			nodeData["completed_at"] = node.CompletedAt
		}
		if node.Duration != nil {
			nodeData["duration"] = node.Duration.String()
		}

		nodes = append(nodes, nodeData)
	}

	edges := make([]map[string]interface{}, 0, len(g.Edges))
	for _, edge := range g.Edges {
		edgeData := map[string]interface{}{
			"id":           edge.ID,
			"from_node_id": edge.FromNodeID,
			"to_node_id":   edge.ToNodeID,
			"type":         string(edge.Type),
			"properties":   edge.Properties,
			"created_at":   edge.CreatedAt,
		}
		edges = append(edges, edgeData)
	}

	return map[string]interface{}{
		"app_name":   g.AppName,
		"nodes":      nodes,
		"edges":      edges,
		"created_at": g.CreatedAt,
		"updated_at": g.UpdatedAt,
	}
}
