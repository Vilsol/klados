package volumebrowser

import (
	"sync"
)

// Tracker keeps an in-memory index of managed pods, keyed by ManagedPod.ID.
// ListForContext / StopForContext scan the map linearly; this is fine given the
// expected scale (a few dozen managed pods per session).
type Tracker struct {
	mu   sync.RWMutex
	pods map[string]*ManagedPod // id → pod
}

func NewTracker() *Tracker {
	return &Tracker{pods: make(map[string]*ManagedPod)}
}

// Add inserts a managed pod. Returns ErrCollision if another pod already exists
// for the same (context, namespace, pvc) tuple.
func (t *Tracker) Add(p *ManagedPod) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, existing := range t.pods {
		if existing.ContextName == p.ContextName &&
			existing.Namespace == p.Namespace &&
			existing.PVCName == p.PVCName {
			return ErrCollision
		}
	}
	t.pods[p.ID] = p
	return nil
}

// AddUnchecked inserts without collision check (used by orphan adoption paths
// and tests that construct multiple pods for the same PVC intentionally).
func (t *Tracker) AddUnchecked(p *ManagedPod) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pods[p.ID] = p
}

func (t *Tracker) Remove(id string) *ManagedPod {
	t.mu.Lock()
	defer t.mu.Unlock()
	p, ok := t.pods[id]
	if !ok {
		return nil
	}
	delete(t.pods, id)
	return p
}

func (t *Tracker) Get(id string) (*ManagedPod, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	p, ok := t.pods[id]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

func (t *Tracker) ListForContext(ctxName string) []*ManagedPod {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var out []*ManagedPod
	for _, p := range t.pods {
		if p.ContextName == ctxName {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out
}

func (t *Tracker) ListAll() []*ManagedPod {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]*ManagedPod, 0, len(t.pods))
	for _, p := range t.pods {
		cp := *p
		out = append(out, &cp)
	}
	return out
}

func (t *Tracker) RemoveAll() []*ManagedPod {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*ManagedPod, 0, len(t.pods))
	for id, p := range t.pods {
		cp := *p
		out = append(out, &cp)
		delete(t.pods, id)
	}
	return out
}

// SetTerminalTabID updates the tab id for a managed pod.
func (t *Tracker) SetTerminalTabID(id, tabID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	p, ok := t.pods[id]
	if !ok {
		return false
	}
	p.TerminalTabID = tabID
	return true
}
