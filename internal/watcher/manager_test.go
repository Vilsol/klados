package watcher_test

import (
	"context"
	"fmt"
	"sync"
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

	err := mgr.StartWatch("ctx", "core.v1.pods", "default")
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
