package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/adrg/xdg"
)

func withTempXDG(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	xdg.Reload()
	t.Cleanup(func() {
		xdg.Reload()
	})
}

func tempConfig(t *testing.T) *Config {
	t.Helper()
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.path = filepath.Join(dir, "config.json")
	return cfg
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	testza.AssertEqual(t, "system", cfg.Theme)
	testza.AssertEqual(t, []string{}, cfg.KubeconfigPaths)
}

func TestSaveAndLoad(t *testing.T) {
	cfg := tempConfig(t)
	cfg.Theme = "dark"
	cfg.KubeconfigPaths = []string{"/tmp/kube"}

	testza.AssertNoError(t, cfg.Save())

	loaded := &Config{path: cfg.path}
	data, err := os.ReadFile(cfg.path)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(data, loaded))

	testza.AssertEqual(t, "dark", loaded.Theme)
	testza.AssertEqual(t, []string{"/tmp/kube"}, loaded.KubeconfigPaths)
}

func TestSaveCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Theme: "light",
		path:  filepath.Join(dir, "nested", "config.json"),
	}
	testza.AssertNoError(t, cfg.Save())

	data, err := os.ReadFile(cfg.path)
	testza.AssertNoError(t, err)

	loaded := &Config{}
	testza.AssertNoError(t, json.Unmarshal(data, loaded))
	testza.AssertEqual(t, "light", loaded.Theme)
}

func TestCorruptJSONReturnsError(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	testza.AssertNoError(t, os.WriteFile(p, []byte("{invalid json"), 0o644))

	data, err := os.ReadFile(p)
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, json.Unmarshal(data, &Config{}))
}

func TestUpdateConcurrency(t *testing.T) {
	cfg := tempConfig(t)

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cfg.Update(func(c *Config) {
				c.Theme = "dark"
			})
		}()
	}
	wg.Wait()

	testza.AssertEqual(t, "dark", cfg.Theme)
}

func TestLoad_NoFile_ReturnsDefaults(t *testing.T) {
	withTempXDG(t)

	cfg, err := Load()
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "system", cfg.Theme)
	testza.AssertEqual(t, []string{}, cfg.KubeconfigPaths)
}

func TestLoad_ExistingFile_RestoresState(t *testing.T) {
	withTempXDG(t)

	// Write config
	initial := DefaultConfig()
	initial.Theme = "dark"
	initial.KubeconfigPaths = []string{"/home/user/.kube/config"}

	p, err := configPath()
	testza.AssertNoError(t, err)
	initial.path = p
	testza.AssertNoError(t, initial.Save())

	loaded, err := Load()
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "dark", loaded.Theme)
	testza.AssertEqual(t, []string{"/home/user/.kube/config"}, loaded.KubeconfigPaths)
}

func TestInsecureRegistries_RoundTrip(t *testing.T) {
	cfg := tempConfig(t)
	testza.AssertNil(t, cfg.InsecureRegistries)

	cfg.InsecureRegistries = []string{"localhost:5000", "registry.internal"}
	testza.AssertNoError(t, cfg.Save())

	data, err := os.ReadFile(cfg.path)
	testza.AssertNoError(t, err)

	loaded := &Config{}
	testza.AssertNoError(t, json.Unmarshal(data, loaded))
	testza.AssertEqual(t, []string{"localhost:5000", "registry.internal"}, loaded.InsecureRegistries)
}

func TestInsecureRegistries_NilOmitted(t *testing.T) {
	cfg := tempConfig(t)
	testza.AssertNoError(t, cfg.Save())

	data, err := os.ReadFile(cfg.path)
	testza.AssertNoError(t, err)
	testza.AssertFalse(t, strings.Contains(string(data), "insecureRegistries"))
}

func TestLoad_CorruptFile_ReturnsError(t *testing.T) {
	withTempXDG(t)

	p, err := configPath()
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	testza.AssertNoError(t, os.WriteFile(p, []byte(`{bad json`), 0o644))

	_, err = Load()
	testza.AssertNotNil(t, err)
}
