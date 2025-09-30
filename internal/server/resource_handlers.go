package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"innominatus/internal/database"
	"strconv"
	"strings"
)

// HandleResources handles resource CRUD operations
func (s *Server) HandleResources(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleListResources(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleResourceDetail handles individual resource operations
func (s *Server) HandleResourceDetail(w http.ResponseWriter, r *http.Request) {
	// Check if we have database and resource manager
	if s.db == nil || s.resourceManager == nil {
		http.Error(w, "Resource management requires database connection", http.StatusServiceUnavailable)
		return
	}

	// Extract resource ID from path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid resource path", http.StatusBadRequest)
		return
	}

	resourceIDStr := pathParts[2]
	resourceID, err := strconv.ParseInt(resourceIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid resource ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		s.handleGetResource(w, r, resourceID)
	case "PUT":
		s.handleUpdateResource(w, r, resourceID)
	case "DELETE":
		s.handleDeleteResource(w, r, resourceID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleResourceTransition handles resource state transitions
func (s *Server) HandleResourceTransition(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if we have database and resource manager
	if s.db == nil || s.resourceManager == nil {
		http.Error(w, "Resource management requires database connection", http.StatusServiceUnavailable)
		return
	}

	// Extract resource ID from path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid resource transition path", http.StatusBadRequest)
		return
	}

	resourceIDStr := pathParts[2]
	resourceID, err := strconv.ParseInt(resourceIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid resource ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req struct {
		NewState    string                 `json:"new_state"`
		Reason      string                 `json:"reason"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Get user from context
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	// Convert string state to ResourceLifecycleState
	newState := database.ResourceLifecycleState(req.NewState)

	// Perform state transition
	err = s.resourceManager.TransitionResourceState(resourceID, newState, req.Reason, user.Username, req.Metadata)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to transition resource state: %v", err), http.StatusBadRequest)
		return
	}

	// Return updated resource
	resource, err := s.resourceManager.GetResource(resourceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get updated resource: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resource); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// HandleResourceHealth handles resource health operations
func (s *Server) HandleResourceHealth(w http.ResponseWriter, r *http.Request) {
	// Check if we have database and resource manager
	if s.db == nil || s.resourceManager == nil {
		http.Error(w, "Resource management requires database connection", http.StatusServiceUnavailable)
		return
	}

	// Extract resource ID from path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid resource health path", http.StatusBadRequest)
		return
	}

	resourceIDStr := pathParts[2]
	resourceID, err := strconv.ParseInt(resourceIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid resource ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		s.handleGetResourceHealth(w, r, resourceID)
	case "POST":
		s.handleCheckResourceHealth(w, r, resourceID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListResources lists all resources, optionally filtered by application
func (s *Server) handleListResources(w http.ResponseWriter, r *http.Request) {
	appName := r.URL.Query().Get("app")

	if appName != "" {
		// List resources for specific application
		var resources interface{}
		var err error

		if s.resourceManager != nil && s.db != nil {
			// Use database storage
			resources, err = s.resourceManager.GetResourcesByApplication(appName)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get resources: %v", err), http.StatusInternalServerError)
				return
			}
		} else {
			// Use local storage
			localResources, err := s.storage.GetResourcesByApplication(appName)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get resources: %v", err), http.StatusInternalServerError)
				return
			}
			resources = localResources
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"application": appName,
			"resources":   resources,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
		}
	} else {
		// Return all deployed applications and their resources
		specs := s.storage.ListStoredSpecs()
		allResources := make(map[string]interface{})

		if s.resourceManager != nil && s.db != nil {
			// Use database storage
			for appName := range specs {
				resources, err := s.resourceManager.GetResourcesByApplication(appName)
				if err != nil {
					continue // Skip apps with errors
				}
				if len(resources) > 0 {
					allResources[appName] = resources
				}
			}
		} else {
			// Use local storage
			for appName := range specs {
				resources, err := s.storage.GetResourcesByApplication(appName)
				if err != nil {
					continue // Skip apps with errors
				}
				if len(resources) > 0 {
					allResources[appName] = resources
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(allResources); err != nil {
			fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
		}
	}
}

// handleGetResource gets a specific resource by ID
func (s *Server) handleGetResource(w http.ResponseWriter, r *http.Request, resourceID int64) {
	resource, err := s.resourceManager.GetResource(resourceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Resource not found: %v", err), http.StatusNotFound)
		return
	}

	// Get state transitions for the resource
	transitions, err := s.resourceManager.GetResourceStateTransitions(resourceID, 10)
	if err != nil {
		// Don't fail the request, just log and continue
		fmt.Printf("Warning: Failed to get state transitions for resource %d: %v\n", resourceID, err)
	}

	// Add transitions to resource
	resource.StateTransitions = transitions

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resource); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// handleUpdateResource updates resource configuration or metadata
func (s *Server) handleUpdateResource(w http.ResponseWriter, r *http.Request, resourceID int64) {
	// Parse request body for updates
	var req struct {
		HealthStatus     *string                `json:"health_status,omitempty"`
		ErrorMessage     *string                `json:"error_message,omitempty"`
		ProviderID       *string                `json:"provider_id,omitempty"`
		ProviderMetadata map[string]interface{} `json:"provider_metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Update health status if provided
	if req.HealthStatus != nil {
		err := s.resourceManager.UpdateResourceHealth(resourceID, *req.HealthStatus, req.ErrorMessage)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to update resource health: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Get updated resource
	resource, err := s.resourceManager.GetResource(resourceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get updated resource: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resource); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// handleDeleteResource deletes a resource
func (s *Server) handleDeleteResource(w http.ResponseWriter, r *http.Request, resourceID int64) {
	// Get user from context
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	err := s.resourceManager.DeleteResource(resourceID, user.Username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete resource: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetResourceHealth gets resource health status
func (s *Server) handleGetResourceHealth(w http.ResponseWriter, r *http.Request, resourceID int64) {
	resource, err := s.resourceManager.GetResource(resourceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Resource not found: %v", err), http.StatusNotFound)
		return
	}

	healthInfo := map[string]interface{}{
		"resource_id":       resource.ID,
		"health_status":     resource.HealthStatus,
		"last_health_check": resource.LastHealthCheck,
		"error_message":     resource.ErrorMessage,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(healthInfo); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// handleCheckResourceHealth performs a health check on a resource
func (s *Server) handleCheckResourceHealth(w http.ResponseWriter, r *http.Request, resourceID int64) {
	err := s.resourceManager.CheckResourceHealth(resourceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check resource health: %v", err), http.StatusInternalServerError)
		return
	}

	// Return updated health status
	s.handleGetResourceHealth(w, r, resourceID)
}