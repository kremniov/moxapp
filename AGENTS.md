# AGENTS.md - MoxApp DNS Load Test (Go)

Guidelines for AI coding agents working in this repository.

## Project Overview

High-performance concurrent HTTP load testing tool with DNS timing metrics. Uses goroutine-based scheduling for non-blocking concurrent request execution.

**Tech Stack:** Go 1.25.6, Cobra (CLI), Viper (config), YAML

## Directory Structure

```
moxapp/
├── cmd/moxapp/             # Entry point, CLI parsing, signal handling
├── internal/
│   ├── api/                # HTTP server, handlers, middleware
│   ├── client/             # HTTP client with DNS tracing
│   ├── config/             # Configuration loading, validation
│   ├── metrics/            # In-memory metrics collection
│   └── scheduler/          # Goroutine-based request scheduling
├── configs/                # YAML endpoint definitions
└── Makefile                # Build automation
```

## Build Commands

```bash
make build          # Build binary to bin/moxapp
make run            # Build and run
make dry-run        # Show configuration without execution
make clean          # Remove build artifacts
```

## Test Commands

```bash
make test                                           # Run all tests
go test -v ./...                                    # Run all tests (verbose)
go test -v -run TestFunctionName ./internal/pkg/   # Run single test
go test -v ./internal/client/                       # Run tests in package
make test-coverage                                  # Run with coverage report
make bench                                          # Run all benchmarks
go test -bench=BenchmarkName -benchmem ./internal/pkg/  # Single benchmark
```

## Lint and Format

```bash
make fmt            # Format code (go fmt ./...)
make vet            # Vet code (go vet ./...)
make lint           # Run golangci-lint (must be installed)
make deps           # Download and tidy dependencies
```

## Code Style Guidelines

### Package Documentation

Every package must have a doc comment:

```go
// Package client provides HTTP client functionality with DNS timing
package client
```

### Import Organization

Three groups separated by blank lines:

```go
import (
    // Standard library
    "context"
    "net/http"

    // Third-party packages
    "github.com/spf13/cobra"

    // Internal packages
    "moxapp/internal/config"
)
```

### Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Exported types/functions | PascalCase | `EndpointMetrics`, `NewCollector` |
| Unexported identifiers | camelCase | `httpClient`, `totalRequests` |
| Constants | PascalCase | `DefaultTimeout` |
| Constructor functions | `New<Type>` or `New` | `NewCollector()`, `New(opts)` |

### Struct Tags

```go
type Endpoint struct {
    Name    string `mapstructure:"name" yaml:"name" json:"name"`
    Enabled bool   `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
    Error   string `json:"error,omitempty"`  // Use omitempty for optional fields
}
```

### Error Handling

1. Return errors as the last return value
2. Wrap errors with `fmt.Errorf` and `%w`:
   ```go
   return nil, fmt.Errorf("failed to load config: %w", err)
   ```
3. Error messages: lowercase, no trailing punctuation
4. For validation, return `[]string` of error messages

### Function Comments

Exported function comments must start with the function name:

```go
// Execute executes an HTTP request for the given endpoint
func (c *Client) Execute(ctx context.Context, endpoint *config.Endpoint) *RequestResult {
```

### Concurrency Patterns

- `sync.RWMutex` for thread-safe map/struct access
- `sync/atomic` for counter operations
- Channels for semaphore-based concurrency control
- `sync.WaitGroup` for goroutine coordination
- `context.Context` for cancellation

```go
type Collector struct {
    totalRequests int64          // Use atomic operations
    endpoints     map[string]*EndpointMetrics
    mu            sync.RWMutex   // Protects endpoints map
}

func (c *Collector) Record(result *client.RequestResult) {
    c.mu.Lock()
    defer c.mu.Unlock()
    atomic.AddInt64(&c.totalRequests, 1)
}
```

### Method Receivers

Use pointer receivers for structs with mutex or mutable state.

### File Organization

1. Package comment
2. Imports
3. Type definitions
4. Constructor functions (`New...`)
5. Public methods
6. Private methods/helpers

### HTTP Response Handling

Always close response bodies and handle connection reuse:

```go
resp, err := c.httpClient.Do(req)
if err != nil {
    return result
}
defer resp.Body.Close()

// Read and discard body to allow connection reuse
bodySize, _ := io.Copy(io.Discard, resp.Body)
```

## Configuration

- Main config: `configs/endpoints.yaml` (copy from `configs/endpoints.example.yaml`)
  - Unified configuration for both outgoing and incoming traffic
  - Outgoing endpoints defined under `outgoing_endpoints:`
  - Incoming routes defined under `incoming_routes:`
  - Incoming feature controlled by `incoming_enabled:`
- Environment variables: `.env` (copy from `.env.example`)
- CLI flags override config file values

## Docker Commands

```bash
make docker         # Build Docker image
make up             # Docker compose up
make down           # Docker compose down
make logs           # View Docker compose logs
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/metrics` | GET | Full metrics snapshot |
| `/api/metrics/reset` | POST | Reset all metrics |
| `/api/config` | GET | Current configuration |
| `/api/endpoints` | GET/POST | List/create endpoints |
| `/api/endpoints/{name}` | GET/PUT/DELETE | CRUD operations |
