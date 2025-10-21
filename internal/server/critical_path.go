package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// CriticalPathNode represents a node in the critical path
type CriticalPathNode struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Duration float64 `json:"duration_seconds"`
	Weight   float64 `json:"weight"`
}

// CriticalPathResponse represents the critical path analysis result
type CriticalPathResponse struct {
	Application    string             `json:"application"`
	Path           []CriticalPathNode `json:"path"`
	TotalDuration  float64            `json:"total_duration_seconds"`
	NodeCount      int                `json:"node_count"`
	CalculatedAt   time.Time          `json:"calculated_at"`
	IsCriticalPath bool               `json:"is_critical_path"`
}

// handleCriticalPath handles /api/graph/<app>/critical-path requests
func (s *Server) handleCriticalPath(w http.ResponseWriter, r *http.Request, appName string) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the graph from the database
	if s.graphAdapter == nil {
		http.Error(w, "Graph adapter not available", http.StatusServiceUnavailable)
		return
	}

	graph, err := s.graphAdapter.GetGraph(appName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get graph: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate critical path
	criticalPath := s.calculateCriticalPath(graph)

	response := CriticalPathResponse{
		Application:    appName,
		Path:           criticalPath,
		TotalDuration:  calculateTotalDuration(criticalPath),
		NodeCount:      len(criticalPath),
		CalculatedAt:   time.Now(),
		IsCriticalPath: len(criticalPath) > 0,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// calculateCriticalPath computes the critical path through the workflow graph
func (s *Server) calculateCriticalPath(graph interface{}) []CriticalPathNode {
	// Type assertion to get the actual graph structure
	graphMap, ok := graph.(map[string]interface{})
	if !ok {
		return []CriticalPathNode{}
	}

	nodes, ok := graphMap["nodes"].(map[string]interface{})
	if !ok {
		return []CriticalPathNode{}
	}

	edges, ok := graphMap["edges"].([]interface{})
	if !ok {
		return []CriticalPathNode{}
	}

	// Build adjacency list and calculate node weights
	nodeWeights := make(map[string]float64)
	adjacencyList := make(map[string][]string)
	reverseAdjList := make(map[string][]string)
	inDegree := make(map[string]int)
	nodeData := make(map[string]map[string]interface{})

	// Initialize nodes
	for nodeID, nodeInfo := range nodes {
		nodeWeights[nodeID] = 1.0 // Default weight
		inDegree[nodeID] = 0
		adjacencyList[nodeID] = []string{}
		reverseAdjList[nodeID] = []string{}

		if nodeMap, ok := nodeInfo.(map[string]interface{}); ok {
			nodeData[nodeID] = nodeMap

			// Calculate weight based on node metadata
			if metadata, ok := nodeMap["metadata"].(map[string]interface{}); ok {
				if duration, ok := metadata["duration"].(float64); ok {
					nodeWeights[nodeID] = duration
				} else if duration, ok := metadata["duration"].(int); ok {
					nodeWeights[nodeID] = float64(duration)
				}
			}
		}
	}

	// Build adjacency lists
	for _, edge := range edges {
		if edgeMap, ok := edge.(map[string]interface{}); ok {
			source, sourceOk := edgeMap["source"].(string)
			target, targetOk := edgeMap["target"].(string)

			if sourceOk && targetOk {
				adjacencyList[source] = append(adjacencyList[source], target)
				reverseAdjList[target] = append(reverseAdjList[target], source)
				inDegree[target]++
			}
		}
	}

	// Find source nodes (nodes with no incoming edges)
	sourceNodes := []string{}
	for nodeID := range nodes {
		if inDegree[nodeID] == 0 {
			sourceNodes = append(sourceNodes, nodeID)
		}
	}

	if len(sourceNodes) == 0 {
		// No source nodes, graph might be cyclic or empty
		return []CriticalPathNode{}
	}

	// Calculate longest path using dynamic programming
	longestPath := make(map[string]float64)
	predecessor := make(map[string]string)

	// Initialize with source nodes
	for _, source := range sourceNodes {
		longestPath[source] = nodeWeights[source]
	}

	// Topological sort using Kahn's algorithm
	queue := make([]string, len(sourceNodes))
	copy(queue, sourceNodes)
	tempInDegree := make(map[string]int)
	for k, v := range inDegree {
		tempInDegree[k] = v
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, neighbor := range adjacencyList[current] {
			// Update longest path to neighbor
			newDist := longestPath[current] + nodeWeights[neighbor]
			if newDist > longestPath[neighbor] {
				longestPath[neighbor] = newDist
				predecessor[neighbor] = current
			}

			tempInDegree[neighbor]--
			if tempInDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Find the node with the maximum distance (end of critical path)
	var maxNode string
	var maxDist float64

	for nodeID, dist := range longestPath {
		if dist > maxDist {
			maxDist = dist
			maxNode = nodeID
		}
	}

	if maxNode == "" {
		return []CriticalPathNode{}
	}

	// Backtrack to build the critical path
	path := []CriticalPathNode{}
	current := maxNode

	for current != "" {
		nodeInfo := nodeData[current]
		nodeName := current
		nodeType := "unknown"

		if name, ok := nodeInfo["name"].(string); ok {
			nodeName = name
		}
		if ntype, ok := nodeInfo["type"].(string); ok {
			nodeType = ntype
		}

		path = append([]CriticalPathNode{{
			ID:       current,
			Name:     nodeName,
			Type:     nodeType,
			Duration: nodeWeights[current],
			Weight:   longestPath[current],
		}}, path...)

		current = predecessor[current]
	}

	return path
}

// calculateTotalDuration sums the durations in the critical path
func calculateTotalDuration(path []CriticalPathNode) float64 {
	total := 0.0
	for _, node := range path {
		total += node.Duration
	}
	return total
}
