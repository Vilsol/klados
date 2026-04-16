# Generic GVR Phase 1 — Discovery Metadata Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend `DiscoverResources()` to collect `additionalPrinterColumns`, subresources, and scale-subresource paths for every resource, and emit this data in the discovery event payload.

**Architecture:** Adds two new types (`AdditionalPrinterColumn`, `ResourceSubresources`) and new fields on `APIResource`. Subresources come from the API server's existing resource list (entries like `deployments/scale`). Printer columns and scale paths come from a single batched `List` call against `apiextensions.k8s.io/v1/customresourcedefinitions` for CRD-backed resources.

**Tech Stack:** Go, k8s.io/client-go (`Discovery`, dynamic), k8s.io/apiextensions-apiserver CRD types, testza for tests.

**Reference spec:** `docs/superpowers/specs/2026-04-16-generic-gvr-capabilities-design.md` §1.

**VCS note:** This repo uses `jj`. Each Task's final step commits via `jj new && jj desc -m "…"` **before** starting the next task (matches the CLAUDE.md requirement to commit after each unit of work).

---

## File Structure

- Modify: `internal/cluster/manager.go` — extend `APIResource`, add new types, enhance `DiscoverResources()`
- Create: `internal/cluster/discovery_metadata.go` — helper functions for subresource detection and CRD parsing (kept separate so `manager.go` stays focused on the connection/lifecycle role)
- Create: `internal/cluster/discovery_metadata_test.go` — unit tests
- Modify: `frontend/bindings/github.com/Vilsol/klados/internal/cluster/models.js` — **regenerated** via `wails3 generate bindings` (do not hand-edit)
- Modify: `frontend/src/lib/stores/cluster.svelte.ts` — add the new fields to the discovery payload consumer (no behavior change — just typed access)

---

## Task 1: Add new types for discovery metadata

**Files:**
- Modify: `internal/cluster/manager.go:466-470` — extend `APIResource`, add new supporting types

- [ ] **Step 1: Extend `APIResource` and add new types**

Open `internal/cluster/manager.go` and replace the existing `APIResource` definition (currently around line 466-470):

```go
// APIResource describes a discoverable resource on the cluster, including
// optional metadata useful for rendering (printer columns, subresources,
// scale subresource paths for CRDs).
type APIResource struct {
	GVR         string                    `json:"gvr"`
	Kind        string                    `json:"kind"`
	Namespaced  bool                      `json:"namespaced"`
	Subresources ResourceSubresources     `json:"subresources"`
	PrinterColumns []AdditionalPrinterColumn `json:"printerColumns,omitempty"`
	ScaleSpec   *ScaleSubresourceSpec     `json:"scaleSpec,omitempty"`
}

// ResourceSubresources captures which well-known subresources are supported.
type ResourceSubresources struct {
	Scale  bool `json:"scale"`
	Status bool `json:"status"`
}

// AdditionalPrinterColumn mirrors the CRD printer column definition used by
// kubectl's column rendering.
type AdditionalPrinterColumn struct {
	Name        string `json:"name"`
	Type        string `json:"type"`        // "string" | "integer" | "number" | "boolean" | "date"
	Format      string `json:"format,omitempty"`
	Description string `json:"description,omitempty"`
	Priority    int32  `json:"priority"`    // 0 = visible by default
	JSONPath    string `json:"jsonPath"`
}

// ScaleSubresourceSpec captures the paths the CRD declared for its scale
// subresource. Defaults are "spec.replicas" / "status.replicas".
type ScaleSubresourceSpec struct {
	SpecReplicasPath   string `json:"specReplicasPath"`
	StatusReplicasPath string `json:"statusReplicasPath"`
}
```

- [ ] **Step 2: Verify the file still compiles**

Run: `go build ./internal/cluster/`
Expected: exits 0, no output.

- [ ] **Step 3: Commit**

```bash
jj desc -m "cluster: extend APIResource with printer columns and subresource metadata"
```

---

## Task 2: Implement subresource detection helper

**Files:**
- Create: `internal/cluster/discovery_metadata.go`
- Create: `internal/cluster/discovery_metadata_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cluster/discovery_metadata_test.go`:

```go
package cluster

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDetectSubresources_FromAPIResourceList(t *testing.T) {
	list := &metav1.APIResourceList{
		GroupVersion: "apps/v1",
		APIResources: []metav1.APIResource{
			{Name: "deployments", Namespaced: true, Kind: "Deployment"},
			{Name: "deployments/scale", Namespaced: true, Kind: "Scale"},
			{Name: "deployments/status", Namespaced: true, Kind: "Deployment"},
			{Name: "replicasets", Namespaced: true, Kind: "ReplicaSet"},
			{Name: "replicasets/scale", Namespaced: true, Kind: "Scale"},
			{Name: "statefulsets", Namespaced: true, Kind: "StatefulSet"},
		},
	}

	subs := DetectSubresources(list)

	testza.AssertTrue(t, subs["deployments"].Scale)
	testza.AssertTrue(t, subs["deployments"].Status)
	testza.AssertTrue(t, subs["replicasets"].Scale)
	testza.AssertFalse(t, subs["replicasets"].Status)
	testza.AssertFalse(t, subs["statefulsets"].Scale)
}

func TestDetectSubresources_Empty(t *testing.T) {
	subs := DetectSubresources(&metav1.APIResourceList{})
	testza.AssertEqual(t, 0, len(subs))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cluster/ -run TestDetectSubresources -v`
Expected: FAIL with "undefined: DetectSubresources".

- [ ] **Step 3: Implement `DetectSubresources`**

Create `internal/cluster/discovery_metadata.go`:

```go
package cluster

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DetectSubresources inspects an APIResourceList and returns a map keyed by
// parent resource name. Each entry records whether well-known subresources
// (scale, status) are served. Kubernetes exposes subresources as separate
// entries like "deployments/scale"; we group them back under the parent.
func DetectSubresources(list *metav1.APIResourceList) map[string]ResourceSubresources {
	out := map[string]ResourceSubresources{}
	if list == nil {
		return out
	}
	for _, r := range list.APIResources {
		name := r.Name
		if idx := strings.Index(name, "/"); idx >= 0 {
			parent := name[:idx]
			sub := name[idx+1:]
			entry := out[parent]
			switch sub {
			case "scale":
				entry.Scale = true
			case "status":
				entry.Status = true
			}
			out[parent] = entry
		} else if _, ok := out[name]; !ok {
			// Ensure parent is present even if it has no subresources yet.
			out[name] = ResourceSubresources{}
		}
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cluster/ -run TestDetectSubresources -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
jj new && jj desc -m "cluster: detect scale/status subresources from APIResourceList"
```

---

## Task 3: Extract printer columns and scale paths from CRDs

**Files:**
- Modify: `internal/cluster/discovery_metadata.go`
- Modify: `internal/cluster/discovery_metadata_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/cluster/discovery_metadata_test.go`:

```go
import (
	// …keep existing imports…
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestExtractCRDMetadata_PrinterColumnsAndScale(t *testing.T) {
	served := true
	crd := apiextv1.CustomResourceDefinition{
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Names: apiextv1.CustomResourceDefinitionNames{Plural: "widgets", Kind: "Widget"},
			Scope: apiextv1.NamespaceScoped,
			Versions: []apiextv1.CustomResourceDefinitionVersion{{
				Name:   "v1",
				Served: served,
				AdditionalPrinterColumns: []apiextv1.CustomResourceColumnDefinition{
					{Name: "Replicas", Type: "integer", JSONPath: ".spec.replicas"},
					{Name: "Ready", Type: "string", JSONPath: ".status.ready", Priority: 1},
				},
				Subresources: &apiextv1.CustomResourceSubresources{
					Scale: &apiextv1.CustomResourceSubresourceScale{
						SpecReplicasPath:   ".spec.size",
						StatusReplicasPath: ".status.currentSize",
					},
					Status: &apiextv1.CustomResourceSubresourceStatus{},
				},
			}},
		},
	}

	md := ExtractCRDMetadata([]apiextv1.CustomResourceDefinition{crd})

	gvr := "example.com.v1.widgets"
	entry, ok := md[gvr]
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, 2, len(entry.PrinterColumns))
	testza.AssertEqual(t, "Replicas", entry.PrinterColumns[0].Name)
	testza.AssertEqual(t, ".spec.replicas", entry.PrinterColumns[0].JSONPath)
	testza.AssertEqual(t, int32(1), entry.PrinterColumns[1].Priority)
	testza.AssertNotNil(t, entry.ScaleSpec)
	testza.AssertEqual(t, ".spec.size", entry.ScaleSpec.SpecReplicasPath)
	testza.AssertEqual(t, ".status.currentSize", entry.ScaleSpec.StatusReplicasPath)
}

func TestExtractCRDMetadata_DefaultScalePaths(t *testing.T) {
	crd := apiextv1.CustomResourceDefinition{
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Names: apiextv1.CustomResourceDefinitionNames{Plural: "things"},
			Versions: []apiextv1.CustomResourceDefinitionVersion{{
				Name: "v1", Served: true,
				Subresources: &apiextv1.CustomResourceSubresources{
					Scale: &apiextv1.CustomResourceSubresourceScale{},
				},
			}},
		},
	}

	md := ExtractCRDMetadata([]apiextv1.CustomResourceDefinition{crd})
	entry := md["example.com.v1.things"]
	testza.AssertEqual(t, ".spec.replicas", entry.ScaleSpec.SpecReplicasPath)
	testza.AssertEqual(t, ".status.replicas", entry.ScaleSpec.StatusReplicasPath)
}

func TestExtractCRDMetadata_SkipsUnservedVersions(t *testing.T) {
	crd := apiextv1.CustomResourceDefinition{
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Names: apiextv1.CustomResourceDefinitionNames{Plural: "widgets"},
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{Name: "v1alpha1", Served: false},
				{Name: "v1", Served: true},
			},
		},
	}

	md := ExtractCRDMetadata([]apiextv1.CustomResourceDefinition{crd})
	_, hasAlpha := md["example.com.v1alpha1.widgets"]
	_, hasV1 := md["example.com.v1.widgets"]
	testza.AssertFalse(t, hasAlpha)
	testza.AssertTrue(t, hasV1)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cluster/ -run TestExtractCRDMetadata -v`
Expected: FAIL with "undefined: ExtractCRDMetadata".

- [ ] **Step 3: Implement `ExtractCRDMetadata`**

Append to `internal/cluster/discovery_metadata.go`:

```go
import (
	// …keep existing imports…
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// CRDMetadata is the per-GVR data extracted from a CRD object.
type CRDMetadata struct {
	PrinterColumns []AdditionalPrinterColumn
	ScaleSpec      *ScaleSubresourceSpec
}

// ExtractCRDMetadata walks a list of CRDs and produces a per-GVR metadata
// map keyed by GVR in the same dot-separated format as APIResource.GVR.
// Only served versions are included.
func ExtractCRDMetadata(crds []apiextv1.CustomResourceDefinition) map[string]CRDMetadata {
	out := map[string]CRDMetadata{}
	for _, crd := range crds {
		group := crd.Spec.Group
		plural := crd.Spec.Names.Plural
		for _, v := range crd.Spec.Versions {
			if !v.Served {
				continue
			}
			gvr := formatGVR(group, v.Name, plural)

			md := CRDMetadata{}
			for _, c := range v.AdditionalPrinterColumns {
				md.PrinterColumns = append(md.PrinterColumns, AdditionalPrinterColumn{
					Name:        c.Name,
					Type:        c.Type,
					Format:      c.Format,
					Description: c.Description,
					Priority:    c.Priority,
					JSONPath:    c.JSONPath,
				})
			}
			if v.Subresources != nil && v.Subresources.Scale != nil {
				spec := v.Subresources.Scale.SpecReplicasPath
				status := v.Subresources.Scale.StatusReplicasPath
				if spec == "" {
					spec = ".spec.replicas"
				}
				if status == "" {
					status = ".status.replicas"
				}
				md.ScaleSpec = &ScaleSubresourceSpec{
					SpecReplicasPath:   spec,
					StatusReplicasPath: status,
				}
			}
			out[gvr] = md
		}
	}
	return out
}

// formatGVR produces the dot-separated GVR string used elsewhere in the
// codebase (e.g. "example.com.v1.widgets", "core.v1.pods"). An empty group
// becomes "core" to match built-in convention.
func formatGVR(group, version, resource string) string {
	if group == "" {
		group = "core"
	}
	return group + "." + version + "." + resource
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cluster/ -run TestExtractCRDMetadata -v`
Expected: all three tests PASS.

- [ ] **Step 5: Commit**

```bash
jj new && jj desc -m "cluster: extract printer columns and scale paths from CRD specs"
```

---

## Task 4: Integrate metadata into DiscoverResources

**Files:**
- Modify: `internal/cluster/manager.go` (in `DiscoverResources`, around lines 472-515)

- [ ] **Step 1: Read the current DiscoverResources to locate the insertion points**

Run: `go doc -all ./internal/cluster` | head -80` (informational — to confirm method set). Also open `internal/cluster/manager.go` and locate the block that iterates `ServerPreferredResources()` and constructs `APIResource` structs. Note that we need to: (a) collect subresources from ALL preferred resources by GroupVersion (the list currently filters them out), (b) fetch CRDs once, (c) merge into the emitted slice.

- [ ] **Step 2: Replace DiscoverResources implementation**

In `internal/cluster/manager.go`, replace the body of `DiscoverResources` with:

```go
func (m *Manager) DiscoverResources(contextName string) ([]APIResource, error) {
	conn, err := m.getConnection(contextName)
	if err != nil {
		return nil, err
	}

	lists, err := conn.Discovery.ServerPreferredResources()
	if err != nil && len(lists) == 0 {
		return nil, err
	}

	// Pass 1: detect subresources per-group, collect primary resources.
	subsByGVR := map[string]ResourceSubresources{}
	var primary []APIResource
	for _, list := range lists {
		if list == nil {
			continue
		}
		subs := DetectSubresources(list)
		gv, gvErr := parseGroupVersion(list.GroupVersion)
		if gvErr != nil {
			slox.Warn(m.ctx, "discovery: invalid group/version", "gv", list.GroupVersion, "err", gvErr)
			continue
		}
		for _, r := range list.APIResources {
			if strings.Contains(r.Name, "/") {
				continue // skip subresource entries
			}
			gvrStr := formatGVR(gv.group, gv.version, r.Name)
			subsByGVR[gvrStr] = subs[r.Name]
			primary = append(primary, APIResource{
				GVR:          gvrStr,
				Kind:         r.Kind,
				Namespaced:   r.Namespaced,
				Subresources: subs[r.Name],
			})
		}
	}

	// Pass 2: fetch CRDs once and merge printer columns + scale specs.
	crdMeta := m.fetchCRDMetadata(conn)
	for i := range primary {
		if md, ok := crdMeta[primary[i].GVR]; ok {
			primary[i].PrinterColumns = md.PrinterColumns
			primary[i].ScaleSpec = md.ScaleSpec
		}
	}

	m.emitEvent(fmt.Sprintf("discovery:%s:resources", contextName), primary)
	return primary, nil
}

// parseGroupVersion splits a "group/version" string; empty group becomes "core".
type groupVersion struct{ group, version string }

func parseGroupVersion(gv string) (groupVersion, error) {
	parts := strings.SplitN(gv, "/", 2)
	if len(parts) == 1 {
		return groupVersion{group: "core", version: parts[0]}, nil
	}
	if parts[0] == "" {
		return groupVersion{}, fmt.Errorf("empty group in %q", gv)
	}
	return groupVersion{group: parts[0], version: parts[1]}, nil
}
```

Also add the import `"strings"` if not already present, and `"github.com/go-slog/slox"` if not already present in this file (look for `slox.` usage).

- [ ] **Step 3: Implement `fetchCRDMetadata`**

Add below `DiscoverResources`:

```go
// fetchCRDMetadata performs a single List against the CRD API group (if the
// cluster exposes it) and returns per-GVR metadata. Empty map is returned on
// any error — CRD absence or RBAC denial should not block discovery.
func (m *Manager) fetchCRDMetadata(conn *Connection) map[string]CRDMetadata {
	if conn.Dynamic == nil {
		return map[string]CRDMetadata{}
	}
	crdGVR := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
	list, err := conn.Dynamic.Resource(crdGVR).List(m.ctx, metav1.ListOptions{})
	if err != nil {
		slox.Debug(m.ctx, "discovery: CRD list unavailable, skipping printer columns", "err", err)
		return map[string]CRDMetadata{}
	}

	crds := make([]apiextv1.CustomResourceDefinition, 0, len(list.Items))
	for i := range list.Items {
		var crd apiextv1.CustomResourceDefinition
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(list.Items[i].Object, &crd); err != nil {
			slox.Debug(m.ctx, "discovery: CRD convert failed", "err", err)
			continue
		}
		crds = append(crds, crd)
	}
	return ExtractCRDMetadata(crds)
}
```

And add these imports at the top of `manager.go` (merge with existing):

```go
import (
	// existing imports…
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)
```

- [ ] **Step 4: Verify Connection has a Dynamic field**

Run: `grep -n "Dynamic" internal/cluster/manager.go | head -20`
Expected: a line like `Dynamic dynamic.Interface` in the `Connection` struct. If that's not present, add it and initialize it in the same place where `Discovery` is initialized (search for `discovery.NewDiscoveryClient` or similar in `manager.go` and `connection.go` if it exists).

If not already initialized, add after the Discovery client is built:

```go
conn.Dynamic, err = dynamic.NewForConfig(restConfig)
if err != nil {
    return nil, fmt.Errorf("dynamic client: %w", err)
}
```

And import `"k8s.io/client-go/dynamic"`.

- [ ] **Step 5: Add go.mod entry if needed**

Run: `go mod tidy`
Expected: `apiextensions-apiserver` and `client-go/dynamic` resolved without errors.

- [ ] **Step 6: Run all cluster tests**

Run: `go test ./internal/cluster/ -v`
Expected: all tests PASS.

- [ ] **Step 7: Build the whole backend**

Run: `go build ./...`
Expected: exits 0.

- [ ] **Step 8: Commit**

```bash
jj new && jj desc -m "cluster: fetch printer columns and subresources in DiscoverResources"
```

---

## Task 5: Integration test for DiscoverResources with fake clientsets

**Files:**
- Create: `internal/cluster/discover_integration_test.go`

- [ ] **Step 1: Write the integration test**

Create `internal/cluster/discover_integration_test.go`:

```go
package cluster

import (
	"context"
	"testing"

	"github.com/MarvinJWendt/testza"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedisc "k8s.io/client-go/discovery/fake"
	fakedyn "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDiscoverResources_EmitsEnrichedPayload(t *testing.T) {
	kfake := fake.NewSimpleClientset()
	disc := kfake.Discovery().(*fakedisc.FakeDiscovery)
	disc.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "pods", Namespaced: true, Kind: "Pod"},
			},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{Name: "deployments", Namespaced: true, Kind: "Deployment"},
				{Name: "deployments/scale", Namespaced: true, Kind: "Scale"},
			},
		},
	}

	crdGVR := schema.GroupVersionResource{
		Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions",
	}
	scheme := runtime.NewScheme()
	_ = apiextv1.AddToScheme(scheme)
	served := true
	widgetCRD := &apiextv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: "widgets.example.com"},
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Names: apiextv1.CustomResourceDefinitionNames{Plural: "widgets", Kind: "Widget"},
			Versions: []apiextv1.CustomResourceDefinitionVersion{{
				Name: "v1", Served: served,
				AdditionalPrinterColumns: []apiextv1.CustomResourceColumnDefinition{
					{Name: "Replicas", Type: "integer", JSONPath: ".spec.replicas"},
				},
			}},
		},
	}
	dyn := fakedyn.NewSimpleDynamicClient(scheme, widgetCRD)
	_ = crdGVR // gvr is used implicitly by the dynamic client

	var events []struct {
		name string
		data any
	}
	emit := func(n string, d any) {
		events = append(events, struct {
			name string
			data any
		}{n, d})
	}

	m := &Manager{
		connections: map[string]*Connection{
			"c": {
				Discovery: disc,
				Dynamic:   dyn,
			},
		},
		emitEvent: emit,
		ctx:       context.Background(),
	}

	got, err := m.DiscoverResources("c")
	testza.AssertNoError(t, err)

	byGVR := map[string]APIResource{}
	for _, r := range got {
		byGVR[r.GVR] = r
	}

	testza.AssertEqual(t, "Pod", byGVR["core.v1.pods"].Kind)
	testza.AssertTrue(t, byGVR["apps.v1.deployments"].Subresources.Scale)

	widget, ok := byGVR["example.com.v1.widgets"]
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, 1, len(widget.PrinterColumns))
	testza.AssertEqual(t, "Replicas", widget.PrinterColumns[0].Name)

	testza.AssertEqual(t, 1, len(events))
	testza.AssertEqual(t, "discovery:c:resources", events[0].name)
}
```

Note: if `Manager.getConnection` uses an internal locking path that this test can't satisfy with direct struct construction, add a small test helper at the bottom of `manager.go` guarded by `//go:build !integration` — OR simply make `getConnection` return the map entry under a read lock (it likely already does). Inspect before proceeding.

- [ ] **Step 2: Run test**

Run: `go test ./internal/cluster/ -run TestDiscoverResources_EmitsEnrichedPayload -v`
Expected: PASS.

- [ ] **Step 3: Run all cluster tests**

Run: `go test ./internal/cluster/ -v`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
jj new && jj desc -m "cluster: integration test for enriched DiscoverResources payload"
```

---

## Task 6: Regenerate Wails bindings and plumb payload to frontend

**Files:**
- Regenerate: `frontend/bindings/github.com/Vilsol/klados/internal/cluster/models.js`
- Modify: `frontend/src/lib/stores/cluster.svelte.ts` — expose the new discovery fields on whatever store/type receives the payload

- [ ] **Step 1: Regenerate bindings**

Run: `wails3 generate bindings`
Expected: exits 0. New types `APIResource`, `AdditionalPrinterColumn`, `ResourceSubresources`, `ScaleSubresourceSpec` appear in `frontend/bindings/github.com/Vilsol/klados/internal/cluster/models.js`.

- [ ] **Step 2: Verify generated files**

Run: `grep -n "PrinterColumns\|Subresources\|ScaleSpec" frontend/bindings/github.com/Vilsol/klados/internal/cluster/models.js | head -20`
Expected: matches showing the new fields.

- [ ] **Step 3: Expose the fields in the frontend store**

Open `frontend/src/lib/stores/cluster.svelte.ts` and find where the discovery event is handled (`Events.On('discovery:${ctx}:resources', ...)`). Currently items are stored as arrays of `{gvr, kind, namespaced}` objects. Update the consumer to keep the full payload, then re-export the richer type. Concrete edit:

Find the existing handler (pattern: `Events.On(\`discovery:${ctxName}:resources\`` ) and change the local variable typing from the narrow shape to the full `APIResource` type imported from bindings.

Specifically, add to the imports at the top:

```typescript
import type { APIResource } from "../../../bindings/github.com/Vilsol/klados/internal/cluster/index.js";
```

And change any place that stores the discovery list (e.g. `availableResources` or `apiResources`) to type `APIResource[]` and pass through unchanged.

If the handler currently normalizes fields, preserve the originals alongside.

- [ ] **Step 4: Type-check the frontend**

Run: `cd frontend && pnpm check`
Expected: exits 0.

- [ ] **Step 5: Run frontend tests**

Run: `cd frontend && pnpm test`
Expected: existing tests pass.

- [ ] **Step 6: Commit**

```bash
jj new && jj desc -m "frontend: regenerate bindings for enriched APIResource payload"
```

---

## Task 7: Manual verification against a live cluster (optional gate)

- [ ] **Step 1: Start dev mode**

Run: `task dev`

- [ ] **Step 2: Connect to a cluster that has at least one CRD with `additionalPrinterColumns`**

Recommended: `cert-manager` (Certificates), `argoproj.io` (Applications), or any HelmRelease CRD.

- [ ] **Step 3: Open browser devtools and inspect the discovery event payload**

In the console: `wails.Events.On('discovery:<ctx>:resources', (e) => console.log(e.data))` then trigger a reconnect. Verify the payload contains `printerColumns` for the CRD and `subresources.scale: true` for Deployments.

- [ ] **Step 4: Commit the phase marker**

```bash
jj new && jj desc -m "docs: phase 1 discovery metadata complete"
```

---

## Self-Review Checklist

- [x] **Spec coverage:** §1 "Enhanced Discovery" fully covered across Tasks 1-4.
- [x] **No placeholders:** all code is complete; no TBD/TODO in steps.
- [x] **Type consistency:** `APIResource`, `ResourceSubresources`, `AdditionalPrinterColumn`, `ScaleSubresourceSpec`, `CRDMetadata` used consistently.
- [x] **Tests precede implementation:** Task 2, 3, 5 follow TDD.
- [x] **Commits per unit:** every task ends with a `jj new && jj desc -m` commit.
