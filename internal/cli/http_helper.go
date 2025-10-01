package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTPHelper provides common HTTP request functionality for the CLI client
type HTTPHelper struct {
	baseURL string
	client  *http.Client
	token   string
}

// newHTTPHelper creates a new HTTP helper instance
func newHTTPHelper(baseURL string, client *http.Client, token string) *HTTPHelper {
	return &HTTPHelper{
		baseURL: baseURL,
		client:  client,
		token:   token,
	}
}

// setAuthHeader adds the Authorization header if token is available
func (h *HTTPHelper) setAuthHeader(req *http.Request) {
	if h.token != "" {
		req.Header.Set("Authorization", "Bearer "+h.token)
	}
}

// doRequest performs a generic HTTP request and unmarshals the response into result
// This eliminates the repetitive request/response handling code
func (h *HTTPHelper) doRequest(method, path string, body io.Reader, contentType string, result interface{}) error {
	url := h.baseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	h.setAuthHeader(req)

	// Execute request
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode >= 400 {
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("not found (404): %s", string(respBody))
		}
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(respBody))
	}

	// Unmarshal response if result is provided
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// doJSONRequest performs a JSON request with automatic marshaling/unmarshaling
func (h *HTTPHelper) doJSONRequest(method, path string, reqBody, respBody interface{}) error {
	var body io.Reader

	// Marshal request body if provided
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	return h.doRequest(method, path, body, "application/json", respBody)
}

// doYAMLRequest performs a request with YAML content
func (h *HTTPHelper) doYAMLRequest(method, path string, yamlBody []byte, result interface{}) error {
	body := bytes.NewReader(yamlBody)
	return h.doRequest(method, path, body, "application/x-yaml", result)
}

// doRequestWithStatus performs a request and validates against expected status codes
func (h *HTTPHelper) doRequestWithStatus(method, path string, body io.Reader, contentType string, expectedStatus int, result interface{}) error {
	url := h.baseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	h.setAuthHeader(req)

	// Execute request
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for expected status code
	if resp.StatusCode != expectedStatus {
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("not found (404): %s", string(respBody))
		}
		return fmt.Errorf("unexpected status %d (expected %d): %s", resp.StatusCode, expectedStatus, string(respBody))
	}

	// Unmarshal response if result is provided
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// GET performs a GET request
func (h *HTTPHelper) GET(path string, result interface{}) error {
	return h.doRequest("GET", path, nil, "", result)
}

// POST performs a POST request with JSON body
func (h *HTTPHelper) POST(path string, reqBody, respBody interface{}) error {
	return h.doJSONRequest("POST", path, reqBody, respBody)
}

// PUT performs a PUT request with JSON body
func (h *HTTPHelper) PUT(path string, reqBody, respBody interface{}) error {
	return h.doJSONRequest("PUT", path, reqBody, respBody)
}

// DELETE performs a DELETE request
func (h *HTTPHelper) DELETE(path string) error {
	return h.doRequest("DELETE", path, nil, "", nil)
}

// POSTWithStatus performs a POST request and validates status code
func (h *HTTPHelper) POSTWithStatus(path string, reqBody interface{}, expectedStatus int, respBody interface{}) error {
	var body io.Reader

	// Marshal request body if provided
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	return h.doRequestWithStatus("POST", path, body, "application/json", expectedStatus, respBody)
}
