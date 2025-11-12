package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Event represents a server-sent event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	AppName   string                 `json:"app_name"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Source    string                 `json:"source"`
}

// SSEClient handles Server-Sent Events streaming
type SSEClient struct {
	serverURL string
	apiKey    string
	client    *http.Client
}

// NewSSEClient creates a new SSE client
func NewSSEClient(serverURL, apiKey string) *SSEClient {
	return &SSEClient{
		serverURL: serverURL,
		apiKey:    apiKey,
		client: &http.Client{
			Timeout: 0, // No timeout for streaming connections
		},
	}
}

// StreamEvents connects to the SSE endpoint and streams events
func (c *SSEClient) StreamEvents(ctx context.Context, appName string, eventHandler func(Event) error) error {
	// Build URL with query parameters
	url := fmt.Sprintf("%s/api/events/stream", c.serverURL)
	if appName != "" {
		url = fmt.Sprintf("%s?app=%s", url, appName)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	// Set SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Make request
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to SSE endpoint: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SSE connection failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	// Read and process SSE stream
	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("error reading stream: %w", err)
			}

			line = strings.TrimSpace(line)

			// Skip empty lines and comments
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}

			// Parse SSE data line
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")

				// Parse JSON event
				var event Event
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					// Skip malformed events
					continue
				}

				// Call handler
				if err := eventHandler(event); err != nil {
					return fmt.Errorf("event handler error: %w", err)
				}
			}
		}
	}
}

// WaitForCompletion waits for a deployment to complete by monitoring events
func (c *SSEClient) WaitForCompletion(ctx context.Context, appName string, timeout time.Duration) error {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Track deployment state
	deploymentComplete := false
	deploymentFailed := false
	var lastError string

	// Event handler
	eventHandler := func(event Event) error {
		switch event.Type {
		case "deployment.completed":
			deploymentComplete = true
			return fmt.Errorf("deployment completed") // Signal to stop streaming

		case "deployment.failed":
			deploymentFailed = true
			if errMsg, ok := event.Data["error"].(string); ok {
				lastError = errMsg
			}
			return fmt.Errorf("deployment failed") // Signal to stop streaming

		case "resource.failed":
			if errMsg, ok := event.Data["error"].(string); ok {
				lastError = errMsg
			}

		case "workflow.failed":
			if errMsg, ok := event.Data["error"].(string); ok {
				lastError = errMsg
			}
		}

		return nil
	}

	// Stream events
	err := c.StreamEvents(timeoutCtx, appName, eventHandler)

	// Check final state
	if deploymentComplete {
		return nil
	}

	if deploymentFailed {
		if lastError != "" {
			return fmt.Errorf("deployment failed: %s", lastError)
		}
		return fmt.Errorf("deployment failed")
	}

	if err != nil && err != context.DeadlineExceeded {
		return err
	}

	if err == context.DeadlineExceeded {
		return fmt.Errorf("deployment timeout after %v", timeout)
	}

	return fmt.Errorf("deployment did not complete")
}
