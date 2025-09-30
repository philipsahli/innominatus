package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"innominatus/internal/users"
)

// Helper function to create an authenticated request with user context
func createAuthenticatedRequest(method, url, body string) *http.Request {
	req := httptest.NewRequest(method, url, strings.NewReader(body))

	// Create a test user
	testUser := &users.User{
		Username: "testuser",
		Team:     "engineering",
		Role:     "developer",
	}

	// Add user to request context
	ctx := context.WithValue(req.Context(), contextKeyUser, testUser)
	req = req.WithContext(ctx)

	return req
}

func TestNewServer(t *testing.T) {
	server := NewServer()

	assert.NotNil(t, server)
	assert.NotNil(t, server.storage)
	assert.NotNil(t, server.workflowAnalyzer)
	assert.NotNil(t, server.teamManager)
	assert.NotNil(t, server.sessionManager)
	assert.NotNil(t, server.loginAttempts)
	assert.NotNil(t, server.memoryWorkflows)
	assert.GreaterOrEqual(t, server.workflowCounter, int64(0))
}

func TestMemoryWorkflowExecution(t *testing.T) {
	now := time.Now()
	errorMsg := "test error"

	execution := &MemoryWorkflowExecution{
		ID:           1,
		AppName:      "test-app",
		WorkflowName: "deploy",
		Status:       "running",
		StartedAt:    now,
		CompletedAt:  nil,
		ErrorMessage: &errorMsg,
		StepCount:    2,
		Steps: []*MemoryWorkflowStep{
			{
				ID:           1,
				StepNumber:   1,
				Name:         "step1",
				Type:         "terraform",
				Status:       "completed",
				StartedAt:    now,
				CompletedAt:  &now,
				ErrorMessage: nil,
			},
			{
				ID:           2,
				StepNumber:   2,
				Name:         "step2",
				Type:         "kubernetes",
				Status:       "failed",
				StartedAt:    now,
				CompletedAt:  &now,
				ErrorMessage: &errorMsg,
			},
		},
	}

	assert.Equal(t, int64(1), execution.ID)
	assert.Equal(t, "test-app", execution.AppName)
	assert.Equal(t, "deploy", execution.WorkflowName)
	assert.Equal(t, "running", execution.Status)
	assert.Equal(t, 2, execution.StepCount)
	assert.Len(t, execution.Steps, 2)
	assert.Equal(t, &errorMsg, execution.ErrorMessage)
}

func TestHandleWorkflowAnalysis(t *testing.T) {
	server := NewServer()

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "valid POST with YAML",
			method:         "POST",
			body:           validScoreSpecYAML(),
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "invalid method GET",
			method:         "GET",
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError:  true,
		},
		{
			name:           "empty body",
			method:         "POST",
			body:           "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:           "invalid YAML",
			method:         "POST",
			body:           "invalid: yaml: content:",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/workflow-analysis", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/yaml")
			w := httptest.NewRecorder()

			server.HandleWorkflowAnalysis(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				// For error cases, just check that we got an error status
				assert.NotEqual(t, http.StatusOK, w.Code)
			} else {
				// Should return valid workflow analysis
				var analysis map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &analysis)
				require.NoError(t, err)
				assert.Contains(t, analysis, "spec")
				assert.Contains(t, analysis, "dependencies")
				assert.Contains(t, analysis, "executionPlan")
			}
		})
	}
}

func TestHandleWorkflowAnalysisPreview(t *testing.T) {
	server := NewServer()

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "valid POST with YAML",
			method:         "POST",
			body:           validScoreSpecYAML(),
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "invalid method GET",
			method:         "GET",
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError:  true,
		},
		{
			name:           "empty body",
			method:         "POST",
			body:           "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/workflow-analysis/preview", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/yaml")
			w := httptest.NewRecorder()

			server.HandleWorkflowAnalysisPreview(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				// For error cases, just check that we got an error status
				assert.NotEqual(t, http.StatusOK, w.Code)
			} else {
				// Should return workflow preview
				var preview map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &preview)
				require.NoError(t, err)
				assert.Contains(t, preview, "executionPlan")
				assert.Contains(t, preview, "summary")
			}
		})
	}
}

func TestHandleSpecs(t *testing.T) {
	server := NewServer()

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "valid POST",
			method:         "POST",
			body:           simpleScoreSpecYAML(),
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
		{
			name:           "valid GET",
			method:         "GET",
			body:           "",
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "invalid method PUT",
			method:         "PUT",
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError:  true,
		},
		{
			name:           "invalid YAML on POST",
			method:         "POST",
			body:           "invalid yaml",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createAuthenticatedRequest(tt.method, "/api/specs", tt.body)
			if tt.method == "POST" {
				req.Header.Set("Content-Type", "application/yaml")
			}
			w := httptest.NewRecorder()

			server.HandleSpecs(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				// For error cases, just check that we got an error status
				assert.NotEqual(t, http.StatusOK, w.Code)
			}
		})
	}
}

func TestHandleHealth(t *testing.T) {
	server := NewServer()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.HandleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
	assert.Contains(t, response, "timestamp")
}

func TestHandleLogin(t *testing.T) {
	server := NewServer()

	tests := []struct {
		name           string
		method         string
		body           map[string]string
		expectedStatus int
		expectedError  bool
	}{
		{
			name:   "valid login",
			method: "POST",
			body: map[string]string{
				"username": "admin",
				"password": "admin123",
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:   "invalid credentials",
			method: "POST",
			body: map[string]string{
				"username": "admin",
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  true,
		},
		{
			name:           "invalid method GET",
			method:         "GET",
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError:  true,
		},
		{
			name:   "missing username",
			method: "POST",
			body: map[string]string{
				"password": "admin",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != nil {
				bodyBytes, _ := json.Marshal(tt.body)
				body = bytes.NewReader(bodyBytes)
			} else {
				body = strings.NewReader("")
			}

			req := httptest.NewRequest(tt.method, "/api/login", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.HandleAPILogin(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				// For error cases, just check that we got an error status
				assert.NotEqual(t, http.StatusOK, w.Code)
			} else {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "token")
			}
		})
	}
}

func TestHandleLogout(t *testing.T) {
	server := NewServer()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "valid logout (redirects)",
			method:         "POST",
			expectedStatus: http.StatusSeeOther, // 303 redirect
		},
		{
			name:           "GET logout (also redirects)",
			method:         "GET",
			expectedStatus: http.StatusSeeOther, // 303 redirect
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/logout", nil)
			w := httptest.NewRecorder()

			server.HandleLogout(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestLoginRateLimit(t *testing.T) {
	server := NewServer()

	// Simulate multiple failed login attempts
	for i := 0; i < 6; i++ {
		body := map[string]string{
			"username": "admin",
			"password": "wrongpassword",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/login", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "127.0.0.1:12345" // Same IP for rate limiting
		w := httptest.NewRecorder()

		server.HandleAPILogin(w, req)

		if i < 5 {
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		} else {
			// 6th attempt should be rate limited
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		}
	}
}

func TestMemoryWorkflowTracking(t *testing.T) {
	server := NewServer()

	// Add a workflow execution
	execution := &MemoryWorkflowExecution{
		ID:           1,
		AppName:      "test-app",
		WorkflowName: "deploy",
		Status:       "running",
		StartedAt:    time.Now(),
		StepCount:    1,
		Steps:        []*MemoryWorkflowStep{},
	}

	server.workflowMutex.Lock()
	server.memoryWorkflows[1] = execution
	server.workflowCounter = 1
	server.workflowMutex.Unlock()

	// Test retrieval
	server.workflowMutex.RLock()
	retrieved := server.memoryWorkflows[1]
	server.workflowMutex.RUnlock()

	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-app", retrieved.AppName)
	assert.Equal(t, "deploy", retrieved.WorkflowName)
	assert.Equal(t, "running", retrieved.Status)
}

func TestServerLifecycle(t *testing.T) {
	server := NewServer()

	// Test that server is properly initialized
	assert.NotNil(t, server.storage)
	assert.NotNil(t, server.workflowAnalyzer)
	assert.NotNil(t, server.teamManager)
	assert.NotNil(t, server.sessionManager)

	// Test workflow counter
	assert.GreaterOrEqual(t, server.workflowCounter, int64(0))

	// Test memory workflows map
	assert.NotNil(t, server.memoryWorkflows)
	assert.GreaterOrEqual(t, len(server.memoryWorkflows), 0)
}

func TestHTTPMethodValidation(t *testing.T) {
	server := NewServer()

	endpoints := []struct {
		path          string
		handler       http.HandlerFunc
		validMethods  []string
		invalidMethod string
	}{
		{
			path:          "/api/workflow-analysis",
			handler:       server.HandleWorkflowAnalysis,
			validMethods:  []string{"POST"},
			invalidMethod: "GET",
		},
		{
			path:          "/api/login",
			handler:       server.HandleAPILogin,
			validMethods:  []string{"POST"},
			invalidMethod: "GET",
		},
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("Test %s method validation", endpoint.path), func(t *testing.T) {
			req := httptest.NewRequest(endpoint.invalidMethod, endpoint.path, nil)
			w := httptest.NewRecorder()

			endpoint.handler(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}

// Helper function to create simple Score spec YAML for testing (no workflows)
func simpleScoreSpecYAML() string {
	return `apiVersion: score.dev/v1b1
metadata:
  name: test-app
containers:
  web:
    image: nginx:latest`
}

// Helper function to create valid Score spec YAML for testing
func validScoreSpecYAML() string {
	return `apiVersion: score.dev/v1b1
metadata:
  name: test-app
containers:
  web:
    image: nginx:latest
    variables:
      DB_HOST: ${resources.db.host}
resources:
  db:
    type: postgres
    params:
      version: "13"
workflows:
  deploy:
    steps:
      - name: setup-infra
        type: terraform
        path: ./terraform
      - name: deploy-app
        type: kubernetes
        namespace: test-app`
}