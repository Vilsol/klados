package services

import (
	"context"
	"fmt"

	"github.com/Vilsol/klados/internal/config"
)

type ConfigService struct {
	ctx    context.Context
	config *config.Config
}

func NewConfigService(ctx context.Context, cfg *config.Config) *ConfigService {
	return &ConfigService{ctx: ctx, config: cfg}
}

func (c *ConfigService) GetTheme() string {
	return c.config.Theme
}

func (c *ConfigService) SetTheme(theme string) error {
	switch theme {
	case "dark", "light", "system":
	default:
		return fmt.Errorf("invalid theme: %q", theme)
	}

	return c.config.Update(func(cfg *config.Config) {
		cfg.Theme = theme
	})
}

func (c *ConfigService) GetTerminalWebGL() bool {
	return c.config.TerminalWebGL
}

func (c *ConfigService) SetTerminalWebGL(enabled bool) error {
	return c.config.Update(func(cfg *config.Config) {
		cfg.TerminalWebGL = enabled
	})
}

func (c *ConfigService) GetInsecureSkipTLSVerify() bool {
	return c.config.InsecureSkipTLSVerify
}

func (c *ConfigService) SetInsecureSkipTLSVerify(skip bool) error {
	return c.config.Update(func(cfg *config.Config) {
		cfg.InsecureSkipTLSVerify = skip
	})
}

func (c *ConfigService) GetColumnPrefs(gvr string) *config.GVRColumnPrefs {
	if c.config.ColumnPrefs == nil {
		return nil
	}
	return c.config.ColumnPrefs[gvr]
}

func (c *ConfigService) SetColumnPrefs(gvr string, prefs *config.GVRColumnPrefs) error {
	return c.config.Update(func(cfg *config.Config) {
		if cfg.ColumnPrefs == nil {
			cfg.ColumnPrefs = make(map[string]*config.GVRColumnPrefs)
		}
		cfg.ColumnPrefs[gvr] = prefs
	})
}

func (c *ConfigService) DeleteColumnPrefs(gvr string) error {
	return c.config.Update(func(cfg *config.Config) {
		delete(cfg.ColumnPrefs, gvr)
	})
}

func (c *ConfigService) GetCompactRows() bool {
	return c.config.CompactRows
}

func (c *ConfigService) SetCompactRows(compact bool) error {
	return c.config.Update(func(cfg *config.Config) {
		cfg.CompactRows = compact
	})
}

func (c *ConfigService) GetConfig() *config.Config {
	return c.config
}
