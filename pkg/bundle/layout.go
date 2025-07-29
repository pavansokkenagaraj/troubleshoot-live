package bundle

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Default layout path constants
const (
	DefaultClusterInfo      = "k8s/cluster-info"
	DefaultClusterResources = "k8s/cluster-resources"
	DefaultPodLogs          = "k8s/pod-logs"
	DefaultPreviousPodLogs  = "k8s/previous-pod-logs"
	DefaultConfigMaps       = "k8s/cluster-resources/configmaps"
	DefaultSecrets          = "k8s/cluster-resources/secrets"
)

// Default skip lists
var (
	DefaultSkipResources = []string{
		// crds are imported during a separate step
		"custom-resource-definitions.json",
		"pod-disruption-budgets-info.json",
		// api-resources from the discovery client
		"resources.json",
		// api-groups from the discovery client
		"groups.json",
		// namespaces are imported as first resource in a separate step
		"namespaces.json",
		// mutatingwebhookconfigurations TODO: fix this
		"mutatingwebhookconfigurations.yaml",
		// validatingwebhookconfigurations TODO: fix this
		"validatingwebhookconfigurations.yaml",
	}

	DefaultSkipDirs = []string{
		"apiservices",
		"auth-cani-list",
		"pod-disruption-budgets",
	}
)

// Layout defines paths under which are particular resources stored.
type Layout interface {
	ClusterInfo() string
	ClusterResources() string
	PodLogs() string
	PreviousPodLogs() string
	ConfigMaps() string
	Secrets() string
	SkipResources() []string
	SkipDirs() []string
}

// PathsConfig represents the configuration for layout paths
type PathsConfig struct {
	ClusterInfo      string `yaml:"clusterInfo"`
	ClusterResources string `yaml:"clusterResources"`
	PodLogs          string `yaml:"podLogs"`
	PreviousPodLogs  string `yaml:"previousPodLogs"`
	ConfigMaps       string `yaml:"configMaps"`
	Secrets          string `yaml:"secrets"`
}

// SkipConfig represents the configuration for skip lists
type SkipConfig struct {
	Resources []string `yaml:"resources"`
	Dirs      []string `yaml:"dirs"`
}

// LayoutConfig represents the configuration structure for layout paths
type LayoutConfig struct {
	Paths PathsConfig `yaml:"paths"`
	Skip  SkipConfig  `yaml:"skip"`
}

type defaultLayout struct{}

func (defaultLayout) ClusterInfo() string {
	return DefaultClusterInfo
}

func (defaultLayout) ClusterResources() string {
	return DefaultClusterResources
}

func (defaultLayout) PodLogs() string {
	return DefaultPodLogs
}

func (defaultLayout) PreviousPodLogs() string {
	return DefaultPreviousPodLogs
}

func (defaultLayout) ConfigMaps() string {
	return DefaultConfigMaps
}

func (defaultLayout) Secrets() string {
	return DefaultSecrets
}

func (defaultLayout) SkipResources() []string {
	return DefaultSkipResources
}

func (defaultLayout) SkipDirs() []string {
	return DefaultSkipDirs
}

// configLayout implements Layout interface using configuration from a YAML file
type configLayout struct {
	config LayoutConfig
}

func (cl configLayout) ClusterInfo() string {
	if cl.config.Paths.ClusterInfo != "" {
		return cl.config.Paths.ClusterInfo
	}
	return DefaultClusterInfo
}

func (cl configLayout) ClusterResources() string {
	if cl.config.Paths.ClusterResources != "" {
		return cl.config.Paths.ClusterResources
	}
	return DefaultClusterResources
}

func (cl configLayout) PodLogs() string {
	if cl.config.Paths.PodLogs != "" {
		return cl.config.Paths.PodLogs
	}
	return DefaultPodLogs
}

func (cl configLayout) PreviousPodLogs() string {
	if cl.config.Paths.PreviousPodLogs != "" {
		return cl.config.Paths.PreviousPodLogs
	}
	return DefaultPreviousPodLogs
}

func (cl configLayout) ConfigMaps() string {
	if cl.config.Paths.ConfigMaps != "" {
		return cl.config.Paths.ConfigMaps
	}
	return DefaultConfigMaps
}

func (cl configLayout) Secrets() string {
	if cl.config.Paths.Secrets != "" {
		return cl.config.Paths.Secrets
	}
	return DefaultSecrets
}

func (cl configLayout) SkipResources() []string {
	if len(cl.config.Skip.Resources) > 0 {
		return cl.config.Skip.Resources
	}
	return DefaultSkipResources
}

func (cl configLayout) SkipDirs() []string {
	if len(cl.config.Skip.Dirs) > 0 {
		return cl.config.Skip.Dirs
	}
	return DefaultSkipDirs
}

// LoadLayoutFromConfig loads layout configuration from a config.yaml file
func LoadLayoutFromConfig(configPath string) (Layout, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default layout if config file doesn't exist
		return defaultLayout{}, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Parse YAML
	var config LayoutConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	return configLayout{config: config}, nil
}

// LoadLayoutFromBundleDir attempts to load layout config from a bundle directory
func LoadLayoutFromBundleDir(bundlePath string) (Layout, error) {
	configPath := filepath.Join(bundlePath, "config.yaml")
	return LoadLayoutFromConfig(configPath)
}

// LoadLayoutFromHome attempts to load layout config from user's home directory
func LoadLayoutFromHome() (Layout, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".troubleshoot-live", "config.yaml")
	return LoadLayoutFromConfig(configPath)
}

// LoadLayoutWithFallback attempts to load layout config with fallback priority:
// 1. Bundle directory config.yaml
// 2. Home directory ~/.troubleshoot-live/config.yaml
// 3. Default layout
func LoadLayoutWithFallback(bundlePath string) (Layout, error) {
	// First try bundle directory
	if bundlePath != "" {
		bundleLayout, err := LoadLayoutFromBundleDir(bundlePath)
		if err == nil && bundleLayout != nil {
			return bundleLayout, nil
		}
	}

	// Then try home directory
	homeLayout, err := LoadLayoutFromHome()
	if err == nil && homeLayout != nil {
		return homeLayout, nil
	}

	// Finally fall back to default
	return defaultLayout{}, nil
}
