package helm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Vilsol/slox"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// secretWatcher abstracts the cluster lookups that the helm watch source
// needs. In production this wraps cluster.Manager; tests inject an in-memory
// fake.
type secretWatcher interface {
	// SecretsClient returns a typed clientset for the given context. The
	// secret-watch loop calls CoreV1().Secrets(ns).Watch / .List directly so
	// it can reuse standard fieldSelector / resourceVersion semantics.
	SecretsClient(contextName string) (kubernetes.Interface, error)
}

// WatchSource is a VirtualWatchSource for helm.v1.releases. It watches the
// underlying release Secrets (type=helm.sh/release.v1) and emits aggregator-
// collapsed virtual events.
type WatchSource struct {
	backend *Backend
	getter  secretWatcher
}

// NewWatchSource constructs a WatchSource that drives the given Backend's
// aggregator. The getter supplies a typed Kubernetes client for the watch
// loop.
func NewWatchSource(backend *Backend, getter secretWatcher) *WatchSource {
	return &WatchSource{backend: backend, getter: getter}
}

// Watch implements watcher.VirtualWatchSource. It starts a goroutine that
// drives an underlying Secret watch into the aggregator and emits collapsed
// release events via emit. The returned stop function cancels the loop and
// resets aggregator state for the watched namespace.
func (w *WatchSource) Watch(ctx context.Context, contextName, namespace, resourceVersion string, emit func(string, any)) (func(), error) {
	if w.backend == nil || w.getter == nil {
		return nil, errors.New("helm watchsource: not fully configured")
	}
	cli, err := w.getter.SecretsClient(contextName)
	if err != nil {
		return nil, err
	}

	loopCtx, cancel := context.WithCancel(ctx)
	go w.run(loopCtx, cli, contextName, namespace, resourceVersion, emit)

	stop := func() {
		cancel()
		w.backend.Aggregator().Reset(namespace)
	}
	return stop, nil
}

func (w *WatchSource) run(ctx context.Context, cli kubernetes.Interface, contextName, namespace, initialRV string, emit func(name string, payload any)) {
	currentRV := initialRV
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// If we have no resource version (cold start or after a Gone), do a
		// full sync from List.
		if currentRV == "" {
			rv, err := w.snapshot(ctx, cli, contextName, namespace, emit)
			if err != nil {
				if isCanceled(ctx, err) {
					return
				}
				slox.Warn(ctx, "helm watchsource: snapshot failed, retrying", "context", contextName, "ns", namespace, "error", err)
				if !sleepCtx(ctx, 5*time.Second) {
					return
				}
				continue
			}
			currentRV = rv
		}

		wi, err := cli.CoreV1().Secrets(namespace).Watch(ctx, metav1.ListOptions{
			FieldSelector:       "type=" + ReleaseSecretType,
			ResourceVersion:     currentRV,
			AllowWatchBookmarks: true,
		})
		if err != nil {
			if isCanceled(ctx, err) {
				return
			}
			if k8serrors.IsGone(err) {
				slox.Warn(ctx, "helm watchsource: RV too old, resyncing", "context", contextName, "ns", namespace, "rv", currentRV)
				currentRV = ""
				continue
			}
			slox.Warn(ctx, "helm watchsource: watch failed, retrying", "context", contextName, "ns", namespace, "error", err)
			if !sleepCtx(ctx, 5*time.Second) {
				return
			}
			continue
		}

		nextRV, gone := w.processStream(ctx, wi, contextName, namespace, currentRV, emit)
		if gone {
			slox.Warn(ctx, "helm watchsource: stream Gone, resyncing", "context", contextName, "ns", namespace, "rv", currentRV)
			currentRV = ""
			continue
		}
		currentRV = nextRV
	}
}

// snapshot performs a full ListSecrets, brackets the result with SYNC_START /
// SYNC_END, and emits one ADDED event per release.
func (w *WatchSource) snapshot(ctx context.Context, cli kubernetes.Interface, contextName, namespace string, emit func(name string, payload any)) (string, error) {
	list, err := cli.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "type=" + ReleaseSecretType,
	})
	if err != nil {
		return "", err
	}
	flat, rerr := ReassembleContinuation(list.Items)
	if rerr != nil {
		slox.Warn(ctx, "helm watchsource: continuation reassembly failed, using raw secrets", "error", rerr)
		flat = list.Items
	}
	// Reset prior aggregator state for the namespace before replaying.
	w.backend.Aggregator().Reset(namespace)
	out, err := w.backend.Aggregator().CollapseSnapshot(flat)
	if err != nil {
		return "", err
	}
	emit(syncEventName(contextName, namespace, syncStart), nil)
	for _, obj := range out {
		emit(watchEventName(contextName, namespace), eventPayload("ADDED", obj))
	}
	emit(syncEventName(contextName, namespace, syncEnd), nil)
	return list.ResourceVersion, nil
}

func (w *WatchSource) processStream(ctx context.Context, wi watch.Interface, contextName, namespace, currentRV string, emit func(name string, payload any)) (string, bool) {
	defer wi.Stop()
	for {
		select {
		case <-ctx.Done():
			return currentRV, false
		case event, ok := <-wi.ResultChan():
			if !ok {
				return currentRV, false
			}
			if event.Type == watch.Error {
				if status, ok := event.Object.(*metav1.Status); ok {
					if status.Reason == metav1.StatusReasonGone || status.Code == 410 {
						return currentRV, true
					}
					slox.Warn(ctx, "helm watchsource: watch error event", "reason", status.Reason, "message", status.Message)
				}
				return currentRV, false
			}
			if event.Type == watch.Bookmark {
				if s, ok := event.Object.(*corev1.Secret); ok && s.ResourceVersion != "" {
					currentRV = s.ResourceVersion
				}
				continue
			}
			secret, ok := event.Object.(*corev1.Secret)
			if !ok {
				continue
			}
			if secret.ResourceVersion != "" {
				currentRV = secret.ResourceVersion
			}
			if secret.Type != ReleaseSecretType {
				continue
			}
			ve, err := w.backend.Aggregator().ApplyDelta(string(event.Type), secret)
			if err != nil {
				slox.Warn(ctx, "helm watchsource: ApplyDelta failed", "secret", secret.Name, "error", err)
				continue
			}
			if ve == nil {
				continue
			}
			emit(watchEventName(contextName, namespace), eventPayload(ve.Type, ve.Object))
		}
	}
}

// syncEventName returns the synthetic SYNC_START / SYNC_END topic for the
// helm watch on (contextName, namespace).
func syncEventName(contextName, namespace string, phase string) string {
	return fmt.Sprintf("watch:%s:helm.v1.releases:%s:%s", contextName, namespace, phase)
}

const (
	syncStart = "sync-start"
	syncEnd   = "sync-end"
)

// watchEventName mirrors the WatchManager's primary topic format for virtual
// dispatch. Emitting via this name allows frontend consumers to subscribe in
// exactly the same way as Kubernetes-backed watches.
func watchEventName(contextName, namespace string) string {
	return fmt.Sprintf("watch:%s:helm.v1.releases:%s", contextName, namespace)
}

// eventPayload matches watcher.WatchEvent's wire shape so the frontend
// ResourceStore decodes virtual events identically to real watches.
func eventPayload(eventType string, object map[string]any) map[string]any {
	return map[string]any{
		"type":   eventType,
		"object": object,
	}
}

// sleepCtx sleeps for d or until ctx is cancelled. Returns true if the full
// sleep elapsed, false if ctx was cancelled first.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func isCanceled(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return true
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
