package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestPackUnpack_RoundTrip(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "manifest.json", `{
		"schemaVersion": 1,
		"name": "test-plugin",
		"version": "0.2.0",
		"displayName": "Test Plugin",
		"minHostVersion": "0.1.0"
	}`)
	writeFile(t, src, "ui/index.js", `export default {}`)
	writeFile(t, src, "descriptors/pods.yaml", `gvr: "core.v1.pods"`)

	archivePath, err := Pack(src, true)
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, strings.HasSuffix(archivePath, ".oci.tar.gz"))

	dest := t.TempDir()
	err = Unpack(archivePath, dest)
	testza.AssertNoError(t, err)

	pluginDir := filepath.Join(dest, "test-plugin")
	assertFileExists(t, pluginDir, "manifest.json")
	assertFileExists(t, pluginDir, "ui/index.js")
	assertFileExists(t, pluginDir, "descriptors/pods.yaml")

	manifestData, _ := os.ReadFile(filepath.Join(pluginDir, "manifest.json"))
	testza.AssertContains(t, string(manifestData), "test-plugin")
}

func TestPack_Uncompressed(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "manifest.json", `{
		"schemaVersion": 1,
		"name": "test-plugin",
		"version": "1.0.0",
		"displayName": "Test Plugin",
		"minHostVersion": "0.1.0"
	}`)

	archivePath, err := Pack(src, false)
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, strings.HasSuffix(archivePath, ".oci.tar"))

	dest := t.TempDir()
	err = Unpack(archivePath, dest)
	testza.AssertNoError(t, err)
	assertFileExists(t, filepath.Join(dest, "test-plugin"), "manifest.json")
}

func TestUnpack_GzipDetection(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "manifest.json", `{
		"schemaVersion": 1,
		"name": "detect-plugin",
		"version": "0.1.0",
		"displayName": "Detect Plugin",
		"minHostVersion": "0.1.0"
	}`)

	// Pack compressed, rename to .oci.tar (wrong extension) — should still work
	archivePath, err := Pack(src, true)
	testza.AssertNoError(t, err)
	renamed := strings.TrimSuffix(archivePath, ".gz")
	err = os.Rename(archivePath, renamed)
	testza.AssertNoError(t, err)

	dest := t.TempDir()
	err = Unpack(renamed, dest)
	testza.AssertNoError(t, err)
	assertFileExists(t, filepath.Join(dest, "detect-plugin"), "manifest.json")
}

func TestPack_WithWasm(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "manifest.json", `{
		"schemaVersion": 1,
		"name": "wasm-plugin",
		"version": "0.1.0",
		"displayName": "Wasm Plugin",
		"minHostVersion": "0.1.0",
		"extensions": {
			"enrichers": {
				"gvrs": ["core.v1.pods"],
				"wasm": "plugin.wasm"
			}
		}
	}`)
	writeFile(t, src, "plugin.wasm", "\x00asm\x01\x00\x00\x00") // fake wasm magic

	archivePath, err := Pack(src, true)
	testza.AssertNoError(t, err)

	dest := t.TempDir()
	err = Unpack(archivePath, dest)
	testza.AssertNoError(t, err)
	assertFileExists(t, filepath.Join(dest, "wasm-plugin"), "plugin.wasm")
}

func TestUnpack_InvalidArchive(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "bad*.tar")
	testza.AssertNoError(t, err)
	f.WriteString("not a tar file")
	f.Close()

	dest := t.TempDir()
	err = Unpack(f.Name(), dest)
	testza.AssertNotNil(t, err)
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		t.Fatal(err)
	}
}

func assertFileExists(t *testing.T, dir, rel string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file %s to exist: %v", path, err)
	}
}
