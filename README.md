# MoxApp DNS Load Test - Golang

High-performance concurrent HTTP load test with DNS timing metrics.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Quick Start](#quick-start)
  - [Build](#build)
  - [Configure](#configure)
  - [Run](#run)
  - [Docker](#docker)
- [API Documentation](#api-documentation)
- [API Endpoints](#api-endpoints)
- [CLI Options](#cli-options)
- [Configuration](#configuration)
  - [Outgoing Endpoints](#outgoing-endpoints-configuration)
  - [Incoming Routes](#incoming-routes-configuration)
  - [Managing Incoming Routes at Runtime](#managing-incoming-routes-at-runtime)
- [Architecture](#architecture)
- [Use Cases](#use-cases)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [License](#license)

## Overview

MoxApp simulates realistic infrastructure load with concurrent outgoing traffic and incoming request handling. It is lightweight, fully configurable via YAML, keeps metrics in memory, and includes a built-in web UI.

## Docker Hub

### Description

Lightweight load testing tool for realistic traffic simulation with a built-in web UI.

### Category

Developer Tools

### Repository Overview

MoxApp is a concurrent load testing tool that measures DNS timing, executes configurable outgoing traffic, and simulates incoming API routes. It ships as a single Go binary with a bundled web UI, exposing metrics and management endpoints on port 8080.

**Quick run**:

```bash
docker run --rm -p 8080:8080 kremniov/moxapp:latest
```

Open the UI at:

```
http://localhost:8080/
```

## Features

### Outgoing Traffic Generation
- **True Concurrent Execution**: Goroutine-based scheduling with semaphore-controlled concurrency
- **DNS Timing Metrics**: Precise DNS resolution timing via `net/http/httptrace`
- **In-Memory Metrics**: No file I/O on the hot path, thread-safe with atomic counters
- **Configurable Endpoints**: YAML configuration with template support for dynamic URLs
- **Multiple Auth Types**: Support for API keys, bearer tokens, and basic auth

### Incoming Traffic Simulation
- **Dynamic Route Configuration**: Define simulated API routes with configurable response patterns
- **Weighted Responses**: Probabilistic response status codes (e.g., 90% success, 10% errors)
- **Latency Simulation**: Random delays within configurable min/max ranges per response
- **Prefix Matching**: Route matching with path prefix support (e.g., `/api/users` matches `/api/users/123`)
- **Method Filtering**: Per-route HTTP method specification with wildcard support
- **Runtime Management**: Create, update, delete, enable/disable routes via REST API
- **Hot Reload**: Reload routes from configuration file without restart
- **Echo Responses**: Returns full request details (method, path, headers, query params, body)

### General
- **HTTP API**: JSON endpoints for metrics, health checks, and route management
- **Graceful Shutdown**: Waits for in-flight requests on SIGINT/SIGTERM

## Quick Start

### Build

```bash
# Download dependencies
go mod download

# Build (includes frontend bundle)
make build

# Or directly
go build -o bin/moxapp ./cmd/moxapp
```

### Configure

1. Copy the example environment file:
```bash
cp .env.example .env
```

2. Edit `.env` with your API URLs and credentials.

3. Copy `configs/endpoints.example.yaml` to `configs/endpoints.yaml` and customize it for your endpoints.

### Run

```bash
# Basic run (will prompt for confirmation)
./bin/moxapp

# Run with custom multiplier (50% load)
./bin/moxapp --multiplier=0.5

# Run without confirmation prompt
./bin/moxapp --yes

# If no config file is present, the app starts with defaults and no endpoints.
# Use the UI to add endpoints or copy configs/endpoints.example.yaml.

# Open UI
# http://localhost:8080/

# Dry run (show config without executing)
./bin/moxapp --dry-run

# Filter specific endpoints
./bin/moxapp --filter=example_a,example_b
```

### Docker

```bash
# Build and run with docker-compose
docker-compose up --build

# Or build image directly
docker build -t moxapp .
docker run -p 8080:8080 --env-file .env moxapp
```

## API Documentation

The API includes built-in interactive documentation using **OpenAPI 3.1** specification.

### Swagger UI

Access the interactive Swagger UI documentation at:

```
http://localhost:8080/api/docs/swagger
```

Swagger UI provides:
- Interactive endpoint testing with "Try it out" functionality
- Request/response schema visualization
- Authentication configuration
- Request examples

### ReDoc

For a more readable, print-friendly documentation:

```
http://localhost:8080/api/docs/redoc
```

### OpenAPI Specification

Download the raw OpenAPI specification:

```bash
# YAML format
curl http://localhost:8080/api/docs/openapi.yaml

# View in browser
open http://localhost:8080/api/docs/openapi.yaml
```

You can import this specification into:
- **Postman**: Import → Link → paste URL
- **Insomnia**: Import/Export → From URL
- **VS Code**: OpenAPI extension for live editing
- **API clients**: Generate client SDKs using OpenAPI Generator

### Documentation Endpoints

| Endpoint | Description |
|----------|-------------|
| `/api/docs` | Redirects to Swagger UI |
| `/api/docs/swagger` | Swagger UI - Interactive documentation |
| `/api/docs/redoc` | ReDoc - Alternative documentation viewer |
| `/api/docs/openapi.yaml` | OpenAPI 3.1 specification (YAML) |

## API Endpoints

Once running, the following API endpoints are available:

### API Documentation

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/docs` | GET | Redirects to Swagger UI |
| `/api/docs/swagger` | GET | Swagger UI - Interactive API documentation |
| `/api/docs/redoc` | GET | ReDoc - Alternative documentation viewer |
| `/api/docs/openapi.yaml` | GET | OpenAPI 3.1 specification (YAML) |

### Outgoing Load Test Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check with memory, goroutine stats, and incoming routes info |
| `/api/metrics` | GET | Metrics summary + snapshots (outgoing + incoming) |
| `/api/metrics/reset` | POST | Reset all metrics (outgoing + incoming) |
| `/api/config` | GET | Current configuration |
| `/api/config/validate` | GET | Validate configuration |

### Incoming Routes Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/incoming/routes` | GET | List all incoming routes |
| `/api/incoming/routes` | POST | Create a new route |
| `/api/incoming/routes/{name}` | GET | Get specific route by name |
| `/api/incoming/routes/{name}` | PUT | Update a route |
| `/api/incoming/routes/{name}` | DELETE | Delete a route |
| `/api/incoming/routes/reload` | POST | Reload all routes from config file |

### Incoming Routes Control

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/incoming/control` | GET | Get incoming routes status |
| `/api/incoming/control` | POST | Enable/disable all incoming routes |
| `/api/incoming/control/route` | POST | Enable/disable a specific route |

### Incoming Routes Metrics

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/incoming/metrics` | GET | Get per-route metrics with status distribution |
| `/api/incoming/metrics/reset` | POST | Reset incoming routes metrics |

### Simulated Incoming Routes

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/sim/*` | ANY | All configured incoming routes (e.g., `/sim/api/call_info`) |

## CLI Options

```
Flags:
  -c, --concurrent int      Number of concurrent requests (default 30)
      --config string       Configuration file path (default "configs/endpoints.yaml")
      --dry-run             Show configuration without running
  -f, --filter string       Comma-separated endpoint name filters
  -h, --help                help for moxapp
      --log-requests        Log all individual requests
  -m, --multiplier float    Global load multiplier (default 1)
      --port int            API server port (default 8080)
      --validate            Validate config and exit
  -y, --yes                 Skip confirmation prompt
```

## Configuration

### Outgoing Endpoints Configuration

Environment variables define the base URLs and credentials for external services to call. Use the example configuration in `configs/endpoints.example.yaml` as a starting point.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `EXAMPLE_BASE_URL` | Base URL for example endpoints |
| `EXAMPLE_API_KEY` | API key for header/query auth examples |
| `EXAMPLE_BEARER_TOKEN` | Bearer token for static bearer example |
| `EXAMPLE_CUSTOM_HEADER` | Token for custom header auth example |
| `EXAMPLE_BASIC_USER` | Username for basic auth example |
| `EXAMPLE_BASIC_PASS` | Password for basic auth example |
| `EXAMPLE_TOKEN_URL` | Token endpoint URL for refresh flow |
| `EXAMPLE_CLIENT_ID` | Client ID for token refresh example |
| `EXAMPLE_CLIENT_SECRET` | Client secret for token refresh example |

### URL Templates

URL templates support the following functions:

| Function | Example | Description |
|----------|---------|-------------|
| `randomString n` | `{{ randomString 8 }}` | Random alphanumeric string |
| `randomInt min max` | `{{ randomInt 1000 9999 }}` | Random integer |
| `randomPhone` | `{{ randomPhone }}` | Random French phone number |
| `randomEmail` | `{{ randomEmail }}` | Random email address |
| `randomUUID` | `{{ randomUUID }}` | Random UUID v4 |
| `now` | `{{ now }}` | Current timestamp (RFC3339) |
| `today` | `{{ today }}` | Today's date (YYYY-MM-DD) |
| `yesterday` | `{{ yesterday }}` | Yesterday's date |
| `urlEncode` | `{{ urlEncode (randomPhone) }}` | URL-encode a value |
| `env` | `{{ env "API_KEY" }}` | Get environment variable |

### Authentication Types

| Auth Type | Description |
|-----------|-------------|
| `none` | No authentication |
| `api_key_header` | API key in header |
| `api_key_query` | API key in query param |
| `bearer_static` | Static bearer token |
| `bearer_refresh` | Bearer token with refresh endpoint |
| `basic_auth` | HTTP basic auth |
| `custom_header` | Custom header token |

### Incoming Routes Configuration

Incoming routes simulate API endpoints that respond with configurable patterns. Routes are defined in the unified `configs/endpoints.yaml` file under the `incoming_routes:` section.

#### Configuration File Structure

```yaml
incoming_enabled: true  # Global enable/disable for incoming routes

incoming_routes:
  - name: call_info                # Unique route identifier
    path: /api/call_info           # URL path (supports prefix matching)
    method: GET                    # HTTP method or "*" for any
    enabled: true                  # Enable/disable the route
    responses:
      - status: 200                # HTTP status code
        share: 0.90                # Probability (0.0-1.0, must sum to 1.0)
        min_response_ms: 100       # Minimum simulated latency
        max_response_ms: 300       # Maximum simulated latency
      - status: 504                # Error response
        share: 0.10                # 10% error rate
        min_response_ms: 500
        max_response_ms: 2000
```

#### Example Configuration

```yaml
incoming_routes:
  - name: user_service
    path: /api/users
    method: "*"                    # Accept any HTTP method
    enabled: true
    responses:
      - status: 200
        share: 0.95                # 95% success rate
        min_response_ms: 50
        max_response_ms: 200
      - status: 500
        share: 0.05                # 5% server errors
        min_response_ms: 100
        max_response_ms: 500

  - name: create_order
    path: /api/orders
    method: POST
    enabled: true
    responses:
      - status: 201
        share: 0.80                # 80% created
        min_response_ms: 200
        max_response_ms: 500
      - status: 400
        share: 0.15                # 15% bad request
        min_response_ms: 50
        max_response_ms: 100
      - status: 503
        share: 0.05                # 5% service unavailable
        min_response_ms: 1000
        max_response_ms: 3000
```

#### Path Prefix Matching

Routes support prefix matching with longest match priority:

- Route configured with path `/api/users`:
  - Matches: `/api/users`, `/api/users/123`, `/api/users/123/profile`
  - Does not match: `/api/user`, `/api/orders`

- If both `/api/users` and `/api/users/detail` are configured:
  - Request to `/api/users/detail/123` → matches `/api/users/detail` (longer match)
  - Request to `/api/users/123` → matches `/api/users`

The path suffix (extra path after the configured route) is captured and included in the response.

#### Accessing Simulated Routes

All configured routes are accessible under the `/sim/` prefix:

```bash
# Route configured with path: /api/call_info
curl http://localhost:8080/sim/api/call_info

# Route configured with path: /api/users
curl http://localhost:8080/sim/api/users/123

# With query parameters
curl "http://localhost:8080/sim/api/orders?status=pending"

# POST with body
curl -X POST http://localhost:8080/sim/api/orders \
  -H "Content-Type: application/json" \
  -d '{"product": "ABC123", "quantity": 5}'
```

#### Response Format

Simulated routes return an echo response with full request details:

```json
{
  "timestamp": "2026-02-04T15:26:21Z",
  "matched_route": {
    "name": "user_service",
    "path": "/api/users",
    "method": "GET"
  },
  "request": {
    "method": "GET",
    "path": "/api/users/123/profile",
    "path_suffix": "/123/profile",
    "headers": {
      "Accept": "*/*",
      "User-Agent": "curl/7.68.0"
    },
    "query_params": {
      "include": ["posts", "comments"]
    },
    "body": null,
    "remote_addr": "127.0.0.1:54321"
  },
  "response": {
    "status": 200,
    "simulated_delay_ms": 156
  }
}
```

### Managing Incoming Routes at Runtime

#### List All Routes

```bash
curl http://localhost:8080/api/incoming/routes
```

#### Create a New Route

```bash
curl -X POST http://localhost:8080/api/incoming/routes \
  -H "Content-Type: application/json" \
  -d '{
    "name": "health_check",
    "path": "/health",
    "method": "GET",
    "enabled": true,
    "responses": [
      {
        "status": 200,
        "share": 1.0,
        "min_response_ms": 10,
        "max_response_ms": 50
      }
    ]
  }'
```

#### Update a Route

```bash
curl -X PUT http://localhost:8080/api/incoming/routes/health_check \
  -H "Content-Type: application/json" \
  -d '{
    "name": "health_check",
    "path": "/health",
    "method": "GET",
    "enabled": true,
    "responses": [
      {
        "status": 200,
        "share": 0.99,
        "min_response_ms": 10,
        "max_response_ms": 50
      },
      {
        "status": 503,
        "share": 0.01,
        "min_response_ms": 100,
        "max_response_ms": 200
      }
    ]
  }'
```

#### Delete a Route

```bash
curl -X DELETE http://localhost:8080/api/incoming/routes/health_check
```

#### Enable/Disable a Specific Route

```bash
# Disable
curl -X POST http://localhost:8080/api/incoming/control/route \
  -H "Content-Type: application/json" \
  -d '{"route_name": "user_service", "enabled": false}'

# Enable
curl -X POST http://localhost:8080/api/incoming/control/route \
  -H "Content-Type: application/json" \
  -d '{"route_name": "user_service", "enabled": true}'
```

#### Enable/Disable All Routes

```bash
# Disable all
curl -X POST http://localhost:8080/api/incoming/control \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'

# Enable all
curl -X POST http://localhost:8080/api/incoming/control \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'
```

#### Reload Routes from Config File

**Warning**: This replaces all runtime-created routes with the static configuration.

```bash
curl -X POST http://localhost:8080/api/incoming/routes/reload
```

#### View Incoming Routes Metrics

```bash
# Per-route metrics with status distribution
curl http://localhost:8080/api/incoming/metrics

# Response includes:
# - Total requests per route
# - Requests by status code (200, 400, 500, etc.)
# - Average, P95, P99 response times
# - Enabled/disabled status
```

#### Reset Incoming Metrics

```bash
curl -X POST http://localhost:8080/api/incoming/metrics/reset
```

### Configuration Validation Rules

#### Incoming Routes

- **Name**: Required, must be unique across all routes
- **Path**: Required, must start with `/`
- **Method**: Required, valid HTTP method (GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD) or `"*"` for any
- **Enabled**: Optional, defaults to `true`
- **Responses**: Required, must have at least one response

#### Response Configuration

- **Status**: Required, valid HTTP status code (100-599)
- **Share**: Required, must be between 0.0 and 1.0, all shares must sum to 1.0
- **Min Response MS**: Required, must be >= 0
- **Max Response MS**: Required, must be >= min_response_ms

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Golang Application                        │
│                      (Single Binary)                          │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────┐                                        │
│  │   main.go        │ CLI parsing, signal handling           │
│  └────────┬─────────┘                                        │
│           │                                                   │
│           v                                                   │
│  ┌─────────────────────────────────────────────┐            │
│  │      Outgoing Traffic Scheduler              │            │
│  │         Scheduler (scheduler.go)             │            │
│  │  - 10ms tick interval                        │            │
│  │  - Spawns goroutines for due requests        │            │
│  │  - Semaphore for concurrency control         │            │
│  └────────┬────────────────────────────────────┘            │
│           │                                                   │
│           v                                                   │
│  ┌─────────────────────────────────────────────┐            │
│  │       HTTP Client (client.go)                │            │
│  │  - DNS timing via httptrace                  │            │
│  │  - Connection pooling                        │            │
│  │  - Authentication handling                   │            │
│  └────────┬────────────────────────────────────┘            │
│           │                                                   │
│           v                                                   │
│  ┌─────────────────────────────────────────────┐            │
│  │  Outgoing Metrics Collector (metrics.go)     │            │
│  │  - In-memory, thread-safe                    │            │
│  │  - Ring buffers for percentiles              │            │
│  │  - Per-endpoint and per-domain stats         │            │
│  └─────────────────────────────────────────────┘            │
│                                                               │
│  ┌─────────────────────────────────────────────┐            │
│  │      Incoming Traffic Simulation             │            │
│  │   Config Manager (config.go)                 │            │
│  │  - Route matching (prefix-based)             │            │
│  │  - CRUD operations for routes                │            │
│  │  - Thread-safe route storage                 │            │
│  └────────┬────────────────────────────────────┘            │
│           │                                                   │
│           v                                                   │
│  ┌─────────────────────────────────────────────┐            │
│  │  Incoming Metrics Collector (incoming.go)    │            │
│  │  - Per-route metrics                         │            │
│  │  - Status code distribution                  │            │
│  │  - Response time percentiles                 │            │
│  └─────────────────────────────────────────────┘            │
│                                                               │
│  ┌─────────────────────────────────────────────┐            │
│  │         HTTP API Server (api.go)             │            │
│  │  - Outgoing metrics & config endpoints       │            │
│  │  - Incoming routes management endpoints      │            │
│  │  - Simulated routes (/sim/*)                 │            │
│  │  - Health checks                             │            │
│  │  - Runs on separate goroutine                │            │
│  └─────────────────────────────────────────────┘            │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

## Use Cases

### Testing Kubernetes Infrastructure

The primary use case for this tool is testing Kubernetes clusters and infrastructure under realistic load conditions:

1. **Outgoing Traffic**: Simulate your application making external API calls
   - Configure endpoints with realistic intervals and patterns
   - Measure DNS resolution timing and connection pooling
   - Test egress network policies and latencies

2. **Incoming Traffic**: Simulate external services calling your APIs
   - Configure routes with realistic success/error ratios
   - Simulate various latency patterns (fast reads, slow writes)
   - Test ingress load balancing and autoscaling

3. **Combined Load**: Run both simultaneously to simulate a complete application
   - Deploy to Kubernetes and generate bidirectional traffic
   - Monitor pod autoscaling, resource usage, and network performance
   - Identify bottlenecks in service mesh, ingress, or cluster networking

### Example: Running in a Cluster

This example shows a typical flow for running the load tester in a cluster:

```bash
# 1. Deploy the load tester to your cluster
kubectl apply -f k8s/deployment.yaml

# 2. Customize the example config
# Copy configs/endpoints.example.yaml to configs/endpoints.yaml and customize it (includes both outgoing_endpoints and incoming_routes)

# 3. Run the load test
./bin/moxapp --multiplier=0.5 --yes

# 4. Monitor metrics via API or logs
curl http://localhost:8080/api/metrics

# 5. Generate incoming traffic to test ingress
curl http://localhost:8080/sim/api/status
```

### Example: Testing API Gateway Behavior

Test how your API gateway handles different response patterns:

```yaml
# configs/endpoints.yaml (incoming_routes section)
incoming_routes:
  - name: fast_endpoint
    path: /api/fast
    method: GET
    enabled: true
    responses:
      - status: 200
        share: 1.0
        min_response_ms: 10
        max_response_ms: 50

  - name: slow_endpoint
    path: /api/slow
    method: GET
    enabled: true
    responses:
      - status: 200
        share: 0.70
        min_response_ms: 2000
        max_response_ms: 5000
      - status: 504
        share: 0.30
        min_response_ms: 10000
        max_response_ms: 10000

  - name: flaky_endpoint
    path: /api/flaky
    method: POST
    enabled: true
    responses:
      - status: 200
        share: 0.50
        min_response_ms: 100
        max_response_ms: 300
      - status: 500
        share: 0.30
        min_response_ms: 50
        max_response_ms: 100
      - status: 503
        share: 0.20
        min_response_ms: 1000
        max_response_ms: 2000
```

Then use a load generator (like Apache Bench, wrk, or k6) to test:

```bash
# Test fast endpoint
ab -n 1000 -c 10 http://localhost:8080/sim/api/fast

# Test slow endpoint (observe timeouts)
ab -n 100 -c 5 -t 60 http://localhost:8080/sim/api/slow

# Test flaky endpoint (observe retry behavior)
ab -n 500 -c 20 http://localhost:8080/sim/api/flaky
```

### Example: Testing Circuit Breaker Patterns

Simulate downstream service failures to test circuit breaker behavior:

```bash
# 1. Start with healthy service
curl -X POST http://localhost:8080/api/incoming/control/route \
  -H "Content-Type: application/json" \
  -d '{"route_name": "payment_service", "enabled": true}'

# 2. Generate baseline traffic
# (your application calls /sim/api/payment)

# 3. Simulate service degradation (increase error rate)
curl -X PUT http://localhost:8080/api/incoming/routes/payment_service \
  -H "Content-Type: application/json" \
  -d '{
    "name": "payment_service",
    "path": "/api/payment",
    "method": "POST",
    "enabled": true,
    "responses": [
      {"status": 200, "share": 0.20, "min_response_ms": 100, "max_response_ms": 200},
      {"status": 500, "share": 0.50, "min_response_ms": 50, "max_response_ms": 100},
      {"status": 503, "share": 0.30, "min_response_ms": 1000, "max_response_ms": 3000}
    ]
  }'

# 4. Observe circuit breaker opening

# 5. Restore service health
curl -X PUT http://localhost:8080/api/incoming/routes/payment_service \
  -H "Content-Type: application/json" \
  -d '{
    "name": "payment_service",
    "path": "/api/payment",
    "method": "POST",
    "enabled": true,
    "responses": [
      {"status": 200, "share": 0.99, "min_response_ms": 100, "max_response_ms": 200},
      {"status": 500, "share": 0.01, "min_response_ms": 50, "max_response_ms": 100}
    ]
  }'

# 6. Observe circuit breaker recovery
```

## Development

```bash
# Run tests
make test

# Run with coverage
make test-coverage

# Run linter (requires golangci-lint)
make lint

# Format code
make fmt

# Build for multiple platforms
make build-all
```

## Troubleshooting

### Incoming Routes Not Loading

**Symptom**: Log shows routes don't appear or incoming routes are empty

**Solutions**:
```bash
# Check if config file exists and has incoming routes section
ls -la configs/endpoints.yaml

# Validate YAML syntax
cat configs/endpoints.yaml | yq eval

# Check for validation errors in logs
./bin/moxapp 2>&1 | grep -i "incoming"

# Verify incoming routes are in the config
grep -A 10 "incoming_routes:" configs/endpoints.yaml
```

### Route Not Matching Requests

**Symptom**: 404 errors when calling simulated routes

**Common Causes**:
1. **Missing `/sim/` prefix**: Routes must be accessed via `/sim/` prefix
   ```bash
   # Wrong
   curl http://localhost:8080/api/users
   
   # Correct
   curl http://localhost:8080/sim/api/users
   ```

2. **Route is disabled**: Check route status
   ```bash
   curl http://localhost:8080/api/incoming/routes | jq '.routes[] | select(.name=="user_service")'
   ```

3. **HTTP method mismatch**: Route configured for POST but using GET
   ```bash
   # Check route configuration
   curl http://localhost:8080/api/incoming/routes/user_service | jq '.method'
   
   # Use correct method or change route to "*"
   curl -X POST http://localhost:8080/sim/api/users
   ```

### Response Share Validation Errors

**Symptom**: "response shares must sum to 1.0" error

**Solution**: Ensure all response shares add up to exactly 1.0
```yaml
# Wrong (sums to 0.95)
responses:
  - status: 200
    share: 0.90
  - status: 500
    share: 0.05

# Correct (sums to 1.0)
responses:
  - status: 200
    share: 0.90
  - status: 500
    share: 0.10
```

### High Memory Usage

**Symptom**: Application memory usage keeps growing

**Causes and Solutions**:

1. **Too many metrics samples**: Metrics use ring buffers (1000 samples per endpoint)
   ```bash
   # Reset metrics periodically
   curl -X POST http://localhost:8080/api/metrics/reset
   curl -X POST http://localhost:8080/api/incoming/metrics/reset
   ```

2. **Too many concurrent requests**: Reduce concurrency
   ```bash
   ./bin/moxapp --concurrent=10  # Default is 30
   ```

3. **Memory leak**: Check health endpoint for goroutine leaks
   ```bash
   curl http://localhost:8080/health | jq '.goroutines'
   # Should be relatively stable, not constantly increasing
   ```

### Port Already in Use

**Symptom**: "bind: address already in use" error

**Solutions**:
```bash
# Check what's using port 8080
lsof -i :8080
netstat -tuln | grep 8080

# Use a different port
./bin/moxapp --port=8081

# Kill existing process
kill $(lsof -t -i:8080)
```

### Cannot Access API from External Host

**Symptom**: API works on localhost but not from external IP

**Cause**: Server binds to localhost by default

**Solution**: The server listens on `0.0.0.0` by default, so this should work. Check:
```bash
# 1. Verify server is listening on all interfaces
netstat -tuln | grep 8080
# Should show 0.0.0.0:8080, not 127.0.0.1:8080

# 2. Check firewall
sudo ufw status
sudo firewall-cmd --list-ports

# 3. Allow port if needed
sudo ufw allow 8080
sudo firewall-cmd --add-port=8080/tcp --permanent
```

### Incoming Routes Return Wrong Status Codes

**Symptom**: Always getting 200 when expecting mixed responses

**Cause**: Weighted selection is probabilistic

**Solution**: Test multiple times to see distribution
```bash
# Test 100 times and count status codes
for i in {1..100}; do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/sim/api/flaky
done | sort | uniq -c

# Example output:
#   50 200
#   30 500
#   20 503

# Check metrics to see actual distribution
curl http://localhost:8080/api/incoming/metrics | jq '.routes[] | select(.name=="flaky")'
```

### Configuration Not Reloading

**Symptom**: Changes to YAML files don't take effect

**Cause**: Application doesn't watch files for changes

**Solutions**:
```bash
# Option 1: Restart the application
# Press Ctrl+C and run again
./bin/moxapp

# Option 2: Use hot reload endpoint (incoming routes only)
curl -X POST http://localhost:8080/api/incoming/routes/reload

# Option 3: Update routes via API (no restart needed)
curl -X PUT http://localhost:8080/api/incoming/routes/route_name \
  -H "Content-Type: application/json" \
  -d @updated_route.json
```

### DNS Timing Shows Zero

**Symptom**: DNS time is always 0ms in metrics

**Causes**:
1. **DNS cached**: Subsequent requests use cached DNS
   ```bash
   # This is normal behavior - DNS is only resolved once per domain
   # Check first request to a new domain for DNS timing
   ```

2. **Using IP address**: No DNS resolution needed
   ```bash
   # URLs with IPs don't need DNS resolution
   # Use domain names instead: example.com (not 1.2.3.4)
   ```

3. **Connection reuse**: HTTP keep-alive reuses connections
   ```bash
   # This is expected - DNS is measured once per new connection
   ```

## License

Proprietary - MoxApp
