package volumebrowser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Vilsol/slox"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Vilsol/klados/internal/config"
)

type spawnKey struct{ ctx, ns, pvc string }

// Manager stitches together the spawner, tracker, and orphan scanner.
// It exposes the public surface consumed by the Wails service layer (Task 3).
type Manager struct {
	ctx         context.Context
	sessionUUID string
	connMgr     ConnectionProvider
	tracker     *Tracker
	spawner     *Spawner

	spawnMu       sync.Mutex
	spawnInFlight map[spawnKey]struct{}
}

func NewManager(ctx context.Context, connMgr ConnectionProvider, sessionUUID string) *Manager {
	return &Manager{
		ctx:           ctx,
		sessionUUID:   sessionUUID,
		connMgr:       connMgr,
		tracker:       NewTracker(),
		spawner:       NewSpawner(sessionUUID),
		spawnInFlight: make(map[spawnKey]struct{}),
	}
}

// SessionUUID returns the session identifier used by this manager.
// (Exposed so the orphan scanner callers can fabricate matching labels in tests.)
func (m *Manager) SessionUUID() string { return m.sessionUUID }

func (m *Manager) acquireSpawnSlot(k spawnKey) bool {
	m.spawnMu.Lock()
	defer m.spawnMu.Unlock()
	if _, ok := m.spawnInFlight[k]; ok {
		return false
	}
	m.spawnInFlight[k] = struct{}{}
	return true
}

func (m *Manager) releaseSpawnSlot(k spawnKey) {
	m.spawnMu.Lock()
	delete(m.spawnInFlight, k)
	m.spawnMu.Unlock()
}

// Spawn creates a browser pod for the given PVC.
//
// The caller is responsible for resolving the effective VolumeBrowserConfig for
// the target cluster before calling (see config.Config.ResolveForCluster).
// Any SpawnOverrides in req are layered on top of resolvedCfg before pod creation.
func (m *Manager) Spawn(ctx context.Context, req SpawnRequest, resolvedCfg config.VolumeBrowserConfig) (*ManagedPod, error) {
	key := spawnKey{ctx: req.ContextName, ns: req.Namespace, pvc: req.PVCName}
	if !m.acquireSpawnSlot(key) {
		return nil, fmt.Errorf("%w: spawn already in flight for %s/%s", ErrCollision, req.Namespace, req.PVCName)
	}
	defer m.releaseSpawnSlot(key)

	conn, err := m.connMgr.GetConnection(req.ContextName)
	if err != nil {
		return nil, fmt.Errorf("getting connection: %w", err)
	}

	// Collision guard BEFORE creating the pod (fast path).
	if existing := m.findByPVC(req.ContextName, req.Namespace, req.PVCName); existing != nil {
		return nil, fmt.Errorf("%w: %s/%s already has managed pod %s", ErrCollision, req.Namespace, req.PVCName, existing.PodName)
	}

	merged := applyOverrides(resolvedCfg, req.Overrides)

	pod, err := m.spawner.Spawn(ctx, conn, SpawnParams{Request: req, Resolved: merged})
	if err != nil {
		return nil, err
	}

	if err := m.tracker.Add(pod); err != nil {
		// Should be impossible under the spawn lock; belt-and-braces cleanup.
		slox.Warn(m.ctx, "volumebrowser: tracker collision after create, deleting pod", "pod", pod.PodName)
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if delErr := conn.Dynamic.Resource(podGVR).Namespace(req.Namespace).Delete(cleanupCtx, pod.PodName, metav1.DeleteOptions{}); delErr != nil {
			slox.Error(m.ctx, "volumebrowser: failed to clean up racy pod", "pod", pod.PodName, "namespace", req.Namespace, "error", delErr)
		}
		return nil, err
	}

	slox.Info(m.ctx, "volumebrowser: spawned pod", "context", req.ContextName, "namespace", req.Namespace, "pvc", req.PVCName, "pod", pod.PodName, "id", pod.ID)
	return pod, nil
}

func (m *Manager) findByPVC(ctxName, namespace, pvc string) *ManagedPod {
	for _, p := range m.tracker.ListForContext(ctxName) {
		if p.Namespace == namespace && p.PVCName == pvc {
			return p
		}
	}
	return nil
}

// FindByPVC returns the managed pod tracking the given (context, namespace, pvc),
// or nil/false if none exists. Used by the service layer to build collision
// errors at the Wails boundary.
func (m *Manager) FindByPVC(ctxName, namespace, pvc string) (*ManagedPod, bool) {
	p := m.findByPVC(ctxName, namespace, pvc)
	return p, p != nil
}

// Stop deletes the pod referenced by id from the cluster and removes it from the tracker.
// Returns an error if id is unknown or pod deletion fails.
func (m *Manager) Stop(ctx context.Context, id string) error {
	p, ok := m.tracker.Get(id)
	if !ok {
		return fmt.Errorf("volumebrowser: unknown managed pod id %q", id)
	}
	return m.stopPod(ctx, p)
}

func (m *Manager) stopPod(ctx context.Context, p *ManagedPod) error {
	conn, err := m.connMgr.GetConnection(p.ContextName)
	if err != nil {
		// Keep the entry so a retry can succeed later — losing it leaves the pod
		// orphaned cluster-side but forgotten locally.
		return fmt.Errorf("getting connection: %w", err)
	}
	err = conn.Dynamic.Resource(podGVR).Namespace(p.Namespace).Delete(ctx, p.PodName, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		slox.Warn(m.ctx, "volumebrowser: delete pod failed", "pod", p.PodName, "error", err)
		return fmt.Errorf("deleting pod %s/%s: %w", p.Namespace, p.PodName, err)
	}
	m.tracker.Remove(p.ID)
	slox.Info(m.ctx, "volumebrowser: stopped pod", "context", p.ContextName, "pod", p.PodName, "id", p.ID)
	return nil
}

// StopForContext deletes all managed pods belonging to the given context.
// Errors are logged but do not halt iteration; the returned error is the last non-nil error encountered.
func (m *Manager) StopForContext(ctx context.Context, contextName string) error {
	var lastErr error
	for _, p := range m.tracker.ListForContext(contextName) {
		if err := m.stopPod(ctx, p); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// StopAll deletes every managed pod across all contexts.
func (m *Manager) StopAll(ctx context.Context) error {
	var lastErr error
	for _, p := range m.tracker.ListAll() {
		if err := m.stopPod(ctx, p); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// ListManaged returns managed pods for a given context, or all pods if contextName is "".
func (m *Manager) ListManaged(contextName string) []*ManagedPod {
	if contextName == "" {
		return m.tracker.ListAll()
	}
	return m.tracker.ListForContext(contextName)
}

// AttachTab associates a terminal tab id with a managed pod.
// Returns an error if the id is unknown.
func (m *Manager) AttachTab(id, tabID string) error {
	if !m.tracker.SetTerminalTabID(id, tabID) {
		return fmt.Errorf("volumebrowser: unknown managed pod id %q", id)
	}
	return nil
}

// ScanOrphans returns pods that carry the klados pvc-browser labels but are not
// owned by the current session, for the given context. The caller is responsible
// for deciding cleanup policy (keep/delete/prompt).
func (m *Manager) ScanOrphans(ctx context.Context, contextName string) ([]OrphanPod, error) {
	conn, err := m.connMgr.GetConnection(contextName)
	if err != nil {
		return nil, fmt.Errorf("getting connection: %w", err)
	}
	return ScanOrphans(ctx, conn, contextName, m.sessionUUID)
}

// CleanupOrphans deletes every orphan pod (pvc-browser pods not owned by this
// session) in the given context. Errors are logged; the returned error is the
// last non-nil error encountered so callers can surface a single failure.
func (m *Manager) CleanupOrphans(ctx context.Context, contextName string) error {
	conn, err := m.connMgr.GetConnection(contextName)
	if err != nil {
		return fmt.Errorf("getting connection: %w", err)
	}
	orphans, err := ScanOrphans(ctx, conn, contextName, m.sessionUUID)
	if err != nil {
		return err
	}
	var lastErr error
	for _, o := range orphans {
		if err := conn.Dynamic.Resource(podGVR).Namespace(o.Namespace).Delete(ctx, o.PodName, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
			slox.Warn(m.ctx, "volumebrowser: failed to delete orphan pod", "pod", o.PodName, "namespace", o.Namespace, "error", err)
			lastErr = err
		}
	}
	return lastErr
}
