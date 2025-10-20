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

			message, err := json.Marshal(update.Graph)
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
			message, err := json.Marshal(graphJSON)
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
