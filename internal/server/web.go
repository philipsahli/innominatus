package server

import (
	"fmt"
	
	"net/http"
	"os"
)

func (s *Server) HandleSwagger(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Score Orchestrator API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin:0;
            background: #fafafa;
            font-family: Arial, sans-serif;
        }
        .nav {
            background: #84cc16;
            padding: 1rem 2rem;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .nav h1 {
            margin: 0;
            color: white;
            display: inline-block;
            margin-right: 2rem;
            font-size: 1.5rem;
        }
        .nav-links {
            display: inline-block;
        }
        .nav-links a {
            color: white;
            text-decoration: none;
            margin-right: 1rem;
            padding: 0.5rem 1rem;
            border-radius: 4px;
            transition: background-color 0.3s;
        }
        .nav-links a:hover {
            background: rgba(255,255,255,0.2);
        }
        .nav-links a.active {
            background: rgba(255,255,255,0.3);
        }
    </style>
</head>
<body>
    <nav class="nav">
        <h1>Score Orchestrator</h1>
        <div class="nav-links">
            <a href="/">Dashboard</a>
            <a href="/swagger" class="active">API Docs</a>
        </div>
    </nav>

    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/swagger.yaml',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(html)); err != nil {
		// Client disconnected or write failed - log but don't error
		fmt.Fprintf(os.Stderr, "failed to write response: %v\n", err)
	}
}

func (s *Server) HandleSwaggerYAML(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("swagger.yaml")
	if err != nil {
		http.Error(w, "Could not read swagger.yaml", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if _, err := w.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write response: %v\n", err)
	}
}