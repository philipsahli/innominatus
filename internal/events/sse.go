package events

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// SSEClient represents a single Server-Sent Events client
type SSEClient struct {
	ID             string
	AppName        string // Empty string means all apps
	EventTypes     []EventType
	MessageChan    chan Event
	CloseChan      chan struct{}
	subscriptionID string
}

// SSEBroker manages SSE connections and broadcasts events
type SSEBroker struct {
	eventBus    EventBus
	clients     map[string]*SSEClient
	clientMutex sync.RWMutex
	stopChan    chan struct{}
}

// NewSSEBroker creates a new SSE broker
func NewSSEBroker(eventBus EventBus) *SSEBroker {
	return &SSEBroker{
		eventBus: eventBus,
		clients:  make(map[string]*SSEClient),
		stopChan: make(chan struct{}),
	}
}

// ServeHTTP handles SSE connections
func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Get query parameters
	appName := r.URL.Query().Get("app")
	eventTypesParam := r.URL.Query().Get("types")

	// Parse event types filter
	var eventTypes []EventType
	// TODO: Parse eventTypesParam when needed
	_ = eventTypesParam // Placeholder for future implementation

	// Create client
	client := &SSEClient{
		ID:          fmt.Sprintf("client-%d", time.Now().UnixNano()),
		AppName:     appName,
		EventTypes:  eventTypes,
		MessageChan: make(chan Event, 100),
		CloseChan:   make(chan struct{}),
	}

	// Register client
	b.clientMutex.Lock()
	b.clients[client.ID] = client
	b.clientMutex.Unlock()

	// Subscribe to event bus
	client.subscriptionID = b.eventBus.Subscribe(appName, eventTypes, func(event Event) {
		select {
		case client.MessageChan <- event:
		case <-client.CloseChan:
			// Client disconnected
		case <-time.After(1 * time.Second):
			log.Warn().
				Str("client_id", client.ID).
				Msg("SSE client channel full, dropping event")
		}
	})

	log.Info().
		Str("client_id", client.ID).
		Str("app_name", appName).
		Msg("SSE client connected")

	// Send initial connection message
	if _, err := fmt.Fprintf(w, "data: {\"type\":\"connected\",\"client_id\":\"%s\"}\n\n", client.ID); err != nil {
		log.Warn().Err(err).Str("client_id", client.ID).Msg("Failed to send connection message")
	}
	flusher.Flush()

	// Handle client lifecycle
	ctx := r.Context()
	defer func() {
		// Cleanup on disconnect
		b.clientMutex.Lock()
		delete(b.clients, client.ID)
		b.clientMutex.Unlock()

		// Unsubscribe from event bus
		b.eventBus.Unsubscribe(client.subscriptionID)
		close(client.CloseChan)

		log.Info().
			Str("client_id", client.ID).
			Msg("SSE client disconnected")
	}()

	// Event loop
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return

		case <-b.stopChan:
			// Broker shutting down
			return

		case event := <-client.MessageChan:
			// Send event to client
			if _, err := fmt.Fprint(w, event.ToSSE()); err != nil {
				log.Warn().Err(err).Str("client_id", client.ID).Msg("Failed to send event")
				return
			}
			flusher.Flush()

		case <-time.After(30 * time.Second):
			// Send keepalive ping
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				log.Warn().Err(err).Str("client_id", client.ID).Msg("Failed to send keepalive")
				return
			}
			flusher.Flush()
		}
	}
}

// GetConnectedClients returns the number of connected clients
func (b *SSEBroker) GetConnectedClients() int {
	b.clientMutex.RLock()
	defer b.clientMutex.RUnlock()
	return len(b.clients)
}

// GetClientsByApp returns clients filtered by app name
func (b *SSEBroker) GetClientsByApp(appName string) []*SSEClient {
	b.clientMutex.RLock()
	defer b.clientMutex.RUnlock()

	var clients []*SSEClient
	for _, client := range b.clients {
		if client.AppName == appName || client.AppName == "" {
			clients = append(clients, client)
		}
	}
	return clients
}

// Close shuts down the SSE broker
func (b *SSEBroker) Close() {
	close(b.stopChan)

	b.clientMutex.Lock()
	defer b.clientMutex.Unlock()

	// Close all client connections
	for id, client := range b.clients {
		close(client.CloseChan)
		b.eventBus.Unsubscribe(client.subscriptionID)
		delete(b.clients, id)
	}

	log.Info().Msg("SSE broker closed")
}

// BroadcastMessage sends a custom message to all connected clients
func (b *SSEBroker) BroadcastMessage(message string) {
	b.clientMutex.RLock()
	defer b.clientMutex.RUnlock()

	for _, client := range b.clients {
		select {
		case client.MessageChan <- NewEvent(
			"broadcast",
			"",
			"sse-broker",
			map[string]interface{}{"message": message},
		):
		default:
			log.Warn().
				Str("client_id", client.ID).
				Msg("Failed to send broadcast message to client")
		}
	}
}
