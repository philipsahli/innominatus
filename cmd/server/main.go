package main

import (
	"context"
	"flag"
	"fmt"
	"innominatus/internal/admin"
	"innominatus/internal/ai"
	"innominatus/internal/database"
	"innominatus/internal/logging"
	"innominatus/internal/metrics"
	"innominatus/internal/server"
	"innominatus/internal/tracing"
	"innominatus/internal/validation"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Build information - set via ldflags during release builds
var (
	version = "dev"
	commit  = "unknown"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func isStaticAsset(path string) bool {
	// Check if path starts with common static asset prefixes
	return strings.HasPrefix(path, "/.next/") ||
		strings.HasPrefix(path, "/favicon") ||
		strings.HasSuffix(path, ".js") ||
		strings.HasSuffix(path, ".css") ||
		strings.HasSuffix(path, ".png") ||
		strings.HasSuffix(path, ".jpg") ||
		strings.HasSuffix(path, ".svg") ||
		strings.HasSuffix(path, ".ico")
}

func main() {
	var port = flag.String("port", "8081", "HTTP server port")
	var disableDB = flag.Bool("disable-db", false, "Disable database features")
	var skipValidation = flag.Bool("skip-validation", false, "Skip configuration validation on startup")
	flag.Parse()

	// Initialize structured logger for server startup
	logger := logging.NewStructuredLogger("server")

	// Run configuration validation before starting
	if !*skipValidation {
		logger.Info("Running configuration validation")
		validation.ValidateConfigurationWithExit()
		logger.Info("Configuration validation passed")
	}

	// Initialize OpenTelemetry tracing
	tp, err := tracing.InitTracer(version, commit)
	if err != nil {
		logger.WarnWithFields("Failed to initialize tracer", map[string]interface{}{
			"error": err.Error(),
		})
		fmt.Println("Continuing without distributed tracing...")
	} else if tp.IsEnabled() {
		logger.Info("OpenTelemetry tracing initialized")
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tp.Shutdown(ctx); err != nil {
				logger.WarnWithFields("Error shutting down tracer", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}()
	}

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		fmt.Printf("Warning: Failed to load admin config: %v\n", err)
		fmt.Println("Continuing without admin configuration...")
	} else {
		fmt.Println("Admin configuration loaded:")
		adminConfig.PrintConfig()
		fmt.Println()
	}

	var srv *server.Server

	if !*disableDB {
		// Try to initialize database
		db, err := database.NewDatabase()
		if err != nil {
			logger.WarnWithFields("Failed to connect to database", map[string]interface{}{
				"error": err.Error(),
			})
			fmt.Println("Starting server without database features...")
			srv = server.NewServer()
		} else {
			// Initialize schema
			err = db.InitSchema()
			if err != nil {
				logger.WarnWithFields("Failed to initialize database schema", map[string]interface{}{
					"error": err.Error(),
				})
				fmt.Println("Starting server without database features...")
				_ = db.Close()
				srv = server.NewServer()
			} else {
				logger.Info("Database connected successfully")
				srv = server.NewServerWithDB(db)
			}
		}
	} else {
		logger.Info("Database features disabled")
		srv = server.NewServer()
	}

	// Initialize AI service (optional - continues without AI if not configured)
	aiService, err := ai.NewServiceFromEnv(context.Background())
	if err != nil {
		logger.WarnWithFields("Failed to initialize AI service", map[string]interface{}{
			"error": err.Error(),
		})
	} else if aiService.IsEnabled() {
		srv.SetAIService(aiService)
		logger.Info("AI assistant service initialized successfully")
	}

	// Helper to apply standard middleware chain (OTel Tracing -> TraceID -> Logging)
	withTrace := func(h http.HandlerFunc) http.HandlerFunc {
		return srv.TracingMiddleware(srv.TraceIDMiddleware(srv.LoggingMiddleware(h)))
	}

	// Helper to apply trace, logging, and CORS
	withTraceCORS := func(h http.HandlerFunc) http.HandlerFunc {
		return srv.TracingMiddleware(srv.TraceIDMiddleware(srv.LoggingMiddleware(srv.CorsMiddleware(h))))
	}

	// Helper to apply trace, logging, and auth
	withTraceAuth := func(h http.HandlerFunc) http.HandlerFunc {
		return srv.TracingMiddleware(srv.TraceIDMiddleware(srv.LoggingMiddleware(srv.AuthMiddleware(h))))
	}

	// Helper to apply full middleware chain (OTel Tracing -> TraceID -> Logging -> CORS -> Auth)
	withTraceCORSAuth := func(h http.HandlerFunc) http.HandlerFunc {
		return srv.TracingMiddleware(srv.TraceIDMiddleware(srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(h)))))
	}

	// Helper to apply full admin middleware chain
	withTraceCORSAdmin := func(h http.HandlerFunc) http.HandlerFunc {
		return srv.TracingMiddleware(srv.TraceIDMiddleware(srv.LoggingMiddleware(srv.CorsMiddleware(srv.AdminOnlyMiddleware(h)))))
	}

	// Authentication routes (with trace ID and logging)
	http.HandleFunc("/auth/login", withTrace(srv.HandleLogin))
	http.HandleFunc("/logout", withTrace(srv.HandleLogout))
	http.HandleFunc("/api/login", withTraceCORS(srv.HandleAPILogin))
	http.HandleFunc("/api/user-info", withTraceAuth(srv.HandleUserInfo))

	// OIDC authentication routes (if enabled via environment variables)
	http.HandleFunc("/auth/oidc/login", withTrace(srv.HandleOIDCLogin))
	http.HandleFunc("/auth/callback", withTrace(srv.HandleOIDCCallback))

	// API routes (with trace ID, logging, CORS, and authentication)
	http.HandleFunc("/api/specs", withTraceCORSAuth(srv.HandleSpecs))
	http.HandleFunc("/api/specs/", withTraceCORSAuth(srv.HandleSpecDetail))
	http.HandleFunc("/api/environments", withTraceCORSAuth(srv.HandleEnvironments))
	http.HandleFunc("/api/workflows", withTraceCORSAuth(srv.HandleWorkflows))
	http.HandleFunc("/api/workflows/", withTraceCORSAuth(srv.HandleWorkflowDetail))
	http.HandleFunc("/api/workflow-analysis", withTraceCORSAuth(srv.HandleWorkflowAnalysis))
	http.HandleFunc("/api/workflow-analysis/preview", withTraceCORSAuth(srv.HandleWorkflowAnalysisPreview))
	http.HandleFunc("/api/stats", withTraceCORSAuth(srv.HandleStats))
	http.HandleFunc("/api/teams", withTraceCORSAdmin(srv.HandleTeams))
	http.HandleFunc("/api/teams/", withTraceCORSAdmin(srv.HandleTeamDetail))

	// Admin-only impersonation routes
	http.HandleFunc("/api/impersonate", withTraceCORSAdmin(srv.HandleImpersonate))
	http.HandleFunc("/api/users", withTraceCORSAdmin(srv.HandleListUsers))

	// Profile management routes (authenticated users only)
	http.HandleFunc("/api/profile", withTraceCORSAuth(srv.HandleGetProfile))
	http.HandleFunc("/api/profile/api-keys", withTraceCORSAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			srv.HandleGetAPIKeys(w, r)
		case http.MethodPost:
			srv.HandleGenerateAPIKey(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	http.HandleFunc("/api/profile/api-keys/", withTraceCORSAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			srv.HandleRevokeAPIKey(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// Demo Environment API routes (with trace ID, logging, CORS, and authentication)
	http.HandleFunc("/api/demo/status", withTraceCORSAuth(srv.HandleDemoStatus))
	http.HandleFunc("/api/demo/time", withTraceCORSAuth(srv.HandleDemoTime))
	http.HandleFunc("/api/demo/nuke", withTraceCORSAuth(srv.HandleDemoNuke))

	// Graph API routes (with trace ID, logging, CORS, and authentication)
	http.HandleFunc("/api/graph", withTraceCORSAuth(srv.HandleGraph))
	http.HandleFunc("/api/graph/", withTraceCORSAuth(srv.HandleGraph))

	// Resource management API routes (with trace ID, logging, CORS, and authentication)
	http.HandleFunc("/api/resources", withTraceCORSAuth(srv.HandleResources))
	http.HandleFunc("/api/resources/", withTraceCORSAuth(srv.HandleResourceDetail))

	// Application management API routes (with trace ID, logging, CORS, and authentication)
	http.HandleFunc("/api/applications/", withTraceCORSAuth(srv.HandleApplicationManagement))

	// Golden path workflow execution API routes (with trace ID, logging, CORS, and authentication)
	http.HandleFunc("/api/workflows/golden-paths/", withTraceCORSAuth(srv.HandleGoldenPathExecution))

	// AI Assistant API routes (with trace ID, logging, CORS, and authentication)
	if aiService != nil && aiService.IsEnabled() {
		http.HandleFunc("/api/ai/chat", withTraceCORSAuth(aiService.HandleChat))
		http.HandleFunc("/api/ai/generate-spec", withTraceCORSAuth(aiService.HandleGenerateSpec))
		http.HandleFunc("/api/ai/status", withTraceCORS(aiService.HandleStatus))
		logger.Info("AI assistant API routes registered")
	}

	// Swagger API documentation routes (with trace ID and logging)
	http.HandleFunc("/swagger", withTrace(srv.HandleSwagger))
	http.HandleFunc("/swagger.yaml", withTrace(srv.HandleSwaggerYAML))
	http.HandleFunc("/swagger-admin", withTrace(srv.HandleSwaggerAdmin))
	http.HandleFunc("/swagger-admin.yaml", withTrace(srv.HandleSwaggerAdminYAML))
	http.HandleFunc("/swagger-user", withTrace(srv.HandleSwaggerUser))
	http.HandleFunc("/swagger-user.yaml", withTrace(srv.HandleSwaggerUserYAML))

	// Health check endpoints (with tracing but no auth - for monitoring systems)
	http.HandleFunc("/health", srv.TracingMiddleware(srv.TraceIDMiddleware(srv.HandleHealth)))
	http.HandleFunc("/ready", srv.TracingMiddleware(srv.TraceIDMiddleware(srv.HandleReady)))
	http.HandleFunc("/metrics", srv.TracingMiddleware(srv.TraceIDMiddleware(srv.HandleMetrics)))

	// Auth configuration endpoint (with tracing but no auth - needed before login)
	http.HandleFunc("/api/auth/config", srv.TracingMiddleware(srv.TraceIDMiddleware(srv.HandleAuthConfig)))

	// Web UI (static files) - no authentication needed for static assets
	fs := http.FileServer(http.Dir("./web-ui/out/"))
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For SPA routing, serve appropriate index.html for non-existent routes that aren't static assets
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			r.URL.Path = "/"
		} else if !fileExists("./web-ui/out"+r.URL.Path) && !isStaticAsset(r.URL.Path) {
			// For /graph/* routes, serve /graph/index.html
			if strings.HasPrefix(r.URL.Path, "/graph/") {
				r.URL.Path = "/graph/"
			} else if strings.HasPrefix(r.URL.Path, "/goldenpaths/") {
				r.URL.Path = "/goldenpaths/"
			} else {
				r.URL.Path = "/"
			}
		}
		fs.ServeHTTP(w, r)
	}))

	// Initialize metrics pusher if PUSHGATEWAY_URL is set
	pushgatewayURL := os.Getenv("PUSHGATEWAY_URL")
	if pushgatewayURL == "" {
		pushgatewayURL = "http://pushgateway.localtest.me"
	}

	var metricsPusher *metrics.MetricsPusher
	if pushgatewayURL != "" && pushgatewayURL != "disabled" {
		pushInterval := 15 * time.Second
		metricsPusher = metrics.NewMetricsPusher(pushgatewayURL, pushInterval, version, commit)
		metricsPusher.StartPushing()
		defer metricsPusher.Stop()
	}

	addr := ":" + *port

	// Log server startup with structured logging
	logger.InfoWithFields("Starting Score Orchestrator server", map[string]interface{}{
		"version":          version,
		"commit":           commit,
		"port":             *port,
		"address":          "http://localhost" + addr,
		"database_enabled": !*disableDB,
		"tracing_enabled":  tp.IsEnabled(),
	})

	fmt.Printf("Starting Score Orchestrator server on http://localhost%s\n", addr)
	fmt.Println("API endpoints:")
	fmt.Println("  POST /api/specs          - Deploy Score spec with embedded workflows (simple deployments)")
	fmt.Println("  POST /api/workflows/golden-paths/deploy-app/execute - Deploy via golden path (recommended)")
	fmt.Println("  GET  /api/specs          - List all deployed specs")
	fmt.Println("  GET  /api/specs/{name}   - Get specific spec details")
	fmt.Println("  DELETE /api/specs/{name} - Delete deployed spec")
	fmt.Println("  GET  /api/environments   - List active environments")
	fmt.Println("  GET  /api/workflows      - List workflow executions")
	fmt.Println("  GET  /api/workflows/{id} - Get workflow execution details")
	fmt.Println("  GET  /api/stats          - Get dashboard statistics")
	fmt.Println("  GET  /api/teams          - List teams")
	fmt.Println("  POST /api/teams          - Create new team")
	fmt.Println("  GET  /api/teams/{id}     - Get team details")
	fmt.Println("  DELETE /api/applications/{name} - Delete application and all resources")
	fmt.Println("  POST /api/applications/{name}/deprovision - Deprovision infrastructure")
	fmt.Println("  DELETE /api/teams/{id}   - Delete team")
	fmt.Println("")
	fmt.Println("Web interface:")
	fmt.Printf("  Dashboard: http://localhost%s/\n", addr)
	fmt.Printf("  API Docs:  http://localhost%s/swagger\n", addr)
	fmt.Println("")
	fmt.Println("Health & Monitoring:")
	fmt.Printf("  Health:    http://localhost%s/health\n", addr)
	fmt.Printf("  Readiness: http://localhost%s/ready\n", addr)
	fmt.Printf("  Metrics:   http://localhost%s/metrics\n", addr)
	fmt.Println("")
	fmt.Println("Database configuration (set via environment variables):")
	fmt.Println("  DB_HOST (default: localhost)")
	fmt.Println("  DB_PORT (default: 5432)")
	fmt.Println("  DB_USER (default: postgres)")
	fmt.Println("  DB_PASSWORD")
	fmt.Println("  DB_NAME (default: idp_orchestrator)")
	fmt.Println("  DB_SSLMODE (default: disable)")

	// Create HTTP server with proper timeouts to prevent resource exhaustion
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.InfoWithFields("HTTP server started", map[string]interface{}{
		"address":       addr,
		"read_timeout":  "15s",
		"write_timeout": "15s",
		"idle_timeout":  "60s",
	})

	if err := httpServer.ListenAndServe(); err != nil {
		logger.ErrorWithFields("Server failed", map[string]interface{}{
			"error": err.Error(),
		})
		log.Fatal(err)
	}
}
