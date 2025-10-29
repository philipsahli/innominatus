package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"sort"

	"innominatus/internal/admin"
	"innominatus/internal/auth"
	"innominatus/internal/database"
	"innominatus/internal/demo"
	"innominatus/internal/goldenpaths"
	"innominatus/internal/graph"
	"innominatus/internal/health"
	"innominatus/internal/metrics"
	"innominatus/internal/queue"
	"innominatus/internal/resources"
	"innominatus/internal/security"
	"innominatus/internal/teams"
	"innominatus/internal/types"
	"innominatus/internal/users"
	"innominatus/internal/workflow"
	providersdk "innominatus/pkg/sdk"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	sdk "github.com/philipsahli/innominatus-graph/pkg/graph"
	"github.com/rs/zerolog/log"

	"gopkg.in/yaml.v3"
)

// AIService interface for AI assistant functionality
type AIService interface {
	HandleChat(w http.ResponseWriter, r *http.Request)
	HandleGenerateSpec(w http.ResponseWriter, r *http.Request)
	HandleStatus(w http.ResponseWriter, r *http.Request)
	IsEnabled() bool
}

// ProviderRegistry interface for provider management
type ProviderRegistry interface {
	ListProviders() []*providersdk.Provider
	GetProvider(name string) (*providersdk.Provider, error)
	Count() (providers int, provisioners int)
}

// LogBuffer captures command output for workflow step logging
type LogBuffer struct {
	buffer strings.Builder
	stepID *int64
	repo   *database.WorkflowRepository
	mu     sync.Mutex
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
	Step         types.Step
	AppName      string
	EnvType      string
	StepID       *int64
	LogBuffer    *LogBuffer
	WorkflowRepo *database.WorkflowRepository
}

// ProvidersReloadFunc is a callback function type for reloading providers
type ProvidersReloadFunc func() error

type Server struct {
	db                  *database.Database
	workflowRepo        *database.WorkflowRepository
	workflowExecutor    *workflow.WorkflowExecutor
	workflowAnalyzer    *workflow.WorkflowAnalyzer
	workflowQueue       *queue.Queue // Async workflow execution queue
	resourceManager     *resources.Manager
	teamManager         *teams.TeamManager
	sessionManager      auth.ISessionManager
	oidcAuthenticator   *auth.OIDCAuthenticator
	healthChecker       *health.HealthChecker
	rateLimiter         *RateLimiter
	graphAdapter        *graph.Adapter
	wsHub               *GraphWebSocketHub  // WebSocket hub for real-time graph updates
	aiService           AIService           // AI assistant service (optional)
	providerRegistry    ProviderRegistry    // Provider registry (optional)
	providersReloadFunc ProvidersReloadFunc // Callback to reload providers from admin-config.yaml
	swaggerFS           fs.FS               // Optional: embedded swagger files
	webUIFS             fs.FS               // Optional: embedded web-ui files
	loginAttempts       map[string][]time.Time
	loginMutex          sync.Mutex
	// In-memory workflow tracking (when database is not available)
	memoryWorkflows map[int64]*MemoryWorkflowExecution
	workflowCounter int64
	workflowMutex   sync.RWMutex
	// Workflow scheduler for periodic execution
	workflowTicker *time.Ticker
	stopScheduler  chan struct{}
}

// SetAIService sets the AI service for the server
func (s *Server) SetAIService(aiSvc AIService) {
	s.aiService = aiSvc
}

// SetProviderRegistry sets the provider registry for the server
func (s *Server) SetProviderRegistry(registry ProviderRegistry) {
	s.providerRegistry = registry
}

// SetProvidersReloadFunc sets the callback function for reloading providers
func (s *Server) SetProvidersReloadFunc(reloadFunc ProvidersReloadFunc) {
	s.providersReloadFunc = reloadFunc
}

// SetSwaggerFS sets the embedded swagger files filesystem
func (s *Server) SetSwaggerFS(fsys fs.FS) {
	s.swaggerFS = fsys
}

// SetWebUIFS sets the embedded web-ui files filesystem
func (s *Server) SetWebUIFS(fsys fs.FS) {
	s.webUIFS = fsys
}

// GetWebUIFS returns the embedded web-ui files filesystem
func (s *Server) GetWebUIFS() fs.FS {
	return s.webUIFS
}

// MemoryWorkflowExecution represents a workflow execution stored in memory
type MemoryWorkflowExecution struct {
	ID           int64                 `json:"id"`
	AppName      string                `json:"app_name"`
	WorkflowName string                `json:"workflow_name"`
	Status       string                `json:"status"`
	StartedAt    time.Time             `json:"started_at"`
	CompletedAt  *time.Time            `json:"completed_at,omitempty"`
	ErrorMessage *string               `json:"error_message,omitempty"`
	StepCount    int                   `json:"step_count"`
	Steps        []*MemoryWorkflowStep `json:"steps"`
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
	// Initialize OIDC authenticator
	oidcConfig := auth.LoadOIDCConfig()
	oidcAuth, err := auth.NewOIDCAuthenticator(oidcConfig)
	if err != nil && oidcConfig.Enabled {
		fmt.Printf("Warning: Failed to initialize OIDC: %v\n", err)
		fmt.Println("Continuing without OIDC authentication...")
	} else if oidcConfig.Enabled {
		fmt.Println("OIDC authentication enabled")
	}

	healthChecker := health.NewHealthChecker()
	// Register basic health checks
	healthChecker.Register(health.NewAlwaysHealthyChecker("server"))

	// Initialize WebSocket hub for real-time graph updates
	wsHub := NewGraphWebSocketHub()
	go wsHub.Run()

	server := &Server{
		workflowAnalyzer:  workflow.NewWorkflowAnalyzer(),
		teamManager:       teams.NewTeamManager(),
		sessionManager:    auth.NewSessionManager(),
		oidcAuthenticator: oidcAuth,
		healthChecker:     healthChecker,
		wsHub:             wsHub,
		loginAttempts:     make(map[string][]time.Time),
		memoryWorkflows:   make(map[int64]*MemoryWorkflowExecution),
		workflowCounter:   0,
	}

	// Load existing workflow executions from disk
	server.loadWorkflowsFromDisk()

	return server
}

func NewServerWithDB(db *database.Database) *Server {
	// Call with nil admin config for backward compatibility
	return NewServerWithDBAndAdminConfig(db, nil)
}

// NewServerWithDBAndAdminConfig creates a new server with database and admin configuration support
// If adminConfig is provided, enables multi-tier workflow executor with product workflows
func NewServerWithDBAndAdminConfig(db *database.Database, adminConfig interface{}) *Server {
	// Initialize OIDC authenticator
	oidcConfig := auth.LoadOIDCConfig()
	oidcAuth, err := auth.NewOIDCAuthenticator(oidcConfig)
	if err != nil && oidcConfig.Enabled {
		fmt.Printf("Warning: Failed to initialize OIDC: %v\n", err)
		fmt.Println("Continuing without OIDC authentication...")
	} else if oidcConfig.Enabled {
		fmt.Println("OIDC authentication enabled")
	}

	// Create repositories
	workflowRepo := database.NewWorkflowRepository(db)
	resourceRepo := database.NewResourceRepository(db)
	resourceManager := resources.NewManager(resourceRepo)

	// Create workflow executor - use multi-tier if admin config available
	var workflowExecutor *workflow.WorkflowExecutor
	if adminConfig != nil {
		// Multi-tier executor with product workflow support
		if adminCfg, ok := adminConfig.(*admin.AdminConfig); ok && adminCfg != nil {
			policies := workflow.WorkflowPolicies{
				RequiredPlatformWorkflows: adminCfg.WorkflowPolicies.RequiredPlatformWorkflows,
				AllowedProductWorkflows:   adminCfg.WorkflowPolicies.AllowedProductWorkflows,
				WorkflowOverrides: struct {
					Platform bool `yaml:"platform"`
					Product  bool `yaml:"product"`
				}{
					Platform: adminCfg.WorkflowPolicies.WorkflowOverrides.Platform,
					Product:  adminCfg.WorkflowPolicies.WorkflowOverrides.Product,
				},
				MaxWorkflowDuration: adminCfg.WorkflowPolicies.MaxWorkflowDuration,
			}

			workflowsRoot := adminCfg.WorkflowPolicies.WorkflowsRoot
			if workflowsRoot == "" {
				workflowsRoot = "./workflows" // Default
			}

			resolver := workflow.NewWorkflowResolver(workflowsRoot, policies)
			workflowExecutor = workflow.NewMultiTierWorkflowExecutorWithResourceManager(workflowRepo, resolver, resourceManager)
			fmt.Println("✅ Multi-tier workflow executor enabled (platform + product + application workflows)")
		} else {
			// Fall back to single-tier if admin config type assertion fails
			fmt.Println("⚠️  Admin config type mismatch, using single-tier executor")
			workflowExecutor = workflow.NewWorkflowExecutorWithResourceManager(workflowRepo, resourceManager)
		}
	} else {
		// Single-tier executor (backward compatible)
		workflowExecutor = workflow.NewWorkflowExecutorWithResourceManager(workflowRepo, resourceManager)
		fmt.Println("ℹ️  Single-tier workflow executor (use admin-config.yaml for product workflows)")
	}

	// Initialize async workflow queue (5 workers)
	workflowQueue := queue.NewQueue(5, workflowExecutor, db)
	workflowQueue.Start()
	fmt.Println("Async workflow queue initialized with 5 workers")

	// Initialize graph adapter
	graphAdapter, err := graph.NewAdapter(db.DB())
	if err != nil {
		fmt.Printf("Warning: Failed to initialize graph adapter: %v\n", err)
		fmt.Println("Continuing without graph tracking...")
	} else {
		fmt.Println("Graph adapter initialized successfully")
		// Set graph adapter on workflow executor
		workflowExecutor.SetGraphAdapter(graphAdapter)
		// Set graph adapter on resource manager for resource tracking
		resourceManager.SetGraphAdapter(graphAdapter)
	}

	healthChecker := health.NewHealthChecker()
	// Register health checks
	healthChecker.Register(health.NewAlwaysHealthyChecker("server"))
	healthChecker.Register(health.NewDatabaseChecker(db.DB(), 5*time.Second))

	// Initialize WebSocket hub for real-time graph updates
	wsHub := NewGraphWebSocketHub()
	go wsHub.Run()

	server := &Server{
		db:                db,
		workflowRepo:      workflowRepo,
		workflowExecutor:  workflowExecutor,
		workflowAnalyzer:  workflow.NewWorkflowAnalyzer(),
		workflowQueue:     workflowQueue,
		resourceManager:   resourceManager,
		teamManager:       teams.NewTeamManager(),
		sessionManager:    auth.NewDBSessionManager(db),
		oidcAuthenticator: oidcAuth,
		healthChecker:     healthChecker,
		wsHub:             wsHub,
		graphAdapter:      graphAdapter,
		loginAttempts:     make(map[string][]time.Time),
		memoryWorkflows:   make(map[int64]*MemoryWorkflowExecution),
		workflowCounter:   0,
	}

	// Start the workflow scheduler only when database is available
	// DISABLED: Dummy workflow scheduler (triggers test workflow every minute)
	// server.startWorkflowScheduler()

	return server
}

// HandleApplications is the preferred endpoint for application management
func (s *Server) HandleApplications(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleListSpecs(w, r)
	case "POST":
		s.handleDeploySpec(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleApplicationDetail handles operations on a specific application
func (s *Server) HandleApplicationDetail(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/api/applications/"):]

	switch r.Method {
	case "GET":
		s.handleGetSpec(w, r, name)
	case "DELETE":
		s.handleDeleteSpec(w, r, name)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleSpecs is DEPRECATED - use HandleApplications instead
// Kept for backward compatibility
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

// HandleSpecsDeprecated wraps HandleSpecs with deprecation warning
func (s *Server) HandleSpecsDeprecated(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-API-Warn", "Deprecated: Use /api/applications instead of /api/specs")
	w.Header().Set("Deprecation", "true")
	s.HandleSpecs(w, r)
}

// HandleSpecDetailDeprecated wraps HandleSpecDetail with deprecation warning
func (s *Server) HandleSpecDetailDeprecated(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-API-Warn", "Deprecated: Use /api/applications/{name} instead of /api/specs/{name}")
	w.Header().Set("Deprecation", "true")
	s.HandleSpecDetail(w, r)
}

func (s *Server) handleListSpecs(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by authentication middleware)
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var apps []*database.Application
	var err error

	// Admin users can see all specs, regular users only see their team's specs
	if user.IsAdmin() {
		apps, err = s.db.ListApplications()
	} else {
		apps, err = s.db.ListApplicationsByTeam(user.Team)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list applications: %v", err), http.StatusInternalServerError)
		return
	}

	response := make(map[string]interface{})
	for _, app := range apps {
		response[app.Name] = map[string]interface{}{
			"metadata":    app.ScoreSpec.Metadata,
			"containers":  app.ScoreSpec.Containers,
			"resources":   app.ScoreSpec.Resources,
			"environment": app.ScoreSpec.Environment,
			"graph":       graph.BuildGraph(app.ScoreSpec),
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
	err = s.db.AddApplication(name, &spec, user.Team, user.Username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error storing application: %v", err), http.StatusInternalServerError)
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

	app, err := s.db.GetApplication(name)
	if err != nil {
		http.Error(w, "Application not found", http.StatusNotFound)
		return
	}

	// Check if user has access to this spec
	if !user.IsAdmin() && app.Team != user.Team {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	response := map[string]interface{}{
		"metadata":    app.ScoreSpec.Metadata,
		"containers":  app.ScoreSpec.Containers,
		"resources":   app.ScoreSpec.Resources,
		"environment": app.ScoreSpec.Environment,
		"graph":       graph.BuildGraph(app.ScoreSpec),
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

	app, err := s.db.GetApplication(name)
	if err != nil {
		http.Error(w, "Application not found", http.StatusNotFound)
		return
	}

	// Check if user has access to this spec
	if !user.IsAdmin() && app.Team != user.Team {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	err = s.db.DeleteApplication(name)
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

	environments, err := s.db.ListEnvironments()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list environments: %v", err), http.StatusInternalServerError)
		return
	}

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

	// Handle /api/graph/<app>/export pattern
	if len(path) > len("/api/graph/") && path[:len("/api/graph/")] == "/api/graph/" {
		remainder := path[len("/api/graph/"):]

		// Check if it's an export request
		if strings.Contains(remainder, "/export") {
			parts := strings.Split(remainder, "/export")
			if len(parts) == 2 && parts[0] != "" {
				appName := parts[0]
				s.handleGraphExport(w, r, appName)
				return
			}
		}

		// Check if it's a history request
		if strings.Contains(remainder, "/history") {
			parts := strings.Split(remainder, "/history")
			if len(parts) == 2 && parts[0] != "" {
				appName := parts[0]
				s.handleGraphHistory(w, r, appName)
				return
			}
		}

		// Check if it's a WebSocket request
		if strings.Contains(remainder, "/ws") {
			parts := strings.Split(remainder, "/ws")
			if len(parts) == 2 && parts[0] != "" {
				appName := parts[0]
				s.handleGraphWebSocket(w, r, appName)
				return
			}
		}

		// Check if it's an annotations request
		if strings.Contains(remainder, "/annotations") {
			parts := strings.Split(remainder, "/annotations")
			if len(parts) == 2 && parts[0] != "" {
				appName := parts[0]
				s.handleGraphAnnotations(w, r, appName)
				return
			}
		}

		// Check if it's a critical path request
		if strings.Contains(remainder, "/critical-path") {
			parts := strings.Split(remainder, "/critical-path")
			if len(parts) == 2 && parts[0] != "" {
				appName := parts[0]
				s.handleCriticalPath(w, r, appName)
				return
			}
		}

		// Check if it's a metrics request
		if strings.Contains(remainder, "/metrics") {
			parts := strings.Split(remainder, "/metrics")
			if len(parts) == 2 && parts[0] != "" {
				appName := parts[0]
				s.handlePerformanceMetrics(w, r, appName)
				return
			}
		}

		// Check if it's a workflow details request: /api/graph/<app>/workflow/<id>
		if strings.Contains(remainder, "/workflow/") {
			parts := strings.Split(remainder, "/workflow/")
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				appName := parts[0]
				workflowID := parts[1]
				s.handleGraphWorkflowDetails(w, r, appName, workflowID)
				return
			}
		}

		// Regular graph request
		s.handleAppGraph(w, r, remainder)
		return
	}

	// Legacy /api/graph endpoint - return first spec for backward compatibility
	apps, err := s.db.ListApplications()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list applications: %v", err), http.StatusInternalServerError)
		return
	}

	if len(apps) == 1 {
		app := apps[0]
		response := map[string]interface{}{
			"metadata":    app.ScoreSpec.Metadata,
			"containers":  app.ScoreSpec.Containers,
			"resources":   app.ScoreSpec.Resources,
			"environment": app.ScoreSpec.Environment,
			"graph":       graph.BuildGraph(app.ScoreSpec),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
		}
		return
	}

	// Return all specs if multiple exist
	s.handleListSpecs(w, r)
}

// handleGraphExport handles /api/graph/<app>/export requests
func (s *Server) handleGraphExport(w http.ResponseWriter, r *http.Request, appName string) {
	// Get the graph from the database via graph adapter
	if s.graphAdapter == nil {
		http.Error(w, "Graph adapter not initialized", http.StatusInternalServerError)
		return
	}

	sdkGraph, err := s.graphAdapter.GetGraph(appName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Application '%s' not found", appName), http.StatusNotFound)
		return
	}

	// Get format from query parameter (default: mermaid)
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "mermaid"
	}

	switch format {
	case "mermaid":
		exporter := graph.NewMermaidExporter()
		mermaidDiagram, err := exporter.ExportGraph(sdkGraph)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to export graph: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-graph.mmd", appName))
		if _, err := fmt.Fprint(w, mermaidDiagram); err != nil {
			log.Error().Err(err).Msg("Failed to write Mermaid diagram response")
		}

	case "mermaid-simple":
		exporter := graph.NewMermaidExporter()
		mermaidDiagram, err := exporter.ExportGraphSimple(sdkGraph)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to export graph: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-graph-simple.mmd", appName))
		if _, err := fmt.Fprint(w, mermaidDiagram); err != nil {
			log.Error().Err(err).Msg("Failed to write Mermaid diagram response")
		}

	case "svg", "png", "dot":
		// Use existing graph adapter export functionality
		data, err := s.graphAdapter.ExportGraph(appName, format)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to export graph as %s: %v", format, err), http.StatusInternalServerError)
			return
		}

		// Set appropriate content type
		contentType := map[string]string{
			"svg": "image/svg+xml",
			"png": "image/png",
			"dot": "text/plain",
		}[format]

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-graph.%s", appName, format))
		if _, err := w.Write(data); err != nil {
			log.Error().Err(err).Msg("Failed to write graph data response")
		}

	case "json":
		// Export as JSON (same as regular graph endpoint)
		response := convertSDKGraphToFrontend(sdkGraph)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-graph.json", appName))
		if err := json.NewEncoder(w).Encode(response); err != nil {
			fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
		}

	default:
		http.Error(w, fmt.Sprintf("Unsupported format: %s. Supported formats: mermaid, mermaid-simple, svg, png, dot, json", format), http.StatusBadRequest)
	}
}

// handleAppGraph handles /api/graph/<app> requests with enhanced graph data
func (s *Server) handleAppGraph(w http.ResponseWriter, r *http.Request, appName string) {
	// Get the graph from the database via graph adapter
	if s.graphAdapter == nil {
		http.Error(w, "Graph adapter not initialized", http.StatusInternalServerError)
		return
	}

	sdkGraph, err := s.graphAdapter.GetGraph(appName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Application '%s' not found", appName), http.StatusNotFound)
		return
	}

	// Convert SDK graph format to frontend-compatible format
	response := convertSDKGraphToFrontend(sdkGraph)

	// Return the graph data
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// handleGraphHistory handles /api/graph/<app>/history requests
// Returns historical snapshots of graph states based on workflow executions
func (s *Server) handleGraphHistory(w http.ResponseWriter, r *http.Request, appName string) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		var parsedLimit int
		if _, err := fmt.Sscanf(limitStr, "%d", &parsedLimit); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Use workflow repository to get execution history
	executions, err := s.workflowRepo.ListWorkflowExecutions(appName, "", "", limit, 0)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query history: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	snapshots := make([]map[string]interface{}, 0, len(executions))
	for _, exec := range executions {
		snapshot := map[string]interface{}{
			"id":              exec.ID,
			"workflow_name":   exec.WorkflowName,
			"status":          exec.Status,
			"started_at":      exec.StartedAt,
			"completed_at":    exec.CompletedAt,
			"total_steps":     exec.TotalSteps,
			"completed_steps": exec.CompletedSteps,
			"failed_steps":    exec.FailedSteps,
		}

		// Calculate duration if completed
		if exec.CompletedAt != nil {
			duration := exec.CompletedAt.Sub(exec.StartedAt)
			snapshot["duration_seconds"] = duration.Seconds()
		}

		snapshots = append(snapshots, snapshot)
	}

	response := map[string]interface{}{
		"application": appName,
		"snapshots":   snapshots,
		"count":       len(snapshots),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// convertSDKGraphToFrontend converts the SDK Graph format to frontend JSON format
func convertSDKGraphToFrontend(sdkGraph *sdk.Graph) map[string]interface{} {
	// Convert nodes map to array with enriched information
	nodes := make([]map[string]interface{}, 0, len(sdkGraph.Nodes))
	for _, node := range sdkGraph.Nodes {
		nodeData := map[string]interface{}{
			"id":          node.ID,
			"name":        node.Name,
			"type":        string(node.Type),
			"status":      string(node.State), // Convert NodeState to string
			"description": node.Description,
			"metadata":    node.Properties,
			"created_at":  node.CreatedAt,
			"updated_at":  node.UpdatedAt,
		}

		// Enrich with step number if available
		if stepNum, ok := node.Properties["step_number"].(int); ok {
			nodeData["step_number"] = stepNum
		} else if stepNum, ok := node.Properties["step_number"].(float64); ok {
			nodeData["step_number"] = int(stepNum)
		}

		// Enrich with total steps if available
		if totalSteps, ok := node.Properties["total_steps"].(int); ok {
			nodeData["total_steps"] = totalSteps
		} else if totalSteps, ok := node.Properties["total_steps"].(float64); ok {
			nodeData["total_steps"] = int(totalSteps)
		}

		// Enrich with workflow execution ID if available
		if wfID, ok := node.Properties["workflow_execution_id"]; ok {
			nodeData["workflow_id"] = wfID
		}

		// Calculate duration if both timestamps exist
		if !node.CreatedAt.IsZero() && !node.UpdatedAt.IsZero() {
			duration := node.UpdatedAt.Sub(node.CreatedAt)
			nodeData["duration_ms"] = duration.Milliseconds()
		}

		nodes = append(nodes, nodeData)
	}

	// Sort nodes for consistent ordering:
	// 1. By node type priority (workflow > step > resource)
	// 2. By creation timestamp (oldest first)
	// 3. By node ID (tie-breaker)
	sort.Slice(nodes, func(i, j int) bool {
		// Type priority: workflow=1, step=2, resource=3, other=4
		typePriority := map[string]int{
			"workflow": 1,
			"step":     2,
			"resource": 3,
		}

		typeI := nodes[i]["type"].(string)
		typeJ := nodes[j]["type"].(string)
		priorityI := typePriority[typeI]
		priorityJ := typePriority[typeJ]
		if priorityI == 0 {
			priorityI = 4
		}
		if priorityJ == 0 {
			priorityJ = 4
		}

		if priorityI != priorityJ {
			return priorityI < priorityJ
		}

		// Sort by creation timestamp
		createdI, okI := nodes[i]["created_at"].(time.Time)
		createdJ, okJ := nodes[j]["created_at"].(time.Time)

		if okI && okJ && !createdI.IsZero() && !createdJ.IsZero() {
			if !createdI.Equal(createdJ) {
				return createdI.Before(createdJ)
			}
		}

		// Final tie-breaker: node ID
		idI := nodes[i]["id"].(string)
		idJ := nodes[j]["id"].(string)
		return idI < idJ
	})

	// Add execution order based on sorted position
	for idx := range nodes {
		nodes[idx]["execution_order"] = idx + 1
	}

	// Convert edges map to array
	edges := make([]map[string]interface{}, 0, len(sdkGraph.Edges))
	for _, edge := range sdkGraph.Edges {
		edges = append(edges, map[string]interface{}{
			"id":          edge.ID,
			"source_id":   edge.FromNodeID, // Map from_node_id to source_id
			"target_id":   edge.ToNodeID,   // Map to_node_id to target_id
			"type":        string(edge.Type),
			"description": edge.Description,
			"metadata":    edge.Properties,
		})
	}

	// Sort edges for consistent ordering (by source, then target)
	sort.Slice(edges, func(i, j int) bool {
		sourceI := edges[i]["source_id"].(string)
		sourceJ := edges[j]["source_id"].(string)
		if sourceI != sourceJ {
			return sourceI < sourceJ
		}
		targetI := edges[i]["target_id"].(string)
		targetJ := edges[j]["target_id"].(string)
		return targetI < targetJ
	})

	return map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
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

	// Check for retry sub-route: /api/workflows/{id}/retry
	if strings.HasSuffix(path, "/retry") {
		if r.Method == "POST" {
			s.handleRetryWorkflow(w, r, workflowID)
			return
		}
		http.Error(w, "Method not allowed - use POST for retry", http.StatusMethodNotAllowed)
		return
	}

	switch r.Method {
	case "GET":
		s.handleGetWorkflow(w, r, workflowID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// PaginatedWorkflowsResponse represents a paginated list of workflow executions
type PaginatedWorkflowsResponse struct {
	Data       []*database.WorkflowExecutionSummary `json:"data"`
	Total      int64                                `json:"total"`
	Page       int                                  `json:"page"`
	PageSize   int                                  `json:"page_size"`
	TotalPages int                                  `json:"total_pages"`
}

func (s *Server) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	appName := r.URL.Query().Get("app")
	searchTerm := r.URL.Query().Get("search")
	statusFilter := r.URL.Query().Get("status")

	limit := 50 // default limit
	page := 1   // default page

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || l != 1 || limit > 100 {
			limit = 50
		}
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := fmt.Sscanf(pageStr, "%d", &page); err != nil || p != 1 || page < 1 {
			page = 1
		}
	}

	// Calculate offset from page number
	offset := (page - 1) * limit

	// Get total count matching filters
	total, err := s.workflowExecutor.CountWorkflowExecutions(appName, searchTerm, statusFilter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to count workflows: %v", err), http.StatusInternalServerError)
		return
	}

	// Get paginated workflows
	workflows, err := s.workflowExecutor.ListWorkflowExecutions(appName, searchTerm, statusFilter, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list workflows: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate total pages
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	// Build paginated response
	response := PaginatedWorkflowsResponse{
		Data:       workflows,
		Total:      total,
		Page:       page,
		PageSize:   limit,
		TotalPages: totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
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

// handleRetryWorkflow handles retrying a failed workflow execution from the first failed step
// @Summary Retry a failed workflow execution
// @Description Retry a failed workflow execution from the first failed step with an updated workflow specification
// @Tags workflows
// @Accept json
// @Produce json
// @Param id path int true "Workflow Execution ID"
// @Param workflow body types.Workflow true "Updated workflow specification to retry with"
// @Success 200 {object} map[string]interface{} "Retry successful"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 404 {object} map[string]string "Workflow execution not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/workflows/{id}/retry [post]
func (s *Server) handleRetryWorkflow(w http.ResponseWriter, r *http.Request, workflowID int64) {
	// Get the parent workflow execution to retrieve app name and workflow name
	parentExec, err := s.workflowExecutor.GetWorkflowExecution(workflowID)
	if err != nil {
		if err.Error() == "workflow execution not found" {
			http.Error(w, "Workflow execution not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get workflow execution: %v", err), http.StatusInternalServerError)
		return
	}

	// Try to parse workflow from request body (optional)
	// If body is empty, reconstruct workflow from database
	var workflowMap map[string]interface{}
	bodyBytes, _ := io.ReadAll(r.Body)

	if len(bodyBytes) > 0 {
		// User provided updated workflow specification
		if err := json.Unmarshal(bodyBytes, &workflowMap); err != nil {
			http.Error(w, fmt.Sprintf("Invalid workflow specification: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// Reconstruct workflow from database (automatic retry)
		reconstructed, err := s.workflowExecutor.GetRepository().ReconstructWorkflowFromExecution(workflowID)
		if err != nil {
			// Check if error is due to missing steps (old workflow executions)
			if strings.Contains(err.Error(), "no steps found") {
				http.Error(w, "This workflow execution has no stored step configuration and cannot be retried automatically. This may be an older workflow execution created before step configuration storage was implemented.", http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("Failed to reconstruct workflow: %v", err), http.StatusInternalServerError)
			return
		}
		workflowMap = reconstructed
	}

	// Convert map to types.Workflow
	workflowJSON, err := json.Marshal(workflowMap)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal workflow: %v", err), http.StatusInternalServerError)
		return
	}

	var workflow types.Workflow
	if err := json.Unmarshal(workflowJSON, &workflow); err != nil {
		http.Error(w, fmt.Sprintf("Failed to unmarshal workflow: %v", err), http.StatusInternalServerError)
		return
	}

	// Execute retry from failed step
	err = s.workflowExecutor.RetryWorkflowFromFailedStep(
		parentExec.ApplicationName,
		parentExec.WorkflowName,
		workflow,
		workflowID,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retry workflow: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success":             true,
		"message":             "Workflow retry completed successfully",
		"parent_execution_id": workflowID,
		"app_name":            parentExec.ApplicationName,
		"workflow_name":       parentExec.WorkflowName,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
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
	//nolint:staticcheck // Simple if statement is clearer for health status check - QF1003
	if healthResponse.Status == health.StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	} else if healthResponse.Status == health.StatusDegraded {
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(healthResponse)
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
	_ = json.NewEncoder(w).Encode(readinessResponse)
}

// HandleMetrics returns Prometheus-format metrics
func (s *Server) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	metricsData := metrics.GetGlobal().Export()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(metricsData))
}

// HandleAuthConfig returns authentication configuration for the frontend
func (s *Server) HandleAuthConfig(w http.ResponseWriter, r *http.Request) {
	oidcEnabled := s.oidcAuthenticator != nil && s.oidcAuthenticator.IsEnabled()

	config := map[string]interface{}{
		"oidc_enabled":       oidcEnabled,
		"oidc_provider_name": "Keycloak",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(config)
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
	if err := os.MkdirAll("data", 0750); err != nil {
		fmt.Printf("Warning: Failed to create data directory: %v\n", err)
		return
	}

	// Marshal workflow data
	data := struct {
		Workflows       map[int64]*MemoryWorkflowExecution `json:"workflows"`
		WorkflowCounter int64                              `json:"workflow_counter"`
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
	if err := os.WriteFile("data/workflows.json", jsonData, 0600); err != nil {
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
		WorkflowCounter int64                              `json:"workflow_counter"`
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
			"name":        result.Name,
			"url":         getComponentURL(result.Name, result.Host),
			"status":      result.Healthy,
			"credentials": getComponentCredentials(result.Name, env),
			"health":      result.Status,
			"latency_ms":  result.Latency.Milliseconds(),
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
	// Minio uses a separate console URL
	if name == "minio" {
		return "http://minio-console.localtest.me"
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

// HandleDemoReset handles POST /api/admin/demo/reset - Reset database to clean state
func (s *Server) HandleDemoReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Require database connection
	if s.workflowRepo == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// Parse request body for confirmation
	var reqBody struct {
		Confirm bool `json:"confirm"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		// If no body provided, treat as not confirmed
		reqBody.Confirm = false
	}

	if !reqBody.Confirm {
		http.Error(w, "Confirmation required: send {\"confirm\": true} in request body", http.StatusBadRequest)
		return
	}

	// Truncate all tables
	tableCount, err := s.db.TruncateAllTables()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset database: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":          true,
		"tables_truncated": tableCount,
		"tasks_stopped":    0, // Future enhancement: stop queue tasks
		"message":          "Database reset complete",
		"timestamp":        time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// HandleAdminConfig handles GET /api/admin/config - Returns admin configuration with masked sensitive data
func (s *Server) HandleAdminConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load admin config: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to masked JSON (passwords/tokens as ****)
	maskedConfig := adminConfig.ToMaskedJSON()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(maskedConfig); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// HandleAdminReload handles POST /api/admin/reload - Reloads admin-config.yaml and providers
func (s *Server) HandleAdminReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// This endpoint requires provider registry and reload function
	if s.providerRegistry == nil || s.providersReloadFunc == nil {
		http.Error(w, "Provider reload not configured", http.StatusServiceUnavailable)
		return
	}

	// Verify admin config exists and is loadable before attempting reload
	_, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load admin config: %v", err), http.StatusBadRequest)
		return
	}

	// Trigger provider reload
	if err := s.providersReloadFunc(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reload providers: %v", err), http.StatusInternalServerError)
		return
	}

	// Get updated provider counts
	providerCount, provisionerCount := s.providerRegistry.Count()

	response := map[string]interface{}{
		"success":      true,
		"message":      "Providers reloaded successfully",
		"providers":    providerCount,
		"provisioners": provisionerCount,
		"timestamp":    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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
	var apps []*database.Application
	var err error
	if user.IsAdmin() {
		apps, err = s.db.ListApplications()
	} else {
		apps, err = s.db.ListApplicationsByTeam(user.Team)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to count applications: %v", err), http.StatusInternalServerError)
		return
	}
	applicationsCount := len(apps)

	// Count active (running) workflows only
	var workflowsCount int
	if s.workflowExecutor != nil {
		// Use database workflow count - filter by "running" status only
		workflows, err := s.workflowExecutor.ListWorkflowExecutions("", "", "running", 0, 0)
		if err == nil {
			workflowsCount = len(workflows)
		}
	} else {
		// Use memory workflow count - count only running workflows
		s.workflowMutex.RLock()
		for _, wf := range s.memoryWorkflows {
			if wf.Status == "running" {
				workflowsCount++
			}
		}
		s.workflowMutex.RUnlock()
	}

	// Count resources across all specs
	resourcesCount := 0
	for _, app := range apps {
		if app.ScoreSpec.Resources != nil {
			resourcesCount += len(app.ScoreSpec.Resources)
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
//
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
	// Validate file path to prevent path traversal
	cleanPath, err := security.SafeFilePath(filePath, "./workflows", "./data")
	if err != nil {
		return nil, fmt.Errorf("invalid workflow path: %w", err)
	}

	data, err := os.ReadFile(cleanPath) // #nosec G304 - path validated above
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
	defer func() { _ = r.Body.Close() }()

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
	defer func() { _ = r.Body.Close() }()

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
			"totalTime":   analysis.ExecutionPlan.TotalTime.String(),
			"phases":      len(analysis.ExecutionPlan.Phases),
			"maxParallel": analysis.ExecutionPlan.MaxParallel,
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
	app, err := s.db.GetApplication(appName)
	if err != nil {
		http.Error(w, "Application not found", http.StatusNotFound)
		return
	}

	// Check if user has access to this application
	if !user.IsAdmin() && app.Team != user.Team {
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

	// Also remove from database (spec records)
	err = s.db.DeleteApplication(appName)
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
	app, err := s.db.GetApplication(appName)
	if err != nil {
		http.Error(w, "Application not found", http.StatusNotFound)
		return
	}

	// Check if user has access to this application
	if !user.IsAdmin() && app.Team != user.Team {
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

// HandleGoldenPaths handles listing and retrieving golden paths
func (s *Server) HandleGoldenPaths(w http.ResponseWriter, r *http.Request) {
	// Extract path to check if it's a specific golden path request
	path := strings.TrimPrefix(r.URL.Path, "/api/golden-paths")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		// List all golden paths
		s.handleListGoldenPaths(w, r)
	} else {
		// Get specific golden path metadata
		goldenPathName := strings.TrimSuffix(path, "/")
		s.handleGetGoldenPath(w, r, goldenPathName)
	}
}

func (s *Server) handleListGoldenPaths(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config, err := goldenpaths.LoadGoldenPaths()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load golden paths: %v", err), http.StatusInternalServerError)
		return
	}

	paths := config.ListPaths()

	// Build response with metadata for each path
	response := make(map[string]interface{})
	for _, pathName := range paths {
		metadata, err := config.GetMetadata(pathName)
		if err != nil {
			continue // Skip paths that fail to load
		}

		pathInfo := map[string]interface{}{
			"description":        metadata.Description,
			"category":           metadata.Category,
			"tags":               metadata.Tags,
			"estimated_duration": metadata.EstimatedDuration,
			"workflow_file":      metadata.WorkflowFile,
		}

		// Add parameter schemas if available
		if len(metadata.Parameters) > 0 {
			pathInfo["parameters"] = metadata.Parameters
		}

		response[pathName] = pathInfo
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) handleGetGoldenPath(w http.ResponseWriter, r *http.Request, pathName string) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config, err := goldenpaths.LoadGoldenPaths()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load golden paths: %v", err), http.StatusInternalServerError)
		return
	}

	metadata, err := config.GetMetadata(pathName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Golden path '%s' not found", pathName), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"name":               pathName,
		"description":        metadata.Description,
		"category":           metadata.Category,
		"tags":               metadata.Tags,
		"estimated_duration": metadata.EstimatedDuration,
		"workflow_file":      metadata.WorkflowFile,
	}

	// Add parameter schemas if available
	if len(metadata.Parameters) > 0 {
		response["parameters"] = metadata.Parameters
	}

	// Add deprecated fields for backward compatibility
	if len(metadata.RequiredParams) > 0 {
		response["required_params"] = metadata.RequiredParams
	}
	if len(metadata.OptionalParams) > 0 {
		response["optional_params"] = metadata.OptionalParams
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

	// Extract golden path parameters from query string (param.KEY=value)
	goldenPathParams := make(map[string]string)
	for key, values := range r.URL.Query() {
		if strings.HasPrefix(key, "param.") {
			paramName := strings.TrimPrefix(key, "param.")
			if len(values) > 0 {
				goldenPathParams[paramName] = values[0]
			}
		}
	}

	// Log parameters if any were provided
	if len(goldenPathParams) > 0 {
		fmt.Printf("   📋 Golden path parameters: %v\n", goldenPathParams)
	}

	// Load golden path workflow
	workflowFile := fmt.Sprintf("./workflows/%s.yaml", goldenPathName)

	// Validate workflow path to prevent path traversal
	cleanPath, err := security.SafeFilePath(workflowFile, "./workflows")
	if err != nil {
		http.Error(w, "Invalid workflow path", http.StatusBadRequest)
		return
	}

	workflowData, err := os.ReadFile(cleanPath) // #nosec G304 - path validated above
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
	err = s.db.AddApplication(spec.Metadata.Name, &spec, user.Team, user.Username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error storing application: %v", err), http.StatusInternalServerError)
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

	// TEMPORARY FIX: Force synchronous execution to ensure resources are provisioned after workflow completes
	// Execute workflow synchronously (disabled async queue for golden paths)
	var taskID string
	_ = taskID // Unused for now
	if s.workflowExecutor != nil {
		// Execute workflow synchronously with golden path parameters
		err = s.workflowExecutor.ExecuteWorkflowWithName(spec.Metadata.Name, fmt.Sprintf("golden-path-%s", goldenPathName), workflow, goldenPathParams)
		if err != nil {
			http.Error(w, fmt.Sprintf("Workflow execution failed: %v", err), http.StatusInternalServerError)
			return
		}
	} else if s.workflowQueue != nil {
		// Fallback: Enqueue workflow for async execution with queue (not recommended for golden paths)
		metadata := map[string]interface{}{
			"user":        user.Username,
			"golden_path": goldenPathName,
			"source":      "api",
			"parameters":  goldenPathParams,
		}
		taskID, err = s.workflowQueue.Enqueue(spec.Metadata.Name, fmt.Sprintf("golden-path-%s", goldenPathName), workflow, metadata)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to enqueue workflow: %v", err), http.StatusInternalServerError)
			return
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
		"message":     fmt.Sprintf("Golden path '%s' enqueued successfully for application '%s'", goldenPathName, spec.Metadata.Name),
		"application": spec.Metadata.Name,
		"golden_path": goldenPathName,
		"task_id":     taskID,
		"status":      "enqueued",
	}

	if taskID != "" {
		response["message"] = fmt.Sprintf("Golden path '%s' enqueued successfully for application '%s'", goldenPathName, spec.Metadata.Name)
	} else {
		response["message"] = fmt.Sprintf("Golden path '%s' executed successfully for application '%s'", goldenPathName, spec.Metadata.Name)
		response["status"] = "completed"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// executeBasicGoldenPathWorkflow executes a workflow without database tracking (fallback)
func (s *Server) executeBasicGoldenPathWorkflow(workflow *types.Workflow, spec *types.ScoreSpec, username string) error {
	fmt.Printf("📋 Executing basic workflow with %d steps for %s\n", len(workflow.Steps), spec.Metadata.Name)

	for i, step := range workflow.Steps {
		fmt.Printf("🔄 Step %d/%d: %s (%s)\n", i+1, len(workflow.Steps), step.Name, step.Type)

		// For basic workflow, create minimal context without database tracking
		stepContext := &StepExecutionContext{
			StepID:       nil, // No database tracking for basic workflow
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
	step.OutputDir = replacer.Replace(step.OutputDir)
	step.WorkingDir = replacer.Replace(step.WorkingDir)

	// Substitute in variables map
	if step.Variables != nil {
		for key, value := range step.Variables {
			if strValue, ok := value.(string); ok {
				step.Variables[key] = replacer.Replace(strValue)
			}
		}
	}
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
	if _, err := fmt.Fprintf(logBuffer, "Starting step: %s (type: %s)", step.Name, step.Type); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write log: %v\n", err)
	}

	// Execute the step based on its type
	switch step.Type {
	case "terraform-generate":
		fmt.Printf("   📝 Executing Terraform Generate step: %s\n", step.Name)
		return s.executeTerraformGenerateStep(step, appName, envType, logBuffer)
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
		if _, err := fmt.Fprintf(logBuffer, "Warning: Unknown step type '%s', skipping execution", step.Type); err != nil {
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

// executeTerraformGenerateStep generates Terraform code from Score resources
func (s *Server) executeTerraformGenerateStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	_, _ = fmt.Fprintf(logBuffer, "Generating Terraform code for: %s", step.Name)

	// Get output directory from step (supports variable substitution)
	outputDir := step.OutputDir
	if outputDir == "" {
		outputDir = fmt.Sprintf("workspaces/%s/terraform", appName)
	}

	// Get resource type to generate
	resourceType := step.Resource
	if resourceType == "" && step.Config != nil {
		if rt, ok := step.Config["resource"].(string); ok {
			resourceType = rt
		}
	}

	if resourceType == "" {
		errMsg := "terraform-generate requires 'resource' field (e.g., 's3', 'postgres')"
		_, _ = logBuffer.Write([]byte(errMsg))
		return fmt.Errorf("terraform-generate requires 'resource' field (e.g., 's3', 'postgres')")
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		errMsg := fmt.Sprintf("Failed to create output directory: %v", err)
		_, _ = logBuffer.Write([]byte(errMsg))
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	_, _ = fmt.Fprintf(logBuffer, "Output directory: %s", outputDir)
	_, _ = fmt.Fprintf(logBuffer, "Resource type: %s", resourceType)

	// Generate Terraform code based on resource type
	switch resourceType {
	case "s3", "minio-s3-bucket":
		return s.generateS3BucketTerraform(outputDir, appName, step, logBuffer)
	case "postgres", "postgresql":
		errMsg := "PostgreSQL Terraform generation not yet implemented"
		_, _ = logBuffer.Write([]byte(errMsg))
		return fmt.Errorf("PostgreSQL Terraform generation not yet implemented")
	default:
		errMsg := fmt.Sprintf("Unsupported resource type for terraform generation: %s", resourceType)
		_, _ = logBuffer.Write([]byte(errMsg))
		return fmt.Errorf("unsupported resource type for terraform generation: %s", resourceType)
	}
}

// generateS3BucketTerraform generates Terraform code for Minio S3 bucket
func (s *Server) generateS3BucketTerraform(outputDir, appName string, step types.Step, logBuffer *LogBuffer) error {
	_, _ = logBuffer.Write([]byte("Generating Minio S3 bucket Terraform configuration"))

	// Get variables from step
	variables := step.Variables
	if variables == nil {
		variables = make(map[string]interface{})
	}

	// Extract Minio configuration
	bucketName, _ := variables["bucket_name"].(string)
	if bucketName == "" {
		bucketName = fmt.Sprintf("%s-storage", appName)
	}

	minioEndpoint, _ := variables["minio_endpoint"].(string)
	if minioEndpoint == "" {
		minioEndpoint = "http://minio.minio-system.svc.cluster.local:9000"
	}

	minioUser, _ := variables["minio_user"].(string)
	if minioUser == "" {
		minioUser = "minioadmin"
	}

	minioPassword, _ := variables["minio_password"].(string)
	if minioPassword == "" {
		minioPassword = "minioadmin"
	}

	// Strip protocol from endpoint for Minio provider (it expects just host:port)
	minioServer := strings.TrimPrefix(minioEndpoint, "http://")
	minioServer = strings.TrimPrefix(minioServer, "https://")

	// Generate main.tf
	mainTf := fmt.Sprintf(`terraform {
  required_providers {
    minio = {
      source  = "aminueza/minio"
      version = "~> 2.0"
    }
  }
}

provider "minio" {
  minio_server   = "%s"
  minio_user     = "%s"
  minio_password = "%s"
  minio_ssl      = false
}

resource "minio_s3_bucket" "bucket" {
  bucket = "%s"
  acl    = "private"
}

output "bucket_name" {
  value = minio_s3_bucket.bucket.bucket
}

output "minio_url" {
  value = "%s"
}

output "endpoint" {
  value = "%s/${minio_s3_bucket.bucket.bucket}"
}

output "bucket_arn" {
  value = "arn:aws:s3:::${minio_s3_bucket.bucket.bucket}"
}
`, minioServer, minioUser, minioPassword, bucketName, minioEndpoint, minioEndpoint)

	// Write main.tf
	mainTfPath := filepath.Join(outputDir, "main.tf")
	if err := os.WriteFile(mainTfPath, []byte(mainTf), 0600); err != nil {
		errMsg := fmt.Sprintf("Failed to write main.tf: %v", err)
		_, _ = logBuffer.Write([]byte(errMsg))
		return fmt.Errorf("failed to write main.tf: %w", err)
	}

	_, _ = fmt.Fprintf(logBuffer, "Generated Terraform configuration: %s", mainTfPath)
	_, _ = fmt.Fprintf(logBuffer, "Bucket name: %s", bucketName)
	_, _ = fmt.Fprintf(logBuffer, "Minio endpoint: %s", minioEndpoint)

	return nil
}

// executeTerraformStep executes a terraform workflow step
func (s *Server) executeTerraformStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	// Use workingDir from step config if provided, otherwise use default
	workDir := step.WorkingDir
	if workDir == "" {
		workDir = fmt.Sprintf("./terraform/%s-%s", appName, envType)
	}

	// Create workspace directory if it doesn't exist
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		err = os.MkdirAll(workDir, 0750)
		if err != nil {
			_, _ = fmt.Fprintf(logBuffer, "Failed to create workspace directory: %v", err)
			return err
		}
	}

	// Copy terraform files from step.Path to workspace
	if step.Path != "" {
		_, _ = fmt.Fprintf(logBuffer, "Copying terraform files from %s to %s", step.Path, workDir)
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
	_, _ = fmt.Fprintf(logBuffer, "Creating namespace: %s", namespace)
	err := s.executeCommand("kubectl", []string{"create", "namespace", namespace}, "", logBuffer)
	if err != nil {
		// Namespace might already exist, which is fine
		_, _ = logBuffer.Write([]byte("Namespace may already exist, continuing..."))
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

	err = os.WriteFile(manifestPath, []byte(manifest), 0600)
	if err != nil {
		_, _ = fmt.Fprintf(logBuffer, "Failed to write manifest file: %v", err)
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

	_, _ = fmt.Fprintf(logBuffer, "Creating Gitea repository: %s", repoName)

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		_, _ = fmt.Fprintf(logBuffer, "Failed to load admin config: %v", err)
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
		_, _ = fmt.Fprintf(logBuffer, "Failed to create repository: %v", err)
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	// If org doesn't exist (404), create under user account instead
	if resp.StatusCode == 404 {
		_, _ = fmt.Fprintf(logBuffer, "Organization '%s' not found, creating repository under user account", owner)
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
			_, _ = fmt.Fprintf(logBuffer, "Failed to create repository: %v", err)
			return fmt.Errorf("failed to create repository: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, _ = io.ReadAll(resp.Body)
	}

	if resp.StatusCode == 409 {
		_, _ = logBuffer.Write([]byte("Repository already exists, continuing..."))
	} else if resp.StatusCode != 201 {
		errMsg := fmt.Sprintf("Failed to create repository, status %d: %s", resp.StatusCode, string(body))
		_, _ = logBuffer.Write([]byte(errMsg))
		return fmt.Errorf("failed to create repository, status %d: %s", resp.StatusCode, string(body))
	} else {
		_, _ = logBuffer.Write([]byte("Repository created successfully"))
	}

	// Clone repository locally for manifest commits
	repoDir := fmt.Sprintf("/tmp/%s-%s-repo", appName, envType)
	repoURL := fmt.Sprintf("%s/%s/%s.git", adminConfig.Gitea.URL, owner, repoName)

	// Remove existing directory if present
	_ = s.executeCommand("rm", []string{"-rf", repoDir}, "", logBuffer)

	// Clone repository
	err = s.executeCommand("git", []string{"clone", repoURL, repoDir}, "", logBuffer)
	if err != nil {
		_, _ = fmt.Fprintf(logBuffer, "Failed to clone repository: %v", err)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	_, _ = fmt.Fprintf(logBuffer, "Repository cloned to: %s", repoDir)
	return nil
}

// executeArgoCDStep executes an ArgoCD application creation step
func (s *Server) executeArgoCDStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	appNameArgo := step.AppName
	if appNameArgo == "" {
		appNameArgo = fmt.Sprintf("%s-%s", appName, envType)
	}

	_, _ = fmt.Fprintf(logBuffer, "Creating ArgoCD application: %s", appNameArgo)

	// Load admin configuration to get Gitea URL
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		_, _ = fmt.Fprintf(logBuffer, "Failed to load admin config: %v", err)
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
	err = os.WriteFile(manifestPath, []byte(manifest), 0600)
	if err != nil {
		_, _ = fmt.Fprintf(logBuffer, "Failed to write ArgoCD manifest: %v", err)
		return err
	}

	return s.executeCommand("kubectl", []string{"apply", "-f", manifestPath}, "", logBuffer)
}

// executeGitCommitStep executes a git commit and push step
func (s *Server) executeGitCommitStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	repoDir := fmt.Sprintf("/tmp/%s-%s-repo", appName, envType)

	_, _ = fmt.Fprintf(logBuffer, "Committing manifests to repository in %s", repoDir)

	// Create manifests directory if it doesn't exist
	manifestDir := fmt.Sprintf("%s/%s", repoDir, step.ManifestPath)
	if step.ManifestPath == "" {
		manifestDir = fmt.Sprintf("%s/manifests", repoDir)
	}

	err := os.MkdirAll(manifestDir, 0750)
	if err != nil {
		_, _ = fmt.Fprintf(logBuffer, "Failed to create manifest directory: %v", err)
		return err
	}

	// Copy kubernetes manifests to repository
	manifestPath := fmt.Sprintf("/tmp/%s-%s-manifests.yaml", appName, envType)
	destPath := fmt.Sprintf("%s/deployment.yaml", manifestDir)

	err = s.executeCommand("cp", []string{manifestPath, destPath}, "", logBuffer)
	if err != nil {
		_, _ = fmt.Fprintf(logBuffer, "Warning: Failed to copy manifests: %v", err)
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
		_, _ = logBuffer.Write([]byte("No changes to commit or commit failed"))
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
	_, _ = fmt.Fprintf(logBuffer, "Executing policy validation for %s in %s environment", appName, envType)

	// Simulate policy execution (would integrate with OPA, Gatekeeper, etc.)
	_, _ = logBuffer.Write([]byte("Policy validation simulated - would require integration with policy engine"))
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
					"workflow_type":   "deploy-app",
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

// HandleGetProfile returns the current user's profile information
func (s *Server) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(contextKeyUser).(*users.User)
	if !ok || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	profile := map[string]interface{}{
		"username": user.Username,
		"team":     user.Team,
		"role":     user.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		http.Error(w, "Failed to encode profile", http.StatusInternalServerError)
	}
}

// HandleGetAPIKeys returns the user's API keys with masked key values
func (s *Server) HandleGetAPIKeys(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(contextKeyUser).(*users.User)
	if !ok || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user exists in users.yaml (local user) or is OIDC user
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	_, err = store.GetUser(user.Username)
	isOIDCUser := err != nil // User not found in yaml = OIDC user

	var keys []users.APIKey

	if isOIDCUser && s.db != nil {
		// Get API keys from database for OIDC user
		dbKeys, err := s.db.GetAPIKeys(user.Username)
		if err != nil {
			http.Error(w, "Failed to retrieve API keys", http.StatusInternalServerError)
			return
		}

		// Convert database records to users.APIKey format
		for _, dbKey := range dbKeys {
			lastUsed := time.Time{}
			if dbKey.LastUsedAt != nil {
				lastUsed = *dbKey.LastUsedAt
			}
			keys = append(keys, users.APIKey{
				Key:        dbKey.KeyHash, // Will be masked anyway
				Name:       dbKey.KeyName,
				CreatedAt:  dbKey.CreatedAt,
				LastUsedAt: lastUsed,
				ExpiresAt:  dbKey.ExpiresAt,
			})
		}
	} else {
		// Get API keys from users.yaml for local user
		keys, err = store.ListAPIKeys(user.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	}

	// Mask keys for security (show only last 8 characters)
	masked := []map[string]interface{}{}
	for _, key := range keys {
		maskedKey := "..."
		if len(key.Key) > 8 {
			maskedKey = "..." + key.Key[len(key.Key)-8:]
		}

		masked = append(masked, map[string]interface{}{
			"name":         key.Name,
			"masked_key":   maskedKey,
			"created_at":   key.CreatedAt.Format(time.RFC3339),
			"last_used_at": formatTimePtr(key.LastUsedAt),
			"expires_at":   key.ExpiresAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(masked); err != nil {
		http.Error(w, "Failed to encode API keys", http.StatusInternalServerError)
	}
}

// HandleGenerateAPIKey creates a new API key for the current user
func (s *Server) HandleGenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(contextKeyUser).(*users.User)
	if !ok || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name       string `json:"name"`
		ExpiryDays int    `json:"expiry_days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "API key name is required", http.StatusBadRequest)
		return
	}

	if req.ExpiryDays <= 0 {
		req.ExpiryDays = 90 // Default to 90 days
	}

	// Check if user exists in users.yaml (local user) or is OIDC user
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	_, err = store.GetUser(user.Username)
	isOIDCUser := err != nil // User not found in yaml = OIDC user

	if isOIDCUser && s.db != nil {
		// Generate API key for OIDC user (store in database)
		apiKey, err := s.generateDatabaseAPIKey(user.Username, req.Name, req.ExpiryDays)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Return the full key only on creation
		response := map[string]interface{}{
			"key":        apiKey.Key,
			"name":       apiKey.Name,
			"created_at": apiKey.CreatedAt.Format(time.RFC3339),
			"expires_at": apiKey.ExpiresAt.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode API key", http.StatusInternalServerError)
		}
	} else {
		// Generate API key for local user (store in users.yaml)
		apiKey, err := store.GenerateAPIKey(user.Username, req.Name, req.ExpiryDays)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Return the full key only on creation
		response := map[string]interface{}{
			"key":        apiKey.Key,
			"name":       apiKey.Name,
			"created_at": apiKey.CreatedAt.Format(time.RFC3339),
			"expires_at": apiKey.ExpiresAt.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode API key", http.StatusInternalServerError)
		}
	}
}

// HandleRevokeAPIKey deletes an API key
func (s *Server) HandleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(contextKeyUser).(*users.User)
	if !ok || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract key name from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}
	keyName := pathParts[len(pathParts)-1]

	// Check if user exists in users.yaml (local user) or is OIDC user
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	_, err = store.GetUser(user.Username)
	isOIDCUser := err != nil // User not found in yaml = OIDC user

	if isOIDCUser && s.db != nil {
		// Delete API key from database for OIDC user
		err = s.db.DeleteAPIKey(user.Username, keyName)
		if err != nil {
			http.Error(w, "Failed to revoke API key", http.StatusInternalServerError)
			return
		}
	} else {
		// Delete API key from users.yaml for local user
		err = store.RevokeAPIKey(user.Username, keyName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// formatTimePtr formats a time pointer to RFC3339 string or returns empty string
func formatTimePtr(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// executeDummyStep executes a dummy workflow step with logging for testing purposes
func (s *Server) executeDummyStep(step types.Step, appName string, envType string, logBuffer *LogBuffer) error {
	_, _ = logBuffer.Write([]byte("INFO: This is a dummy workflow for testing the logging system"))
	_, _ = fmt.Fprintf(logBuffer, "Executing dummy step '%s' for application: %s", step.Name, appName)
	_, _ = fmt.Fprintf(logBuffer, "Environment: %s", envType)

	// Simulate some processing time with multiple log entries
	_, _ = logBuffer.Write([]byte("Step 1: Initializing dummy process..."))
	time.Sleep(500 * time.Millisecond)

	_, _ = logBuffer.Write([]byte("Step 2: Processing dummy data..."))
	_, _ = logBuffer.Write([]byte("INFO: Dummy workflow is demonstrating log capture functionality"))
	time.Sleep(500 * time.Millisecond)

	_, _ = logBuffer.Write([]byte("Step 3: Finalizing dummy process..."))
	time.Sleep(300 * time.Millisecond)

	_, _ = logBuffer.Write([]byte("SUCCESS: Dummy workflow step completed successfully"))
	_, _ = logBuffer.Write([]byte("This log message confirms the enhanced logging system is working"))

	return nil
}

// generateDatabaseAPIKey generates an API key for OIDC users and stores it in the database
func (s *Server) generateDatabaseAPIKey(username, keyName string, expiryDays int) (*users.APIKey, error) {
	// Check if database is available
	if s.db == nil {
		return nil, fmt.Errorf("database not available for OIDC user API keys")
	}

	// Check if API key name already exists for this user
	existingKeys, err := s.db.GetAPIKeys(username)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing API keys: %w", err)
	}

	for _, key := range existingKeys {
		if key.KeyName == keyName {
			return nil, fmt.Errorf("API key with name '%s' already exists for user '%s'", keyName, username)
		}
	}

	// Generate a cryptographically secure API key
	apiKeyString, err := generateAPIKeyString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the API key for storage
	keyHash := hashAPIKey(apiKeyString)

	// Calculate expiration
	expiresAt := time.Now().Add(time.Duration(expiryDays) * 24 * time.Hour)

	// Store in database
	err = s.db.CreateAPIKey(username, keyHash, keyName, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to store API key: %w", err)
	}

	// Return API key (similar structure to file-based keys)
	return &users.APIKey{
		Key:       apiKeyString,
		Name:      keyName,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}, nil
}

// handleGraphWorkflowDetails handles /api/graph/<app>/workflow/<id> requests
// Returns workflow execution details including steps with configuration and logs
func (s *Server) handleGraphWorkflowDetails(w http.ResponseWriter, r *http.Request, appName, workflowID string) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.workflowExecutor == nil {
		http.Error(w, "Workflow executor not initialized", http.StatusInternalServerError)
		return
	}

	// Parse workflow ID
	id, err := strconv.ParseInt(workflowID, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid workflow ID: %v", err), http.StatusBadRequest)
		return
	}

	// Get workflow execution details
	execution, err := s.workflowExecutor.GetWorkflowExecution(id)
	if err != nil {
		if err.Error() == "workflow execution not found" {
			http.Error(w, "Workflow execution not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get workflow execution: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the full workflow execution with steps
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(execution); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// generateAPIKeyString generates a cryptographically secure API key
func generateAPIKeyString() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashAPIKey creates a SHA-256 hash of an API key
func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}
