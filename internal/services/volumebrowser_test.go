package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/klados/internal/config"
	"github.com/Vilsol/klados/internal/volumebrowser"
)

// ---- fakes ----

type fakeVBManager struct {
	spawnCalls     int
	stopCalls      []string
	stopAllCalled  bool
	attachCalls    [][2]string
	listResult     []*volumebrowser.ManagedPod
	scanResult     []volumebrowser.OrphanPod
	scanErr        error
	cleanupCalls   []string
	cleanupErr     error
	findResult     *volumebrowser.ManagedPod
	spawnErr       error
	lastSpawnReq   volumebrowser.SpawnRequest
	lastResolvedCfg config.VolumeBrowserConfig
	spawnResult    *volumebrowser.ManagedPod
	callOrder      []string
	stopAllDelay   time.Duration
}

func (f *fakeVBManager) Spawn(ctx context.Context, req volumebrowser.SpawnRequest, resolved config.VolumeBrowserConfig) (*volumebrowser.ManagedPod, error) {
	f.spawnCalls++
	f.lastSpawnReq = req
	f.lastResolvedCfg = resolved
	f.callOrder = append(f.callOrder, "Spawn")
	if f.spawnErr != nil {
		return nil, f.spawnErr
	}
	if f.spawnResult != nil {
		return f.spawnResult, nil
	}
	return &volumebrowser.ManagedPod{
		ID:          "id-1",
		ContextName: req.ContextName,
		Namespace:   req.Namespace,
		PodName:     "klados-browser-xxx",
		PVCName:     req.PVCName,
		CreatedAt:   time.Now(),
	}, nil
}

func (f *fakeVBManager) Stop(ctx context.Context, id string) error {
	f.stopCalls = append(f.stopCalls, id)
	f.callOrder = append(f.callOrder, "Stop")
	return nil
}

func (f *fakeVBManager) StopAll(ctx context.Context) error {
	f.stopAllCalled = true
	if f.stopAllDelay > 0 {
		select {
		case <-time.After(f.stopAllDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (f *fakeVBManager) ListManaged(contextName string) []*volumebrowser.ManagedPod {
	return f.listResult
}

func (f *fakeVBManager) AttachTab(id, tabID string) error {
	f.attachCalls = append(f.attachCalls, [2]string{id, tabID})
	return nil
}

func (f *fakeVBManager) ScanOrphans(ctx context.Context, contextName string) ([]volumebrowser.OrphanPod, error) {
	if f.scanErr != nil {
		return nil, f.scanErr
	}
	return f.scanResult, nil
}

func (f *fakeVBManager) CleanupOrphans(ctx context.Context, contextName string) error {
	f.cleanupCalls = append(f.cleanupCalls, contextName)
	return f.cleanupErr
}

func (f *fakeVBManager) FindByPVC(ctxName, namespace, pvc string) (*volumebrowser.ManagedPod, bool) {
	if f.findResult == nil {
		return nil, false
	}
	return f.findResult, true
}

type fakeCfgResolver struct {
	prefs config.ResolvedPrefs
}

func (f *fakeCfgResolver) ResolveForCluster(ctxName string) config.ResolvedPrefs {
	return f.prefs
}

// ---- tests ----

func TestVolumeBrowserService_Spawn_HappyPath(t *testing.T) {
	mgr := &fakeVBManager{}
	cfg := &fakeCfgResolver{prefs: config.ResolvedPrefs{
		VolumeBrowser: config.VolumeBrowserConfig{Image: "alpine:edge"},
	}}
	svc := newVolumeBrowserServiceForTest(mgr, cfg)

	img := "busybox:latest"
	res, err := svc.Spawn(SpawnRequestDTO{
		ContextName: "ctx1",
		Namespace:   "default",
		PVCName:     "data",
		Overrides:   &SpawnOverridesDTO{Image: &img},
	})
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "id-1", res.ID)
	testza.AssertEqual(t, "klados-browser-xxx", res.PodName)
	testza.AssertEqual(t, 1, mgr.spawnCalls)
	testza.AssertEqual(t, "alpine:edge", mgr.lastResolvedCfg.Image)
	testza.AssertNotNil(t, mgr.lastSpawnReq.Overrides)
	testza.AssertEqual(t, "busybox:latest", *mgr.lastSpawnReq.Overrides.Image)
}

func TestVolumeBrowserService_Spawn_PVCNotBound(t *testing.T) {
	mgr := &fakeVBManager{
		spawnErr: fmt.Errorf("%w: /data", volumebrowser.ErrPVCNotBound),
	}
	svc := newVolumeBrowserServiceForTest(mgr, &fakeCfgResolver{})

	_, err := svc.Spawn(SpawnRequestDTO{ContextName: "c", Namespace: "ns", PVCName: "data"})
	testza.AssertNotNil(t, err)
	testza.AssertTrue(t, errors.Is(err, volumebrowser.ErrPVCNotBound))
}

func TestVolumeBrowserService_Spawn_CollisionReturnsCollisionError(t *testing.T) {
	existing := &volumebrowser.ManagedPod{ID: "pre-existing-id", PodName: "pre-pod"}
	mgr := &fakeVBManager{
		spawnErr:   fmt.Errorf("%w: clash", volumebrowser.ErrCollision),
		findResult: existing,
	}
	svc := newVolumeBrowserServiceForTest(mgr, &fakeCfgResolver{})

	_, err := svc.Spawn(SpawnRequestDTO{ContextName: "c", Namespace: "ns", PVCName: "data"})
	testza.AssertNotNil(t, err)

	var ce *CollisionError
	testza.AssertTrue(t, errors.As(err, &ce))
	testza.AssertEqual(t, "pre-pod", ce.ExistingPodName)
	testza.AssertEqual(t, "pre-existing-id", ce.ExistingID)

	// Only one Spawn call — no retry after collision.
	testza.AssertEqual(t, 1, mgr.spawnCalls)
}

func TestVolumeBrowserService_Stop_Delegates(t *testing.T) {
	mgr := &fakeVBManager{}
	svc := newVolumeBrowserServiceForTest(mgr, &fakeCfgResolver{})

	testza.AssertNoError(t, svc.Stop("id-42"))
	testza.AssertLen(t, mgr.stopCalls, 1)
	testza.AssertEqual(t, "id-42", mgr.stopCalls[0])
}

func TestVolumeBrowserService_Replace_StopThenSpawn(t *testing.T) {
	mgr := &fakeVBManager{}
	svc := newVolumeBrowserServiceForTest(mgr, &fakeCfgResolver{})

	_, err := svc.Replace("old-id", SpawnRequestDTO{ContextName: "c", Namespace: "ns", PVCName: "data"})
	testza.AssertNoError(t, err)

	testza.AssertLen(t, mgr.callOrder, 2)
	testza.AssertEqual(t, "Stop", mgr.callOrder[0])
	testza.AssertEqual(t, "Spawn", mgr.callOrder[1])
	testza.AssertEqual(t, "old-id", mgr.stopCalls[0])
}

func TestVolumeBrowserService_AttachTab_Delegates(t *testing.T) {
	mgr := &fakeVBManager{}
	svc := newVolumeBrowserServiceForTest(mgr, &fakeCfgResolver{})

	testza.AssertNoError(t, svc.AttachTab("id-1", "tab-7"))
	testza.AssertLen(t, mgr.attachCalls, 1)
	testza.AssertEqual(t, "id-1", mgr.attachCalls[0][0])
	testza.AssertEqual(t, "tab-7", mgr.attachCalls[0][1])
}

func TestVolumeBrowserService_ListManaged_ReturnsDTOs(t *testing.T) {
	now := time.Now()
	mgr := &fakeVBManager{
		listResult: []*volumebrowser.ManagedPod{
			{ID: "a", ContextName: "c", Namespace: "ns", PodName: "p1", PVCName: "pvc1", CreatedAt: now, SessionUUID: "s", TerminalTabID: "t"},
		},
	}
	svc := newVolumeBrowserServiceForTest(mgr, &fakeCfgResolver{})

	out := svc.ListManaged("c")
	testza.AssertLen(t, out, 1)
	testza.AssertEqual(t, "a", out[0].ID)
	testza.AssertEqual(t, "p1", out[0].PodName)
	testza.AssertEqual(t, "t", out[0].TerminalTabID)
}

func TestVolumeBrowserService_ScanOrphans_ReturnsDTOs(t *testing.T) {
	mgr := &fakeVBManager{
		scanResult: []volumebrowser.OrphanPod{{ContextName: "c", Namespace: "ns", PodName: "p", PVCName: "pvc"}},
	}
	svc := newVolumeBrowserServiceForTest(mgr, &fakeCfgResolver{})

	out, err := svc.ScanOrphans("c")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, out, 1)
	testza.AssertEqual(t, "p", out[0].PodName)
}

func TestVolumeBrowserService_CleanupOrphans_Delegates(t *testing.T) {
	mgr := &fakeVBManager{}
	svc := newVolumeBrowserServiceForTest(mgr, &fakeCfgResolver{})

	testza.AssertNoError(t, svc.CleanupOrphans("c"))
	testza.AssertLen(t, mgr.cleanupCalls, 1)
	testza.AssertEqual(t, "c", mgr.cleanupCalls[0])
}

func TestVolumeBrowserService_ServiceShutdown_CallsStopAllWithinTimeout(t *testing.T) {
	mgr := &fakeVBManager{}
	svc := newVolumeBrowserServiceForTest(mgr, &fakeCfgResolver{})

	done := make(chan error, 1)
	go func() { done <- svc.ServiceShutdown() }()

	select {
	case err := <-done:
		testza.AssertNoError(t, err)
		testza.AssertTrue(t, mgr.stopAllCalled)
	case <-time.After(6 * time.Second):
		t.Fatal("ServiceShutdown did not return within 6s")
	}
}
