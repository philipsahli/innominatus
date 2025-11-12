package events

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// subscription represents a single event subscriber
type subscription struct {
	id         string
	appName    string
	eventTypes map[EventType]bool // For fast lookup
	handler    EventHandler
	eventChan  chan Event
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// eventBus implements the EventBus interface with channel-based pub/sub
type eventBus struct {
	subscriptions map[string]*subscription
	mu            sync.RWMutex
	closed        bool
	closeChan     chan struct{}
	wg            sync.WaitGroup //nolint:unused // Reserved for future coordination
}

// NewEventBus creates a new event bus instance
func NewEventBus() EventBus {
	bus := &eventBus{
		subscriptions: make(map[string]*subscription),
		closeChan:     make(chan struct{}),
	}

	log.Debug().Msg("Event bus created")
	return bus
}

// Publish sends an event to all matching subscribers
func (b *eventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		log.Warn().
			Str("event_type", string(event.Type)).
			Str("app_name", event.AppName).
			Msg("Attempted to publish to closed event bus")
		return
	}

	// Count matching subscribers for logging
	matchCount := 0

	// Send to all matching subscribers
	for _, sub := range b.subscriptions {
		// Check if this subscription matches the event
		if sub.appName != "" && sub.appName != event.AppName {
			continue // App name filter doesn't match
		}

		if len(sub.eventTypes) > 0 && !sub.eventTypes[event.Type] {
			continue // Event type filter doesn't match
		}

		// Try to send event (non-blocking)
		select {
		case sub.eventChan <- event:
			matchCount++
		default:
			// Channel full - subscriber is slow
			log.Warn().
				Str("subscription_id", sub.id).
				Str("event_type", string(event.Type)).
				Msg("Subscriber channel full, dropping event")
		}
	}

	log.Debug().
		Str("event_type", string(event.Type)).
		Str("app_name", event.AppName).
		Str("source", event.Source).
		Int("subscribers", matchCount).
		Msg("Event published")
}

// Subscribe registers a handler for events matching the criteria
func (b *eventBus) Subscribe(appName string, eventTypes []EventType, handler EventHandler) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		log.Warn().Msg("Attempted to subscribe to closed event bus")
		return ""
	}

	// Generate unique subscription ID
	subID := fmt.Sprintf("sub-%d", time.Now().UnixNano())

	// Convert event types slice to map for fast lookup
	eventTypeMap := make(map[EventType]bool)
	for _, et := range eventTypes {
		eventTypeMap[et] = true
	}

	// Create subscription
	sub := &subscription{
		id:         subID,
		appName:    appName,
		eventTypes: eventTypeMap,
		handler:    handler,
		eventChan:  make(chan Event, 256), // Buffered to prevent blocking publishers
		stopChan:   make(chan struct{}),
	}

	// Start event handler goroutine
	sub.wg.Add(1)
	go b.runSubscription(sub)

	b.subscriptions[subID] = sub

	log.Info().
		Str("subscription_id", subID).
		Str("app_name", appName).
		Int("event_types", len(eventTypes)).
		Msg("New subscriber registered")

	return subID
}

// runSubscription processes events for a single subscription
func (b *eventBus) runSubscription(sub *subscription) {
	defer sub.wg.Done()

	for {
		select {
		case event := <-sub.eventChan:
			// Call handler in a goroutine to prevent blocking
			go func(e Event) {
				defer func() {
					if r := recover(); r != nil {
						log.Error().
							Str("subscription_id", sub.id).
							Interface("panic", r).
							Msg("Panic in event handler")
					}
				}()

				sub.handler(e)
			}(event)

		case <-sub.stopChan:
			log.Debug().
				Str("subscription_id", sub.id).
				Msg("Subscription stopped")
			return

		case <-b.closeChan:
			log.Debug().
				Str("subscription_id", sub.id).
				Msg("Event bus closed, stopping subscription")
			return
		}
	}
}

// Unsubscribe removes a subscription by ID
func (b *eventBus) Unsubscribe(subscriptionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub, exists := b.subscriptions[subscriptionID]
	if !exists {
		log.Warn().
			Str("subscription_id", subscriptionID).
			Msg("Attempted to unsubscribe non-existent subscription")
		return
	}

	// Stop the subscription goroutine
	close(sub.stopChan)

	// Wait for goroutine to finish
	sub.wg.Wait()

	// Remove from map
	delete(b.subscriptions, subscriptionID)

	log.Info().
		Str("subscription_id", subscriptionID).
		Msg("Subscriber unregistered")
}

// Close shuts down the event bus and all subscriptions
func (b *eventBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true
	close(b.closeChan)

	log.Info().
		Int("active_subscriptions", len(b.subscriptions)).
		Msg("Closing event bus")

	// Stop all subscriptions
	for id, sub := range b.subscriptions {
		close(sub.stopChan)
		sub.wg.Wait()
		delete(b.subscriptions, id)
	}

	log.Info().Msg("Event bus closed")
}
