package resource_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MarvinJWendt/testza"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/fake"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/resource"
)

type fakeProvider struct {
	dynamic *fake.FakeDynamicClient
}

func (f *fakeProvider) GetConnection(_ string) (*cluster.Connection, error) {
	conn := &cluster.Connection{}
	conn.Dynamic = f.dynamic
	return conn, nil
}

var podGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

func newFakeEngine(objects ...runtime.Object) (*resource.ResourceEngine, *fake.FakeDynamicClient) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		podGVR: "PodList",
	}, objects...)

	enricherReg := resource.NewEnricherRegistry()
	return resource.NewResourceEngine(&fakeProvider{dyn}, enricherReg), dyn
}

func makePod(name, namespace string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func TestResourceEngine_List(t *testing.T) {
	engine, _ := newFakeEngine(makePod("test-pod", "default"))

	items, _, err := engine.List(context.Background(), "ctx", "core.v1.pods", "default")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(items))

	meta, ok := items[0]["metadata"].(map[string]any)
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, "test-pod", meta["name"])
}

func TestResourceEngine_List_AllNamespaces(t *testing.T) {
	engine, _ := newFakeEngine(makePod("pod-a", "ns1"), makePod("pod-b", "ns2"))

	items, _, err := engine.List(context.Background(), "ctx", "core.v1.pods", "")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 2, len(items))
}

func TestResourceEngine_Get(t *testing.T) {
	engine, _ := newFakeEngine(makePod("my-pod", "default"))

	obj, err := engine.Get(context.Background(), "ctx", "core.v1.pods", "default", "my-pod")
	testza.AssertNoError(t, err)

	meta, ok := obj["metadata"].(map[string]any)
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, "my-pod", meta["name"])
}

func TestResourceEngine_Delete(t *testing.T) {
	engine, dyn := newFakeEngine(makePod("del-pod", "default"))

	err := engine.Delete(context.Background(), "ctx", "core.v1.pods", "default", "del-pod")
	testza.AssertNoError(t, err)

	list, err := dyn.Resource(podGVR).Namespace("default").List(context.Background(), metav1.ListOptions{})
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 0, len(list.Items))
}

func TestResourceEngine_ForceDelete(t *testing.T) {
	engine, dyn := newFakeEngine(makePod("force-pod", "default"))

	err := engine.ForceDelete(context.Background(), "ctx", "core.v1.pods", "default", "force-pod")
	testza.AssertNoError(t, err)

	list, err := dyn.Resource(podGVR).Namespace("default").List(context.Background(), metav1.ListOptions{})
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 0, len(list.Items))
}

func TestResourceEngine_Update(t *testing.T) {
	engine, _ := newFakeEngine(makePod("update-pod", "default"))

	obj, err := engine.Get(context.Background(), "ctx", "core.v1.pods", "default", "update-pod")
	testza.AssertNoError(t, err)

	meta := obj["metadata"].(map[string]any)
	if meta["labels"] == nil {
		meta["labels"] = map[string]any{}
	}
	meta["labels"].(map[string]any)["env"] = "test"

	updated, err := engine.Update(context.Background(), "ctx", "core.v1.pods", "default", obj)
	testza.AssertNoError(t, err)

	updatedMeta := updated["metadata"].(map[string]any)
	labels := updatedMeta["labels"].(map[string]any)
	testza.AssertEqual(t, "test", labels["env"])
}

func TestResourceEngine_Patch(t *testing.T) {
	engine, _ := newFakeEngine(makePod("patch-pod", "default"))

	patch := []byte(`{"metadata":{"labels":{"patched":"true"}}}`)
	updated, err := engine.Patch(context.Background(), "ctx", "core.v1.pods", "default", "patch-pod",
		types.MergePatchType, patch)
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, updated)
}

func TestResourceEngine_Create(t *testing.T) {
	// Use a minimal scheme (no typed Pod) so the fake client stores as pure unstructured
	scheme := runtime.NewScheme()
	crdGVR := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		crdGVR: "WidgetList",
	})
	enricherReg := resource.NewEnricherRegistry()
	engine := resource.NewResourceEngine(&fakeProvider{dyn}, enricherReg)

	newWidget := map[string]any{
		"apiVersion": "example.com/v1",
		"kind":       "Widget",
		"metadata": map[string]any{
			"name":      "my-widget",
			"namespace": "default",
		},
	}

	result, err := engine.Create(context.Background(), "ctx", "example.com.v1.widgets", "default", newWidget)
	testza.AssertNoError(t, err)

	meta := result["metadata"].(map[string]any)
	testza.AssertEqual(t, "my-widget", meta["name"])
}

type errorProvider struct{}

func (e *errorProvider) GetConnection(_ string) (*cluster.Connection, error) {
	return nil, fmt.Errorf("connection refused")
}

func TestResourceEngine_List_Error(t *testing.T) {
	enricherReg := resource.NewEnricherRegistry()
	engine := resource.NewResourceEngine(&errorProvider{}, enricherReg)

	_, _, err := engine.List(context.Background(), "ctx", "core.v1.pods", "default")
	testza.AssertNotNil(t, err)
}

func TestResourceEngine_Get_Error(t *testing.T) {
	enricherReg := resource.NewEnricherRegistry()
	engine := resource.NewResourceEngine(&errorProvider{}, enricherReg)

	_, err := engine.Get(context.Background(), "ctx", "core.v1.pods", "default", "my-pod")
	testza.AssertNotNil(t, err)
}

func TestResourceEngine_Scale_UnknownContext(t *testing.T) {
	enricherReg := resource.NewEnricherRegistry()
	engine := resource.NewResourceEngine(&errorProvider{}, enricherReg)

	err := engine.Scale(context.Background(), "ctx", "apps.v1.deployments", "default", "my-deploy", 3)
	testza.AssertNotNil(t, err)
}

func TestResourceEngine_ScaleViaMergePatch_UnknownContext(t *testing.T) {
	enricherReg := resource.NewEnricherRegistry()
	engine := resource.NewResourceEngine(&errorProvider{}, enricherReg)

	err := engine.ScaleViaMergePatch(context.Background(), "ctx", "apps.v1.deployments", "default", "my-deploy", 3)
	testza.AssertNotNil(t, err)
}

type fakeVirtualBackend struct {
	items    []map[string]any
	rv       string
	listErr  error
	getObj   map[string]any
	getErr   error
	gotCtx   string
	gotNS    string
	gotName  string
	listCall int
}

func (f *fakeVirtualBackend) List(_ context.Context, contextName, namespace string) ([]map[string]any, string, error) {
	f.listCall++
	f.gotCtx = contextName
	f.gotNS = namespace
	if f.listErr != nil {
		return nil, "", f.listErr
	}
	return f.items, f.rv, nil
}

func (f *fakeVirtualBackend) Get(_ context.Context, contextName, namespace, name string) (map[string]any, error) {
	f.gotCtx = contextName
	f.gotNS = namespace
	f.gotName = name
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getObj, nil
}

type tagEnricher struct{ tag string }

func (e *tagEnricher) Enrich(_ string, u *unstructured.Unstructured) error {
	status, _, _ := unstructured.NestedMap(u.Object, "status")
	if status == nil {
		status = map[string]any{}
	}
	status["enrichTag"] = e.tag
	u.Object["status"] = status
	return nil
}

func TestResourceEngine_List_VirtualDispatch(t *testing.T) {
	enricherReg := resource.NewEnricherRegistry()
	engine := resource.NewResourceEngine(&fakeProvider{}, enricherReg)

	backend := &fakeVirtualBackend{
		items: []map[string]any{
			{"metadata": map[string]any{"name": "rel-a", "namespace": "ns1"}},
			{"metadata": map[string]any{"name": "rel-b", "namespace": "ns1"}},
		},
		rv: "rv-42",
	}
	engine.RegisterVirtual("helm.v1.releases", backend)

	items, rv, err := engine.List(context.Background(), "my-ctx", "helm.v1.releases", "ns1")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 2, len(items))
	testza.AssertEqual(t, "rv-42", rv)
	testza.AssertEqual(t, "my-ctx", backend.gotCtx)
	testza.AssertEqual(t, "ns1", backend.gotNS)
	testza.AssertEqual(t, 1, backend.listCall)
}

func TestResourceEngine_List_VirtualAppliesEnrichers(t *testing.T) {
	enricherReg := resource.NewEnricherRegistry()
	enricherReg.Register("helm.v1.releases", &tagEnricher{tag: "v"})
	engine := resource.NewResourceEngine(&fakeProvider{}, enricherReg)

	backend := &fakeVirtualBackend{
		items: []map[string]any{{"metadata": map[string]any{"name": "rel-a"}}},
		rv:    "rv",
	}
	engine.RegisterVirtual("helm.v1.releases", backend)

	items, _, err := engine.List(context.Background(), "ctx", "helm.v1.releases", "")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(items))
	status, _ := items[0]["status"].(map[string]any)
	testza.AssertEqual(t, "v", status["enrichTag"])
}

func TestResourceEngine_Get_VirtualDispatch(t *testing.T) {
	enricherReg := resource.NewEnricherRegistry()
	enricherReg.Register("helm.v1.releases", &tagEnricher{tag: "g"})
	engine := resource.NewResourceEngine(&fakeProvider{}, enricherReg)

	backend := &fakeVirtualBackend{
		getObj: map[string]any{"metadata": map[string]any{"name": "rel-x", "namespace": "ns"}},
	}
	engine.RegisterVirtual("helm.v1.releases", backend)

	obj, err := engine.Get(context.Background(), "ctx", "helm.v1.releases", "ns", "rel-x")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "rel-x", backend.gotName)
	status, _ := obj["status"].(map[string]any)
	testza.AssertEqual(t, "g", status["enrichTag"])
}

func TestResourceEngine_Virtuals_NoConflict(t *testing.T) {
	enricherReg := resource.NewEnricherRegistry()
	engine := resource.NewResourceEngine(&fakeProvider{}, enricherReg)

	a := &fakeVirtualBackend{items: []map[string]any{{"metadata": map[string]any{"name": "a"}}}, rv: "1"}
	b := &fakeVirtualBackend{items: []map[string]any{{"metadata": map[string]any{"name": "b1"}}, {"metadata": map[string]any{"name": "b2"}}}, rv: "2"}
	engine.RegisterVirtual("helm.v1.releases", a)
	engine.RegisterVirtual("flux.v1.releases", b)

	itemsA, rvA, err := engine.List(context.Background(), "ctx", "helm.v1.releases", "")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(itemsA))
	testza.AssertEqual(t, "1", rvA)

	itemsB, rvB, err := engine.List(context.Background(), "ctx", "flux.v1.releases", "")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 2, len(itemsB))
	testza.AssertEqual(t, "2", rvB)
}

func TestResourceEngine_List_VirtualBackendError(t *testing.T) {
	enricherReg := resource.NewEnricherRegistry()
	engine := resource.NewResourceEngine(&fakeProvider{}, enricherReg)

	engine.RegisterVirtual("helm.v1.releases", &fakeVirtualBackend{listErr: fmt.Errorf("boom")})
	_, _, err := engine.List(context.Background(), "ctx", "helm.v1.releases", "")
	testza.AssertNotNil(t, err)
}

func TestParseGVR(t *testing.T) {
	tests := []struct {
		input   string
		group   string
		version string
		res     string
	}{
		{"core.v1.pods", "", "v1", "pods"},
		{"apps.v1.deployments", "apps", "v1", "deployments"},
		{"networking.k8s.io.v1.ingresses", "networking.k8s.io", "v1", "ingresses"},
		{"batch.v1.jobs", "batch", "v1", "jobs"},
	}

	for _, tt := range tests {
		gvr, err := resource.ParseGVR(tt.input)
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, tt.group, gvr.Group)
		testza.AssertEqual(t, tt.version, gvr.Version)
		testza.AssertEqual(t, tt.res, gvr.Resource)
	}
}
