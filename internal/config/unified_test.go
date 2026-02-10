package config

import (
	"testing"
)

func TestUnifiedConfigLoad(t *testing.T) {
	manager := NewManager()
	
	// Load the merged config file
	if err := manager.LoadFromFile("../../configs/endpoints.yaml"); err != nil {
		t.Fatalf("Failed to load unified config: %v", err)
	}
	
	cfg := manager.GetConfig()
	
	// Verify outgoing endpoints
	if len(cfg.Endpoints) == 0 {
		t.Error("Expected outgoing endpoints to be loaded")
	}
	
	// Verify incoming routes
	if len(cfg.IncomingRoutes) == 0 {
		t.Error("Expected incoming routes to be loaded")
	}
	
	// Verify incoming enabled flag
	if !cfg.IncomingEnabled {
		t.Error("Expected incoming to be enabled")
	}
	
	t.Logf("Successfully loaded config:")
	t.Logf("  Outgoing endpoints: %d", len(cfg.Endpoints))
	t.Logf("  Incoming routes: %d", len(cfg.IncomingRoutes))
	t.Logf("  Incoming enabled: %v", cfg.IncomingEnabled)
}
