package helm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Vilsol/slox"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/kube"
	release "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// ErrOperationInProgress is returned when a verb is invoked against a release
// that already has another verb running.
var ErrOperationInProgress = errors.New("helm: operation in progress for this release")

// RollbackOpts captures the subset of Helm rollback options surfaced by klados.
type RollbackOpts struct {
	Wait         bool
	Timeout      time.Duration
	DisableHooks bool
}

// UninstallOpts captures the subset of Helm uninstall options surfaced by klados.
type UninstallOpts struct {
	Wait         bool
	Timeout      time.Duration
	DisableHooks bool
	KeepHistory  bool
}

// TestOpts captures the subset of Helm test options surfaced by klados.
type TestOpts struct {
	Timeout time.Duration
	Filters []string
}

// TestResult is returned from a synchronous helm-test run.
type TestResult struct {
	Phase string
	Logs  string
}

// secretDeleter abstracts the per-secret delete needed by
// ForceDeleteReleaseSecret. Task 3 wires a dynamic-client backed implementation.
type secretDeleter interface {
	DeleteSecret(ctx context.Context, contextName, namespace, name string) error
}

// actionRunner abstracts the bottom of the Helm action stack so tests can
// avoid building a real *action.Configuration. Production wires through the
// realRunner implementation below.
type actionRunner interface {
	Rollback(ctx context.Context, contextName, ns, releaseName string, version int, opts RollbackOpts) error
	Uninstall(ctx context.Context, contextName, ns, releaseName string, opts UninstallOpts) error
	Test(ctx context.Context, contextName, ns, releaseName string, opts TestOpts) (TestResult, error)
}

// Actions wraps the Helm verbs with klados conventions: per-release mutexes
// (TryLock), context-aware logging, and an injection seam for tests.
type Actions struct {
	cache   *ClientCache
	runner  actionRunner
	deleter secretDeleter

	mu    sync.Mutex
	locks map[lockKey]*sync.Mutex
}

type lockKey struct {
	ctx     string
	ns      string
	release string
}

// NewActions wires real Helm action implementations. Pass nil for cache to
// disable the per-(ctx,ns) Configuration cache (tests).
func NewActions(cache *ClientCache, getterFn func(contextName, namespace string) (genericclioptions.RESTClientGetter, error), logHandler slog.Handler, deleter secretDeleter) *Actions {
	a := &Actions{
		cache:   cache,
		deleter: deleter,
		locks:   map[lockKey]*sync.Mutex{},
	}
	a.runner = &realRunner{cache: cache, getterFn: getterFn, logHandler: logHandler}
	return a
}

// newActionsWithRunner constructs an Actions wired to a custom runner — used
// by tests.
func newActionsWithRunner(runner actionRunner, deleter secretDeleter) *Actions {
	return &Actions{
		runner:  runner,
		deleter: deleter,
		locks:   map[lockKey]*sync.Mutex{},
	}
}

// lockFor returns the mutex guarding a given (ctx, ns, release). Lazy-init.
func (a *Actions) lockFor(k lockKey) *sync.Mutex {
	a.mu.Lock()
	defer a.mu.Unlock()
	if m, ok := a.locks[k]; ok {
		return m
	}
	m := &sync.Mutex{}
	a.locks[k] = m
	return m
}

func (a *Actions) tryAcquire(k lockKey) (func(), error) {
	m := a.lockFor(k)
	if !m.TryLock() {
		return nil, ErrOperationInProgress
	}
	return func() { m.Unlock() }, nil
}

// Rollback invokes helm rollback for (ctx, ns, releaseName) at the given
// revision.
func (a *Actions) Rollback(ctx context.Context, contextName, ns, releaseName string, version int, opts RollbackOpts) error {
	release, err := a.tryAcquire(lockKey{contextName, ns, releaseName})
	if err != nil {
		return err
	}
	defer release()
	return a.runner.Rollback(ctx, contextName, ns, releaseName, version, opts)
}

// Uninstall invokes helm uninstall for (ctx, ns, releaseName).
func (a *Actions) Uninstall(ctx context.Context, contextName, ns, releaseName string, opts UninstallOpts) error {
	release, err := a.tryAcquire(lockKey{contextName, ns, releaseName})
	if err != nil {
		return err
	}
	defer release()
	return a.runner.Uninstall(ctx, contextName, ns, releaseName, opts)
}

// Test runs helm-test against the release and returns the aggregated result
// after completion. Logs are bulk-read, not streamed.
func (a *Actions) Test(ctx context.Context, contextName, ns, releaseName string, opts TestOpts) (TestResult, error) {
	release, err := a.tryAcquire(lockKey{contextName, ns, releaseName})
	if err != nil {
		return TestResult{}, err
	}
	defer release()
	return a.runner.Test(ctx, contextName, ns, releaseName, opts)
}

// ForceDeleteReleaseSecret deletes the underlying Secret for a specific
// revision. Useful for unsticking pending-upgrade releases.
func (a *Actions) ForceDeleteReleaseSecret(ctx context.Context, contextName, ns, releaseName string, rev int) error {
	if a.deleter == nil {
		return errors.New("helm: no secret deleter configured")
	}
	secretName := fmt.Sprintf("sh.helm.release.v1.%s.v%d", releaseName, rev)
	return a.deleter.DeleteSecret(ctx, contextName, ns, secretName)
}

// realRunner is the production actionRunner: builds *action.Configuration
// via the ClientCache and dispatches to helm.sh/helm/v4/pkg/action.
type realRunner struct {
	cache      *ClientCache
	getterFn   func(contextName, namespace string) (genericclioptions.RESTClientGetter, error)
	logHandler slog.Handler
}

func (r *realRunner) cfg(ctx context.Context, contextName, ns string) (*action.Configuration, error) {
	if r.cache == nil || r.getterFn == nil {
		return nil, errors.New("helm: action runner not fully configured")
	}
	getter, err := r.getterFn(contextName, ns)
	if err != nil {
		return nil, err
	}
	return r.cache.Get(ctx, contextName, ns, getter, r.logHandler)
}

func (r *realRunner) Rollback(ctx context.Context, contextName, ns, releaseName string, version int, opts RollbackOpts) error {
	cfg, err := r.cfg(ctx, contextName, ns)
	if err != nil {
		return err
	}
	rb := action.NewRollback(cfg)
	rb.Version = version
	rb.Timeout = opts.Timeout
	rb.DisableHooks = opts.DisableHooks
	rb.WaitStrategy = pickWaitStrategy(opts.Wait)
	return rb.Run(releaseName)
}

func (r *realRunner) Uninstall(ctx context.Context, contextName, ns, releaseName string, opts UninstallOpts) error {
	cfg, err := r.cfg(ctx, contextName, ns)
	if err != nil {
		return err
	}
	un := action.NewUninstall(cfg)
	un.Timeout = opts.Timeout
	un.DisableHooks = opts.DisableHooks
	un.KeepHistory = opts.KeepHistory
	un.WaitStrategy = pickWaitStrategy(opts.Wait)
	_, err = un.Run(releaseName)
	return err
}

func (r *realRunner) Test(ctx context.Context, contextName, ns, releaseName string, opts TestOpts) (TestResult, error) {
	cfg, err := r.cfg(ctx, contextName, ns)
	if err != nil {
		return TestResult{}, err
	}
	tst := action.NewReleaseTesting(cfg)
	tst.Timeout = opts.Timeout
	if len(opts.Filters) > 0 {
		tst.Filters = map[string][]string{action.IncludeNameFilter: opts.Filters}
	}
	reli, shutdown, err := tst.Run(releaseName)
	if shutdown != nil {
		defer func() { _ = shutdown() }()
	}
	if err != nil {
		return TestResult{Phase: "Error"}, err
	}
	relV1 := releaseToV1(ctx, reli)
	var buf bytes.Buffer
	if relV1 != nil {
		_ = tst.GetPodLogs(&buf, relV1)
	}
	return TestResult{Phase: phaseFromRelease(relV1), Logs: buf.String()}, nil
}

// releaseToV1 unwraps the action package's Releaser interface into the v1
// concrete type used elsewhere in this package. Returns nil if the underlying
// concrete type isn't v1.
func releaseToV1(ctx context.Context, r any) *release.Release {
	if v, ok := r.(*release.Release); ok {
		return v
	}
	slox.Warn(ctx, "helm: ReleaseTesting.Run returned unexpected release type", "type", fmt.Sprintf("%T", r))
	return nil
}

func phaseFromRelease(rel *release.Release) string {
	if rel == nil || rel.Info == nil {
		return "Unknown"
	}
	switch rel.Info.Status.String() {
	case "deployed":
		return "Success"
	case "failed":
		return "Failure"
	}
	return string(rel.Info.Status)
}

func pickWaitStrategy(wait bool) kube.WaitStrategy {
	if wait {
		return kube.StatusWatcherStrategy
	}
	return kube.HookOnlyStrategy
}
