package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Vilsol/slox"
	"github.com/adrg/xdg"
	"github.com/sasha-s/go-deadlock"
)

type MetricsConfig struct {
	PrometheusURL string `json:"prometheusUrl,omitempty"`
}

type ResourceReqs struct {
	Requests map[string]string `json:"requests,omitempty"`
	Limits   map[string]string `json:"limits,omitempty"`
}

type VolumeBrowserConfig struct {
	Image                  string            `json:"image,omitempty"`
	MountPath              string            `json:"mountPath,omitempty"`
	ReadOnly               *bool             `json:"readOnly,omitempty"`
	ActiveDeadlineSeconds  *int64            `json:"activeDeadlineSeconds,omitempty"`
	Resources              *ResourceReqs     `json:"resources,omitempty"`
	NodeSelector           map[string]string `json:"nodeSelector,omitempty"`
	Tolerations            []map[string]any  `json:"tolerations,omitempty"`
	PromptBeforeSpawn      *bool             `json:"promptBeforeSpawn,omitempty"`
	OrphanCleanupOnStartup string            `json:"orphanCleanupOnStartup,omitempty"`
}

func defaultVolumeBrowser() VolumeBrowserConfig {
	deadline := int64(3600)
	readOnly := false
	promptBeforeSpawn := false
	return VolumeBrowserConfig{
		Image:                  "alpine:edge",
		MountPath:              "/mnt/volume",
		ReadOnly:               &readOnly,
		ActiveDeadlineSeconds:  &deadline,
		PromptBeforeSpawn:      &promptBeforeSpawn,
		OrphanCleanupOnStartup: "prompt",
	}
}

type ColumnSettings struct {
	Width int `json:"width,omitempty"`
}

type SortPrefs struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
}

type GVRColumnPrefs struct {
	Columns map[string]ColumnSettings `json:"columns"`
	Order   []string                  `json:"order"`
	Sort    *SortPrefs                `json:"sort,omitempty"`
}

type ClusterPrefs struct {
	ReadOnly      *bool                      `json:"readOnly,omitempty"`
	CompactRows   *bool                      `json:"compactRows,omitempty"`
	AccentColor   *string                    `json:"accentColor,omitempty"`
	DisplayName   *string                    `json:"displayName,omitempty"`
	Metrics       *MetricsConfig             `json:"metrics,omitempty"`
	ColumnPrefs   map[string]*GVRColumnPrefs `json:"columnPrefs,omitempty"`
	FavoriteNS    []string                   `json:"favoriteNamespaces,omitempty"`
	SavedFilters  map[string][]SavedFilter   `json:"savedFilters,omitempty"`
	VolumeBrowser *VolumeBrowserConfig       `json:"volumeBrowser,omitempty"`
}

type SavedFilter struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Search      string            `json:"search,omitempty"`
}

type SavedPortForward struct {
	ID         string `json:"id"`
	Namespace  string `json:"namespace"`
	Resource   string `json:"resource"`
	TargetKind string `json:"targetKind"`
	TargetName string `json:"targetName"`
	TargetGVR  string `json:"targetGVR,omitempty"`
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
	Enabled    bool   `json:"enabled"`
}

type Config struct {
	Theme                  string                        `json:"theme"`
	KubeconfigPaths        []string                      `json:"kubeconfigPaths"`
	TerminalWebGL          bool                          `json:"terminalWebGL"`
	DisabledPlugins        []string                      `json:"disabledPlugins,omitempty"`
	InsecureRegistries     []string                      `json:"insecureRegistries,omitempty"`
	InsecureSkipTLSVerify  bool                          `json:"insecureSkipTLSVerify,omitempty"`
	Metrics                map[string]*MetricsConfig     `json:"metrics,omitempty"`
	ColumnPrefs            map[string]*GVRColumnPrefs    `json:"columnPrefs,omitempty"`
	CompactRows            bool                          `json:"compactRows,omitempty"`
	ReadOnly               bool                          `json:"readOnly,omitempty"`
	PortForwards           map[string][]SavedPortForward `json:"portForwards,omitempty"`
	Clusters               map[string]*ClusterPrefs      `json:"clusters,omitempty"`
	Keybindings            map[string]string             `json:"keybindings,omitempty"`
	SavedFilters           map[string][]SavedFilter      `json:"savedFilters,omitempty"`
	StartupBehavior        string                        `json:"startupBehavior,omitempty"`
	StartupCluster         string                        `json:"startupCluster,omitempty"`
	AccentColor            string                        `json:"accentColor,omitempty"`
	FontSize               int                           `json:"fontSize,omitempty"`
	ContextualAutocomplete *bool                         `json:"contextualAutocomplete,omitempty"`
	VolumeBrowser          VolumeBrowserConfig           `json:"volumeBrowser,omitempty"`

	mu   deadlock.Mutex
	path string
	emit func(string, any)
}

func DefaultConfig() *Config {
	return &Config{
		Theme:           "system",
		KubeconfigPaths: []string{},
		VolumeBrowser:   defaultVolumeBrowser(),
	}
}

func configPath() (string, error) {
	return xdg.ConfigFile(filepath.Join("klados", "config.json"))
}

func Load() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			slox.Info(context.Background(), "config not found, using defaults", "path", p)
			cfg := DefaultConfig()
			cfg.path = p
			return cfg, nil
		}
		return nil, err
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.path = p
	slox.Debug(context.Background(), "config loaded", "path", p)
	return cfg, nil
}

func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.saveLocked()
}

func (c *Config) saveLocked() error {
	if c.path == "" {
		p, err := configPath()
		if err != nil {
			return err
		}
		c.path = p
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(c.path, data, 0o644); err != nil {
		slox.Error(context.Background(), "config save failed", "path", c.path, "error", err)
		return err
	}
	if c.emit != nil {
		c.emit("config:updated", nil)
	}
	return nil
}

func (c *Config) SetEmit(fn func(string, any)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.emit = fn
}

func (c *Config) Update(fn func(*Config)) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn(c)
	return c.saveLocked()
}

func (c *Config) Read(fn func(*Config)) {
	c.mu.Lock()
	fn(c)
	c.mu.Unlock()
}
