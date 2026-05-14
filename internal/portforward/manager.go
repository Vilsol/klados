package portforward

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/sasha-s/go-deadlock"
	"time"

	"github.com/Vilsol/slox"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/config"
)

type TargetKind string

const (
	TargetKindPod         TargetKind = "pod"
	TargetKindStatefulPod TargetKind = "statefulpod"
	TargetKindSelector    TargetKind = "selector"
)

type ForwardStatus string

const (
	StatusActive       ForwardStatus = "active"
	StatusReconnecting ForwardStatus = "reconnecting"
	StatusFailed       ForwardStatus = "failed"
	StatusStopped      ForwardStatus = "stopped"
)

type ForwardSpec struct {
	ID          string        `json:"id"`
	ContextName string        `json:"contextName"`
	Namespace   string        `json:"namespace"`
	TargetKind  TargetKind    `json:"targetKind"`
	TargetName  string        `json:"targetName"`
	TargetGVR   string        `json:"targetGVR,omitempty"`
	LocalPort   int           `json:"localPort"`
	RemotePort  int           `json:"remotePort"`
	Status      ForwardStatus `json:"status"`
	PodName     string        `json:"podName,omitempty"`
	Error       string        `json:"error,omitempty"`
}

type ConnectionProvider interface {
	GetConnection(contextName string) (*cluster.Connection, error)
}

// tunnelFunc is the function signature used to run a port-forward tunnel.
// It calls onReady with the assigned local port once established,
// then blocks until the tunnel drops or ctx is cancelled.
type tunnelFunc func(ctx context.Context, conn *cluster.Connection, namespace, podName string, localPort, remotePort int, onReady func(uint16)) error

type forwardEntry struct {
	spec   ForwardSpec
	cancel context.CancelFunc
}

type Manager struct {
	mu        deadlock.Mutex
	forwards  map[string]*forwardEntry
	connMgr   ConnectionProvider
	cfg       *config.Config
	emitEvent func(string, any)
	ctx       context.Context
	tunnel    tunnelFunc
}

func NewManager(connMgr ConnectionProvider, cfg *config.Config, emitEvent func(string, any), ctx context.Context) *Manager {
	return &Manager{
		forwards:  make(map[string]*forwardEntry),
		connMgr:   connMgr,
		cfg:       cfg,
		emitEvent: emitEvent,
		ctx:       ctx,
		tunnel:    defaultRunTunnel,
	}
}

func (m *Manager) SaveForward(ctxName string, fwd config.SavedPortForward) error {
	if fwd.ID == "" {
		id, err := newForwardID()
		if err != nil {
			return err
		}
		fwd.ID = id
	}
	return m.cfg.Update(func(c *config.Config) {
		if c.PortForwards == nil {
			c.PortForwards = make(map[string][]config.SavedPortForward)
		}
		forwards := c.PortForwards[ctxName]
		for i, f := range forwards {
			if f.ID == fwd.ID {
				forwards[i] = fwd
				c.PortForwards[ctxName] = forwards
				return
			}
		}
		c.PortForwards[ctxName] = append(forwards, fwd)
	})
}

func (m *Manager) RemoveSavedForward(ctxName, id string) error {
	return m.cfg.Update(func(c *config.Config) {
		forwards := c.PortForwards[ctxName]
		filtered := forwards[:0]
		for _, f := range forwards {
			if f.ID != id {
				filtered = append(filtered, f)
			}
		}
		if len(filtered) == 0 {
			delete(c.PortForwards, ctxName)
		} else {
			c.PortForwards[ctxName] = filtered
		}
	})
}

func (m *Manager) SetForwardEnabled(ctxName, id string, enabled bool) error {
	return m.cfg.Update(func(c *config.Config) {
		for i, f := range c.PortForwards[ctxName] {
			if f.ID == id {
				c.PortForwards[ctxName][i].Enabled = enabled
				return
			}
		}
	})
}

func (m *Manager) ListSavedForwards(ctxName string) []config.SavedPortForward {
	var result []config.SavedPortForward
	m.cfg.Read(func(c *config.Config) {
		forwards := c.PortForwards[ctxName]
		result = make([]config.SavedPortForward, len(forwards))
		copy(result, forwards)
	})
	return result
}

func (m *Manager) ReconnectSaved(ctxName string) {
	var saved []config.SavedPortForward
	m.cfg.Read(func(c *config.Config) {
		forwards := c.PortForwards[ctxName]
		saved = make([]config.SavedPortForward, len(forwards))
		copy(saved, forwards)
	})

	for _, fwd := range saved {
		if !fwd.Enabled {
			continue
		}
		spec := ForwardSpec{
			ID:          fwd.ID,
			ContextName: ctxName,
			Namespace:   fwd.Namespace,
			TargetKind:  TargetKind(fwd.TargetKind),
			TargetName:  fwd.TargetName,
			TargetGVR:   fwd.TargetGVR,
			LocalPort:   fwd.LocalPort,
			RemotePort:  fwd.RemotePort,
		}
		if _, err := m.StartForward(spec); err != nil {
			slox.Warn(m.ctx, "reconnect saved forward failed", "id", fwd.ID, "error", err)
			errSpec := ForwardSpec{
				ID:          fwd.ID,
				ContextName: ctxName,
				Namespace:   fwd.Namespace,
				TargetKind:  TargetKind(fwd.TargetKind),
				TargetName:  fwd.TargetName,
				LocalPort:   fwd.LocalPort,
				RemotePort:  fwd.RemotePort,
				Status:      StatusFailed,
				Error:       err.Error(),
			}
			m.emitEvent(fmt.Sprintf("portforward:%s:%s", ctxName, fwd.ID), errSpec)
			m.emitEvent(fmt.Sprintf("portforward:%s:updated", ctxName), nil)
		}
	}
}

func (m *Manager) StartForward(spec ForwardSpec) (ForwardSpec, error) {
	if _, err := m.connMgr.GetConnection(spec.ContextName); err != nil {
		return ForwardSpec{}, fmt.Errorf("getting connection: %w", err)
	}

	if spec.ID == "" {
		id, err := newForwardID()
		if err != nil {
			return ForwardSpec{}, err
		}
		spec.ID = id
	}
	spec.Status = StatusReconnecting

	fwCtx, cancel := context.WithCancel(context.Background())
	entry := &forwardEntry{spec: spec, cancel: cancel}

	m.mu.Lock()
	m.forwards[spec.ID] = entry
	m.mu.Unlock()

	slox.Info(m.ctx, "starting port forward", "id", spec.ID, "target", spec.TargetName, "namespace", spec.Namespace, "remotePort", spec.RemotePort)

	go m.runLoop(fwCtx, entry)

	return spec, nil
}

func (m *Manager) runLoop(ctx context.Context, entry *forwardEntry) {
	backoff := time.Second
	const maxBackoff = 30 * time.Second
	// Only reset backoff if a tunnel stayed active for at least this long.
	// Otherwise a flapping connection (onReady fires, tunnel drops immediately)
	// would loop at 1s forever and spam reconnects.
	const stableThreshold = 15 * time.Second

	for {
		conn, err := m.connMgr.GetConnection(entry.spec.ContextName)
		if err != nil {
			m.updateStatus(entry, StatusFailed, "", err.Error())
			return
		}

		podName, err := resolvePodTarget(ctx, conn, &entry.spec)
		if err != nil {
			if ctx.Err() != nil {
				m.updateStatus(entry, StatusStopped, "", "")
				return
			}
			slox.Warn(m.ctx, "port-forward pod resolution failed", "id", entry.spec.ID, "target", entry.spec.TargetName, "error", err)
			if entry.spec.TargetKind == TargetKindPod {
				m.mu.Lock()
				delete(m.forwards, entry.spec.ID)
				m.mu.Unlock()
				m.emitEvent(fmt.Sprintf("portforward:%s:updated", entry.spec.ContextName), nil)
				return
			}
			m.updateStatus(entry, StatusFailed, "", err.Error())
			select {
			case <-ctx.Done():
				m.updateStatus(entry, StatusStopped, "", "")
				return
			case <-time.After(backoff):
				backoff = minDuration(backoff*2, maxBackoff)
				continue
			}
		}

		slox.Info(m.ctx, "port-forward connecting", "id", entry.spec.ID, "pod", podName, "remotePort", entry.spec.RemotePort)
		m.updateStatus(entry, StatusReconnecting, podName, "")

		var activeSince time.Time
		tunnelErr := m.tunnel(ctx, conn, entry.spec.Namespace, podName, entry.spec.LocalPort, entry.spec.RemotePort,
			func(assignedPort uint16) {
				m.mu.Lock()
				if entry.spec.LocalPort == 0 {
					entry.spec.LocalPort = int(assignedPort)
				}
				m.mu.Unlock()
				activeSince = time.Now()
				slox.Info(m.ctx, "port-forward active", "id", entry.spec.ID, "pod", podName, "localPort", int(assignedPort), "remotePort", entry.spec.RemotePort)
				m.updateStatus(entry, StatusActive, podName, "")
			},
		)

		if ctx.Err() != nil {
			m.updateStatus(entry, StatusStopped, "", "")
			return
		}

		errMsg := ""
		if tunnelErr != nil {
			errMsg = tunnelErr.Error()
			slox.Warn(m.ctx, "port-forward tunnel dropped", "id", entry.spec.ID, "pod", podName, "error", tunnelErr)
		} else {
			slox.Info(m.ctx, "port-forward tunnel closed", "id", entry.spec.ID, "pod", podName)
		}

		if entry.spec.TargetKind == TargetKindPod {
			slox.Info(m.ctx, "port-forward not retrying (raw pod target)", "id", entry.spec.ID, "pod", podName)
			m.mu.Lock()
			delete(m.forwards, entry.spec.ID)
			m.mu.Unlock()
			m.emitEvent(fmt.Sprintf("portforward:%s:updated", entry.spec.ContextName), nil)
			return
		}

		if !activeSince.IsZero() && time.Since(activeSince) >= stableThreshold {
			backoff = time.Second
		}

		slox.Info(m.ctx, "port-forward reconnecting", "id", entry.spec.ID, "backoff", backoff)
		m.updateStatus(entry, StatusReconnecting, podName, errMsg)
		select {
		case <-ctx.Done():
			m.updateStatus(entry, StatusStopped, "", "")
			return
		case <-time.After(backoff):
			backoff = minDuration(backoff*2, maxBackoff)
		}
	}
}

func (m *Manager) updateStatus(entry *forwardEntry, status ForwardStatus, podName, errMsg string) {
	m.mu.Lock()
	entry.spec.Status = status
	if podName != "" {
		entry.spec.PodName = podName
	}
	entry.spec.Error = errMsg
	specCopy := entry.spec
	m.mu.Unlock()

	// When a forward goes active with an auto-assigned port, persist it so
	// reconnects use the same port and the management page shows a real port.
	if status == StatusActive && specCopy.LocalPort != 0 && m.cfg != nil {
		_ = m.cfg.Update(func(c *config.Config) {
			for i, f := range c.PortForwards[specCopy.ContextName] {
				if f.ID == specCopy.ID && f.LocalPort == 0 {
					c.PortForwards[specCopy.ContextName][i].LocalPort = specCopy.LocalPort
					return
				}
			}
		})
	}

	// Emit per-forward event and aggregate event for list subscribers.
	m.emitEvent(fmt.Sprintf("portforward:%s:%s", specCopy.ContextName, specCopy.ID), specCopy)
	m.emitEvent(fmt.Sprintf("portforward:%s:updated", specCopy.ContextName), nil)
}

func (m *Manager) StopForward(id string) error {
	m.mu.Lock()
	entry, ok := m.forwards[id]
	if ok {
		delete(m.forwards, id)
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("forward %q not found", id)
	}
	slox.Info(m.ctx, "stopping port forward", "id", id)
	entry.cancel()
	return nil
}

func (m *Manager) ListForwards(contextName string) []ForwardSpec {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []ForwardSpec
	for _, entry := range m.forwards {
		if contextName == "" || entry.spec.ContextName == contextName {
			result = append(result, entry.spec)
		}
	}
	if result == nil {
		return []ForwardSpec{}
	}
	return result
}

func (m *Manager) Cfg() *config.Config {
	return m.cfg
}

func (m *Manager) StopAll() {
	slox.Info(m.ctx, "stopping all port forwards")
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, entry := range m.forwards {
		entry.cancel()
		delete(m.forwards, id)
	}
}

func newForwardID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
