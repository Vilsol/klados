package volumebrowser

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/slox"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/config"
)

// fakeMultiProvider returns a different connection per context name.
type fakeMultiProvider struct {
	conns map[string]*cluster.Connection
	err   error
}

func (f *fakeMultiProvider) GetConnection(name string) (*cluster.Connection, error) {
	if f.err != nil {
		return nil, f.err
	}
	c, ok := f.conns[name]
	if !ok {
		return nil, errors.New("context not found: " + name)
	}
	return c, nil
}

func testCtx() context.Context {
	return slox.Into(context.Background(), slog.Default())
}

func TestManager_Spawn_TracksEntry(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	mgr := NewManager(testCtx(), &fakeMultiProvider{conns: map[string]*cluster.Connection{"ctx1": conn}}, "session-1")

	pod, err := mgr.Spawn(context.Background(), SpawnRequest{
		ContextName: "ctx1", Namespace: "default", PVCName: "data",
	}, config.VolumeBrowserConfig{})
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, pod)

	testza.AssertLen(t, mgr.ListManaged("ctx1"), 1)
	got := mgr.ListManaged("ctx1")
	testza.AssertEqual(t, pod.ID, got[0].ID)
}

func TestManager_Spawn_CollisionReturnsError(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	mgr := NewManager(testCtx(), &fakeMultiProvider{conns: map[string]*cluster.Connection{"ctx1": conn}}, "session-1")

	_, err := mgr.Spawn(context.Background(), SpawnRequest{
		ContextName: "ctx1", Namespace: "default", PVCName: "data",
	}, config.VolumeBrowserConfig{})
	testza.AssertNoError(t, err)

	_, err = mgr.Spawn(context.Background(), SpawnRequest{
		ContextName: "ctx1", Namespace: "default", PVCName: "data",
	}, config.VolumeBrowserConfig{})
	testza.AssertNotNil(t, err)
	testza.AssertTrue(t, errors.Is(err, ErrCollision))
}

func TestManager_Stop_DeletesPodAndRemovesFromTracker(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	mgr := NewManager(testCtx(), &fakeMultiProvider{conns: map[string]*cluster.Connection{"ctx1": conn}}, "session-1")

	pod, err := mgr.Spawn(context.Background(), SpawnRequest{
		ContextName: "ctx1", Namespace: "default", PVCName: "data",
	}, config.VolumeBrowserConfig{})
	testza.AssertNoError(t, err)

	err = mgr.Stop(context.Background(), pod.ID)
	testza.AssertNoError(t, err)

	testza.AssertLen(t, mgr.ListManaged("ctx1"), 0)

	_, getErr := conn.Dynamic.Resource(podGVR).Namespace("default").Get(context.Background(), pod.PodName, metav1.GetOptions{})
	testza.AssertNotNil(t, getErr)
}

func TestManager_Stop_UnknownID(t *testing.T) {
	mgr := NewManager(testCtx(), &fakeMultiProvider{conns: map[string]*cluster.Connection{}}, "session-1")
	err := mgr.Stop(context.Background(), "nope")
	testza.AssertNotNil(t, err)
}

func TestManager_StopForContext_OnlyAffectsThatContext(t *testing.T) {
	pvc1 := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	pvc2 := boundPVC("data", "default", "pv-2", []string{"ReadWriteOnce"})
	conn1 := connWithObjs(pvc1)
	conn2 := connWithObjs(pvc2)

	mgr := NewManager(testCtx(), &fakeMultiProvider{conns: map[string]*cluster.Connection{"ctx1": conn1, "ctx2": conn2}}, "session-1")

	_, err := mgr.Spawn(context.Background(), SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: "data"}, config.VolumeBrowserConfig{})
	testza.AssertNoError(t, err)
	_, err = mgr.Spawn(context.Background(), SpawnRequest{ContextName: "ctx2", Namespace: "default", PVCName: "data"}, config.VolumeBrowserConfig{})
	testza.AssertNoError(t, err)

	testza.AssertNoError(t, mgr.StopForContext(context.Background(), "ctx1"))
	testza.AssertLen(t, mgr.ListManaged("ctx1"), 0)
	testza.AssertLen(t, mgr.ListManaged("ctx2"), 1)
}

func TestManager_StopAll_DrainsAcrossContexts(t *testing.T) {
	pvc1 := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	pvc2 := boundPVC("data", "default", "pv-2", []string{"ReadWriteOnce"})
	conn1 := connWithObjs(pvc1)
	conn2 := connWithObjs(pvc2)

	mgr := NewManager(testCtx(), &fakeMultiProvider{conns: map[string]*cluster.Connection{"ctx1": conn1, "ctx2": conn2}}, "session-1")

	_, _ = mgr.Spawn(context.Background(), SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: "data"}, config.VolumeBrowserConfig{})
	_, _ = mgr.Spawn(context.Background(), SpawnRequest{ContextName: "ctx2", Namespace: "default", PVCName: "data"}, config.VolumeBrowserConfig{})

	testza.AssertNoError(t, mgr.StopAll(context.Background()))
	testza.AssertLen(t, mgr.ListManaged(""), 0)
}

func TestManager_AttachTab(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	mgr := NewManager(testCtx(), &fakeMultiProvider{conns: map[string]*cluster.Connection{"ctx1": conn}}, "session-1")

	pod, _ := mgr.Spawn(context.Background(), SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: "data"}, config.VolumeBrowserConfig{})
	testza.AssertNoError(t, mgr.AttachTab(pod.ID, "tab-7"))

	list := mgr.ListManaged("ctx1")
	testza.AssertEqual(t, "tab-7", list[0].TerminalTabID)

	testza.AssertNotNil(t, mgr.AttachTab("missing", "x"))
}

func TestManager_Spawn_OverridesApplied(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	mgr := NewManager(testCtx(), &fakeMultiProvider{conns: map[string]*cluster.Connection{"ctx1": conn}}, "session-1")

	img := "busybox:latest"
	ro := true
	pod, err := mgr.Spawn(context.Background(), SpawnRequest{
		ContextName: "ctx1", Namespace: "default", PVCName: "data",
		Overrides: &SpawnOverrides{Image: &img, ReadOnly: &ro},
	}, config.VolumeBrowserConfig{Image: "alpine:edge"})
	testza.AssertNoError(t, err)

	created, _ := conn.Dynamic.Resource(podGVR).Namespace("default").Get(context.Background(), pod.PodName, metav1.GetOptions{})
	cs, _, _ := containerMap(created)
	testza.AssertEqual(t, "busybox:latest", cs["image"])
}

func containerMap(created interface {
	UnstructuredContent() map[string]any
}) (map[string]any, bool, error) {
	obj := created.UnstructuredContent()
	spec, _ := obj["spec"].(map[string]any)
	containers, _ := spec["containers"].([]any)
	return containers[0].(map[string]any), true, nil
}

func TestManager_ScanOrphans(t *testing.T) {
	orphan := orphanPod("stale", "default", "pvc-old", "session-old")
	conn := connWithObjs(orphan)
	mgr := NewManagerWithIdentity(testCtx(), &fakeMultiProvider{conns: map[string]*cluster.Connection{"ctx1": conn}}, "session-new", "host-me", "user-me")

	orphans, err := mgr.ScanOrphans(context.Background(), "ctx1")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, orphans, 1)
	testza.AssertEqual(t, "pvc-old", orphans[0].PVCName)
}

func TestManager_Spawn_ConcurrentSamePVC_OneWinsOneCollides(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	mgr := NewManager(testCtx(), &fakeMultiProvider{conns: map[string]*cluster.Connection{"ctx1": conn}}, "session-1")

	var wg sync.WaitGroup
	var successes, collisions int32
	start := make(chan struct{})

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := mgr.Spawn(context.Background(), SpawnRequest{
				ContextName: "ctx1", Namespace: "default", PVCName: "data",
			}, config.VolumeBrowserConfig{})
			if err == nil {
				atomic.AddInt32(&successes, 1)
			} else if errors.Is(err, ErrCollision) {
				atomic.AddInt32(&collisions, 1)
			} else {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	testza.AssertEqual(t, int32(1), atomic.LoadInt32(&successes))
	testza.AssertEqual(t, int32(1), atomic.LoadInt32(&collisions))

	testza.AssertLen(t, mgr.ListManaged("ctx1"), 1)

	podList, err := conn.Dynamic.Resource(podGVR).Namespace("default").List(context.Background(), metav1.ListOptions{})
	testza.AssertNoError(t, err)
	testza.AssertLen(t, podList.Items, 1)
}

type flakyProvider struct {
	conn  *cluster.Connection
	calls int32
}

func (f *flakyProvider) GetConnection(name string) (*cluster.Connection, error) {
	n := atomic.AddInt32(&f.calls, 1)
	// Fail only on the second call (the first Stop attempt after the spawn call).
	if n == 2 {
		return nil, errors.New("transient connection failure")
	}
	return f.conn, nil
}

func TestManager_Stop_TransientConnectionErrorKeepsEntry(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	fp := &flakyProvider{conn: conn}
	mgr := NewManager(testCtx(), fp, "session-1")

	pod, err := mgr.Spawn(context.Background(), SpawnRequest{
		ContextName: "ctx1", Namespace: "default", PVCName: "data",
	}, config.VolumeBrowserConfig{})
	testza.AssertNoError(t, err)

	// Second GetConnection call fails.
	err = mgr.Stop(context.Background(), pod.ID)
	testza.AssertNotNil(t, err)
	testza.AssertLen(t, mgr.ListManaged("ctx1"), 1, "entry must remain after transient failure")

	// Third call succeeds.
	err = mgr.Stop(context.Background(), pod.ID)
	testza.AssertNoError(t, err)
	testza.AssertLen(t, mgr.ListManaged("ctx1"), 0)
}
