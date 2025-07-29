package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLayoutFromConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
paths:
  clusterInfo: "custom/cluster-info"
  clusterResources: "custom/resources"
  podLogs: "custom/logs"
  previousPodLogs: "custom/previous-logs"
  configMaps: "custom/resources/configmaps"
  secrets: "custom/resources/secrets"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load layout from config
	layout, err := LoadLayoutFromConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load layout from config: %v", err)
	}

	// Test that custom paths are used
	if layout.ClusterInfo() != "custom/cluster-info" {
		t.Errorf("Expected ClusterInfo() to return 'custom/cluster-info', got '%s'", layout.ClusterInfo())
	}

	if layout.ClusterResources() != "custom/resources" {
		t.Errorf("Expected ClusterResources() to return 'custom/resources', got '%s'", layout.ClusterResources())
	}

		if layout.PodLogs() != "custom/logs" {
		t.Errorf("Expected PodLogs() to return 'custom/logs', got '%s'", layout.PodLogs())
	}
	
	if layout.PreviousPodLogs() != "custom/previous-logs" {
		t.Errorf("Expected PreviousPodLogs() to return 'custom/previous-logs', got '%s'", layout.PreviousPodLogs())
	}
	
	if layout.ConfigMaps() != "custom/resources/configmaps" {
		t.Errorf("Expected ConfigMaps() to return 'custom/resources/configmaps', got '%s'", layout.ConfigMaps())
	}
	
	if layout.Secrets() != "custom/resources/secrets" {
		t.Errorf("Expected Secrets() to return 'custom/resources/secrets', got '%s'", layout.Secrets())
	}
}

func TestLoadLayoutFromConfig_NonExistentFile(t *testing.T) {
	// Test loading from non-existent config file
	layout, err := LoadLayoutFromConfig("/non/existent/path/config.yaml")
	if err != nil {
		t.Fatalf("Expected no error when config file doesn't exist, got: %v", err)
	}

	// Should return default layout
	if layout.ClusterInfo() != DefaultClusterInfo {
		t.Errorf("Expected default ClusterInfo() path, got '%s'", layout.ClusterInfo())
	}
}

func TestConfigLayout_FallbackToDefaults(t *testing.T) {
	// Test config with empty values falls back to defaults
	config := LayoutConfig{
		Paths: PathsConfig{
			ClusterInfo: "custom/cluster-info",
			// Other fields are empty, should fall back to defaults
		},
	}

	layout := configLayout{config: config}

	// Custom path should be used
	if layout.ClusterInfo() != "custom/cluster-info" {
		t.Errorf("Expected custom ClusterInfo() path, got '%s'", layout.ClusterInfo())
	}

	// Default paths should be used for empty fields
	if layout.ClusterResources() != DefaultClusterResources {
		t.Errorf("Expected default ClusterResources() path, got '%s'", layout.ClusterResources())
	}

	if layout.PodLogs() != DefaultPodLogs {
		t.Errorf("Expected default PodLogs() path, got '%s'", layout.PodLogs())
	}
}

func TestLoadLayoutFromHome(t *testing.T) {
	// Test loading from home directory
	layout, err := LoadLayoutFromHome()
	if err != nil {
		// It's okay if the home config doesn't exist, should fall back to default
		if layout.ClusterInfo() != DefaultClusterInfo {
			t.Errorf("Expected default ClusterInfo() path when home config doesn't exist, got '%s'", layout.ClusterInfo())
		}
		return
	}

	// If home config exists, test that it's being used
	if layout.ClusterInfo() == DefaultClusterInfo {
		t.Log("Home config exists but returns default values")
	}
}

func TestSkipLists(t *testing.T) {
	// Test default skip lists
	layout := defaultLayout{}

	// Test default skip resources
	skipResources := layout.SkipResources()
	if len(skipResources) == 0 {
		t.Error("Expected default skip resources to be non-empty")
	}

	// Test default skip dirs
	skipDirs := layout.SkipDirs()
	if len(skipDirs) == 0 {
		t.Error("Expected default skip dirs to be non-empty")
	}

	// Test that specific expected items are in the lists
	expectedResources := []string{"custom-resource-definitions.json", "namespaces.json"}
	for _, expected := range expectedResources {
		if !contains(skipResources, expected) {
			t.Errorf("Expected skip resources to contain '%s'", expected)
		}
	}

	expectedDirs := []string{"apiservices", "pod-disruption-budgets"}
	for _, expected := range expectedDirs {
		if !contains(skipDirs, expected) {
			t.Errorf("Expected skip dirs to contain '%s'", expected)
		}
	}
}

func TestConfigSkipLists(t *testing.T) {
	// Create a temporary config file with custom skip lists
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
skip:
  resources:
    - "custom-resource.json"
    - "my-resource.yaml"
  dirs:
    - "my-custom-dir"
    - "another-dir"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load layout from config
	layout, err := LoadLayoutFromConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load layout from config: %v", err)
	}

	// Test custom skip resources
	skipResources := layout.SkipResources()
	expectedResources := []string{"custom-resource.json", "my-resource.yaml"}
	for _, expected := range expectedResources {
		if !contains(skipResources, expected) {
			t.Errorf("Expected skip resources to contain '%s'", expected)
		}
	}

	// Test custom skip dirs
	skipDirs := layout.SkipDirs()
	expectedDirs := []string{"my-custom-dir", "another-dir"}
	for _, expected := range expectedDirs {
		if !contains(skipDirs, expected) {
			t.Errorf("Expected skip dirs to contain '%s'", expected)
		}
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
