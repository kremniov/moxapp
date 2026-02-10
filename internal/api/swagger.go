// Package api provides the HTTP API server for metrics and configuration
package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"strings"
)

//go:embed openapi.yaml
var openAPISpec []byte

// serverURLPlaceholder is replaced at runtime with the actual server URL
const serverURLPlaceholder = "__SERVER_URL__"

// SwaggerUIHTML is the HTML template for Swagger UI
// Uses unpkg CDN to load Swagger UI assets
const SwaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MoxApp - API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; background: #fafafa; }
        .swagger-ui .topbar { display: none; }
        .swagger-ui .info { margin: 20px 0; }
        .swagger-ui .info .title { font-size: 32px; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "/api/docs/openapi.yaml",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 1,
                docExpansion: "list",
                filter: true,
                showExtensions: true,
                showCommonExtensions: true,
                tryItOutEnabled: true
            });
            window.ui = ui;
        };
    </script>
</body>
</html>`

// ReDocHTML is the HTML template for ReDoc (alternative documentation viewer)
const ReDocHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MoxApp - API Documentation</title>
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
        body { margin: 0; padding: 0; }
    </style>
</head>
<body>
    <redoc spec-url='/api/docs/openapi.yaml'></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`

// handleSwaggerUI serves the Swagger UI HTML page
func (s *Server) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(SwaggerUIHTML))
}

// handleReDoc serves the ReDoc HTML page
func (s *Server) handleReDoc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(ReDocHTML))
}

// handleOpenAPISpec serves the OpenAPI specification file with dynamic server URL
func (s *Server) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	// Determine the server URL from the request
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Check for X-Forwarded-Proto header (common behind load balancers/proxies)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	serverURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	// Replace placeholder with actual server URL
	spec := strings.Replace(string(openAPISpec), serverURLPlaceholder, serverURL, 1)

	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(spec))
}

// handleDocsRoute routes documentation requests
func (s *Server) handleDocsRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/docs")

	switch {
	case path == "" || path == "/":
		// Redirect to Swagger UI
		http.Redirect(w, r, "/api/docs/swagger", http.StatusMovedPermanently)
	case path == "/swagger" || path == "/swagger/":
		s.handleSwaggerUI(w, r)
	case path == "/redoc" || path == "/redoc/":
		s.handleReDoc(w, r)
	case path == "/openapi.yaml" || path == "/openapi.yml":
		s.handleOpenAPISpec(w, r)
	case path == "/openapi.json":
		// Could add YAML to JSON conversion here if needed
		writeError(w, "JSON format not supported, use /api/docs/openapi.yaml", http.StatusNotImplemented)
	default:
		http.NotFound(w, r)
	}
}
