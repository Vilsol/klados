package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func makeLoader(t *testing.T) (*Loader, string) {
	t.Helper()
	dir := t.TempDir()
	loader, err := NewLoader(dir)
	testza.AssertNoError(t, err)
	return loader, dir
}

func writeManifest(t *testing.T, dir, name string, manifest map[string]any) string {
	t.Helper()
	pluginDir := filepath.Join(dir, name)
	testza.AssertNoError(t, os.MkdirAll(pluginDir, 0o755))
	data, _ := json.Marshal(manifest)
	p := filepath.Join(pluginDir, "manifest.json")
	testza.AssertNoError(t, os.WriteFile(p, data, 0o644))
	return pluginDir
}

func validManifest() map[string]any {
	return map[string]any{
		"schemaVersion":  1,
		"name":           "test-plugin",
		"version":        "1.0.0",
		"displayName":    "Test Plugin",
		"minHostVersion": "1.0.0",
	}
}

func TestLoaderEmptyDir(t *testing.T) {
	loader, _ := makeLoader(t)
	plugins, errs := loader.Load()
	testza.AssertLen(t, plugins, 0)
	testza.AssertLen(t, errs, 0)
}

func TestLoaderNonExistentDir(t *testing.T) {
	loader, err := NewLoader("/tmp/klados-nonexistent-plugins-dir-xyz")
	testza.AssertNoError(t, err)
	plugins, errs := loader.Load()
	testza.AssertLen(t, plugins, 0)
	testza.AssertLen(t, errs, 0)
}

func TestLoaderValidManifest(t *testing.T) {
	loader, dir := makeLoader(t)
	writeManifest(t, dir, "myplugin", validManifest())

	plugins, errs := loader.Load()
	testza.AssertLen(t, errs, 0)
	testza.AssertLen(t, plugins, 1)
	testza.AssertEqual(t, "test-plugin", plugins[0].Manifest.Name)
	testza.AssertEqual(t, "1.0.0", plugins[0].Manifest.Version)
}

func TestLoaderMissingRequiredField(t *testing.T) {
	loader, dir := makeLoader(t)
	m := validManifest()
	delete(m, "name")
	writeManifest(t, dir, "myplugin", m)

	plugins, errs := loader.Load()
	testza.AssertLen(t, plugins, 0)
	testza.AssertLen(t, errs, 1)
	testza.AssertContains(t, errs[0].Error(), "validating")
}

func TestLoaderWrongSchemaVersion(t *testing.T) {
	loader, dir := makeLoader(t)
	m := validManifest()
	m["schemaVersion"] = 2
	writeManifest(t, dir, "myplugin", m)

	plugins, errs := loader.Load()
	testza.AssertLen(t, plugins, 0)
	testza.AssertLen(t, errs, 1)
}

func TestLoaderMinHostVersionTooNew(t *testing.T) {
	loader, dir := makeLoader(t)
	m := validManifest()
	m["minHostVersion"] = "99.0.0"
	writeManifest(t, dir, "myplugin", m)

	plugins, errs := loader.Load()
	testza.AssertLen(t, plugins, 0)
	testza.AssertLen(t, errs, 1)
	testza.AssertContains(t, errs[0].Error(), "requires host")
}

func TestLoaderMissingDescriptorFile(t *testing.T) {
	loader, dir := makeLoader(t)
	m := validManifest()
	m["extensions"] = map[string]any{
		"descriptors": []string{"descriptors/missing.yaml"},
	}
	writeManifest(t, dir, "myplugin", m)

	plugins, errs := loader.Load()
	testza.AssertLen(t, plugins, 0)
	testza.AssertLen(t, errs, 1)
	testza.AssertContains(t, errs[0].Error(), "loading descriptor")
}

func TestLoaderDescriptorLoaded(t *testing.T) {
	loader, dir := makeLoader(t)
	pluginDir := filepath.Join(dir, "myplugin")
	_ = os.MkdirAll(filepath.Join(pluginDir, "descriptors"), 0o755)
	_ = os.WriteFile(filepath.Join(pluginDir, "manifest.json"), func() []byte {
		m := validManifest()
		m["extensions"] = map[string]any{
			"descriptors": []string{"descriptors/cert.yaml"},
		}
		b, _ := json.Marshal(m)
		return b
	}(), 0o644)
	_ = os.WriteFile(filepath.Join(pluginDir, "descriptors", "cert.yaml"), []byte(`
group: cert-manager.io
version: v1
resource: certificates
kind: Certificate
columns:
  - name: Name
    expr: "metadata.name"
    renderType: text
`), 0o644)

	plugins, errs := loader.Load()
	testza.AssertLen(t, errs, 0)
	testza.AssertLen(t, plugins, 1)
	testza.AssertLen(t, plugins[0].Descriptors, 1)
	testza.AssertEqual(t, "cert-manager.io", plugins[0].Descriptors[0].Group)
}

func TestLoaderPartialFailure(t *testing.T) {
	loader, dir := makeLoader(t)
	writeManifest(t, dir, "good-plugin", validManifest())
	bad := validManifest()
	bad["minHostVersion"] = "99.0.0"
	writeManifest(t, dir, "bad-plugin", bad)

	plugins, errs := loader.Load()
	testza.AssertLen(t, errs, 1)
	testza.AssertLen(t, plugins, 1)
	testza.AssertEqual(t, "test-plugin", plugins[0].Manifest.Name)
}
