package main

import (
	"flag"
	"fmt"
	"innominatus/internal/admin"
	"innominatus/internal/database"
	"innominatus/internal/server"
	"innominatus/internal/validation"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
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

	// Run configuration validation before starting
	if !*skipValidation {
		fmt.Println("Running configuration validation...")
		validation.ValidateConfigurationWithExit()
		fmt.Println("✅ Configuration validation passed")
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
			fmt.Printf("Warning: Failed to connect to database: %v\n", err)
			fmt.Println("Starting server without database features...")
			srv = server.NewServer()
		} else {
			// Initialize schema
			err = db.InitSchema()
			if err != nil {
				fmt.Printf("Warning: Failed to initialize database schema: %v\n", err)
				fmt.Println("Starting server without database features...")
				_ = db.Close()
				srv = server.NewServer()
			} else {
				fmt.Println("Database connected successfully")
				srv = server.NewServerWithDB(db)
			}
		}
	} else {
		srv = server.NewServer()
	}

	// Authentication routes (with logging)
	http.HandleFunc("/auth/login", srv.LoggingMiddleware(srv.HandleLogin))
	http.HandleFunc("/logout", srv.LoggingMiddleware(srv.HandleLogout))
	http.HandleFunc("/api/login", srv.LoggingMiddleware(srv.CorsMiddleware(srv.HandleAPILogin)))
	http.HandleFunc("/api/user-info", srv.LoggingMiddleware(srv.AuthMiddleware(srv.HandleUserInfo)))

	// API routes (with logging, CORS, and authentication)
	http.HandleFunc("/api/specs", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleSpecs))))
	http.HandleFunc("/api/specs/", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleSpecDetail))))
	http.HandleFunc("/api/environments", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleEnvironments))))
	http.HandleFunc("/api/workflows", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleWorkflows))))
	http.HandleFunc("/api/workflows/", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleWorkflowDetail))))
	http.HandleFunc("/api/workflow-analysis", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleWorkflowAnalysis))))
	http.HandleFunc("/api/workflow-analysis/preview", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleWorkflowAnalysisPreview))))
	http.HandleFunc("/api/stats", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleStats))))
	http.HandleFunc("/api/teams", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AdminOnlyMiddleware(srv.HandleTeams))))
	http.HandleFunc("/api/teams/", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AdminOnlyMiddleware(srv.HandleTeamDetail))))

	// Admin-only impersonation routes
	http.HandleFunc("/api/impersonate", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AdminOnlyMiddleware(srv.HandleImpersonate))))
	http.HandleFunc("/api/users", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AdminOnlyMiddleware(srv.HandleListUsers))))

	// Demo Environment API routes (with logging, CORS, and authentication)
	http.HandleFunc("/api/demo/status", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleDemoStatus))))
	http.HandleFunc("/api/demo/time", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleDemoTime))))
	http.HandleFunc("/api/demo/nuke", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleDemoNuke))))

	// Graph API routes (with logging, CORS, and authentication)
	http.HandleFunc("/api/graph", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleGraph))))
	http.HandleFunc("/api/graph/", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleGraph))))

	// Resource management API routes (with logging, CORS, and authentication)
	http.HandleFunc("/api/resources", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleResources))))
	http.HandleFunc("/api/resources/", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleResourceDetail))))

	// Application management API routes (with logging, CORS, and authentication)
	http.HandleFunc("/api/applications/", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleApplicationManagement))))

	// Golden path workflow execution API routes (with logging, CORS, and authentication)
	http.HandleFunc("/api/workflows/golden-paths/", srv.LoggingMiddleware(srv.CorsMiddleware(srv.AuthMiddleware(srv.HandleGoldenPathExecution))))

	// Swagger API documentation routes
	http.HandleFunc("/swagger", srv.LoggingMiddleware(srv.HandleSwagger))
	http.HandleFunc("/swagger.yaml", srv.LoggingMiddleware(srv.HandleSwaggerYAML))

	// Health check endpoints (no authentication needed - for monitoring systems)
	http.HandleFunc("/health", srv.HandleHealth)
	http.HandleFunc("/ready", srv.HandleReady)
	http.HandleFunc("/metrics", srv.HandleMetrics)

	// Web UI (static files) - no authentication needed for static assets
	fs := http.FileServer(http.Dir("./web-ui/out/"))
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve static assets directly (don't redirect _next, favicon, etc.)
		if r.URL.Path == "/" ||
			r.URL.Path == "/index.html" ||
			!fileExists("./web-ui/out"+r.URL.Path) &&
				!isStaticAsset(r.URL.Path) {
			// For SPA routing, serve index.html for non-existent routes that aren't static assets
			r.URL.Path = "/"
		}
		fs.ServeHTTP(w, r)
	}))

	addr := ":" + *port
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
	log.Fatal(httpServer.ListenAndServe())
}
