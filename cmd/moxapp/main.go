// MoxApp - Golang Implementation
// High-performance concurrent HTTP load test with DNS timing metrics
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"moxapp/internal/api"
	"moxapp/internal/client"
	"moxapp/internal/config"
	"moxapp/internal/metrics"
	"moxapp/internal/scheduler"
)

var (
	// CLI flags
	multiplier  float64
	concurrent  int
	filter      string
	validate    bool
	dryRun      bool
	configFile  string
	apiPort     int
	logRequests bool
	noConfirm   bool

	// Version info
	version   = "1.0.0"
	buildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "moxapp",
	Short: "DNS load test for MoxApp",
	Long: `MoxApp - Golang Implementation
High-performance concurrent HTTP load test with DNS timing metrics.

This tool simulates production-like traffic patterns to test DNS resolution
and API endpoint performance under load.`,
	Run: runLoadTest,
}

func init() {
	rootCmd.Flags().Float64VarP(&multiplier, "multiplier", "m", 1.0, "Global load multiplier (e.g., 0.5 for 50% load)")
	rootCmd.Flags().IntVarP(&concurrent, "concurrent", "c", 30, "Number of concurrent requests")
	rootCmd.Flags().StringVarP(&filter, "filter", "f", "", "Comma-separated endpoint name filters")
	rootCmd.Flags().BoolVar(&validate, "validate", false, "Validate config and exit")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show configuration without running")
	rootCmd.Flags().StringVar(&configFile, "config", "configs/endpoints.yaml", "Configuration file path")
	rootCmd.Flags().IntVar(&apiPort, "port", 8080, "API server port")
	rootCmd.Flags().BoolVar(&logRequests, "log-requests", false, "Log all individual requests")
	rootCmd.Flags().BoolVarP(&noConfirm, "yes", "y", false, "Skip confirmation prompt")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("moxapp version %s (built: %s)\n", version, buildTime)
		},
	})
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runLoadTest(cmd *cobra.Command, args []string) {
	printBanner()

	// Create configuration manager
	configManager := config.NewManager()

	// Load configuration (optional)
	if err := configManager.LoadFromFile(configFile); err != nil {
		var notFoundErr viper.ConfigFileNotFoundError
		if errors.As(err, &notFoundErr) || errors.Is(err, fs.ErrNotExist) || errors.Is(err, os.ErrNotExist) {
			fmt.Printf("Config file not found (%s). Starting with defaults and no endpoints.\n", configFile)
		} else {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}
	}

	// Show incoming routes info
	incomingRoutes := configManager.GetIncomingRoutes()
	if len(incomingRoutes) > 0 {
		fmt.Printf("Loaded %d incoming routes from config\n", len(incomingRoutes))
	}

	// Override with CLI flags (only if explicitly set)
	if cmd.Flags().Changed("multiplier") {
		configManager.SetGlobalMultiplier(multiplier)
	}
	if cmd.Flags().Changed("concurrent") {
		configManager.SetConcurrentRequests(concurrent)
	}

	// Handle API port: CLI flag takes priority, then env var, then default
	if cmd.Flags().Changed("port") {
		configManager.SetAPIPort(apiPort) // CLI flag was explicitly set
	} else {
		configManager.SetAPIPort(configManager.GetAPIPortFromEnv()) // Use env or default
	}

	configManager.SetLogAllRequests(logRequests)

	// Get config snapshot for validation and display
	cfg := configManager.GetConfig()

	// Apply endpoint filter (this creates a filtered snapshot, not modifying manager)
	var filteredEndpoints []config.Endpoint
	if filter != "" {
		filteredEndpoints = configManager.FilterEndpoints(filter)
		if len(filteredEndpoints) == 0 {
			fmt.Fprintf(os.Stderr, "No endpoints matched filter: %s\n", filter)
			os.Exit(1)
		}
		// Update cfg snapshot for display purposes
		cfg.Endpoints = filteredEndpoints
	}

	// Validate
	if validate || dryRun {
		validateAndShowConfig(configManager, cfg)
		if validate {
			return
		}
	}

	if len(cfg.Endpoints) == 0 {
		fmt.Println("No endpoints configured. API server will start, but no outgoing traffic will run.")
	}

	// Show configuration summary
	showConfigSummary(configManager, cfg)

	// Confirm start
	if !noConfirm {
		if !confirmStart() {
			fmt.Println("Aborted.")
			return
		}
	}

	fmt.Println()
	fmt.Println("Starting load test... (Press Ctrl+C to stop gracefully)")
	fmt.Println()

	// Initialize components
	metricsCollector := metrics.NewCollector()
	incomingMetrics := metrics.NewIncomingCollector()

	// Initialize token manager for auth configs
	tokenManager := client.NewTokenManager(cfg.AuthConfigs, configManager)

	clientOpts := client.DefaultOptions()
	clientOpts.Timeout = 30 * time.Second
	clientOpts.MaxConns = cfg.ConcurrentRequests * 2
	clientOpts.LogRequests = cfg.LogAllRequests
	clientOpts.EnvGetter = configManager
	clientOpts.AuthConfigs = cfg.AuthConfigs
	clientOpts.TokenManager = tokenManager
	httpClient := client.New(clientOpts)

	// Create scheduler with config manager for live updates
	sched := scheduler.New(configManager, httpClient, func(result *client.RequestResult) {
		metricsCollector.Record(result)
		if configManager.GetConfig().LogAllRequests {
			logResult(result)
		}
	})

	// Create API server with config manager for CRUD operations
	apiAddr := fmt.Sprintf(":%d", cfg.APIPort)
	apiServer := api.NewServerWithManager(apiAddr, metricsCollector, configManager)
	apiServer.SetScheduler(sched)
	apiServer.SetTokenManager(tokenManager)
	apiServer.SetIncomingMetrics(incomingMetrics)

	// Start API server in background
	go func() {
		fmt.Printf("API server listening on http://localhost:%d\n", cfg.APIPort)
		fmt.Printf("  - Web UI:    http://localhost:%d/\n", cfg.APIPort)
		fmt.Printf("  - API Docs:  http://localhost:%d/api/docs/swagger\n", cfg.APIPort)
		fmt.Printf("  - Metrics:   http://localhost:%d/api/metrics\n", cfg.APIPort)
		fmt.Printf("  - Outgoing:  http://localhost:%d/api/outgoing/endpoints\n", cfg.APIPort)
		fmt.Printf("  - Incoming:  http://localhost:%d/api/incoming/routes\n", cfg.APIPort)
		fmt.Printf("  - Health:    http://localhost:%d/health\n", cfg.APIPort)
		fmt.Println()
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "API server error: %v\n", err)
		}
	}()

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start token manager background refresh
	tokenManager.StartBackgroundRefresh(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println()
		fmt.Println("Received shutdown signal, stopping gracefully...")
		cancel()
	}()

	// Start live metrics display
	stopDisplay := make(chan struct{})
	go displayLiveMetrics(metricsCollector, stopDisplay)

	// Run scheduler (blocks until context is cancelled)
	if err := sched.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Scheduler error: %v\n", err)
	}

	// Stop live display
	close(stopDisplay)

	// Shutdown API server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "API server shutdown error: %v\n", err)
	}

	fmt.Println()
	fmt.Println("Load test stopped.")
	showFinalStats(metricsCollector, incomingMetrics)
}

func printBanner() {
	fmt.Println("=============================================================")
	fmt.Println("  MoxApp (Golang)")
	fmt.Printf("  Version: %s\n", version)
	fmt.Println("=============================================================")
	fmt.Println()
}

func validateAndShowConfig(manager *config.Manager, cfg *config.Config) {
	errors := manager.Validate()

	if len(errors) > 0 {
		fmt.Println("Configuration Errors:")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		fmt.Println()
		os.Exit(1)
	}

	fmt.Println("Configuration is valid.")
	fmt.Println()
	showConfigSummary(manager, cfg)
}

func showConfigSummary(manager *config.Manager, cfg *config.Config) {
	baseReqPerMin := manager.GetTotalBaseRequestsPerMin()
	adjustedReqPerMin := manager.GetAdjustedRequestsPerMin()

	fmt.Println("Configuration Summary:")
	fmt.Println("-------------------------------------------------------------")
	fmt.Printf("  Config File:                %s\n", configFile)
	fmt.Printf("  Global Multiplier:          %.2f\n", cfg.GlobalMultiplier)
	fmt.Printf("  Concurrent Requests:        %d\n", cfg.ConcurrentRequests)
	fmt.Printf("  Total Endpoints:            %d\n", len(cfg.Endpoints))
	fmt.Printf("  Base Requests/min:          %.2f\n", baseReqPerMin)
	fmt.Printf("  Adjusted Requests/min:      %.2f\n", adjustedReqPerMin)
	fmt.Printf("  Estimated Requests/sec:     %.2f\n", adjustedReqPerMin/60)
	fmt.Printf("  API Port:                   %d\n", cfg.APIPort)
	fmt.Printf("  Log All Requests:           %v\n", cfg.LogAllRequests)
	fmt.Println("-------------------------------------------------------------")
	fmt.Println()

	// Show endpoint summary by auth type
	authCounts := make(map[string]int)
	for _, ep := range cfg.Endpoints {
		authName := "none"
		if ep.ResolvedAuth != nil {
			authName = ep.ResolvedAuth.Name
		}
		authCounts[authName]++
	}

	fmt.Println("Endpoints by Auth Type:")
	for authType, count := range authCounts {
		fmt.Printf("  %-20s %d\n", authType+":", count)
	}
	fmt.Println()
}

func confirmStart() bool {
	fmt.Print("Start load test? (yes/no) [yes]: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "" || response == "yes" || response == "y"
}

func logResult(result *client.RequestResult) {
	status := "OK"
	if !result.Success {
		status = "FAIL"
	}
	fmt.Printf("\r[%s] %s %s %s (dns:%.1fms total:%.1fms)\n",
		status,
		result.Method,
		result.EndpointName,
		result.Hostname,
		result.DNSTimeMs,
		result.TotalTimeMs,
	)
}

func displayLiveMetrics(collector *metrics.Collector, stop chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			total := collector.GetTotalRequests()
			rate := collector.GetRequestsPerSecond()
			successRate := collector.GetSuccessRate()
			fmt.Printf("\r[LIVE] Requests: %d | Rate: %.1f req/s | Success: %.1f%%     ",
				total, rate, successRate)
		}
	}
}

func showFinalStats(collector *metrics.Collector, incomingCollector *metrics.IncomingCollector) {
	snapshot := collector.Snapshot()

	fmt.Println("Final Statistics (Outgoing Traffic):")
	fmt.Println("=============================================================")
	fmt.Printf("  Uptime:                     %.2f seconds\n", snapshot.UptimeSeconds)
	fmt.Printf("  Total Requests:             %d\n", snapshot.TotalRequests)
	fmt.Printf("  Successful:                 %d (%.2f%%)\n", snapshot.TotalSuccesses, snapshot.SuccessRate)
	fmt.Printf("  Failed:                     %d\n", snapshot.TotalFailures)
	fmt.Printf("  Requests/sec:               %.2f\n", snapshot.RequestsPerSecond)
	fmt.Println("=============================================================")
	fmt.Println()

	// Show top failures
	type failureInfo struct {
		name  string
		count int64
		error string
	}
	var failures []failureInfo

	for name, ep := range snapshot.Endpoints {
		if ep.Failed > 0 {
			failures = append(failures, failureInfo{
				name:  name,
				count: ep.Failed,
				error: ep.LastError,
			})
		}
	}

	if len(failures) > 0 {
		// Sort by failure count descending
		sort.Slice(failures, func(i, j int) bool {
			return failures[i].count > failures[j].count
		})

		fmt.Println("Endpoints with Failures (top 10):")
		for i, f := range failures {
			if i >= 10 {
				break
			}
			fmt.Printf("  %s: %d failures\n", f.name, f.count)
			if f.error != "" {
				fmt.Printf("    Last error: %s\n", f.error)
			}
		}
		fmt.Println()
	}

	// Show DNS stats
	if len(snapshot.DNSStatsByDomain) > 0 {
		fmt.Println("DNS Resolution Stats by Domain:")
		for hostname, stats := range snapshot.DNSStatsByDomain {
			if stats.FailedLookups > 0 {
				fmt.Printf("  %s: %d failed lookups (avg: %.2fms, p95: %.2fms)\n",
					hostname, stats.FailedLookups, stats.AvgResolutionMs, stats.P95ResolutionMs)
			} else if stats.SuccessfulLookups > 0 {
				fmt.Printf("  %s: avg %.2fms, p95 %.2fms (total: %d lookups)\n",
					hostname, stats.AvgResolutionMs, stats.P95ResolutionMs, stats.TotalLookups)
			}
		}
		fmt.Println()
	}

	// Show incoming routes stats
	if incomingCollector != nil {
		incomingSnapshot := incomingCollector.Snapshot()
		if incomingSnapshot.TotalRequests > 0 {
			fmt.Println("Incoming Routes Statistics:")
			fmt.Println("=============================================================")
			fmt.Printf("  Total Incoming Requests:    %d\n", incomingSnapshot.TotalRequests)
			fmt.Printf("  Requests/sec:               %.2f\n", incomingSnapshot.RequestsPerSecond)
			fmt.Println()

			// Show per-route stats
			for name, route := range incomingSnapshot.Routes {
				fmt.Printf("  Route: %s (%s)\n", name, route.RoutePath)
				fmt.Printf("    Requests: %d | Avg: %.2fms | P95: %.2fms\n",
					route.TotalRequests, route.AvgResponseMs, route.P95ResponseMs)
				// Show response status distribution
				var statusParts []string
				for status, count := range route.ResponsesByStatus {
					statusParts = append(statusParts, fmt.Sprintf("%d: %d", status, count))
				}
				if len(statusParts) > 0 {
					fmt.Printf("    Status codes: %s\n", strings.Join(statusParts, ", "))
				}
			}
			fmt.Println("=============================================================")
			fmt.Println()
		}
	}
}
