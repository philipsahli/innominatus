package main

import (
	"context"
	"embed"
	"flag"
	"innominatus/internal/admin"
	"innominatus/internal/ai"
	"innominatus/internal/database"
	"innominatus/internal/logging"
	"innominatus/internal/metrics"
	"innominatus/internal/providers"
	"innominatus/internal/server"
	"innominatus/internal/tracing"
	"innominatus/internal/validation"
	"innominatus/pkg/sdk"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//go:embed swagger-admin.yaml swagger-user.yaml
var swaggerFilesFS embed.FS

//go:embed web-ui-out
var webUIFS embed.FS

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
		logger.WarnWithFields("Failed to initialize tracer, continuing without distributed tracing", map[string]interface{}{
			"error": err.Error(),
		})
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
		logger.WarnWithFields("Failed to load admin config, continuing without admin configuration", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		logger.InfoWithFields("Admin configuration loaded", map[string]interface{}{
			"config": adminConfig.String(),
		})
	}

	// Initialize provider registry and load providers
	providerRegistry := providers.NewRegistry()
	if adminConfig != nil && len(adminConfig.Providers) > 0 {
		logger.InfoWithFields("Loading providers", map[string]interface{}{
			"count": len(adminConfig.Providers),
		})

		fsLoader := providers.NewLoader(version)
		gitLoader := providers.NewGitLoader("/tmp/innominatus-providers", version)

		for _, providerSrc := range adminConfig.Providers {
			if !providerSrc.Enabled {
				logger.DebugWithFields("Skipping disabled provider", map[string]interface{}{
					"name": providerSrc.Name,
				})
				continue
			}

			var provider *sdk.Provider
			var loadErr error

			switch providerSrc.Type {
			case "filesystem":
				// Load from filesystem path
				manifestPath := providerSrc.Path + "/provider.yaml"
				if _, statErr := os.Stat(manifestPath); os.IsNotExist(statErr) {
					// Try legacy platform.yaml
					manifestPath = providerSrc.Path + "/platform.yaml"
				}
				provider, loadErr = fsLoader.LoadFromFile(manifestPath)

			case "git":
				// Load from Git repository
				provider, loadErr = gitLoader.LoadFromGit(providers.GitProviderSource{
					Name:       providerSrc.Name,
					Repository: providerSrc.Repository,
					Ref:        providerSrc.Ref,
				})

			default:
				logger.WarnWithFields("Unknown provider type", map[string]interface{}{
					"name": providerSrc.Name,
					"type": providerSrc.Type,
				})
				continue
			}

			if loadErr != nil {
				logger.WarnWithFields("Failed to load provider", map[string]interface{}{
					"name":  providerSrc.Name,
					"type":  providerSrc.Type,
					"error": loadErr.Error(),
				})
				continue
			}

			// Register provider
			if err := providerRegistry.RegisterProvider(provider); err != nil {
				logger.WarnWithFields("Failed to register provider", map[string]interface{}{
					"name":  provider.Metadata.Name,
					"error": err.Error(),
				})
				continue
			}

			logger.InfoWithFields("Provider loaded successfully", map[string]interface{}{
				"name":         provider.Metadata.Name,
				"version":      provider.Metadata.Version,
				"category":     provider.Metadata.Category,
				"provisioners": len(provider.Provisioners),
			})
		}

		providerCount, provisionerCount := providerRegistry.Count()
		logger.InfoWithFields("Provider loading complete", map[string]interface{}{
			"providers":    providerCount,
			"provisioners": provisionerCount,
		})
	} else {
		logger.Info("No providers configured in admin-config.yaml")
	}

	var srv *server.Server

	if !*disableDB {
		// Try to initialize database
		db, err := database.NewDatabase()
		if err != nil {
			logger.WarnWithFields("Failed to connect to database, starting without database features", map[string]interface{}{
				"error": err.Error(),
			})
			srv = server.NewServer()
		} else {
			// Initialize schema
			err = db.InitSchema()
			if err != nil {
				logger.WarnWithFields("Failed to initialize database schema, starting without database features", map[string]interface{}{
					"error": err.Error(),
				})
				_ = db.Close()
				srv = server.NewServer()
			} else {
				logger.Info("Database connected successfully")

				// Set embedded migrations filesystem
				migrationsSubFS, err := fs.Sub(migrationsFS, "migrations")
				if err != nil {
					logger.WarnWithFields("Failed to create migrations sub-filesystem", map[string]interface{}{
						"error": err.Error(),
					})
				} else {
					db.SetMigrationsFS(migrationsSubFS)
					logger.Info("Embedded migrations filesystem configured")
				}

				// Pass admin config to enable multi-tier workflows
				srv = server.NewServerWithDBAndAdminConfig(db, adminConfig)
			}
		}
	} else {
		logger.Info("Database features disabled")
		srv = server.NewServer()
	}

	// Set provider registry on server
	if providerRegistry != nil {
		srv.SetProviderRegistry(providerRegistry)
		logger.Info("Provider registry configured")
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
	} else {
		logger.Info("AI assistant service disabled (missing API keys)")
	}

	// Set embedded swagger files filesystem
	srv.SetSwaggerFS(swaggerFilesFS)
	logger.Info("Embedded swagger files filesystem configured")

	// Set embedded web-ui files filesystem
	webUISubFS, err := fs.Sub(webUIFS, "web-ui-out")
	if err != nil {
		logger.WarnWithFields("Failed to create web-ui sub-filesystem", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		srv.SetWebUIFS(webUISubFS)
		logger.Info("Embedded web-ui filesystem configured")
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
	// Applications endpoints (preferred)
	http.HandleFunc("/api/applications", withTraceCORSAuth(srv.HandleApplications))
	http.HandleFunc("/api/applications/", withTraceCORSAuth(srv.HandleApplicationDetail))
	// Deprecated: /api/specs endpoints (kept for backward compatibility)
	http.HandleFunc("/api/specs", withTraceCORSAuth(srv.HandleSpecsDeprecated))
	http.HandleFunc("/api/specs/", withTraceCORSAuth(srv.HandleSpecDetailDeprecated))

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

	// User management routes (admin only)
	http.HandleFunc("/api/admin/users", withTraceCORSAdmin(srv.HandleUserManagement))
	http.HandleFunc("/api/admin/users/", withTraceCORSAdmin(func(w http.ResponseWriter, r *http.Request) {
		// Route to appropriate handler based on path
		if strings.Contains(r.URL.Path, "/api-keys/") {
			// /api/admin/users/{username}/api-keys/{keyname}
			srv.HandleAdminUserAPIKeyDetail(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/api-keys") {
			// /api/admin/users/{username}/api-keys
			srv.HandleAdminUserAPIKeys(w, r)
		} else {
			// /api/admin/users/{username}
			srv.HandleUserDetail(w, r)
		}
	}))

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

	// Admin-only demo routes
	http.HandleFunc("/api/admin/demo/reset", withTraceCORSAdmin(srv.HandleDemoReset))

	// Admin configuration routes
	http.HandleFunc("/api/admin/config", withTraceCORSAdmin(srv.HandleAdminConfig))

	// Graph API routes (with trace ID, logging, CORS, and authentication)
	http.HandleFunc("/api/graph", withTraceCORSAuth(srv.HandleGraph))
	// WebSocket endpoint needs special handling - no response-wrapping middleware
	http.HandleFunc("/api/graph/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/ws") {
			// WebSocket: auth only, no middleware that wraps ResponseWriter
			// Response wrappers prevent WebSocket upgrades (http.Hijacker interface required)
			srv.AuthMiddleware(srv.HandleGraph)(w, r)
		} else {
			// Regular API: full middleware stack
			withTraceCORSAuth(srv.HandleGraph)(w, r)
		}
	})

	// Resource management API routes (with trace ID, logging, CORS, and authentication)
	http.HandleFunc("/api/resources", withTraceCORSAuth(srv.HandleResources))
	http.HandleFunc("/api/resources/", withTraceCORSAuth(srv.HandleResourceDetail))

	// Golden path API routes (with trace ID, logging, CORS, and authentication)
	http.HandleFunc("/api/golden-paths", withTraceCORSAuth(srv.HandleGoldenPaths))

	// Provider management API routes (with trace ID, logging, CORS, and authentication)
	http.HandleFunc("/api/providers", withTraceCORSAuth(srv.HandleListProviders))
	http.HandleFunc("/api/providers/stats", withTraceCORSAuth(srv.HandleProviderStats))
	http.HandleFunc("/api/golden-paths/", withTraceCORSAuth(srv.HandleGoldenPaths))

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
	// Use embedded FS if available (production), otherwise use filesystem (development)
	var staticFS http.Handler
	var webUIBasePath string
	webUIEmbedFS := srv.GetWebUIFS()

	if webUIEmbedFS != nil {
		// Production: use embedded filesystem
		staticFS = http.FileServer(http.FS(webUIEmbedFS))
		webUIBasePath = "" // embedded FS is already at the right root
		logger.Info("Using embedded web-ui files")
	} else {
		// Development: use filesystem
		staticFS = http.FileServer(http.Dir("./web-ui/out/"))
		webUIBasePath = "./web-ui/out"
		logger.Info("Using filesystem web-ui files")
	}

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For SPA routing, serve appropriate index.html for non-existent routes that aren't static assets
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			r.URL.Path = "/"
		} else {
			// Check if file exists (different logic for embedded vs filesystem)
			fileExistsInUI := false
			if webUIEmbedFS != nil {
				// Check in embedded FS
				_, err := fs.Stat(webUIEmbedFS, strings.TrimPrefix(r.URL.Path, "/"))
				fileExistsInUI = err == nil
			} else {
				// Check in filesystem
				fileExistsInUI = fileExists(webUIBasePath + r.URL.Path)
			}

			if !fileExistsInUI && !isStaticAsset(r.URL.Path) {
				// For /graph/* routes, serve /graph/index.html
				if strings.HasPrefix(r.URL.Path, "/graph/") {
					r.URL.Path = "/graph/"
				} else if strings.HasPrefix(r.URL.Path, "/goldenpaths/") {
					r.URL.Path = "/goldenpaths/"
				} else {
					r.URL.Path = "/"
				}
			}
		}
		staticFS.ServeHTTP(w, r)
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

	logger.InfoWithFields("Server startup information", map[string]interface{}{
		"address":   "http://localhost" + addr,
		"dashboard": "http://localhost" + addr + "/",
		"api_docs":  "http://localhost" + addr + "/swagger",
		"health":    "http://localhost" + addr + "/health",
		"readiness": "http://localhost" + addr + "/ready",
		"metrics":   "http://localhost" + addr + "/metrics",
		"key_endpoints": []string{
			"POST /api/specs - Deploy Score spec",
			"POST /api/workflows/golden-paths/deploy-app/execute - Deploy via golden path",
			"GET  /api/workflows - List workflow executions",
		},
	})

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
