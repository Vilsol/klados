package cluster

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func writeTestKubeconfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config")

	content := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
    namespace: default
  name: test-context
- context:
    cluster: test-cluster
    user: test-user
    namespace: kube-system
  name: test-context-2
current-context: test-context
users:
- name: test-user
  user:
    token: fake-token
`
	testza.AssertNoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func TestLoadKubeconfigs(t *testing.T) {
	p := writeTestKubeconfig(t)
	t.Setenv("KUBECONFIG", p)

	var events []string
	var mu sync.Mutex
	emit := func(name string, data any) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	}

	mgr := NewManager(emit, nil, noopLogger())
	testza.AssertNoError(t, mgr.LoadKubeconfigs(nil))

	contexts := mgr.ListContexts()
	testza.AssertTrue(t, len(contexts) >= 2)

	names := make(map[string]bool)
	for _, c := range contexts {
		names[c.Name] = true
	}
	testza.AssertTrue(t, names["test-context"])
	testza.AssertTrue(t, names["test-context-2"])
}

func TestLoadKubeconfigs_ExtraPaths(t *testing.T) {
	p := writeTestKubeconfig(t)
	t.Setenv("KUBECONFIG", "")

	mgr := NewManager(nil, nil, noopLogger())
	testza.AssertNoError(t, mgr.LoadKubeconfigs([]string{p}))

	contexts := mgr.ListContexts()
	testza.AssertTrue(t, len(contexts) >= 2)
}

func TestDisconnectNonexistent(t *testing.T) {
	mgr := NewManager(func(string, any) {}, nil, noopLogger())
	err := mgr.Disconnect("test-context")
	testza.AssertNoError(t, err)
}

func TestGetConnectionNotConnected(t *testing.T) {
	mgr := NewManager(nil, nil, noopLogger())
	_, err := mgr.GetConnection("nonexistent")
	testza.AssertNotNil(t, err)
}

func TestDisconnectAllClearsConnections(t *testing.T) {
	var events []string
	var mu sync.Mutex
	emit := func(name string, _ any) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	}

	mgr := NewManager(emit, nil, noopLogger())
	// Inject fake connections directly (same package — private field accessible)
	ctx1cancel := func() {}
	ctx2cancel := func() {}
	mgr.connections["ctx1"] = &Connection{
		KubeContext: KubeContext{Name: "ctx1", Status: StatusConnected},
		cancel:      ctx1cancel,
	}
	mgr.connections["ctx2"] = &Connection{
		KubeContext: KubeContext{Name: "ctx2", Status: StatusConnected},
		cancel:      ctx2cancel,
	}

	testza.AssertNoError(t, mgr.DisconnectAll())
	testza.AssertEqual(t, 0, len(mgr.connections))

	_, err1 := mgr.GetConnection("ctx1")
	testza.AssertNotNil(t, err1)
	_, err2 := mgr.GetConnection("ctx2")
	testza.AssertNotNil(t, err2)
}

func TestIndependentConnectionsIsolated(t *testing.T) {
	mgr := NewManager(func(string, any) {}, nil, noopLogger())
	mgr.connections["ctx1"] = &Connection{
		KubeContext: KubeContext{Name: "ctx1", Status: StatusConnected},
		cancel:      func() {},
	}
	mgr.connections["ctx2"] = &Connection{
		KubeContext: KubeContext{Name: "ctx2", Status: StatusConnected},
		cancel:      func() {},
	}

	// Disconnecting ctx1 must not affect ctx2
	testza.AssertNoError(t, mgr.Disconnect("ctx1"))

	_, err := mgr.GetConnection("ctx1")
	testza.AssertNotNil(t, err)

	conn, err := mgr.GetConnection("ctx2")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "ctx2", conn.Name)
}

func TestConcurrentAccess(t *testing.T) {
	p := writeTestKubeconfig(t)
	t.Setenv("KUBECONFIG", p)

	mgr := NewManager(func(string, any) {}, nil, noopLogger())
	testza.AssertNoError(t, mgr.LoadKubeconfigs(nil))

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = mgr.ListContexts()
		}()
	}
	wg.Wait()
}
