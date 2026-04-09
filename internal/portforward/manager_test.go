package portforward

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/slox"
	"github.com/adrg/xdg"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/config"
)

type fakeProvider struct {
	mu   sync.Mutex
	conn *cluster.Connection
	err  error
}

func (f *fakeProvider) GetConnection(_ string) (*cluster.Connection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return f.conn, nil
}

// blockingTunnel blocks until ctx is cancelled, then calls onReady immediately.
// Use this to simulate a live tunnel.
func blockingTunnel(ctx context.Context, _ *cluster.Connection, _, _ string, localPort, _ int, onReady func(uint16)) error {
	onReady(uint16(localPort))
	<-ctx.Done()
	return ctx.Err()
}

// failingTunnel returns an error immediately — simulates a connection failure.
func failingTunnel(_ context.Context, _ *cluster.Connection, _, _ string, _, _ int, _ func(uint16)) error {
	return fmt.Errorf("connection refused")
}

func newTestManager(p ConnectionProvider) *Manager {
	return newTestManagerWithConfig(p, config.DefaultConfig())
}

func newTestManagerWithConfig(p ConnectionProvider, cfg *config.Config) *Manager {
	ctx := slox.Into(context.Background(), slog.Default())
	m := NewManager(p, cfg, func(_ string, _ any) {}, ctx)
	return m
}

func newTestManagerWithEvents(p ConnectionProvider, cfg *config.Config, emit func(string, any)) *Manager {
	ctx := slox.Into(context.Background(), slog.Default())
	return NewManager(p, cfg, emit, ctx)
}

func TestStartForward_UnknownContext(t *testing.T) {
	m := newTestManager(&fakeProvider{err: fmt.Errorf("not connected")})
	_, err := m.StartForward(ForwardSpec{
		ContextName: "ctx1",
		TargetKind:  TargetKindPod,
		TargetName:  "my-pod",
		Namespace:   "default",
		RemotePort:  8080,
	})
	testza.AssertNotNil(t, err)
}

func TestStartForward_UniqueIDs(t *testing.T) {
	pod := makePod("my-pod", "Running", true, time.Now())
	m := newTestManager(&fakeProvider{conn: fakeConnWithPods(pod)})
	m.tunnel = blockingTunnel

	spec := ForwardSpec{
		ContextName: "ctx1",
		TargetKind:  TargetKindPod,
		TargetName:  "my-pod",
		Namespace:   "default",
		RemotePort:  8080,
	}
	s1, err := m.StartForward(spec)
	testza.AssertNoError(t, err)
	s2, err := m.StartForward(spec)
	testza.AssertNoError(t, err)

	testza.AssertNotEqual(t, s1.ID, s2.ID)
	testza.AssertLen(t, s1.ID, 32)
	testza.AssertLen(t, s2.ID, 32)

	m.StopAll()
}

func TestListForwards_Empty(t *testing.T) {
	m := newTestManager(&fakeProvider{conn: fakeConnWithPods()})
	result := m.ListForwards("ctx1")
	testza.AssertNotNil(t, result)
	testza.AssertLen(t, result, 0)
}

func TestListForwards_FiltersContext(t *testing.T) {
	pod := makePod("my-pod", "Running", true, time.Now())
	m := newTestManager(&fakeProvider{conn: fakeConnWithPods(pod)})
	m.tunnel = blockingTunnel

	_, err := m.StartForward(ForwardSpec{
		ContextName: "ctx1",
		TargetKind:  TargetKindPod,
		TargetName:  "my-pod",
		Namespace:   "default",
		RemotePort:  8080,
	})
	testza.AssertNoError(t, err)
	_, err = m.StartForward(ForwardSpec{
		ContextName: "ctx2",
		TargetKind:  TargetKindPod,
		TargetName:  "my-pod",
		Namespace:   "default",
		RemotePort:  8080,
	})
	testza.AssertNoError(t, err)

	ctx1 := m.ListForwards("ctx1")
	testza.AssertLen(t, ctx1, 1)
	testza.AssertEqual(t, "ctx1", ctx1[0].ContextName)

	all := m.ListForwards("")
	testza.AssertLen(t, all, 2)

	m.StopAll()
}

func TestStopForward_Cleanup(t *testing.T) {
	pod := makePod("my-pod", "Running", true, time.Now())
	m := newTestManager(&fakeProvider{conn: fakeConnWithPods(pod)})
	m.tunnel = blockingTunnel

	spec, err := m.StartForward(ForwardSpec{
		ContextName: "ctx1",
		TargetKind:  TargetKindPod,
		TargetName:  "my-pod",
		Namespace:   "default",
		RemotePort:  8080,
	})
	testza.AssertNoError(t, err)

	err = m.StopForward(spec.ID)
	testza.AssertNoError(t, err)

	testza.AssertLen(t, m.ListForwards("ctx1"), 0)
}

func TestStopForward_NonExistent(t *testing.T) {
	m := newTestManager(&fakeProvider{conn: fakeConnWithPods()})
	err := m.StopForward("nonexistent")
	testza.AssertNotNil(t, err)
}

func TestStopAll_ClearsAll(t *testing.T) {
	pod := makePod("my-pod", "Running", true, time.Now())
	m := newTestManager(&fakeProvider{conn: fakeConnWithPods(pod)})
	m.tunnel = blockingTunnel

	for range 3 {
		_, err := m.StartForward(ForwardSpec{
			ContextName: "ctx1",
			TargetKind:  TargetKindPod,
			TargetName:  "my-pod",
			Namespace:   "default",
			RemotePort:  8080,
		})
		testza.AssertNoError(t, err)
	}

	m.mu.Lock()
	count := len(m.forwards)
	m.mu.Unlock()
	testza.AssertEqual(t, 3, count)

	m.StopAll()

	m.mu.Lock()
	count = len(m.forwards)
	m.mu.Unlock()
	testza.AssertEqual(t, 0, count)
}

func TestStartForward_StatusSetToReconnecting(t *testing.T) {
	pod := makePod("my-pod", "Running", true, time.Now())
	m := newTestManager(&fakeProvider{conn: fakeConnWithPods(pod)})
	m.tunnel = blockingTunnel

	spec, err := m.StartForward(ForwardSpec{
		ContextName: "ctx1",
		TargetKind:  TargetKindPod,
		TargetName:  "my-pod",
		Namespace:   "default",
		RemotePort:  8080,
	})
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, StatusReconnecting, spec.Status)

	m.StopAll()
}

func TestSaveForward_PersistsToConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	m := newTestManagerWithConfig(&fakeProvider{}, cfg)

	fwd := config.SavedPortForward{
		Namespace:  "default",
		Resource:   "pods/my-pod",
		TargetKind: "pod",
		TargetName: "my-pod",
		LocalPort:  8080,
		RemotePort: 80,
		Enabled:    true,
	}
	err := m.SaveForward("ctx1", fwd)
	testza.AssertNoError(t, err)

	saved := m.ListSavedForwards("ctx1")
	testza.AssertLen(t, saved, 1)
	testza.AssertNotEqual(t, "", saved[0].ID)
	testza.AssertEqual(t, "my-pod", saved[0].TargetName)
	testza.AssertEqual(t, true, saved[0].Enabled)
}

func TestRemoveSavedForward_DeletesEntry(t *testing.T) {
	cfg := config.DefaultConfig()
	m := newTestManagerWithConfig(&fakeProvider{}, cfg)

	fwd := config.SavedPortForward{
		Namespace:  "default",
		Resource:   "pods/my-pod",
		TargetKind: "pod",
		TargetName: "my-pod",
		LocalPort:  8080,
		RemotePort: 80,
		Enabled:    true,
	}
	testza.AssertNoError(t, m.SaveForward("ctx1", fwd))
	saved := m.ListSavedForwards("ctx1")
	testza.AssertLen(t, saved, 1)

	testza.AssertNoError(t, m.RemoveSavedForward("ctx1", saved[0].ID))
	testza.AssertLen(t, m.ListSavedForwards("ctx1"), 0)
}

func TestSetForwardEnabled_SkippedByReconnect(t *testing.T) {
	cfg := config.DefaultConfig()
	var started []string
	var mu sync.Mutex

	// Track StartForward calls via a tunnel that records which pod was targeted
	conn := fakeConnWithPods(makePod("my-pod", "Running", true, time.Now()))
	m := newTestManagerWithConfig(&fakeProvider{conn: conn}, cfg)
	m.tunnel = func(ctx context.Context, _ *cluster.Connection, _, podName string, _, _ int, onReady func(uint16)) error {
		mu.Lock()
		started = append(started, podName)
		mu.Unlock()
		onReady(0)
		<-ctx.Done()
		return ctx.Err()
	}

	fwd := config.SavedPortForward{
		Namespace:  "default",
		Resource:   "pods/my-pod",
		TargetKind: "pod",
		TargetName: "my-pod",
		LocalPort:  18080,
		RemotePort: 80,
		Enabled:    true,
	}
	testza.AssertNoError(t, m.SaveForward("ctx1", fwd))
	saved := m.ListSavedForwards("ctx1")

	testza.AssertNoError(t, m.SetForwardEnabled("ctx1", saved[0].ID, false))

	m.ReconnectSaved("ctx1")

	// Give goroutines time to start (they shouldn't, but allow a moment)
	time.Sleep(50 * time.Millisecond)
	m.StopAll()

	mu.Lock()
	count := len(started)
	mu.Unlock()
	testza.AssertEqual(t, 0, count)
}

func TestReconnectSaved_EmitsErrorOnFailure(t *testing.T) {
	cfg := config.DefaultConfig()

	var events []string
	var mu sync.Mutex
	emit := func(name string, _ any) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	}

	m := newTestManagerWithEvents(&fakeProvider{err: fmt.Errorf("not connected")}, cfg, emit)

	fwd := config.SavedPortForward{
		ID:         "test-id-1",
		Namespace:  "default",
		Resource:   "pods/my-pod",
		TargetKind: "pod",
		TargetName: "my-pod",
		LocalPort:  18081,
		RemotePort: 80,
		Enabled:    true,
	}
	testza.AssertNoError(t, m.SaveForward("ctx1", fwd))

	m.ReconnectSaved("ctx1")

	mu.Lock()
	evtCount := len(events)
	mu.Unlock()
	testza.AssertTrue(t, evtCount > 0)
}

func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	xdg.Reload()
	t.Cleanup(func() { xdg.Reload() })

	cfg, err := config.Load()
	testza.AssertNoError(t, err)

	m := newTestManagerWithConfig(&fakeProvider{}, cfg)

	fwd := config.SavedPortForward{
		Namespace:  "default",
		Resource:   "pods/my-pod",
		TargetKind: "pod",
		TargetName: "my-pod",
		LocalPort:  9090,
		RemotePort: 80,
		Enabled:    true,
	}
	testza.AssertNoError(t, m.SaveForward("ctx1", fwd))

	// Verify file was written
	cfgPath := dir + "/klados/config.json"
	_, statErr := os.Stat(cfgPath)
	testza.AssertNoError(t, statErr)

	// Reload from disk
	cfg2, err := config.Load()
	testza.AssertNoError(t, err)

	m2 := newTestManagerWithConfig(&fakeProvider{}, cfg2)
	saved := m2.ListSavedForwards("ctx1")
	testza.AssertLen(t, saved, 1)
	testza.AssertEqual(t, "my-pod", saved[0].TargetName)
	testza.AssertEqual(t, 9090, saved[0].LocalPort)
}
