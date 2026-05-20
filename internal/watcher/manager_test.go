package watcher_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/resource"
	"github.com/Vilsol/klados/internal/watcher"
)

type fakeProvider struct{}

func (f *fakeProvider) GetConnection(_ string) (*cluster.Connection, error) {
	return nil, fmt.Errorf("no connection")
}

func newTestManager(emit func(string, any)) *watcher.WatchManager {
	return watcher.NewWatchManager(
		&fakeProvider{},
		resource.NewEnricherRegistry(),
		emit,
		context.Background(),
	)
}

func TestWatchManager_StartStopLifecycle(t *testing.T) {
	mgr := newTestManager(func(string, any) {})

	err := mgr.StartWatch("ctx", "core.v1.pods", "default", "")
	testza.AssertNotNil(t, err) // expected: fakeProvider returns error

	mgr.StopAll()
}

func TestWatchManager_GraceTimerCancelled(t *testing.T) {
	var mu sync.Mutex
	emitted := 0
	mgr := newTestManager(func(_ string, _ any) {
		mu.Lock()
		emitted++
		mu.Unlock()
	})

	// StopWatch on a non-existent watch should be a no-op
	mgr.StopWatch("ctx", "core.v1.pods", "default")

	// Give grace timer time to not fire (it wasn't started)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	testza.AssertEqual(t, 0, emitted)
	mu.Unlock()
}

func TestWatchManager_StopAll(t *testing.T) {
	mgr := newTestManager(func(string, any) {})
	// StopAll on empty manager is safe
	mgr.StopAll()
}

func TestWatchManager_SyncEventConstants(t *testing.T) {
	testza.AssertEqual(t, "SYNC_START", watcher.EventSyncStart)
	testza.AssertEqual(t, "SYNC_END", watcher.EventSyncEnd)
}

type fakeVirtualSource struct {
	gotCtxName string
	gotNS      string
	gotRV      string
	emit       func(string, any)
	stopped    atomic.Bool
	wantErr    error
}

func (f *fakeVirtualSource) Watch(_ context.Context, contextName, namespace, rv string, emit func(string, any)) (func(), error) {
	if f.wantErr != nil {
		return nil, f.wantErr
	}
	f.gotCtxName = contextName
	f.gotNS = namespace
	f.gotRV = rv
	f.emit = emit
	return func() { f.stopped.Store(true) }, nil
}

func TestWatchManager_RegisterVirtual_Dispatch(t *testing.T) {
	var mu sync.Mutex
	emitted := map[string]any{}
	mgr := newTestManager(func(name string, payload any) {
		mu.Lock()
		emitted[name] = payload
		mu.Unlock()
	})

	src := &fakeVirtualSource{}
	mgr.RegisterVirtual("helm.v1.releases", src)

	err := mgr.StartWatch("my-ctx", "helm.v1.releases", "ns1", "rv-1")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "my-ctx", src.gotCtxName)
	testza.AssertEqual(t, "ns1", src.gotNS)
	testza.AssertEqual(t, "rv-1", src.gotRV)

	src.emit("watch:my-ctx:helm.v1.releases:ns1", map[string]string{"hello": "world"})
	mu.Lock()
	payload, ok := emitted["watch:my-ctx:helm.v1.releases:ns1"]
	mu.Unlock()
	testza.AssertTrue(t, ok)
	testza.AssertNotNil(t, payload)

	// StopAll must invoke the virtual source's stop callback.
	mgr.StopAll()
	testza.AssertTrue(t, src.stopped.Load())
}

func TestWatchManager_StopWatch_VirtualStopAfterGrace(t *testing.T) {
	// The grace-period callback path is the trickier teardown — verify it
	// invokes the virtual source's stop callback once the timer fires.
	orig := watcher.GracePeriodForTest()
	watcher.SetGracePeriodForTest(20 * time.Millisecond)
	defer watcher.SetGracePeriodForTest(orig)

	mgr := newTestManager(func(string, any) {})
	src := &fakeVirtualSource{}
	mgr.RegisterVirtual("helm.v1.releases", src)

	testza.AssertNoError(t, mgr.StartWatch("ctx", "helm.v1.releases", "ns", ""))
	mgr.StopWatch("ctx", "helm.v1.releases", "ns")

	// Stop callback not invoked yet — grace timer is still running.
	testza.AssertFalse(t, src.stopped.Load())

	// Wait for the grace period plus a small buffer.
	time.Sleep(80 * time.Millisecond)
	testza.AssertTrue(t, src.stopped.Load())
}

func TestWatchManager_MultipleVirtuals_NoConflict(t *testing.T) {
	mgr := newTestManager(func(string, any) {})
	a := &fakeVirtualSource{}
	b := &fakeVirtualSource{}
	mgr.RegisterVirtual("helm.v1.releases", a)
	mgr.RegisterVirtual("flux.v1.releases", b)

	testza.AssertNoError(t, mgr.StartWatch("ctx", "helm.v1.releases", "ns", ""))
	testza.AssertNoError(t, mgr.StartWatch("ctx", "flux.v1.releases", "ns", ""))

	testza.AssertEqual(t, "ns", a.gotNS)
	testza.AssertEqual(t, "ns", b.gotNS)

	mgr.StopAll()
	testza.AssertTrue(t, a.stopped.Load())
	testza.AssertTrue(t, b.stopped.Load())
}

func TestWatchManager_StartWatch_WithOptions_BackCompat(t *testing.T) {
	// Existing callers using empty-options form continue to compile and run.
	mgr := newTestManager(func(string, any) {})
	err := mgr.StartWatch("ctx", "core.v1.pods", "default", "", watcher.WatchOptions{FieldSelector: "metadata.name=foo"})
	// fakeProvider returns error — we're only verifying signature compat.
	testza.AssertNotNil(t, err)
}
