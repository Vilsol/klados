package portforward

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/slox"

	"github.com/Vilsol/klados/internal/cluster"
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
	ctx := slox.Into(context.Background(), slog.Default())
	m := NewManager(p, func(_ string, _ any) {}, ctx)
	return m
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
