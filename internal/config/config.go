package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"github.com/sasha-s/go-deadlock"

	"github.com/adrg/xdg"
)

type MetricsConfig struct {
	PrometheusURL string `json:"prometheusUrl,omitempty"`
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

type Config struct {
	Theme                 string                     `json:"theme"`
	KubeconfigPaths       []string                   `json:"kubeconfigPaths"`
	TerminalWebGL         bool                       `json:"terminalWebGL"`
	DisabledPlugins       []string                   `json:"disabledPlugins,omitempty"`
	InsecureRegistries    []string                   `json:"insecureRegistries,omitempty"`
	InsecureSkipTLSVerify bool                       `json:"insecureSkipTLSVerify,omitempty"`
	Metrics               map[string]*MetricsConfig  `json:"metrics,omitempty"`
	ColumnPrefs           map[string]*GVRColumnPrefs `json:"columnPrefs,omitempty"`
	CompactRows           bool                       `json:"compactRows,omitempty"`

	mu   deadlock.Mutex
	path string
}

func DefaultConfig() *Config {
	return &Config{
		Theme:           "system",
		KubeconfigPaths: []string{},
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
	return cfg, nil
}

func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

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

	return os.WriteFile(c.path, data, 0o644)
}

func (c *Config) Update(fn func(*Config)) error {
	c.mu.Lock()
	fn(c)
	c.mu.Unlock()
	return c.Save()
}
