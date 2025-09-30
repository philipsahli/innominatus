package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPHelper_GET(t *testing.T) {
	t.Run("successful GET request", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/api/test", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "success"})
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "")

		var result map[string]string
		err := helper.GET("/api/test", &result)

		require.NoError(t, err)
		assert.Equal(t, "success", result["message"])
	})

	t.Run("GET request with authentication", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"authenticated": "true"})
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "test-token")

		var result map[string]string
		err := helper.GET("/api/test", &result)

		require.NoError(t, err)
		assert.Equal(t, "true", result["authenticated"])
	})

	t.Run("GET request with 404 error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("resource not found"))
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "")

		var result map[string]string
		err := helper.GET("/api/test", &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found (404)")
	})

	t.Run("GET request with server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal server error"))
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "")

		var result map[string]string
		err := helper.GET("/api/test", &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "server error (500)")
	})

	t.Run("GET request with invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "")

		var result map[string]string
		err := helper.GET("/api/test", &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse response")
	})
}

func TestHTTPHelper_POST(t *testing.T) {
	t.Run("successful POST request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var reqBody map[string]string
			json.NewDecoder(r.Body).Decode(&reqBody)
			assert.Equal(t, "test-value", reqBody["test-key"])

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"created": "true"})
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "")

		reqBody := map[string]string{"test-key": "test-value"}
		var respBody map[string]string
		err := helper.POST("/api/test", reqBody, &respBody)

		require.NoError(t, err)
		assert.Equal(t, "true", respBody["created"])
	})

	t.Run("POST request with nil body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "")

		var respBody map[string]string
		err := helper.POST("/api/test", nil, &respBody)

		require.NoError(t, err)
		assert.Equal(t, "ok", respBody["status"])
	})
}

func TestHTTPHelper_DELETE(t *testing.T) {
	t.Run("successful DELETE request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "DELETE", r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "")

		err := helper.DELETE("/api/test/123")

		require.NoError(t, err)
	})

	t.Run("DELETE request with 404 error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("resource not found"))
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "")

		err := helper.DELETE("/api/test/123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found (404)")
	})
}

func TestHTTPHelper_doYAMLRequest(t *testing.T) {
	t.Run("successful YAML request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/x-yaml", r.Header.Get("Content-Type"))

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"processed": "true"})
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "")

		yamlData := []byte("apiVersion: v1\nkind: Test")
		var result map[string]string
		err := helper.doYAMLRequest("POST", "/api/test", yamlData, &result)

		require.NoError(t, err)
		assert.Equal(t, "true", result["processed"])
	})
}

func TestHTTPHelper_doRequestWithStatus(t *testing.T) {
	t.Run("successful request with expected status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"created": "true"})
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "")

		var result map[string]string
		err := helper.doRequestWithStatus("POST", "/api/test", nil, "", http.StatusCreated, &result)

		require.NoError(t, err)
		assert.Equal(t, "true", result["created"])
	})

	t.Run("request with unexpected status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "")

		var result map[string]string
		err := helper.doRequestWithStatus("POST", "/api/test", nil, "", http.StatusCreated, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status 200 (expected 201)")
	})
}

func TestHTTPHelper_PUT(t *testing.T) {
	t.Run("successful PUT request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)

			var reqBody map[string]string
			json.NewDecoder(r.Body).Decode(&reqBody)
			assert.Equal(t, "updated-value", reqBody["key"])

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"updated": "true"})
		}))
		defer server.Close()

		helper := newHTTPHelper(server.URL, &http.Client{Timeout: 5 * time.Second}, "")

		reqBody := map[string]string{"key": "updated-value"}
		var respBody map[string]string
		err := helper.PUT("/api/test/123", reqBody, &respBody)

		require.NoError(t, err)
		assert.Equal(t, "true", respBody["updated"])
	})
}

func TestHTTPHelper_setAuthHeader(t *testing.T) {
	t.Run("sets auth header when token present", func(t *testing.T) {
		helper := newHTTPHelper("http://test.com", &http.Client{}, "test-token")

		req, _ := http.NewRequest("GET", "http://test.com/api/test", nil)
		helper.setAuthHeader(req)

		assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
	})

	t.Run("does not set auth header when token empty", func(t *testing.T) {
		helper := newHTTPHelper("http://test.com", &http.Client{}, "")

		req, _ := http.NewRequest("GET", "http://test.com/api/test", nil)
		helper.setAuthHeader(req)

		assert.Empty(t, req.Header.Get("Authorization"))
	})
}

func TestHTTPHelper_NetworkError(t *testing.T) {
	t.Run("handles network connection error", func(t *testing.T) {
		// Use invalid URL that will fail to connect
		helper := newHTTPHelper("http://invalid-host-that-does-not-exist-12345.com", &http.Client{Timeout: 1 * time.Second}, "")

		var result map[string]string
		err := helper.GET("/api/test", &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "request failed")
	})
}
