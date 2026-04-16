# Generic GVR Phase 7 — Interactive Scaling for CRDs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Switch the backend scaling path from a hardcoded `spec.replicas` MergePatch to the Kubernetes `scale` subresource API, so any resource that declares a scale subresource (built-in or CRD) can be scaled from Klados's existing Scale dialog and BulkScaleDialog.

**Architecture:** Add `ResourceEngine.Scale()` which uses the dynamic client's `UpdateScale()` (via `autoscaling/v1.Scale`). `ResourceService.ScaleResource` delegates to the engine. If the resource lacks a scale subresource (detected via Phase 1 discovery metadata propagated into the engine), fall back to the existing MergePatch for legacy compatibility. Replicas column for scalable CRDs is already added by Phase 2 — this phase focuses on making Scale actually work for them.

**Tech Stack:** Go, k8s.io/client-go dynamic client with `Scale` interface, autoscaling/v1 Scale type.

**Depends on:** Phases 1-2 complete (descriptor has `scale` action for CRDs with scale subresource).

**Reference spec:** `docs/superpowers/specs/2026-04-16-generic-gvr-capabilities-design.md` §5.

---

## File Structure

- Modify: `internal/resource/engine.go` — new `Scale(ctx, contextName, gvr, namespace, name, replicas)` method
- Modify: `internal/services/resource.go` — `ScaleResource` delegates to engine
- Modify: `internal/services/resource_test.go` — update / add tests covering:
  - scale via subresource path (Deployment, CRD with scale)
  - MergePatch fallback when scale subresource is absent
- Modify: `internal/cluster/manager.go` — expose a lookup `func (m *Manager) HasScaleSubresource(ctx, gvr string) bool` using the discovery cache (from Phase 1)
- Regenerate: Wails bindings (signature unchanged, safe re-gen)

No frontend code changes are required — the Scale dialog already calls `ScaleResource`. Optimistic update is already in place via watch subscriptions.

---

## Task 1: Backend scaling infrastructure

**Files:**
- Modify: `internal/cluster/manager.go`
- Modify: `internal/cluster/manager_test.go` (or `discover_integration_test.go`)
- Modify: `internal/resource/engine.go`
- Modify: `internal/resource/engine_test.go` (create if absent — check first)

### Expose scale-subresource lookup on cluster.Manager

- [ ] **Step 1: Store the discovered APIResources on the Manager**

In `internal/cluster/manager.go`, the Phase 1 implementation emits the payload but doesn't retain it. Add a field to `Manager`:

```go
// inside Manager struct:
discoveredResources map[string][]APIResource // keyed by contextName
```

Initialize in `NewManager` or constructor: `discoveredResources: map[string][]APIResource{}`.

In `DiscoverResources`, before the `emitEvent` call, store the result:

```go
m.mu.Lock()
m.discoveredResources[contextName] = primary
m.mu.Unlock()
```

Replace `m.mu.Lock()` with the actual existing mutex field name used in the struct (e.g. `m.mu`).

- [ ] **Step 2: Implement `HasScaleSubresource`**

Add:

```go
// HasScaleSubresource returns true when the given GVR declared a scale
// subresource during the most recent discovery pass for this context. Returns
// false when discovery hasn't run, the context is unknown, or the resource
// lacks scale.
func (m *Manager) HasScaleSubresource(contextName, gvr string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.discoveredResources[contextName] {
		if r.GVR == gvr {
			return r.Subresources.Scale
		}
	}
	return false
}
```

- [ ] **Step 3: Unit test HasScaleSubresource**

Append to `internal/cluster/discover_integration_test.go`:

```go
func TestHasScaleSubresource(t *testing.T) {
	m := &Manager{
		discoveredResources: map[string][]APIResource{
			"c": {
				{GVR: "apps.v1.deployments", Subresources: ResourceSubresources{Scale: true}},
				{GVR: "core.v1.pods", Subresources: ResourceSubresources{}},
			},
		},
	}
	testza.AssertTrue(t, m.HasScaleSubresource("c", "apps.v1.deployments"))
	testza.AssertFalse(t, m.HasScaleSubresource("c", "core.v1.pods"))
	testza.AssertFalse(t, m.HasScaleSubresource("c", "unknown"))
	testza.AssertFalse(t, m.HasScaleSubresource("other-ctx", "apps.v1.deployments"))
}
```

- [ ] **Step 4: Run cluster tests**

Run: `go test ./internal/cluster/ -run TestHasScaleSubresource -v`
Expected: PASS.

### Engine.Scale using the dynamic client's scale subresource

- [ ] **Step 5: Check for existing engine tests**

Run: `ls internal/resource/*_test.go`
Note which exist. If `engine_test.go` is absent, create it with an appropriate fake clientset setup (mirror the patterns used in `internal/services/resource_test.go`).

- [ ] **Step 6: Write failing test**

Add to `internal/resource/engine_test.go`:

```go
package resource

import (
	"context"
	"testing"

	"github.com/MarvinJWendt/testza"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedyn "k8s.io/client-go/dynamic/fake"
)

func TestEngineScale_UsesScaleSubresource(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = autoscalingv1.AddToScheme(scheme)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "my-deploy", Namespace: "default"},
		Spec:       appsv1.DeploymentSpec{Replicas: ptrInt32(1)},
	}

	gvrMap := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}:       "DeploymentList",
		{Group: "apps", Version: "v1", Resource: "deployments/scale"}: "ScaleList",
	}
	dyn := fakedyn.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrMap, deploy)

	// Fake dynamic doesn't implement /scale, but UpdateScale on unstructured
	// Resource works via its subresource API. The fake client supports
	// subresource update via the "scale" subresource name.
	// (The assertion below verifies we called the scale subresource path.)
	// For a realistic test, we'd need to register a reactor; simpler:
	// just exercise the code path and assert no error.

	eng := &ResourceEngine{/* construct with dyn as needed; see existing engine setup */}
	_ = eng // silence unused if scaffolding differs

	err := eng.Scale(context.Background(), "c", "apps.v1.deployments", "default", "my-deploy", 3)
	testza.AssertNoError(t, err)
}

func ptrInt32(n int32) *int32 { return &n }
```

This test mostly exercises the code path; a more thorough test would use a reactor (`dyn.PrependReactor("update", "deployments", ...)`) to intercept and assert the subresource was "scale".

- [ ] **Step 7: Implement `Engine.Scale`**

Add to `internal/resource/engine.go`:

```go
import (
	// existing imports…
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// Scale updates a resource's replica count via the scale subresource. This
// works for any resource that declares a scale subresource, including CRDs.
// If the server rejects the subresource call with 404 (no scale subresource),
// the caller should fall back to MergePatch.
func (e *ResourceEngine) Scale(ctx context.Context, contextName, gvr, namespace, name string, replicas int32) error {
	conn, err := e.clusterMgr.GetConnection(contextName)
	if err != nil {
		return err
	}
	parsed, err := ParseGVR(gvr) // existing helper
	if err != nil {
		return err
	}

	// Read current scale to preserve resourceVersion for optimistic concurrency.
	current, getErr := conn.Dynamic.Resource(parsed).Namespace(namespace).Get(
		ctx, name, metav1.GetOptions{}, "scale",
	)
	if getErr != nil {
		return getErr
	}

	var scale autoscalingv1.Scale
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(current.Object, &scale); err != nil {
		return err
	}
	scale.Spec.Replicas = replicas

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&scale)
	if err != nil {
		return err
	}
	current.Object = u

	_, err = conn.Dynamic.Resource(parsed).Namespace(namespace).Update(
		ctx, current, metav1.UpdateOptions{}, "scale",
	)
	return err
}

// ScaleViaMergePatch is the legacy fallback for resources without a scale
// subresource. Preserves today's behavior for non-standard CRDs.
func (e *ResourceEngine) ScaleViaMergePatch(ctx context.Context, contextName, gvr, namespace, name string, replicas int32) error {
	patch := []byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, replicas))
	_, err := e.Patch(ctx, contextName, gvr, namespace, name, types.MergePatchType, patch)
	return err
}
```

If `ParseGVR` isn't exported, check the existing resource package for a GVR parsing helper (the CLAUDE.md references a `ParseGVR`) and use its real name. Add `"fmt"` to imports if not already present.

- [ ] **Step 8: Run engine tests**

Run: `go test ./internal/resource/ -v`
Expected: PASS. If the fake dynamic client doesn't support the scale subresource path, adjust the test to use a reactor that asserts the subresource name.

The controller prepared a fresh working-copy commit for Task 1. Do NOT run `jj new` or `jj desc` — snapshot captures your changes automatically.

---

## Task 2: Wire ScaleResource to engine + fallback path

**Files:**
- Modify: `internal/services/resource.go`
- Modify: `internal/services/resource_test.go`

- [ ] **Step 1: Update ScaleResource**

Replace the existing `ScaleResource` implementation with:

```go
func (s *ResourceService) ScaleResource(contextName, gvr, namespace, name string, replicas int32) error {
	mgr := s.appService.ClusterManager()
	if mgr.HasScaleSubresource(contextName, gvr) {
		if err := s.engine.Scale(s.ctx, contextName, gvr, namespace, name, replicas); err == nil {
			return nil
		} else if !errors.IsNotFound(err) && !errors.IsMethodNotSupported(err) {
			return err
		}
		// Fallthrough to MergePatch on 404 / 405 (subresource unexpectedly absent)
	}
	return s.engine.ScaleViaMergePatch(s.ctx, contextName, gvr, namespace, name, replicas)
}
```

Add imports: `errors "k8s.io/apimachinery/pkg/api/errors"`.

- [ ] **Step 2: Update existing test**

The existing `TestResourceService_ScaleResource` test (around line 60 of `resource_test.go`) exercises the MergePatch path with a Deployment. Keep it as-is but additionally verify:

Add a new test:

```go
func TestResourceService_ScaleResource_UsesSubresource_WhenAvailable(t *testing.T) {
	svc, kfake, dyn := newTestResourceServiceWithScale(t)
	// Build fixtures...
	// Register a reactor on dyn to assert subresource == "scale"
	var subresourceObserved string
	dyn.PrependReactor("update", "deployments", func(action clienttesting.Action) (bool, runtime.Object, error) {
		if ua, ok := action.(clienttesting.UpdateAction); ok {
			subresourceObserved = ua.GetSubresource()
		}
		return false, nil, nil // let default handler continue
	})

	err := svc.ScaleResource("ctx", "apps.v1.deployments", "default", "my-deploy", 7)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "scale", subresourceObserved)
	_ = kfake
}
```

If a helper like `newTestResourceServiceWithScale` doesn't exist, adapt the existing service test scaffolding to also construct a fake dynamic client and return it. The existing scaffolding is around lines 45-60 of `resource_test.go`.

Add import `clienttesting "k8s.io/client-go/testing"`.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/services/ -run TestResourceService_Scale -v`
Expected: PASS.

- [ ] **Step 4: Run full backend tests**

Run: `go test ./internal/... -v` (skipping CGO-requiring packages per CLAUDE.md: `go test ./internal/config/ ./internal/session/ ./internal/cluster/ ./internal/streaming/ ./internal/watcher/ ./internal/resource/ ./internal/services/ -v`)
Expected: all PASS.

The controller prepared a fresh working-copy commit for Task 2. Do NOT run `jj new` or `jj desc` — snapshot captures your changes automatically.

---

## Task 3: Bindings regen + RBAC + manual verification

**Files:**
- Modify: `frontend/src/lib/components/panels/ActionsToolbar.svelte` — confirm error handling already shows notification (likely does via `notificationStore`)

- [ ] **Step 1: Regenerate bindings**

Run: `wails3 generate bindings`
Expected: no-op or trivial diff (signature unchanged).

- [ ] **Step 2: Verify error handling path**

Open `frontend/src/lib/components/panels/ActionsToolbar.svelte`. Find the `doScale` handler. Confirm that on error the existing helper surfaces a notification (pattern seen in structural exploration: `"Scaled ${name} to ${replicas}"` / `"Scale failed"`).

- [ ] **Step 3: If the error surface only shows a generic "Scale failed"**

Enhance to include the error message. Example edit:

Before:
```typescript
() => ScaleResource(ctxName, gvr, namespace, name, scaleReplicas),
`Scaled ${name} to ${scaleReplicas}`,
"Scale failed",
```

After:
```typescript
() => ScaleResource(ctxName, gvr, namespace, name, scaleReplicas),
`Scaled ${name} to ${scaleReplicas}`,
(err) => `Scale failed: ${err?.message ?? err}`,
```

Only change this if the helper supports a function-form error message. Otherwise leave as-is — a generic error is acceptable.

- [ ] **Step 4: No test needed if nothing changed**

If you edited, run: `cd frontend && pnpm test`. Otherwise skip.

- [ ] **Step 5: Launch dev mode, test on a CRD with scale subresource**

Run: `task dev`

Candidate CRDs: `keda.sh/v1alpha1/scaledobjects`, `argoproj.io/v1alpha1/rollouts`, or any Helm chart that ships a CRD with `subresources.scale`.

Verify:
- The CRD's list page shows the Scale action in the row action menu.
- Clicking Scale opens the existing dialog.
- Submitting updates replicas (confirm via `kubectl get -o yaml` externally).
- Bulk scale works from the list page.
- A Deployment still scales correctly (existing path preserved).
- A resource without scale (e.g. a ConfigMap — which should not even show the action) does not attempt scale.

- [ ] **Step 6: Verify RBAC error surface**

Create a limited-privilege kubeconfig that lacks `patch/update` on `deployments/scale`. Confirm the scale attempt surfaces a clear error notification.

No additional commit needed — the controller will close out Phase 7 after Task 3 passes.

---

## Self-Review Checklist

- [x] Scale subresource used when available, MergePatch fallback otherwise (3 tasks not 5).
- [x] Resources without scale subresource don't even show the Scale action (Phase 2 handles this — auto-generated descriptors omit scale action for non-scalable GVRs; built-in descriptors were already correct).
- [x] Existing Scale dialog and BulkScaleDialog reused — no new UI components.
- [x] RBAC failures surface via existing notification path.
- [x] Discovery cache on Manager populated in Phase 1 is now read by this phase.
- [x] Commits per task.
