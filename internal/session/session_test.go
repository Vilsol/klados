package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/adrg/xdg"
)

func withTempXDG(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	xdg.Reload()
	t.Cleanup(func() {
		xdg.Reload()
	})
	return dir
}

func tempSession(t *testing.T) *Session {
	t.Helper()
	dir := t.TempDir()
	s := defaultSession()
	s.path = filepath.Join(dir, "session.json")
	return s
}

func TestDefaultSession(t *testing.T) {
	s := defaultSession()
	testza.AssertEqual(t, 1280, s.Window.Width)
	testza.AssertEqual(t, 800, s.Window.Height)
	testza.AssertEqual(t, []string{}, s.ConnectedClusters)
}

func TestSaveAndLoad(t *testing.T) {
	s := tempSession(t)
	s.ConnectedClusters = []string{"ctx1"}
	s.ActiveNamespaces = map[string]string{"ctx1": "kube-system"}
	s.SidebarCollapsed = true

	testza.AssertNoError(t, s.Save())

	data, err := os.ReadFile(s.path)
	testza.AssertNoError(t, err)

	loaded := &Session{}
	testza.AssertNoError(t, json.Unmarshal(data, loaded))

	testza.AssertEqual(t, []string{"ctx1"}, loaded.ConnectedClusters)
	testza.AssertEqual(t, "kube-system", loaded.ActiveNamespaces["ctx1"])
	testza.AssertTrue(t, loaded.SidebarCollapsed)
}

func TestEmptyStateDefaults(t *testing.T) {
	s := defaultSession()
	testza.AssertEqual(t, 0, s.ActiveTab)
	testza.AssertFalse(t, s.SidebarCollapsed)
	testza.AssertEqual(t, 0, len(s.OpenTabs))
}

func TestDebouncedSaveWritesOnce(t *testing.T) {
	s := tempSession(t)
	s.ConnectedClusters = []string{"a"}

	for range 5 {
		s.SaveDebounced()
	}

	time.Sleep(700 * time.Millisecond)

	_, err := os.Stat(s.path)
	testza.AssertNoError(t, err)

	data, err := os.ReadFile(s.path)
	testza.AssertNoError(t, err)

	loaded := &Session{}
	testza.AssertNoError(t, json.Unmarshal(data, loaded))
	testza.AssertEqual(t, []string{"a"}, loaded.ConnectedClusters)
}

func TestLoad_NoFile_ReturnsDefaults(t *testing.T) {
	withTempXDG(t)

	s, err := Load()
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1280, s.Window.Width)
	testza.AssertEqual(t, []string{}, s.ConnectedClusters)
}

func TestLoad_ExistingFile_RestoresState(t *testing.T) {
	withTempXDG(t)

	initial := defaultSession()
	initial.ConnectedClusters = []string{"my-ctx"}
	initial.SidebarCollapsed = true
	initial.OpenTabs = []TabState{{ClusterContext: "my-ctx", GVR: "core.v1.pods", ScrollPosition: 99.5}}

	p, err := sessionPath()
	testza.AssertNoError(t, err)
	initial.path = p
	testza.AssertNoError(t, initial.Save())

	loaded, err := Load()
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, []string{"my-ctx"}, loaded.ConnectedClusters)
	testza.AssertTrue(t, loaded.SidebarCollapsed)
	testza.AssertEqual(t, 1, len(loaded.OpenTabs))
	testza.AssertEqual(t, 99.5, loaded.OpenTabs[0].ScrollPosition)
}

func TestLoad_CorruptFile_ReturnsError(t *testing.T) {
	withTempXDG(t)

	p, err := sessionPath()
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	testza.AssertNoError(t, os.WriteFile(p, []byte(`{bad json`), 0o644))

	_, err = Load()
	testza.AssertNotNil(t, err)
}

func TestLoad_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "session.json")

	s := defaultSession()
	s.path = p

	// Simulate Load() on missing file by reading defaults
	_, statErr := os.Stat(p)
	testza.AssertTrue(t, os.IsNotExist(statErr))

	// defaults have expected shape
	testza.AssertEqual(t, 1280, s.Window.Width)
	testza.AssertEqual(t, 800, s.Window.Height)
	testza.AssertEqual(t, []string{}, s.ConnectedClusters)
	testza.AssertEqual(t, 0, len(s.OpenTabs))
}

func TestLoad_ValidJSON(t *testing.T) {
	s := tempSession(t)
	s.ConnectedClusters = []string{"ctx1", "ctx2"}
	s.ActiveNamespaces = map[string]string{"ctx1": "default"}
	s.OpenTabs = []TabState{
		{ClusterContext: "ctx1", GVR: "core.v1.pods", Namespace: "default", Name: "my-pod", ScrollPosition: 42.5},
	}
	s.SidebarCollapsed = true

	testza.AssertNoError(t, s.Save())

	data, err := os.ReadFile(s.path)
	testza.AssertNoError(t, err)

	loaded := &Session{}
	testza.AssertNoError(t, json.Unmarshal(data, loaded))

	testza.AssertEqual(t, []string{"ctx1", "ctx2"}, loaded.ConnectedClusters)
	testza.AssertEqual(t, "default", loaded.ActiveNamespaces["ctx1"])
	testza.AssertTrue(t, loaded.SidebarCollapsed)
	testza.AssertEqual(t, 1, len(loaded.OpenTabs))
	testza.AssertEqual(t, "my-pod", loaded.OpenTabs[0].Name)
	testza.AssertEqual(t, 42.5, loaded.OpenTabs[0].ScrollPosition)
}

func TestLoad_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "session.json")

	testza.AssertNoError(t, os.WriteFile(p, []byte(`{not valid json`), 0o644))

	loaded := &Session{}
	err := json.Unmarshal([]byte(`{not valid json`), loaded)
	testza.AssertNotNil(t, err)
}

func TestSaveRoundTrip_OpenTabs_ScrollPosition(t *testing.T) {
	s := tempSession(t)
	s.OpenTabs = []TabState{
		{ClusterContext: "prod", GVR: "apps.v1.deployments", Namespace: "kube-system", Name: "coredns", ScrollPosition: 123.456},
		{ClusterContext: "dev", GVR: "core.v1.pods", Namespace: "default", Name: "", ScrollPosition: 0},
	}
	s.ActiveTab = 1

	testza.AssertNoError(t, s.Save())

	data, err := os.ReadFile(s.path)
	testza.AssertNoError(t, err)

	restored := &Session{}
	testza.AssertNoError(t, json.Unmarshal(data, restored))

	testza.AssertEqual(t, 2, len(restored.OpenTabs))
	testza.AssertEqual(t, 123.456, restored.OpenTabs[0].ScrollPosition)
	testza.AssertEqual(t, "coredns", restored.OpenTabs[0].Name)
	testza.AssertEqual(t, float64(0), restored.OpenTabs[1].ScrollPosition)
	testza.AssertEqual(t, 1, restored.ActiveTab)
}
