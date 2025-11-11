package events

import (
	"encoding/json"
	"fmt"
	"time"
)

// EventType represents the type of deployment event
type EventType string

const (
	// Spec events
	EventTypeSpecCreated   EventType = "spec.created"
	EventTypeSpecValidated EventType = "spec.validated"

	// Resource lifecycle events
	EventTypeResourceCreated      EventType = "resource.created"
	EventTypeResourceRequested    EventType = "resource.requested"
	EventTypeResourceProvisioning EventType = "resource.provisioning"
	EventTypeResourceActive       EventType = "resource.active"
	EventTypeResourceFailed       EventType = "resource.failed"

	// Workflow lifecycle events
	EventTypeWorkflowCreated   EventType = "workflow.created"
	EventTypeWorkflowStarted   EventType = "workflow.started"
	EventTypeWorkflowCompleted EventType = "workflow.completed"
	EventTypeWorkflowFailed    EventType = "workflow.failed"

	// Step execution events
	EventTypeStepStarted   EventType = "step.started"
	EventTypeStepCompleted EventType = "step.completed"
	EventTypeStepFailed    EventType = "step.failed"
	EventTypeStepProgress  EventType = "step.progress"

	// Provider resolution
	EventTypeProviderResolved EventType = "provider.resolved"

	// Deployment lifecycle
	EventTypeDeploymentStarted   EventType = "deployment.started"
	EventTypeDeploymentCompleted EventType = "deployment.completed"
	EventTypeDeploymentFailed    EventType = "deployment.failed"
)

// Event represents a deployment event that can be streamed to watchers
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	AppName   string                 `json:"app_name"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Source    string                 `json:"source"` // e.g., "orchestration-engine", "workflow-executor"
}

// EventHandler is a callback function for handling events
type EventHandler func(event Event)

// EventBus defines the interface for publishing and subscribing to events
type EventBus interface {
	// Publish sends an event to all subscribers
	Publish(event Event)

	// Subscribe registers a handler for events matching the criteria
	// Returns a subscription ID that can be used to unsubscribe
	Subscribe(appName string, eventTypes []EventType, handler EventHandler) string

	// Unsubscribe removes a subscription by ID
	Unsubscribe(subscriptionID string)

	// Close shuts down the event bus and cleans up resources
	Close()
}

// ToSSE formats the event as a Server-Sent Event message
func (e Event) ToSSE() string {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Sprintf("data: {\"error\": \"failed to marshal event: %v\"}\n\n", err)
	}
	return fmt.Sprintf("data: %s\n\n", data)
}

// NewEvent creates a new event with a generated ID and current timestamp
func NewEvent(eventType EventType, appName, source string, data map[string]interface{}) Event {
	return Event{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      eventType,
		AppName:   appName,
		Timestamp: time.Now(),
		Data:      data,
		Source:    source,
	}
}
