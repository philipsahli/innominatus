package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, you should validate the origin
		return true
	},
}

// GraphWebSocketHub manages WebSocket connections for graph updates
type GraphWebSocketHub struct {
	// Registered clients mapped by application name
	clients map[string]map[*websocket.Conn]bool

	// Broadcast channel for graph updates
	broadcast chan GraphUpdate

	// Register requests from clients
	register chan ClientRegistration

	// Unregister requests from clients
	unregister chan ClientRegistration

	mu sync.RWMutex
}

// GraphUpdate represents a graph state update to broadcast
type GraphUpdate struct {
	AppName string
	Graph   interface{}
	Event   *GraphEvent `json:"event,omitempty"` // Optional event metadata for activity feed
}

// GraphEvent represents a specific change event in the graph
type GraphEvent struct {
	Type      string                 `json:"type"`       // node_added, node_state_changed, node_updated, edge_added, graph_updated
	Timestamp string                 `json:"timestamp"`  // RFC3339 timestamp
	NodeID    string                 `json:"node_id,omitempty"`
	NodeType  string                 `json:"node_type,omitempty"`
	NodeName  string                 `json:"node_name,omitempty"`
	OldState  string                 `json:"old_state,omitempty"`
	NewState  string                 `json:"new_state,omitempty"`
	EdgeID    string                 `json:"edge_id,omitempty"`
	EdgeType  string                 `json:"edge_type,omitempty"`
	FromNode  string                 `json:"from_node,omitempty"`
	ToNode    string                 `json:"to_node,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ClientRegistration represents a client connection registration
type ClientRegistration struct {
	AppName string
	Conn    *websocket.Conn
}

// NewGraphWebSocketHub creates a new hub for managing graph WebSocket connections
func NewGraphWebSocketHub() *GraphWebSocketHub {
	return &GraphWebSocketHub{
		clients:    make(map[string]map[*websocket.Conn]bool),
		broadcast:  make(chan GraphUpdate, 256),
		register:   make(chan ClientRegistration),
		unregister: make(chan ClientRegistration),
	}
}

// Run starts the hub's main loop
func (h *GraphWebSocketHub) Run() {
	for {
		select {
		case registration := <-h.register:
			h.mu.Lock()
			if h.clients[registration.AppName] == nil {
				h.clients[registration.AppName] = make(map[*websocket.Conn]bool)
			}
			h.clients[registration.AppName][registration.Conn] = true
			h.mu.Unlock()

		case registration := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[registration.AppName]; ok {
				if _, ok := clients[registration.Conn]; ok {
					delete(clients, registration.Conn)
					_ = registration.Conn.Close()
				}
				// Clean up empty app maps
				if len(clients) == 0 {
					delete(h.clients, registration.AppName)
				}
			}
			h.mu.Unlock()

		case update := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[update.AppName]
			h.mu.RUnlock()

			// Create message payload with graph and optional event
			payload := map[string]interface{}{
				"graph": update.Graph,
			}
			if update.Event != nil {
				payload["event"] = update.Event
			}

			message, err := json.Marshal(payload)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to marshal graph update: %v\n", err)
				continue
			}

			// Send to all clients watching this app
			for conn := range clients {
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					// Client disconnected, unregister it
					h.unregister <- ClientRegistration{
						AppName: update.AppName,
						Conn:    conn,
					}
				}
			}
		}
	}
}

// BroadcastGraphUpdate sends a graph update to all connected clients for an app
func (h *GraphWebSocketHub) BroadcastGraphUpdate(appName string, graph interface{}) {
	select {
	case h.broadcast <- GraphUpdate{AppName: appName, Graph: graph}:
	case <-time.After(time.Second):
		fmt.Fprintf(os.Stderr, "timeout broadcasting graph update for app: %s\n", appName)
	}
}

// BroadcastGraphUpdateWithEvent sends a graph update with event metadata to all connected clients
func (h *GraphWebSocketHub) BroadcastGraphUpdateWithEvent(appName string, graph interface{}, event interface{}) {
	// Convert event map to GraphEvent struct if needed
	var graphEvent *GraphEvent
	if eventMap, ok := event.(map[string]interface{}); ok {
		graphEvent = &GraphEvent{
			Type:      getStringFromMap(eventMap, "type"),
			Timestamp: getStringFromMap(eventMap, "timestamp"),
			NodeID:    getStringFromMap(eventMap, "node_id"),
			NodeType:  getStringFromMap(eventMap, "node_type"),
			NodeName:  getStringFromMap(eventMap, "node_name"),
			OldState:  getStringFromMap(eventMap, "old_state"),
			NewState:  getStringFromMap(eventMap, "new_state"),
			EdgeID:    getStringFromMap(eventMap, "edge_id"),
			EdgeType:  getStringFromMap(eventMap, "edge_type"),
			FromNode:  getStringFromMap(eventMap, "from_node"),
			ToNode:    getStringFromMap(eventMap, "to_node"),
			Metadata:  eventMap,
		}
	}

	select {
	case h.broadcast <- GraphUpdate{AppName: appName, Graph: graph, Event: graphEvent}:
	case <-time.After(time.Second):
		fmt.Fprintf(os.Stderr, "timeout broadcasting graph update with event for app: %s\n", appName)
	}
}

// Helper function to safely get string from map
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// handleGraphWebSocket handles WebSocket connections for real-time graph updates
func (s *Server) handleGraphWebSocket(w http.ResponseWriter, r *http.Request, appName string) {
	// Authenticate the connection using existing session system
	_, exists := s.getSessionFromRequestWithToken(r)
	if !exists {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to upgrade websocket: %v\n", err)
		return
	}

	// Register the client
	s.wsHub.register <- ClientRegistration{
		AppName: appName,
		Conn:    conn,
	}

	// Send initial graph state
	if s.graphAdapter != nil {
		graph, err := s.graphAdapter.GetGraph(appName)
		if err == nil {
			graphJSON := convertSDKGraphToFrontend(graph)
			// Send in same format as updates (with graph field, no event for initial load)
			payload := map[string]interface{}{
				"graph": graphJSON,
			}
			message, err := json.Marshal(payload)
			if err == nil {
				_ = conn.WriteMessage(websocket.TextMessage, message)
			}
		}
	}

	// Keep connection alive and handle pings
	go func() {
		defer func() {
			s.wsHub.unregister <- ClientRegistration{
				AppName: appName,
				Conn:    conn,
			}
		}()

		// Set up ping/pong handlers for connection health
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		// Read loop (for ping/pong and potential client messages)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()

	// Send periodic pings
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return
		}
	}
}
