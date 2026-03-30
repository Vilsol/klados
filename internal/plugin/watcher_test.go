package plugin_test

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/klados/internal/plugin"
)

func TestPluginWatcher_DetectsFileChange(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "my-plugin")
	testza.AssertNil(t, os.Mkdir(pluginDir, 0o755))

	// Write initial manifest
	testza.AssertNil(t, os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte("{}"), 0o644))

	var reloads atomic.Int32
	w, err := plugin.NewPluginWatcher(t.Context(), func(name string) {
		if name == "my-plugin" {
			reloads.Add(1)
		}
	})
	testza.AssertNil(t, err)

	testza.AssertNil(t, w.Watch("my-plugin", pluginDir))
	w.Start()
	t.Cleanup(w.Stop)

	// Modify a file
	testza.AssertNil(t, os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"v":2}`), 0o644))

	// Wait for debounce + processing
	time.Sleep(400 * time.Millisecond)
	testza.AssertTrue(t, reloads.Load() >= 1)
}

func TestPluginWatcher_Debounce(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "my-plugin")
	testza.AssertNil(t, os.Mkdir(pluginDir, 0o755))

	var reloads atomic.Int32
	w, err := plugin.NewPluginWatcher(t.Context(), func(name string) {
		reloads.Add(1)
	})
	testza.AssertNil(t, err)

	testza.AssertNil(t, w.Watch("my-plugin", pluginDir))
	w.Start()
	t.Cleanup(w.Stop)

	// Write many files rapidly
	for i := range 10 {
		testza.AssertNil(t, os.WriteFile(filepath.Join(pluginDir, "file.js"), []byte{byte(i)}, 0o644))
		time.Sleep(20 * time.Millisecond)
	}

	// Wait for debounce to settle
	time.Sleep(400 * time.Millisecond)

	// Should be 1 reload, not 10
	testza.AssertTrue(t, reloads.Load() == 1)
}

func TestPluginWatcher_NewSubdir(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "my-plugin")
	testza.AssertNil(t, os.Mkdir(pluginDir, 0o755))

	var reloads atomic.Int32
	w, err := plugin.NewPluginWatcher(t.Context(), func(_ string) {
		reloads.Add(1)
	})
	testza.AssertNil(t, err)

	testza.AssertNil(t, w.Watch("my-plugin", pluginDir))
	w.Start()
	t.Cleanup(w.Stop)

	// Create a new subdir
	subDir := filepath.Join(pluginDir, "ui")
	testza.AssertNil(t, os.Mkdir(subDir, 0o755))
	time.Sleep(100 * time.Millisecond) // let watcher add the subdir

	// Write a file inside it
	testza.AssertNil(t, os.WriteFile(filepath.Join(subDir, "Component.js"), []byte("export default {}"), 0o644))
	time.Sleep(400 * time.Millisecond)

	testza.AssertTrue(t, reloads.Load() >= 1)
}

func TestPluginWatcher_Unwatch(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "my-plugin")
	testza.AssertNil(t, os.Mkdir(pluginDir, 0o755))
	testza.AssertNil(t, os.WriteFile(filepath.Join(pluginDir, "f.js"), []byte("1"), 0o644))

	var reloads atomic.Int32
	w, err := plugin.NewPluginWatcher(t.Context(), func(_ string) {
		reloads.Add(1)
	})
	testza.AssertNil(t, err)
	testza.AssertNil(t, w.Watch("my-plugin", pluginDir))
	w.Start()
	t.Cleanup(w.Stop)

	// Unwatch before any file change
	w.Unwatch(pluginDir)
	time.Sleep(50 * time.Millisecond)

	testza.AssertNil(t, os.WriteFile(filepath.Join(pluginDir, "f.js"), []byte("2"), 0o644))
	time.Sleep(400 * time.Millisecond)

	testza.AssertEqual(t, int32(0), reloads.Load())
}
