package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// APIClient handles HTTP requests to the innominatus API
type APIClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL, token string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		token:   token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get performs a GET request
func (c *APIClient) Get(ctx context.Context, endpoint string) (string, error) {
	return c.request(ctx, "GET", endpoint, nil)
}

// Post performs a POST request with JSON body
func (c *APIClient) Post(ctx context.Context, endpoint string, body interface{}) (string, error) {
	var bodyBytes []byte
	var err error

	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	return c.request(ctx, "POST", endpoint, bodyBytes)
}

// PostYAML performs a POST request with YAML body
func (c *APIClient) PostYAML(ctx context.Context, endpoint string, yamlBody string) (string, error) {
	return c.requestWithContentType(ctx, "POST", endpoint, []byte(yamlBody), "application/yaml")
}

// request performs an HTTP request
func (c *APIClient) request(ctx context.Context, method, endpoint string, body []byte) (string, error) {
	return c.requestWithContentType(ctx, method, endpoint, body, "application/json")
}

// requestWithContentType performs an HTTP request with custom Content-Type
func (c *APIClient) requestWithContentType(ctx context.Context, method, endpoint string, body []byte, contentType string) (string, error) {
	url := c.baseURL + endpoint

	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}

	if err != nil {
		log.Error().Err(err).Str("method", method).Str("url", url).Msg("Failed to create HTTP request")
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", contentType)

	// Execute request
	log.Debug().Str("method", method).Str("url", url).Msg("Executing API request")
	resp, err := c.client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("method", method).Str("url", url).Msg("HTTP request failed")
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("method", method).Str("url", url).Msg("Failed to read response body")
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warn().
			Int("status", resp.StatusCode).
			Str("method", method).
			Str("url", url).
			Str("response", string(respBody)).
			Msg("HTTP request returned error status")
		return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	log.Debug().
		Int("status", resp.StatusCode).
		Str("method", method).
		Str("url", url).
		Msg("API request successful")

	return string(respBody), nil
}
