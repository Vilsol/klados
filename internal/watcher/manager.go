package watcher

import (
	"context"
	"fmt"
	"github.com/sasha-s/go-deadlock"
	"time"

	"github.com/Vilsol/slox"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/resource"
)

const gracePeriod = 30 * time.Second

type watchKey struct {
	contextName string
	gvr         string
	namespace   string
}

type watchState struct {
	cancel     context.CancelFunc
	graceTimer *time.Timer
}

type WatchEvent struct {
	Type   string         `json:"type"`
	Object map[string]any `json:"object"`
}

type ConnectionProvider interface {
	GetConnection(contextName string) (*cluster.Connection, error)
}

type WatchManager struct {
	mu          deadlock.Mutex
	clusterMgr  ConnectionProvider
	enricherReg *resource.EnricherRegistry
	emitEvent   func(string, any)
	ctx         context.Context
	watches     map[watchKey]*watchState
}

func NewWatchManager(mgr ConnectionProvider, enricherReg *resource.EnricherRegistry, emit func(string, any), ctx context.Context) *WatchManager {
	return &WatchManager{
		clusterMgr:  mgr,
		enricherReg: enricherReg,
		emitEvent:   emit,
		ctx:         ctx,
		watches:     make(map[watchKey]*watchState),
	}
}

func (m *WatchManager) StartWatch(contextName, gvr, namespace string) error {
	key := watchKey{contextName, gvr, namespace}

	m.mu.Lock()
	defer m.mu.Unlock()

	if state, ok := m.watches[key]; ok {
		if state.graceTimer != nil {
			state.graceTimer.Stop()
			state.graceTimer = nil
		}
		return nil
	}

	slox.Debug(m.ctx, "watch started", "context", contextName, "gvr", gvr, "namespace", namespace)

	conn, err := m.clusterMgr.GetConnection(contextName)
	if err != nil {
		return err
	}

	gvrParsed, err := resource.ParseGVR(gvr)
	if err != nil {
		return err
	}

	var ri dynamic.ResourceInterface
	dr := conn.Dynamic.Resource(gvrParsed)
	if namespace != "" {
		ri = dr.Namespace(namespace)
	} else {
		ri = dr
	}

	enrichers := m.enricherReg.GetAll(gvr)
	eventName := fmt.Sprintf("watch:%s:%s:%s", contextName, gvr, namespace)

	ctx, cancel := context.WithCancel(context.Background())
	m.watches[key] = &watchState{cancel: cancel}

	go m.runWatch(ctx, ri, enrichers, eventName, key, contextName)
	return nil
}

func (m *WatchManager) runWatch(
	ctx context.Context,
	ri dynamic.ResourceInterface,
	enrichers []resource.Enricher,
	eventName string,
	key watchKey,
	contextName string,
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		wi, err := ri.Watch(ctx, metav1.ListOptions{})
		if err != nil {
			slox.Warn(m.ctx, "watch failed, retrying", "event", eventName, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		m.processEvents(ctx, wi, enrichers, eventName, contextName)
	}
}

func (m *WatchManager) processEvents(
	ctx context.Context,
	wi watch.Interface,
	enrichers []resource.Enricher,
	eventName string,
	contextName string,
) {
	defer wi.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-wi.ResultChan():
			if !ok {
				return
			}

			obj, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				continue
			}

			for _, enricher := range enrichers {
				if err := enricher.Enrich(contextName, obj); err != nil {
					slox.Debug(m.ctx, "enricher error on watch event", "error", err)
				}
			}

			m.emitEvent(eventName, WatchEvent{
				Type:   string(event.Type),
				Object: obj.Object,
			})
		}
	}
}

func (m *WatchManager) StopWatch(contextName, gvr, namespace string) {
	key := watchKey{contextName, gvr, namespace}

	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.watches[key]
	if !ok {
		return
	}

	if state.graceTimer != nil {
		return
	}

	state.graceTimer = time.AfterFunc(gracePeriod, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if s, ok := m.watches[key]; ok && s.graceTimer != nil {
			slox.Debug(m.ctx, "watch stopped", "context", contextName, "gvr", gvr)
			s.cancel()
			delete(m.watches, key)
		}
	})
}

func (m *WatchManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, state := range m.watches {
		if state.graceTimer != nil {
			state.graceTimer.Stop()
		}
		state.cancel()
		delete(m.watches, key)
	}
}
