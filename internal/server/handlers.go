package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"net/http"
	"os"
	"os/exec"
	"innominatus/internal/admin"
	"innominatus/internal/auth"
	"innominatus/internal/database"
	"innominatus/internal/demo"
	"innominatus/internal/graph"
	"innominatus/internal/health"
	"innominatus/internal/metrics"
	"innominatus/internal/resources"
	"innominatus/internal/teams"
	"innominatus/internal/types"
	"innominatus/internal/workflow"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// LogBuffer captures command output for workflow step logging
type LogBuffer struct {
	buffer   strings.Builder
	stepID   *int64
	repo     *database.WorkflowRepository
	mu       sync.Mutex
}

// NewLogBuffer creates a new log buffer for a workflow step
func NewLogBuffer(stepID *int64, repo *database.WorkflowRepository) *LogBuffer {
	return &LogBuffer{
		stepID: stepID,
		repo:   repo,
	}
}

// Write implements io.Writer interface for capturing command output
func (lb *LogBuffer) Write(p []byte) (n int, err error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Add timestamp to each line
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	lines := strings.Split(string(p), "\n")

	for i, line := range lines {
		if line != "" || i < len(lines)-1 {
			if line != "" {
				formattedLine := fmt.Sprintf("[%s] %s\n", timestamp, line)
				lb.buffer.WriteString(formattedLine)
			} else if i < len(lines)-1 {
				lb.buffer.WriteString("\n")
			}
		}
	}

	// Store logs in database if step ID is available
	if lb.stepID != nil && *lb.stepID > 0 && lb.repo != nil {
		logContent := lb.buffer.String()
		if logContent != "" {
			if err := lb.repo.AddWorkflowStepLogs(*lb.stepID, logContent); err != nil {
				// Log error but don't fail the write operation
				fmt.Fprintf(os.Stderr, "failed to store workflow logs: %v\n", err)
			}
			lb.buffer.Reset() // Clear buffer after storing
		}
	}

	return len(p), nil
}

// GetLogs returns the accumulated logs
func (lb *LogBuffer) GetLogs() string {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.buffer.String()
}

// StepExecutionContext contains context for executing a workflow step
type StepExecutionContext struct {
	Step       types.Step
	AppName    string
	EnvType    string
	StepID     *int64
	LogBuffer  *LogBuffer
	WorkflowRepo *database.WorkflowRepository
}

type Server struct {
	storage          *Storage
	db               *database.Database
	workflowRepo     *database.WorkflowRepository
	workflowExecutor *workflow.WorkflowExecutor
	workflowAnalyzer *workflow.WorkflowAnalyzer
	resourceManager  *resources.Manager
	teamManager      *teams.TeamManager
	sessionManager   *auth.SessionManager
	healthChecker    *health.HealthChecker
	loginAttempts    map[string][]time.Time
	loginMutex       sync.Mutex
	// In-memory workflow tracking (when database is not available)
	memoryWorkflows  map[int64]*MemoryWorkflowExecution
	workflowCounter  int64
	workflowMutex    sync.RWMutex
	// Workflow scheduler for periodic execution
	workflowTicker   *time.Ticker
	stopScheduler    chan struct{}
}

// MemoryWorkflowExecution represents a workflow execution stored in memory
type MemoryWorkflowExecution struct {
	ID           int64                    `json:"id"`
	AppName      string                   `json:"app_name"`
	WorkflowName string                   `json:"workflow_name"`
	Status       string                   `json:"status"`
	StartedAt    time.Time                `json:"started_at"`
	CompletedAt  *time.Time               `json:"completed_at,omitempty"`
	ErrorMessage *string                  `json:"error_message,omitempty"`
	StepCount    int                      `json:"step_count"`
	Steps        []*MemoryWorkflowStep    `json:"steps"`
}

// MemoryWorkflowStep represents a workflow step stored in memory
type MemoryWorkflowStep struct {
	ID           int64      `json:"id"`
	StepNumber   int        `json:"step_number"`
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
}

func NewServer() *Server {
	healthChecker := health.NewHealthChecker()
	// Register basic health checks
	healthChecker.Register(health.NewAlwaysHealthyChecker("server"))

	server := &Server{
		storage:          NewStorage(),
		workflowAnalyzer: workflow.NewWorkflowAnalyzer(),
		teamManager:      teams.NewTeamManager(),
		sessionManager:   auth.NewSessionManager(),
		healthChecker:    healthChecker,
		loginAttempts:    make(map[string][]time.Time),
		memoryWorkflows:  make(map[int64]*MemoryWorkflowExecution),
		workflowCounter:  0,
	}

	// Load existing workflow executions from disk
	server.loadWorkflowsFromDisk()

	return server
}

func NewServerWithDB(db *database.Database) *Server {
	workflowRepo := database.NewWorkflowRepository(db)
	resourceRepo := database.NewResourceRepository(db)
	resourceManager := resources.NewManager(resourceRepo)
	workflowExecutor := workflow.NewWorkflowExecutorWithResourceManager(workflowRepo, resourceManager)

	healthChecker := health.NewHealthChecker()
	// Register health checks
	healthChecker.Register(health.NewAlwaysHealthyChecker("server"))
	healthChecker.Register(health.NewDatabaseChecker(db.DB(), 5*time.Second))

	server := &Server{
		storage:          NewStorage(),
		db:               db,
		workflowRepo:     workflowRepo,
		workflowExecutor: workflowExecutor,
		workflowAnalyzer: workflow.NewWorkflowAnalyzer(),
		resourceManager:  resourceManager,
		teamManager:      teams.NewTeamManager(),
		sessionManager:   auth.NewSessionManager(),
		healthChecker:    healthChecker,
		loginAttempts:    make(map[string][]time.Time),
		memoryWorkflows:  make(map[int64]*MemoryWorkflowExecution),
		workflowCounter:  0,
	}

	// Start the workflow scheduler only when database is available
	server.startWorkflowScheduler()

	return server
}

func (s *Server) HandleSpecs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleListSpecs(w, r)
	case "POST":
		s.handleDeploySpec(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleListSpecs(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by authentication middleware)
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var specs map[string]*types.ScoreSpec

	// Admin users can see all specs, regular users only see their team's specs
	if user.IsAdmin() {
		specs = s.storage.ListSpecs()
	} else {
		specs = s.storage.ListSpecsByTeam(user.Team)
	}

	response := make(map[string]interface{})
	for name, spec := range specs {
		response[name] = map[string]interface{}{
			"metadata":    spec.Metadata,
			"containers":  spec.Containers,
			"resources":   spec.Resources,
			"environment": spec.Environment,
			"graph":       graph.BuildGraph(spec),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) handleDeploySpec(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by authentication middleware)
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	var spec types.ScoreSpec
	err = yaml.Unmarshal(body, &spec)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing YAML: %v", err), http.StatusBadRequest)
		return
	}

	name := spec.Metadata.Name
	err = s.storage.AddSpec(name, &spec, user.Team, user.Username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error storing spec: %v", err), http.StatusInternalServerError)
		return
	}

	// Create resource instances if database is available
	if s.resourceManager != nil && s.db != nil {
		fmt.Printf("Creating resource instances for app '%s'...\n", name)
		err = s.resourceManager.CreateResourceFromSpec(name, &spec, user.Username)
		if err != nil {
			fmt.Printf("Warning: Failed to create resource instances: %v\n", err)
			// Don't fail the deployment - continue with workflows
		}

		// If environment type is kubernetes, create GitOps pipeline resources automatically
		if spec.Environment != nil && spec.Environment.Type == "kubernetes" {
			fmt.Printf("\n🚀 Creating GitOps pipeline for '%s'...\n", name)

			// Step 1: Create Gitea repository for application manifests
			fmt.Printf("\n📚 Step 1/3: Creating Gitea repository for '%s'...\n", name)
			giteaResource, err := s.resourceManager.CreateResourceInstance(
				name,
				fmt.Sprintf("%s-gitea", name), // unique resource name
				"gitea-repo",
				map[string]interface{}{
					"repo_name":   name,
					"description": fmt.Sprintf("GitOps repository for %s", name),
					"private":     false,
				},
			)

			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to create gitea-repo resource: %v\n", err)
			} else {
				fmt.Printf("✅ Created gitea-repo resource instance: %d\n", giteaResource.ID)

				err = s.resourceManager.ProvisionResource(
					giteaResource.ID,
					"gitea-provisioner",
					map[string]interface{}{
						"repo_name":   name,
						"description": fmt.Sprintf("GitOps repository for %s", name),
						"private":     false,
					},
					user.Username,
				)
				if err != nil {
					fmt.Printf("⚠️  Warning: Gitea repository provisioning failed: %v\n", err)
				}
			}

			// Step 2: Create Kubernetes deployment
			fmt.Printf("\n☸️  Step 2/3: Creating Kubernetes deployment for '%s'...\n", name)
			k8sResource, err := s.resourceManager.CreateResourceInstance(
				name,
				fmt.Sprintf("%s-k8s", name), // unique resource name
				"kubernetes",
				map[string]interface{}{
					"namespace":  name,
					"score_spec": &spec,
				},
			)

			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to create kubernetes resource: %v\n", err)
			} else {
				fmt.Printf("✅ Created kubernetes resource instance: %d\n", k8sResource.ID)

				err = s.resourceManager.ProvisionResource(
					k8sResource.ID,
					"kubernetes-provisioner",
					map[string]interface{}{
						"namespace":  name,
						"score_spec": &spec,
					},
					user.Username,
				)
				if err != nil {
					fmt.Printf("⚠️  Warning: Kubernetes provisioning failed: %v\n", err)
				}
			}

			// Step 3: Create ArgoCD Application
			fmt.Printf("\n🔄 Step 3/3: Creating ArgoCD Application for '%s'...\n", name)
			argoResource, err := s.resourceManager.CreateResourceInstance(
				name,
				fmt.Sprintf("%s-argocd", name), // unique resource name
				"argocd-app",
				map[string]interface{}{
					"repo_name":   name,
					"namespace":   name,
					"sync_policy": "manual", // Start with manual sync
				},
			)

			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to create argocd-app resource: %v\n", err)
			} else {
				fmt.Printf("✅ Created argocd-app resource instance: %d\n", argoResource.ID)

				err = s.resourceManager.ProvisionResource(
					argoResource.ID,
					"argocd-provisioner",
					map[string]interface{}{
						"repo_name":   name,
						"namespace":   name,
						"sync_policy": "manual",
					},
					user.Username,
				)
				if err != nil {
					fmt.Printf("⚠️  Warning: ArgoCD application provisioning failed: %v\n", err)
				}
			}

			fmt.Printf("\n✅ GitOps pipeline creation completed for '%s'\n\n", name)
		}
	}

	// Track overall deployment status based on workflow execution results
	var deploymentFailed bool
	var failedWorkflows []string
	var workflowErrors []string

	// Execute workflows if defined
	if spec.Workflows != nil {
		for workflowName, workflowDef := range spec.Workflows {
			fmt.Printf("Executing workflow '%s' for app '%s'...\n", workflowName, name)

			// Track workflow execution in memory (for non-database mode) or database
			var memoryExecution *MemoryWorkflowExecution
			if s.workflowExecutor == nil {
				// Use in-memory tracking when database is not available
				memoryExecution = s.CreateMemoryWorkflowExecution(name, workflowName, len(workflowDef.Steps))
				fmt.Printf("📝 Tracking workflow execution ID %d in memory\n", memoryExecution.ID)
			}

			// Use enhanced workflow execution with appName and envType
			err = s.runWorkflowWithTracking(workflowDef, name, "default", memoryExecution)

			if err != nil {
				// Update tracking with error
				if memoryExecution != nil {
					errorMsg := err.Error()
					s.UpdateMemoryWorkflowExecutionStatus(memoryExecution.ID, "failed", &errorMsg)
				}

				// Mark deployment as failed and collect error information
				deploymentFailed = true
				failedWorkflows = append(failedWorkflows, workflowName)
				workflowErrors = append(workflowErrors, err.Error())

				fmt.Printf("❌ Workflow '%s' execution failed for '%s': %v\n", workflowName, name, err)
			} else {
				// Update tracking with success
				if memoryExecution != nil {
					s.UpdateMemoryWorkflowExecutionStatus(memoryExecution.ID, "completed", nil)
				}
				fmt.Printf("✅ Workflow '%s' completed successfully for '%s'\n", workflowName, name)
			}
		}
	}

	// Prepare response based on workflow execution results
	var response map[string]interface{}
	var statusCode int

	if deploymentFailed {
		// Deployment failed due to workflow failures
		response = map[string]interface{}{
			"message":          fmt.Sprintf("Deployment of '%s' failed", name),
			"name":             name,
			"status":           "failed",
			"failed_workflows": failedWorkflows,
			"errors":           workflowErrors,
		}
		statusCode = http.StatusInternalServerError
	} else {
		// Deployment succeeded
		response = map[string]interface{}{
			"message": fmt.Sprintf("Successfully deployed '%s'", name),
			"name":    name,
			"status":  "success",
		}
		statusCode = http.StatusCreated
	}

	// Add environment creation message if applicable
	if spec.Environment != nil && spec.Environment.Type == "ephemeral" {
		response["environment"] = fmt.Sprintf("Creating ephemeral environment with TTL=%s", spec.Environment.TTL)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) HandleSpecDetail(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/api/specs/"):]
	
	switch r.Method {
	case "GET":
		s.handleGetSpec(w, r, name)
	case "DELETE":
		s.handleDeleteSpec(w, r, name)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetSpec(w http.ResponseWriter, r *http.Request, name string) {
	// Get user from context (set by authentication middleware)
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	storedSpec, exists := s.storage.GetStoredSpec(name)
	if !exists {
		http.Error(w, "Spec not found", http.StatusNotFound)
		return
	}

	// Check if user has access to this spec
	if !user.IsAdmin() && storedSpec.Team != user.Team {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	response := map[string]interface{}{
		"metadata":    storedSpec.Spec.Metadata,
		"containers":  storedSpec.Spec.Containers,
		"resources":   storedSpec.Spec.Resources,
		"environment": storedSpec.Spec.Environment,
		"graph":       graph.BuildGraph(storedSpec.Spec),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) handleDeleteSpec(w http.ResponseWriter, r *http.Request, name string) {
	// Get user from context (set by authentication middleware)
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	storedSpec, exists := s.storage.GetStoredSpec(name)
	if !exists {
		http.Error(w, "Spec not found", http.StatusNotFound)
		return
	}

	// Check if user has access to this spec
	if !user.IsAdmin() && storedSpec.Team != user.Team {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	err := s.storage.DeleteSpec(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": fmt.Sprintf("Successfully deleted '%s'", name),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) HandleEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	environments := s.storage.ListEnvironments()
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(environments); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// Legacy endpoint for compatibility
func (s *Server) HandleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path

	// Handle /api/graph/<app> pattern
	if len(path) > len("/api/graph/") && path[:len("/api/graph/")] == "/api/graph/" {
		appName := path[len("/api/graph/"):]
		s.handleAppGraph(w, r, appName)
		return
	}

	// Legacy /api/graph endpoint - return first spec for backward compatibility
	specs := s.storage.ListSpecs()

	if len(specs) == 1 {
		for _, spec := range specs {
			response := map[string]interface{}{
				"metadata":    spec.Metadata,
				"containers":  spec.Containers,
				"resources":   spec.Resources,
				"environment": spec.Environment,
				"graph":       graph.BuildGraph(spec),
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
			return
		}
	}

	// Return all specs if multiple exist
	s.handleListSpecs(w, r)
}

// handleAppGraph handles /api/graph/<app> requests with enhanced graph data
func (s *Server) handleAppGraph(w http.ResponseWriter, r *http.Request, appName string) {
	// Get the spec for the application
	spec, exists := s.storage.GetSpec(appName)
	if !exists {
		http.Error(w, fmt.Sprintf("Application '%s' not found", appName), http.StatusNotFound)
		return
	}

	// Build enhanced resource graph
	resourceGraph := graph.BuildResourceGraph(appName, spec)

	// Add workflow data to the graph
	s.addWorkflowDataToGraph(resourceGraph, appName)

	// Add mock resource status (for demonstration)
	s.addMockResourceStatus(resourceGraph)

	// Return the enhanced graph
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resourceGraph); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// addWorkflowDataToGraph adds workflow execution data to the resource graph
func (s *Server) addWorkflowDataToGraph(resourceGraph *graph.Graph, appName string) {
	s.workflowMutex.RLock()
	defer s.workflowMutex.RUnlock()

	// Find workflows for this app
	for _, workflow := range s.memoryWorkflows {
		if workflow.AppName == appName {
			// Extract step names
			stepNames := make([]string, len(workflow.Steps))
			for i, step := range workflow.Steps {
				stepNames[i] = step.Name
			}

			// Add workflow node to graph
			resourceGraph.AddWorkflowNodes(workflow.WorkflowName, workflow.Status, stepNames)
		}
	}
}

// addMockResourceStatus adds mock infrastructure resource status for demonstration
func (s *Server) addMockResourceStatus(resourceGraph *graph.Graph) {
	// Detect Postgres resources and add mock status
	postgresNodes := resourceGraph.DetectPostgresResources()
	for _, node := range postgresNodes {
		resourceGraph.UpdateResourceStatus(node.Name, graph.NodeStatusCompleted, map[string]interface{}{
			"connection_string": "postgresql://localhost:5432/" + node.Name,
			"status":           "running",
			"host":             "postgres.demo.local",
			"port":             5432,
			"database":         node.Name,
		})
	}

	// Add mock status for other resource types
	for _, node := range resourceGraph.Nodes {
		if node.Type == graph.NodeTypeResource && node.Status == graph.NodeStatusUnknown {
			resourceType, _ := node.Metadata["resource_type"].(string)
			switch resourceType {
			case "redis":
				resourceGraph.UpdateResourceStatus(node.Name, graph.NodeStatusCompleted, map[string]interface{}{
					"endpoint": "redis.demo.local:6379",
					"status":   "running",
				})
			case "volume":
				resourceGraph.UpdateResourceStatus(node.Name, graph.NodeStatusCompleted, map[string]interface{}{
					"mount_path": "/data/" + node.Name,
					"size":       "10Gi",
					"status":     "bound",
				})
			case "route":
				resourceGraph.UpdateResourceStatus(node.Name, graph.NodeStatusCompleted, map[string]interface{}{
					"url":    "https://" + node.Name + ".demo.local",
					"status": "active",
				})
			default:
				resourceGraph.UpdateResourceStatus(node.Name, graph.NodeStatusCompleted, map[string]interface{}{
					"status": "provisioned",
				})
			}
		}
	}
}

// HandleWorkflows handles workflow-related API requests
func (s *Server) HandleWorkflows(w http.ResponseWriter, r *http.Request) {
	if s.workflowExecutor == nil {
		// Use in-memory workflow tracking when database is not available
		s.handleListMemoryWorkflows(w, r)
		return
	}

	switch r.Method {
	case "GET":
		s.handleListWorkflows(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleWorkflowDetail handles individual workflow execution requests
func (s *Server) HandleWorkflowDetail(w http.ResponseWriter, r *http.Request) {
	if s.workflowExecutor == nil {
		// Use in-memory workflow tracking when database is not available
		s.handleGetMemoryWorkflow(w, r)
		return
	}

	// Extract workflow ID from URL path
	path := r.URL.Path[len("/api/workflows/"):]
	if path == "" {
		http.Error(w, "Workflow ID required", http.StatusBadRequest)
		return
	}

	var workflowID int64
	_, err := fmt.Sscanf(path, "%d", &workflowID)
	if err != nil {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		s.handleGetWorkflow(w, r, workflowID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	appName := r.URL.Query().Get("app")
	limit := 50 // default limit
	offset := 0 // default offset

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || l != 1 || limit > 100 {
			limit = 50
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil || o != 1 || offset < 0 {
			offset = 0
		}
	}

	workflows, err := s.workflowExecutor.ListWorkflowExecutions(appName, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list workflows: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(workflows); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) handleGetWorkflow(w http.ResponseWriter, r *http.Request, workflowID int64) {
	workflow, err := s.workflowExecutor.GetWorkflowExecution(workflowID)
	if err != nil {
		if err.Error() == "workflow execution not found" {
			http.Error(w, "Workflow not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get workflow: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(workflow); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// HandleTeams handles team-related API requests
func (s *Server) HandleTeams(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleListTeams(w, r)
	case "POST":
		s.handleCreateTeam(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleTeamDetail handles individual team requests
func (s *Server) HandleTeamDetail(w http.ResponseWriter, r *http.Request) {
	// Extract team ID from URL path
	path := r.URL.Path[len("/api/teams/"):]
	if path == "" {
		http.Error(w, "Team ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		s.handleGetTeam(w, r, path)
	case "DELETE":
		s.handleDeleteTeam(w, r, path)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleListTeams(w http.ResponseWriter, r *http.Request) {
	teams := s.teamManager.ListTeams()
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(teams); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if req.Name == "" {
		http.Error(w, "Team name is required", http.StatusBadRequest)
		return
	}
	
	team, err := s.teamManager.CreateTeam(req.Name, req.Description)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create team: %v", err), http.StatusConflict)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(team); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) handleGetTeam(w http.ResponseWriter, r *http.Request, teamID string) {
	team, exists := s.teamManager.GetTeam(teamID)
	if !exists {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(team); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) handleDeleteTeam(w http.ResponseWriter, r *http.Request, teamID string) {
	err := s.teamManager.DeleteTeam(teamID)
	if err != nil {
		if teamID == "default-team" {
			http.Error(w, "Cannot delete default team", http.StatusForbidden)
		} else {
			http.Error(w, fmt.Sprintf("Failed to delete team: %v", err), http.StatusNotFound)
		}
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Team '%s' deleted successfully", teamID),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// Rate limiting for login attempts
const (
	maxLoginAttempts = 5
	lockoutDuration  = 15 * time.Minute
)

func (s *Server) isRateLimited(clientIP string) bool {
	s.loginMutex.Lock()
	defer s.loginMutex.Unlock()

	now := time.Now()
	attempts, exists := s.loginAttempts[clientIP]

	if !exists {
		return false
	}

	// Remove old attempts outside the lockout window
	validAttempts := []time.Time{}
	for _, attempt := range attempts {
		if now.Sub(attempt) < lockoutDuration {
			validAttempts = append(validAttempts, attempt)
		}
	}
	s.loginAttempts[clientIP] = validAttempts

	return len(validAttempts) >= maxLoginAttempts
}

func (s *Server) recordLoginAttempt(clientIP string) {
	s.loginMutex.Lock()
	defer s.loginMutex.Unlock()

	now := time.Now()
	s.loginAttempts[clientIP] = append(s.loginAttempts[clientIP], now)
}

func (s *Server) clearLoginAttempts(clientIP string) {
	s.loginMutex.Lock()
	defer s.loginMutex.Unlock()

	delete(s.loginAttempts, clientIP)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// HandleHealth handles GET /health - Returns server health status
// HandleHealth returns the health status of the service and its dependencies
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	healthResponse := s.healthChecker.CheckAll(ctx)

	// Set HTTP status code based on health status
	statusCode := http.StatusOK
	if healthResponse.Status == health.StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	} else if healthResponse.Status == health.StatusDegraded {
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(healthResponse)
}

// HandleReady returns the readiness status for Kubernetes readiness probes
func (s *Server) HandleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	readinessResponse := s.healthChecker.IsReady(ctx)

	statusCode := http.StatusOK
	if !readinessResponse.Ready {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(readinessResponse)
}

// HandleMetrics returns Prometheus-format metrics
func (s *Server) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	metricsData := metrics.GetGlobal().Export()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(metricsData))
}

// Memory workflow tracking methods

// CreateMemoryWorkflowExecution creates a new workflow execution in memory
func (s *Server) CreateMemoryWorkflowExecution(appName, workflowName string, stepCount int) *MemoryWorkflowExecution {
	s.workflowMutex.Lock()
	defer s.workflowMutex.Unlock()

	s.workflowCounter++
	execution := &MemoryWorkflowExecution{
		ID:           s.workflowCounter,
		AppName:      appName,
		WorkflowName: workflowName,
		Status:       "running",
		StartedAt:    time.Now(),
		StepCount:    stepCount,
		Steps:        make([]*MemoryWorkflowStep, 0, stepCount),
	}

	s.memoryWorkflows[execution.ID] = execution

	// Save workflows to disk
	s.saveWorkflowsToDisk()

	return execution
}

// CreateMemoryWorkflowStep creates a new workflow step in memory
func (s *Server) CreateMemoryWorkflowStep(executionID int64, stepNumber int, name, stepType string) *MemoryWorkflowStep {
	s.workflowMutex.Lock()
	defer s.workflowMutex.Unlock()

	execution, exists := s.memoryWorkflows[executionID]
	if !exists {
		return nil
	}

	step := &MemoryWorkflowStep{
		ID:         int64(len(execution.Steps) + 1),
		StepNumber: stepNumber,
		Name:       name,
		Type:       stepType,
		Status:     "running",
		StartedAt:  time.Now(),
	}

	execution.Steps = append(execution.Steps, step)

	// Save workflows to disk
	s.saveWorkflowsToDisk()

	return step
}

// UpdateMemoryWorkflowStepStatus updates a workflow step status in memory
func (s *Server) UpdateMemoryWorkflowStepStatus(executionID int64, stepNumber int, status string, errorMessage *string) {
	s.workflowMutex.Lock()
	defer s.workflowMutex.Unlock()

	execution, exists := s.memoryWorkflows[executionID]
	if !exists {
		return
	}

	for _, step := range execution.Steps {
		if step.StepNumber == stepNumber {
			step.Status = status
			now := time.Now()
			step.CompletedAt = &now
			if errorMessage != nil {
				step.ErrorMessage = errorMessage
			}
			break
		}
	}

	// Save workflows to disk
	s.saveWorkflowsToDisk()
}

// UpdateMemoryWorkflowExecutionStatus updates a workflow execution status in memory
func (s *Server) UpdateMemoryWorkflowExecutionStatus(executionID int64, status string, errorMessage *string) {
	s.workflowMutex.Lock()
	defer s.workflowMutex.Unlock()

	execution, exists := s.memoryWorkflows[executionID]
	if !exists {
		return
	}

	execution.Status = status
	now := time.Now()
	execution.CompletedAt = &now
	if errorMessage != nil {
		execution.ErrorMessage = errorMessage
	}

	// Save workflows to disk
	s.saveWorkflowsToDisk()
}

// GetMemoryWorkflowExecution retrieves a workflow execution from memory
func (s *Server) GetMemoryWorkflowExecution(executionID int64) *MemoryWorkflowExecution {
	s.workflowMutex.RLock()
	defer s.workflowMutex.RUnlock()

	return s.memoryWorkflows[executionID]
}

// ListMemoryWorkflowExecutions lists workflow executions from memory
func (s *Server) ListMemoryWorkflowExecutions(appName string, limit, offset int) []*MemoryWorkflowExecution {
	s.workflowMutex.RLock()
	defer s.workflowMutex.RUnlock()

	var executions []*MemoryWorkflowExecution
	for _, execution := range s.memoryWorkflows {
		if appName == "" || execution.AppName == appName {
			executions = append(executions, execution)
		}
	}

	// Sort by ID (newest first)
	for i := 0; i < len(executions)-1; i++ {
		for j := i + 1; j < len(executions); j++ {
			if executions[i].ID < executions[j].ID {
				executions[i], executions[j] = executions[j], executions[i]
			}
		}
	}

	// Apply pagination
	start := offset
	if start >= len(executions) {
		return []*MemoryWorkflowExecution{}
	}

	end := start + limit
	if end > len(executions) {
		end = len(executions)
	}

	return executions[start:end]
}

// runWorkflowWithTracking executes a workflow with step-by-step tracking
func (s *Server) runWorkflowWithTracking(workflowDef types.Workflow, appName, envType string, memoryExecution *MemoryWorkflowExecution) error {
	// If database is available, use the standard database-tracked execution
	if s.workflowExecutor != nil {
		return workflow.RunWorkflow(workflowDef, appName, envType)
	}

	// Otherwise, use in-memory tracking - just delegate to the existing RunWorkflow for now
	// In the future, we could create a custom implementation that tracks each step
	return workflow.RunWorkflow(workflowDef, appName, envType)
}

// handleListMemoryWorkflows handles listing workflow executions from memory
func (s *Server) handleListMemoryWorkflows(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	appName := r.URL.Query().Get("app")
	limit := 50 // default limit
	offset := 0 // default offset

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || l != 1 || limit > 100 {
			limit = 50
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil || o != 1 || offset < 0 {
			offset = 0
		}
	}

	workflows := s.ListMemoryWorkflowExecutions(appName, limit, offset)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(workflows); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// handleGetMemoryWorkflow handles getting a specific workflow execution from memory
func (s *Server) handleGetMemoryWorkflow(w http.ResponseWriter, r *http.Request) {
	// Extract workflow ID from URL path
	path := r.URL.Path[len("/api/workflows/"):]
	if path == "" {
		http.Error(w, "Workflow ID required", http.StatusBadRequest)
		return
	}

	var workflowID int64
	_, err := fmt.Sscanf(path, "%d", &workflowID)
	if err != nil {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		workflow := s.GetMemoryWorkflowExecution(workflowID)
		if workflow == nil {
			http.Error(w, "Workflow not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(workflow); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Workflow persistence methods

// saveWorkflowsToDisk saves workflow executions to disk
func (s *Server) saveWorkflowsToDisk() {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll("data", 0755); err != nil {
		fmt.Printf("Warning: Failed to create data directory: %v\n", err)
		return
	}

	// Marshal workflow data
	data := struct {
		Workflows       map[int64]*MemoryWorkflowExecution `json:"workflows"`
		WorkflowCounter int64                               `json:"workflow_counter"`
	}{
		Workflows:       s.memoryWorkflows,
		WorkflowCounter: s.workflowCounter,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("Warning: Failed to marshal workflow data: %v\n", err)
		return
	}

	// Write to file
	if err := os.WriteFile("data/workflows.json", jsonData, 0644); err != nil {
		fmt.Printf("Warning: Failed to write workflow file: %v\n", err)
	}
}

// loadWorkflowsFromDisk loads workflow executions from disk
func (s *Server) loadWorkflowsFromDisk() {
	filePath := "data/workflows.json"

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist, start with empty workflows
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Warning: Failed to read workflow file: %v\n", err)
		return
	}

	var workflowData struct {
		Workflows       map[int64]*MemoryWorkflowExecution `json:"workflows"`
		WorkflowCounter int64                               `json:"workflow_counter"`
	}

	if err := json.Unmarshal(data, &workflowData); err != nil {
		fmt.Printf("Warning: Failed to unmarshal workflow data: %v\n", err)
		return
	}

	// Load data into memory
	if workflowData.Workflows != nil {
		s.memoryWorkflows = workflowData.Workflows
		fmt.Printf("⚙️  Loaded %d workflow executions from disk\n", len(s.memoryWorkflows))
	}

	if workflowData.WorkflowCounter > 0 {
		s.workflowCounter = workflowData.WorkflowCounter
	}
}

// Demo Environment API handlers

// HandleDemoStatus handles GET /api/demo/status - Returns component health status
func (s *Server) HandleDemoStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create demo environment and health checker
	env := demo.NewDemoEnvironment()
	healthChecker := demo.NewHealthChecker(5 * time.Second)

	// Perform actual health checks
	healthResults := healthChecker.CheckAll(env)

	// Build response with actual health status
	components := []map[string]interface{}{}
	for _, result := range healthResults {
		component := map[string]interface{}{
			"name":       result.Name,
			"url":        getComponentURL(result.Name, result.Host),
			"status":     result.Healthy,
			"credentials": getComponentCredentials(result.Name, env),
			"health":     result.Status,
			"latency_ms": result.Latency.Milliseconds(),
		}
		if result.Error != "" {
			component["error"] = result.Error
		}
		components = append(components, component)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"components": components,
		"timestamp":  time.Now(),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// getComponentURL returns the URL for a component based on its name and host
func getComponentURL(name, host string) string {
	if host == "" {
		return ""
	}
	protocol := "http"
	if name == "kubernetes-dashboard" {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s", protocol, host)
}

// getComponentCredentials returns the credentials for a component from environment
func getComponentCredentials(name string, env *demo.DemoEnvironment) string {
	// Find component in environment
	for _, comp := range env.Components {
		if comp.Name == name {
			if username, ok := comp.Credentials["username"]; ok {
				if password, ok := comp.Credentials["password"]; ok {
					return fmt.Sprintf("%s / %s", username, password)
				}
			}
		}
	}

	// Special cases for components without username/password
	switch name {
	case "vault":
		return "root"
	case "prometheus":
		return "none"
	case "kubernetes-dashboard":
		return "kubectl -n kubernetes-dashboard create token admin-user"
	case "demo-app":
		return "-"
	case "nginx-ingress":
		return "-"
	}
	return ""
}

// HandleDemoTime handles POST /api/demo/time - Execute demo-time command
func (s *Server) HandleDemoTime(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Integrate with existing CLI demo functionality
	// This would call the demo.DemoTimeCommand() from internal/cli/commands.go

	response := map[string]interface{}{
		"message":   "Demo Time command initiated",
		"status":    "running",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// HandleDemoNuke handles POST /api/demo/nuke - Execute demo-nuke command
func (s *Server) HandleDemoNuke(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Integrate with existing CLI demo functionality
	// This would call the demo.DemoNukeCommand() from internal/cli/commands.go

	response := map[string]interface{}{
		"message":   "Demo Nuke command initiated",
		"status":    "running",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// HandleStats handles GET /api/stats - Returns dashboard statistics
func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by authentication middleware)
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Count applications
	var specs map[string]*types.ScoreSpec
	if user.IsAdmin() {
		specs = s.storage.ListSpecs()
	} else {
		specs = s.storage.ListSpecsByTeam(user.Team)
	}
	applicationsCount := len(specs)

	// Count workflows
	var workflowsCount int
	if s.workflowExecutor != nil {
		// Use database workflow count if available
		workflows, err := s.workflowExecutor.ListWorkflowExecutions("", 1000, 0)
		if err == nil {
			workflowsCount = len(workflows)
		}
	} else {
		// Use memory workflow count
		s.workflowMutex.RLock()
		workflowsCount = len(s.memoryWorkflows)
		s.workflowMutex.RUnlock()
	}

	// Count resources across all specs
	resourcesCount := 0
	for _, spec := range specs {
		if spec.Resources != nil {
			resourcesCount += len(spec.Resources)
		}
	}

	// Count users (simple implementation - could be enhanced)
	usersCount := 4 // Default fallback - in production this would count actual users

	stats := map[string]interface{}{
		"applications": applicationsCount,
		"workflows":    workflowsCount,
		"resources":    resourcesCount,
		"users":        usersCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// startWorkflowScheduler starts a background goroutine that triggers dummy workflows every minute
func (s *Server) startWorkflowScheduler() {
	s.workflowTicker = time.NewTicker(1 * time.Minute)
	s.stopScheduler = make(chan struct{})

	go func() {
		fmt.Println("Workflow scheduler started - triggering dummy workflow every minute")
		for {
			select {
			case <-s.workflowTicker.C:
				s.triggerDummyWorkflow()
			case <-s.stopScheduler:
				fmt.Println("Workflow scheduler stopped")
				return
			}
		}
	}()
}

// stopWorkflowScheduler stops the background workflow scheduler
//nolint:unused // Reserved for future graceful shutdown implementation
func (s *Server) stopWorkflowScheduler() {
	if s.workflowTicker != nil {
		s.workflowTicker.Stop()
	}
	if s.stopScheduler != nil {
		close(s.stopScheduler)
	}
}

// triggerDummyWorkflow loads and executes the dummy workflow
func (s *Server) triggerDummyWorkflow() {
	// Only trigger if we have a workflow executor (database available)
	if s.workflowExecutor == nil {
		return
	}

	// Load the dummy workflow from file
	dummyWorkflow, err := s.loadWorkflowFromFile("workflows/dummy.yaml")
	if err != nil {
		fmt.Printf("Failed to load dummy workflow: %v\n", err)
		return
	}

	// Execute the dummy workflow
	fmt.Println("Triggering scheduled dummy workflow execution...")
	err = s.workflowExecutor.ExecuteWorkflowWithName("scheduled", "dummy", *dummyWorkflow)
	if err != nil {
		fmt.Printf("Failed to execute dummy workflow: %v\n", err)
	} else {
		fmt.Println("✅ Scheduled dummy workflow completed successfully")
	}
}

// loadWorkflowFromFile loads a workflow definition from a YAML file
func (s *Server) loadWorkflowFromFile(filePath string) (*types.Workflow, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	var workflow types.Workflow
	err = yaml.Unmarshal(data, &workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	return &workflow, nil
}

// HandleWorkflowAnalysis handles workflow analysis API requests
func (s *Server) HandleWorkflowAnalysis(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		s.handleAnalyzeWorkflow(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAnalyzeWorkflow analyzes a Score specification and returns workflow analysis
func (s *Server) handleAnalyzeWorkflow(w http.ResponseWriter, r *http.Request) {
	// Initialize analyzer if not already done
	if s.workflowAnalyzer == nil {
		s.workflowAnalyzer = workflow.NewWorkflowAnalyzer()
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse Score specification
	var spec types.ScoreSpec
	if err := yaml.Unmarshal(body, &spec); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse Score specification: %v", err), http.StatusBadRequest)
		return
	}

	// Validate spec has required fields
	if spec.Metadata.Name == "" {
		http.Error(w, "Score specification must have metadata.name", http.StatusBadRequest)
		return
	}

	// Perform workflow analysis
	analysis, err := s.workflowAnalyzer.AnalyzeSpec(&spec)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to analyze workflow: %v", err), http.StatusInternalServerError)
		return
	}

	// Return analysis result
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(analysis); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleWorkflowAnalysisPreview handles quick analysis preview for Score specs
func (s *Server) HandleWorkflowAnalysisPreview(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		s.handleAnalyzeWorkflowPreview(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAnalyzeWorkflowPreview provides a quick analysis preview
func (s *Server) handleAnalyzeWorkflowPreview(w http.ResponseWriter, r *http.Request) {
	// Initialize analyzer if not already done
	if s.workflowAnalyzer == nil {
		s.workflowAnalyzer = workflow.NewWorkflowAnalyzer()
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Check for empty body
	if len(body) == 0 {
		http.Error(w, "Request body cannot be empty", http.StatusBadRequest)
		return
	}

	// Parse Score specification
	var spec types.ScoreSpec
	if err := yaml.Unmarshal(body, &spec); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse Score specification: %v", err), http.StatusBadRequest)
		return
	}

	// Perform workflow analysis
	analysis, err := s.workflowAnalyzer.AnalyzeSpec(&spec)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to analyze workflow: %v", err), http.StatusInternalServerError)
		return
	}

	// Create a simplified preview response
	preview := map[string]interface{}{
		"summary": analysis.Summary,
		"executionPlan": map[string]interface{}{
			"totalTime":    analysis.ExecutionPlan.TotalTime.String(),
			"phases":       len(analysis.ExecutionPlan.Phases),
			"maxParallel":  analysis.ExecutionPlan.MaxParallel,
		},
		"resourceGraph": map[string]interface{}{
			"nodes": len(analysis.ResourceGraph.Nodes),
			"edges": len(analysis.ResourceGraph.Edges),
		},
		"warnings":        analysis.Warnings,
		"recommendations": analysis.Recommendations,
	}

	// Return preview result
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(preview); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleApplicationManagement handles application-level operations (delete, deprovision)
func (s *Server) HandleApplicationManagement(w http.ResponseWriter, r *http.Request) {
	// Extract application name from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Application name required", http.StatusBadRequest)
		return
	}

	appName := pathParts[3]

	// Handle deprovision endpoint
	if len(pathParts) == 5 && pathParts[4] == "deprovision" {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleDeprovisionApplication(w, r, appName)
		return
	}

	// Handle delete endpoint
	if len(pathParts) == 4 {
		if r.Method != "DELETE" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleDeleteApplication(w, r, appName)
		return
	}

	http.Error(w, "Invalid endpoint", http.StatusNotFound)
}

// handleDeleteApplication performs complete application deletion (infrastructure + database records)
func (s *Server) handleDeleteApplication(w http.ResponseWriter, r *http.Request, appName string) {
	// Get user from context (set by authentication middleware)
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if application exists
	storedSpec, exists := s.storage.GetStoredSpec(appName)
	if !exists {
		http.Error(w, "Application not found", http.StatusNotFound)
		return
	}

	// Check if user has access to this application
	if !user.IsAdmin() && storedSpec.Team != user.Team {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Use resource manager to delete application if available
	if s.resourceManager != nil {
		err := s.resourceManager.DeleteApplication(appName, user.Username)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete application: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Also remove from storage (spec records)
	err := s.storage.DeleteSpec(appName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete application spec: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": fmt.Sprintf("Successfully deleted application '%s' and all its resources", appName),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// handleDeprovisionApplication performs infrastructure teardown with audit trail preserved
func (s *Server) handleDeprovisionApplication(w http.ResponseWriter, r *http.Request, appName string) {
	// Get user from context (set by authentication middleware)
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if application exists
	storedSpec, exists := s.storage.GetStoredSpec(appName)
	if !exists {
		http.Error(w, "Application not found", http.StatusNotFound)
		return
	}

	// Check if user has access to this application
	if !user.IsAdmin() && storedSpec.Team != user.Team {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Use resource manager to deprovision application if available
	if s.resourceManager != nil {
		err := s.resourceManager.DeprovisionApplication(appName, user.Username)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to deprovision application: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Resource management not available", http.StatusServiceUnavailable)
		return
	}

	response := map[string]string{
		"message": fmt.Sprintf("Successfully deprovisioned infrastructure for application '%s'", appName),
		"note":    "Application metadata and audit trail preserved in database",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// HandleGoldenPathExecution handles golden path workflow execution with resource management integration
func (s *Server) HandleGoldenPathExecution(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract golden path name from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "Golden path name required", http.StatusBadRequest)
		return
	}

	goldenPathName := pathParts[4]
	if !strings.HasSuffix(goldenPathName, "/execute") {
		goldenPathName = strings.TrimSuffix(goldenPathName, "/execute")
	} else {
		goldenPathName = strings.TrimSuffix(goldenPathName, "/execute")
	}

	// Get user from context (set by authentication middleware)
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Read Score spec from request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	var spec types.ScoreSpec
	err = yaml.Unmarshal(body, &spec)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing Score spec: %v", err), http.StatusBadRequest)
		return
	}

	if spec.Metadata.Name == "" {
		http.Error(w, "Score spec must have metadata.name", http.StatusBadRequest)
		return
	}

	fmt.Printf("🚀 Executing golden path '%s' for application: %s\n", goldenPathName, spec.Metadata.Name)

	// Load golden path workflow
	workflowFile := fmt.Sprintf("./workflows/%s.yaml", goldenPathName)
	workflowData, err := os.ReadFile(workflowFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Golden path '%s' not found", goldenPathName), http.StatusNotFound)
		return
	}

	var workflowSpec types.WorkflowSpec
	err = yaml.Unmarshal(workflowData, &workflowSpec)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse workflow: %v", err), http.StatusInternalServerError)
		return
	}

	// Extract the actual workflow from the spec
	workflow := workflowSpec.Spec

	// Store the Score spec first
	err = s.storage.AddSpec(spec.Metadata.Name, &spec, user.Team, user.Username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error storing spec: %v", err), http.StatusInternalServerError)
		return
	}

	// Create resource instances if database is available
	if s.resourceManager != nil && s.db != nil {
		fmt.Printf("📦 Creating resource instances for app '%s'...\n", spec.Metadata.Name)
		err = s.resourceManager.CreateResourceFromSpec(spec.Metadata.Name, &spec, user.Username)
		if err != nil {
			fmt.Printf("Warning: Failed to create resource instances: %v\n", err)
			// Continue with workflow execution even if resource creation fails
		}
	}

	// Execute workflow with enhanced resource management integration
	var workflowExecution *database.WorkflowExecution
	if s.workflowExecutor != nil {
		// Create workflow execution record
		workflowExecution, err = s.workflowRepo.CreateWorkflowExecution(spec.Metadata.Name, fmt.Sprintf("golden-path-%s", goldenPathName), len(workflow.Steps))
		if err != nil {
			fmt.Printf("Warning: Failed to create workflow execution record: %v\n", err)
		}

		// Execute workflow with resource integration
		err = s.executeGoldenPathWorkflowWithResources(&workflow, &spec, user.Username, workflowExecution)
		if err != nil {
			// Mark workflow as failed
			if workflowExecution != nil {
				errorMsg := err.Error()
				if updateErr := s.workflowRepo.UpdateWorkflowExecution(workflowExecution.ID, database.WorkflowStatusFailed, &errorMsg); updateErr != nil {
					fmt.Fprintf(os.Stderr, "failed to update workflow status: %v\n", updateErr)
				}
			}
			http.Error(w, fmt.Sprintf("Workflow execution failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Mark workflow as completed
		if workflowExecution != nil {
			if err := s.workflowRepo.UpdateWorkflowExecution(workflowExecution.ID, database.WorkflowStatusCompleted, nil); err != nil {
				fmt.Fprintf(os.Stderr, "failed to update workflow status: %v\n", err)
			}
		}
	} else {
		// Fallback to basic workflow execution without database tracking
		err = s.executeBasicGoldenPathWorkflow(&workflow, &spec, user.Username)
		if err != nil {
			http.Error(w, fmt.Sprintf("Workflow execution failed: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Provision resources after successful workflow execution
	if s.resourceManager != nil && s.db != nil {
		err = s.provisionResourcesAfterWorkflow(spec.Metadata.Name, user.Username)
		if err != nil {
			fmt.Printf("Warning: Resource provisioning failed: %v\n", err)
			// Don't fail the entire golden path execution
		}
	}

	response := map[string]interface{}{
		"message":      fmt.Sprintf("Golden path '%s' executed successfully for application '%s'", goldenPathName, spec.Metadata.Name),
		"application":  spec.Metadata.Name,
		"golden_path":  goldenPathName,
		"workflow_id":  nil,
	}

	if workflowExecution != nil {
		response["workflow_id"] = workflowExecution.ID
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// executeGoldenPathWorkflowWithResources executes a workflow with full resource management integration
func (s *Server) executeGoldenPathWorkflowWithResources(workflow *types.Workflow, spec *types.ScoreSpec, username string, execution *database.WorkflowExecution) error {
	fmt.Printf("📋 Executing workflow with %d steps for %s\n", len(workflow.Steps), spec.Metadata.Name)

	for i, step := range workflow.Steps {
		fmt.Printf("🔄 Step %d/%d: %s (%s)\n", i+1, len(workflow.Steps), step.Name, step.Type)

		// Create step execution record if database tracking is available
		var stepRecord *database.WorkflowStepExecution
		if execution != nil {
			stepConfig := map[string]interface{}{
				"name":      step.Name,
				"type":      step.Type,
				"path":      step.Path,
				"namespace": step.Namespace,
			}

			var err error
			stepRecord, err = s.workflowRepo.CreateWorkflowStep(execution.ID, i+1, step.Name, step.Type, stepConfig)
			if err != nil {
				fmt.Printf("Warning: Failed to create step record: %v\n", err)
			} else {
				// Mark step as running
				if err := s.workflowRepo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusRunning, nil); err != nil {
					fmt.Fprintf(os.Stderr, "failed to update step status: %v\n", err)
				}
			}
		}

		// Execute the step
		stepContext := &StepExecutionContext{
			StepID:      &stepRecord.ID,
			WorkflowRepo: s.workflowRepo,
		}
		err := s.runWorkflowStepWithTracking(step, spec.Metadata.Name, "default", stepContext)
		if err != nil {
			// Mark step as failed
			if stepRecord != nil {
				errorMsg := err.Error()
				if updateErr := s.workflowRepo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusFailed, &errorMsg); updateErr != nil {
					fmt.Fprintf(os.Stderr, "failed to update step status: %v\n", updateErr)
				}
			}
			return fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		// Mark step as completed
		if stepRecord != nil {
			if err := s.workflowRepo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusCompleted, nil); err != nil {
				fmt.Fprintf(os.Stderr, "failed to update step status: %v\n", err)
			}
		}

		fmt.Printf("✅ Step %s completed successfully\n", step.Name)
	}

	return nil
}

// executeBasicGoldenPathWorkflow executes a workflow without database tracking (fallback)
func (s *Server) executeBasicGoldenPathWorkflow(workflow *types.Workflow, spec *types.ScoreSpec, username string) error {
	fmt.Printf("📋 Executing basic workflow with %d steps for %s\n", len(workflow.Steps), spec.Metadata.Name)

	for i, step := range workflow.Steps {
		fmt.Printf("🔄 Step %d/%d: %s (%s)\n", i+1, len(workflow.Steps), step.Name, step.Type)

		// For basic workflow, create minimal context without database tracking
		stepContext := &StepExecutionContext{
			StepID:      nil, // No database tracking for basic workflow
			WorkflowRepo: nil,
		}
		err := s.runWorkflowStepWithTracking(step, spec.Metadata.Name, "default", stepContext)
		if err != nil {
			return fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		fmt.Printf("✅ Step %s completed successfully\n", step.Name)
	}

	return nil
}

// substituteVariables replaces template variables in step fields
func substituteVariables(step *types.Step, appName string, envType string) {
	replacer := strings.NewReplacer(
		"${metadata.name}", appName,
		"${environment.type}", envType,
	)

	// Substitute in common string fields
	step.Namespace = replacer.Replace(step.Namespace)
	step.RepoName = replacer.Replace(step.RepoName)
	step.AppName = replacer.Replace(step.AppName)
	step.Description = replacer.Replace(step.Description)
	step.Path = replacer.Replace(step.Path)
	step.Playbook = replacer.Replace(step.Playbook)
	step.ManifestPath = replacer.Replace(step.ManifestPath)
	step.TargetPath = replacer.Replace(step.TargetPath)
	step.Owner = replacer.Replace(step.Owner)
	step.SyncPolicy = replacer.Replace(step.SyncPolicy)
}

// runWorkflowStepWithTracking executes a single workflow step with real command execution and output capture
func (s *Server) runWorkflowStepWithTracking(step types.Step, appName string, envType string, stepContext *StepExecutionContext) error {
	// Substitute variables in step fields
	substituteVariables(&step, appName, envType)

	// Create log buffer for this step
	logBuffer := &LogBuffer{
		repo: stepContext.WorkflowRepo,
	}

	// Only set stepID if we have database tracking enabled
	if stepContext.StepID != nil {
		logBuffer.stepID = stepContext.StepID
	}

	// Log step start
	if _, err := logBuffer.Write([]byte(fmt.Sprintf("Starting step: %s (type: %s)", step.Name, step.Type))); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write log: %v\n", err)
	}

	// Execute the step based on its type
	switch step.Type {
	case "terraform":
		fmt.Printf("   🏗️  Executing Terraform step: %s\n", step.Name)
		return s.executeTerraformStep(step, appName, envType, logBuffer)
	case "kubernetes":
		fmt.Printf("   ⚓ Executing Kubernetes step: %s\n", step.Name)
		return s.executeKubernetesStep(step, appName, envType, logBuffer)
	case "gitea-repo":
		fmt.Printf("   🗂️  Executing Gitea repository step: %s\n", step.Name)
		return s.executeGiteaRepoStep(step, appName, envType, logBuffer)
	case "argocd-app":
		fmt.Printf("   🔄 Executing ArgoCD application step: %s\n", step.Name)
		return s.executeArgoCDStep(step, appName, envType, logBuffer)
	case "git-commit-manifests":
		fmt.Printf("   📝 Executing Git commit step: %s\n", step.Name)
		return s.executeGitCommitStep(step, appName, envType, logBuffer)
	case "ansible":
		fmt.Printf("   🔧 Executing Ansible step: %s\n", step.Name)
		return s.executeAnsibleStep(step, appName, envType, logBuffer)
	case "policy":
		fmt.Printf("   📋 Executing Policy step: %s\n", step.Name)
		return s.executePolicyStep(step, appName, envType, logBuffer)
	case "dummy":
		fmt.Printf("   🎭 Executing Dummy step: %s\n", step.Name)
		return s.executeDummyStep(step, appName, envType, logBuffer)
	default:
		fmt.Printf("   ❓ Executing unknown step type: %s\n", step.Type)
		if _, err := logBuffer.Write([]byte(fmt.Sprintf("Warning: Unknown step type '%s', skipping execution", step.Type))); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write log: %v\n", err)
		}
		return nil
	}
}

// executeCommand runs a command and captures output to the log buffer
func (s *Server) executeCommand(command string, args []string, workDir string, logBuffer *LogBuffer) error {
	cmd := exec.Command(command, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Set up combined output capture
	cmd.Stdout = logBuffer
	cmd.Stderr = logBuffer

	execMsg := fmt.Sprintf("Executing: %s %s", command, strings.Join(args, " "))
	if _, err := logBuffer.Write([]byte(execMsg)); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write log: %v\n", err)
	}
	fmt.Println("   " + execMsg)

	err := cmd.Run()
	if err != nil {
		errMsg := fmt.Sprintf("Command failed with error: %v", err)
		if _, writeErr := logBuffer.Write([]byte(errMsg)); writeErr != nil {
			fmt.Fprintf(os.Stderr, "failed to write error log: %v\n", writeErr)
		}
		fmt.Println("   " + errMsg)
		return err
	}

	if _, err := logBuffer.Write([]byte("Command completed successfully")); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write log: %v\n", err)
	}
	fmt.Println("   Command completed successfully")
	return nil
}

// executeTerraformStep executes a terraform workflow step
func (s *Server) executeTerraformStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	workDir := fmt.Sprintf("./terraform/%s-%s", appName, envType)

	// Create workspace directory if it doesn't exist
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		err = os.MkdirAll(workDir, 0755)
		if err != nil {
			_, _ = logBuffer.Write([]byte(fmt.Sprintf("Failed to create workspace directory: %v", err)))
			return err
		}
	}

	// Copy terraform files from step.Path to workspace
	if step.Path != "" {
		_, _ = logBuffer.Write([]byte(fmt.Sprintf("Copying terraform files from %s to %s", step.Path, workDir)))
		err := s.executeCommand("cp", []string{"-r", step.Path + "/.", workDir}, "", logBuffer)
		if err != nil {
			return err
		}
	}

	// Run terraform init
	err := s.executeCommand("terraform", []string{"init"}, workDir, logBuffer)
	if err != nil {
		return err
	}

	// Run terraform plan
	err = s.executeCommand("terraform", []string{"plan"}, workDir, logBuffer)
	if err != nil {
		return err
	}

	// Run terraform apply
	return s.executeCommand("terraform", []string{"apply", "-auto-approve"}, workDir, logBuffer)
}

// executeKubernetesStep executes a kubernetes workflow step
func (s *Server) executeKubernetesStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	namespace := step.Namespace
	if namespace == "" {
		namespace = fmt.Sprintf("%s-%s", appName, envType)
	}

	// Create namespace if it doesn't exist
	_, _ = logBuffer.Write([]byte(fmt.Sprintf("Creating namespace: %s", namespace)))
	err := s.executeCommand("kubectl", []string{"create", "namespace", namespace}, "", logBuffer)
	if err != nil {
		// Namespace might already exist, which is fine
		logBuffer.Write([]byte("Namespace may already exist, continuing..."))
	}

	// Generate and apply kubernetes manifests (simplified for now)
	manifestPath := fmt.Sprintf("/tmp/%s-%s-manifests.yaml", appName, envType)

	// Create a simple deployment manifest
	manifest := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: main
        image: nginx:latest
        ports:
        - containerPort: 80
`, appName, namespace, appName, appName)

	err = os.WriteFile(manifestPath, []byte(manifest), 0644)
	if err != nil {
		logBuffer.Write([]byte(fmt.Sprintf("Failed to write manifest file: %v", err)))
		return err
	}

	return s.executeCommand("kubectl", []string{"apply", "-f", manifestPath}, "", logBuffer)
}

// executeGiteaRepoStep executes a gitea repository creation step
func (s *Server) executeGiteaRepoStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	repoName := step.RepoName
	if repoName == "" {
		repoName = fmt.Sprintf("%s-%s", appName, envType)
	}

	logBuffer.Write([]byte(fmt.Sprintf("Creating Gitea repository: %s", repoName)))

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		logBuffer.Write([]byte(fmt.Sprintf("Failed to load admin config: %v", err)))
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	owner := step.Owner
	if owner == "" {
		owner = adminConfig.Gitea.OrgName
	}

	// Create repository using Gitea API
	repoData := map[string]interface{}{
		"name":        repoName,
		"description": step.Description,
		"private":     false,
		"auto_init":   true,
	}

	repoJSON, err := json.Marshal(repoData)
	if err != nil {
		return fmt.Errorf("failed to marshal repository data: %w", err)
	}

	// Try creating in organization first, fallback to user if that fails
	createURL := fmt.Sprintf("%s/api/v1/orgs/%s/repos", adminConfig.Gitea.URL, owner)
	req, err := http.NewRequest("POST", createURL, strings.NewReader(string(repoJSON)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(adminConfig.Gitea.Username, adminConfig.Gitea.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logBuffer.Write([]byte(fmt.Sprintf("Failed to create repository: %v", err)))
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// If org doesn't exist (404), create under user account instead
	if resp.StatusCode == 404 {
		logBuffer.Write([]byte(fmt.Sprintf("Organization '%s' not found, creating repository under user account", owner)))
		createURL = fmt.Sprintf("%s/api/v1/user/repos", adminConfig.Gitea.URL)
		owner = adminConfig.Gitea.Username

		req, err = http.NewRequest("POST", createURL, strings.NewReader(string(repoJSON)))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.SetBasicAuth(adminConfig.Gitea.Username, adminConfig.Gitea.Password)
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(req)
		if err != nil {
			logBuffer.Write([]byte(fmt.Sprintf("Failed to create repository: %v", err)))
			return fmt.Errorf("failed to create repository: %w", err)
		}
		defer resp.Body.Close()

		body, _ = io.ReadAll(resp.Body)
	}

	if resp.StatusCode == 409 {
		logBuffer.Write([]byte("Repository already exists, continuing..."))
	} else if resp.StatusCode != 201 {
		errMsg := fmt.Sprintf("Failed to create repository, status %d: %s", resp.StatusCode, string(body))
		logBuffer.Write([]byte(errMsg))
		return fmt.Errorf("failed to create repository, status %d: %s", resp.StatusCode, string(body))
	} else {
		logBuffer.Write([]byte("Repository created successfully"))
	}

	// Clone repository locally for manifest commits
	repoDir := fmt.Sprintf("/tmp/%s-%s-repo", appName, envType)
	repoURL := fmt.Sprintf("%s/%s/%s.git", adminConfig.Gitea.URL, owner, repoName)

	// Remove existing directory if present
	_ = s.executeCommand("rm", []string{"-rf", repoDir}, "", logBuffer)

	// Clone repository
	err = s.executeCommand("git", []string{"clone", repoURL, repoDir}, "", logBuffer)
	if err != nil {
		logBuffer.Write([]byte(fmt.Sprintf("Failed to clone repository: %v", err)))
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	logBuffer.Write([]byte(fmt.Sprintf("Repository cloned to: %s", repoDir)))
	return nil
}

// executeArgoCDStep executes an ArgoCD application creation step
func (s *Server) executeArgoCDStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	appNameArgo := step.AppName
	if appNameArgo == "" {
		appNameArgo = fmt.Sprintf("%s-%s", appName, envType)
	}

	logBuffer.Write([]byte(fmt.Sprintf("Creating ArgoCD application: %s", appNameArgo)))

	// Load admin configuration to get Gitea URL
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		logBuffer.Write([]byte(fmt.Sprintf("Failed to load admin config: %v", err)))
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	// Construct repository URL from repoName
	// Use internal URL for ArgoCD (in-cluster access)
	repoName := step.RepoName
	if repoName == "" {
		repoName = appName
	}

	// Use internal URL if available, fallback to external URL
	giteaURL := adminConfig.Gitea.InternalURL
	if giteaURL == "" {
		giteaURL = adminConfig.Gitea.URL
	}
	repoURL := fmt.Sprintf("%s/%s/%s", giteaURL, adminConfig.Gitea.Username, repoName)

	targetPath := step.TargetPath
	if targetPath == "" {
		targetPath = "manifests"
	}

	namespace := step.Namespace
	if namespace == "" {
		namespace = fmt.Sprintf("%s-%s", appName, envType)
	}

	// Create ArgoCD application manifest
	manifest := fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: argocd
spec:
  project: default
  source:
    repoURL: %s
    targetRevision: HEAD
    path: %s
  destination:
    server: https://kubernetes.default.svc
    namespace: %s
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
`, appNameArgo, repoURL, targetPath, namespace)

	manifestPath := fmt.Sprintf("/tmp/%s-argocd-app.yaml", appNameArgo)
	err = os.WriteFile(manifestPath, []byte(manifest), 0644)
	if err != nil {
		logBuffer.Write([]byte(fmt.Sprintf("Failed to write ArgoCD manifest: %v", err)))
		return err
	}

	return s.executeCommand("kubectl", []string{"apply", "-f", manifestPath}, "", logBuffer)
}

// executeGitCommitStep executes a git commit and push step
func (s *Server) executeGitCommitStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	repoDir := fmt.Sprintf("/tmp/%s-%s-repo", appName, envType)

	logBuffer.Write([]byte(fmt.Sprintf("Committing manifests to repository in %s", repoDir)))

	// Create manifests directory if it doesn't exist
	manifestDir := fmt.Sprintf("%s/%s", repoDir, step.ManifestPath)
	if step.ManifestPath == "" {
		manifestDir = fmt.Sprintf("%s/manifests", repoDir)
	}

	err := os.MkdirAll(manifestDir, 0755)
	if err != nil {
		logBuffer.Write([]byte(fmt.Sprintf("Failed to create manifest directory: %v", err)))
		return err
	}

	// Copy kubernetes manifests to repository
	manifestPath := fmt.Sprintf("/tmp/%s-%s-manifests.yaml", appName, envType)
	destPath := fmt.Sprintf("%s/deployment.yaml", manifestDir)

	err = s.executeCommand("cp", []string{manifestPath, destPath}, "", logBuffer)
	if err != nil {
		logBuffer.Write([]byte(fmt.Sprintf("Warning: Failed to copy manifests: %v", err)))
	}

	// Add files
	err = s.executeCommand("git", []string{"add", "."}, repoDir, logBuffer)
	if err != nil {
		return err
	}

	// Commit
	commitMessage := step.CommitMessage
	if commitMessage == "" {
		commitMessage = fmt.Sprintf("Deploy %s to %s environment", appName, envType)
	}

	err = s.executeCommand("git", []string{"commit", "-m", commitMessage}, repoDir, logBuffer)
	if err != nil {
		// Ignore error if nothing to commit
		logBuffer.Write([]byte("No changes to commit or commit failed"))
	}

	// Push
	return s.executeCommand("git", []string{"push", "origin", "main"}, repoDir, logBuffer)
}

// executeAnsibleStep executes an ansible playbook step
func (s *Server) executeAnsibleStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	playbookPath := step.Playbook
	if playbookPath == "" {
		playbookPath = "./ansible/post-deploy.yml"
	}

	// Set environment variables for ansible
	extraVars := fmt.Sprintf("app_name=%s env_type=%s", appName, envType)

	return s.executeCommand("ansible-playbook", []string{playbookPath, "-e", extraVars}, "", logBuffer)
}

// executePolicyStep executes a policy validation step
func (s *Server) executePolicyStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	logBuffer.Write([]byte(fmt.Sprintf("Executing policy validation for %s in %s environment", appName, envType)))

	// Simulate policy execution (would integrate with OPA, Gatekeeper, etc.)
	logBuffer.Write([]byte("Policy validation simulated - would require integration with policy engine"))
	time.Sleep(1 * time.Second)

	return nil
}

// provisionResourcesAfterWorkflow provisions all resources for an application after successful workflow execution
func (s *Server) provisionResourcesAfterWorkflow(appName, username string) error {
	fmt.Printf("🔧 Provisioning resources for application: %s\n", appName)

	// Get all resources for the application
	resources, err := s.resourceManager.GetResourcesByApplication(appName)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	if len(resources) == 0 {
		fmt.Printf("No resources found for application: %s\n", appName)
		return nil
	}

	// Provision each resource
	for _, resource := range resources {
		if resource.State == "provisioning" {
			fmt.Printf("📦 Provisioning resource: %s (%s)\n", resource.ResourceName, resource.ResourceType)

			// Provision the resource using the resource manager
			err := s.resourceManager.ProvisionResource(resource.ID, "golden-path-provisioner",
				map[string]interface{}{
					"provisioned_via": "golden_path_workflow",
					"workflow_type": "deploy-app",
				}, username)
			if err != nil {
				fmt.Printf("❌ Failed to provision resource %s: %v\n", resource.ResourceName, err)
				continue
			}

			fmt.Printf("✅ Successfully provisioned resource: %s\n", resource.ResourceName)
		}
	}

	return nil
}

// executeDummyStep executes a dummy workflow step with logging for testing purposes
func (s *Server) executeDummyStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	logBuffer.Write([]byte("INFO: This is a dummy workflow for testing the logging system"))
	logBuffer.Write([]byte(fmt.Sprintf("Executing dummy step '%s' for application: %s", step.Name, appName)))
	logBuffer.Write([]byte(fmt.Sprintf("Environment: %s", envType)))

	// Simulate some processing time with multiple log entries
	logBuffer.Write([]byte("Step 1: Initializing dummy process..."))
	time.Sleep(500 * time.Millisecond)

	logBuffer.Write([]byte("Step 2: Processing dummy data..."))
	logBuffer.Write([]byte("INFO: Dummy workflow is demonstrating log capture functionality"))
	time.Sleep(500 * time.Millisecond)

	logBuffer.Write([]byte("Step 3: Finalizing dummy process..."))
	time.Sleep(300 * time.Millisecond)

	logBuffer.Write([]byte("SUCCESS: Dummy workflow step completed successfully"))
	logBuffer.Write([]byte("This log message confirms the enhanced logging system is working"))

	return nil
}