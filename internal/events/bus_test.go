package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventBus_PublishSubscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	received := make(chan Event, 10)
	handler := func(event Event) {
		received <- event
	}

	// Subscribe to all events for app "test-app"
	subID := bus.Subscribe("test-app", nil, handler)
	require.NotEmpty(t, subID, "Subscription ID should not be empty")

	// Publish an event
	event := NewEvent(EventTypeResourceCreated, "test-app", "test-source", map[string]interface{}{
		"resource_name": "test-resource",
	})
	bus.Publish(event)

	// Wait for event to be received
	select {
	case receivedEvent := <-received:
		assert.Equal(t, event.ID, receivedEvent.ID)
		assert.Equal(t, event.Type, receivedEvent.Type)
		assert.Equal(t, event.AppName, receivedEvent.AppName)
		assert.Equal(t, "test-resource", receivedEvent.Data["resource_name"])
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	// Create multiple subscribers
	received1 := make(chan Event, 10)
	received2 := make(chan Event, 10)
	received3 := make(chan Event, 10)

	sub1 := bus.Subscribe("test-app", nil, func(e Event) { received1 <- e })
	sub2 := bus.Subscribe("test-app", nil, func(e Event) { received2 <- e })
	sub3 := bus.Subscribe("test-app", nil, func(e Event) { received3 <- e })

	require.NotEmpty(t, sub1)
	require.NotEmpty(t, sub2)
	require.NotEmpty(t, sub3)

	// Publish an event
	event := NewEvent(EventTypeResourceActive, "test-app", "test", nil)
	bus.Publish(event)

	// All subscribers should receive the event
	timeout := time.After(1 * time.Second)

	for i := 0; i < 3; i++ {
		select {
		case e := <-received1:
			assert.Equal(t, event.ID, e.ID)
		case e := <-received2:
			assert.Equal(t, event.ID, e.ID)
		case e := <-received3:
			assert.Equal(t, event.ID, e.ID)
		case <-timeout:
			t.Fatal("Timeout waiting for events")
		}
	}
}

func TestEventBus_FilterByAppName(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	receivedApp1 := make(chan Event, 10)
	receivedApp2 := make(chan Event, 10)

	// Subscribe to different apps
	bus.Subscribe("app1", nil, func(e Event) { receivedApp1 <- e })
	bus.Subscribe("app2", nil, func(e Event) { receivedApp2 <- e })

	// Publish events for different apps
	event1 := NewEvent(EventTypeResourceCreated, "app1", "test", nil)
	event2 := NewEvent(EventTypeResourceCreated, "app2", "test", nil)

	bus.Publish(event1)
	bus.Publish(event2)

	// Wait a bit for events to propagate
	time.Sleep(100 * time.Millisecond)

	// app1 subscriber should only receive app1 event
	select {
	case e := <-receivedApp1:
		assert.Equal(t, "app1", e.AppName)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for app1 event")
	}

	// app2 subscriber should only receive app2 event
	select {
	case e := <-receivedApp2:
		assert.Equal(t, "app2", e.AppName)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for app2 event")
	}

	// No additional events should be received
	select {
	case e := <-receivedApp1:
		t.Fatalf("app1 subscriber received unexpected event: %v", e)
	case e := <-receivedApp2:
		t.Fatalf("app2 subscriber received unexpected event: %v", e)
	case <-time.After(100 * time.Millisecond):
		// Good - no extra events
	}
}

func TestEventBus_FilterByEventType(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	receivedCreated := make(chan Event, 10)
	receivedActive := make(chan Event, 10)

	// Subscribe to specific event types
	bus.Subscribe("test-app", []EventType{EventTypeResourceCreated}, func(e Event) {
		receivedCreated <- e
	})
	bus.Subscribe("test-app", []EventType{EventTypeResourceActive}, func(e Event) {
		receivedActive <- e
	})

	// Publish different event types
	eventCreated := NewEvent(EventTypeResourceCreated, "test-app", "test", nil)
	eventActive := NewEvent(EventTypeResourceActive, "test-app", "test", nil)
	eventFailed := NewEvent(EventTypeResourceFailed, "test-app", "test", nil)

	bus.Publish(eventCreated)
	bus.Publish(eventActive)
	bus.Publish(eventFailed)

	// Wait for events to propagate
	time.Sleep(100 * time.Millisecond)

	// Verify correct filtering
	select {
	case e := <-receivedCreated:
		assert.Equal(t, EventTypeResourceCreated, e.Type)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for created event")
	}

	select {
	case e := <-receivedActive:
		assert.Equal(t, EventTypeResourceActive, e.Type)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for active event")
	}

	// No additional events should be received
	select {
	case e := <-receivedCreated:
		t.Fatalf("Received unexpected event: %v", e)
	case e := <-receivedActive:
		t.Fatalf("Received unexpected event: %v", e)
	case <-time.After(100 * time.Millisecond):
		// Good - failed event was filtered out
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	received := make(chan Event, 10)
	handler := func(e Event) { received <- e }

	subID := bus.Subscribe("test-app", nil, handler)

	// Publish event - should be received
	event1 := NewEvent(EventTypeResourceCreated, "test-app", "test", nil)
	bus.Publish(event1)

	select {
	case e := <-received:
		assert.Equal(t, event1.ID, e.ID)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for first event")
	}

	// Unsubscribe
	bus.Unsubscribe(subID)
	time.Sleep(100 * time.Millisecond) // Wait for unsubscribe to complete

	// Publish another event - should NOT be received
	event2 := NewEvent(EventTypeResourceActive, "test-app", "test", nil)
	bus.Publish(event2)

	select {
	case e := <-received:
		t.Fatalf("Received event after unsubscribe: %v", e)
	case <-time.After(200 * time.Millisecond):
		// Good - no event received
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	received := make(chan Event, 1000)
	handler := func(e Event) { received <- e }

	bus.Subscribe("test-app", nil, handler)

	// Publish many events concurrently
	numGoroutines := 10
	eventsPerGoroutine := 100
	totalEvents := numGoroutines * eventsPerGoroutine

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := NewEvent(EventTypeResourceCreated, "test-app", "test", map[string]interface{}{
					"goroutine": goroutineID,
					"event":     j,
				})
				bus.Publish(event)
			}
		}(i)
	}

	wg.Wait()

	// Wait for all events to be processed
	time.Sleep(500 * time.Millisecond)

	// Count received events
	receivedCount := len(received)
	assert.Equal(t, totalEvents, receivedCount, "Should receive all published events")
}

func TestEventBus_SubscribeToClosedBus(t *testing.T) {
	bus := NewEventBus()
	bus.Close()

	// Subscribing to closed bus should return empty subscription ID
	subID := bus.Subscribe("test-app", nil, func(e Event) {})
	assert.Empty(t, subID, "Should not allow subscription to closed bus")
}

func TestEventBus_PublishToClosedBus(t *testing.T) {
	bus := NewEventBus()
	received := make(chan Event, 10)
	bus.Subscribe("test-app", nil, func(e Event) { received <- e })

	bus.Close()

	// Publishing to closed bus should not panic
	event := NewEvent(EventTypeResourceCreated, "test-app", "test", nil)
	assert.NotPanics(t, func() {
		bus.Publish(event)
	}, "Publishing to closed bus should not panic")

	// No event should be received
	select {
	case e := <-received:
		t.Fatalf("Received event from closed bus: %v", e)
	case <-time.After(100 * time.Millisecond):
		// Good - no event received
	}
}

func TestEventBus_GracefulShutdown(t *testing.T) {
	bus := NewEventBus()

	// Create multiple subscribers
	for i := 0; i < 5; i++ {
		bus.Subscribe("test-app", nil, func(e Event) {
			// Simulate slow processing
			time.Sleep(50 * time.Millisecond)
		})
	}

	// Publish some events
	for i := 0; i < 10; i++ {
		event := NewEvent(EventTypeResourceCreated, "test-app", "test", nil)
		bus.Publish(event)
	}

	// Close should wait for all subscriptions to finish
	assert.NotPanics(t, func() {
		bus.Close()
	}, "Close should handle graceful shutdown")
}

func TestEventBus_PanicRecovery(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	received := make(chan Event, 10)

	// Subscribe with a handler that panics
	bus.Subscribe("test-app", nil, func(e Event) {
		panic("test panic")
	})

	// Subscribe with a normal handler
	bus.Subscribe("test-app", nil, func(e Event) {
		received <- e
	})

	// Publish event
	event := NewEvent(EventTypeResourceCreated, "test-app", "test", nil)
	bus.Publish(event)

	// Normal handler should still receive event despite other handler panicking
	select {
	case e := <-received:
		assert.Equal(t, event.ID, e.ID)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for event - panic recovery may have failed")
	}
}

func TestEvent_ToSSE(t *testing.T) {
	event := NewEvent(EventTypeResourceCreated, "test-app", "test-source", map[string]interface{}{
		"resource_name": "test-resource",
		"status":        "active",
	})

	sse := event.ToSSE()

	// Verify SSE format
	assert.Contains(t, sse, "data: {")
	assert.Contains(t, sse, "\"type\":\"resource.created\"")
	assert.Contains(t, sse, "\"app_name\":\"test-app\"")
	assert.Contains(t, sse, "\"source\":\"test-source\"")
	assert.Contains(t, sse, "\"resource_name\":\"test-resource\"")
	assert.Contains(t, sse, "\n\n", "SSE should end with double newline")
}

func TestEventBus_WildcardSubscription(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	received := make(chan Event, 10)

	// Subscribe with empty app name (wildcard - all apps)
	bus.Subscribe("", nil, func(e Event) {
		received <- e
	})

	// Publish events for different apps
	event1 := NewEvent(EventTypeResourceCreated, "app1", "test", nil)
	event2 := NewEvent(EventTypeResourceCreated, "app2", "test", nil)
	event3 := NewEvent(EventTypeResourceCreated, "app3", "test", nil)

	bus.Publish(event1)
	bus.Publish(event2)
	bus.Publish(event3)

	// Should receive all events
	receivedEvents := make(map[string]bool)
	timeout := time.After(1 * time.Second)

	for i := 0; i < 3; i++ {
		select {
		case e := <-received:
			receivedEvents[e.AppName] = true
		case <-timeout:
			t.Fatal("Timeout waiting for events")
		}
	}

	assert.True(t, receivedEvents["app1"], "Should receive app1 event")
	assert.True(t, receivedEvents["app2"], "Should receive app2 event")
	assert.True(t, receivedEvents["app3"], "Should receive app3 event")
}

func TestEventBus_MultipleEventTypeFilter(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	received := make(chan Event, 10)

	// Subscribe to multiple event types
	bus.Subscribe("test-app", []EventType{
		EventTypeResourceCreated,
		EventTypeResourceActive,
		EventTypeResourceFailed,
	}, func(e Event) {
		received <- e
	})

	// Publish various events
	events := []Event{
		NewEvent(EventTypeResourceCreated, "test-app", "test", nil),
		NewEvent(EventTypeResourceActive, "test-app", "test", nil),
		NewEvent(EventTypeResourceFailed, "test-app", "test", nil),
		NewEvent(EventTypeResourceProvisioning, "test-app", "test", nil), // Should be filtered out
		NewEvent(EventTypeWorkflowStarted, "test-app", "test", nil),      // Should be filtered out
	}

	for _, event := range events {
		bus.Publish(event)
	}

	// Wait for events
	time.Sleep(100 * time.Millisecond)

	// Should receive exactly 3 events (created, active, failed)
	receivedCount := len(received)
	assert.Equal(t, 3, receivedCount, "Should only receive matching event types")

	// Verify event types
	receivedTypes := make(map[EventType]bool)
	for i := 0; i < receivedCount; i++ {
		e := <-received
		receivedTypes[e.Type] = true
	}

	assert.True(t, receivedTypes[EventTypeResourceCreated])
	assert.True(t, receivedTypes[EventTypeResourceActive])
	assert.True(t, receivedTypes[EventTypeResourceFailed])
	assert.False(t, receivedTypes[EventTypeResourceProvisioning])
	assert.False(t, receivedTypes[EventTypeWorkflowStarted])
}
